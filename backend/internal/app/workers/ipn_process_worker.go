package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/disbursement"
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

// IPNProcessWorker drives the FSM transition for a verified disbursement
// IPN event off the HTTP path.
type IPNProcessWorker struct {
	providerTxs      *disbursement.WalletPaymentService
	walletIPNRepo    wallet.WalletIPNRepository
	timesheetUpdater TransferTimesheetUpdater
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

		return nil
	case errors.Is(err, domaintx.ErrNotFound):
		w.logger.Warn("disbursement IPN references unknown invoice — dropping",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"request_id", p.RequestID,
			"status", p.Status)
		w.markIPN(ctx, p.IPNRecordID, "ignored", nil, "wallet_payment not found")
		return nil
	}

	var terminalErr domaintx.TerminalStateError
	if errors.As(err, &terminalErr) {
		w.logger.Info("disbursement IPN ignored — row already terminal (idempotency)",
			"provider", p.Provider,
			"invoice_no", p.InvoiceNo,
			"current_state", terminalErr.State,
			"trigger", terminalErr.Trigger)
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
