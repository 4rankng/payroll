package employee

import (
	"context"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// ResponseBuilder handles building employee response DTOs
type ResponseBuilder struct {
	employeeService        *employee.EmployeeService
	timesheetService       *timesheet.TimesheetService
	projectEmployeeService *project.ProjectEmployeeService
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(
	employeeService *employee.EmployeeService,
	timesheetService *timesheet.TimesheetService,
	projectEmployeeService *project.ProjectEmployeeService,
) *ResponseBuilder {
	return &ResponseBuilder{
		employeeService:        employeeService,
		timesheetService:       timesheetService,
		projectEmployeeService: projectEmployeeService,
	}
}

// BuildEmployeeResponse converts an employee to basic response DTO
func (rb *ResponseBuilder) BuildEmployeeResponse(ctx context.Context, emp *domain.Employee) dto.EmployeeResponse {
	var dateOfBirthStr *string
	if emp.DateOfBirth != nil {
		dobStr := emp.DateOfBirth.Format("2006-01-02")
		dateOfBirthStr = &dobStr
	}

	var bankInfo *dto.EmployeeBankInfo
	if emp.Bank != nil {
		bankInfo = &dto.EmployeeBankInfo{
			ID:         emp.Bank.ID,
			BranchName: emp.Bank.BranchName,
		}
	}

	response := dto.EmployeeResponse{
		ID:                emp.ID,
		Fullname:          emp.Fullname,
		Email:             emp.Email,
		CCCD:              emp.CCCD,
		Address:           emp.Address,
		Mobile:            emp.Mobile,
		Bank:              bankInfo,
		BankAccountNumber: emp.BankAccountNumber,
		BankAccountName:   emp.BankAccountName,
		DateOfBirth:       dateOfBirthStr,
		CreatedBy:         emp.CreatedBy,
		CreatedAt:         emp.CreatedAt,
		UpdatedAt:         emp.UpdatedAt,
		CurrentProjects:   []dto.EmployeeProjectInfo{}, // Initialize empty array
	}

	// Get current project assignments if service is available
	if rb.projectEmployeeService != nil {
		filters := domain.ProjectEmployeeFilters{
			EmployeeID: &emp.ID,
			ActiveOnly: true, // Only get active assignments
			SortBy:     "created_at",
			SortOrder:  "desc",
		}

		if assignments, err := rb.projectEmployeeService.ListAssignments(ctx, filters); err == nil {
			var currentProjects []dto.EmployeeProjectInfo

			for _, assignment := range assignments {
				// Only include active assignments (LastDate is nil)
				if assignment.LastDate == nil {
					currentProjects = append(currentProjects, rb.BuildProjectInfo(*assignment))
				}
			}

			response.CurrentProjects = currentProjects
		}
	}

	return response
}

// BuildDetailedEmployeeResponse converts an employee to comprehensive response DTO with related data
func (rb *ResponseBuilder) BuildDetailedEmployeeResponse(ctx context.Context, emp *domain.Employee) (*dto.EmployeeDetailedResponse, error) {
	var dateOfBirthStr *string
	if emp.DateOfBirth != nil {
		dobStr := emp.DateOfBirth.Format("2006-01-02")
		dateOfBirthStr = &dobStr
	}

	var bankInfo *dto.EmployeeBankInfo
	if emp.Bank != nil {
		bankInfo = &dto.EmployeeBankInfo{
			ID:         emp.Bank.ID,
			BranchName: emp.Bank.BranchName,
		}
	}

	log := observability.GetLogger()
	log.Info("Building employee response", "employee_id", emp.ID, "user_id", emp.UserID, "user_nil", emp.User == nil)
	var username *string
	if emp.User != nil {
		username = &emp.User.Username
		log.Info("Found username from preload", "username", emp.User.Username)
	} else if emp.UserID != nil {
		// Preload didn't work, fetch user manually
		if user, err := rb.employeeService.GetUserByID(ctx, *emp.UserID); err == nil {
			username = &user.Username
			log.Info("Found username from manual fetch", "username", user.Username)
		} else {
			log.Error("Failed to fetch user manually", "error", err)
		}
	} else {
		log.Info("employee has no user_id")
	}

	response := &dto.EmployeeDetailedResponse{
		ID:                emp.ID,
		Username:          username,
		Fullname:          emp.Fullname,
		Email:             emp.Email,
		CCCD:              emp.CCCD,
		Address:           emp.Address,
		Mobile:            emp.Mobile,
		Bank:              bankInfo,
		BankAccountNumber: emp.BankAccountNumber,
		BankAccountName:   emp.BankAccountName,
		DateOfBirth:       dateOfBirthStr,
		CreatedBy:         emp.CreatedBy,
		CreatedAt:         emp.CreatedAt,
		UpdatedAt:         emp.UpdatedAt,
	}

	// Get payroll summary
	if rb.employeeService != nil {
		if payrollSummary, err := rb.employeeService.GetEmployeeSummary(ctx, emp.ID); err == nil {
			var lastPaymentDate string
			if payrollSummary.LastPaymentDate != nil {
				lastPaymentDate = payrollSummary.LastPaymentDate.Format("2006-01-02")
			}
			response.PayrollSummary = &dto.EmployeeSummaryResponse{
				TotalPayrollPayments: payrollSummary.TotalPayrollPayments,
				TotalEarningsVND:     payrollSummary.TotalEarningsVND,
				LastPaymentDate:      lastPaymentDate,
				AvgWeeklyEarningsVND: payrollSummary.AvgWeeklyEarningsVND,
			}
		}
	}

	// Get timesheet summary if service is available
	if rb.timesheetService != nil {
		filters := domain.TimesheetFilters{}
		if summary, err := rb.timesheetService.GetEmployeeTimesheetSummary(ctx, emp.ID, filters); err == nil && summary.EmployeeName != "" {
			totalHours := float64(0)
			// Calculate total hours from the map
			for _, hours := range summary.TotalHours {
				totalHours += hours
			}

			lastDate := ""
			if summary.LastEntryDate != nil {
				lastDate = *summary.LastEntryDate
			}

			response.TimesheetSummary = &dto.EmployeeTimesheetSummary{
				TotalTimesheets:    summary.WorkingDays, // Using working days as timesheet count
				TotalHoursWorked:   totalHours,
				PendingTimesheets:  summary.PendingEntries,
				ApprovedTimesheets: summary.ApprovedEntries,
				RejectedTimesheets: 0, // Not available in domain summary
				CurrentWeekHours:   summary.CurrentWeekHours,
				LastTimesheetDate:  lastDate,
			}
		}
	}

	return response, nil
}

// FormatOptionalDate formats a time pointer to YYYY-MM-DD string
// Returns nil if the pointer is nil
func (rb *ResponseBuilder) FormatOptionalDate(date *time.Time) *string {
	if date == nil {
		return nil
	}
	formatted := date.Format("2006-01-02")
	return &formatted
}

// BuildBankInfo creates a bank info DTO from a bank entity
func (rb *ResponseBuilder) BuildBankInfo(bank *domain.Bank) *dto.EmployeeBankInfo {
	if bank == nil {
		return nil
	}
	return &dto.EmployeeBankInfo{
		ID:         bank.ID,
		BranchName: bank.BranchName,
	}
}

// BuildProjectInfo creates a project info DTO from a project employee assignment
func (rb *ResponseBuilder) BuildProjectInfo(assignment domain.ProjectEmployee) dto.EmployeeProjectInfo {
	projectName := ""
	projectCode := ""
	clientName := ""
	if assignment.Project.ID > 0 {
		projectName = assignment.Project.Name
		projectCode = assignment.Project.Code
		clientName = assignment.Project.ClientName
	}

	return dto.EmployeeProjectInfo{
		ProjectID:              assignment.ProjectID,
		ProjectEmployeeID:      assignment.ID,
		Name:                   projectName,
		Code:                   projectCode,
		ClientName:             clientName,
		Position:               assignment.Position,
		StartDate:              assignment.StartDate.Format("2006-01-02"),
		LastDate:               rb.FormatOptionalDate(assignment.LastDate),
		PaymentSchedule:        assignment.PaymentSchedule,
		PendingPaymentSchedule: assignment.PendingPaymentSchedule,
		ScheduleEffectiveFrom:  rb.FormatOptionalDate(assignment.ScheduleEffectiveFrom),
		IsFlexible:             assignment.Project.IsFlexible,
		CheckInEnabled:         assignment.CheckInEnabled,
	}
}
