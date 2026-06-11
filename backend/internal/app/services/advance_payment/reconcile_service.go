package advance_payment

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/timeutil"

	"gorm.io/gorm"
)

// ReconcileService handles advance payment reconciliation (Sao ke)
type ReconcileService struct {
	logger                    *slog.Logger
	db                        *gorm.DB
	advancePaymentRepo        domain.AdvancePaymentRepository
	advancePaymentRequestRepo domain.AdvancePaymentRequestRepository
	projectEmployeeRepo       domain.ProjectEmployeeRepository
	settingsConfig            *config.SettingsConfigService
	idempotencyService        *infrastructure.IdempotencyService
	eventBus                  domain.EventBus
	notificationService       *notification.NotificationService
	employeeNotifier          notification.EmployeeNotifier
	bulkTransferFileRepo      domain.BulkTransferFileRepository
	transactionRepo           domain.TransactionRepository
	ledgerRepo                domain.LedgerEntryRepository
}

// NewReconcileService creates a new reconcile service for advance payments
func NewReconcileService(
	db *gorm.DB,
	advancePaymentRepo domain.AdvancePaymentRepository,
	advancePaymentRequestRepo domain.AdvancePaymentRequestRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	settingsConfig *config.SettingsConfigService,
	idempotencyService *infrastructure.IdempotencyService,
	eventBus domain.EventBus,
	notificationService *notification.NotificationService,
	employeeNotifier notification.EmployeeNotifier,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	transactionRepo domain.TransactionRepository,
	ledgerRepo domain.LedgerEntryRepository,
) *ReconcileService {
	return &ReconcileService{
		logger:                    observability.GetLogger().With("component", "AdvancePaymentReconcileService"),
		db:                        db,
		advancePaymentRepo:        advancePaymentRepo,
		advancePaymentRequestRepo: advancePaymentRequestRepo,
		projectEmployeeRepo:       projectEmployeeRepo,
		settingsConfig:            settingsConfig,
		idempotencyService:        idempotencyService,
		eventBus:                  eventBus,
		notificationService:       notificationService,
		employeeNotifier:          employeeNotifier,
		bulkTransferFileRepo:      bulkTransferFileRepo,
		transactionRepo:           transactionRepo,
		ledgerRepo:                ledgerRepo,
	}
}

// ReconcileAdvancePaymentsRequest represents the request to reconcile advance payments
type ReconcileAdvancePaymentsRequest struct {
	ForMonth     string  `json:"for_month" validate:"required,datetime=2006-01"`
	ProjectIDs   []uint  `json:"project_ids"`
	EmployeeIDs  []uint  `json:"employee_ids"`
	CustomDate   *string `json:"custom_date"`
	CustomDateTo *string `json:"custom_date_to"`
	// ForceExportAll - if true, export ALL paid requests including from previous months
	ForceExportAll bool `json:"force_export_all"`
}

// ReconcileAdvancePaymentsResponse represents the response from reconciliation
type ReconcileAdvancePaymentsResponse struct {
	Success             bool                            `json:"success"`
	Message             string                          `json:"message"`
	ExportedRequests    int64                           `json:"exported_requests"`
	CancelledRequests   int64                           `json:"cancelled_requests"`
	ExportFile          *dto.ExportBulkTransferResponse `json:"export_file,omitempty"`
	CancelledRequestIDs []uint64                        `json:"cancelled_request_ids"`
}

// ExportSaoKe handles the sao ke reconciliation process for advance payments
func (s *ReconcileService) ExportSaoKe(ctx context.Context, req *ReconcileAdvancePaymentsRequest, userID uint) (*ReconcileAdvancePaymentsResponse, error) {
	startTime := clock.Now()

	// Validate request
	if err := s.validateReconcileRequest(req); err != nil {
		return nil, err
	}

	// Calculate date range
	fromDate, toDate, err := s.calculateDateRange(req, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate date range: %w", err)
	}

	s.logger.Info("Starting sao ke reconciliation for advance payments",
		"for_month", req.ForMonth,
		"from_date", fromDate.Format(timeutil.DateFormat),
		"to_date", toDate.Format(timeutil.DateFormat),
		"project_ids", req.ProjectIDs,
		"employee_ids", req.EmployeeIDs,
		"force_export_all", req.ForceExportAll)

	// Use idempotency key to prevent duplicate processing
	idempotencyKey := fmt.Sprintf("reconcile_sao_ke:%s:%d:%d", req.ForMonth,
		toInt64(req.ProjectIDs), toInt64(req.EmployeeIDs))

	acquired, state, err := s.idempotencyService.TryAcquireLock(ctx, idempotencyKey)
	if err != nil {
		s.logger.Error("Failed to acquire idempotency lock", "key", idempotencyKey, "error", err)
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		if state == infrastructure.StateCompleted {
			// Return previous result if already completed
			prevResult, err := s.getPreviousReconcileResult(ctx, idempotencyKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get previous result: %w", err)
			}
			return prevResult, nil
		}
		// Return error if already processing
		return nil, fmt.Errorf("reconciliation already in progress")
	}

	// Process reconciliation in transaction
	var result *ReconcileAdvancePaymentsResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error

		// Step 1: Cancel pending advance payment requests
		cancelledCount, cancelledIDs, err := s.cancelPendingAdvanceRequests(ctx, tx, req, userID)
		if err != nil {
			return fmt.Errorf("failed to cancel pending requests: %w", err)
		}

		// Step 2: Get paid requests for export
		paidRequests, err := s.getPaidRequestsForExport(ctx, tx, req, userID)
		if err != nil {
			return fmt.Errorf("failed to get paid requests: %w", err)
		}

		// Step 3: Export reconciliation file
		exportResponse, err := s.exportReconciliationFile(ctx, paidRequests, req, userID)
		if err != nil {
			return fmt.Errorf("failed to export reconciliation file: %w", err)
		}

		// Step 4: Store result
		result = &ReconcileAdvancePaymentsResponse{
			Success:             true,
			Message:             "Reconciliation completed successfully",
			ExportedRequests:    int64(len(paidRequests)),
			CancelledRequests:   cancelledCount,
			ExportFile:          exportResponse,
			CancelledRequestIDs: cancelledIDs,
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to process reconciliation", "error", err)
		// Release lock on error
		if releaseErr := s.idempotencyService.ReleaseLock(ctx, idempotencyKey); releaseErr != nil {
			s.logger.Error("Failed to release idempotency lock", "key", idempotencyKey, "error", releaseErr)
		}
		return nil, err
	}

	// Mark as completed
	if err := s.idempotencyService.MarkCompleted(ctx, idempotencyKey); err != nil {
		s.logger.Error("Failed to mark idempotency key as completed", "key", idempotencyKey, "error", err)
	}

	s.logger.Info("Successfully completed sao ke reconciliation",
		"duration", time.Since(startTime),
		"exported_requests", result.ExportedRequests,
		"cancelled_requests", result.CancelledRequests)

	return result, nil
}

// validateReconcileRequest validates the reconciliation request
func (s *ReconcileService) validateReconcileRequest(req *ReconcileAdvancePaymentsRequest) error {
	if req.ForMonth == "" {
		return domain.NewValidationError(constants.MsgMonthRequiredVN)
	}

	// Validate month format
	if _, err := time.Parse("01-2006", req.ForMonth); err != nil {
		return domain.NewValidationError(constants.MsgInvalidMonthFormatYYYYMMVN)
	}

	return nil
}

// calculateDateRange calculates the date range for reconciliation
func (s *ReconcileService) calculateDateRange(req *ReconcileAdvancePaymentsRequest, userID uint) (time.Time, time.Time, error) {
	var (
		fromDate time.Time
		toDate   time.Time
		err      error
	)

	if req.CustomDate != nil {
		fromDate, err = time.Parse(timeutil.DateFormat, *req.CustomDate)
		if err != nil {
			return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgInvalidDateFormatGenericVN)
		}

		if req.CustomDateTo != nil {
			toDate, err = time.Parse(timeutil.DateFormat, *req.CustomDateTo)
			if err != nil {
				return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgInvalidDateFormatGenericVN)
			}
		} else {
			// Default to end of month if custom_date_to not provided
			toDate = endOfMonth(fromDate)
		}
	} else {
		// Use for_month as default
		parsedTime, err := time.Parse("01-2006", req.ForMonth)
		if err != nil {
			return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgInvalidMonthFormatVN)
		}

		fromDate = parsedTime
		toDate = endOfMonth(parsedTime)
	}

	return fromDate, toDate, nil
}

// cancelPendingAdvanceRequests cancels all pending advance payment requests
func (s *ReconcileService) cancelPendingAdvanceRequests(ctx context.Context, tx *gorm.DB, req *ReconcileAdvancePaymentsRequest, userID uint) (int64, []uint64, error) {
	var (
		cancelledCount int64 = 0
		cancelledIDs   []uint64
	)

	// Build filter for pending requests
	filters := domain.AdvancePaymentRequestFilters{
		// Include PENDING status
		Status: &[]string{string(domain.AdvancePaymentStatusPending)}[0],
	}

	if len(req.ProjectIDs) > 0 {
		projectID := uint64(req.ProjectIDs[0])
		filters.ProjectID = &projectID
	}

	if len(req.EmployeeIDs) > 0 {
		employeeID := uint64(req.EmployeeIDs[0])
		filters.EmployeeID = &employeeID
	}

	// Get pending requests
	pendingRequests, _, err := s.advancePaymentRequestRepo.List(ctx, filters)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get pending requests: %w", err)
	}

	// Cancel each pending request
	for _, request := range pendingRequests {
		// Check if request is within date range
		if !req.ForceExportAll && request.CreatedAt.Before(clock.Now().AddDate(0, -1, 0)) {
			// Skip requests older than 1 month unless force_export_all is true
			continue
		}

		if err := s.advancePaymentRequestRepo.Cancel(ctx, uint64(request.ID)); err != nil {
			s.logger.Error("Failed to cancel request", "request_id", request.ID, "error", err)
			continue
		}

		cancelledCount++
		cancelledIDs = append(cancelledIDs, uint64(request.ID))

		// Notify employee about cancelled request
		if s.employeeNotifier != nil {
			s.employeeNotifier.NotifyAdvancePaymentStatusChanged(ctx, request.EmployeeID, domain.AdvancePaymentStatusCancelled, request.RequestAmount)
		}
	}

	return cancelledCount, cancelledIDs, nil
}

// getPaidRequestsForExport gets paid requests that need to be included in reconciliation
func (s *ReconcileService) getPaidRequestsForExport(ctx context.Context, tx *gorm.DB, req *ReconcileAdvancePaymentsRequest, userID uint) ([]*domain.AdvancePaymentRequest, error) {
	var paidRequests []*domain.AdvancePaymentRequest

	// Build filter for paid requests
	filters := domain.AdvancePaymentRequestFilters{
		// Include COMPLETED and FAILED requests
		Status: &[]string{string(domain.AdvancePaymentStatusCompleted), string(domain.AdvancePaymentStatusFailed)}[0],
	}

	if len(req.ProjectIDs) > 0 {
		projectID := uint64(req.ProjectIDs[0])
		filters.ProjectID = &projectID
	}

	if len(req.EmployeeIDs) > 0 {
		employeeID := uint64(req.EmployeeIDs[0])
		filters.EmployeeID = &employeeID
	}

	// Get paid requests
	allPaidRequests, _, err := s.advancePaymentRequestRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get paid requests: %w", err)
	}

	// Filter by date range
	for _, request := range allPaidRequests {
		if request.PaidAt == nil {
			continue // Skip requests without payment date
		}

		// Check if paid date is within the specified range
		if request.PaidAt.After(clock.Now().AddDate(0, 0, -3*30)) { // Last 3 months by default
			paidRequests = append(paidRequests, request)
		}
	}

	return paidRequests, nil
}

// exportReconciliationFile creates the reconciliation export file
func (s *ReconcileService) exportReconciliationFile(ctx context.Context, paidRequests []*domain.AdvancePaymentRequest, req *ReconcileAdvancePaymentsRequest, userID uint) (*dto.ExportBulkTransferResponse, error) {
	// Calculate date range
	fromDate, toDate, err := s.calculateDateRange(req, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate date range: %w", err)
	}

	// Group paid requests by employee and project for bank transfer
	groupedData, err := s.groupPaidRequestsForExport(ctx, paidRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to group requests: %w", err)
	}

	// Create export request
	exportReq := &dto.ExportBulkTransferRequest{
		ForMonth:    req.ForMonth,
		ProjectIDs:  req.ProjectIDs,
		EmployeeIDs: req.EmployeeIDs,
	}

	// Generate reconciliation file
	response, err := s.generateReconciliationExcel(ctx, groupedData, exportReq, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reconciliation file: %w", err)
	}

	return response, nil
}

// groupPaidRequestsForExport groups paid requests by employee and project
func (s *ReconcileService) groupPaidRequestsForExport(ctx context.Context, paidRequests []*domain.AdvancePaymentRequest) ([]*domain.EmployeePendingRequests, error) {
	// Group by employee and project
	grouped := make(map[string]*domain.EmployeePendingRequests)

	for _, request := range paidRequests {
		key := fmt.Sprintf("%d:%d", request.EmployeeID, request.ProjectID)

		if _, exists := grouped[key]; !exists {
			// Get employee info
			employee, err := s.advancePaymentRepo.GetEmployeeByID(ctx, uint64(request.EmployeeID))
			if err != nil {
				s.logger.Error("Failed to get employee", "employee_id", request.EmployeeID, "error", err)
				continue
			}

			// Get project assignment
			projectAssignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, request.ProjectID, request.EmployeeID)
			if err != nil {
				s.logger.Error("Failed to get project assignment", "employee_id", request.EmployeeID, "project_id", request.ProjectID, "error", err)
				continue
			}

			// Get bank name from employee's bank relationship
			bankName := ""
			if employee.Bank != nil {
				bankName = employee.Bank.BranchName
			}

			grouped[key] = &domain.EmployeePendingRequests{
				EmployeeID:    uint64(request.EmployeeID),
				EmployeeName:  employee.Fullname,
				EmployeeCCCD:  employee.CCCD,
				ProjectID:     uint64(request.ProjectID),
				ProjectCode:   projectAssignment.Project.Code,
				AccountNumber: employee.BankAccountNumber,
				AccountName:   employee.BankAccountName,
				BankName:      bankName,
				RequestIDs:    make([]uint64, 0),
			}
		}

		// Update totals and add request ID
		grouped[key].TotalAmount += request.RequestAmount
		grouped[key].TotalFee += request.Fee
		grouped[key].NetAmount += request.NetAmount
		grouped[key].RequestIDs = append(grouped[key].RequestIDs, uint64(request.ID))
	}

	// Convert slice
	var result []*domain.EmployeePendingRequests
	for _, data := range grouped {
		result = append(result, data)
	}

	return result, nil
}

// generateReconciliationExcel generates the Excel file for reconciliation
func (s *ReconcileService) generateReconciliationExcel(ctx context.Context, groupedData []*domain.EmployeePendingRequests, req *dto.ExportBulkTransferRequest, fromDate, toDate time.Time) (*dto.ExportBulkTransferResponse, error) {
	// Create a new bulk transfer file record
	now := clock.Now()
	filename := fmt.Sprintf("MBank_reconcile_flexible_%s.xlsx", now.Format("20060102_150405"))

	// Convert to bulk transfer data format
	bulkTransferData := make([]dto.BulkTransferFileData, 0, len(groupedData))
	for i, employeeData := range groupedData {
		// Convert RequestIDs from []uint64 to []uint
		advPayReqIDs := make([]uint, len(employeeData.RequestIDs))
		for j, id := range employeeData.RequestIDs {
			advPayReqIDs[j] = uint(id)
		}

		bulkTransferData = append(bulkTransferData, dto.BulkTransferFileData{
			STT:           i + 1,
			EmployeeID:    uint(employeeData.EmployeeID),
			ProjectID:     uint(employeeData.ProjectID),
			AdvPayReqIDs:  advPayReqIDs,
			AccountNumber: employeeData.AccountNumber,
			AccountName:   employeeData.AccountName,
			BankName:      employeeData.BankName,
			Amount:        int64(employeeData.NetAmount),
		})
	}

	// Save bulk transfer file record
	// Convert bulkTransferData to JSON for storage
	dataBytes, err := json.Marshal(bulkTransferData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bulk transfer data: %w", err)
	}

	bulkFile := &domain.BulkTransferFile{
		Filename:          filename,
		Cycle:             domain.StringPtr(string(domain.PaymentScheduleFlexible)),
		FromDate:          &fromDate, // Need to pass these from calling function
		ToDate:            &toDate,   // Need to pass these from calling function
		TransactionsCount: len(bulkTransferData),
		TransferAmount:    calculateTransferAmount(bulkTransferData),
		Data:              string(dataBytes),
		CreatedBy:         0, // Will be set by handler
	}

	if err := s.bulkTransferFileRepo.Create(ctx, bulkFile); err != nil {
		return nil, fmt.Errorf("failed to save bulk transfer file: %w", err)
	}

	// Generate Excel content
	// This would typically use the existing excel service to generate the file
	// For now, return a simple response
	response := &dto.ExportBulkTransferResponse{
		Data: nil, // Actual Excel file bytes would be here
		Files: []dto.BulkTransferFileInfo{
			{
				BankID:        0, // Would be populated based on bank
				BankPrefix:    "MBANK",
				Filename:      filename,
				EmployeeCount: len(groupedData),
			},
		},
		SkippedEmployees: nil,
		TotalEmployees:   len(groupedData),
		IncludedCount:    len(groupedData),
		SkippedCount:     0,
		FromDate:         fromDate.Format("2006-01-02"),
		ToDate:           toDate.Format("2006-01-02"),
		Cycle:            "flexible",
	}

	return response, nil
}

// getPreviousReconcileResult retrieves previous reconciliation result
func (s *ReconcileService) getPreviousReconcileResult(ctx context.Context, idempotencyKey string) (*ReconcileAdvancePaymentsResponse, error) {
	// This would typically retrieve the result from the idempotency service
	// For now, return a placeholder response
	return &ReconcileAdvancePaymentsResponse{
		Success:           true,
		Message:           "Reconciliation already completed",
		ExportedRequests:  0,
		CancelledRequests: 0,
	}, nil
}

// Helper functions
func endOfMonth(t time.Time) time.Time {
	return t.AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func toInt64(ids []uint) int64 {
	var result int64
	for _, id := range ids {
		result += int64(id)
	}
	return result
}

func calculateTransferAmount(data []dto.BulkTransferFileData) int64 {
	var total int64
	for _, item := range data {
		total += item.Amount
	}
	return total
}
