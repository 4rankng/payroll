package bulktransfer

import (
	"context"
)

// ParsedResultRow represents a normalized row from a bank result file
type ParsedResultRow struct {
	STT             int
	AccountNumber   string
	AccountName     string
	BankName        string
	Amount          float64
	TransactionCode string // For new strategy
	Status          string // "completed" or "failed"
	BankTxnRef      string // Bank reference OR error message
	ErrorMessage    string
	// Legacy fields (for H2/H3 strategy)
	ProjectID    uint
	TimesheetIDs []uint
}

// UpdateStats tracks statistics for data updates
type UpdateStats struct {
	TotalTransactions int
	MatchedCount      int
	CompletedCount    int
	FailedCount       int
	UnmatchedCount    int
}

// ResultProcessingStrategy defines the interface for processing bank result files
// Supports both legacy (H2/H3 metadata) and new (transaction_code based) formats
type ResultProcessingStrategy interface {
	// Detect returns true if this strategy can handle the given file
	Detect(rows [][]string) bool

	// ParseRows parses all data rows and returns normalized results
	ParseRows(ctx context.Context, rows [][]string) ([]*ParsedResultRow, error)

	// GetLookupKey returns the key to look up the corresponding bulk_transfer_file
	// For legacy: returns project_id + timesheet_ids combination
	// For new: returns checksum calculated from transaction codes
	GetLookupKey(ctx context.Context, rows [][]string) (string, error)

	// Name returns the strategy name for logging
	Name() string
}
