package services

import (
	"context"
	"fmt"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ValidateDuplicateTimesheet checks for duplicate timesheets
func (s *TimesheetValidationService) ValidateDuplicateTimesheet(ctx context.Context, timesheet *domain.Timesheet) error {
	existing, err := s.timesheetRepo.GetByProjectEmployeeDatePaytype(ctx, timesheet.ProjectID, timesheet.EmployeeID, timesheet.Date, timesheet.PayType)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			return domain.NewInternalError("Lỗi kiểm tra bảng chấm công trùng lặp", err)
		}
		// Not found is expected - no duplicate exists
		return nil
	}

	// If existing timesheet found and it's not the same one being updated
	if existing != nil && existing.ID != timesheet.ID {
		// Get employee name for proper error format
		employee, err := s.employeeRepo.GetByID(ctx, timesheet.EmployeeID)
		employeeName := "Unknown"
		if err == nil && employee != nil {
			employeeName = employee.Fullname
		}

		// Parse the PayType to extract dayType and hourType
		dayType := "Ngày thường" // Default
		hourType := "ca ngày"    // Default

		// Try to extract dayType and hourType from PayType
		parsedDayType, parsedHourType := s.parsePaytype(timesheet.PayType)
		if parsedDayType != "" {
			dayType = parsedDayType
		}
		if parsedHourType != "" {
			hourType = parsedHourType
		}

		// Build error message in required format: "Công %s %s của %s ngày YYYY-MM-DD đã tồn tại"
		errorMessage := fmt.Sprintf("Công %s %s của %s ngày %s đã tồn tại",
			dayType, hourType, employeeName, timesheet.Date.Format("2006-01-02"))

		return domain.NewConflictError(errorMessage)
	}

	return nil
}

// CanApproveTimesheet checks if timesheet can be approved
func (s *TimesheetValidationService) CanApproveTimesheet(ctx context.Context, timesheetID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewNotFoundError("Không tìm thấy bảng chấm công")
		}
		return domain.NewInternalError("Lỗi lấy thông tin bảng chấm công", err)
	}

	if timesheet.Status != domain.TimesheetStatusPendingApproval {
		return domain.NewValidationError("timesheet is not pending approval")
	}

	return nil
}

// CanRejectTimesheet checks if timesheet can be rejected
func (s *TimesheetValidationService) CanRejectTimesheet(ctx context.Context, timesheetID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewNotFoundError("Không tìm thấy bảng chấm công")
		}
		return domain.NewInternalError("Lỗi lấy thông tin bảng chấm công", err)
	}

	if timesheet.Status != domain.TimesheetStatusPendingApproval {
		return domain.NewValidationError("timesheet is not pending approval")
	}

	return nil
}

// ValidateTimesheetNotPaid checks if timesheet has not been paid, failed, or cancelled
func (s *TimesheetValidationService) ValidateTimesheetNotPaid(ctx context.Context, timesheetID uint) error {
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewNotFoundError("Không tìm thấy bảng chấm công")
		}
		return domain.NewInternalError("Lỗi lấy thông tin bảng chấm công", err)
	}

	if timesheet.IsPaid() {
		errorMessage := s.GenerateDetailedPaidTimesheetError(ctx, timesheet)
		return domain.NewValidationError(errorMessage)
	}

	return nil
}

// GenerateDetailedPaidTimesheetError creates a detailed error message for paid timesheet validation
func (s *TimesheetValidationService) GenerateDetailedPaidTimesheetError(ctx context.Context, timesheet *domain.Timesheet) string {
	// Get employee name for detailed error message
	employeeName := "Unknown"
	if employee, err := s.employeeRepo.GetByID(ctx, timesheet.EmployeeID); err == nil && employee != nil {
		employeeName = employee.Fullname
	}

	// Parse the PayType to extract dayType and hourType
	dayType := "Ngày thường" // Default
	hourType := "ca ngày"    // Default

	parsedDayType, parsedHourType := s.parsePaytype(timesheet.PayType)
	if parsedDayType != "" {
		dayType = parsedDayType
	}
	if parsedHourType != "" {
		hourType = parsedHourType
	}

	// Get project name for detailed error message
	projectName := "Unknown"
	if project, err := s.projectRepo.GetByID(ctx, timesheet.ProjectID); err == nil && project != nil {
		projectName = project.Name
	}

	// Build detailed error message
	errorMessage := fmt.Sprintf(constants.MsgCannotEditPaidTimesheetDetailedVN,
		dayType, hourType, timesheet.Date.Format("2006-01-02"), employeeName, projectName)

	return errorMessage
}
