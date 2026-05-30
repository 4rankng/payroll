package domain

import (
	"context"

	"gorm.io/gorm"
)

// TransactionManager provides transaction management abstraction
type TransactionManager interface {
	// WithTransaction executes the given function within a database transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// WithTransactionResult executes the given function within a database transaction and returns a result
	WithTransactionResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error)
}

// UnitOfWork represents a unit of work pattern for coordinating multiple operations
type UnitOfWork interface {
	// RegisterNew registers a new entity to be created
	RegisterNew(ctx context.Context, entity interface{}) error

	// RegisterDirty registers an entity to be updated
	RegisterDirty(ctx context.Context, entity interface{}) error

	// RegisterDeleted registers an entity to be deleted
	RegisterDeleted(ctx context.Context, entity interface{}) error

	// Commit commits all registered operations in a single transaction
	Commit(ctx context.Context) error

	// Rollback rolls back all pending operations
	Rollback(ctx context.Context) error
}

// TransactionContext represents context for transaction operations
type TransactionContext struct {
	// TX holds the database transaction
	TX *gorm.DB

	// IsTransactional indicates if operations should run in transaction
	IsTransactional bool

	// afterCommitCallbacks holds functions to execute after the transaction commits
	afterCommitCallbacks []func()
}

// RegisterAfterCommit registers a callback to be executed after the transaction successfully commits.
// If not in a transaction, the callback is executed immediately.
func RegisterAfterCommit(ctx context.Context, fn func()) {
	txCtx, ok := GetTransactionFromContext(ctx)
	if !ok || !txCtx.IsTransactional {
		// Not in a transaction, execute immediately
		go fn()
		return
	}
	txCtx.afterCommitCallbacks = append(txCtx.afterCommitCallbacks, fn)
}

// RunAfterCommitCallbacks executes all registered after-commit callbacks.
// This should only be called by the transaction manager after a successful commit.
func (tc *TransactionContext) RunAfterCommitCallbacks() {
	for _, cb := range tc.afterCommitCallbacks {
		go cb()
	}
	tc.afterCommitCallbacks = nil
}

// TransactionContextKey is the key for storing transaction context
type TransactionContextKey struct{}

// GetTransactionFromContext retrieves transaction context from context
func GetTransactionFromContext(ctx context.Context) (*TransactionContext, bool) {
	txCtx, ok := ctx.Value(TransactionContextKey{}).(*TransactionContext)
	return txCtx, ok
}

// WithTransactionContext adds transaction context to the context
func WithTransactionContext(ctx context.Context, txCtx *TransactionContext) context.Context {
	return context.WithValue(ctx, TransactionContextKey{}, txCtx)
}
