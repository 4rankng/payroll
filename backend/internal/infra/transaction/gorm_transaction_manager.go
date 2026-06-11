package transaction

import (
	"context"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// GormTransactionManager implements TransactionManager using GORM
type GormTransactionManager struct {
	db *gorm.DB
}

// NewGormTransactionManager creates a new GORM transaction manager
func NewGormTransactionManager(db *gorm.DB) domain.TransactionManager {
	return &GormTransactionManager{
		db: db,
	}
}

// WithTransaction executes the given function within a database transaction
func (tm *GormTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction context
		txCtx := &domain.TransactionContext{
			TX:              tx,
			IsTransactional: true,
		}

		// Add transaction context to the context
		newCtx := domain.WithTransactionContext(ctx, txCtx)

		// Execute the function with transaction context
		return fn(newCtx)
	})
}

// WithTransactionResult executes the given function within a database transaction and returns a result
func (tm *GormTransactionManager) WithTransactionResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	var result interface{}
	var resultErr error

	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction context
		txCtx := &domain.TransactionContext{
			TX:              tx,
			IsTransactional: true,
		}

		// Add transaction context to the context
		newCtx := domain.WithTransactionContext(ctx, txCtx)

		// Execute the function with transaction context
		result, resultErr = fn(newCtx)
		return resultErr
	})

	if err != nil {
		return nil, err
	}

	return result, resultErr
}
