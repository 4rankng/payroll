package repositories

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

// TimesheetCommandRepository handles write operations for timesheets
type TimesheetCommandRepository struct {
	db             *gorm.DB
	batchProcessor *common.BatchProcessor
	errorHandler   *common.RepoErrorHandler
}

// NewTimesheetCommandRepository creates a new command repository
func NewTimesheetCommandRepository(db *gorm.DB) *TimesheetCommandRepository {
	return &TimesheetCommandRepository{
		db:             db,
		batchProcessor: common.NewBatchProcessor(common.DefaultBatchConfig()),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

// Create creates a new timesheet
func (r *TimesheetCommandRepository) Create(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.getDB(ctx).Create(timesheet).Error
}

// Update updates an existing timesheet
func (r *TimesheetCommandRepository) Update(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.getDB(ctx).Save(timesheet).Error
}

// Delete soft deletes a timesheet
func (r *TimesheetCommandRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Timesheet{}, id).Error
}

// HardDelete permanently removes a timesheet (no soft-delete).
// Used by BCC import "latest wins" overwrite to avoid accumulation of stale soft-deleted rows.
func (r *TimesheetCommandRepository) HardDelete(ctx context.Context, id uint) error {
	result := r.getDB(ctx).
		Unscoped().
		Where("id = ? AND timesheet_status <> ? AND payment_status = ?",
			id, domain.TimesheetStatusApproved, domain.PaymentStatusPending).
		Delete(&domain.Timesheet{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("timesheet %d became protected during replacement", id)
	}
	return nil
}

// BulkCreate creates multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkCreate(ctx context.Context, timesheets []*domain.Timesheet) error {
	if len(timesheets) == 0 {
		return nil
	}

	return r.getDB(ctx).CreateInBatches(timesheets, 100).Error
}

// BulkUpdate updates multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkUpdate(ctx context.Context, timesheets []*domain.Timesheet) error {
	if len(timesheets) == 0 {
		return nil
	}

	// Updates are more expensive than reads, so use smaller batches
	updateConfig := common.BatchConfig{
		DefaultSize:    100,
		MaxSize:        100,
		QuerySizeLimit: 32768,
	}
	updateProcessor := common.NewBatchProcessor(updateConfig)

	return updateProcessor.ProcessInBatches(ctx, timesheets, func(batch interface{}) error {
		batchTimesheets := batch.([]*domain.Timesheet)

		// Use transaction for batch consistency
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, timesheet := range batchTimesheets {
				if err := tx.Save(timesheet).Error; err != nil {
					return err
				}
			}
			return nil
		})
	})
}

// Approve approves a timesheet
func (r *TimesheetCommandRepository) Approve(ctx context.Context, id uint, approvedBy uint) error {
	now := clock.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusApproved,
			"approved_by":      approvedBy,
			"approved_at":      now,
			"rejection_reason": "",
			"allowed_edit":     false,
			"request_edit_id":  nil,
		}).Error
}

// BulkApprove approves multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error {
	if len(ids) == 0 {
		return nil
	}

	now := clock.Now()

	// Use BatchProcessor for batch approval
	return r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return r.getDB(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Updates(map[string]interface{}{
				"timesheet_status": domain.TimesheetStatusApproved,
				"approved_by":      approvedBy,
				"approved_at":      now,
				"rejection_reason": "",
				"allowed_edit":     false,
				"request_edit_id":  nil,
			}).Error
	})
}

// BulkReject rejects multiple timesheets in batches
func (r *TimesheetCommandRepository) BulkReject(ctx context.Context, ids []uint, rejectionReason string) error {
	if len(ids) == 0 {
		return nil
	}

	return r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return r.db.WithContext(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Updates(map[string]interface{}{
				"timesheet_status": domain.TimesheetStatusRejected,
				"rejection_reason": rejectionReason,
				"approved_by":      nil,
				"approved_at":      nil,
				"allowed_edit":     false,
				"request_edit_id":  nil,
			}).Error
	})
}

// Reject rejects a timesheet
func (r *TimesheetCommandRepository) Reject(ctx context.Context, id uint, rejectionReason string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusRejected,
			"rejection_reason": rejectionReason,
			"approved_by":      nil,
			"approved_at":      nil,
			"allowed_edit":     false,
			"request_edit_id":  nil,
		}).Error
}

// Reset resets a timesheet to pending approval status
func (r *TimesheetCommandRepository) Reset(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"timesheet_status": domain.TimesheetStatusPendingApproval,
			"approved_by":      nil,
			"approved_at":      nil,
			"rejection_reason": "",
			"force_payroll":    false,
			"request_edit_id":  nil,
		}).Error
}

// SetForcePayroll sets the force_payroll flag on a timesheet
func (r *TimesheetCommandRepository) SetForcePayroll(ctx context.Context, id uint, flag bool) error {
	return r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Update("force_payroll", flag).Error
}

// BulkUpdatePaymentStatus updates payment status for multiple timesheets in a single transaction
func (r *TimesheetCommandRepository) BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			updateFields := map[string]interface{}{
				"payment_status": update.PaymentStatus,
			}

			if update.PaymentReference != nil {
				updateFields["payment_reference"] = *update.PaymentReference
			}

			if update.PaymentDate != nil {
				updateFields["payment_date"] = *update.PaymentDate
			}

			if update.PaidAmount != nil {
				updateFields["paid_amount"] = *update.PaidAmount
			}

			if update.PaidAt != nil {
				updateFields["paid_at"] = *update.PaidAt
			} else if update.PaymentStatus == domain.PaymentStatusPaid {
				// Automatically set paid_at to current time when status is set to paid
				updateFields["paid_at"] = clock.Now()
			} else {
				// Clear paid_at when status is not paid
				updateFields["paid_at"] = nil
			}

			if err := tx.Model(&domain.Timesheet{}).
				Where("id = ?", update.TimesheetID).
				Updates(updateFields).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchUpdatePaymentStatusToFailed marks timesheets as failed
// Must be called within a transaction
func (r *TimesheetCommandRepository) BatchUpdatePaymentStatusToFailed(ctx context.Context, tx interface{}, timesheetIDs []uint) error {
	db, err := common.ExtractDB(tx)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	if len(timesheetIDs) == 0 {
		return nil
	}

	return db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id IN ?", timesheetIDs).
		Update("payment_status", domain.PaymentStatusFailed).Error
}

// BulkUpdateRevenuePaid marks revenue as paid for specified timesheets
func (r *TimesheetCommandRepository) BulkUpdateRevenuePaid(ctx context.Context, timesheetIDs []uint) error {
	if len(timesheetIDs) == 0 {
		return nil
	}

	// Use BatchProcessor to process in batches
	return r.batchProcessor.ProcessInBatches(ctx, timesheetIDs, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return r.db.WithContext(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Update("revenue_paid", true).Error
	})
}

// BulkUpdateRevenueReceivable updates revenue_receivable for specified timesheets using batch update
func (r *TimesheetCommandRepository) BulkUpdateRevenueReceivable(ctx context.Context, updates map[uint]int64) error {
	if len(updates) == 0 {
		return nil
	}

	// For small batches, individual updates might be faster, but for consistency and performance with larger batches,
	// use a single bulk update with CASE WHEN statement
	if len(updates) <= 10 {
		// Use individual updates for very small batches to avoid query complexity overhead
		for timesheetID, revenueReceivable := range updates {
			if err := r.db.WithContext(ctx).
				Model(&domain.Timesheet{}).
				Where("id = ?", timesheetID).
				Update("revenue_receivable", revenueReceivable).Error; err != nil {
				return err
			}
		}
		return nil
	}

	// Build CASE WHEN statement for bulk update
	cases := make([]string, 0, len(updates))
	ids := make([]uint, 0, len(updates))

	for id, value := range updates {
		cases = append(cases, fmt.Sprintf("WHEN id = %d THEN %d", id, value))
		ids = append(ids, id)
	}

	// Execute single bulk update query
	query := fmt.Sprintf(`
		UPDATE timesheets
		SET revenue_receivable = CASE %s ELSE revenue_receivable END
		WHERE id IN ?`,
		strings.Join(cases, " "))

	return r.db.WithContext(ctx).Raw(query, ids).Error
}

// BulkUpdateTransactionID links timesheets to a single transaction
func (r *TimesheetCommandRepository) BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error {
	if len(timesheetIDs) == 0 {
		return nil
	}

	// Get DB connection (respect existing transaction context)
	db := r.getDB(ctx)

	// Use BatchProcessor to process in batches
	return r.batchProcessor.ProcessInBatches(ctx, timesheetIDs, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		return db.Model(&domain.Timesheet{}).
			Where("id IN ?", batchIDs).
			Update("transaction_id", transactionID).Error
	})
}

// getDB gets the appropriate DB instance from context or falls back to default
func (r *TimesheetCommandRepository) getDB(ctx context.Context) *gorm.DB {
	// Check if there's a transaction context
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}
