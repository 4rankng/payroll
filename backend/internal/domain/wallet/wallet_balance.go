package wallet

// WalletBalance represents the wallet balance computed from DB.
type WalletBalance struct {
	Available  int64  `json:"available"`
	PendingIn  int64  `json:"pending_in"`
	PendingOut int64  `json:"pending_out"`
	Limbo      int64  `json:"limbo"` // unreconciled failed — visible but not deducted from Available
	Currency   string `json:"currency"`
	AsOf       string `json:"as_of"`
}

// UnifiedTransaction represents a normalized transaction from both topups and payments.
type UnifiedTransaction struct {
	ID           uint64 `json:"id"`
	Type         string `json:"type"`   // "topup" | "payment"
	Amount       int64  `json:"amount"` // signed: + for topup, - for payment
	Status       string `json:"status"`
	OccurredAt   string `json:"occurred_at"`
	Reference    string `json:"reference"`    // bank_ref for topups, request_id for payments
	Counterparty string `json:"counterparty"` // empty for topups
	Note         string `json:"note"`
}

// TransactionFilter represents filter options for the unified transaction view.
type TransactionFilter struct {
	Type     string  `json:"type"` // "topup" | "payment" | "" (both)
	Status   string  `json:"status"`
	Search   string  `json:"search"` // matches counterparty, reference, or note
	FromDate *string `json:"from_date"`
	ToDate   *string `json:"to_date"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// ReconciliationJob represents an async reconciliation processing job.
type ReconciliationJob struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"` // "processing", "completed", "failed"
	TotalRows   int        `json:"total_rows"`
	Matched     int        `json:"matched"`
	Unmatched   int        `json:"unmatched"`
	Headers     []string   `json:"headers,omitempty"`
	RawRows     [][]string `json:"raw_rows,omitempty"`
	Error       string     `json:"error,omitempty"`
	CompletedAt string     `json:"completed_at,omitempty"`
}
