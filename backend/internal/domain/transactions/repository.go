package transactions

import (
	"context"
	"time"
)

// WalletPaymentRepository persists wallet_payment rows.
type WalletPaymentRepository interface {
	Create(ctx context.Context, t *WalletPayment) error
	GetByID(ctx context.Context, id uint64) (*WalletPayment, error)
	GetByRequestID(ctx context.Context, requestID string) (*WalletPayment, error)
	GetByProviderInvoiceNo(ctx context.Context, provider, invoiceNo string) (*WalletPayment, error)

	// TODO: TASK-039 — ListByProviderStatuses for per-provider orphan recovery.
	// ListByProviderStatuses(ctx context.Context, provider string, statuses []State) ([]*WalletPayment, error)
	GetByTxnID(ctx context.Context, txnID string) (*WalletPayment, error)
	ListRecent(ctx context.Context, limit int) ([]*WalletPayment, error)

	// UpdateExpected applies the patch and bumps version, but only if
	// the row's current version equals expectedVersion.
	UpdateExpected(ctx context.Context, id uint64, expectedVersion int64, patch UpdatePatch) error

	// MarkReconciled directly updates status and reconciled_at, bypassing
	// the FSM. Used by reconciliation when overriding status (e.g. failed → completed).
	// Guarded by a status precondition: only rows in a reconcilable prior state
	// (failed/completed/authorised/verified) are updated; others return ErrNotFound.
	MarkReconciled(ctx context.Context, id uint64, status State, reconciledAt time.Time) error

	// ListByStatuses returns rows matching any of the given statuses within
	// the date range [from, to), ordered by created_at ASC.
	ListByStatuses(ctx context.Context, statuses []State, from, to time.Time) ([]*WalletPayment, error)

	StatsByErrorCode(ctx context.Context, from, to time.Time) ([]ErrorCodeStat, error)
	StatsByStatus(ctx context.Context, from, to time.Time) ([]StatusStat, error)
	ListByBatchID(ctx context.Context, batchID string) ([]*WalletPayment, error)

	// ListByProviderAndCreatedRange returns unreconciled rows for a provider
	// within [from, to), ordered by created_at ASC. Used by inquiry-based
	// reconciliation (e.g. OnePay) to poll individual transfer status.
	ListByProviderAndCreatedRange(ctx context.Context, provider string, from, to time.Time) ([]*WalletPayment, error)

	// ListStaleAuthorised returns authorised payments older than cutoff for a
	// given provider, ordered oldest-first. Used by the status inquiry poller
	// to resolve stuck payments when IPN has not arrived.
	ListStaleAuthorised(ctx context.Context, provider string, cutoff time.Time, limit int) ([]*WalletPayment, error)

	// HasPendingForRecipient checks whether a non-terminal (pending or authorised)
	// wallet payment already exists for the given recipient + provider.
	// Used to prevent double disbursement when both auto-poller and manual
	// admin flows target the same employee simultaneously.
	HasPendingForRecipient(ctx context.Context, accountNo, bank, provider string) (bool, error)
}

// UpdatePatch captures the mutable fields of a WalletPayment.
type UpdatePatch struct {
	Status           *State
	InvoiceNo        *string
	ErrorCode        *string
	ErrorMessage     *string
	Description      *string
	Fee              *int64
	ReconciledAt     *time.Time
	ResolutionSource *string
}

type ErrorCodeStat struct {
	ErrorCode      string
	Count          int64
	LastOccurredAt *time.Time
}

type StatusStat struct {
	Status               State
	Count                int64
	TotalRequestedAmount int64
	TotalFee             int64
}
