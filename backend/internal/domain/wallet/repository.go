package wallet

import (
	"context"
	"time"
)

// WalletTopupRepository defines the interface for wallet topup persistence.
type WalletTopupRepository interface {
	Create(ctx context.Context, topup *WalletTopup) error
	GetByID(ctx context.Context, id uint64) (*WalletTopup, error)
	GetByBankRef(ctx context.Context, bankRef string) (*WalletTopup, error)
	Sum(ctx context.Context) (int64, error)
	SumByDateRange(ctx context.Context, start, end time.Time) (int64, error)
	List(ctx context.Context, filter WalletTopupFilter) ([]*WalletTopup, int64, error)
}

// WalletPaymentRepository defines the interface for wallet payment persistence.
type WalletPaymentRepository interface {
	Create(ctx context.Context, payment *WalletPayment) error
	GetByID(ctx context.Context, id uint64) (*WalletPayment, error)
	GetByTxnID(ctx context.Context, txnID string) (*WalletPayment, error)
	GetByRequestID(ctx context.Context, requestID string) (*WalletPayment, error)
	GetByProviderInvoiceNo(ctx context.Context, provider, invoiceNo string) (*WalletPayment, error)
	Update(ctx context.Context, payment *WalletPayment) error
	UpdateStatus(ctx context.Context, id uint64, status string, settledAt, reconciledAt *time.Time) error
	List(ctx context.Context, filter WalletPaymentFilter) ([]*WalletPayment, int64, error)
	SumByStatuses(ctx context.Context, statuses []string) (int64, error)
	SumUnreconciledByStatuses(ctx context.Context, statuses []string) (int64, error)
	// GetCompletedUnsettled returns completed wallet payments whose linked
	// advance_payment_requests have NOT yet been ledger-settled
	// (settlement_transaction_id IS NULL). Payments are bucketed by completion
	// day using settled_at (created_at fallback), not updated_at which advances
	// on every write. When date is the zero value, every stranded payment is
	// returned regardless of completion day so the worker can backfill days
	// whose single pickup run was missed; otherwise only payments completed on
	// that calendar day are returned. Results are ordered by completion time.
	GetCompletedUnsettled(ctx context.Context, date time.Time) ([]*WalletPayment, error)
}

// WalletIPNRepository defines the interface for wallet IPN audit persistence.
type WalletIPNRepository interface {
	Create(ctx context.Context, ipn *WalletIPN) (uint64, error)
	UpdateProcessingResult(ctx context.Context, id uint64, status string, walletPaymentID *uint64, processingError string) error
	ListByInvoiceNo(ctx context.Context, invoiceNo string) ([]*WalletIPN, error)
}
