package services

import (
	"context"

	"api-server/internal/app/services/settlement"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
)

// TransactionAdapter implements serviceports.TransactionPort using existing TransactionService
type TransactionAdapter struct {
	service *settlement.TransactionService
}

// NewTransactionAdapter creates a new transaction service adapter
func NewTransactionAdapter(service *settlement.TransactionService) serviceports.TransactionPort {
	return &TransactionAdapter{
		service: service,
	}
}

// Create creates a new transaction
func (a *TransactionAdapter) Create(ctx context.Context, transaction *domain.Transaction) error {
	_, _, err := a.service.CreateTransaction(ctx, transaction)
	return err
}

// Update updates transaction fields
func (a *TransactionAdapter) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	// Extract evidence fields from updates map
	var url *string
	var assetID *uint

	if urlVal, ok := updates["url"].(string); ok {
		url = &urlVal
	}
	if assetIDVal, ok := updates["asset_id"].(uint); ok {
		assetID = &assetIDVal
	}

	_, err := a.service.UpdateTransactionEvidence(ctx, id, url, assetID)
	return err
}

// GetByID retrieves a transaction by ID
func (a *TransactionAdapter) GetByID(ctx context.Context, id uint) (*domain.Transaction, error) {
	return a.service.GetTransaction(ctx, id)
}

// GetByCode retrieves a transaction by code
func (a *TransactionAdapter) GetByCode(ctx context.Context, code string) (*domain.Transaction, error) {
	// The existing service doesn't have GetByCode method
	// We need to use the repository directly or add this method to the service
	// For now, return not implemented error
	return nil, domain.NewNotFoundError("GetByCode not implemented in transaction service")
}

// List retrieves transactions with filters
func (a *TransactionAdapter) List(ctx context.Context, filters domain.TransactionFilters) ([]*domain.Transaction, error) {
	return a.service.ListTransactions(ctx, filters)
}

// Delete deletes a transaction
func (a *TransactionAdapter) Delete(ctx context.Context, id uint) error {
	return a.service.DeleteTransaction(ctx, id)
}

// CreateTransaction creates a new transaction with ledger entries
func (a *TransactionAdapter) CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error) {
	return a.service.CreateTransaction(ctx, txn)
}

// CreateSettlement creates a settlement for a transaction
func (a *TransactionAdapter) CreateSettlement(ctx context.Context, transactionID uint, settlement *domain.Settlement) (*domain.Transaction, *domain.Settlement, []*domain.LedgerEntry, error) {
	return a.service.CreateSettlement(ctx, transactionID, settlement)
}
