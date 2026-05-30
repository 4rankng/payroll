package timesheet

import (
	"context"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

type BulkOperations struct {
	TimesheetRepo      domain.TimesheetRepository
	TransactionManager domain.TransactionManager
}

func NewBulkOperations(
	timesheetRepo domain.TimesheetRepository,
	transactionManager domain.TransactionManager,
) *BulkOperations {
	return &BulkOperations{
		TimesheetRepo:      timesheetRepo,
		TransactionManager: transactionManager,
	}
}

func (b *BulkOperations) BulkApprove(ctx context.Context, timesheetIDs []uint, approvedBy uint, approvalNote string) (*domain.BulkOperationResult, error) {
	if len(timesheetIDs) == 0 {
		return &domain.BulkOperationResult{
			Results: make([]domain.BulkItemResult, 0),
		}, nil
	}

	// Wrap entire operation in a transaction for atomicity
	resultInterface, err := b.TransactionManager.WithTransactionResult(ctx, func(txCtx context.Context) (interface{}, error) {
		result := &domain.BulkOperationResult{
			Results: make([]domain.BulkItemResult, 0, len(timesheetIDs)),
		}

		// Batch fetch all timesheets in one query
		timesheets, err := b.TimesheetRepo.GetByIDs(txCtx, timesheetIDs)
		if err != nil {
			return nil, err
		}

		// Create a map for quick lookup
		timesheetMap := make(map[uint]*domain.Timesheet, len(timesheets))
		for _, ts := range timesheets {
			timesheetMap[ts.ID] = ts
		}

		// Process each timesheet and collect IDs to approve
		var idsToApprove []uint
		for _, id := range timesheetIDs {
			timesheet, exists := timesheetMap[id]
			if !exists {
				result.Failed++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:    id,
					Error: constants.MsgTimesheetNotFoundVN,
				})
				continue
			}

			// Skip already approved timesheets instead of treating them as errors
			if timesheet.IsApproved() {
				result.Skipped++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:     id,
					Status: string(domain.TimesheetStatusApproved),
				})
				continue
			}

			if !timesheet.CanBeApproved() {
				result.Failed++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:    id,
					Error: constants.MsgTimesheetCannotBeApprovedVN,
				})
				continue
			}

			// Mark for approval
			if err := timesheet.Approve(approvedBy); err != nil {
				result.Failed++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:    id,
					Error: constants.MsgTimesheetApprovalFailedVN,
				})
				continue
			}

			idsToApprove = append(idsToApprove, id)
			result.Results = append(result.Results, domain.BulkItemResult{
				ID:     id,
				Status: string(domain.TimesheetStatusApproved),
			})
		}

		// Batch update all approved timesheets in a single query
		if len(idsToApprove) > 0 {
			if err := b.TimesheetRepo.BulkApprove(txCtx, idsToApprove, approvedBy); err != nil {
				return nil, err // Transaction will rollback
			}
			result.Approved = len(idsToApprove)
		}

		return result, nil
	})

	if err != nil {
		return nil, err
	}

	result := resultInterface.(*domain.BulkOperationResult)
	return result, nil
}

func (b *BulkOperations) BulkReject(ctx context.Context, timesheetIDs []uint, rejectionReason string, rejectedBy uint, notifyPartners bool) (*domain.BulkOperationResult, error) {
	if len(timesheetIDs) == 0 {
		return &domain.BulkOperationResult{
			Results: make([]domain.BulkItemResult, 0),
		}, nil
	}

	// Wrap entire operation in a transaction for atomicity
	resultInterface, err := b.TransactionManager.WithTransactionResult(ctx, func(txCtx context.Context) (interface{}, error) {
		result := &domain.BulkOperationResult{
			Results: make([]domain.BulkItemResult, 0, len(timesheetIDs)),
		}

		// Batch-fetch all timesheets in one query
		timesheets, err := b.TimesheetRepo.GetByIDs(txCtx, timesheetIDs)
		if err != nil {
			return nil, err
		}

		// Build map for O(1) lookup
		timesheetMap := make(map[uint]*domain.Timesheet, len(timesheets))
		for _, ts := range timesheets {
			timesheetMap[ts.ID] = ts
		}

		// Validate and collect IDs to reject
		var idsToReject []uint
		for _, id := range timesheetIDs {
			timesheet, exists := timesheetMap[id]
			if !exists {
				result.Failed++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:    id,
					Error: constants.MsgTimesheetNotFoundVN,
				})
				continue
			}

			if !timesheet.CanBeRejected() {
				result.Failed++
				result.Results = append(result.Results, domain.BulkItemResult{
					ID:    id,
					Error: constants.MsgTimesheetCannotBeRejectedVN,
				})
				continue
			}

			idsToReject = append(idsToReject, id)
			result.Results = append(result.Results, domain.BulkItemResult{
				ID:              id,
				Status:          string(domain.TimesheetStatusRejected),
				RejectionReason: rejectionReason,
			})
		}

		// Batch-update all rejected timesheets in a single query
		if len(idsToReject) > 0 {
			if err := b.TimesheetRepo.BulkReject(txCtx, idsToReject, rejectionReason); err != nil {
				return nil, err // Transaction will rollback
			}
			result.Rejected = len(idsToReject)
		}

		return result, nil
	})

	if err != nil {
		return nil, err
	}

	result := resultInterface.(*domain.BulkOperationResult)

	if notifyPartners {
		result.NotificationsSent = result.Rejected
	}

	return result, nil
}
