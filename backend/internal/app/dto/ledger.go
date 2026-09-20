package dto

import (
	"time"
)

// CreateLedgerEntryRequest represents the request to create a new ledger entry
type CreateLedgerEntryRequest struct {
	Account string `json:"account" binding:"required"`
	Party   string `json:"party" binding:"required"`
	Debit   int64  `json:"debit" binding:"min=0"`
	Credit  int64  `json:"credit" binding:"min=0"`
	Date    string `json:"date" binding:"required"`
	AssetID *uint  `json:"asset_id,omitempty"`
}

// ReverseLedgerEntryRequest represents the request to reverse a ledger entry.
type ReverseLedgerEntryRequest struct {
	Reason string `json:"reason,omitempty"`
}

// LedgerEntryResponse represents the response containing ledger entry data
type LedgerEntryResponse struct {
	ID        uint           `json:"id"`
	Account   string         `json:"account"`
	Party     string         `json:"party"`
	Debit     int64          `json:"debit"`
	Credit    int64          `json:"credit"`
	Balance   int64          `json:"balance"`    // Running ledger balance (cash - payable + receivable)
	NetAmount int64          `json:"net_amount"` // Intuitive balance for this entry
	Date      string         `json:"date"`
	AssetID   *uint          `json:"asset_id"`
	Asset     *AssetResponse `json:"asset,omitempty"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	// ReversalOfEntryID is set when this row is a mirror written by a reversal:
	// it names the entry it offsets. ReversalReason carries the operator's reason
	// for that reversal.
	ReversalOfEntryID *uint  `json:"reversal_of_entry_id,omitempty"`
	ReversalReason    string `json:"reversal_reason,omitempty"`
	// IsReversed tells the client that this entry already has a mirror, so the
	// reversal action can be hidden instead of offered and then refused.
	IsReversed bool `json:"is_reversed"`
}

// ListLedgerEntriesResponse represents the response for listing ledger entries
type ListLedgerEntriesResponse struct {
	Entries []LedgerEntryResponse `json:"entries"`
	Total   int64                 `json:"total"`
}

// CashFlowSummaryResponse represents the response for cash flow summary
type CashFlowSummaryResponse struct {
	StartDate      string                            `json:"start_date"`
	EndDate        string                            `json:"end_date"`
	TotalInflow    int64                             `json:"total_inflow"`
	TotalOutflow   int64                             `json:"total_outflow"`
	NetCashFlow    int64                             `json:"net_cash_flow"`
	OpeningBalance int64                             `json:"opening_balance"`
	ClosingBalance int64                             `json:"closing_balance"`
	ByAccount      map[string]AccountSummaryResponse `json:"by_account"`
}

// AccountSummaryResponse represents account summary in cash flow
type AccountSummaryResponse struct {
	TotalDebits  int64 `json:"total_debits"`
	TotalCredits int64 `json:"total_credits"`
	NetAmount    int64 `json:"net_amount"`
}

// LedgerSummaryResponse represents the ledger summary API response
type LedgerSummaryResponse struct {
	Period    PeriodInfo                `json:"period"`
	Totals    TotalInfo                 `json:"totals"`
	ByAccount map[string]AccountSummary `json:"by_account"`
	ByOwners  []OwnerContribution       `json:"by_owners"`
}

// PeriodInfo contains the date range information
type PeriodInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TotalInfo contains overall balance information
type TotalInfo struct {
	OpeningBalance int64 `json:"opening_balance"`
	ClosingBalance int64 `json:"closing_balance"`
	NetCashflow    int64 `json:"net_cashflow"`
}

// AccountSummary contains account-specific summary
type AccountSummary struct {
	Debit     int64 `json:"debit"`
	Credit    int64 `json:"credit"`
	NetAmount int64 `json:"net_amount"` // Intuitive balance for user display
}

// CreateBulkLedgerEntriesRequest represents the request to create one or more ledger entries
// It can accept both a single object or an array of objects
type CreateBulkLedgerEntriesRequest []CreateLedgerEntryRequest

// BulkLedgerEntriesResponse represents the response for creating bulk ledger entries
type BulkLedgerEntriesResponse struct {
	Entries      []LedgerEntryResponse `json:"entries"`
	TotalEntries int                   `json:"total_entries"`
	TotalDebits  int64                 `json:"total_debits"`
	TotalCredits int64                 `json:"total_credits"`
}

// AccountMetadataResponse represents account type metadata
type AccountMetadataResponse struct {
	Value      string `json:"value"`
	Label      string `json:"label"`
	Category   string `json:"category"`
	NormalSide string `json:"normal_side"` // "debit" or "credit"
}

// AccountsMetadataResponse represents the response for account metadata
type AccountsMetadataResponse struct {
	Accounts []AccountMetadataResponse `json:"accounts"`
}

// OwnerContribution represents capital contribution by an owner
type OwnerContribution struct {
	Owner             string  `json:"owner"`
	TotalContribution int64   `json:"total_contribution"`
	Percentage        float64 `json:"percentage"`
}
