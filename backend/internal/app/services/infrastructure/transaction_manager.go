package infrastructure

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"gorm.io/gorm"
)

// TransactionManager provides centralized transaction management for service orchestration
type TransactionManager struct {
	db *gorm.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

// ExecuteInTransaction executes multiple operations within a single transaction.
// The transaction is also injected into the context via domain.WithTransactionContext so
// that downstream repositories and domain services that read the tx from context (using
// domain.GetTransactionFromContext) participate in the same transaction. Callbacks
// registered via domain.RegisterAfterCommit run after the transaction commits.
func (tm *TransactionManager) ExecuteInTransaction(ctx context.Context, operations func(tx *gorm.DB) error) error {
	txCtx := &domain.TransactionContext{
		IsTransactional: true,
	}
	// Bind the transaction context to ctx before opening the transaction. We mutate the
	// shared *TransactionContext inside the closure once the tx is available so that
	// downstream callers reading the tx from ctx see the live transaction.
	ctxWithTx := domain.WithTransactionContext(ctx, txCtx)
	err := tm.db.WithContext(ctxWithTx).Transaction(func(tx *gorm.DB) error {
		txCtx.TX = tx
		return operations(tx)
	})
	if err != nil {
		return err
	}
	txCtx.RunAfterCommitCallbacks()
	return nil
}

// ExecuteWithRollback executes operations and provides rollback capability
func (tm *TransactionManager) ExecuteWithRollback(ctx context.Context, operations func(tx *gorm.DB) error) error {
	tx := tm.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := operations(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDB returns the underlying database connection
func (tm *TransactionManager) GetDB() *gorm.DB {
	return tm.db
}

// WithTransaction executes the given function within a database transaction
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	txCtx := &domain.TransactionContext{
		IsTransactional: true,
	}
	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx.TX = tx
		ctxWithTx := domain.WithTransactionContext(ctx, txCtx)
		return fn(ctxWithTx)
	})
	if err != nil {
		return err
	}
	txCtx.RunAfterCommitCallbacks()
	return nil
}

// WithTransactionResult executes the given function within a database transaction and returns a result
func (tm *TransactionManager) WithTransactionResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	var result interface{}
	txCtx := &domain.TransactionContext{
		IsTransactional: true,
	}
	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx.TX = tx
		ctxWithTx := domain.WithTransactionContext(ctx, txCtx)
		var err error
		result, err = fn(ctxWithTx)
		return err
	})
	if err != nil {
		return result, err
	}
	txCtx.RunAfterCommitCallbacks()
	return result, nil
}
