package services

import (
	"context"

	"api-server/internal/domain"
)

// SettlementPort defines the interface for cross-domain settlement service communication
type SettlementPort interface {
	Create(ctx context.Context, settlement *domain.Settlement) error
	GetByID(ctx context.Context, id uint) (*domain.Settlement, error)
	GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.Settlement, error)
	List(ctx context.Context, filters domain.SettlementFilters) ([]*domain.Settlement, error)
	GetTotalSettledAmount(ctx context.Context, transactionID uint) (int64, error)
}
