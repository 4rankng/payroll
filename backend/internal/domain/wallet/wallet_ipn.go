package wallet

import (
	"time"
)

// WalletIPN records every inbound IPN message from 9pay for audit and
// discrepancy resolution. Each row is immutable once processing_status
// is set to its final value (applied | ignored | error).
type WalletIPN struct {
	ID               uint64     `json:"id"`
	Provider         string     `json:"provider"`
	InvoiceNo        string     `json:"invoice_no"`
	RequestID        string     `json:"request_id"`
	Status           string     `json:"status"`
	Amount           int64      `json:"amount"`
	RawErrorCode     string     `json:"raw_error_code"`
	FailureReason    *string    `json:"failure_reason"`
	RawPayload       []byte     `json:"raw_payload"`
	ProcessingStatus string     `json:"processing_status"` // pending | applied | ignored | error
	ProcessingError  *string    `json:"processing_error"`
	WalletPaymentID  *uint64    `json:"wallet_payment_id"`
	CreatedAt        time.Time  `json:"created_at"`
	ProcessedAt      *time.Time `json:"processed_at"`
}

func (WalletIPN) TableName() string { return "wallet_ipn" }
