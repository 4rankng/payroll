package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/pkg/clock"

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
	walletPaymentRepo    domaintx.WalletPaymentRepository
	walletPaymentService *disbursement.WalletPaymentService
	registry             *disbursement.Registry
	timesheetUpdater     TransferTimesheetUpdater
	logger               *slog.Logger
}

func NewStatusInquiryPollerWorker(
	walletPaymentRepo domaintx.WalletPaymentRepository,
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	timesheetUpdater TransferTimesheetUpdater,
	logger *slog.Logger,
) *StatusInquiryPollerWorker {
	return &StatusInquiryPollerWorker{
		walletPaymentRepo:    walletPaymentRepo,
		walletPaymentService: walletPaymentService,
		registry:             registry,
		timesheetUpdater:     timesheetUpdater,
		logger:               logger,
	}
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

	// Step 3: Query stale authorised payments
	payments, err := w.walletPaymentRepo.ListStaleAuthorised(ctx, providerName, cutoff, statusInquiryBatchSize)
	if err != nil {
		return fmt.Errorf("status inquiry: query stale authorised: %w", err)
	}
	if len(payments) == 0 {
		return nil
	}

	w.logger.Info("status inquiry: polling stale authorised payments",
		"provider", providerName,
		"count", len(payments),
		"cutoff", cutoff.Format("2006-01-02 15:04:05"))

	// Step 4: Poll payments concurrently with bounded parallelism
	g := new(errgroup.Group)
	g.SetLimit(statusInquiryConcurrency)

	var resolved, skipped, errorCount atomic.Int64

	for _, p := range payments {
		g.Go(func() error {
			w.processPayment(ctx, poller, providerName, p, &resolved, &skipped, &errorCount)
			return nil // never propagate error — one failure must not cancel the batch
		})
	}

	_ = g.Wait() // always nil since closures return nil

	r := resolved.Load()
	s := skipped.Load()
	e := errorCount.Load()
	w.logger.Info("status inquiry: batch complete",
		"polled", len(payments),
		"resolved", r,
		"skipped", s,
		"errors", e)

	return nil
}

// processPayment polls a single payment via the provider's inquiry endpoint
// and applies the result through RecordIPN. Counters are updated via atomic
// operations for safe concurrent use.
func (w *StatusInquiryPollerWorker) processPayment(
	ctx context.Context,
	poller infrastructure.StatusPoller,
	providerName string,
	p *domaintx.WalletPayment,
	resolved, skipped, errorCount *atomic.Int64,
) {
	result, err := poller.CheckStatus(ctx, p.RequestID)
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

	// Update timesheet payment status (mirrors IPNProcessWorker post-RecordIPN logic)
	if w.timesheetUpdater != nil && row != nil && row.IsTerminal() {
		completed := row.Status == domaintx.StateCompleted
		if updateErr := w.timesheetUpdater.UpdateForTransfer(ctx, row.RequestID, completed, row.GetInvoiceNo()); updateErr != nil {
			w.logger.Warn("status inquiry: per-transfer timesheet update failed (non-fatal)",
				"request_id", row.RequestID,
				"error", updateErr)
		}
	}

	w.logger.Info("status inquiry: resolved",
		"request_id", p.RequestID,
		"payment_id", p.ID,
		"status", string(result.Status),
		"provider_ref", result.ProviderRef)
	resolved.Add(1)
}
