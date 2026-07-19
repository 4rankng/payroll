package bootstrap

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// walletBulkTimesheetLinker adds row locking to the existing timesheet port
// without broadening the repository contract used throughout the application.
type walletBulkTimesheetLinker struct {
	db   *gorm.DB
	repo domain.TimesheetRepository
}

func (a walletBulkTimesheetLinker) GetByIDsForUpdate(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	if len(ids) == 0 {
		return []*domain.Timesheet{}, nil
	}
	db := a.db.WithContext(ctx)
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		db = txCtx.TX.WithContext(ctx)
	}
	var timesheets []*domain.Timesheet
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Find(&timesheets).Error; err != nil {
		return nil, fmt.Errorf("lock wallet-bulk timesheets: %w", err)
	}
	return timesheets, nil
}

func (a walletBulkTimesheetLinker) BulkUpdateTransactionID(ctx context.Context, transactionID uint, ids []uint) error {
	return a.repo.BulkUpdateTransactionID(ctx, transactionID, ids)
}

func (a walletBulkTimesheetLinker) ClearTransactionID(ctx context.Context, transactionID uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	db := a.db.WithContext(ctx)
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		db = txCtx.TX.WithContext(ctx)
	}
	result := db.Model(&domain.Timesheet{}).
		Where("id IN ? AND transaction_id = ?", ids, transactionID).
		Update("transaction_id", nil)
	if result.Error != nil {
		return fmt.Errorf("clear wallet-bulk timesheet transaction: %w", result.Error)
	}
	if result.RowsAffected != int64(len(ids)) {
		return fmt.Errorf("clear wallet-bulk timesheets: expected %d linked rows, updated %d", len(ids), result.RowsAffected)
	}
	return nil
}
