package dto

import (
	"time"
)

// CreateLenderRequest represents the request to create a new lender
type CreateLenderRequest struct {
	Name              string  `json:"name" binding:"required,max=255"`
	CCCD              *string `json:"cccd" binding:"omitempty,max=50"`
	Email             *string `json:"email" binding:"omitempty,email,max=255"`
	Mobile            *string `json:"mobile" binding:"omitempty,max=50"`
	Notes             *string `json:"notes"`
	BankID            *uint   `json:"bank_id" binding:"omitempty"`
	BankAccountNumber *string `json:"bank_account_number" binding:"omitempty,max=30"`
	BankAccountName   *string `json:"bank_account_name" binding:"omitempty,max=255"`
}

// UpdateLenderRequest represents the request to update a lender
type UpdateLenderRequest struct {
	Name              *string `json:"name" binding:"omitempty,max=255"`
	CCCD              *string `json:"cccd" binding:"omitempty,max=50"`
	Email             *string `json:"email" binding:"omitempty,email,max=255"`
	Mobile            *string `json:"mobile" binding:"omitempty,max=50"`
	Notes             *string `json:"notes"`
	BankID            *uint   `json:"bank_id" binding:"omitempty"`
	BankAccountNumber *string `json:"bank_account_number" binding:"omitempty,max=30"`
	BankAccountName   *string `json:"bank_account_name" binding:"omitempty,max=255"`
}

// LenderResponse represents the response containing lender data
type LenderResponse struct {
	ID                uint      `json:"id"`
	Name              string    `json:"name"`
	CCCD              *string   `json:"cccd"`
	Email             *string   `json:"email"`
	Mobile            *string   `json:"mobile"`
	Notes             *string   `json:"notes"`
	BankID            *uint     `json:"bank_id"`
	BankAccountNumber *string   `json:"bank_account_number"`
	BankAccountName   *string   `json:"bank_account_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// LenderSummary represents financial summary for a lender
type LenderSummary struct {
	TotalPrincipalBorrowed int64 `json:"total_principal_borrowed"`
	TotalPrincipalRepaid   int64 `json:"total_principal_repaid"`
	OutstandingPrincipal   int64 `json:"outstanding_principal"`
	ActiveLoansCount       int   `json:"active_loans_count"`
}

// LenderWithSummaryResponse represents lender with financial summary
type LenderWithSummaryResponse struct {
	LenderResponse
	Summary LenderSummary `json:"summary"`
}
