package settlement

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"

	"api-server/internal/app/services/asset"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	auditctx "api-server/internal/pkg/context"

	"gorm.io/gorm"
)

// SettlementApplier applies a settlement for a single transaction synchronously:
// it locks the transaction, re-validates the remaining amount, creates the
// settlement record + double-entry ledger entries, and marks the linked timesheets
// revenue_paid=1 — all inside one DB transaction with deadlock retry.
//
// The settlement-upload flow calls this inline instead of publishing a
// fire-and-forget event, so a dropped event can no longer silently strand a
// receivable (the txn 99 / timesheet 11579 bug). Implemented by
// *events.SettlementEventHandler.
type SettlementApplier interface {
	ApplySettlement(ctx context.Context, transactionID uint, amount int64, fileID uint, filename string, timesheetIDs []uint) error
}

// SettlementUploadService orchestrates the settlement upload process
type SettlementUploadService struct {
	db                *gorm.DB
	excelParser       *SettlementExcelParser
	timesheetLinker   *SettlementTimesheetLinker
	assetService      *asset.AssetService
	ledgerRepo        domain.LedgerEntryRepository
	eventBus          domain.EventBus
	timesheetReader   domain.TimesheetReader
	advPayReqRepo     domain.AdvancePaymentRequestRepository
	advPayRepo        domain.AdvancePaymentRepository
	notificationRepo  domain.NotificationRepository
	settlementApplier SettlementApplier
}

// SetSettlementApplier wires the synchronous settlement applier. Called once during
// bootstrap after the event handler (which implements SettlementApplier) is
// constructed. Breaking the cycle this way avoids reordering service construction.
func (s *SettlementUploadService) SetSettlementApplier(applier SettlementApplier) {
	s.settlementApplier = applier
}

// NewSettlementUploadService creates a new settlement upload service
func NewSettlementUploadService(
	db *gorm.DB,
	timesheetRepo domain.TimesheetRepository,
	transactionRepo domain.TransactionRepository,
	assetService *asset.AssetService,
	ledgerRepo domain.LedgerEntryRepository,
	eventBus domain.EventBus,
	advPayReqRepo domain.AdvancePaymentRequestRepository,
	advPayRepo domain.AdvancePaymentRepository,
	notificationRepo domain.NotificationRepository,
) *SettlementUploadService {
	timesheetLinker := NewSettlementTimesheetLinker(db, timesheetRepo, transactionRepo)

	return &SettlementUploadService{
		db:               db,
		excelParser:      NewSettlementExcelParser(),
		timesheetLinker:  timesheetLinker,
		assetService:     assetService,
		ledgerRepo:       ledgerRepo,
		eventBus:         eventBus,
		timesheetReader:  timesheetRepo,
		advPayReqRepo:    advPayReqRepo,
		advPayRepo:       advPayRepo,
		notificationRepo: notificationRepo,
	}
}

// ProcessSettlementFile processes the uploaded settlement file.
// Detects file type from INTERNAL sheet header and routes accordingly:
// - type:advance_payment → settle advance payment requests
// - type:timesheet (or legacy) → settle timesheets
// Returns: validation result, total received amount, asset ID, filename, error
func (s *SettlementUploadService) ProcessSettlementFile(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	userID uint,
	timesheetRepo domain.TimesheetRepository,
) (*SettlementValidationResult, int64, uint, string, error) {
	// Parse Excel file
	fileData, err := s.excelParser.ParseSettlementFile(fileHeader)
	if err != nil {
		return nil, 0, 0, "", err
	}

	// INTERNAL sheet is mandatory
	if !fileData.HasInternalSheet {
		return nil, 0, 0, "", domain.NewValidationError(
			"File đối soát phải có sheet INTERNAL chứa danh sách IDs")
	}

	// Route based on file type
	if fileData.IsAdvancePayment() {
		return s.processAdvancePaymentSettlement(ctx, fileData, fileHeader, userID)
	}

	return s.processTimesheetSettlement(ctx, fileData, fileHeader, userID, timesheetRepo)
}

// processAdvancePaymentSettlement handles advance payment sao ke files
func (s *SettlementUploadService) processAdvancePaymentSettlement(
	ctx context.Context,
	fileData *SettlementFileData,
	fileHeader *multipart.FileHeader,
	userID uint,
) (*SettlementValidationResult, int64, uint, string, error) {
	requestIDs := make([]uint64, len(fileData.TimesheetIDs))
	for i, id := range fileData.TimesheetIDs {
		requestIDs[i] = uint64(id)
	}

	requests, err := s.advPayReqRepo.GetByIDs(ctx, requestIDs)
	if err != nil {
		return nil, 0, 0, "", domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	if len(requests) == 0 {
		return nil, 0, 0, "", domain.NewValidationError(
			"Không tìm thấy advance payment request nào tương ứng với IDs trong file")
	}

	// Upload file as asset
	assetRecord, err := s.uploadSettlementProof(ctx, fileHeader, userID)
	if err != nil {
		return nil, 0, 0, "", err
	}

	// Build result — map each request to its transaction for settlement
	transactionAllocations := make(map[uint]int64)
	var settledIDs []uint

	for _, req := range requests {
		settledIDs = append(settledIDs, req.ID)
		if req.SettlementTransactionID != nil {
			transactionAllocations[*req.SettlementTransactionID] += int64(req.RequestAmount)
		}
	}

	// Settle each transaction inline (synchronous) so settlement records,
	// transaction status updates, and ledger entries are created atomically.
	// Uses settlementApplier wired at bootstrap — NOT async fire-and-forget events
	// which silently swallow errors (txn 99 / timesheet 11579 class of bug).
	if len(transactionAllocations) > 0 {
		if s.settlementApplier == nil {
			return nil, 0, 0, "", domain.NewInternalError("Settlement applier not configured", nil)
		}
		for txnID, amount := range transactionAllocations {
			if err := s.settlementApplier.ApplySettlement(
				ctx,
				txnID,
				amount,
				assetRecord.ID,
				fileHeader.Filename,
				nil, // advance payments have no timesheet IDs to mark
			); err != nil {
				observability.GetLogger().Error("Failed to apply settlement for advance payment",
					"transaction_id", txnID,
					"amount", amount,
					"asset_id", assetRecord.ID,
					"error", err)
				return nil, 0, 0, "", domain.NewInternalError(constants.MsgCannotPublishSettlementEventVN, err)
			}
			observability.GetLogger().Info("Settlement applied for advance payment",
				"transaction_id", txnID,
				"amount", amount,
				"asset_id", assetRecord.ID)
		}
	}

	// Mark receivable settled only after all settlements succeed
	settledAt := clock.Now()
	if _, err := s.advPayReqRepo.MarkReceivableSettled(ctx, requestIDs, settledAt); err != nil {
		return nil, 0, 0, "", domain.NewInternalError("Không thể đánh dấu đã thanh toán", err)
	}

	result := &SettlementValidationResult{
		Transactions:      transactionAllocations,
		TimesheetIDs:      settledIDs,
		SettledInternally: true,
	}

	s.createSettlementNotification(ctx, userID, fileData.SettlementAmount, assetRecord.ID, settledIDs, settledAt, true)

	return result, fileData.SettlementAmount, assetRecord.ID, fileHeader.Filename, nil
}

// processTimesheetSettlement handles timesheet sao ke files (original flow)
func (s *SettlementUploadService) processTimesheetSettlement(
	ctx context.Context,
	fileData *SettlementFileData,
	fileHeader *multipart.FileHeader,
	userID uint,
	timesheetRepo domain.TimesheetRepository,
) (*SettlementValidationResult, int64, uint, string, error) {

	// Build InternalSheetRow from timesheets
	// Fetch timesheets to get their revenue_receivable amounts
	timesheets, err := timesheetRepo.GetByIDs(ctx, fileData.TimesheetIDs)
	if err != nil {
		return nil, 0, 0, "", domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	rows := make([]InternalSheetRow, 0, len(timesheets))
	for _, ts := range timesheets {
		rows = append(rows, InternalSheetRow{
			TimesheetID: ts.ID,
			Amount:      ts.RevenueReceivable,
		})
	}

	// Validate INTERNAL sheet using new validation method
	// Pass the E2 amount (money received from client) for validation
	result, err := s.timesheetLinker.ValidateInternalSheet(ctx, rows, fileData.SettlementAmount)
	if err != nil {
		return nil, 0, 0, "", err
	}

	// Upload file as asset
	asset, err := s.uploadSettlementProof(ctx, fileHeader, userID)
	if err != nil {
		return nil, 0, 0, "", err
	}

	return result, fileData.SettlementAmount, asset.ID, fileHeader.Filename, nil
}

// CreateSettlementNotification creates a notification record for the sao ke history dialog.
// Should be called after settlement events succeed to avoid orphaned records.
func (s *SettlementUploadService) CreateSettlementNotification(
	ctx context.Context,
	userID uint,
	totalAmount int64,
	assetID uint,
	ids []uint,
	isAdvancePayment bool,
) {
	s.createSettlementNotification(ctx, userID, totalAmount, assetID, ids, clock.Now(), isAdvancePayment)
}

// SettlementDedupResult holds the outcome of a dedup-aware settlement file processing.
type SettlementDedupResult struct {
	ProcessedTimesheets int
	SkippedTimesheets   int
	SkippedIDs          []uint
	SettlementsCreated  int
	SettlementAmount    int64
	AssetID             uint
}

// ProcessSettlementFileWithDedup processes a settlement file, skipping already-paid timesheets.
// It returns info about what was processed vs skipped, enabling idempotent re-uploads.
// Routes advance payment files to processAdvancePaymentSettlement.
func (s *SettlementUploadService) ProcessSettlementFileWithDedup(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	userID uint,
) (*SettlementDedupResult, error) {
	// Parse Excel file
	fileData, err := s.excelParser.ParseSettlementFile(fileHeader)
	if err != nil {
		return nil, err
	}

	if !fileData.HasInternalSheet {
		return nil, domain.NewValidationError(
			"File đối soát phải có sheet INTERNAL chứa danh sách IDs")
	}

	// Route advance payment files through the advance payment settlement flow
	if fileData.IsAdvancePayment() {
		result, totalAmount, assetID, _, err := s.processAdvancePaymentSettlement(ctx, fileData, fileHeader, userID)
		if err != nil {
			return nil, err
		}
		return &SettlementDedupResult{
			ProcessedTimesheets: len(result.TimesheetIDs),
			SettlementsCreated:  len(result.Transactions),
			SettlementAmount:    totalAmount,
			AssetID:             assetID,
		}, nil
	}

	// Build rows from all timesheet IDs (we'll let the linker filter out paid ones)
	timesheets, err := s.timesheetReader.GetByIDs(ctx, fileData.TimesheetIDs)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	rows := make([]InternalSheetRow, 0, len(timesheets))
	for _, ts := range timesheets {
		rows = append(rows, InternalSheetRow{
			TimesheetID: ts.ID,
			Amount:      ts.RevenueReceivable,
		})
	}

	// Validate with dedup: skip already-paid timesheets
	result, skippedIDs, err := s.timesheetLinker.ValidateInternalSheetWithDedup(ctx, rows, fileData.SettlementAmount)
	if err != nil {
		return nil, err
	}

	// All timesheets already paid
	if len(result.TimesheetIDs) == 0 {
		return &SettlementDedupResult{
			ProcessedTimesheets: 0,
			SkippedTimesheets:   len(skippedIDs),
			SkippedIDs:          skippedIDs,
			SettlementsCreated:  0,
			SettlementAmount:    0,
		}, nil
	}

	// Upload file as asset
	asset, err := s.uploadSettlementProof(ctx, fileHeader, userID)
	if err != nil {
		return nil, err
	}

	// Calculate effective settlement amount from unpaid timesheets only
	var effectiveAmount int64
	for _, amount := range result.Transactions {
		effectiveAmount += amount
	}

	// Emit settlement events
	if err := s.settleSynchronously(ctx, result, effectiveAmount, asset.ID, fileHeader.Filename); err != nil {
		return nil, err
	}

	return &SettlementDedupResult{
		ProcessedTimesheets: len(result.TimesheetIDs),
		SkippedTimesheets:   len(skippedIDs),
		SkippedIDs:          skippedIDs,
		SettlementsCreated:  len(result.Transactions),
		SettlementAmount:    effectiveAmount,
		AssetID:             asset.ID,
	}, nil
}

// createSettlementNotification creates a notification record for the sao ke history dialog.
func (s *SettlementUploadService) createSettlementNotification(
	ctx context.Context,
	userID uint,
	totalAmount int64,
	assetID uint,
	ids []uint,
	settledAt time.Time,
	isAdvancePayment bool,
) {
	var userIDPtr *uint
	if userID > 0 {
		userIDPtr = &userID
	}

	notifType := domain.NotificationTypePayrollReport
	title := "Đối soát sao kê"
	if isAdvancePayment {
		notifType = domain.NotificationTypeAdvancePaymentReport
		title = "Đối soát sao kê ứng lương"
	}

	metadata := domain.PayrollEmailMetadata{
		TotalAmount:  totalAmount,
		SaoKeAssetID: &assetID,
		SettledAt:    &settledAt,
	}
	if len(ids) > 0 {
		metadata.TimesheetIDs = ids
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		observability.GetLogger().Error("Failed to marshal settlement notification metadata", "error", err)
		return
	}
	metadataStr := string(metadataJSON)

	notification := &domain.Notification{
		SenderID:    userID,
		RecipientID: userIDPtr,
		Type:        notifType,
		Channel:     domain.NotificationChannelEmail,
		Title:       title,
		Message:     title,
		ContentType: domain.NotificationContentTypePlainText,
		Metadata:    &metadataStr,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		observability.GetLogger().Error("Failed to create settlement notification", "error", err)
	}
}

// uploadSettlementProof uploads the settlement file as an asset
func (s *SettlementUploadService) uploadSettlementProof(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	userID uint,
) (*domain.Asset, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgCannotOpenFileForUploadVN)
	}
	defer func() {
		_ = file.Close()
	}()

	asset, err := s.assetService.UploadAsset(ctx, file, fileHeader, domain.AssetUploadRequest{
		UploadType: "settlement_proof",
	}, userID)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgCannotUploadFileVN, err)
	}

	return asset, nil
}

// SettleFromMetadata processes settlement using pre-known timesheet IDs and total amount.
// This is used when settling from email history (Đã nhận button) without uploading a file.
// saoKeAssetID is the ID of the already-saved sao ke Excel file asset.
func (s *SettlementUploadService) SettleFromMetadata(ctx context.Context, timesheetIDs []uint, totalReceived int64, saoKeAssetID uint) error {
	if len(timesheetIDs) == 0 {
		return domain.NewValidationError(constants.MsgTimesheetListNotEmptyVN)
	}
	if totalReceived <= 0 {
		return domain.NewValidationError(constants.MsgReceiveAmountMustBePositiveVN)
	}
	if saoKeAssetID == 0 {
		return domain.NewValidationError(constants.MsgReconciledFileNotFoundVN)
	}

	// Fetch timesheets to get their transaction IDs and revenue amounts
	timesheets, err := s.timesheetReader.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		return domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}

	result := &SettlementValidationResult{
		TimesheetIDs:           timesheetIDs,
		Transactions:           make(map[uint]int64),
		TimesheetToTransaction: make(map[uint]uint),
	}

	for _, ts := range timesheets {
		if ts.TransactionID == nil {
			// A timesheet with no transaction can never be settled, and leaving it in
			// result.TimesheetIDs would trip verifyRevenuePaid AFTER some settlements
			// have already committed (partial-success-masked-as-error). Fail fast here,
			// before any DB write, naming the offending timesheet.
			return domain.NewValidationError(
				fmt.Sprintf("Bảng chấm công #%d chưa được liên kết với giao dịch nào, không thể đối soát. "+
					"Vui lòng kiểm tra lại hoặc gỡ bảng chấm công này khỏi danh sách.", ts.ID))
		}
		result.Transactions[*ts.TransactionID] += ts.RevenueReceivable
		result.TimesheetToTransaction[ts.ID] = *ts.TransactionID
	}

	return s.settleSynchronously(ctx, result, totalReceived, saoKeAssetID, "sao_ke_email")
}

// settleSynchronously settles every transaction in result inline via the
// SettlementApplier (one DB transaction per transaction, with deadlock retry),
// then verifies that every INTERNAL timesheet was actually flipped to revenue_paid=1.
//
// Unlike the previous fire-and-forget event publish, each settlement is applied
// synchronously and any failure is returned to the caller — a dropped settlement
// can no longer strand a receivable silently (the txn 99 / timesheet 11579 bug).
// After success, an informational SettlementAppliedFromUploadEvent is re-published
// per transaction for audit logging; ApplySettlement is idempotent (skips when the
// timesheets are already revenue_paid), so the re-publish cannot double-settle.
func (s *SettlementUploadService) settleSynchronously(
	ctx context.Context,
	result *SettlementValidationResult,
	totalReceived int64,
	assetID uint,
	filename string,
) error {
	logger := observability.GetLogger()

	// Validate assetID
	if assetID == 0 {
		return domain.NewValidationError(constants.MsgInvalidAssetIDVN)
	}

	if s.settlementApplier == nil {
		return domain.NewInternalError(constants.MsgCannotPublishSettlementEventVN,
			fmt.Errorf("settlement applier not configured"))
	}

	// Build per-transaction timesheet ID map so revenue_paid marking is atomic with settlement creation
	txnTimesheets := make(map[uint][]uint, len(result.Transactions))
	for tsID, txnID := range result.TimesheetToTransaction {
		txnTimesheets[txnID] = append(txnTimesheets[txnID], tsID)
	}

	// 1) Settle each transaction synchronously. A failure here is returned to the
	//    admin instead of being lost in an async worker. Settlements are applied
	//    per-transaction (each in its own DB tx with a row lock); if a later
	//    transaction fails, the earlier ones stay committed and the admin can
	//    re-upload — the dedup path skips the already-settled rows.
	settled := 0
	total := len(result.Transactions)
	for transactionID, amount := range result.Transactions {
		if err := s.settlementApplier.ApplySettlement(ctx, transactionID, amount, assetID, filename, txnTimesheets[transactionID]); err != nil {
			logger.Error("Synchronous settlement failed for transaction",
				"transaction_id", transactionID,
				"amount", amount,
				"asset_id", assetID,
				"settled_so_far", settled,
				"total", total,
				"error", err)
			return domain.NewInternalError(
				fmt.Sprintf("Đối soát thất bại tại giao dịch #%d (đã thanh toán %d/%d giao dịch). "+
					"Các giao dịch đã thanh toán được ghi nhận; vui lòng đối soát lại file để thanh toán phần còn lại.",
					transactionID, settled, total), err)
		}
		settled++
		logger.Info("Synchronous settlement applied",
			"transaction_id", transactionID,
			"amount", amount,
			"asset_id", assetID)
	}

	// 2) Double-check: every INTERNAL timesheet must now be revenue_paid=1. This is
	//    the guarantee that a settlement upload cannot silently leave a receivable
	//    behind (the txn 99 / 11579 class of bug).
	if err := s.verifyRevenuePaid(ctx, result.TimesheetIDs); err != nil {
		return err
	}

	// Calculate total allocated to transactions (for leftover/rounding below)
	var totalAllocated int64
	for _, amount := range result.Transactions {
		totalAllocated += amount
	}

	// 3) Handle leftover amount (rounding difference)
	leftover := totalReceived - totalAllocated
	if leftover > 0 {
		logger.Info("Leftover amount detected after settlement allocation",
			"leftover", leftover,
			"total_received", totalReceived,
			"total_allocated", totalAllocated,
			"tolerance", constants.RoundingTolerance)

		// This leftover represents cash received that wasn't allocated to any transaction
		// Record as miscellaneous revenue (cash on hand + revenue)
		// Get user ID from context
		var userID uint
		if userIDPtr := auditctx.GetUserID(ctx); userIDPtr != nil {
			userID = *userIDPtr
		}
		if userID == 0 {
			logger.Warn("User ID not found in context, using default for leftover ledger entry")
			userID = 1 // Default system user
		}

		// Create double-entry ledger entries
		entries := []*domain.LedgerEntry{
			// Debit: Cash on hand (asset increases)
			{
				Date:      clock.Now(),
				Account:   domain.AccountCash,
				Party:     "Miscellaneous Revenue (Rounding)",
				Debit:     leftover,
				Credit:    0,
				AssetID:   &assetID,
				CreatedBy: userID,
			},
			// Credit: Revenue (revenue increases)
			{
				Date:      clock.Now(),
				Account:   domain.AccountRevenue,
				Party:     "Miscellaneous Revenue (Rounding)",
				Debit:     0,
				Credit:    leftover,
				AssetID:   &assetID,
				CreatedBy: userID,
			},
		}

		// Create ledger entries as a transaction (double-entry)
		if err := s.ledgerRepo.CreateTransaction(ctx, entries); err != nil {
			logger.Error("Failed to create ledger entries for leftover amount",
				"leftover", leftover,
				"asset_id", assetID,
				"error", err)
			return domain.NewInternalError(constants.MsgCannotWriteFractionalLedgerVN, err)
		}

		logger.Info("Successfully recorded leftover as miscellaneous revenue",
			"amount", leftover,
			"asset_id", assetID,
			"cash_entry_id", entries[0].ID,
			"revenue_entry_id", entries[1].ID)
	}

	// 4) Informational audit events. The settlement is already applied above; the
	//    handler's idempotency guard makes these audit-only (no double settle).
	for transactionID, amount := range result.Transactions {
		event := domain.NewSettlementAppliedFromUploadEvent(
			ctx,
			transactionID,
			amount,
			assetID,
			filename,
			txnTimesheets[transactionID],
		)
		if err := s.eventBus.Publish(ctx, event); err != nil {
			logger.Warn("Failed to publish informational SettlementAppliedFromUploadEvent (settlement already applied)",
				"transaction_id", transactionID,
				"error", err)
		}
	}

	return nil
}

// verifyRevenuePaid re-fetches the given timesheet IDs and fails if any EXISTING
// timesheet did not reach revenue_paid=1 after settlement. This guarantees a
// settlement upload cannot silently leave a receivable unsettled (the txn 99 /
// timesheet 11579 bug). Timesheets that no longer exist (concurrently deleted) are
// not treated as failures — they cannot be settled and are only logged as a warning.
func (s *SettlementUploadService) verifyRevenuePaid(ctx context.Context, timesheetIDs []uint) error {
	if len(timesheetIDs) == 0 {
		return nil
	}
	recheck, err := s.timesheetReader.GetByIDs(ctx, timesheetIDs)
	if err != nil {
		return domain.NewInternalError(constants.MsgCannotGetTimesheetInfoVN, err)
	}
	present := make(map[uint]*domain.Timesheet, len(recheck))
	for _, ts := range recheck {
		present[ts.ID] = ts
	}
	var notUpdated []uint
	var missing []uint
	for _, id := range timesheetIDs {
		ts, ok := present[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if !ts.RevenuePaid {
			notUpdated = append(notUpdated, id)
		}
	}
	if len(missing) > 0 {
		observability.GetLogger().Warn("Timesheets vanished between settlement and verification (concurrently deleted?) — not treated as failure",
			"timesheet_ids", missing)
	}
	if len(notUpdated) > 0 {
		return domain.NewValidationError(
			fmt.Sprintf("Đối soát thất bại: %d timesheet không được đánh dấu thanh toán sau đối soát "+
				"(IDs: %v). Vui lòng kiểm tra lại giao dịch liên kết và đối soát lại.",
				len(notUpdated), notUpdated))
	}
	return nil
}
