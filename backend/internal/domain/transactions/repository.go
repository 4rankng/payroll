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

	// HasNonTerminalByEntityID reports whether any wallet_payment row linked to
	// the given advance_payment_requests.id (column entity_id) is still in a
	// non-terminal state (pending/verified/authorised). Used by the retry-
	// disbursement endpoint to refuse enqueuing a second concurrent task while
	// one is already in flight. Terminal rows (completed/failed/reversed) and
	// rows with a NULL entity_id are ignored.
	HasNonTerminalByEntityID(ctx context.Context, entityID uint64) (bool, error)

	// Bulk-transfer worker helpers (Phase 3 of the wallet bulk transfer pipeline).
	// All operate on wallet_payments rows linked to a bulk_transfer_batch via
	// bulk_transfer_batch_id (migration 093).

	// UpdateBulkBatchLink stamps the (batch_id, order, vfic_code) linkage onto
	// an existing row. Idempotent — safe to call on retry. Does NOT touch
	// entity_id (notification path is handled separately in notifyEmployee).
	UpdateBulkBatchLink(ctx context.Context, rowID uint64, batchID uint64, order uint, vficCode string) error

	// CountByBatchAndStatuses returns SELECT COUNT(*) WHERE bulk_transfer_batch_id=?
	// AND status IN (?). Used by markRowTerminal to detect batch completion
	// inside the lock transaction.
	CountByBatchAndStatuses(ctx context.Context, batchID uint64, statuses []State) (int64, error)

	// SumFeeByBatchAndStatuses returns SELECT COALESCE(SUM(fee),0) WHERE
	// bulk_transfer_batch_id=? AND status IN (?). Used by book_batch_ledger
	// to compute the aggregate Expense amount. Includes failed rows whose
	// fee wasn't waived (OnePay charges per call to the transfer endpoint
	// regardless of success), excludes pre-flight rejections (fee=0).
	SumFeeByBatchAndStatuses(ctx context.Context, batchID uint64, statuses []State) (int64, error)

	// ListByBatchIDOrdered returns all wallet_payments rows for a batch ordered
	// by bulk_transfer_order ASC. Used by KQ generation.
	ListByBatchIDOrdered(ctx context.Context, batchID uint64) ([]*WalletPayment, error)

	// IncrementSweeperRetry bumps sweeper_retry_count and returns the new value.
	// Used by the stale-enqueue sweeper to cap retries at 3 per row.
	IncrementSweeperRetry(ctx context.Context, rowID uint64) (uint, error)
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
