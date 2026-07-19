package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/wallet_bulk"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
)

// IPNJob is the worker's input — a verified, parsed disbursement IPN
// event ready for FSM application.
type IPNJob struct {
	Provider      string
	InvoiceNo     string
	RequestID     string
	Status        string
	Amount        int64
	RawErrorCode  string
	FailureReason string
	IPNRecordID   uint64
}

// TransferTimesheetUpdater is called after a terminal IPN to immediately
// update the related timesheet payment statuses for that single transfer.
type TransferTimesheetUpdater interface {
	UpdateForTransfer(ctx context.Context, requestID string, completed bool, invoiceNo string) error
}

// BulkBatchFinalizer closes out a wallet bulk-transfer batch when one of
// its rows reaches a terminal state via IPN. Optional — nil means the
// worker is wired for non-bulk flows (manual disbursement etc.) and the
// batch finalization is skipped.
type BulkBatchFinalizer interface {
	FinalizeBulkBatchForIPN(ctx context.Context, payment *domaintx.WalletPayment) error
}

// BulkPaymentReversalHandler adjusts an already-booked batch when OnePay
// changes one employee payment from completed to reversed.
type BulkPaymentReversalHandler interface {
	HandleReversedPayment(ctx context.Context, payment *domaintx.WalletPayment) error
}

// IPNProcessWorker drives the FSM transition for a verified disbursement
// IPN event off the HTTP path.
type IPNProcessWorker struct {
	providerTxs      *disbursement.WalletPaymentService
	walletIPNRepo    wallet.WalletIPNRepository
	timesheetUpdater TransferTimesheetUpdater
	batchFinalizer   BulkBatchFinalizer
	logger           *slog.Logger
}

func NewIPNProcessWorker(
	providerTxs *disbursement.WalletPaymentService,
	walletIPNRepo wallet.WalletIPNRepository,
	timesheetUpdater TransferTimesheetUpdater,
	logger *slog.Logger,
) *IPNProcessWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &IPNProcessWorker{
		providerTxs:      providerTxs,
		walletIPNRepo:    walletIPNRepo,
		timesheetUpdater: timesheetUpdater,
		logger:           logger.With("component", "IPNProcessWorker"),
	}
}

// WithBulkBatchFinalizer wires the optional bulk-batch finalizer. When
// set, terminal IPN rows that belong to a bulk batch trigger
// FinalizeBulkBatchForIPN so the batch can flip to 'completing' → 'completed'
// without waiting on the 5-minute completing-recovery cron. Returns the
// worker for chaining at bootstrap.
func (w *IPNProcessWorker) WithBulkBatchFinalizer(f BulkBatchFinalizer) *IPNProcessWorker {
	w.batchFinalizer = f
	return w
}

// bulkBatchFinalizerAdapter is the concrete BulkBatchFinalizer — a thin
// closure over the same repos the row worker uses. Declared here so
// bootstrap can compose it without exposing finalizeBulkBatch publicly.
type bulkBatchFinalizerAdapter struct {
	batchRepo   domain.BulkTransferBatchRepository
	paymentRepo domaintx.WalletPaymentRepository
	asynqClient wallet_bulk.BulkTransferEnqueuer
	reversals   BulkPaymentReversalHandler
	logger      *slog.Logger
}

// NewBulkBatchFinalizer constructs a BulkBatchFinalizer suitable for
// injection into NewIPNProcessWorker().WithBulkBatchFinalizer(...).
func NewBulkBatchFinalizer(
	batchRepo domain.BulkTransferBatchRepository,
	paymentRepo domaintx.WalletPaymentRepository,
	asynqClient wallet_bulk.BulkTransferEnqueuer,
	reversals BulkPaymentReversalHandler,
	logger *slog.Logger,
) BulkBatchFinalizer {
	if logger == nil {
		logger = slog.Default()
	}
	return &bulkBatchFinalizerAdapter{
		batchRepo:   batchRepo,
		paymentRepo: paymentRepo,
		asynqClient: asynqClient,
		reversals:   reversals,
		logger:      logger,
	}
}

func (a *bulkBatchFinalizerAdapter) FinalizeBulkBatchForIPN(ctx context.Context, payment *domaintx.WalletPayment) error {
	if payment == nil || payment.BulkTransferBatchID == nil {
		return nil
	}
	if payment.Status == domaintx.StateReversed && a.reversals != nil {
		batch, err := a.batchRepo.GetByID(ctx, *payment.BulkTransferBatchID)
		if err != nil {
			return err
		}
		if batch.Status == domain.BulkTransferBatchStatusCompleted {
			return a.reversals.HandleReversedPayment(ctx, payment)
		}
	}
	return FinalizeBulkBatchForIPN(ctx, *payment.BulkTransferBatchID, a.batchRepo, a.paymentRepo, a.asynqClient, a.logger)
}

func (w *IPNProcessWorker) ProcessJob(ctx context.Context, p IPNJob) (err error) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic in disbursement IPN processing",
				"provider", p.Provider,
				"invoice_no", p.InvoiceNo,
				"request_id", p.RequestID,
				"panic", r)
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	w.logger.Info("processing disbursement IPN",
		"provider", p.Provider,
		"invoice_no", p.InvoiceNo,
		"request_id", p.RequestID,
		"status", p.Status,
		"amount", p.Amount,
	)

	row, err := w.providerTxs.RecordIPN(ctx, disbursement.IPNResult{
		Provider:      p.Provider,
		Status:        infrastructure.TransferStatus(p.Status),
		InvoiceNo:     p.InvoiceNo,
		RequestID:     p.RequestID,
		Amount:        p.Amount,
		RawErrorCode:  p.RawErrorCode,
		FailureReason: p.FailureReason,
		Source:        domaintx.ResolutionSourceIPN,
	})

	switch {
	case err == nil:
		w.logger.Info("disbursement IPN applied",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"request_id", p.RequestID,
			"status", p.Status)
		w.markIPN(ctx, p.IPNRecordID, "applied", row)

		// Immediately update timesheets for this single transfer.
		if row != nil && row.IsTerminal() && w.timesheetUpdater != nil {
			completed := row.Status == domaintx.StateCompleted
			if updateErr := w.timesheetUpdater.UpdateForTransfer(ctx, row.RequestID, completed, row.GetInvoiceNo()); updateErr != nil {
				w.logger.Warn("per-transfer timesheet update failed (non-fatal)",
					"request_id", row.RequestID,
					"error", updateErr)
			}
		}

		// Bulk-batch completion (async OnePay flow): the row worker returns
		// after Initiate without a terminal status, so markRowTerminal never
		// fires from that side. IPN is the only path that learns the
		// terminal outcome in the async flow — drive the batch finalizer
		// here so the batch flips to completed in seconds, not via the
		// 5-minute completing-recovery cron. Non-fatal: finalization is
		// idempotent and the cron still catches it on failure.
		if row != nil && row.IsTerminal() && row.BulkTransferBatchID != nil && w.batchFinalizer != nil {
			if finErr := w.batchFinalizer.FinalizeBulkBatchForIPN(ctx, row); finErr != nil {
				// A late reversal of an already-booked batch has no periodic
				// finalizer to repair its ledger/timesheet adjustment. Return the
				// error so Asynq retries the idempotent reconciliation.
				if row.Status == domaintx.StateReversed {
					w.markIPN(ctx, p.IPNRecordID, "error", row, finErr.Error())
					return fmt.Errorf("reconcile reversed bulk payment: %w", finErr)
				}
				w.logger.Warn("bulk batch finalize via IPN failed (non-fatal — completing-recovery cron will retry)",
					"batch_id", *row.BulkTransferBatchID,
					"request_id", row.RequestID,
					"error", finErr)
			}
		}

		return nil
	case errors.Is(err, domaintx.ErrNotFound):
		w.logger.Warn("disbursement IPN references unknown invoice — dropping",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"request_id", p.RequestID,
			"status", p.Status)
		w.markIPN(ctx, p.IPNRecordID, "ignored", nil, "wallet_payment not found")
		return nil
	case errors.Is(err, disbursement.ErrIPNAmountMismatch):
		w.logger.Error("disbursement IPN rejected — amount mismatch (forged/tampered), dropping",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"request_id", p.RequestID,
			"reported_amount", p.Amount)
		w.markIPN(ctx, p.IPNRecordID, "rejected", nil, "amount mismatch")
		return nil
	}

	var terminalErr domaintx.TerminalStateError
	if errors.As(err, &terminalErr) {
		w.logger.Info("disbursement IPN ignored — row already terminal (idempotency)",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"current_state", terminalErr.State,
			"trigger", terminalErr.Trigger)
		// A completed→reversed transition may have committed before a later
		// reconciliation step failed. Retried IPNs then see an already-terminal
		// row, so explicitly retry the idempotent batch adjustment here.
		if current, getErr := w.providerTxs.GetByRequestID(ctx, p.RequestID); getErr == nil &&
			current.Status == domaintx.StateReversed && current.BulkTransferBatchID != nil && w.batchFinalizer != nil {
			if finErr := w.batchFinalizer.FinalizeBulkBatchForIPN(ctx, current); finErr != nil {
				w.markIPN(ctx, p.IPNRecordID, "error", current, finErr.Error())
				return fmt.Errorf("retry reversed bulk reconciliation: %w", finErr)
			}
		}
		w.markIPN(ctx, p.IPNRecordID, "ignored", nil, "already terminal")
		return nil
	}

	w.logger.Warn("disbursement IPN apply failed — asynq will retry",
		"provider", p.Provider,
		"invoice_no", p.InvoiceNo,
		"request_id", p.RequestID,
		"error", err)
	w.markIPN(ctx, p.IPNRecordID, "error", nil, err.Error())
	return err
}

func (w *IPNProcessWorker) markIPN(ctx context.Context, ipnID uint64, status string, row *domaintx.WalletPayment, errMsg ...string) {
	if ipnID == 0 || w.walletIPNRepo == nil {
		return
	}
	var walletPaymentID *uint64
	if row != nil {
		walletPaymentID = &row.ID
	}
	var processingError string
	if len(errMsg) > 0 {
		processingError = errMsg[0]
	}
	if err := w.walletIPNRepo.UpdateProcessingResult(ctx, ipnID, status, walletPaymentID, processingError); err != nil {
		w.logger.Warn("failed to update IPN processing result", "ipn_id", ipnID, "error", err)
	}
}
