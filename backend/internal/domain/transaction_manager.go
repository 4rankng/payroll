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

// WithoutTransactionContext preserves the caller's request-scoped values while
// shadowing any transaction inherited from the parent context. Use it for work
// that runs after commit so repositories cannot reuse an already-committed TX.
func WithoutTransactionContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, TransactionContextKey{}, struct{}{})
}
