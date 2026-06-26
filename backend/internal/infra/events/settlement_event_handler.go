package events

import (
	"api-server/internal/pkg/clock"
	auditctx "api-server/internal/pkg/context"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"api-server/internal/app/services/ledger"
	"api-server/internal/app/services/notification"
	"api-server/internal/constants"
	"api-server/internal/domain"
	serviceports "api-server/internal/domain/ports/services"
	"api-server/internal/infra/observability"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SettlementEventHandler handles settlement-related domain events
type SettlementEventHandler struct {
	db                 *gorm.DB
	timesheetRepo      domain.TimesheetRepository
	transactionRepo    domain.TransactionRepository
	settlementRepo     domain.SettlementRepository
	ledgerRepo         domain.LedgerEntryRepository
	transactionService serviceports.TransactionPort
	eventBus           domain.EventBus
	employeeNotifier   notification.EmployeeNotifier
	logger             *slog.Logger
}

// NewSettlementEventHandler creates a new settlement event handler
func NewSettlementEventHandler(
	db *gorm.DB,
	timesheetRepo domain.TimesheetRepository,
	transactionRepo domain.TransactionRepository,
	settlementRepo domain.SettlementRepository,
	ledgerRepo domain.LedgerEntryRepository,
	transactionService serviceports.TransactionPort,
	eventBus domain.EventBus,
	employeeNotifier notification.EmployeeNotifier,
) *SettlementEventHandler {
	return &SettlementEventHandler{
		db:                 db,
		timesheetRepo:      timesheetRepo,
		transactionRepo:    transactionRepo,
		settlementRepo:     settlementRepo,
		ledgerRepo:         ledgerRepo,
		transactionService: transactionService,
		eventBus:           eventBus,
		employeeNotifier:   employeeNotifier,
		logger:             observability.GetLogger().With("component", "SettlementEventHandler"),
	}
}

// CanHandle checks if the handler can handle the given event type
func (h *SettlementEventHandler) CanHandle(eventType string) bool {
	switch eventType {
	case "SettlementUploadProcessed":
		return true
	case "TimesheetMarking":
		return true
	case "SettlementAppliedFromUpload":
		return true
	case "TimesheetsRevenuePaidFromInternal":
		return true
	default:
		return false
	}
}

// Handle processes settlement-related events
func (h *SettlementEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	logger := observability.GetLogger()

	switch e := event.(type) {
	case domain.SettlementUploadProcessedEvent:
		return h.handleSettlementUploadProcessed(ctx, e)
	case domain.TimesheetMarkingEvent:
		return h.handleTimesheetMarking(ctx, e)
	case domain.SettlementAppliedFromUploadEvent:
		return h.handleSettlementAppliedFromUpload(ctx, e)
	case domain.TimesheetsRevenuePaidFromInternalEvent:
		return h.handleTimesheetsRevenuePaidFromInternal(ctx, e)
	default:
		logger.Warn("Unknown event type for settlement handler", "event_type", event.EventType())
		return fmt.Errorf("unknown event type: %s", event.EventType())
	}
}

// handleSettlementUploadProcessed handles SettlementUploadProcessedEvent
// This emits a TimesheetMarkingEvent for the specific timesheets from INTERNAL sheet
func (h *SettlementEventHandler) handleSettlementUploadProcessed(ctx context.Context, event domain.SettlementUploadProcessedEvent) error {
	logger := observability.GetLogger()

	// Emit TimesheetMarkingEvent for the INTERNAL sheet timesheets
	timesheetMarkingEvent := domain.NewTimesheetMarkingEvent(
		ctx,
		event.TimesheetIDs,
		"paid",
		"upload_internal_sheet",
		&event.AssetID,
	)

	if err := h.eventBus.Publish(ctx, timesheetMarkingEvent); err != nil {
		logger.Error("Failed to publish TimesheetMarkingEvent",
			"asset_id", event.AssetID,
			"timesheet_count", len(event.TimesheetIDs),
			"error", err)
		return err
	}

	logger.Info("Successfully emitted TimesheetMarkingEvent for upload processing",
		"asset_id", event.AssetID,
		"timesheet_count", len(event.TimesheetIDs))

	return nil
}

// handleTimesheetMarking handles TimesheetMarkingEvent
// This actually marks the specific timesheets as paid
func (h *SettlementEventHandler) handleTimesheetMarking(ctx context.Context, event domain.TimesheetMarkingEvent) error {
	logger := observability.GetLogger()

	if event.MarkedAs != "paid" {
		logger.Warn("Unsupported marking operation", "marked_as", event.MarkedAs)
		return fmt.Errorf("unsupported marking operation: %s", event.MarkedAs)
	}

	// Validate that none of the timesheets are already paid
	timesheets, err := h.timesheetRepo.GetByIDs(ctx, event.TimesheetIDs)
	if err != nil {
		logger.Error("Failed to get timesheets for marking",
			"timesheet_ids", event.TimesheetIDs,
			"error", err)
		return err
	}

	var alreadyPaidIDs []uint
	for _, ts := range timesheets {
		if ts.RevenuePaid {
			alreadyPaidIDs = append(alreadyPaidIDs, ts.ID)
		}
	}

	if len(alreadyPaidIDs) > 0 {
		logger.Warn("Some timesheets are already paid, skipping them",
			"already_paid_ids", alreadyPaidIDs,
			"total_requested", len(event.TimesheetIDs))

		// Filter out already paid timesheets
		var validIDs []uint
		timesheetSet := make(map[uint]struct{})
		for _, id := range alreadyPaidIDs {
			timesheetSet[id] = struct{}{}
		}
		for _, id := range event.TimesheetIDs {
			if _, exists := timesheetSet[id]; !exists {
				validIDs = append(validIDs, id)
			}
		}
		event.TimesheetIDs = validIDs
	}

	if len(event.TimesheetIDs) == 0 {
		logger.Info("No valid timesheets to mark")
		return nil
	}

	// Mark timesheets as paid
	if err := h.timesheetRepo.BulkUpdateRevenuePaid(ctx, event.TimesheetIDs); err != nil {
		logger.Error("Failed to mark timesheets as paid",
			"source", event.Source,
			"timesheet_count", len(event.TimesheetIDs),
			"error", err)
		return err
	}

	logger.Info("Successfully marked timesheets as paid",
		"source", event.Source,
		"timesheet_count", len(event.TimesheetIDs))

	// Notify employees about paid timesheets
	if h.employeeNotifier != nil {
		paidTimesheets, err := h.timesheetRepo.GetByIDs(ctx, event.TimesheetIDs)
		if err != nil {
			logger.Error("Failed to fetch paid timesheets for notification", "error", err)
		} else {
			h.employeeNotifier.NotifyTimesheetPaid(ctx, paidTimesheets)
		}
	}

	return nil
}

// ApplySettlement creates the settlement record, the double-entry ledger pair, and
// marks the given timesheets revenue_paid=1 for a single transaction — all inside
// one DB transaction with deadlock retry.
//
// It is invoked SYNCHRONOUSLY by the settlement-upload flow (SettlementApplier),
// so a dropped async event can never silently strand a receivable. The async event
// path (advance-payment settlements, and informational re-publishes) still routes
// here via handleSettlementAppliedFromUpload.
//
// Idempotent: if every timesheet in timesheetIDs is already revenue_paid=1 the call
// is a no-op. This makes a post-settlement informational re-publish of
// SettlementAppliedFromUploadEvent safe (audit-only) and protects against duplicate
// events. The actor (CreatedBy) is read from the request context.
func (h *SettlementEventHandler) ApplySettlement(
	ctx context.Context,
	transactionID uint,
	amount int64,
	fileID uint,
	filename string,
	timesheetIDs []uint,
) error {
	logger := observability.GetLogger()

	var createdBy uint
	if uid := auditctx.GetUserID(ctx); uid != nil {
		createdBy = *uid
	}

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Create transaction context
			txCtx := domain.WithTransactionContext(ctx, &domain.TransactionContext{
				TX:              tx,
				IsTransactional: true,
			})

			// Idempotency guard: if all linked timesheets are already settled, this is a
			// duplicate (e.g. informational re-publish after a synchronous settlement) — no-op.
			// Advance-payment events pass nil timesheetIDs and bypass this guard.
			if len(timesheetIDs) > 0 {
				existing, err := h.timesheetRepo.GetByIDs(txCtx, timesheetIDs)
				if err != nil {
					return fmt.Errorf("failed to load timesheets for idempotency check: %w", err)
				}
				allPaid := len(existing) > 0
				for _, ts := range existing {
					if !ts.RevenuePaid {
						allPaid = false
						break
					}
				}
				if allPaid {
					logger.Info("Settlement already applied (timesheets already revenue_paid), skipping",
						"transaction_id", transactionID)
					return nil
				}
			}

			// Get the transaction with row-level lock to prevent concurrent settlement race conditions
			txn, err := h.transactionRepo.GetByIDForUpdate(txCtx, transactionID)
			if err != nil {
				logger.Error("Failed to get transaction for settlement",
					"transaction_id", transactionID,
					"error", err)
				return err
			}

			// Re-validate remaining amount under lock to skip duplicate settlements idempotently
			remainingBeforeSettlement := txn.GetRemainingAmount()
			if amount > remainingBeforeSettlement+constants.RoundingTolerance {
				logger.Warn("Skipping settlement — transaction already settled",
					"transaction_id", txn.ID,
					"settlement_amount", amount,
					"remaining", remainingBeforeSettlement)
				return nil
			}

			// Create settlement record
			settlement := &domain.Settlement{
				TransactionID:  transactionID,
				Amount:         amount,
				SettlementDate: clock.Now(),
				ProofAssetID:   &fileID,
				PaymentMethod:  "bank_transfer",
				Notes:          fmt.Sprintf("Settlement from upload: %s", filename),
				CreatedBy:      createdBy,
			}

			// Generate a settlement UUID for idempotency and ledger processing consistency
			settlementUUID := uuid.NewString()
			settlement.SettlementUUID = &settlementUUID

			// Validate settlement
			if err := settlement.Validate(); err != nil {
				logger.Error("Settlement validation failed",
					"transaction_id", transactionID,
					"error", err)
				return err
			}

			// Create settlement in database
			if err := h.settlementRepo.Create(txCtx, settlement); err != nil {
				logger.Error("Failed to create settlement record",
					"transaction_id", transactionID,
					"amount", amount,
					"error", err)
				return err
			}

			// Recalculate settled amount from settlements (they are the source of truth)
			// Since we just created a settlement, we need to get total settled amount
			totalSettled, err := h.settlementRepo.GetTotalSettledAmount(txCtx, transactionID)
			if err != nil {
				logger.Error("Failed to get total settled amount",
					"transaction_id", transactionID,
					"error", err)
				return err
			}

			// Check for potential overflow or data corruption
			var remaining int64
			if totalSettled > txn.Amount {
				logger.Error("Total settled exceeds transaction amount - possible data corruption",
					"transaction_id", txn.ID,
					"amount", txn.Amount,
					"total_settled", totalSettled)
				remaining = 0
			} else {
				remaining = txn.Amount - totalSettled
			}

			// Keep transaction.settled_amount in sync with settlements (single source of truth)
			txn.SettledAmount = totalSettled
			txn.NormalizeSettledAmount(totalSettled > 0)

			// Check if transaction should be marked as fully settled
			if remaining <= constants.RoundingTolerance {
				// Transaction is fully settled (within tolerance)
				// Clamp settled_amount to full amount for a clean state
				txn.SettledAmount = txn.Amount
				txn.Status = domain.TransactionStatusSettled
			} else if txn.SettledAmount > 0 {
				// Transaction is partially settled
				txn.Status = domain.TransactionStatusPartiallySettled
			} else {
				// No effective settlement applied
				txn.Status = domain.TransactionStatusPending
			}

			// Update transaction status first before creating ledger entries / publishing event
			if err := h.transactionRepo.Update(txCtx, txn); err != nil {
				logger.Error("Failed to update transaction after settlement",
					"transaction_id", txn.ID,
					"error", err)
				return err
			}

			// Create the settlement's double-entry ledger pair INSIDE this DB transaction.
			// Settlement + revenue_paid marking + ledger entries now commit atomically.
			// This eliminates the orphan-state class of bug (settlement exists, ledger entries don't yet)
			// that previously required a recovery script + waiting for the 1-minute outbox cron.
			ledgerEntries, err := ledger.BuildSettlementLedgerEntries(txn, settlement)
			if err != nil {
				logger.Error("Failed to build settlement ledger entries",
					"settlement_id", settlement.ID,
					"transaction_id", txn.ID,
					"error", err)
				return fmt.Errorf("failed to build settlement ledger entries: %w", err)
			}
			if err := h.ledgerRepo.CreateTransaction(txCtx, ledgerEntries); err != nil {
				logger.Error("Failed to persist settlement ledger entries",
					"settlement_id", settlement.ID,
					"transaction_id", txn.ID,
					"error", err)
				return fmt.Errorf("failed to create settlement ledger entries: %w", err)
			}
			logger.Info("Settlement ledger entries created inline",
				"settlement_id", settlement.ID,
				"transaction_id", txn.ID,
				"entries", len(ledgerEntries))

			// Mark timesheets revenue_paid=1 atomically with settlement creation.
			// Both happen in the same DB transaction — if the settlement creation deadlocks and rolls
			// back, the revenue_paid flag is not set either. This prevents the orphan state
			// (revenue_paid=1 but no settlement record) that previously required manual recovery.
			if len(timesheetIDs) > 0 {
				if err := h.timesheetRepo.BulkUpdateRevenuePaid(txCtx, timesheetIDs); err != nil {
					logger.Error("Failed to mark timesheets as revenue paid",
						"settlement_id", settlement.ID,
						"timesheet_count", len(timesheetIDs),
						"error", err)
					return err
				}
				logger.Info("Marked timesheets as revenue paid",
					"settlement_id", settlement.ID,
					"timesheet_count", len(timesheetIDs))
			} else {
				logger.Warn("No timesheets to mark as revenue paid for settlement",
					"settlement_id", settlement.ID,
					"transaction_id", transactionID)
			}

			return nil
		})

		if err == nil {
			return nil
		}

		if strings.Contains(err.Error(), "Deadlock") || strings.Contains(err.Error(), "1213") {
			// Exponential backoff with jitter to reduce concurrent retry collisions
			backoff := time.Duration(50*(1<<attempt)) * time.Millisecond
			jitter := time.Duration(rand.Intn(50)) * time.Millisecond
			logger.Warn("Deadlock on settlement, retrying",
				"transaction_id", transactionID,
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"backoff_ms", (backoff + jitter).Milliseconds())
			time.Sleep(backoff + jitter)
			continue
		}

		return err
	}

	logger.Error("Settlement failed after retries due to persistent deadlock",
		"transaction_id", transactionID)
	return fmt.Errorf("persistent deadlock for transaction %d after %d retries", transactionID, maxRetries)
}

// handleSettlementAppliedFromUpload handles SettlementAppliedFromUploadEvent.
// Timesheet settlements are now applied synchronously in the upload service; this
// async handler still serves the advance-payment flow and informational re-publishes
// (ApplySettlement is idempotent, so a re-publish is an audit-only no-op).
func (h *SettlementEventHandler) handleSettlementAppliedFromUpload(ctx context.Context, event domain.SettlementAppliedFromUploadEvent) error {
	return h.ApplySettlement(ctx, event.TransactionID, event.Amount, event.FileID, event.Filename, event.TimesheetIDs)
}

// handleTimesheetsRevenuePaidFromInternal handles TimesheetsRevenuePaidFromInternalEvent
// This marks all timesheets from INTERNAL sheet as revenue_paid
func (h *SettlementEventHandler) handleTimesheetsRevenuePaidFromInternal(ctx context.Context, event domain.TimesheetsRevenuePaidFromInternalEvent) error {
	logger := observability.GetLogger()

	if len(event.TimesheetIDs) == 0 {
		logger.Info("No timesheets to mark as paid")
		return nil
	}

	// Validate that none of the timesheets are already paid (idempotency)
	timesheets, err := h.timesheetRepo.GetByIDs(ctx, event.TimesheetIDs)
	if err != nil {
		logger.Error("Failed to get timesheets for marking",
			"timesheet_ids", event.TimesheetIDs,
			"error", err)
		return err
	}

	var alreadyPaidIDs []uint
	var validIDs []uint
	for _, ts := range timesheets {
		if ts.RevenuePaid {
			alreadyPaidIDs = append(alreadyPaidIDs, ts.ID)
		} else {
			validIDs = append(validIDs, ts.ID)
		}
	}

	if len(alreadyPaidIDs) > 0 {
		logger.Warn("Some timesheets are already paid, skipping them",
			"already_paid_ids", alreadyPaidIDs,
			"total_requested", len(event.TimesheetIDs))
	}

	if len(validIDs) == 0 {
		logger.Info("No valid timesheets to mark (all already paid)")
		return nil
	}

	// Mark timesheets as paid
	if err := h.timesheetRepo.BulkUpdateRevenuePaid(ctx, validIDs); err != nil {
		logger.Error("Failed to mark timesheets as paid from INTERNAL sheet",
			"file_id", event.FileID,
			"timesheet_count", len(validIDs),
			"error", err)
		return err
	}

	logger.Info("Successfully marked timesheets as paid from INTERNAL sheet",
		"file_id", event.FileID,
		"filename", event.Filename,
		"timesheet_count", len(validIDs),
		"skipped_count", len(alreadyPaidIDs))

	return nil
}
