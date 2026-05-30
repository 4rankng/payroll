package timesheet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

type Validator struct {
	TimesheetRepo  domain.TimesheetRepository
	ProjectEmpRepo domain.ProjectEmployeeRepository
	HolidayService HolidayServiceInterface
}

type HolidayServiceInterface interface {
	IsSaturday(date time.Time) bool
	GetDayType(date time.Time) string
}

func NewValidator(
	timesheetRepo domain.TimesheetRepository,
	projectEmpRepo domain.ProjectEmployeeRepository,
	holidayService HolidayServiceInterface,
) *Validator {
	return &Validator{
		TimesheetRepo:  timesheetRepo,
		ProjectEmpRepo: projectEmpRepo,
		HolidayService: holidayService,
	}
}

// DetermineDayType determines the day type based on Vietnam rules
// If user provides DayType, validate and use it; otherwise auto-infer
func (v *Validator) DetermineDayType(date time.Time, userProvidedDayType *string) (string, error) {
	// If user provided day type, validate and use it
	if userProvidedDayType != nil && *userProvidedDayType != "" {
		normalizedDayType := strings.ToLower(*userProvidedDayType)
		if normalizedDayType != "ngày thường" && normalizedDayType != "ngày nghỉ" && normalizedDayType != "ngày lễ" {
			return "", domain.NewValidationError(fmt.Sprintf("Loại ngày '%s' không hợp lệ. Phải là 'ngày thường' (ngày làm việc), 'ngày nghỉ' (ngày nghỉ), hoặc 'ngày lễ' (ngày lễ công cộng)", *userProvidedDayType))
		}
		return normalizedDayType, nil
	}

	// Auto-infer day type if not provided by user
	dayType := v.HolidayService.GetDayType(date)
	// Normalize to lowercase for consistency
	return strings.ToLower(dayType), nil
}

// ParsePaytype extracts position, day type, and hour type from a paytype string
// Format: "position.dayType.hourType"
func (v *Validator) ParsePaytype(paytype string) (position, dayType, hourType string) {
	parts := strings.Split(paytype, ".")
	if len(parts) >= 3 {
		position = parts[0]
		dayType = parts[1]
		hourType = strings.Join(parts[2:], ".") // Handle hour types with dots like "08:00-17:00"
	} else if len(parts) == 2 {
		// Fallback for old format
		position = "phổ thông"
		dayType = parts[0]
		hourType = parts[1]
	} else {
		// Very old or invalid format
		position = "phổ thông"
		dayType = "Ngày thường"
		hourType = paytype
	}
	return
}

// ValidateEmployeeAssignment checks if employee is assigned to project
func (v *Validator) ValidateEmployeeAssignment(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	assignment, err := v.ProjectEmpRepo.GetByProjectAndEmployee(ctx, projectID, employeeID)
	if err != nil || assignment == nil {
		return nil, domain.NewValidationError(fmt.Sprintf("Nhân viên ID %d không được phân công cho Dự án ID %d. Vui lòng đảm bảo nhân viên được phân công cho dự án này trước khi tạo bảng chấm công", employeeID, projectID))
	}
	return assignment, nil
}

// ValidateDailyHoursLimitForUpsert checks if upsert operation would exceed 24 hours per day limit
func (v *Validator) ValidateDailyHoursLimitForUpsert(ctx context.Context, employeeID uint, date time.Time, newHours float64, existingRecord *domain.Timesheet) error {
	// First check: single entry cannot exceed 24 hours
	if newHours > 24.0 {
		return domain.NewValidationError(fmt.Sprintf("Một mục chấm công không thể vượt quá 24 giờ. Đang cố gắng thiết lập: %.2f giờ", newHours))
	}

	// Get all existing timesheets for the employee on the specified date
	timesheets, err := v.TimesheetRepo.GetByEmployee(ctx, employeeID, date, date)
	if err != nil {
		return fmt.Errorf("failed to get existing timesheets for daily hours validation: %w", err)
	}

	// Calculate total hours after the upsert operation
	totalHours := 0.0
	for _, ts := range timesheets {
		if existingRecord != nil && ts.ID == existingRecord.ID {
			// Replace existing hours with new hours (upsert behavior)
			totalHours += newHours
		} else {
			totalHours += ts.HoursWorked
		}
	}

	// If this is a new record (not an update), add the new hours
	if existingRecord == nil {
		totalHours += newHours
	}

	// Check if total would exceed 24 hours
	if totalHours > 24.0 {
		if existingRecord != nil {
			return domain.NewValidationError(fmt.Sprintf("Tổng số giờ làm việc mỗi ngày không thể vượt quá 24 giờ. Tổng hiện tại: %.2f giờ, đang cố gắng cập nhật thành: %.2f giờ (tổng mới: %.2f giờ)",
				totalHours-newHours, newHours, totalHours))
		} else {
			return domain.NewValidationError(fmt.Sprintf("Tổng số giờ làm việc mỗi ngày không thể vượt quá 24 giờ. Tổng hiện tại: %.2f giờ, đang cố gắng thêm: %.2f giờ",
				totalHours-newHours, newHours))
		}
	}

	return nil
}

// ValidateNoDuplicateEntry checks for duplicate timesheet entry
func (v *Validator) ValidateNoDuplicateEntry(ctx context.Context, projectID, employeeID uint, date time.Time, paytype string) error {
	existing, err := v.TimesheetRepo.GetByProjectEmployeeDatePaytype(ctx, projectID, employeeID, date, paytype)
	if err == nil && existing != nil {
		return domain.NewValidationError(fmt.Sprintf("Mục chấm công đã tồn tại cho Nhân viên ID %d, Dự án ID %d, Ngày %s, và Loại thanh toán '%s'. Hãy xem xét cập nhật mục hiện có thay vì tạo mới", employeeID, projectID, date.Format("2006-01-02"), paytype))
	}
	return nil
}

// ValidateOneProjectPerDay checks if employee already has timesheet for a different project on the same date
func (v *Validator) ValidateOneProjectPerDay(ctx context.Context, employeeID, projectID uint, date time.Time) error {
	existingProjects, err := v.TimesheetRepo.GetProjectsForEmployeeOnDate(ctx, employeeID, date)
	if err != nil {
		return fmt.Errorf("failed to get existing projects for employee on date: %w", err)
	}

	// If no existing projects, validation passes
	if len(existingProjects) == 0 {
		return nil
	}

	// Check if any existing project is different from the requested project
	for _, existingProjectID := range existingProjects {
		if existingProjectID != projectID {
			return domain.NewValidationError(fmt.Sprintf(constants.MsgEmployeeAlreadyHasTimesheetVN, employeeID, existingProjectID, date.Format("2006-01-02")))
		}
	}

	return nil
}

// ValidateEmployeeWorkPeriod validates that the timesheet date is within employee's work period for the project
func (v *Validator) ValidateEmployeeWorkPeriod(ctx context.Context, projectID, employeeID uint, date time.Time) error {
	assignment, err := v.ProjectEmpRepo.GetByProjectAndEmployee(ctx, projectID, employeeID)
	if err != nil || assignment == nil {
		return domain.NewValidationError(fmt.Sprintf("Nhân viên ID %d không được phân công cho Dự án ID %d. Vui lòng đảm bảo nhân viên được phân công cho dự án này trước khi tạo bảng chấm công", employeeID, projectID))
	}

	// Use the domain method to check if timesheet can be created on this date
	if !assignment.CanCreateTimesheet(date) {
		// Format date strings for the error message
		startDateStr := assignment.StartDate.Format("2006-01-02")
		endDateStr := "hiện tại"
		if assignment.LastDate != nil {
			endDateStr = assignment.LastDate.Format("2006-01-02")
		}

		return domain.NewValidationError(fmt.Sprintf(constants.MsgTimesheetDateOutsideWorkPeriodVN,
			date.Format("2006-01-02"), employeeID, startDateStr, endDateStr))
	}

	return nil
}
