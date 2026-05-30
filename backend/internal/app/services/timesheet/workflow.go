package timesheet

import (
	"context"
	"fmt"

	"api-server/internal/domain"
)

type Workflow struct {
	TimesheetRepo domain.TimesheetRepository
}

func NewWorkflow(
	timesheetRepo domain.TimesheetRepository,
) *Workflow {
	return &Workflow{
		TimesheetRepo: timesheetRepo,
	}
}

func (w *Workflow) ApproveTimesheet(ctx context.Context, id uint, approvedBy uint) error {
	if err := w.TimesheetRepo.Approve(ctx, id, approvedBy); err != nil {
		return fmt.Errorf("failed to approve timesheet: %w", err)
	}

	return nil
}

func (w *Workflow) RejectTimesheet(ctx context.Context, id uint, rejectionReason string, rejectedBy uint) error {
	if err := w.TimesheetRepo.Reject(ctx, id, rejectionReason); err != nil {
		return fmt.Errorf("failed to reject timesheet: %w", err)
	}

	return nil
}
