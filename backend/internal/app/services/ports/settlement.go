package ports

import (
	"context"

	"api-server/internal/domain"
)

// TransactionPort defines the interface for transaction operations
// This prevents direct dependency on settlement.TransactionService
type TransactionPort interface {
	CreateTransaction(ctx context.Context, transaction *domain.Transaction) error
	UpdateTransaction(ctx context.Context, id uint, updates map[string]interface{}) error
	GetTransactionByID(ctx context.Context, id uint) (*domain.Transaction, error)
}

// LedgerPort defines the interface for ledger operations
type LedgerPort interface {
	CreateLedgerEntry(ctx context.Context, entry *domain.LedgerEntry) error
	GetLedgerEntriesForTransaction(ctx context.Context, transactionID uint) ([]*domain.LedgerEntry, error)
	ReconcileLedger(ctx context.Context, transactionID uint) error
}

// SettlementPort defines the interface for settlement operations
type SettlementPort interface {
	CreateSettlement(ctx context.Context, settlement *domain.Settlement) error
	ProcessSettlement(ctx context.Context, settlementID uint) error
}
