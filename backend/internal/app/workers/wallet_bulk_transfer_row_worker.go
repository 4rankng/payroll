package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/wallet_bulk"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

// WalletBulkTransferRowWorker processes ONE row of an uploaded bulk transfer
// file. It mirrors disbursement_execute_worker.go:73-237 line-for-line, with
// these key additions:
//
//  1. Step 0 pre-flight balance check (C6 fix): if the wallet can't cover
//     this row's amount, skip without charging a fee. Mirrors canonical Step 0.
//  2. Terminal-error accounting (C1 fix): when Initiate returns
//     ErrDuplicatePaymentInProgress or ErrFeeResolution, no wallet_payment
//     row was inserted. We decrement the batch's total_count so the
//     completion threshold (success+failed==total) still fires.
//  3. ensureBulkBatchLink stamps (bulk_transfer_batch_id, bulk_transfer_order,
//     vfic_code) onto the wallet_payment so SUM(fee) GROUP BY
//     bulk_transfer_batch_id works at ledger time.
//  4. markRowTerminal atomically detects batch completion (via UpdateWithLock)
//     and enqueues the book_batch_ledger task if this was the last row.
//
// The wallet_payments insertion is delegated to WalletPaymentService.Initiate
// (sole insertion point — fee stamps correctly from the schedule at INSERT).
type WalletBulkTransferRowWorker struct {
	walletPaymentSvc WalletPaymentInitiator
	walletSvc        WalletBalanceReader // for Step 0 balance guard; nil = skip guard
	registry         DisbursementRegistry
	paymentRepo      domaintx.WalletPaymentRepository
	batchRepo        domain.BulkTransferBatchRepository
	asynqClient      wallet_bulk.BulkTransferEnqueuer
	logger           *slog.Logger
}

// WalletPaymentInitiator is the narrow port the worker needs from
// disbursement.WalletPaymentService. Declared here to keep the worker
// testable without the full service.
type WalletPaymentInitiator interface {
	Initiate(ctx context.Context, in disbursement.InitiateInput) (*domaintx.WalletPayment, error)
	RecordAccountCheck(ctx context.Context, requestID string, outcome disbursement.AccountCheckOutcome) (*domaintx.WalletPayment, error)
	RecordSyncResponse(ctx context.Context, requestID string, result disbursement.SyncResult) (*domaintx.WalletPayment, error)
}

// WalletBalanceReader is the narrow port for the Step-0 pre-flight balance
// check. Implemented by wallet.WalletService; nil-safe (worker skips the
// guard when nil — useful in tests).
type WalletBalanceReader interface {
	GetBalance(ctx context.Context) (*wallet.WalletBalance, error)
}

// DisbursementRegistry is the narrow port for resolving the active provider.
type DisbursementRegistry interface {
	Active(ctx context.Context) (infrastructure.DisbursementProvider, error)
}

// NewWalletBulkTransferRowWorker wires the worker. walletSvc may be nil —
// when nil, the Step-0 balance guard is skipped (mirrors the canonical
// worker's `if w.walletSvc != nil` guard).
func NewWalletBulkTransferRowWorker(
	walletPaymentSvc WalletPaymentInitiator,
	walletSvc WalletBalanceReader,
	registry DisbursementRegistry,
	paymentRepo domaintx.WalletPaymentRepository,
	batchRepo domain.BulkTransferBatchRepository,
	asynqClient wallet_bulk.BulkTransferEnqueuer,
	logger *slog.Logger,
) *WalletBulkTransferRowWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &WalletBulkTransferRowWorker{
		walletPaymentSvc: walletPaymentSvc,
		walletSvc:        walletSvc,
		registry:         registry,
		paymentRepo:      paymentRepo,
		batchRepo:        batchRepo,
		asynqClient:      asynqClient,
		logger:           logger,
	}
}

// ProcessJob handles one wallet:bulk_transfer_row task.
//
// Steps mirror disbursement_execute_worker.go:73-237:
//
//  1. Initiate wallet_payment (sole insertion point — fee stamps here).
//     - ErrDuplicatePaymentInProgress → return nil (already in-flight).
//     - ErrFeeResolution → SkipRetry (config gap, not transient).
//  2. StatePending guard — if row is already past pending, another worker
//     processed it. Stamp batch link and check terminal.
//  3. Stamp batch link (idempotent).
//  4. Resolve active provider.
//  5. Check the beneficiary account when the provider supports it. A rejected
//     employee becomes failed with fee=0 and the other rows continue.
//  6. provider.InitiateTransfer — rate-limited by QueuedProvider (3 TPS).
//     - ErrPreflightValidation → markRowFailed(feeWaived=true).
//  7. Record sync response with explicit Accepted flag.
//  8. Synchronous rejection calls markRowTerminal immediately; accepted
//     transfers wait for their terminal IPN/status-poll result.
func (w *WalletBulkTransferRowWorker) ProcessJob(ctx context.Context, t *asynqlib.Task) error {
	var p wallet_bulk.RowTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		w.logger.Error("wallet_bulk_row: malformed payload", "error", err)
		return fmt.Errorf("unmarshal: %w: %w", err, asynqlib.SkipRetry)
	}

	logger := w.logger.With("batch_id", p.BatchID, "vfic", p.Row.VFICCode)

	// === Step 0: Pre-flight balance check (C6 fix — mirrors canonical worker Step 0).
	// OnePay charges per call to the transfer endpoint; an underfunded wallet
	// would still cost 3,850 VND per row at failure. Skip without charging.
	if w.walletSvc != nil {
		balance, balErr := w.walletSvc.GetBalance(ctx)
		if balErr != nil {
			logger.Warn("wallet_bulk_row: balance check failed, proceeding", "error", balErr)
		} else if balance != nil && balance.Available < p.Row.Amount {
			logger.Info("wallet_bulk_row: skipping — insufficient balance",
				"available", balance.Available, "requested", p.Row.Amount)
			// Decrement total_count — no row was inserted, no fee charged,
			// the orphan-recovery path (top-up + manual re-enqueue) is the
			// admin's remedy.
			return w.skipRow(ctx, p.BatchID)
		}
	}

	// === Step 1: Initiate wallet_payment (sole insertion point — fee stamps here) ===
	row, err := w.walletPaymentSvc.Initiate(ctx, disbursement.InitiateInput{
		RequestID:          p.Row.VFICCode, // ≤20 chars
		RequestedAmount:    p.Row.Amount,
		RecipientName:      p.Row.AccountName,
		RecipientAccountNo: p.Row.AccountNo,
		RecipientBank:      p.Row.SwiftCode, // SWIFT — Initiate accepts it directly
		Description:        p.Row.PaymentDetail,
		// EntityID left nil — bulk transfers aren't linked to advance_payment_requests.
		// notifyEmployee handles nil-entity_id rows via recipient_account_no.
	})
	if err != nil {
		// === CANONICAL TERMINAL-ERROR BRANCHES ===
		if errors.Is(err, disbursement.ErrDuplicatePaymentInProgress) {
			// C1 fix: no wallet_payment row was inserted, so it will never
			// count toward batch completion. Decrement total_count and run
			// the completion detector so the batch can still finish.
			logger.Info("wallet_bulk_row: skipping — duplicate payment already in progress")
			return w.skipRow(ctx, p.BatchID)
		}
		if errors.Is(err, disbursement.ErrFeeResolution) {
			// Fee schedule misconfigured — config gap, not transient. Don't
			// retry (schedule/allowlist entry must be added). Also decrement
			// total_count so the batch doesn't deadlock (C1 fix).
			logger.Error("wallet_bulk_row: fee resolution failed — terminal (config gap)",
				"error", err)
			if decErr := w.batchRepo.DecrementTotalCount(ctx, p.BatchID, 1); decErr != nil {
				logger.Error("wallet_bulk_row: decrement total_count failed", "error", decErr)
			}
			return fmt.Errorf("initiate: %w: fee config gap", asynqlib.SkipRetry)
		}
		return fmt.Errorf("initiate: %w", err)
	}

	// === CANONICAL STATE GUARD (mirror disbursement_execute_worker.go:127-131) ===
	if row.Status != domaintx.StatePending {
		logger.Info("wallet_bulk_row: skipping — already processed", "status", row.Status)
		w.ensureBulkBatchLink(ctx, row.ID, p.BatchID, p.Row)
		return w.markRowTerminal(ctx, p.BatchID)
	}

	w.ensureBulkBatchLink(ctx, row.ID, p.BatchID, p.Row)

	// === Step 2: Get active provider ===
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return fmt.Errorf("no active provider: %w", err)
	}

	// === Step 3: Verify beneficiary account when supported ===
	transferAccountName := p.Row.AccountName
	if verifier, ok := provider.(infrastructure.AccountVerifier); ok {
		check, err := verifier.CheckAccount(ctx, infrastructure.AccountCheckRequest{
			RequestID:   p.Row.VFICCode,
			BankCode:    p.Row.SwiftCode,
			SwiftCode:   p.Row.SwiftCode,
			AccountNo:   p.Row.AccountNo,
			AccountName: p.Row.AccountName,
			Amount:      p.Row.Amount,
			AccountType: infrastructure.AccountTypeBankAccount,
		})
		if err != nil {
			return fmt.Errorf("check account: %w", err)
		}
		if _, err := w.walletPaymentSvc.RecordAccountCheck(ctx, p.Row.VFICCode, disbursement.AccountCheckOutcome{
			Verified: check.Valid, RawErrorCode: check.RawErrorCode,
			RawMessage: check.RawMessage, FeeWaived: !check.Valid,
		}); err != nil {
			return fmt.Errorf("record account check: %w", err)
		}
		if !check.Valid {
			logger.Info("wallet_bulk_row: account check rejected",
				"error_code", check.RawErrorCode, "message", check.RawMessage)
			return w.markRowTerminal(ctx, p.BatchID)
		}
		if check.AccountName != "" {
			transferAccountName = check.AccountName
		}
	} else if _, err := w.walletPaymentSvc.RecordAccountCheck(ctx, p.Row.VFICCode, disbursement.AccountCheckOutcome{Verified: true}); err != nil {
		return fmt.Errorf("record account check: %w", err)
	}

	// === Step 4: Initiate transfer via provider (rate-limited by QueuedProvider) ===
	result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
		RequestID:   p.Row.VFICCode,
		Amount:      p.Row.Amount,
		SwiftCode:   p.Row.SwiftCode,
		AccountNo:   p.Row.AccountNo,
		AccountName: transferAccountName,
		Description: p.Row.PaymentDetail,
		AccountType: infrastructure.AccountTypeBankAccount,
	})
	if err != nil {
		if errors.Is(err, infrastructure.ErrPreflightValidation) {
			// Pre-flight rejection — transfer endpoint never called, fee not
			// charged. Zero the fee via FeeWaived and terminate as failed.
			logger.Info("wallet_bulk_row: rejected at pre-flight (fee waived)", "error", err)
			return w.markRowFailed(ctx, p.BatchID, p.Row.VFICCode, err, true /*feeWaived*/)
		}
		return fmt.Errorf("initiate_transfer: %w", err)
	}

	// === Step 5: Record sync response — EXACT canonical field mapping ===
	// Accepted MUST be set explicitly (red-team v2 R2-1).
	accepted := result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess
	if _, err := w.walletPaymentSvc.RecordSyncResponse(ctx, p.Row.VFICCode, disbursement.SyncResult{
		Accepted:     accepted,
		InvoiceNo:    result.ProviderRef, // canonical mapping
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	}); err != nil {
		return fmt.Errorf("record_sync_response: %w", err)
	}

	logger.Info("wallet_bulk_row: transfer initiated",
		"provider_ref", result.ProviderRef, "status", string(result.Status))
	if !accepted {
		// The transfer endpoint was called, so the provider fee remains. A
		// synchronous rejection is already terminal and may not receive an IPN;
		// count it now so a mixed batch cannot remain processing forever.
		return w.markRowTerminal(ctx, p.BatchID)
	}

	// Step 6: terminal state reached later via IPN webhook or status-inquiry poller.
	return nil
}

// ensureBulkBatchLink stamps bulk_transfer_batch_id + order + vfic_code onto
// the row. Idempotent — safe to call on retry.
//
// entity_id is intentionally NOT stamped here. notifyEmployee (Validation
// Decision V6) handles entity_id == nil rows by resolving the employee via
// recipient_account_no. Bulk transfers have no advance_payment_request, so
// there's nothing for entity_id to point at in the existing schema.
func (w *WalletBulkTransferRowWorker) ensureBulkBatchLink(ctx context.Context, rowID uint64, batchID uint64, row wallet_bulk.BulkTransferRow) {
	order := uint(row.OrderNo)
	if err := w.paymentRepo.UpdateBulkBatchLink(ctx, rowID, batchID, order, row.VFICCode); err != nil {
		w.logger.Warn("wallet_bulk_row: ensureBulkBatchLink failed (non-fatal)",
			"row_id", rowID, "batch_id", batchID, "error", err)
	}
}

// skipRow handles the case where a row is skipped BEFORE any wallet_payment
// is inserted (insufficient balance, ErrDuplicatePaymentInProgress). It
// atomically decrements the batch's total_count so the success+failed==total
// threshold still fires, then runs the completion detector.
//
// Without this, skipped rows would be invisible to markRowTerminal and the
// batch would strand in `processing` forever (C1 fix).
func (w *WalletBulkTransferRowWorker) skipRow(ctx context.Context, batchID uint64) error {
	if err := w.batchRepo.DecrementTotalCount(ctx, batchID, 1); err != nil {
		w.logger.Error("wallet_bulk_row: decrement total_count failed", "batch_id", batchID, "error", err)
		return fmt.Errorf("skipRow: decrement total_count: %w", err)
	}
	return w.markRowTerminal(ctx, batchID)
}

// markRowFailed routes a row through the FSM via RecordSyncResponse, NOT via
// direct UpdateStatus (red-team v2 Security C5). feeWaived=true zeroes the
// stamped fee when the failure was a pre-flight rejection.
func (w *WalletBulkTransferRowWorker) markRowFailed(ctx context.Context, batchID uint64, requestID string, cause error, feeWaived bool) error {
	if _, err := w.walletPaymentSvc.RecordSyncResponse(ctx, requestID, disbursement.SyncResult{
		Accepted:     false, // TriggerReject
		RawErrorCode: extractErrorCode(cause),
		RawMessage:   cause.Error(),
		FeeWaived:    feeWaived,
	}); err != nil {
		return fmt.Errorf("markRowFailed: record_sync_response: %w", err)
	}
	return w.markRowTerminal(ctx, batchID)
}

// extractErrorCode pulls a short error tag from the cause. For
// ErrPreflightValidation we return the canonical "preflight_validation" tag
// so KQ column I and admin search can find these rows. For all other errors
// we return a fixed "transfer_failed" tag — raw error strings can leak DB
// driver internals / network details into the KQ Excel column I; the full
// diagnostic stays in error_message (M3 fix).
func extractErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, infrastructure.ErrPreflightValidation) {
		return "preflight_validation"
	}
	return "transfer_failed"
}

// markRowTerminal is the atomic batch-completion detector. Uses
// batchRepo.UpdateWithLock to:
//  1. Acquire SELECT FOR UPDATE on the batch row.
//  2. Recount success_count + failed_count from wallet_payments.
//  3. If success+failed == total, set fee_booked_at + status='completing'
//     and signal shouldBook=true.
//  4. Commit (releasing the lock).
//
// The ledger write happens in the book_batch_ledger asynq task AFTER the
// lock releases so we never hold the row lock across CreateTransaction.
func (w *WalletBulkTransferRowWorker) markRowTerminal(ctx context.Context, batchID uint64) error {
	return finalizeBulkBatch(ctx, batchID, w.batchRepo, w.paymentRepo, w.asynqClient, w.logger)
}

// FinalizeBulkBatchForIPN is the package-level entry point the IPN worker
// calls when a bulk-row wallet_payment reaches its terminal state via IPN
// (the async OnePay path — sync Initiate returns "initiated" without a
// terminal status, so markRowTerminal never runs from the row worker).
//
// Same atomic flow as markRowTerminal (lock → recount → flip to completing
// → enqueue book_batch_ledger). Idempotent — UpdateWithLock no-ops when
// the batch is already terminal or fee_booked_at is set, so multiple
// terminal rows in the same batch racing through IPN won't double-book.
func FinalizeBulkBatchForIPN(
	ctx context.Context,
	batchID uint64,
	batchRepo domain.BulkTransferBatchRepository,
	paymentRepo domaintx.WalletPaymentRepository,
	asynqClient wallet_bulk.BulkTransferEnqueuer,
	logger *slog.Logger,
) error {
	return finalizeBulkBatch(ctx, batchID, batchRepo, paymentRepo, asynqClient, logger)
}

// finalizeBulkBatch is the shared body of markRowTerminal + the IPN path.
// Kept package-local so the row worker's markRowTerminal stays a thin
// receiver wrapper while the IPN worker can call it without needing the
// full WalletBulkTransferRowWorker wiring (it has no Initiate/registry deps).
func finalizeBulkBatch(
	ctx context.Context,
	batchID uint64,
	batchRepo domain.BulkTransferBatchRepository,
	paymentRepo domaintx.WalletPaymentRepository,
	asynqClient wallet_bulk.BulkTransferEnqueuer,
	logger *slog.Logger,
) error {
	batch, shouldBook, err := batchRepo.UpdateWithLock(ctx, batchID, func(b *domain.BulkTransferBatch) (bool, error) {
		// Already finalized by an earlier worker — no-op.
		if b.Status == domain.BulkTransferBatchStatusCompleted ||
			b.Status == domain.BulkTransferBatchStatusFailed ||
			b.FeeBookedAt != nil {
			return false, nil
		}

		// Recount terminal rows for this batch.
		success, err := paymentRepo.CountByBatchAndStatuses(ctx, batchID, []domaintx.State{domaintx.StateCompleted})
		if err != nil {
			return false, fmt.Errorf("count success: %w", err)
		}
		failed, err := paymentRepo.CountByBatchAndStatuses(ctx, batchID, []domaintx.State{domaintx.StateFailed, domaintx.StateReversed})
		if err != nil {
			return false, fmt.Errorf("count failed: %w", err)
		}
		b.SuccessCount = int(success)
		b.FailedCount = int(failed)

		if int(success)+int(failed) < b.TotalCount {
			// Not done yet — keep status as-is (processing).
			return false, nil
		}

		// All rows terminal. Flip to 'completing' + stamp fee_booked_at so a
		// crash between here and the book_batch_ledger task is recoverable.
		now := clock.Now()
		b.FeeBookedAt = &now
		b.Status = domain.BulkTransferBatchStatusCompleting
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("finalize_bulk_batch: update_with_lock: %w", err)
	}
	if !shouldBook {
		return nil
	}

	// Enqueue the book_batch_ledger task. It runs OUTSIDE the lock.
	if err := asynqClient.EnqueueBookBatchLedger(wallet_bulk.BookLedgerPayload{BatchID: batch.ID}); err != nil {
		return fmt.Errorf("finalize_bulk_batch: enqueue book_ledger: %w", err)
	}
	return nil
}

// wallClock is a package-level indirection so tests can stub the clock.
// Default: clock.Now() (Asia/Ho_Chi_Minh).
var wallClock = clock.Now

// _time is imported for the test file's time-travel helper.
var _ = time.Time{}
