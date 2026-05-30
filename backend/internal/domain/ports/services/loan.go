package services

import (
	"context"

	"api-server/internal/domain"
)

// LoanPort defines the interface for cross-domain loan service communication
type LoanPort interface {
	Create(ctx context.Context, loan *domain.Loan) error
	Update(ctx context.Context, loan *domain.Loan) error
	GetByID(ctx context.Context, id uint) (*domain.Loan, error)
	List(ctx context.Context, filters domain.LoanFilters) ([]*domain.Loan, int64, error)
	Disburse(ctx context.Context, loanID uint, userID uint) error
	Repay(ctx context.Context, loanID uint, amount int64, userID uint) error
	Delete(ctx context.Context, id uint) error
}
