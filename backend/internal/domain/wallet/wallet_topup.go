package wallet

import (
	"time"
)

// WalletTopup represents a manual wallet topup transaction.
// The actual topup is done via bank app, so topups are confirmed on insert — no status field.
type WalletTopup struct {
	ID         uint64    `json:"id"`
	Provider   string    `json:"provider"`
	Amount     int64     `json:"amount"`
	BankRef    string    `json:"bank_ref"`
	OccurredAt time.Time `json:"occurred_at"`
	Note       *string   `json:"note"`
	CreatedBy  uint64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Version    int64     `json:"version"`
}

// CreateWalletTopupRequest represents a request to create a wallet topup.
type CreateWalletTopupRequest struct {
	Amount     int64     `json:"amount" binding:"required"`
	BankRef    string    `json:"bank_ref" binding:"required"`
	OccurredAt time.Time `json:"occurred_at" binding:"required"`
	Note       *string   `json:"note"`
}

// WalletTopupFilter represents filter options for querying wallet topups.
type WalletTopupFilter struct {
	BankRef   string     `json:"bank_ref"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

// WalletTopupResponse represents a wallet topup response.
type WalletTopupResponse struct {
	ID         uint64  `json:"id"`
	Amount     int64   `json:"amount"`
	BankRef    string  `json:"bank_ref"`
	OccurredAt string  `json:"occurred_at"`
	Note       *string `json:"note"`
	CreatedBy  uint64  `json:"created_by"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}
