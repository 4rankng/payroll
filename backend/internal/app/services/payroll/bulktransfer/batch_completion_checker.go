package bulktransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/infra/observability"
)

// NinePayBatchCompletionChecker detects when all transfers in a 9Pay bulk
// transfer batch have reached a terminal state (completed or failed), then
// publishes a BulkTransferResultParsedEvent so the existing payment and
// transaction workers update timesheet statuses and create ledger entries.
type NinePayBatchCompletionChecker struct {
	transactionCodeRepo TransactionCodeRepository
	fileRepo            BulkTransferFileRepository
	walletPaymentRepo   domaintx.WalletPaymentRepository
	idempotencyService  *infrastructure.IdempotencyService
	eventBus            domain.EventBus
	logger              *slog.Logger
}

func NewNinePayBatchCompletionChecker(
	transactionCodeRepo TransactionCodeRepository,
	fileRepo BulkTransferFileRepository,
	walletPaymentRepo domaintx.WalletPaymentRepository,
	idempotencyService *infrastructure.IdempotencyService,
	eventBus domain.EventBus,
	logger *slog.Logger,
) *NinePayBatchCompletionChecker {
	if logger == nil {
		logger = observability.GetLogger()
	}
	return &NinePayBatchCompletionChecker{
		transactionCodeRepo: transactionCodeRepo,
		fileRepo:            fileRepo,
		walletPaymentRepo:   walletPaymentRepo,
		idempotencyService:  idempotencyService,
		eventBus:            eventBus,
		logger:              logger.With("component", "NinePayBatchCompletionChecker"),
	}
}

// CheckAndCompleteBatch checks if the wallet payment belonging to the given
// requestID is part of a 9Pay bulk transfer batch, and if all transfers in
// that batch are terminal, publishes the completion event.
func (c *NinePayBatchCompletionChecker) CheckAndCompleteBatch(ctx context.Context, requestID string) error {
	// Look up the transaction code to find the batch
	tc, err := c.transactionCodeRepo.GetByCode(ctx, requestID)
	if err != nil {
		// Not all transaction codes belong to bulk transfers — ignore silently
		return nil
	}

	var tcData domain.TransactionCodeData
	if err := json.Unmarshal(tc.Data, &tcData); err != nil {
		return nil
	}

	fileID := tcData.GetFileID()
	if fileID == nil {
		return nil // No batch linked
	}

	// Load the bulk transfer file
	btf, err := c.fileRepo.GetByID(ctx, *fileID)
	if err != nil {
		return nil
	}

	// Only process 9Pay batches
	if btf.Source != "ninepay" {
		return nil
	}

	// Query all wallet payments for this batch
	payments, err := c.walletPaymentRepo.ListByBatchID(ctx, btf.Filename)
	if err != nil {
		return fmt.Errorf("list wallet payments: %w", err)
	}

	// Count terminal states
	var completedCount, failedCount int
	for _, p := range payments {
		if p.IsTerminal() {
			if p.Status == domaintx.StateCompleted {
				completedCount++
			} else {
				failedCount++
			}
		}
	}

	// Not all terminal yet
	if completedCount+failedCount < btf.TransactionsCount {
		c.logger.Info("Batch not yet complete",
			"batch_id", btf.Filename,
			"total", btf.TransactionsCount,
			"completed", completedCount,
			"failed", failedCount)

		// Update counts on the file record for progress tracking
		_ = c.fileRepo.UpdateCounts(ctx, btf.ID, completedCount, failedCount)
		return nil
	}

	// All terminal — acquire idempotency lock
	lockKey := fmt.Sprintf("ninepay_batch_complete:%d", btf.ID)
	acquired, state, err := c.idempotencyService.TryAcquireLock(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	if !acquired {
		c.logger.Info("Batch completion already in progress or done",
			"batch_id", btf.Filename, "state", state)
		return nil
	}

	c.logger.Info("All transfers terminal — completing batch",
		"batch_id", btf.Filename,
		"total", btf.TransactionsCount,
		"completed", completedCount,
		"failed", failedCount)

	// Update file counts
	if err := c.fileRepo.UpdateCounts(ctx, btf.ID, completedCount, failedCount); err != nil {
		c.logger.Warn("Failed to update batch counts", "error", err)
	}

	// Build updated data from wallet payments
	fileData, err := dto.ParseBulkTransferFileData(btf.Data)
	if err != nil {
		return fmt.Errorf("parse file data: %w", err)
	}

	// Merge wallet payment statuses into file data
	paymentByRequestID := make(map[string]*domaintx.WalletPayment, len(payments))
	for _, p := range payments {
		paymentByRequestID[p.RequestID] = p
	}

	var totalAmount int64
	for i := range fileData {
		p, ok := paymentByRequestID[fileData[i].TransactionCode]
		if !ok {
			continue
		}
		if p.Status == domaintx.StateCompleted {
			fileData[i].TransferStatus = ResultStatusCompleted
			totalAmount += fileData[i].Amount
		} else {
			fileData[i].TransferStatus = ResultStatusFailed
			if p.ErrorMessage != nil {
				fileData[i].ErrorMessage = *p.ErrorMessage
			}
		}
	}

	updatedDataJSON, _ := json.Marshal(fileData)

	// Publish BulkTransferResultParsedEvent — same event the manual flow uses
	event := domain.NewBulkTransferResultParsedEvent(
		ctx,
		btf.ID,
		0, // no asset for 9Pay batches
		btf.Filename,
		btf.TransactionsCount,
		completedCount,
		failedCount,
		totalAmount,
		"", // no parsed data JSON needed
		string(updatedDataJSON),
		btf.CreatedBy,
	)

	if err := c.eventBus.Publish(ctx, event); err != nil {
		// Release lock so we can retry
		_ = c.idempotencyService.ReleaseLock(ctx, lockKey)
		return fmt.Errorf("publish event: %w", err)
	}

	if err := c.idempotencyService.MarkCompleted(ctx, lockKey); err != nil {
		c.logger.Warn("Failed to mark idempotency key as completed", "error", err)
	}

	c.logger.Info("Batch completion event published",
		"batch_id", btf.Filename,
		"total_amount", totalAmount)

	return nil
}
