package dto

import "api-server/internal/domain"

// CreateTransactionRequest represents the request to create a new transaction
type CreateTransactionRequest struct {
	Description     string                   `json:"description" binding:"required"`
	TransactionType domain.TransactionType   `json:"transaction_type" binding:"required,oneof=expense revenue capital write_off loan_disbursement"`
	Amount          int64                    `json:"amount" binding:"required,gt=0"`
	Party           string                   `json:"party" binding:"required"`
	Status          domain.TransactionStatus `json:"status" binding:"required,oneof=settled pending"`
	URL             string                   `json:"url,omitempty"`
	AssetID         *uint                    `json:"asset_id,omitempty"`
	UserID          *uint                    `json:"user_id,omitempty"`
}

// TransactionResponse represents the transaction data in API responses
type TransactionResponse struct {
	ID                    uint                   `json:"id"`
	Description           string                 `json:"description"`
	TransactionCode       string                 `json:"transaction_code"`
	TransactionType       domain.TransactionType `json:"transaction_type"`
	Amount                int64                  `json:"amount"`
	SettledAmount         int64                  `json:"settled_amount"`
	PendingAmount         int64                  `json:"pending_amount"`
	Party                 string                 `json:"party"`
	Status                string                 `json:"status"` // Can be "pending", "settled", or "partially_settled"
	URL                   string                 `json:"url,omitempty"`
	AssetID               *uint                  `json:"asset_id,omitempty"`
	Asset                 interface{}            `json:"asset,omitempty"`
	ReversedTransactionID *uint                  `json:"reversed_transaction_id,omitempty"`
	CreatedBy             uint                   `json:"created_by"`
	CreatedAt             string                 `json:"created_at"`
	UpdatedAt             string                 `json:"updated_at"`
	Settlements           []SettlementResponse   `json:"settlements,omitempty"`
}

// SettleTransactionRequest represents the request to create a settlement
type SettleTransactionRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	SettlementDate string `json:"settlement_date" binding:"required"`
	ProofURL       string `json:"proof_url,omitempty"`
	ProofAssetID   *uint  `json:"proof_asset_id,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

// SettlementResponse represents settlement data in API responses
type SettlementResponse struct {
	ID             uint   `json:"id"`
	TransactionID  uint   `json:"transaction_id"`
	Amount         int64  `json:"amount"`
	SettlementDate string `json:"settlement_date"`
	ProofURL       string `json:"proof_url,omitempty"`
	ProofAssetID   *uint  `json:"proof_asset_id,omitempty"`
	PaymentMethod  string `json:"payment_method"`
	Notes          string `json:"notes,omitempty"`
	CreatedBy      uint   `json:"created_by"`
	CreatedAt      string `json:"created_at"`
}

// TransactionWithLedgerResponse includes the ledger entries created
type TransactionWithLedgerResponse struct {
	Transaction   TransactionResponse `json:"transaction"`
	LedgerEntries []interface{}       `json:"ledger_entries"`
}

// TransactionWithSettlementsResponse includes settlements and ledger entries
type TransactionWithSettlementsResponse struct {
	Transaction   TransactionResponse  `json:"transaction"`
	Settlement    *SettlementResponse  `json:"settlement,omitempty"`
	Settlements   []SettlementResponse `json:"settlements,omitempty"`
	LedgerEntries []interface{}        `json:"ledger_entries,omitempty"`
}

// TransactionMetadata represents metadata for a transaction type
type TransactionMetadata struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// TransactionMetadataResponse represents the response for transaction metadata
type TransactionMetadataResponse struct {
	TransactionTypes []TransactionMetadata `json:"transaction_types"`
	Statuses         []TransactionMetadata `json:"statuses"`
}

// ReverseTransactionRequest represents the request to reverse a transaction
type ReverseTransactionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ReverseTransactionResponse represents the response after reversing a transaction
type ReverseTransactionResponse struct {
	OriginalTransaction TransactionResponse `json:"original_transaction"`
	ReversalTransaction TransactionResponse `json:"reversal_transaction"`
	LedgerEntries       []interface{}       `json:"ledger_entries"`
}

// UpdateTransactionEvidenceRequest represents a request to update only evidence fields
// Only url and/or asset_id are accepted; other fields are immutable
type UpdateTransactionEvidenceRequest struct {
	URL     *string `json:"url,omitempty"`
	AssetID *uint   `json:"asset_id,omitempty"`
}

// UploadSettlementResultResponse represents the response after processing settlement upload
type UploadSettlementResultResponse struct {
	TotalTimesheets    int    `json:"total_timesheets"`
	SettledTimesheets  int    `json:"settled_timesheets"`
	SettlementAmount   int64  `json:"settlement_amount"`
	TimesheetIDs       []uint `json:"timesheet_ids"`
	AssetID            uint   `json:"asset_id"`
	SettlementsCreated int    `json:"settlements_created"`
}
