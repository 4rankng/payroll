package wallet

import (
	"time"
)

// WalletPayment represents a wallet payment transaction (transfer to employee account).
type WalletPayment struct {
	ID                 uint64     `json:"id"`
	TxnID              string     `json:"txn_id"`
	RequestID          string     `json:"request_id"`
	InvoiceNo          *string    `json:"invoice_no"`
	Provider           string     `json:"provider"`
	RequestedAmount    int64      `json:"requested_amount"`
	Fee                int64      `json:"fee"`
	RecipientName      string     `json:"recipient_name"`
	RecipientAccountNo string     `json:"recipient_account_no"`
	RecipientBank      string     `json:"recipient_bank"`
	Description        *string    `json:"description"`
	Status             string     `json:"status"`
	ErrorCode          *string    `json:"error_code"`
	ErrorMessage       *string    `json:"error_message"`
	EntityID           *uint64    `json:"entity_id"`
	CreatedBy          *uint64    `json:"created_by"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	SettledAt          *time.Time `json:"settled_at"`
}

// WalletPaymentFilter represents filter options for querying wallet payments.
type WalletPaymentFilter struct {
	TxnID            string     `json:"txn_id"`
	RequestID        string     `json:"request_id"`
	InvoiceNo        string     `json:"invoice_no"`
	Status           string     `json:"status"`
	ErrorCode        string     `json:"error_code"`
	RecipientName    string     `json:"recipient_name"`
	RecipientAccount string     `json:"recipient_account"`
	RecipientBank    string     `json:"recipient_bank"`
	StartDate        *time.Time `json:"start_date"`
	EndDate          *time.Time `json:"end_date"`
	EntityID         *uint64    `json:"entity_id"`
	Page             int        `json:"page"`
	PageSize         int        `json:"page_size"`
}

// WalletPaymentResponse represents a wallet payment response.
type WalletPaymentResponse struct {
	ID                 uint64  `json:"id"`
	TxnID              string  `json:"txn_id"`
	RequestID          string  `json:"request_id"`
	InvoiceNo          *string `json:"invoice_no"`
	RequestedAmount    int64   `json:"requested_amount"`
	Fee                int64   `json:"fee"`
	RecipientName      string  `json:"recipient_name"`
	RecipientAccountNo string  `json:"recipient_account_no"`
	RecipientBank      string  `json:"recipient_bank"`
	Description        *string `json:"description"`
	Status             string  `json:"status"`
	ErrorCode          *string `json:"error_code"`
	ErrorMessage       *string `json:"error_message"`
	EntityID           *uint64 `json:"entity_id"`
	CreatedBy          *uint64 `json:"created_by"`
	Version            int64   `json:"version"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	SettledAt          *string `json:"settled_at"`
}

// WalletPaymentStats represents statistics for wallet payments.
type WalletPaymentStats struct {
	TotalRequests int64   `json:"total_requests"`
	Completed     int64   `json:"completed"`
	Failed        int64   `json:"failed"`
	Pending       int64   `json:"pending"`
	TotalAmount   int64   `json:"total_amount"`
	TotalFee      int64   `json:"total_fee"`
	SuccessRate   float64 `json:"success_rate"`
}
