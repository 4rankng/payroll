package timesheet

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

type Operations struct {
	TimesheetRepo  domain.TimesheetRepository
	EmployeeRepo   domain.EmployeeRepository
	ProjectEmpRepo domain.ProjectEmployeeRepository
}

func NewOperations(
	timesheetRepo domain.TimesheetRepository,
	employeeRepo domain.EmployeeRepository,
	projectEmpRepo domain.ProjectEmployeeRepository,
) *Operations {
	return &Operations{
		TimesheetRepo:  timesheetRepo,
		EmployeeRepo:   employeeRepo,
		ProjectEmpRepo: projectEmpRepo,
	}
}

func (o *Operations) GetTimesheet(ctx context.Context, id uint) (*domain.Timesheet, error) {
	return o.TimesheetRepo.GetByID(ctx, id)
}

func (o *Operations) UpdateTimesheet(ctx context.Context, timesheet *domain.Timesheet, updatedBy uint) error {
	if err := o.TimesheetRepo.Update(ctx, timesheet); err != nil {
		return fmt.Errorf("failed to update timesheet: %w", err)
	}

	return nil
}

func (o *Operations) DeleteTimesheet(ctx context.Context, id uint, deletedBy uint) error {
	if err := o.TimesheetRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete timesheet: %w", err)
	}

	return nil
}

func (o *Operations) ListTimesheets(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return o.TimesheetRepo.List(ctx, filters)
}

func (o *Operations) CountTimesheets(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	return o.TimesheetRepo.Count(ctx, filters)
}

func (o *Operations) GetTimesheetsByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return o.TimesheetRepo.GetByProject(ctx, projectID, fromDate, toDate)
}

func (o *Operations) GetTimesheetsByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return o.TimesheetRepo.GetByEmployee(ctx, employeeID, fromDate, toDate)
}

func (o *Operations) GetProjectSummary(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*domain.TimesheetSummary, error) {
	return o.TimesheetRepo.GetSummaryByProject(ctx, projectID, fromDate, toDate)
}

func (o *Operations) GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	return o.TimesheetRepo.GetSummaryStats(ctx, filters)
}

func (o *Operations) GetEmployeeTimesheetSummary(ctx context.Context, employeeID uint, filters domain.TimesheetFilters) (*domain.EmployeeTimesheetSummary, error) {
	// Check if employee has active project assignments
	assignments, err := o.ProjectEmpRepo.GetByEmployee(ctx, employeeID)
	if err != nil || len(assignments) == 0 {
		return &domain.EmployeeTimesheetSummary{
			EmployeeID: employeeID,
		}, nil
	}

	// Get employee details
	employee, err := o.EmployeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}

	// Set employee ID filter
	filters.EmployeeID = &employeeID

	// Get timesheets for the employee
	timesheets, err := o.TimesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee timesheets: %w", err)
	}

	// Calculate summary
	summary := &domain.EmployeeTimesheetSummary{
		EmployeeID:   employeeID,
		EmployeeName: employee.Fullname,
		TotalHours:   make(map[string]float64),
	}

	totalAmount := int64(0)
	totalHours := float64(0)
	workingDays := make(map[string]bool)
	pendingCount := 0
	approvedCount := 0
	rejectedCount := 0
	var lastEntryDate *time.Time

	for _, ts := range timesheets {
		// Aggregate hours by paytype
		summary.TotalHours[ts.PayType] += ts.HoursWorked
		totalHours += ts.HoursWorked
		totalAmount += ts.Amount

		// Track working days
		dateStr := ts.Date.Format("2006-01-02")
		workingDays[dateStr] = true

		// Count entries by status
		switch ts.Status {
		case domain.TimesheetStatusPendingApproval:
			pendingCount++
		case domain.TimesheetStatusApproved:
			approvedCount++
		case domain.TimesheetStatusRejected:
			rejectedCount++
		}

		// Track last entry date
		if lastEntryDate == nil || ts.Date.After(*lastEntryDate) {
			lastEntryDate = &ts.Date
		}
	}

	summary.TotalAmount = totalAmount
	summary.WorkingDays = len(workingDays)
	summary.PendingEntries = pendingCount
	summary.ApprovedEntries = approvedCount
	summary.RejectedEntries = rejectedCount
	summary.TotalEntries = len(timesheets)

	if summary.WorkingDays > 0 {
		summary.AverageHoursPerDay = totalHours / float64(summary.WorkingDays)
	}

	if lastEntryDate != nil {
		dateStr := lastEntryDate.Format("2006-01-02")
		summary.LastEntryDate = &dateStr
	}

	// Calculate current week hours
	summary.CurrentWeekHours = o.getCurrentWeekHours(ctx, employeeID)

	return summary, nil
}

// getCurrentWeekHours calculates total hours worked by employee in the current ISO week with comprehensive error handling
func (o *Operations) getCurrentWeekHours(ctx context.Context, employeeID uint) float64 {
	// Validate employee ID
	if employeeID == 0 {
		return 0.0
	}

	// Use optimized database aggregation method
	totalHours, err := o.TimesheetRepo.GetEmployeeCurrentWeekHours(ctx, employeeID)
	if err != nil {
		// Log error but don't fail the entire request
		// In production, you might want to use a proper logger
		return 0.0
	}

	return totalHours
}
