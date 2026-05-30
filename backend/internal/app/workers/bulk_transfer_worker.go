package workers

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/domain"
	cacheport "api-server/internal/domain/ports/infrastructure"

	"gorm.io/gorm"
)

// BulkTransferPaymentWorker handles per-transfer timesheet payment updates
// triggered directly from IPN processing (no longer event-driven).
type BulkTransferPaymentWorker struct {
	logger              *slog.Logger
	db                  *gorm.DB
	timesheetRepo       domain.TimesheetRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	tcRepo              domain.TransactionCodeRepository
	settingsConfig      *config.SettingsConfigService
	idempotencyService  *infrastructure.IdempotencyService
	notificationService *notification.NotificationService
	employeeNotifier    notification.EmployeeNotifier
	cache               cacheport.CachePort
}

// NewBulkTransferPaymentWorker creates a new BulkTransferPaymentWorker
func NewBulkTransferPaymentWorker(
	db *gorm.DB,
	timesheetRepo domain.TimesheetRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	tcRepo domain.TransactionCodeRepository,
	settingsConfig *config.SettingsConfigService,
	idempotencyService *infrastructure.IdempotencyService,
	notificationService *notification.NotificationService,
	employeeNotifier notification.EmployeeNotifier,
	cache cacheport.CachePort,
) *BulkTransferPaymentWorker {
	return &BulkTransferPaymentWorker{
		logger:              slog.Default().With("component", "BulkTransferPaymentWorker"),
		db:                  db,
		timesheetRepo:       timesheetRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		tcRepo:              tcRepo,
		settingsConfig:      settingsConfig,
		idempotencyService:  idempotencyService,
		notificationService: notificationService,
		employeeNotifier:    employeeNotifier,
		cache:               cache,
	}
}

// UpdateForTransfer updates timesheet payment statuses for a single completed or
// failed transfer, identified by its requestID (the transaction_code).
func (w *BulkTransferPaymentWorker) UpdateForTransfer(ctx context.Context, requestID string, completed bool, invoiceNo string) error {
	// Idempotency — skip if already processed
	idempotencyKey := fmt.Sprintf("transfer_ts_update:%s", requestID)
	acquired, state, err := w.idempotencyService.TryAcquireLock(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("acquire idempotency lock: %w", err)
	}
	if !acquired {
		w.logger.Info("timesheet update already processed", "request_id", requestID, "state", state)
		return nil
	}

	// Look up transaction code to find timesheet IDs
	tc, err := w.tcRepo.GetByCode(ctx, requestID)
	if err != nil {
		// Not a bulk transfer (e.g. flex pay) — nothing to do
		_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
		return nil
	}

	var tcData domain.TransactionCodeData
	if err := json.Unmarshal(tc.Data, &tcData); err != nil {
		_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
		return nil
	}

	timesheetIDs := tcData.GetTimesheetIDs()
	if len(timesheetIDs) == 0 {
		_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
		return nil
	}

	timesheets, err := w.timesheetRepo.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
		return fmt.Errorf("get timesheets: %w", err)
	}

	// Filter out already-paid timesheets
	var pendingTimesheets []*domain.Timesheet
	for _, ts := range timesheets {
		if ts.PaymentStatus == domain.PaymentStatusPaid {
			continue
		}
		pendingTimesheets = append(pendingTimesheets, ts)
	}
	if len(pendingTimesheets) == 0 {
		_ = w.idempotencyService.MarkCompleted(ctx, idempotencyKey)
		return nil
	}

	processedAt := clock.Now()

	var updates []domain.PaymentStatusUpdate
	if completed {
		paidAmounts, calcErr := w.calculatePaidAmounts(ctx, pendingTimesheets)
		if calcErr != nil {
			_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
			return fmt.Errorf("calculate paid amounts: %w", calcErr)
		}

		paymentRef := invoiceNo
		for _, ts := range pendingTimesheets {
			paidAmt := paidAmounts[ts.ID]
			updates = append(updates, domain.PaymentStatusUpdate{
				TimesheetID:      ts.ID,
				PaymentStatus:    domain.PaymentStatusPaid,
				PaymentReference: &paymentRef,
				PaymentDate:      &processedAt,
				PaidAmount:       &paidAmt,
			})
		}
	} else {
		for _, ts := range pendingTimesheets {
			updates = append(updates, domain.PaymentStatusUpdate{
				TimesheetID:   ts.ID,
				PaymentStatus: domain.PaymentStatusFailed,
				PaymentDate:   &processedAt,
			})
		}
	}

	if err := w.timesheetRepo.BulkUpdatePaymentStatus(ctx, updates); err != nil {
		_ = w.idempotencyService.ReleaseLock(ctx, idempotencyKey)
		return fmt.Errorf("update payment statuses: %w", err)
	}

	// Invalidate timesheet caches so list/summary queries reflect the new payment status
	if w.cache != nil {
		_ = w.cache.InvalidatePattern(ctx, "timesheets:list:*")
		_ = w.cache.InvalidatePattern(ctx, "timesheets:summary:*")
	}

	// Update revenue_receivable for paid timesheets
	if completed {
		feePercentage := w.settingsConfig.GetAdvanceCashFeePercentage(ctx)
		if err := w.updateRevenueReceivable(ctx, updates, feePercentage); err != nil {
			w.logger.Error("failed to update revenue_receivable", "error", err)
		}
	}

	if err := w.idempotencyService.MarkCompleted(ctx, idempotencyKey); err != nil {
		w.logger.Error("failed to mark idempotency key as completed", "key", idempotencyKey, "error", err)
	}

	// Notify employees
	if w.employeeNotifier != nil && completed {
		w.employeeNotifier.NotifyTimesheetPaid(ctx, pendingTimesheets)
	}

	w.logger.Info("per-transfer timesheet update applied",
		"request_id", requestID,
		"completed", completed,
		"timesheet_count", len(updates))

	return nil
}

// Handle processes domain events (implements domain.EventHandler).
// When a BulkTransferResultParsed event arrives (from result file upload or
// 9Pay batch completion), it iterates over each completed transfer and calls
// UpdateForTransfer to mark the associated timesheets as paid.
func (w *BulkTransferPaymentWorker) Handle(ctx context.Context, event domain.DomainEvent) error {
	e, ok := event.(domain.BulkTransferResultParsedEvent)
	if !ok {
		return nil
	}

	var updatedData []dto.BulkTransferFileData
	if err := json.Unmarshal([]byte(e.UpdatedDataJSON), &updatedData); err != nil {
		return fmt.Errorf("unmarshal updated data: %w", err)
	}

	var completedCount, failedCount int
	for _, item := range updatedData {
		completed := item.TransferStatus == bulktransfer.ResultStatusCompleted
		if err := w.UpdateForTransfer(ctx, item.TransactionCode, completed, item.BankTxnRef); err != nil {
			w.logger.Error("failed to update timesheet for transfer",
				"transaction_code", item.TransactionCode,
				"error", err)
			continue
		}
		if completed {
			completedCount++
		} else if item.TransferStatus == bulktransfer.ResultStatusFailed {
			failedCount++
		}
	}

	w.logger.Info("bulk transfer payment status updates applied",
		"event_id", e.EventID,
		"completed", completedCount,
		"failed", failedCount,
		"total", len(updatedData))

	return nil
}

// CanHandle returns true for BulkTransferResultParsed events
func (w *BulkTransferPaymentWorker) CanHandle(eventType string) bool {
	return eventType == "BulkTransferResultParsed"
}

// calculatePaidAmounts calculates the paid amount for each timesheet based on the employee's payment schedule
func (w *BulkTransferPaymentWorker) calculatePaidAmounts(ctx context.Context, timesheets []*domain.Timesheet) (map[uint]int64, error) {
	result := make(map[uint]int64)

	assignmentMap, err := w.batchFetchAssignments(ctx, timesheets)
	if err != nil {
		w.logger.Error("Failed to batch fetch assignments", "error", err)
		assignmentMap = make(map[string]*domain.ProjectEmployee)
	}

	for _, timesheet := range timesheets {
		assignmentKey := fmt.Sprintf("%d:%d", timesheet.ProjectID, timesheet.EmployeeID)
		assignment, exists := assignmentMap[assignmentKey]

		var percentage float64
		if !exists {
			w.logger.Warn("Failed to get assignment for timesheet, using default weekly schedule",
				"timesheet_id", timesheet.ID,
				"employee_id", timesheet.EmployeeID,
				"project_id", timesheet.ProjectID)
			percentage = w.settingsConfig.GetWeeklyPaymentPercentage(ctx)
		} else {
			if assignment.PaymentSchedule == string(domain.PaymentScheduleMonthly) {
				percentage = w.settingsConfig.GetMonthlyPaymentPercentage(ctx)
			} else {
				percentage = w.settingsConfig.GetWeeklyPaymentPercentage(ctx)
			}
		}

		paidAmount := int64(float64(timesheet.Amount) * percentage)
		result[timesheet.ID] = paidAmount
	}

	return result, nil
}

// batchFetchAssignments fetches all project employee assignments for the given timesheets
func (w *BulkTransferPaymentWorker) batchFetchAssignments(ctx context.Context, timesheets []*domain.Timesheet) (map[string]*domain.ProjectEmployee, error) {
	if len(timesheets) == 0 {
		return make(map[string]*domain.ProjectEmployee), nil
	}

	type pair struct {
		ProjectID  uint
		EmployeeID uint
	}
	pairMap := make(map[pair]bool)
	for _, ts := range timesheets {
		pairMap[pair{ProjectID: ts.ProjectID, EmployeeID: ts.EmployeeID}] = true
	}

	projectIDs := make([]uint, 0, len(pairMap))
	employeeIDs := make([]uint, 0, len(pairMap))
	for p := range pairMap {
		projectIDs = append(projectIDs, p.ProjectID)
		employeeIDs = append(employeeIDs, p.EmployeeID)
	}

	assignments, err := w.projectEmployeeRepo.GetActiveAssignmentsByProjectsAndEmployees(ctx, projectIDs, employeeIDs)
	if err != nil {
		return nil, err
	}

	assignmentMap := make(map[string]*domain.ProjectEmployee)
	for _, assignment := range assignments {
		key := fmt.Sprintf("%d:%d", assignment.ProjectID, assignment.EmployeeID)
		assignmentMap[key] = assignment
	}

	return assignmentMap, nil
}

// updateRevenueReceivable updates the revenue_receivable field for paid timesheets
func (w *BulkTransferPaymentWorker) updateRevenueReceivable(
	ctx context.Context,
	paymentUpdates []domain.PaymentStatusUpdate,
	feePercentage float64,
) error {
	revenueUpdates := make(map[uint]int64)

	for _, update := range paymentUpdates {
		if update.PaidAmount == nil {
			continue
		}
		revenueReceivable := int64(float64(*update.PaidAmount) * (1 + feePercentage))
		revenueUpdates[update.TimesheetID] = revenueReceivable
	}

	if err := w.timesheetRepo.BulkUpdateRevenueReceivable(ctx, revenueUpdates); err != nil {
		return fmt.Errorf("failed to bulk update revenue_receivable: %w", err)
	}

	return nil
}
