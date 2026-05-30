package timesheet

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

type Creator struct {
	TimesheetRepo domain.TimesheetRepository
	PayrateRepo   domain.PayrateRepository
	Validator     *Validator
}

// CreateTimesheetRequestWithTypes holds timesheet data with separate hour and day types
type CreateTimesheetRequestWithTypes struct {
	Timesheet *domain.Timesheet
	HourType  string
	DayType   *string
}

func NewCreator(
	timesheetRepo domain.TimesheetRepository,
	payrateRepo domain.PayrateRepository,
	validator *Validator,
) *Creator {
	return &Creator{
		TimesheetRepo: timesheetRepo,
		PayrateRepo:   payrateRepo,
		Validator:     validator,
	}
}

// CreateTimesheet creates a timesheet with proper day type determination
func (c *Creator) CreateTimesheet(ctx context.Context, timesheet *domain.Timesheet, hourType string, userDayType *string, createdBy uint, userRole string) (*domain.Timesheet, error) {
	// Validate employee assignment to project
	assignment, err := c.Validator.ValidateEmployeeAssignment(ctx, timesheet.ProjectID, timesheet.EmployeeID)
	if err != nil {
		return nil, err
	}

	// Validate that timesheet date is within employee's work period for the project
	if err := c.Validator.ValidateEmployeeWorkPeriod(ctx, timesheet.ProjectID, timesheet.EmployeeID, timesheet.Date); err != nil {
		return nil, err
	}

	// Check if project status allows timesheet creation
	if assignment.Project.IsDraft() {
		return nil, domain.NewValidationError(fmt.Sprintf("Không thể tạo bảng chấm công cho dự án '%s'. Dự án đang ở trạng thái nháp. Vui lòng kích hoạt dự án trước khi thêm mục chấm công", assignment.Project.Name))
	}
	if assignment.Project.IsPaused() {
		return nil, domain.NewValidationError(fmt.Sprintf("Không thể tạo bảng chấm công cho dự án '%s'. Dự án hiện đang tạm dừng. Vui lòng kích hoạt dự án trước khi thêm mục chấm công", assignment.Project.Name))
	}

	// Determine day type based on date and rules
	dayType, err := c.Validator.DetermineDayType(timesheet.Date, userDayType)
	if err != nil {
		return nil, err
	}

	// Get position from employee assignment (level 1 of 3-level structure)
	position := assignment.Position
	if position == "" {
		position = "phổ thông" // Default position
	}

	// Construct the 3-level paytype: position.dayType.hourType (all lowercase for consistency)
	paytype := fmt.Sprintf("%s.%s.%s", strings.ToLower(position), strings.ToLower(dayType), strings.ToLower(hourType))
	timesheet.PayType = paytype

	// Log paytype construction for debugging
	logger := observability.GetLogger()
	logger.Info("Constructed paytype for timesheet",
		"paytype", paytype,
		"position", position,
		"dayType", dayType,
		"hourType", hourType,
		"employeeID", timesheet.EmployeeID,
		"projectID", timesheet.ProjectID,
		"projectName", assignment.Project.Name,
		"date", timesheet.Date.Format("2006-01-02"))

	// Check for existing timesheet entry (upsert behavior) - lookup by hour type
	existing, err := c.TimesheetRepo.GetByProjectEmployeeDateHourType(ctx, timesheet.ProjectID, timesheet.EmployeeID, timesheet.Date, hourType)
	isUpdate := err == nil && existing != nil

	// Check if trying to update an approved timesheet as a partner
	if isUpdate && userRole == "partner" && existing.Status == domain.TimesheetStatusApproved {
		return nil, fmt.Errorf("không thể sửa đổi bảng chấm công đã được phê duyệt")
	}

	// Validate one project per day constraint
	if err := c.Validator.ValidateOneProjectPerDay(ctx, timesheet.EmployeeID, timesheet.ProjectID, timesheet.Date); err != nil {
		return nil, err
	}

	// Validate daily hours limit (24 hours per day across all hour types)
	// For upsert: calculate total after replacing existing hours with new hours
	if err := c.Validator.ValidateDailyHoursLimitForUpsert(ctx, timesheet.EmployeeID, timesheet.Date, timesheet.HoursWorked, existing); err != nil {
		return nil, err
	}

	// Get active payrate configuration for the project
	payrate, err := c.PayrateRepo.GetActiveByProjectAndDate(ctx, timesheet.ProjectID, timesheet.Date)
	if err != nil || payrate == nil {
		return nil, domain.NewValidationError(fmt.Sprintf("Không tìm thấy cấu hình mức lương hiệu lực cho dự án '%s' vào ngày %s. Vui lòng đảm bảo cấu hình mức lương được thiết lập và hoạt động cho dự án này", assignment.Project.Name, timesheet.Date.Format("2006-01-02")))
	}

	// Check if hourType rate is 0 (not allowed for timesheet creation)
	rateValue, err := payrate.GetRateValue(paytype)
	if err != nil {
		return nil, fmt.Errorf("failed to get payrate for path '%s': %w", paytype, err)
	}

	// Prevent timesheet creation when hourType has rate of 0
	if rateValue == 0 {
		return nil, domain.NewValidationError(fmt.Sprintf(constants.MsgCannotCreateTimesheetZeroRateVN, hourType))
	}

	// Set payrate using the 3-level path
	if err := timesheet.SetPayRateFromConfig(payrate, paytype); err != nil {
		return nil, fmt.Errorf("failed to set payrate for path '%s': %w", paytype, err)
	}

	// Set initial status and last updated by
	timesheet.Status = domain.TimesheetStatusPendingApproval
	timesheet.CreatedBy = createdBy

	// Perform upsert operation
	if isUpdate {
		// Update existing record
		existing.HoursWorked = timesheet.HoursWorked
		existing.PayRate = timesheet.PayRate
		existing.CreatedBy = createdBy

		// Set status based on user role
		if userRole == "admin" {
			now := clock.Now()
			existing.Status = domain.TimesheetStatusApproved
			existing.ApprovedBy = &createdBy
			existing.ApprovedAt = &now
		} else {
			// Partner submissions require approval
			existing.Status = domain.TimesheetStatusPendingApproval
		}

		if err := c.TimesheetRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update timesheet: %w", err)
		}
		timesheet = existing // Return the updated record
	} else {
		// Create new record - set status based on user role
		if userRole == "admin" {
			now := clock.Now()
			timesheet.Status = domain.TimesheetStatusApproved
			timesheet.ApprovedBy = &createdBy
			timesheet.ApprovedAt = &now
		} else {
			// Partner submissions require approval
			timesheet.Status = domain.TimesheetStatusPendingApproval
		}

		if err := c.TimesheetRepo.Create(ctx, timesheet); err != nil {
			return nil, fmt.Errorf("failed to create timesheet: %w", err)
		}
	}

	return timesheet, nil
}

// BulkCreateTimesheets creates multiple timesheets in a single transaction
func (c *Creator) BulkCreateTimesheets(ctx context.Context, requests []CreateTimesheetRequestWithTypes, createdBy uint, userRole string) ([]domain.Timesheet, []error) {
	var timesheets []domain.Timesheet
	var errors []error

	for _, req := range requests {
		timesheet, err := c.CreateTimesheet(ctx, req.Timesheet, req.HourType, req.DayType, createdBy, userRole)
		if err != nil {
			errors = append(errors, err)
			timesheets = append(timesheets, domain.Timesheet{})
		} else {
			errors = append(errors, nil)
			timesheets = append(timesheets, *timesheet)
		}
	}

	return timesheets, errors
}
