package wallet

import (
	"context"
	"errors"
)

// Sentinel errors for SyncBalance — the handler uses these to return the
// correct HTTP status code (400 for config errors, 500 for runtime failures).
var (
	ErrProviderNotConfigured      = errors.New("wallet: no payment provider configured")
	ErrProviderBalanceUnsupported = errors.New("wallet: provider does not support balance inquiry")
)

// SyncBalanceResult holds the outcome of a balance sync with the provider.
type SyncBalanceResult struct {
	ProviderBalance int64
	LocalBalance    int64
	Adjusted        bool
}

// WalletService defines the interface for wallet business logic.
type WalletService interface {
	// Balance (computed from DB, no external API)
	GetBalance(ctx context.Context) (*WalletBalance, error)

	// SyncBalance fetches provider balance, compares with local, auto-adjusts if mismatch.
	SyncBalance(ctx context.Context, userID uint64) (*SyncBalanceResult, error)

	// Unified transaction view
	GetTransactions(ctx context.Context, filter TransactionFilter) ([]*UnifiedTransaction, int64, error)

	// Topups (immutable, no status)
	CreateTopup(ctx context.Context, req CreateWalletTopupRequest, userID uint64) (*WalletTopup, error)
	GetTopupByID(ctx context.Context, id uint64) (*WalletTopup, error)
	ListTopups(ctx context.Context, filter WalletTopupFilter) ([]*WalletTopup, int64, error)

	// Payments (read-only from wallet, written by disbursement service)
	GetPaymentByID(ctx context.Context, id uint64) (*WalletPayment, error)
	GetPaymentByTxnID(ctx context.Context, txnID string) (*WalletPayment, error)
	GetPaymentByRequestID(ctx context.Context, requestID string) (*WalletPayment, error)
	ListPayments(ctx context.Context, filter WalletPaymentFilter) ([]*WalletPayment, int64, error)
	ResolvePayment(ctx context.Context, id uint64, action string, reason string, adminUserID uint64) (*WalletPayment, error)

	// Reconciliation (9pay CSV matching against wallet_payments only)
	UploadReconciliation(ctx context.Context, csvData []byte, userID uint64) (string, error)
	GetReconciliationJobStatus(ctx context.Context, jobID string) (*ReconciliationJob, error)

	// Monthly reconciliation report export (cycle: 16th prev month to 15th current month)
	ExportReconciliationReport(ctx context.Context, month string) ([]byte, error)
}
