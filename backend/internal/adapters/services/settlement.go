package services

import (
	"context"

	"api-server/internal/app/services/settlement"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
)

// SettlementAdapter implements serviceports.SettlementPort using existing TransactionService
// Note: Settlements are managed through the TransactionService in the existing architecture
type SettlementAdapter struct {
	service *settlement.TransactionService
}

// NewSettlementAdapter creates a new settlement service adapter
func NewSettlementAdapter(service *settlement.TransactionService) serviceports.SettlementPort {
	return &SettlementAdapter{
		service: service,
	}
}

// Create creates a new settlement
func (a *SettlementAdapter) Create(ctx context.Context, settlementEntity *domain.Settlement) error {
	// Settlements are created through CreateSettlement which requires transaction ID
	// This is a simplified interface that doesn't match exactly
	// Return not implemented for now
	return domain.NewNotFoundError("Create not implemented - use CreateSettlement with transaction ID")
}

// GetByID retrieves a settlement by ID
func (a *SettlementAdapter) GetByID(ctx context.Context, id uint) (*domain.Settlement, error) {
	// The existing service doesn't have GetSettlementByID
	// This would need to use the repository directly
	return nil, domain.NewNotFoundError("GetByID not implemented in settlement adapter")
}

// GetByTransactionID retrieves all settlements for a transaction
func (a *SettlementAdapter) GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.Settlement, error) {
	return a.service.GetTransactionSettlements(ctx, transactionID)
}

// List retrieves settlements with filters
func (a *SettlementAdapter) List(ctx context.Context, filters domain.SettlementFilters) ([]*domain.Settlement, error) {
	// The existing service doesn't have List with filters
	// This would need to use the repository directly
	return nil, domain.NewNotFoundError("List not implemented in settlement adapter")
}

// GetTotalSettledAmount gets the total settled amount for a transaction
func (a *SettlementAdapter) GetTotalSettledAmount(ctx context.Context, transactionID uint) (int64, error) {
	// Get transaction to access its calculated settled amount
	txn, err := a.service.GetTransaction(ctx, transactionID)
	if err != nil {
		return 0, err
	}

	return txn.SettledAmount, nil
}
