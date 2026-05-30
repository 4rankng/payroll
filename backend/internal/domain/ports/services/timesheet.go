package services

import (
	"context"
	"time"

	"api-server/internal/domain"
)

// TimesheetPort defines the interface for cross-domain timesheet service communication
type TimesheetPort interface {
	Create(ctx context.Context, timesheet *domain.Timesheet) error
	Update(ctx context.Context, timesheet *domain.Timesheet) error
	GetByID(ctx context.Context, id uint) (*domain.Timesheet, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error)
	List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error)
	Approve(ctx context.Context, id uint, approvedBy uint) error
	BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error
	Reject(ctx context.Context, id uint, rejectionReason string) error
	Delete(ctx context.Context, id uint) error
	GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error)
	BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error
}
