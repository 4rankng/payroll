package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/app/services/settlement"
	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"

	"gorm.io/gorm"
)

// BulkTransferTransactionWorker handles asynchronous transaction and ledger creation for bulk transfers
type BulkTransferTransactionWorker struct {
	logger             *slog.Logger
	db                 *gorm.DB
	transactionRepo    domain.TransactionRepository
	timesheetRepo      domain.TimesheetRepository
	assetRepo          domain.AssetRepository
	ledgerService      *settlement.LedgerService
	transactionService *settlement.TransactionService
	settingsConfig     *config.SettingsConfigService
	idempotencyService *infrastructure.IdempotencyService
	eventBus           domain.EventBus
}

// NewBulkTransferTransactionWorker creates a new BulkTransferTransactionWorker
func NewBulkTransferTransactionWorker(
	db *gorm.DB,
	transactionRepo domain.TransactionRepository,
	timesheetRepo domain.TimesheetRepository,
	assetRepo domain.AssetRepository,
	ledgerService *settlement.LedgerService,
	transactionService *settlement.TransactionService,
	settingsConfig *config.SettingsConfigService,
	idempotencyService *infrastructure.IdempotencyService,
	eventBus domain.EventBus,
) *BulkTransferTransactionWorker {
	return &BulkTransferTransactionWorker{
		logger:             slog.Default().With("component", "BulkTransferTransactionWorker"),
		db:                 db,
		transactionRepo:    transactionRepo,
		timesheetRepo:      timesheetRepo,
		assetRepo:          assetRepo,
		ledgerService:      ledgerService,
		transactionService: transactionService,
		settingsConfig:     settingsConfig,
		idempotencyService: idempotencyService,
		eventBus:           eventBus,
	}
}

// Handle processes domain events
func (w *BulkTransferTransactionWorker) Handle(ctx context.Context, event domain.DomainEvent) error {
	// Add panic recovery to prevent silent failures
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("🚨 [CRITICAL] BulkTransferTransactionWorker panicked during event handling",
				"event_type", event.EventType(),
				"panic", r,
				"stack_trace", fmt.Sprintf("%+v", r))
		}
	}()

	w.logger.Info("BulkTransferTransactionWorker received event",
		"event_type", event.EventType(),
		"event_value", fmt.Sprintf("%T", event))

	switch e := event.(type) {
	case domain.BulkTransferResultParsedEvent:
		w.logger.Info("Matched BulkTransferResultParsedEvent, processing...",
			"bulk_file_id", e.BulkFileID,
			"event_id", e.EventID,
			"total_amount", e.TotalAmount)

		// Add additional validation before processing
		if e.EventID == "" {
			w.logger.Error("❌ [VALIDATION] BulkTransferResultParsedEvent has empty EventID",
				"bulk_file_id", e.BulkFileID)
			return fmt.Errorf("invalid event: empty EventID")
		}

		err := w.handleBulkTransferResultParsed(ctx, e)
		if err != nil {
			w.logger.Error("❌ [ERROR] Failed to handle BulkTransferResultParsed event",
				"event_id", e.EventID,
				"bulk_file_id", e.BulkFileID,
				"error", err)
			return err
		}

		w.logger.Info("✅ [SUCCESS] BulkTransferResultParsed event processed successfully",
			"event_id", e.EventID,
			"bulk_file_id", e.BulkFileID)
		return nil

	default:
		w.logger.Debug("Event type not handled by BulkTransferTransactionWorker", "event_type", event.EventType())
		return nil // Ignore other events
	}
}

// CanHandle returns true if this worker can handle the given event type
func (w *BulkTransferTransactionWorker) CanHandle(eventType string) bool {
	return eventType == "BulkTransferResultParsed"
}

// handleBulkTransferResultParsed processes BulkTransferResultParsed events
func (w *BulkTransferTransactionWorker) handleBulkTransferResultParsed(ctx context.Context, event domain.BulkTransferResultParsedEvent) error {
	// Add panic recovery for the processing function
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("🚨 [CRITICAL] handleBulkTransferResultParsed panicked",
				"event_id", event.EventID,
				"bulk_file_id", event.BulkFileID,
				"panic", r)
		}
	}()

	w.logger.Info("🔵 [TRACE] Processing BulkTransferResultParsed event for transaction creation",
		"bulk_file_id", event.BulkFileID,
		"total_amount", event.TotalAmount,
		"asset_id", event.AssetID,
		"event_id", event.EventID,
		"filename", event.Filename)

	// Skip if no completed transfers (no revenue to record)
	if event.TotalAmount <= 0 {
		w.logger.Info("⚠️ [TRACE] No completed transfers, skipping transaction creation", "bulk_file_id", event.BulkFileID)
		return nil
	}

	// Use EventID (UUID) for idempotency instead of row ID
	idempotencyKey := fmt.Sprintf("bulk_transfer_transaction:%s", event.EventID)
	w.logger.Info("🔵 [TRACE] Attempting to acquire idempotency lock", "key", idempotencyKey, "event_id", event.EventID)
	acquired, state, err := w.idempotencyService.TryAcquireLock(ctx, idempotencyKey)
	if err != nil {
		w.logger.Error("❌ [TRACE] Failed to acquire idempotency lock",
			"key", idempotencyKey,
			"event_id", event.EventID,
			"error", err)
		// Don't return error here - this could be a Redis connectivity issue
		// and we want the event to be retried
		return fmt.Errorf("failed to acquire idempotency lock: %w", err)
	}

	w.logger.Info("🔵 [TRACE] Idempotency lock result",
		"acquired", acquired,
		"state", state,
		"key", idempotencyKey,
		"event_id", event.EventID)

	if !acquired {
		if state == infrastructure.StateCompleted {
			w.logger.Info("⚠️ [TRACE] Transaction already created (idempotency key completed)", "bulk_file_id", event.BulkFileID, "key", idempotencyKey)
			return nil
		}
		if state == infrastructure.StateProcessing {
			w.logger.Info("⚠️ [TRACE] Transaction already being created (idempotency key processing), will retry", "bulk_file_id", event.BulkFileID, "key", idempotencyKey)
			return fmt.Errorf("transaction creation in progress for key %s, retry after lock TTL", idempotencyKey)
		}
	}

	// Get asset details (optional for 9Pay batches with no uploaded file)
	var asset *domain.Asset
	if event.AssetID != 0 {
		asset, err = w.assetRepo.GetByID(ctx, event.AssetID)
		if err != nil {
			return fmt.Errorf("failed to get asset: %w", err)
		}
	} else {
		w.logger.Info("No asset linked, creating transaction without file reference")
	}

	// Unmarshal updated data
	var updatedData []dto.BulkTransferFileData
	if err := json.Unmarshal([]byte(event.UpdatedDataJSON), &updatedData); err != nil {
		w.logger.Error("❌ [TRACE] Failed to unmarshal updated data", "error", err)
		return fmt.Errorf("failed to unmarshal updated data: %w", err)
	}
	w.logger.Info("🔵 [TRACE] Unmarshaled updated data", "items_count", len(updatedData))

	// Step 1: resolve processedBy user
	processedBy := event.ProcessedBy
	if processedBy == 0 {
		processedBy = event.ActorUserID
	}
	if processedBy == 0 {
		processedBy = constants.SystemUserID
	}

	// Propagate the actor into ctx so downstream audit emits that read from
	// ctx (audit messages, IP/UA enrichment, future event factories) see the
	// real uploader instead of falling back to user_id=0. The event-bus worker
	// hands this handler context.Background(), which strips the request actor.
	ctx = auditctx.WithUserID(ctx, processedBy)

	// Step 2: check if a transaction already exists for this file (idempotency across retries)
	var createdTransactionID uint
	var existingTxn *domain.Transaction
	var getTxnErr error
	if asset != nil {
		existingTxn, getTxnErr = w.transactionRepo.GetByAssetID(ctx, asset.ID)
		if getTxnErr != nil && !domain.IsNotFoundError(getTxnErr) {
			if releaseErr := w.idempotencyService.ReleaseLock(ctx, idempotencyKey); releaseErr != nil {
				w.logger.Error("Failed to release idempotency lock after error", "key", idempotencyKey, "error", releaseErr)
			}
			return fmt.Errorf("failed to check existing transaction: %w", getTxnErr)
		}
	}

	var txn *domain.Transaction
	if existingTxn != nil {
		// Transaction for this bulk transfer already exists - reuse it for linking timesheets.
		w.logger.Info("✅ [TRACE] Found existing transaction for asset, reusing",
			"asset_id", asset.ID,
			"transaction_id", existingTxn.ID,
			"transaction_code", existingTxn.TransactionCode)
		txn = existingTxn
	} else {
		// Step 3: create transaction + ledger entries (each uses its own transactional logic)
		w.logger.Info("🔵 [TRACE] Creating new transaction and ledger entries",
			"amount", event.TotalAmount,
			"processed_by", processedBy)

		txn, err = w.createTransactionAndLedger(ctx, event.TotalAmount, asset, event.Filename, processedBy)
		if err != nil {
			w.logger.Error("❌ [TRACE] Failed to create transaction and ledger", "error", err)
			// Release lock so handler-level retry can safely re-acquire and retry this operation.
			if releaseErr := w.idempotencyService.ReleaseLock(ctx, idempotencyKey); releaseErr != nil {
				w.logger.Error("❌ [TRACE] Failed to release idempotency lock after error", "key", idempotencyKey, "error", releaseErr)
			}
			return fmt.Errorf("failed to create transaction and ledger: %w", err)
		}
		w.logger.Info("✅ [TRACE] Transaction created successfully",
			"transaction_id", txn.ID,
			"transaction_code", txn.TransactionCode)
	}

	createdTransactionID = txn.ID

	// Step 4: link timesheets to the transaction (separate operation for better retry semantics)
	w.logger.Info("🔵 [TRACE] Linking timesheets to transaction",
		"transaction_id", createdTransactionID,
		"items_count", len(updatedData))
	if err := w.linkTimesheetsToTransaction(ctx, createdTransactionID, updatedData); err != nil {
		w.logger.Error("❌ [TRACE] Failed to link timesheets to transaction",
			"transaction_id", createdTransactionID,
			"error", err)
		// Release lock so handler-level retry can safely re-acquire and retry this operation.
		if releaseErr := w.idempotencyService.ReleaseLock(ctx, idempotencyKey); releaseErr != nil {
			w.logger.Error("❌ [TRACE] Failed to release idempotency lock after link failure",
				"key", idempotencyKey,
				"error", releaseErr)
		}
		return fmt.Errorf("failed to link timesheets to transaction: %w", err)
	}
	w.logger.Info("✅ [TRACE] Timesheets linked successfully", "transaction_id", createdTransactionID)

	// Mark as completed
	w.logger.Info("🔵 [TRACE] Marking idempotency key as completed", "key", idempotencyKey)
	if err := w.idempotencyService.MarkCompleted(ctx, idempotencyKey); err != nil {
		w.logger.Error("❌ [TRACE] Failed to mark idempotency key as completed", "key", idempotencyKey, "error", err)
	} else {
		w.logger.Info("✅ [TRACE] Idempotency key marked as completed", "key", idempotencyKey)
	}

	w.logger.Info("✅✅✅ [TRACE] Successfully created transaction and linked timesheets",
		"bulk_file_id", event.BulkFileID,
		"transaction_id", createdTransactionID,
		"key", idempotencyKey)

	return nil
}

// createTransactionAndLedger creates transaction and ledger entries
func (w *BulkTransferTransactionWorker) createTransactionAndLedger(
	ctx context.Context,
	amount int64,
	asset *domain.Asset,
	filename string,
	processedBy uint,
) (*domain.Transaction, error) {
	partnerCompany := w.settingsConfig.GetPartnerCompany(ctx)

	// Build ledger plan to calculate receivable amount (including fee)
	plan := bulktransfer.BuildLedgerPlan(ctx, w.settingsConfig, float64(amount), partnerCompany, filename)

	// Create transaction record for the payroll bulk transfer revenue
	// Amount should be the receivable amount (including fee)
	transaction := &domain.Transaction{
		Amount:          plan.ReceivableAmount,
		TransactionType: domain.TransactionTypeRevenue,
		Description:     plan.Description,
		Status:          domain.TransactionStatusPending,
		AssetID:         assetIDPtr(asset),
		CreatedBy:       processedBy,
		Party:           partnerCompany,
	}

	// Use transaction service to create with proper outbox event handling
	// This will trigger TransactionCreated event → LedgerWorker creates Receivable debit + Revenue credit
	createdTxn, _, err := w.transactionService.CreateTransaction(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create manual ledger entries for cash outflow and revenue offset
	// These are the bulk transfer specific entries (Cash credit + Revenue debit)
	entries := plan.Entries(createdTxn.CreatedAt, processedBy, partnerCompany)

	// Link entries to the transaction
	for _, entry := range entries {
		entry.TransactionID = &createdTxn.ID
	}

	if _, err := w.ledgerService.CreateEntries(ctx, entries, processedBy); err != nil {
		if domain.IsConflictError(err) {
			w.logger.Info("Ledger entries already exist, skipping",
				"transaction_id", createdTxn.ID)
		} else {
			return nil, fmt.Errorf("failed to create ledger entries: %w", err)
		}
	}

	w.logger.Info("Transaction and ledger entries ready",
		"transaction_id", createdTxn.ID,
		"receivable_amount", plan.ReceivableAmount,
		"cash_out_amount", plan.CashOutAmount,
		"net_revenue", plan.ReceivableAmount-plan.CashOutAmount)

	return createdTxn, nil
}

// linkTimesheetsToTransaction links all successfully paid timesheets to the created revenue transaction
func (w *BulkTransferTransactionWorker) linkTimesheetsToTransaction(
	ctx context.Context,
	transactionID uint,
	updatedData []dto.BulkTransferFileData,
) error {
	w.logger.Info("🔵 [TRACE] linkTimesheetsToTransaction called", "transaction_id", transactionID, "items_count", len(updatedData))

	var timesheetIDs []uint

	for i, item := range updatedData {
		w.logger.Info("🔵 [TRACE] Processing item", "index", i, "transfer_status", item.TransferStatus, "timesheet_ids", item.TimesheetIDs)
		if item.TransferStatus != bulktransfer.ResultStatusCompleted {
			w.logger.Info("⚠️ [TRACE] Skipping item - not completed", "index", i, "transfer_status", item.TransferStatus)
			continue
		}
		timesheetIDs = append(timesheetIDs, item.TimesheetIDs...)
	}

	w.logger.Info("🔵 [TRACE] Collected timesheet IDs to link", "count", len(timesheetIDs), "ids", timesheetIDs)

	if len(timesheetIDs) == 0 {
		w.logger.Info("⚠️ [TRACE] No timesheets to link", "transaction_id", transactionID)
		return nil
	}

	w.logger.Info("🔵 [TRACE] Calling BulkUpdateTransactionID", "transaction_id", transactionID, "timesheet_ids", timesheetIDs)
	err := w.timesheetRepo.BulkUpdateTransactionID(ctx, transactionID, timesheetIDs)
	if err != nil {
		w.logger.Error("❌ [TRACE] BulkUpdateTransactionID failed", "error", err)
		return err
	}
	w.logger.Info("✅ [TRACE] BulkUpdateTransactionID succeeded", "updated_count", len(timesheetIDs))
	return nil
}
func assetIDPtr(a *domain.Asset) *uint {
	if a == nil {
		return nil
	}
	return &a.ID
}
