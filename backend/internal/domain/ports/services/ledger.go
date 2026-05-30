package services

import (
	"context"
	"time"

	"api-server/internal/domain"
)

// LedgerPort defines the interface for cross-domain ledger service communication
type LedgerPort interface {
	CreateEntry(ctx context.Context, entry *domain.LedgerEntry) error
	CreateTransaction(ctx context.Context, entries []*domain.LedgerEntry) error
	GetByID(ctx context.Context, id uint) (*domain.LedgerEntry, error)
	GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.LedgerEntry, error)
	List(ctx context.Context, filters domain.LedgerFilters) ([]*domain.LedgerEntry, error)
	GetBalance(ctx context.Context) (int64, error)
	GetBalanceByAccount(ctx context.Context, account domain.LedgerAccount) (int64, error)
	GetCashFlowSummary(ctx context.Context, start, end time.Time) (*domain.CashFlowSummary, error)
}
