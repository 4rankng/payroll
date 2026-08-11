package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionContextKey(t *testing.T) {
	key1 := TransactionContextKey{}
	key2 := TransactionContextKey{}

	// TransactionContextKey should be a comparable type
	assert.Equal(t, key1, key2)
}

func TestWithTransactionContext(t *testing.T) {
	ctx := context.Background()
	txCtx := &TransactionContext{
		TX:              nil,
		IsTransactional: true,
	}

	newCtx := WithTransactionContext(ctx, txCtx)
	assert.NotNil(t, newCtx)

	// Verify the context is different from the original
	assert.NotEqual(t, ctx, newCtx)
}

func TestGetTransactionFromContext_WithTransactionContext(t *testing.T) {
	ctx := context.Background()
	originalTxCtx := &TransactionContext{
		TX:              nil,
		IsTransactional: true,
	}

	newCtx := WithTransactionContext(ctx, originalTxCtx)
	retrievedTxCtx, ok := GetTransactionFromContext(newCtx)

	assert.True(t, ok)
	assert.NotNil(t, retrievedTxCtx)
	assert.Equal(t, originalTxCtx, retrievedTxCtx)
	assert.True(t, retrievedTxCtx.IsTransactional)
}

func TestGetTransactionFromContext_WithoutTransactionContext(t *testing.T) {
	ctx := context.Background()
	retrievedTxCtx, ok := GetTransactionFromContext(ctx)

	assert.False(t, ok)
	assert.Nil(t, retrievedTxCtx)
}

func TestTransactionContext_IsTransactional(t *testing.T) {
	tests := []struct {
		name            string
		isTransactional bool
	}{
		{
			name:            "Transactional context",
			isTransactional: true,
		},
		{
			name:            "Non-transactional context",
			isTransactional: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txCtx := &TransactionContext{
				TX:              nil,
				IsTransactional: tt.isTransactional,
			}

			assert.Equal(t, tt.isTransactional, txCtx.IsTransactional)
		})
	}
}

func TestTransactionContext_FieldValues(t *testing.T) {
	txCtx := &TransactionContext{
		TX:              nil,
		IsTransactional: true,
	}

	assert.Nil(t, txCtx.TX)
	assert.True(t, txCtx.IsTransactional)
}

func TestWithTransactionContext_ChainedContexts(t *testing.T) {
	ctx := context.Background()

	// Add first transaction context
	txCtx1 := &TransactionContext{
		TX:              nil,
		IsTransactional: true,
	}
	ctx = WithTransactionContext(ctx, txCtx1)

	// Verify first context
	retrieved1, ok := GetTransactionFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, txCtx1, retrieved1)

	// Replace with second transaction context
	txCtx2 := &TransactionContext{
		TX:              nil,
		IsTransactional: false,
	}
	ctx = WithTransactionContext(ctx, txCtx2)

	// Verify second context replaces first
	retrieved2, ok := GetTransactionFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, txCtx2, retrieved2)
	assert.NotEqual(t, txCtx1, retrieved2)
}

func TestWithoutTransactionContext_PreservesOtherValues(t *testing.T) {
	type requestKey struct{}
	ctx := context.WithValue(context.Background(), requestKey{}, "request-value")
	ctx = WithTransactionContext(ctx, &TransactionContext{IsTransactional: true})

	detached := WithoutTransactionContext(ctx)

	txCtx, ok := GetTransactionFromContext(detached)
	assert.False(t, ok)
	assert.Nil(t, txCtx)
	assert.Equal(t, "request-value", detached.Value(requestKey{}))
}

func TestWithTransactionContext_NilTransactionContext(t *testing.T) {
	ctx := context.Background()
	newCtx := WithTransactionContext(ctx, nil)

	retrieved, ok := GetTransactionFromContext(newCtx)
	assert.True(t, ok) // Key exists but value is nil
	assert.Nil(t, retrieved)
}
