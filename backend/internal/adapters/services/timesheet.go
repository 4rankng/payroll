package services

import (
	"context"
	"time"

	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
)

// TimesheetAdapter implements serviceports.TimesheetPort using existing TimesheetService
type TimesheetAdapter struct {
	service *timesheet.TimesheetService
}

// NewTimesheetAdapter creates a new timesheet service adapter
func NewTimesheetAdapter(service *timesheet.TimesheetService) serviceports.TimesheetPort {
	return &TimesheetAdapter{
		service: service,
	}
}

// Create creates a new timesheet
func (a *TimesheetAdapter) Create(ctx context.Context, ts *domain.Timesheet) error {
	// Extract hour type and day type from paytype if available
	// Default values if not provided
	hourType := "full"
	dayType := "weekday"

	// Note: The existing service requires createdBy and userRole parameters
	// These should ideally come from the context or be part of the timesheet entity
	createdBy := ts.CreatedBy

	_, err := a.service.CreateTimesheet(ctx, ts, hourType, dayType, createdBy, "")
	return err
}

// Update updates a timesheet
func (a *TimesheetAdapter) Update(ctx context.Context, ts *domain.Timesheet) error {
	// Use CreatedBy as the updater if no specific updater is provided
	updatedBy := ts.CreatedBy

	return a.service.UpdateTimesheet(ctx, ts.ID, ts, updatedBy)
}

// GetByID retrieves a timesheet by ID
func (a *TimesheetAdapter) GetByID(ctx context.Context, id uint) (*domain.Timesheet, error) {
	return a.service.GetTimesheet(ctx, id)
}

// GetByIDs retrieves multiple timesheets by their IDs
func (a *TimesheetAdapter) GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	// The existing service doesn't have GetByIDs
	// We need to fetch them individually or add bulk fetch to the service
	result := make([]*domain.Timesheet, 0, len(ids))
	for _, id := range ids {
		ts, err := a.service.GetTimesheet(ctx, id)
		if err != nil {
			if domain.IsNotFoundError(err) {
				continue
			}
			return nil, err
		}
		result = append(result, ts)
	}
	return result, nil
}

// List retrieves timesheets with filters
func (a *TimesheetAdapter) List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return a.service.ListTimesheets(ctx, filters)
}

// Approve approves a timesheet
func (a *TimesheetAdapter) Approve(ctx context.Context, id uint, approvedBy uint) error {
	return a.service.ApproveTimesheet(ctx, id, approvedBy)
}

// BulkApprove approves multiple timesheets
func (a *TimesheetAdapter) BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error {
	_, err := a.service.BulkApprove(ctx, ids, approvedBy)
	return err
}

// Reject rejects a timesheet
func (a *TimesheetAdapter) Reject(ctx context.Context, id uint, rejectionReason string) error {
	// The existing service requires rejectedBy parameter
	// For now, use a default system user ID (0)
	return a.service.RejectTimesheet(ctx, id, rejectionReason, 0)
}

// Delete deletes a timesheet
func (a *TimesheetAdapter) Delete(ctx context.Context, id uint) error {
	// The existing service requires deletedBy parameter
	// For now, use a default system user ID (0)
	return a.service.DeleteTimesheet(ctx, id, 0)
}

// GetByEmployee retrieves timesheets for an employee in a date range
func (a *TimesheetAdapter) GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return a.service.GetTimesheetsByEmployee(ctx, employeeID, fromDate, toDate)
}

// BulkUpdatePaymentStatus updates payment status for multiple timesheets
func (a *TimesheetAdapter) BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	// The existing service doesn't have this specific method
	// This would need to be implemented in the timesheet service
	// For now, return not implemented
	return domain.NewNotFoundError("BulkUpdatePaymentStatus not implemented in timesheet service")
}
