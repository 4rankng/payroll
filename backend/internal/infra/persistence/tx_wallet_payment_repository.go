package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domaintx "api-server/internal/domain/transactions"

	"gorm.io/gorm"
)

// TxWalletPaymentRepository persists wallet_payments rows.
// The interesting part is UpdateExpected — every UPDATE that mutates
// state-machine fields carries the row's expected version and is
// rejected (ErrConcurrentModification) if another writer has bumped it.
// See state_machine.go for the FSM that decides which transitions are
// legal, and service.go for the retry loop that handles version misses.
type TxWalletPaymentRepository struct {
	*BaseRepository
}

// NewTxWalletPaymentRepository wires a repository onto the shared
// database. Returns the domain interface to keep service-layer wiring
// dependent on the port, not the concrete struct.
func NewTxWalletPaymentRepository(db *Database) domaintx.WalletPaymentRepository {
	return &TxWalletPaymentRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new wallet_payments row. Version is left at
// the column default (0) on INSERT; subsequent UpdateExpected calls
// must pass version=0 the first time, then version=1, and so on.
func (r *TxWalletPaymentRepository) Create(ctx context.Context, t *domaintx.WalletPayment) error {
	if err := r.DB.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("wallet_payments: create: %w", err)
	}
	return nil
}

// GetByID returns the row with the given primary key, or ErrNotFound.
func (r *TxWalletPaymentRepository) GetByID(ctx context.Context, id uint64) (*domaintx.WalletPayment, error) {
	var row domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaintx.ErrNotFound
		}
		return nil, fmt.Errorf("wallet_payments: get by id: %w", err)
	}
	return &row, nil
}

// GetByRequestID looks up by our internal idempotency key. Useful as a
// fallback in IPN processing if invoice_no isn't available.
func (r *TxWalletPaymentRepository) GetByRequestID(ctx context.Context, requestID string) (*domaintx.WalletPayment, error) {
	var row domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).First(&row, "request_id = ?", requestID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaintx.ErrNotFound
		}
		return nil, fmt.Errorf("wallet_payments: get by request_id: %w", err)
	}
	return &row, nil
}

// GetByProviderInvoiceNo looks up by the (provider, invoice_no) composite
// key. The IPN webhook carries invoice_no; provider disambiguates when
// multiple providers coexist. Returns ErrNotFound when no row matches.
func (r *TxWalletPaymentRepository) GetByProviderInvoiceNo(ctx context.Context, provider, invoiceNo string) (*domaintx.WalletPayment, error) {
	var row domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).First(&row, "provider = ? AND invoice_no = ?", provider, invoiceNo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaintx.ErrNotFound
		}
		return nil, fmt.Errorf("wallet_payments: get by provider+invoice_no: %w", err)
	}
	return &row, nil
}

// GetByTxnID looks up by the public-facing UUID txn_id. Used by the
// manual-disbursement status polling endpoint.
func (r *TxWalletPaymentRepository) GetByTxnID(ctx context.Context, txnID string) (*domaintx.WalletPayment, error) {
	var row domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).First(&row, "txn_id = ?", txnID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaintx.ErrNotFound
		}
		return nil, fmt.Errorf("wallet_payments: get by txn_id: %w", err)
	}
	return &row, nil
}

// ListRecent returns the most recent rows ordered by created_at DESC.
// limit caps the result count; 0 or negative is treated as 20.
func (r *TxWalletPaymentRepository) ListRecent(ctx context.Context, limit int) ([]*domaintx.WalletPayment, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []*domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_payments: list recent: %w", err)
	}
	return rows, nil
}

// UpdateExpected applies patch and bumps version, but only if the row's
// current version equals expectedVersion. Returns ErrConcurrentModification
// when rows-affected = 0, so callers can re-fetch and retry.
//
// settled_at is set to NOW() on the first transition into a terminal
// state — i.e. when patch.Status is set to a terminal value. We set it
// unconditionally on every terminal-state UPDATE; that's safe because
// the FSM rejects all transitions out of a terminal row except
// completed → reversed, and the late-reversal IPN should overwrite the
// original settled_at with the reversal time anyway.
func (r *TxWalletPaymentRepository) UpdateExpected(ctx context.Context, id uint64, expectedVersion int64, patch domaintx.UpdatePatch) error {
	updates := map[string]any{
		"version":    gorm.Expr("version + 1"),
		"updated_at": clock.Now(),
	}
	if patch.Status != nil {
		updates["status"] = string(*patch.Status)
		if domaintx.IsTerminal(*patch.Status) {
			updates["settled_at"] = clock.Now()
		}
	}
	if patch.InvoiceNo != nil {
		updates["invoice_no"] = *patch.InvoiceNo
	}
	if patch.ErrorCode != nil {
		updates["error_code"] = *patch.ErrorCode
	}
	if patch.ErrorMessage != nil {
		updates["error_message"] = *patch.ErrorMessage
	}
	if patch.Description != nil {
		updates["description"] = *patch.Description
	}
	if patch.Fee != nil {
		updates["fee"] = *patch.Fee
	}
	if patch.ReconciledAt != nil {
		updates["reconciled_at"] = *patch.ReconciledAt
	}
	if patch.ResolutionSource != nil {
		updates["resolution_source"] = *patch.ResolutionSource
	}

	res := r.DB.WithContext(ctx).
		Model(&domaintx.WalletPayment{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("wallet_payments: update expected: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domaintx.ErrConcurrentModification
	}
	return nil
}

// StatsByErrorCode aggregates rows by error_code over [from, to]. NULL
// error codes are coalesced into the empty string key. Sorted by count
// DESC then error_code ASC for stable presentation.
func (r *TxWalletPaymentRepository) StatsByErrorCode(ctx context.Context, from, to time.Time) ([]domaintx.ErrorCodeStat, error) {
	// LastOccurredAt scans as sql.NullString so the same code path works
	// against both MySQL DATETIME(3) and SQLite's TEXT-stored datetimes
	// (the test environment). MySQL's driver returns time.Time directly
	// for the column, but the standard library can convert that to a
	// string scan target; SQLite returns a string. We parse on the way
	// out into the domain DTO's *time.Time.
	type row struct {
		ErrorCode      string         `gorm:"column:error_code"`
		Count          int64          `gorm:"column:cnt"`
		LastOccurredAt sql.NullString `gorm:"column:last_occurred_at"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).
		Table("wallet_payments").
		Select("COALESCE(error_code, '') AS error_code, COUNT(*) AS cnt, MAX(created_at) AS last_occurred_at").
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("error_code").
		Order("cnt DESC, error_code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("wallet_payments: stats by error_code: %w", err)
	}
	out := make([]domaintx.ErrorCodeStat, len(rows))
	for i, rr := range rows {
		stat := domaintx.ErrorCodeStat{
			ErrorCode: rr.ErrorCode,
			Count:     rr.Count,
		}
		if rr.LastOccurredAt.Valid {
			if t, ok := parseDatetime(rr.LastOccurredAt.String); ok {
				stat.LastOccurredAt = &t
			}
		}
		out[i] = stat
	}
	return out, nil
}

// parseDatetime is a small helper for the few group-by queries where
// MAX(datetime_column) returns through a string scan path. MySQL's driver
// formats datetimes as "2006-01-02 15:04:05" or "2006-01-02 15:04:05.000";
// SQLite emits the same with no fractional seconds. Try both layouts and
// fall back to an empty zero-time on parse failure (caller treats nil as
// "unknown").
func parseDatetime(s string) (time.Time, bool) {
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// StatsByStatus aggregates rows by status over [from, to]. SUM(fee)
// is plain int64 because fee is NOT NULL DEFAULT 0 — every row carries
// a numeric value (rows pre-migration-049 backfilled to 0). COALESCE
// guards the empty-bucket case where SUM returns NULL.
func (r *TxWalletPaymentRepository) StatsByStatus(ctx context.Context, from, to time.Time) ([]domaintx.StatusStat, error) {
	type row struct {
		Status               string `gorm:"column:status"`
		Count                int64  `gorm:"column:cnt"`
		TotalRequestedAmount int64  `gorm:"column:total_requested"`
		TotalFee             int64  `gorm:"column:total_fee"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).
		Table("wallet_payments").
		Select(`status,
		         COUNT(*) AS cnt,
		         COALESCE(SUM(requested_amount), 0) AS total_requested,
		         COALESCE(SUM(fee), 0) AS total_fee`).
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("status").
		Order("status ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("wallet_payments: stats by status: %w", err)
	}
	out := make([]domaintx.StatusStat, len(rows))
	for i, r := range rows {
		out[i] = domaintx.StatusStat{
			Status:               domaintx.State(r.Status),
			Count:                r.Count,
			TotalRequestedAmount: r.TotalRequestedAmount,
			TotalFee:             r.TotalFee,
		}
	}
	return out, nil
}

// reconcilablePriorStates are the wallet_payment statuses from which the
// reconciliation path may override status. Sourced from
// WalletPaymentService.ReconcilePayment's switch (failed / completed /
// authorised / verified). Rows in any other status (pending, reversed) are
// not reconcilable and MarkReconciled leaves them untouched.
var reconcilablePriorStates = []domaintx.State{
	domaintx.StateFailed,
	domaintx.StateCompleted,
	domaintx.StateAuthorised,
	domaintx.StateVerified,
}

// MarkReconciled directly updates status and reconciled_at, bypassing
// the FSM optimistic-locking path. Used by reconciliation when overriding
// status (e.g. failed -> completed after recon contradicts IPN). Guards the
// override with a status precondition: only rows currently in a reconcilable
// prior state (see reconcilablePriorStates) are updated, so a pending/reversed
// row cannot be silently flipped by the reconcile path.
func (r *TxWalletPaymentRepository) MarkReconciled(ctx context.Context, id uint64, status domaintx.State, reconciledAt time.Time) error {
	priorStates := make([]string, len(reconcilablePriorStates))
	for i, s := range reconcilablePriorStates {
		priorStates[i] = string(s)
	}
	updates := map[string]any{
		"status":            string(status),
		"reconciled_at":     reconciledAt,
		"resolution_source": domaintx.ResolutionSourceReconciliation,
		"version":           gorm.Expr("version + 1"),
		"updated_at":        clock.Now(),
	}
	res := r.DB.WithContext(ctx).
		Table("wallet_payments").
		Where("id = ? AND status IN ?", id, priorStates).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("wallet_payments: mark reconciled: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domaintx.ErrNotFound
	}
	return nil
}

// ListByStatuses returns rows matching any of the given statuses within
// the date range [from, to), ordered by created_at ASC.
func (r *TxWalletPaymentRepository) ListByStatuses(ctx context.Context, statuses []domaintx.State, from, to time.Time) ([]*domaintx.WalletPayment, error) {
	statusStrs := make([]string, len(statuses))
	for i, s := range statuses {
		statusStrs[i] = string(s)
	}
	var rows []*domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).
		Where("status IN ? AND created_at >= ? AND created_at < ?", statusStrs, from, to).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_payments: list by statuses: %w", err)
	}
	return rows, nil
}

// ListByBatchID returns all wallet_payments rows matching the given batch_id.
func (r *TxWalletPaymentRepository) ListByBatchID(ctx context.Context, batchID string) ([]*domaintx.WalletPayment, error) {
	var rows []*domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).
		Where("batch_id = ?", batchID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_payments: list by batch_id: %w", err)
	}
	return rows, nil
}

// ListByProviderAndCreatedRange returns unreconciled rows for a provider
// within [from, to), ordered by created_at ASC. Used by inquiry-based
// reconciliation (e.g. OnePay) to poll individual transfer status.
func (r *TxWalletPaymentRepository) ListByProviderAndCreatedRange(ctx context.Context, provider string, from, to time.Time) ([]*domaintx.WalletPayment, error) {
	var rows []*domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).
		Where("provider = ? AND created_at >= ? AND created_at < ? AND reconciled_at IS NULL", provider, from, to).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_payments: list by provider and created range: %w", err)
	}
	return rows, nil
}

// ListStaleAuthorised returns authorised payments older than cutoff for a
// given provider, ordered oldest-first. Used by the status inquiry poller
// to resolve stuck payments when IPN has not arrived.
func (r *TxWalletPaymentRepository) ListStaleAuthorised(ctx context.Context, provider string, cutoff time.Time, limit int) ([]*domaintx.WalletPayment, error) {
	var rows []*domaintx.WalletPayment
	if err := r.DB.WithContext(ctx).
		Where("provider = ? AND status = ? AND updated_at < ?",
			provider, string(domaintx.StateAuthorised), cutoff).
		Order("updated_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_payments: list stale authorised: %w", err)
	}
	return rows, nil
}

func (r *TxWalletPaymentRepository) HasPendingForRecipient(ctx context.Context, accountNo, bank, provider string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&domaintx.WalletPayment{}).
		Where("recipient_account_no = ? AND recipient_bank = ? AND provider = ? AND status IN ?",
			accountNo, bank, provider, []domaintx.State{domaintx.StatePending, domaintx.StateAuthorised}).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("wallet_payments: check pending for recipient: %w", err)
	}
	return count > 0, nil
}

// HasNonTerminalByEntityID reports whether any wallet_payment linked to the
// given advance_payment_requests.id (entity_id) is still in flight. The
// terminal states (completed/failed/reversed) are excluded; everything else
// (pending/verified/authorised) counts as in-flight, so the retry-disbursement
// endpoint can refuse to pile a second task onto an active disbursement.
// entity_id is indexed; NULL rows never match the equality predicate.
func (r *TxWalletPaymentRepository) HasNonTerminalByEntityID(ctx context.Context, entityID uint64) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&domaintx.WalletPayment{}).
		Where("entity_id = ? AND status NOT IN ?",
			entityID,
			[]domaintx.State{domaintx.StateCompleted, domaintx.StateFailed, domaintx.StateReversed}).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("wallet_payments: check non-terminal by entity_id: %w", err)
	}
	return count > 0, nil
}

// UpdateBulkBatchLink stamps the (bulk_transfer_batch_id, bulk_transfer_order,
// vfic_code) linkage columns onto an existing row. Idempotent — safe to call
// on asynq retry. Used by the bulk-transfer row worker after Initiate returns
// the row, so SUM(fee) GROUP BY bulk_transfer_batch_id works at ledger time.
// Does NOT touch entity_id (notification path is resolved separately in
// notifyEmployee).
func (r *TxWalletPaymentRepository) UpdateBulkBatchLink(ctx context.Context, rowID uint64, batchID uint64, order uint, vficCode string) error {
	updates := map[string]interface{}{
		"bulk_transfer_batch_id": batchID,
		"bulk_transfer_order":    order,
		"vfic_code":              vficCode,
	}
	if err := r.DB.WithContext(ctx).
		Model(&domaintx.WalletPayment{}).
		Where("id = ?", rowID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("wallet_payments: update bulk batch link: %w", err)
	}
	return nil
}

// CountByBatchAndStatuses returns SELECT COUNT(*) WHERE
// bulk_transfer_batch_id=? AND status IN (?). Used by the bulk-transfer row
// worker's markRowTerminal to detect batch completion inside the lock tx.
func (r *TxWalletPaymentRepository) CountByBatchAndStatuses(ctx context.Context, batchID uint64, statuses []domaintx.State) (int64, error) {
	if len(statuses) == 0 {
		return 0, nil
	}
	var count int64
	err := r.DB.WithContext(ctx).Model(&domaintx.WalletPayment{}).
		Where("bulk_transfer_batch_id = ? AND status IN ?", batchID, statuses).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("wallet_payments: count by batch+statuses: %w", err)
	}
	return count, nil
}

// SumFeeByBatchAndStatuses returns SELECT COALESCE(SUM(fee),0) WHERE
// bulk_transfer_batch_id=? AND status IN (?). Used by book_batch_ledger to
// compute the aggregate Expense amount. Includes failed rows whose fee wasn't
// waived (OnePay charges per call to the transfer endpoint regardless of
// outcome); pre-flight rejections carry fee=0 (syncPatch zeroes it via
// FeeWaived=true) so they contribute 0 naturally.
func (r *TxWalletPaymentRepository) SumFeeByBatchAndStatuses(ctx context.Context, batchID uint64, statuses []domaintx.State) (int64, error) {
	if len(statuses) == 0 {
		return 0, nil
	}
	var total sql.NullInt64
	err := r.DB.WithContext(ctx).
		Model(&domaintx.WalletPayment{}).
		Where("bulk_transfer_batch_id = ? AND status IN ?", batchID, statuses).
		Select("COALESCE(SUM(fee), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("wallet_payments: sum fee by batch+statuses: %w", err)
	}
	return total.Int64, nil
}

// ListByBatchIDOrdered returns all wallet_payments rows for a batch ordered
// by bulk_transfer_order ASC (NULLS LAST), used by the KQ Excel generator to
// render rows in original input order.
func (r *TxWalletPaymentRepository) ListByBatchIDOrdered(ctx context.Context, batchID uint64) ([]*domaintx.WalletPayment, error) {
	var rows []*domaintx.WalletPayment
	err := r.DB.WithContext(ctx).
		Where("bulk_transfer_batch_id = ?", batchID).
		Order("bulk_transfer_order ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("wallet_payments: list by batch ordered: %w", err)
	}
	return rows, nil
}

// IncrementSweeperRetry bumps sweeper_retry_count and returns the new value.
// Used by the stale-enqueue sweeper to cap per-row retries at 3.
func (r *TxWalletPaymentRepository) IncrementSweeperRetry(ctx context.Context, rowID uint64) (uint, error) {
	var result struct {
		Count uint `gorm:"column:count"`
	}
	err := r.DB.WithContext(ctx).
		Model(&domaintx.WalletPayment{}).
		Where("id = ?", rowID).
		UpdateColumn("sweeper_retry_count", gorm.Expr("sweeper_retry_count + 1")).Error
	if err != nil {
		return 0, fmt.Errorf("wallet_payments: increment sweeper retry: %w", err)
	}
	if err := r.DB.WithContext(ctx).
		Model(&domaintx.WalletPayment{}).
		Where("id = ?", rowID).
		Select("sweeper_retry_count AS count").
		Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("wallet_payments: read sweeper retry count: %w", err)
	}
	return result.Count, nil
}
