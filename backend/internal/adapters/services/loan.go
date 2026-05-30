package services

import (
	"context"

	"api-server/internal/app/services/loan"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
	"api-server/internal/pkg/clock"
)

// LoanAdapter implements serviceports.LoanPort using existing LoanService
type LoanAdapter struct {
	service *loan.LoanService
}

// NewLoanAdapter creates a new loan service adapter
func NewLoanAdapter(service *loan.LoanService) serviceports.LoanPort {
	return &LoanAdapter{
		service: service,
	}
}

// Create creates a new loan
func (a *LoanAdapter) Create(ctx context.Context, loanEntity *domain.Loan) error {
	// The existing service doesn't have a standalone Create method
	// Loans are typically created and then disbursed
	// For now, return not implemented
	return domain.NewNotFoundError("Create not implemented - use repository directly for loan creation")
}

// Update updates a loan
func (a *LoanAdapter) Update(ctx context.Context, loanEntity *domain.Loan) error {
	// Prepare updates map from loan entity
	updates := make(map[string]interface{})

	if loanEntity.Description != nil {
		updates["description"] = loanEntity.Description
	}

	if loanEntity.PaymentDayOfMonth > 0 {
		updates["payment_day_of_month"] = loanEntity.PaymentDayOfMonth
	}

	_, err := a.service.UpdateLoan(ctx, loanEntity.ID, updates)
	return err
}

// GetByID retrieves a loan by ID
func (a *LoanAdapter) GetByID(ctx context.Context, id uint) (*domain.Loan, error) {
	return a.service.GetLoan(ctx, id)
}

// List retrieves loans with filters
func (a *LoanAdapter) List(ctx context.Context, filters domain.LoanFilters) ([]*domain.Loan, int64, error) {
	return a.service.ListLoans(ctx, filters)
}

// Disburse disburses a loan
func (a *LoanAdapter) Disburse(ctx context.Context, loanID uint, userID uint) error {
	// Note: The existing service requires disbursementDate and reference parameters
	// For adapter simplicity, use current time and generate a reference
	_, _, err := a.service.DisburseLoan(ctx, loanID, clock.Now(), "Disbursement", userID)
	return err
}

// Repay makes a loan repayment
func (a *LoanAdapter) Repay(ctx context.Context, loanID uint, amount int64, userID uint) error {
	// Note: The existing service requires payment date and reference parameters
	_, _, err := a.service.RepayPrincipal(ctx, loanID, amount, clock.Now(), "Repayment", nil, userID)
	return err
}

// Delete deletes a loan
func (a *LoanAdapter) Delete(ctx context.Context, id uint) error {
	return a.service.DeleteLoan(ctx, id)
}
