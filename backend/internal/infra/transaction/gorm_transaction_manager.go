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

// GormUnitOfWork implements UnitOfWork using GORM
type GormUnitOfWork struct {
	tx              *gorm.DB
	newEntities     []interface{}
	dirtyEntities   []interface{}
	deletedEntities []interface{}
}

// NewGormUnitOfWork creates a new GORM unit of work
func NewGormUnitOfWork(db *gorm.DB) domain.UnitOfWork {
	return &GormUnitOfWork{
		tx:              db,
		newEntities:     make([]interface{}, 0),
		dirtyEntities:   make([]interface{}, 0),
		deletedEntities: make([]interface{}, 0),
	}
}

// RegisterNew registers a new entity to be created
func (uow *GormUnitOfWork) RegisterNew(ctx context.Context, entity interface{}) error {
	uow.newEntities = append(uow.newEntities, entity)
	return nil
}

// RegisterDirty registers an entity to be updated
func (uow *GormUnitOfWork) RegisterDirty(ctx context.Context, entity interface{}) error {
	uow.dirtyEntities = append(uow.dirtyEntities, entity)
	return nil
}

// RegisterDeleted registers an entity to be deleted
func (uow *GormUnitOfWork) RegisterDeleted(ctx context.Context, entity interface{}) error {
	uow.deletedEntities = append(uow.deletedEntities, entity)
	return nil
}

// Commit commits all registered operations in a single transaction
func (uow *GormUnitOfWork) Commit(ctx context.Context) error {
	return uow.tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create new entities
		for _, entity := range uow.newEntities {
			if err := tx.Create(entity).Error; err != nil {
				return err
			}
		}

		// Update dirty entities
		for _, entity := range uow.dirtyEntities {
			if err := tx.Save(entity).Error; err != nil {
				return err
			}
		}

		// Delete entities
		for _, entity := range uow.deletedEntities {
			if err := tx.Delete(entity).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Rollback rolls back all pending operations
func (uow *GormUnitOfWork) Rollback(ctx context.Context) error {
	// Clear all registered entities
	uow.newEntities = uow.newEntities[:0]
	uow.dirtyEntities = uow.dirtyEntities[:0]
	uow.deletedEntities = uow.deletedEntities[:0]
	return nil
}
