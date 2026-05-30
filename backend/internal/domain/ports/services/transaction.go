package services

import (
	"context"

	"api-server/internal/domain"
)

// TransactionPort defines the interface for cross-domain transaction service communication
type TransactionPort interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	Update(ctx context.Context, id uint, updates map[string]interface{}) error
	GetByID(ctx context.Context, id uint) (*domain.Transaction, error)
	GetByCode(ctx context.Context, code string) (*domain.Transaction, error)
	List(ctx context.Context, filters domain.TransactionFilters) ([]*domain.Transaction, error)
	Delete(ctx context.Context, id uint) error

	// Domain-specific operations with ledger entries
	CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error)
	CreateSettlement(ctx context.Context, transactionID uint, settlement *domain.Settlement) (*domain.Transaction, *domain.Settlement, []*domain.LedgerEntry, error)
}
