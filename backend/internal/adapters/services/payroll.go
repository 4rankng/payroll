package services

import (
	"context"
	"time"

	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
)

// PayrollAdapter implements serviceports.PayrollPort using existing services
// Note: The PayrollPort is mainly for cross-domain calculations, which are handled by TimesheetService
type PayrollAdapter struct {
	timesheetService *timesheet.TimesheetService
}

// NewPayrollAdapter creates a new payroll service adapter
func NewPayrollAdapter(timesheetService *timesheet.TimesheetService) serviceports.PayrollPort {
	return &PayrollAdapter{
		timesheetService: timesheetService,
	}
}

// CalculateSalary calculates salary for an employee in a date range
func (a *PayrollAdapter) CalculateSalary(ctx context.Context, employeeID uint, fromDate, toDate time.Time) (int64, error) {
	// Get employee timesheet summary which includes calculated pay
	filters := domain.TimesheetFilters{
		EmployeeID:      &employeeID,
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
	}

	summary, err := a.timesheetService.GetEmployeeTimesheetSummary(ctx, employeeID, filters)
	if err != nil {
		return 0, err
	}

	return summary.TotalAmount, nil
}

// GetPendingSalary gets total pending salary for a date range
func (a *PayrollAdapter) GetPendingSalary(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusPendingApproval},
	}

	// Get employee summary instead of stats
	// Note: This is a workaround since TimesheetSummaryStats doesn't have amount info
	// In reality, we'd need to sum amounts from the timesheet list
	timesheets, err := a.timesheetService.ListTimesheets(ctx, filters)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, ts := range timesheets {
		total += ts.Amount
	}

	return total, nil
}

// GetPaidSalary gets total paid salary for a date range
func (a *PayrollAdapter) GetPaidSalary(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
	}

	// Get employee summary instead of stats
	// Note: This is a workaround since TimesheetSummaryStats doesn't have amount info
	// In reality, we'd need to sum amounts from the timesheet list
	timesheets, err := a.timesheetService.ListTimesheets(ctx, filters)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, ts := range timesheets {
		total += ts.Amount
	}

	return total, nil
}

// ProcessPayroll processes payroll for a date range
func (a *PayrollAdapter) ProcessPayroll(ctx context.Context, fromDate, toDate time.Time) error {
	// Get all pending timesheets in the date range
	filters := domain.TimesheetFilters{
		FromDate:        &fromDate,
		ToDate:          &toDate,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusPendingApproval},
	}

	timesheets, err := a.timesheetService.ListTimesheets(ctx, filters)
	if err != nil {
		return err
	}

	// Extract IDs
	ids := make([]uint, len(timesheets))
	for i, ts := range timesheets {
		ids[i] = ts.ID
	}

	// Bulk approve (using system user ID 0)
	if len(ids) > 0 {
		_, err = a.timesheetService.BulkApprove(ctx, ids, 0)
		return err
	}

	return nil
}
