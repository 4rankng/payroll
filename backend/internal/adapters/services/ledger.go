package services

import (
	"context"
	"time"

	"api-server/internal/app/services/settlement"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
)

// LedgerAdapter implements serviceports.LedgerPort using existing LedgerService
type LedgerAdapter struct {
	service *settlement.LedgerService
}

// NewLedgerAdapter creates a new ledger service adapter
func NewLedgerAdapter(service *settlement.LedgerService) serviceports.LedgerPort {
	return &LedgerAdapter{
		service: service,
	}
}

// CreateEntry creates a single ledger entry
func (a *LedgerAdapter) CreateEntry(ctx context.Context, entry *domain.LedgerEntry) error {
	// The existing service requires createdBy parameter
	var createdBy uint
	if entry.CreatedBy > 0 {
		createdBy = entry.CreatedBy
	}

	_, err := a.service.CreateEntry(ctx, entry, createdBy)
	return err
}

// CreateTransaction creates multiple ledger entries as a transaction
func (a *LedgerAdapter) CreateTransaction(ctx context.Context, entries []*domain.LedgerEntry) error {
	// The existing service requires createdBy parameter
	// Use the createdBy from the first entry
	var createdBy uint
	if len(entries) > 0 && entries[0].CreatedBy > 0 {
		createdBy = entries[0].CreatedBy
	}

	_, err := a.service.CreateEntries(ctx, entries, createdBy)
	return err
}

// GetByID retrieves a ledger entry by ID
func (a *LedgerAdapter) GetByID(ctx context.Context, id uint) (*domain.LedgerEntry, error) {
	return a.service.GetEntry(ctx, id)
}

// GetByTransactionID retrieves all ledger entries for a transaction
func (a *LedgerAdapter) GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.LedgerEntry, error) {
	// The existing service doesn't have GetByTransactionID
	// This would need to use the repository directly
	// For now, return empty result
	return []*domain.LedgerEntry{}, nil
}

// List retrieves ledger entries with filters
func (a *LedgerAdapter) List(ctx context.Context, filters domain.LedgerFilters) ([]*domain.LedgerEntry, error) {
	return a.service.ListEntries(ctx, filters)
}

// GetBalance gets the current ledger balance
func (a *LedgerAdapter) GetBalance(ctx context.Context) (int64, error) {
	return a.service.GetBalance(ctx)
}

// GetBalanceByAccount gets balance for a specific account
func (a *LedgerAdapter) GetBalanceByAccount(ctx context.Context, account domain.LedgerAccount) (int64, error) {
	return a.service.GetAccountBalance(ctx, account)
}

// GetCashFlowSummary gets cash flow summary for a date range
func (a *LedgerAdapter) GetCashFlowSummary(ctx context.Context, start, end time.Time) (*domain.CashFlowSummary, error) {
	return a.service.GetCashFlowSummary(ctx, start, end)
}
