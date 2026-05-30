package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

// NinePayBulkExecuteWorker processes a 9Pay bulk transfer batch by calling
// the 9Pay disbursement API for each row, rate-limited by the provider queue.
type NinePayBulkExecuteWorker struct {
	walletPaymentService *disbursement.WalletPaymentService
	registry             *disbursement.Registry
	fileRepo             domain.BulkTransferFileRepository
	logger               *slog.Logger
}

func NewNinePayBulkExecuteWorker(
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	fileRepo domain.BulkTransferFileRepository,
	logger *slog.Logger,
) *NinePayBulkExecuteWorker {
	return &NinePayBulkExecuteWorker{
		walletPaymentService: walletPaymentService,
		registry:             registry,
		fileRepo:             fileRepo,
		logger:               logger.With("component", "NinePayBulkExecuteWorker"),
	}
}

// ProcessJob handles a bulk_transfer:ninepay_execute task.
func (w *NinePayBulkExecuteWorker) ProcessJob(ctx context.Context, t *asynqlib.Task) error {
	var p bulktransfer.NinePayExecutePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		w.logger.Error("ninepay execute: malformed payload", "error", err)
		return fmt.Errorf("unmarshal payload: %w", asynqlib.SkipRetry)
	}

	logger := w.logger.With("batch_file_id", p.BatchFileID, "batch_id", p.BatchID)
	logger.Info("Processing 9Pay bulk transfer batch")

	// Load the batch file
	btf, err := w.fileRepo.GetByID(ctx, p.BatchFileID)
	if err != nil {
		return fmt.Errorf("batch file not found: %w", err)
	}

	// Parse file data
	fileData, err := dto.ParseBulkTransferFileData(btf.Data)
	if err != nil {
		return fmt.Errorf("parse file data: %w", err)
	}

	// Get active provider
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return fmt.Errorf("no active provider: %w", err)
	}

	var completedCount, failedCount int
	dataChanged := false

	for i := range fileData {
		row := &fileData[i]

		// Skip already processed rows (idempotent retry)
		if row.TransferStatus == bulktransfer.ResultStatusCompleted || row.TransferStatus == bulktransfer.ResultStatusFailed {
			if row.TransferStatus == bulktransfer.ResultStatusCompleted {
				completedCount++
			} else {
				failedCount++
			}
			continue
		}

		rowLogger := logger.With(
			"transaction_code", row.TransactionCode,
			"employee_id", row.EmployeeID,
			"amount", row.Amount,
		)

		// Validate bank details
		if row.AccountNumber == "" || row.BankCode == "" {
			rowLogger.Warn("Skipping row: missing bank details")
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = "missing bank details"
			failedCount++
			dataChanged = true
			continue
		}

		// Step 1: Initiate wallet payment record (pending state)
		walletPayment, err := w.walletPaymentService.Initiate(ctx, disbursement.InitiateInput{
			RequestID:          row.TransactionCode,
			RequestedAmount:    row.Amount,
			RecipientName:      row.AccountName,
			RecipientAccountNo: row.AccountNumber,
			RecipientBank:      row.BankCode,
			BatchID:            &p.BatchID,
		})
		if err != nil {
			rowLogger.Error("Failed to initiate wallet payment", "error", err)
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = err.Error()
			failedCount++
			dataChanged = true
			continue
		}

		// If already past pending, skip to counting
		if walletPayment.Status != domaintx.StatePending {
			rowLogger.Info("Wallet payment already processed", "status", walletPayment.Status)
			if walletPayment.IsTerminal() {
				if walletPayment.Status == domaintx.StateCompleted {
					completedCount++
				} else {
					failedCount++
				}
			}
			continue
		}

		// Step 2: Verify account (pending → verified) — auto-verify since we
		// already validated bank details above.
		_, err = w.walletPaymentService.RecordAccountCheck(ctx, row.TransactionCode, disbursement.AccountCheckOutcome{
			Verified: true,
		})
		if err != nil {
			rowLogger.Error("Failed to record account check", "error", err)
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = err.Error()
			failedCount++
			dataChanged = true
			continue
		}

		// Step 3: Initiate transfer via provider (rate-limited by queue)
		result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
			RequestID:   row.TransactionCode,
			Amount:      row.Amount,
			BankCode:    row.BankCode,
			AccountNo:   row.AccountNumber,
			AccountName: row.AccountName,
			Description: fmt.Sprintf("Payroll batch %s", p.BatchID),
			AccountType: infrastructure.AccountTypeBankAccount,
		})
		if err != nil {
			rowLogger.Error("Failed to initiate transfer", "error", err)
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = err.Error()
			failedCount++
			dataChanged = true
			continue
		}

		// Step 4: Record sync response (verified → authorised)
		accepted := result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess
		_, err = w.walletPaymentService.RecordSyncResponse(ctx, row.TransactionCode, disbursement.SyncResult{
			Accepted:     accepted,
			InvoiceNo:    result.ProviderRef,
			RawErrorCode: result.RawErrorCode,
			RawMessage:   result.RawMessage,
		})
		if err != nil {
			rowLogger.Error("Failed to record sync response", "error", err)
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = err.Error()
			failedCount++
			dataChanged = true
			continue
		}

		if accepted {
			row.TransferStatus = bulktransfer.ResultStatusCompleted
			completedCount++
			dataChanged = true
			rowLogger.Info("Transfer initiated successfully", "provider_ref", result.ProviderRef)
		} else {
			rowLogger.Warn("Transfer rejected by provider", "error_code", result.RawErrorCode)
			row.TransferStatus = bulktransfer.ResultStatusFailed
			row.ErrorMessage = result.RawMessage
			failedCount++
			dataChanged = true
		}
	}

	// Persist per-row status changes so GetBatchStatus can return failed_items.
	if dataChanged {
		updatedJSON, _ := json.Marshal(fileData)
		if err := w.fileRepo.UpdateWithLock(ctx, btf.ID, map[string]interface{}{"data": string(updatedJSON)}); err != nil {
			logger.Warn("Failed to persist per-row status", "error", err)
		}
	}

	logger.Info("9Pay bulk transfer batch processed",
		"total", len(fileData),
		"completed", completedCount,
		"failed", failedCount)

	return nil
}
