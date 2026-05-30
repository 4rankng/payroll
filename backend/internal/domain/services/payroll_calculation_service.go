package services

import (
	"context"
	"time"

	"api-server/internal/domain"
)

// PayrollCalculationService handles business logic for payroll calculations
type PayrollCalculationService struct {
	timesheetRepo domain.TimesheetRepository
	payrateRepo   domain.PayrateRepository
	employeeRepo  domain.EmployeeRepository
}

// NewPayrollCalculationService creates a new payroll calculation service
func NewPayrollCalculationService(
	timesheetRepo domain.TimesheetRepository,
	payrateRepo domain.PayrateRepository,
	employeeRepo domain.EmployeeRepository,
) *PayrollCalculationService {
	return &PayrollCalculationService{
		timesheetRepo: timesheetRepo,
		payrateRepo:   payrateRepo,
		employeeRepo:  employeeRepo,
	}
}

// CalculateEmployeeSalary calculates total approved salary for an employee in a date range
func (s *PayrollCalculationService) CalculateEmployeeSalary(ctx context.Context, employeeID uint, fromDate, toDate time.Time) (int64, error) {
	timesheets, err := s.timesheetRepo.List(ctx, domain.TimesheetFilters{
		EmployeeID:      &employeeID,
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
	})
	if err != nil {
		return 0, err
	}

	var totalAmount int64
	for _, ts := range timesheets {
		totalAmount += ts.Amount
	}
	return totalAmount, nil
}

// CalculateProjectCost calculates total approved cost for a project in a date range
func (s *PayrollCalculationService) CalculateProjectCost(ctx context.Context, projectID uint, fromDate, toDate time.Time) (int64, error) {
	timesheets, err := s.timesheetRepo.List(ctx, domain.TimesheetFilters{
		ProjectIDs:      []uint{projectID},
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
	})
	if err != nil {
		return 0, err
	}

	var totalCost int64
	for _, ts := range timesheets {
		totalCost += ts.Amount
	}
	return totalCost, nil
}

// GetPayrollSummary generates payroll summary for a period
func (s *PayrollCalculationService) GetPayrollSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.PayrollSummary, error) {
	// Get all approved timesheets in the period
	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
	}

	timesheets, err := s.timesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	summary := &domain.PayrollSummary{
		FromDate:      fromDate,
		ToDate:        toDate,
		TotalAmount:   0,
		TotalHours:    0,
		EmployeeCount: make(map[uint]struct{}),
		ProjectCosts:  make(map[uint]int64),
	}

	for _, timesheet := range timesheets {
		summary.TotalAmount += timesheet.Amount
		summary.TotalHours += timesheet.HoursWorked
		summary.EmployeeCount[timesheet.EmployeeID] = struct{}{}
		summary.ProjectCosts[timesheet.ProjectID] += timesheet.Amount
	}

	return summary, nil
}

// ValidatePayrollPeriod validates if payroll can be generated for given period
func (s *PayrollCalculationService) ValidatePayrollPeriod(ctx context.Context, fromDate, toDate time.Time) error {
	// Check if there are any pending timesheets in the period
	count, err := s.timesheetRepo.GetPendingApprovalCount(ctx, fromDate, toDate)
	if err != nil {
		return err
	}

	if count > 0 {
		return domain.NewValidationError("cannot generate payroll while there are pending timesheet approvals")
	}

	return nil
}

// CalculateTimesheetAmount calculates the amount for a timesheet based on hours and payrate
func (s *PayrollCalculationService) CalculateTimesheetAmount(ctx context.Context, timesheet *domain.Timesheet) (int64, error) {
	// Get the active payrate for the project on the timesheet date
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, timesheet.ProjectID, timesheet.Date)
	if err != nil {
		return 0, domain.NewValidationError("không tìm thấy cấu hình mức lương hiệu lực cho dự án vào ngày này")
	}

	// Ensure payrate is not nil
	if payrate == nil {
		return 0, domain.NewValidationError("không tìm thấy cấu hình mức lương hiệu lực cho dự án vào ngày này")
	}

	// Get the specific rate for the pay type from the payrate configuration
	rate, err := payrate.Payrate.GetRate(timesheet.PayType)
	if err != nil {
		return 0, domain.NewValidationError("không tìm thấy loại lương trong cấu hình mức lương")
	}

	// Set payrate fields on the timesheet (required for foreign key constraint)
	timesheet.PayrateID = payrate.ID
	timesheet.PayRate = int64(rate)

	// Calculate amount: hours * rate
	amount := int64(timesheet.HoursWorked * float64(rate))
	return amount, nil
}
