package bulktransfer

import (
	"api-server/internal/pkg/clock"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"

	"gorm.io/gorm"
)

// ResultProcessor handles processing of bulk transfer result files
// Synchronous: Parse Excel file, VFIC lookup, duplicate check, upload asset, in-memory merge, publish event
// Asynchronous: Payment updates and transaction creation handled by asynq workers with automatic retries
type ResultProcessor struct {
	// Core dependencies
	db                  *gorm.DB
	fileRepo            BulkTransferFileRepository
	transactionCodeRepo TransactionCodeRepository
	assetRepo           domain.AssetRepository
	assetService        AssetService
	excelConverter      ExcelConverter
	eventBus            domain.EventBus
	asynqClient         BulkTransferEnqueuer

	// Strategy-based components for parsing
	strategyFactory *ResultStrategyFactory
	dataUpdater     *DataUpdater
	employeeRepo    EmployeeRepository
	rowParser       *RowParser
}

// NewResultProcessor creates a new ResultProcessor instance
func NewResultProcessor(
	db *gorm.DB,
	fileRepo BulkTransferFileRepository,
	transactionCodeRepo TransactionCodeRepository,
	employeeRepo EmployeeRepository,
	assetService AssetService,
	assetRepo domain.AssetRepository,
	excelConverter ExcelConverter,
	eventBus domain.EventBus,
	asynqClient BulkTransferEnqueuer,
) *ResultProcessor {
	rowParser := NewRowParser()
	strategyFactory := NewResultStrategyFactory(rowParser)
	dataUpdater := NewDataUpdater()

	return &ResultProcessor{
		db:                  db,
		fileRepo:            fileRepo,
		transactionCodeRepo: transactionCodeRepo,
		assetRepo:           assetRepo,
		assetService:        assetService,
		excelConverter:      excelConverter,
		eventBus:            eventBus,
		asynqClient:         asynqClient,
		strategyFactory:     strategyFactory,
		dataUpdater:         dataUpdater,
		employeeRepo:        employeeRepo,
		rowParser:           rowParser,
	}
}

// ProcessBulkTransferResultWithChecksum processes bank result files
// Synchronous: Parse Excel file, VFIC lookup, duplicate check, upload asset, in-memory merge, publish event
// Asynchronous: Payment updates and transaction creation handled by asynq workers (bulk_transfer:transaction and bulk_transfer:payment)
func (p *ResultProcessor) ProcessBulkTransferResultWithChecksum(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	processedBy uint,
) (*dto.BulkTransferResultResponse, error) {
	logger := observability.GetLogger()

	var response *dto.BulkTransferResultResponse

	// Create transaction context for outbox service (must survive outside transaction closure)
	txCtx := &domain.TransactionContext{
		IsTransactional: true,
	}

	// Execute all operations within a database transaction
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx.TX = tx
		newCtx := domain.WithTransactionContext(ctx, txCtx)

		// Step 1: Parse Excel file once and cache rows in memory (SYNCHRONOUS)
		rows, err := p.getExcelRows(fileHeader)
		if err != nil {
			logger.Error("Failed to get Excel rows", "error", err, "file", fileHeader.Filename)
			return fmt.Errorf("failed to get Excel rows: %w", err)
		}

		// Step 2: Detect file format and parse results using cached rows (SYNCHRONOUS)
		parsedResults, strategy, err := p.parseResultFileFromRows(newCtx, rows)
		if err != nil {
			logger.Error("Failed to parse bulk transfer result file", "error", err, "strategy", fmt.Sprintf("%T", strategy))
			return fmt.Errorf("failed to parse result file: %w", err)
		}
		logger.Info("Parsed results", "count", len(parsedResults), "strategy", strategy.Name())

		// Step 3: VFIC lookup - find matching entries by transaction codes (SYNCHRONOUS)
		transactionCodes := p.extractTransactionCodes(parsedResults)
		matchedEntries, detectedCycle, err := p.findDataByTransactionCodes(newCtx, transactionCodes)
		if err != nil {
			logger.Error("Failed to find matching bulk transfer entries", "error", err)
			return err
		}
		logger.Info("Found matching bulk transfer entries", "count", len(matchedEntries))

		// Step 4: Duplicate check - SHA-256 of file content, check existing assets (SYNCHRONOUS)
		existingAsset, err := p.checkDuplicateUpload(newCtx, fileHeader)
		if err != nil {
			logger.Error("Duplicate upload check failed", "error", err)
			return err
		}

		if existingAsset != nil {
			// Duplicate file: update history data and enqueue workers for self-healing.
			// Workers are idempotent — if transaction/ledger already exist, they short-circuit.
			logger.Info("Duplicate file detected, updating history and enqueuing workers", "asset_id", existingAsset.ID)

			updatedData, stats, totalAmount, err := p.mergeResultsWithData(newCtx, matchedEntries, parsedResults, &existingAsset.CreatedAt)
			if err != nil {
				return fmt.Errorf("failed to merge results with data: %w", err)
			}

			dataJSON, err := json.Marshal(updatedData)
			if err != nil {
				return fmt.Errorf("failed to marshal updated data: %w", err)
			}

			existingRecord, err := p.fileRepo.GetByAssetID(newCtx, existingAsset.ID)
			if err != nil {
				return fmt.Errorf("failed to find history record for asset %d: %w", existingAsset.ID, err)
			}

			if err := p.fileRepo.UpdateWithLock(newCtx, existingRecord.ID, map[string]interface{}{
				"data": string(dataJSON),
			}); err != nil {
				return fmt.Errorf("failed to update history data: %w", err)
			}

			logger.Info("History data updated for duplicate upload", "record_id", existingRecord.ID)

			// Publish event and enqueue workers so missing transactions/ledgers are created.
			// If the work was already done, workers are idempotent and will skip.
			parsedDataJSON, err := json.Marshal(parsedResults)
			if err != nil {
				return fmt.Errorf("failed to marshal parsed results: %w", err)
			}

			updatedDataJSON, err := json.Marshal(updatedData)
			if err != nil {
				return fmt.Errorf("failed to marshal updated data: %w", err)
			}

			event := domain.NewBulkTransferResultParsedEvent(
				newCtx,
				existingRecord.ID,
				existingAsset.ID,
				fileHeader.Filename,
				stats.TotalTransactions,
				stats.CompletedCount,
				stats.FailedCount,
				totalAmount,
				string(parsedDataJSON),
				string(updatedDataJSON),
				processedBy,
			)

			if err := p.eventBus.Publish(newCtx, event); err != nil {
				logger.Error("Failed to publish BulkTransferResultParsed event for duplicate", "error", err)
				return fmt.Errorf("failed to publish bulk transfer event: %w", err)
			}

			taskPayload := BulkTransferTaskPayload{
				EventID:         event.EventID,
				AssetID:         event.AssetID,
				Filename:        event.Filename,
				BulkFileID:      event.BulkFileID,
				TotalTransfers:  event.TotalTransfers,
				CompletedCount:  event.CompletedCount,
				FailedCount:     event.FailedCount,
				TotalAmount:     event.TotalAmount,
				ParsedDataJSON:  event.ParsedDataJSON,
				UpdatedDataJSON: event.UpdatedDataJSON,
				ProcessedBy:     event.ProcessedBy,
				ActorUserID:     event.ActorUserID,
			}
			asynqCli := p.asynqClient
			domain.RegisterAfterCommit(newCtx, func() {
				if err := asynqCli.EnqueueBulkTransferTransaction(taskPayload); err != nil {
					logger.Error("Failed to enqueue bulk_transfer:transaction task for duplicate",
						"event_id", taskPayload.EventID, "error", err)
				}
				if err := asynqCli.EnqueueBulkTransferPayment(taskPayload); err != nil {
					logger.Error("Failed to enqueue bulk_transfer:payment task for duplicate",
						"event_id", taskPayload.EventID, "error", err)
				}
			})

			logger.Info("Queued asynq tasks for duplicate upload (self-healing)",
				"event_id", event.EventID, "asset_id", existingAsset.ID)

			response = p.buildResponseFromResults(newCtx, parsedResults, updatedData, stats)
			dupAmount := p.calculateTotalTransferAmount(updatedData)
			p.publishImportAudit(newCtx, processedBy, existingAsset.ID, fileHeader.Filename, stats, dupAmount, true)
			return nil
		}

		// Step 5: Upload result file as asset (SYNCHRONOUS)
		asset, err := p.uploadResultAsset(newCtx, fileHeader, processedBy)
		if err != nil {
			logger.Error("Failed to upload bulk transfer result asset", "error", err, "filename", fileHeader.Filename)
			return fmt.Errorf("failed to upload result asset: %w", err)
		}
		logger.Info("Asset uploaded", "asset_id", asset.ID)

		// Step 6: In-memory merge of result statuses into matched data (SYNCHRONOUS)
		updatedData, stats, totalAmount, err := p.mergeResultsWithData(newCtx, matchedEntries, parsedResults, &asset.CreatedAt)
		if err != nil {
			logger.Error("Failed to merge results with data", "error", err)
			return fmt.Errorf("failed to merge results with data: %w", err)
		}
		logger.Info("Merged results with data", "matched", stats.MatchedCount)

		// Step 7: Marshal parsed results and updated data to JSON
		parsedDataJSON, err := json.Marshal(parsedResults)
		if err != nil {
			logger.Error("Failed to marshal parsed results", "error", err)
			return fmt.Errorf("failed to marshal parsed results: %w", err)
		}

		updatedDataJSON, err := json.Marshal(updatedData)
		if err != nil {
			logger.Error("Failed to marshal updated bulk transfer data", "error", err)
			return fmt.Errorf("failed to marshal updated data: %w", err)
		}

		// Step 8: Publish event to in-memory bus (for audit, cache, settlement global subscribers)
		// and register after-commit callbacks to enqueue asynq tasks for the two bulk-transfer workers.
		// BulkFileID=0 decouples result processing from single bulk_transfer_file record.
		event := domain.NewBulkTransferResultParsedEvent(
			newCtx,
			0, // BulkFileID=0 - decoupled from single bulk_transfer_file
			asset.ID,
			fileHeader.Filename,
			stats.TotalTransactions,
			stats.CompletedCount,
			stats.FailedCount,
			totalAmount,
			string(parsedDataJSON),
			string(updatedDataJSON),
			processedBy,
		)

		// Publish to in-memory bus so AuditEventHandler, CacheInvalidationHandler,
		// and SettlementEventHandler still see this event.
		if err := p.eventBus.Publish(newCtx, event); err != nil {
			logger.Error("Failed to publish BulkTransferResultParsed event", "error", err)
			return fmt.Errorf("failed to publish bulk transfer event: %w", err)
		}

		// Enqueue asynq tasks after the transaction commits. Using after-commit
		// ensures we never enqueue work for a rolled-back transaction.
		taskPayload := BulkTransferTaskPayload{
			EventID:         event.EventID,
			AssetID:         event.AssetID,
			Filename:        event.Filename,
			BulkFileID:      event.BulkFileID,
			TotalTransfers:  event.TotalTransfers,
			CompletedCount:  event.CompletedCount,
			FailedCount:     event.FailedCount,
			TotalAmount:     event.TotalAmount,
			ParsedDataJSON:  event.ParsedDataJSON,
			UpdatedDataJSON: event.UpdatedDataJSON,
			ProcessedBy:     event.ProcessedBy,
			ActorUserID:     event.ActorUserID,
		}
		asynqCli := p.asynqClient
		domain.RegisterAfterCommit(newCtx, func() {
			if err := asynqCli.EnqueueBulkTransferTransaction(taskPayload); err != nil {
				logger.Error("Failed to enqueue bulk_transfer:transaction task",
					"event_id", taskPayload.EventID, "error", err)
			}
			if err := asynqCli.EnqueueBulkTransferPayment(taskPayload); err != nil {
				logger.Error("Failed to enqueue bulk_transfer:payment task",
					"event_id", taskPayload.EventID, "error", err)
			}
		})

		logger.Info("Queued asynq tasks for bulk transfer result processing",
			"event_id", event.EventID)

		// Step 9: Build response for immediate feedback to user
		response = p.buildResponseFromResults(newCtx, parsedResults, updatedData, stats)

		// Step 10: Store result metadata in bulk_transfer_files for history tracking
		btfID, err := p.storeResultMetadata(newCtx, asset, updatedData, stats, totalAmount, processedBy, detectedCycle)
		if err != nil {
			logger.Error("Failed to store result metadata", "error", err, "asset_id", asset.ID)
			return fmt.Errorf("failed to store result metadata: %w", err)
		}

		// Step 11: Update transaction codes with the new bulk_transfer_file ID
		if len(transactionCodes) > 0 {
			if err := p.transactionCodeRepo.UpdateFileIDByCodes(newCtx, transactionCodes, btfID); err != nil {
				logger.Error("Failed to update transaction code file IDs", "error", err, "btf_id", btfID)
				return fmt.Errorf("failed to update transaction code file IDs: %w", err)
			}
		}

		logger.Info("Bulk transfer result parsed successfully (async processing queued)",
			"total", response.TotalTxn,
			"completed", response.CompletedTxn,
			"failed", response.FailedTxn)

		// Top-level IMPORT audit emit. The downstream BulkTransferResultParsed
		// event still produces its own BULK_CREATE/transaction audit row when
		// workers process it; this one is the explicit IMPORT/bulk_transfer_file
		// row admins filter on.
		p.publishImportAudit(newCtx, processedBy, asset.ID, fileHeader.Filename, stats, totalAmount, false)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Fire after-commit callbacks
	txCtx.RunAfterCommitCallbacks()

	return response, nil
}

// parseResultFileFromRows parses already-loaded Excel rows and detects appropriate strategy
func (p *ResultProcessor) parseResultFileFromRows(ctx context.Context, rows [][]string) ([]*ParsedResultRow, ResultProcessingStrategy, error) {
	strategy, err := p.strategyFactory.DetectStrategy(rows)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect file format: %w", err)
	}

	parsedResults, err := strategy.ParseRows(ctx, rows)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse rows: %w", err)
	}

	return parsedResults, strategy, nil
}

// getExcelRows extracts rows from Excel file
func (p *ResultProcessor) getExcelRows(fileHeader *multipart.FileHeader) ([][]string, error) {
	excelFile, err := p.excelConverter.ProcessExcelFile(fileHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to process Excel file: %w", err)
	}
	defer func() {
		if err := excelFile.Data.Close(); err != nil {
			// Log the error but don't return it as it's not critical
			observability.GetLogger().Warn("error closing Excel file", "error", err)
		}
	}()

	sheets := p.excelConverter.GetSheetNames(excelFile)
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := p.excelConverter.GetRowsFromSheet(excelFile, sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet: %w", err)
	}

	return rows, nil
}

// extractTransactionCodes extracts transaction codes from parsed results
func (p *ResultProcessor) extractTransactionCodes(parsedResults []*ParsedResultRow) []string {
	codes := make([]string, 0, len(parsedResults))
	for _, result := range parsedResults {
		if result.TransactionCode != "" {
			codes = append(codes, result.TransactionCode)
		}
	}
	return codes
}

// findDataByTransactionCodes looks up bulk transfer file entries by transaction codes (VFIC codes)
func (p *ResultProcessor) findDataByTransactionCodes(ctx context.Context, transactionCodes []string) ([]domain.BulkTransferFileDataEntry, string, error) {
	if len(transactionCodes) == 0 {
		return nil, "", fmt.Errorf("no transaction codes found in result file")
	}

	matchedEntries, cycle, err := p.fileRepo.FindDataByTransactionCodes(ctx, transactionCodes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find data by transaction codes: %w", err)
	}

	return matchedEntries, cycle, nil
}

// checkDuplicateUpload checks if this exact file has been uploaded before.
// Returns (existingAsset, nil) if duplicate found, (nil, nil) if new file, (nil, err) on error.
func (p *ResultProcessor) checkDuplicateUpload(ctx context.Context, fileHeader *multipart.FileHeader) (*domain.Asset, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file for checksum calculation: %w", err)
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate file checksum: %w", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	existingAsset, err := p.assetRepo.GetByChecksum(ctx, checksum, domain.UploadTypeBulkTransferResult)
	if err == nil && existingAsset != nil {
		return existingAsset, nil
	}

	if !domain.IsNotFoundError(err) && err != nil {
		return nil, fmt.Errorf("failed to check for duplicate asset: %w", err)
	}

	return nil, nil
}

// mergeResultsWithData merges parsed result statuses with matched bulk transfer data in memory
// Reuses DataUpdater pattern for the merge logic but doesn't write to database
func (p *ResultProcessor) mergeResultsWithData(
	ctx context.Context,
	matchedEntries []domain.BulkTransferFileDataEntry,
	parsedResults []*ParsedResultRow,
	transferAt *time.Time,
) ([]dto.BulkTransferFileData, *UpdateStats, int64, error) {
	logger := observability.GetLogger()

	// Build lookup from transaction_code -> ParsedResultRow so we can copy bank fields
	// (AccountNumber, AccountName, BankName, STT are already parsed from the result Excel)
	parsedByCode := make(map[string]*ParsedResultRow, len(parsedResults))
	for _, r := range parsedResults {
		if r.TransactionCode != "" {
			parsedByCode[r.TransactionCode] = r
		}
	}

	// Build original data slice from matched entries
	originalData := make([]dto.BulkTransferFileData, 0, len(matchedEntries))
	entryByCode := make(map[string]*dto.BulkTransferFileData)

	for _, entry := range matchedEntries {
		parsed := parsedByCode[entry.TransactionCode]
		accountNumber, accountName, bankName, stt := "", "", "", 0
		if parsed != nil {
			accountNumber = parsed.AccountNumber
			accountName = parsed.AccountName
			bankName = parsed.BankName
			stt = parsed.STT
		}
		data := dto.BulkTransferFileData{
			STT:             stt,
			EmployeeID:      entry.EmployeeID,
			TransactionCode: entry.TransactionCode,
			Amount:          entry.Amount,
			TimesheetIDs:    entry.TimesheetIDs,
			AccountNumber:   accountNumber,
			AccountName:     accountName,
			BankName:        bankName,
			// Initialize with empty/default values
			TransferStatus: "",
			BankTxnRef:     "",
			ErrorMessage:   "",
		}
		originalData = append(originalData, data)
		entryByCode[entry.TransactionCode] = &originalData[len(originalData)-1]
	}

	// Build lookup map from parsed results
	resultMap := make(map[string]*ParsedResultRow)
	for _, result := range parsedResults {
		if result.TransactionCode != "" {
			resultMap[result.TransactionCode] = result
		}
	}

	// Statistics
	stats := &UpdateStats{
		TotalTransactions: len(originalData),
	}

	// Merge results by matching transaction_code
	for i := range originalData {
		txCode := originalData[i].TransactionCode

		// Look up matching result
		result, found := resultMap[txCode]
		if !found {
			logger.Warn("Transaction code from bulk data not found in result file",
				"transaction_code", txCode,
				"employee_id", originalData[i].EmployeeID)
			stats.UnmatchedCount++
			continue
		}

		stats.MatchedCount++

		// Update status fields based on result
		originalData[i].TransferStatus = result.Status

		if result.Status == ResultStatusCompleted {
			originalData[i].BankTxnRef = result.BankTxnRef
			originalData[i].ErrorMessage = "" // Clear any previous error
			stats.CompletedCount++
		} else {
			originalData[i].ErrorMessage = result.ErrorMessage
			originalData[i].BankTxnRef = "" // Clear any previous bank ref
			stats.FailedCount++
		}
	}

	// Calculate total transfer amount
	totalAmount := p.calculateTotalTransferAmount(originalData)

	logger.Info("Data merge completed",
		"total", stats.TotalTransactions,
		"matched", stats.MatchedCount,
		"completed", stats.CompletedCount,
		"failed", stats.FailedCount,
		"unmatched", stats.UnmatchedCount)

	return originalData, stats, totalAmount, nil
}

// uploadResultAsset uploads the result file as an asset
func (p *ResultProcessor) uploadResultAsset(ctx context.Context, fileHeader *multipart.FileHeader, processedBy uint) (*domain.Asset, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log the error but don't return it as it's not critical
			observability.GetLogger().Warn("error closing file", "error", err)
		}
	}()

	asset, err := p.assetService.UploadAsset(ctx, file, fileHeader, domain.AssetUploadRequest{
		UploadType: domain.UploadTypeBulkTransferResult,
	}, processedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset: %w", err)
	}

	return asset, nil
}

// calculateTotalTransferAmount sums up completed transaction amounts
func (p *ResultProcessor) calculateTotalTransferAmount(data []dto.BulkTransferFileData) int64 {
	var total int64
	for _, item := range data {
		if item.TransferStatus == ResultStatusCompleted {
			total += item.Amount
		}
	}
	return total
}

// buildResponseFromResults constructs the API response
func (p *ResultProcessor) buildResponseFromResults(
	ctx context.Context,
	parsedResults []*ParsedResultRow,
	updatedData []dto.BulkTransferFileData,
	stats *UpdateStats,
) *dto.BulkTransferResultResponse {
	// Build transaction code to employee ID mapping
	txCodeToEmployeeID := p.buildTransactionCodeMap(updatedData)

	// Fetch employee CCCD data in bulk
	employeeMap := p.fetchEmployeeCCCDMap(ctx, txCodeToEmployeeID)

	// Build response items
	items := make([]dto.BulkTransferResultItem, 0, len(parsedResults))
	processedAt := clock.Now()

	for _, result := range parsedResults {
		item := p.buildResponseItem(result, txCodeToEmployeeID, employeeMap, processedAt)
		items = append(items, item)
	}

	return &dto.BulkTransferResultResponse{
		Data:         items,
		TotalTxn:     stats.TotalTransactions,
		CompletedTxn: stats.CompletedCount,
		FailedTxn:    stats.FailedCount,
	}
}

// storeResultMetadata stores result summary and detail data as a bulk_transfer_file record
// for later retrieval by the bulk transfer history endpoints
func (p *ResultProcessor) storeResultMetadata(ctx context.Context, asset *domain.Asset, updatedData []dto.BulkTransferFileData, stats *UpdateStats, totalAmount int64, processedBy uint, cycle string) (uint, error) {
	dataJSON, err := json.Marshal(updatedData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal result detail data: %w", err)
	}

	var cyclePtr *string
	if cycle != "" {
		cyclePtr = domain.StringPtr(cycle)
	}

	now := clock.Now()
	file := &domain.BulkTransferFile{
		Filename:          asset.Filename,
		Cycle:             cyclePtr,
		CreatedBy:         processedBy,
		TransactionsCount: stats.TotalTransactions,
		CompletedCount:    stats.CompletedCount,
		FailedCount:       stats.FailedCount,
		TransferAmount:    totalAmount,
		Data:              string(dataJSON),
		AssetID:           &asset.ID,
		Status:            "uploaded",
		UploadedAt:        &now,
	}

	if err := p.fileRepo.Create(ctx, file); err != nil {
		return 0, err
	}

	return file.ID, nil
}

// buildTransactionCodeMap creates mapping from transaction code to employee ID
func (p *ResultProcessor) buildTransactionCodeMap(data []dto.BulkTransferFileData) map[string]uint {
	txCodeToEmployeeID := make(map[string]uint)
	for _, item := range data {
		if item.TransactionCode != "" {
			txCodeToEmployeeID[item.TransactionCode] = item.EmployeeID
		}
	}
	return txCodeToEmployeeID
}

// fetchEmployeeCCCDMap fetches employee CCCD data in bulk
func (p *ResultProcessor) fetchEmployeeCCCDMap(ctx context.Context, txCodeToEmployeeID map[string]uint) map[uint]string {
	logger := observability.GetLogger()

	// Collect unique employee IDs
	employeeIDSet := make(map[uint]bool)
	for _, empID := range txCodeToEmployeeID {
		employeeIDSet[empID] = true
	}

	if len(employeeIDSet) == 0 {
		return make(map[uint]string)
	}

	// Build ID slice for bulk query
	ids := make([]int64, 0, len(employeeIDSet))
	for empID := range employeeIDSet {
		ids = append(ids, int64(empID))
	}

	// Single bulk query instead of N individual queries
	employees, err := p.employeeRepo.GetByIDs(ctx, ids)
	if err != nil {
		logger.Error("Failed to fetch employees in bulk", "error", err)
		return make(map[uint]string)
	}

	employeeMap := make(map[uint]string, len(employees))
	for _, emp := range employees {
		employeeMap[emp.ID] = emp.CCCD
	}

	return employeeMap
}

// buildResponseItem constructs a single response item
func (p *ResultProcessor) buildResponseItem(
	result *ParsedResultRow,
	txCodeToEmployeeID map[string]uint,
	employeeMap map[uint]string,
	processedAt time.Time,
) dto.BulkTransferResultItem {
	paymentStatus := PaymentStatusFailed
	if result.Status == "completed" {
		paymentStatus = PaymentStatusPaid
	}

	var paidAt *string
	if paymentStatus == PaymentStatusPaid {
		paidAtStr := processedAt.Format(time.RFC3339)
		paidAt = &paidAtStr
	}

	// Look up employee CCCD
	cccd := ""
	if empID, found := txCodeToEmployeeID[result.TransactionCode]; found {
		if empCCCD, exists := employeeMap[empID]; exists {
			cccd = empCCCD
		}
	}

	return dto.BulkTransferResultItem{
		Row:                   result.STT,
		EmployeeBank:          result.BankName,
		EmployeeAccountNumber: result.AccountNumber,
		EmployeeName:          result.AccountName,
		EmployeeCCCD:          cccd,
		Amount:                fmt.Sprintf("%.0f", result.Amount),
		PaymentStatus:         paymentStatus,
		PaidAt:                paidAt,
	}
}

// publishImportAudit emits the top-level IMPORT audit event for a bulk-transfer
// result file. Best-effort — failures are logged but do not fail the import
// (the data has already been committed by this point).
func (p *ResultProcessor) publishImportAudit(
	ctx context.Context,
	processedBy uint,
	assetID uint,
	filename string,
	stats *UpdateStats,
	totalAmount int64,
	isDuplicate bool,
) {
	if p.eventBus == nil {
		return
	}
	if stats == nil {
		stats = &UpdateStats{}
	}
	event := domain.NewBulkTransferResultImportedEvent(
		ctx,
		processedBy,
		assetID,
		filename,
		stats.TotalTransactions,
		stats.CompletedCount,
		stats.FailedCount,
		totalAmount,
		isDuplicate,
	)
	if err := p.eventBus.Publish(ctx, event); err != nil {
		observability.GetLogger().Warn("Failed to publish BulkTransferResultImported audit event",
			"asset_id", assetID,
			"filename", filename,
			"error", err)
	}
}
