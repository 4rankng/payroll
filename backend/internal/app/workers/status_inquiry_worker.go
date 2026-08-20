package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/pkg/clock"

	asynqlib "github.com/hibiken/asynq"
	"golang.org/x/sync/errgroup"
)

const (
	// TaskStatusInquiry is the asynq task type for the periodic status inquiry poller.
	TaskStatusInquiry = "disbursement:status_inquiry"

	statusInquiryMinAge      = 5 * time.Minute // wait before polling (give IPN time to arrive)
	statusInquiryBatchSize   = 10
	statusInquiryConcurrency = 3 // defensive cap — inquiry endpoint has no documented rate limit
)

// StatusInquiryPollerWorker is a periodic task that polls the active provider's
// inquiry endpoint for wallet payments stuck in "authorised" state. It acts as
// a safety net when IPN webhooks are lost or delayed.
//
// The worker reuses WalletPaymentService.RecordIPN to drive the same FSM
// transitions as real IPN callbacks — idempotent by design.
type StatusInquiryPollerWorker struct {
	walletPaymentRepo    statusInquiryPaymentRepository
	finalizationRepo     bulkFinalizationCandidateRepository
	walletPaymentService statusInquiryPaymentService
	registry             *disbursement.Registry
	timesheetUpdater     TransferTimesheetUpdater
	batchFinalizer       BulkBatchFinalizer
	bankRepo             bankCodeResolver
	disbursementTasks    disbursementTaskEnqueuer
	logger               *slog.Logger
}

// WithBulkBatchFinalizer wires the same idempotent bulk-batch finalizer used
// by the IPN worker. Status inquiry is a substitute resolution source when an
// IPN is lost, so it must complete the same downstream batch workflow.
func (w *StatusInquiryPollerWorker) WithBulkBatchFinalizer(f BulkBatchFinalizer) *StatusInquiryPollerWorker {
	w.batchFinalizer = f
	return w
}

// WithBankRepository enables safe automatic re-send after OnePay explicitly
// confirms a verified transfer ID does not exist.
func (w *StatusInquiryPollerWorker) WithBankRepository(repo bankCodeResolver) *StatusInquiryPollerWorker {
	w.bankRepo = repo
	return w
}

// WithDisbursementTaskEnqueuer enables recovery of a persisted pre-transfer
// payment when its original queue task was lost. The recovery task uses the
// original provider idempotency key and recipient snapshot.
func (w *StatusInquiryPollerWorker) WithDisbursementTaskEnqueuer(enqueuer disbursementTaskEnqueuer) *StatusInquiryPollerWorker {
	w.disbursementTasks = enqueuer
	return w
}

type statusInquiryPaymentRepository interface {
	ListStaleAuthorised(ctx context.Context, provider string, cutoff time.Time, limit int) ([]*domaintx.WalletPayment, error)
	ListStaleAdvancePending(ctx context.Context, provider string, cutoff time.Time, limit int) ([]*domaintx.WalletPayment, error)
}

type disbursementTaskEnqueuer interface {
	Enqueue(task *asynqlib.Task, opts ...asynqlib.Option) (*asynqlib.TaskInfo, error)
}

type bulkFinalizationCandidateRepository interface {
	ListStaleBulkFinalizationCandidates(ctx context.Context, provider string, cutoff time.Time, limit int) ([]*domaintx.WalletPayment, error)
}

type statusInquiryPaymentService interface {
	syncResponseRecorder
	RecordIPN(ctx context.Context, result disbursement.IPNResult) (*domaintx.WalletPayment, error)
}

func NewStatusInquiryPollerWorker(
	walletPaymentRepo statusInquiryPaymentRepository,
	walletPaymentService statusInquiryPaymentService,
	registry *disbursement.Registry,
	timesheetUpdater TransferTimesheetUpdater,
	logger *slog.Logger,
) *StatusInquiryPollerWorker {
	worker := &StatusInquiryPollerWorker{
		walletPaymentRepo:    walletPaymentRepo,
		walletPaymentService: walletPaymentService,
		registry:             registry,
		timesheetUpdater:     timesheetUpdater,
		logger:               logger,
	}
	worker.finalizationRepo, _ = walletPaymentRepo.(bulkFinalizationCandidateRepository)
	return worker
}

// ProcessJob queries stale authorised payments and polls each via the
// provider's inquiry endpoint. Terminal results are applied through
// RecordIPN (same FSM path as webhooks).
func (w *StatusInquiryPollerWorker) ProcessJob(ctx context.Context) error {
	// Step 1: Resolve active provider
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return nil // no provider configured — no-op
	}

	// Step 2: Check if provider supports status polling (OnePay: yes, 9Pay: no)
	poller, ok := provider.(infrastructure.StatusPoller)
	if !ok {
		return nil // provider has no inquiry endpoint — no-op
	}

	providerName := provider.Name()
	cutoff := clock.Now().Add(-statusInquiryMinAge)

	// A pending row has not reached the transfer call yet, so provider status
	// inquiry cannot resolve it. Re-enqueue it from durable DB state with its
	// original idempotency key; the execute worker resumes the existing row.
	pendingPayments, err := w.walletPaymentRepo.ListStaleAdvancePending(ctx, providerName, cutoff, statusInquiryBatchSize)
	if err != nil {
		return fmt.Errorf("status inquiry: query stale advance pending: %w", err)
	}
	for _, payment := range pendingPayments {
		if err := w.enqueuePendingRecovery(payment); err != nil {
			w.logger.Warn("status inquiry: re-enqueue stale pending payment failed",
				"payment_id", payment.ID, "request_id", payment.RequestID, "error", err)
		}
	}

	// Step 3: Query stale authorised payments
	payments, err := w.walletPaymentRepo.ListStaleAuthorised(ctx, providerName, cutoff, statusInquiryBatchSize)
	if err != nil {
		return fmt.Errorf("status inquiry: query stale authorised: %w", err)
	}
	if len(payments) > 0 {
		w.logger.Info("status inquiry: polling stale authorised payments",
			"provider", providerName,
			"count", len(payments),
			"cutoff", cutoff.Format("2006-01-02 15:04:05"))
	}

	// Step 4: Poll payments concurrently with bounded parallelism
	g := new(errgroup.Group)
	g.SetLimit(statusInquiryConcurrency)

	var resolved, skipped, errorCount atomic.Int64

	for _, p := range payments {
		g.Go(func() error {
			w.processPayment(ctx, provider, poller, providerName, p, &resolved, &skipped, &errorCount)
			return nil // never propagate error — one failure must not cancel the batch
		})
	}

	_ = g.Wait() // always nil since closures return nil

	// A payment transition and its bulk-batch finalization are separate DB
	// operations. If finalization failed after the payment committed terminal,
	// retry it from a durable query source on subsequent poller runs.
	if w.batchFinalizer != nil && w.finalizationRepo != nil {
		candidates, listErr := w.finalizationRepo.ListStaleBulkFinalizationCandidates(ctx, providerName, cutoff, statusInquiryBatchSize)
		if listErr != nil {
			return fmt.Errorf("status inquiry: list bulk finalization candidates: %w", listErr)
		}
		for _, candidate := range candidates {
			if finalizeErr := w.batchFinalizer.FinalizeBulkBatchForIPN(ctx, candidate); finalizeErr != nil {
				w.logger.Warn("status inquiry: retry bulk batch finalize failed",
					"request_id", candidate.RequestID,
					"error", finalizeErr)
				errorCount.Add(1)
			}
		}
	}

	r := resolved.Load()
	s := skipped.Load()
	e := errorCount.Load()
	w.logger.Info("status inquiry: batch complete",
		"polled", len(payments),
		"pending_recovery", len(pendingPayments),
		"resolved", r,
		"skipped", s,
		"errors", e)

	return nil
}

func (w *StatusInquiryPollerWorker) enqueuePendingRecovery(payment *domaintx.WalletPayment) error {
	if w.disbursementTasks == nil {
		return errors.New("disbursement task enqueuer unavailable")
	}
	if payment == nil || payment.EntityID == nil {
		return errors.New("stale pending payment is not linked to an advance request")
	}

	payload, err := json.Marshal(DisbursementExecutePayload{
		RequestID:          payment.RequestID,
		AdvanceRequestID:   *payment.EntityID,
		RequestedAmount:    payment.RequestedAmount,
		RecipientName:      payment.RecipientName,
		RecipientAccountNo: payment.RecipientAccountNo,
		RecipientBank:      payment.RecipientBank,
	})
	if err != nil {
		return fmt.Errorf("marshal pending recovery payload: %w", err)
	}

	task := asynqlib.NewTask(TaskDisbursementExecute, payload)
	if _, err := w.disbursementTasks.Enqueue(task,
		asynqlib.MaxRetry(5),
		asynqlib.TaskID(fmt.Sprintf("disbursement:recover:%d", payment.ID)),
	); err != nil {
		return fmt.Errorf("enqueue pending recovery: %w", err)
	}

	w.logger.Info("status inquiry: re-enqueued stale pending advance payment",
		"payment_id", payment.ID, "request_id", payment.RequestID, "advance_request_id", *payment.EntityID)
	return nil
}

// processPayment polls a single payment via the provider's inquiry endpoint
// and applies the result through RecordIPN. Counters are updated via atomic
// operations for safe concurrent use.
func (w *StatusInquiryPollerWorker) processPayment(
	ctx context.Context,
	provider infrastructure.DisbursementProvider,
	poller infrastructure.StatusPoller,
	providerName string,
	p *domaintx.WalletPayment,
	resolved, skipped, errorCount *atomic.Int64,
) {
	var result *infrastructure.TransferResult
	var err error
	if p.Status == domaintx.StateVerified {
		var notFound bool
		result, notFound, err = recoverVerifiedTransfer(ctx, provider, w.walletPaymentService, p)
		if notFound {
			var retriedRow *domaintx.WalletPayment
			result, retriedRow, err = retryVerifiedTransfer(ctx, provider, w.walletPaymentService, w.bankRepo, p)
			if err == nil && retriedRow != nil && retriedRow.IsTerminal() {
				w.applyTerminalSideEffects(ctx, retriedRow, errorCount)
				resolved.Add(1)
				return
			}
		}
	} else {
		result, err = poller.CheckStatus(ctx, p.RequestID)
	}
	if err != nil {
		w.logger.Warn("status inquiry: check failed",
			"request_id", p.RequestID,
			"payment_id", p.ID,
			"error", err)
		errorCount.Add(1)
		return
	}

	// Skip non-terminal states — IPN may still arrive
	if result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusUnknown {
		skipped.Add(1)
		return
	}

	// Apply terminal result through the same FSM path as IPN webhooks
	var invoiceNo string
	if p.InvoiceNo != nil {
		invoiceNo = *p.InvoiceNo
	}
	row, ipnErr := w.walletPaymentService.RecordIPN(ctx, disbursement.IPNResult{
		Provider:     providerName,
		Status:       result.Status,
		InvoiceNo:    invoiceNo,
		RequestID:    p.RequestID,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
		Source:       domaintx.ResolutionSourceStatusInquiry,
	})
	if ipnErr != nil {
		if _, ok := errors.AsType[domaintx.TerminalStateError](ipnErr); ok {
			// Already resolved by IPN — expected race, not an error
			resolved.Add(1)
			return
		}
		w.logger.Warn("status inquiry: record IPN failed",
			"request_id", p.RequestID,
			"payment_id", p.ID,
			"error", ipnErr)
		errorCount.Add(1)
		return
	}

	w.applyTerminalSideEffects(ctx, row, errorCount)

	w.logger.Info("status inquiry: resolved",
		"request_id", p.RequestID,
		"payment_id", p.ID,
		"status", string(result.Status),
		"provider_ref", result.ProviderRef)
	resolved.Add(1)
}

func (w *StatusInquiryPollerWorker) applyTerminalSideEffects(ctx context.Context, row *domaintx.WalletPayment, errorCount *atomic.Int64) {
	if row == nil || !row.IsTerminal() {
		return
	}
	if w.timesheetUpdater != nil {
		completed := row.Status == domaintx.StateCompleted
		if updateErr := w.timesheetUpdater.UpdateForTransfer(ctx, row.RequestID, completed, row.GetInvoiceNo()); updateErr != nil {
			w.logger.Warn("status inquiry: per-transfer timesheet update failed (non-fatal)",
				"request_id", row.RequestID,
				"error", updateErr)
		}
	}
	if row.BulkTransferBatchID != nil && w.batchFinalizer != nil {
		if finalizeErr := w.batchFinalizer.FinalizeBulkBatchForIPN(ctx, row); finalizeErr != nil {
			w.logger.Warn("status inquiry: bulk batch finalize failed; durable poller retry will re-attempt",
				"request_id", row.RequestID,
				"batch_id", *row.BulkTransferBatchID,
				"error", finalizeErr)
			errorCount.Add(1)
		}
	}
}
