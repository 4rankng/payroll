package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/excelkit"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListUploadedFiles lists all advance payment file history from assets table
func (h *AdvancePaymentHandler) ListUploadedFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	uploadTypes := []string{
		domain.UploadTypeFlexPayImport,
		domain.UploadTypeAdvancePaymentResult,
		domain.UploadTypeAdvancePaymentExport,
		domain.UploadTypeAdvancePaymentSaoKeExport,
		domain.UploadTypeAdvancePaymentSaoKeResult,
	}

	offset := (page - 1) * pageSize
	filters := domain.AssetFilters{
		UploadTypes: uploadTypes,
		Limit:       pageSize,
		Offset:      offset,
		SortBy:      "created_at",
		SortOrder:   "DESC",
	}

	totalRecords, err := h.service.GetConfig().AssetRepo.Count(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể đếm file")
		return
	}

	assets, err := h.service.GetConfig().AssetRepo.List(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách file")
		return
	}

	items := make([]dto.UploadedFileResponse, 0, len(assets))
	for _, asset := range assets {
		items = append(items, dto.UploadedFileResponse{
			ID:         asset.ID,
			Filename:   asset.Filename,
			UploadType: asset.UploadType,
			CreatedAt:  asset.CreatedAt,
			UploadedBy: asset.Uploader.Fullname,
		})
	}

	totalPages := int(totalRecords) / pageSize
	if int(totalRecords)%pageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, items, "Lấy danh sách file thành công",
		response.Pagination{Page: page, PageSize: pageSize, TotalPages: totalPages, TotalRecords: int(totalRecords)})
}

// GetImportJobStatus returns the status of an import job
func (h *AdvancePaymentHandler) GetImportJobStatus(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID job không hợp lệ")
		return
	}

	asset, err := h.assetRepo.GetByID(c.Request.Context(), uint(jobID))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, "Không tìm thấy job")
			return
		}
		response.InternalServerError(c, "Không thể lấy trạng thái job")
		return
	}

	// Verify the upload type
	if asset.UploadType != domain.UploadTypeFlexPayImport {
		response.NotFound(c, "Không tìm thấy job")
		return
	}

	// Read progress from Redis
	progress, err := h.importProgressSvc.GetProgress(c.Request.Context(), uint(jobID))
	if err != nil {
		// No progress found — job hasn't started or Redis key expired
		resp := dto.ImportJobStatusResponse{
			ID:        asset.ID,
			Status:    "pending",
			Filename:  asset.Filename,
			CreatedAt: asset.CreatedAt.Format(time.RFC3339),
		}
		response.Success(c, resp, "Lấy trạng thái job thành công")
		return
	}

	percentage := calculatePercentage(progress.TotalRows, progress.ProcessedRows)
	status := progress.Status
	if percentage == 100 {
		status = "completed"
	} else if status == "pending" && percentage > 0 {
		status = "processing"
	}

	resp := dto.ImportJobStatusResponse{
		ID:            asset.ID,
		Status:        status,
		ForMonth:      progress.ForMonth,
		Filename:      asset.Filename,
		TotalRows:     progress.TotalRows,
		ProcessedRows: progress.ProcessedRows,
		Percentage:    percentage,
		Error:         progress.Error,
		CreatedAt:     asset.CreatedAt.Format(time.RFC3339),
	}

	if !progress.StartedAt.IsZero() {
		startedAt := progress.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &startedAt
	}
	if status == "completed" && !progress.UpdatedAt.IsZero() {
		completedAt := progress.UpdatedAt.Format(time.RFC3339)
		resp.CompletedAt = &completedAt
	}

	// Parse result if available
	if progress.Result != nil && *progress.Result != "" {
		var importResult dto.ImportFlexPayFileResult
		if err := json.Unmarshal([]byte(*progress.Result), &importResult); err == nil {
			resp.Result = &importResult
		}
	}

	response.Success(c, resp, "Lấy trạng thái job thành công")
}

// ImportFlexPayFile imports a flexible payroll file asynchronously
func (h *AdvancePaymentHandler) ImportFlexPayFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	// The salary month is always DERIVED from the upload date — the admin no
	// longer selects it in the UI: an upload between day 20 of month M and
	// day 8 of M+1 belongs to M's salary period (GetCurrentMonth resolves the
	// advance-period month from the injected clock). A stale form value, if
	// still sent by an old client, is ignored.
	forMonth := advance_payment.GetCurrentMonth()
	if len(forMonth) != 7 {
		response.BadRequest(c, "Tháng (forMonth) không hợp lệ với định dạng YYYY-MM")
		return
	}

	forceReprocess, err := strconv.ParseBool(c.DefaultPostForm("force_reprocess", "false"))
	if err != nil {
		response.BadRequest(c, "force_reprocess không hợp lệ")
		return
	}

	fileContent, filename, ok := uploadguard.Receive(c, false)
	if !ok {
		return
	}

	// Validate Excel file before doing anything else
	xlsxFile, err := excelkit.OpenReader(bytes.NewReader(fileContent))
	if err != nil {
		response.BadRequest(c, "File Excel không hợp lệ")
		return
	}

	// Count rows for audit log
	totalRows := 0
	sheets := xlsxFile.GetSheetList()
	if len(sheets) > 0 {
		rows, _ := xlsxFile.GetRows(sheets[0])
		// Subtract header row
		if len(rows) > 0 {
			totalRows = len(rows) - 1
		}
	}
	_ = xlsxFile.Close()

	hash := sha256.Sum256(fileContent)
	checksum := fmt.Sprintf("%x", hash)

	// The default path reuses an existing upload. When an admin explicitly
	// requests reprocessing, create a new asset so the worker runs again; the
	// flexible import's get-or-create and BatchUpsert paths keep the employee,
	// assignment, and advance-limit records duplicate-free.
	if !forceReprocess {
		existingAsset, lookupErr := h.service.GetConfig().AssetRepo.GetByChecksum(c.Request.Context(), checksum, domain.UploadTypeFlexPayImport)
		if lookupErr == nil && existingAsset != nil {
			assetReady := h.ensureAssetFileExists(c, existingAsset, fileContent, filename)
			if !assetReady {
				return
			}

			if err := h.asynqClient.EnqueueImportJob(existingAsset.ID, forMonth); err != nil {
				logger := observability.GetLogger()
				logger.Error("failed to enqueue import job", "error", err, "asset_id", existingAsset.ID)
			}
			response.SuccessCreated(c, dto.ImportJobResponse{
				ID:        existingAsset.ID,
				Status:    "pending",
				Message:   "File đang được xử lý lại, các bản ghi đã tồn tại sẽ được bỏ qua",
				CreatedAt: existingAsset.CreatedAt.Format(time.RFC3339),
			}, "File đang được xử lý lại, các bản ghi đã tồn tại sẽ được bỏ qua")
			return
		}
		if lookupErr != nil && !domain.IsNotFoundError(lookupErr) {
			response.InternalServerError(c, "Không thể kiểm tra file")
			return
		}
	}

	// New file — store to disk and create asset
	storedFile, err := h.fileStorage.StoreBytes(fileContent, filename, domain.UploadTypeFlexPayImport)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("Không thể lưu file: %v", err))
		return
	}

	createdBy := userID.(uint)

	asset := &domain.Asset{
		Filename:   filename,
		FilePath:   storedFile.FilePath,
		UploadType: domain.UploadTypeFlexPayImport,
		Checksum:   &checksum,
		UploadedBy: createdBy,
	}

	createdAsset, err := h.assetRepo.Create(c.Request.Context(), asset)
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("failed to create asset for flex pay import", "error", err, "filename", filename)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Duplicate entry") || strings.Contains(errMsg, "1062") {
			response.Conflict(c, "File này đã được import trước đó")
			return
		}
		response.InternalServerError(c, "Không thể tạo record cho file")
		return
	}

	// Enqueue async processing using Asynq
	if err := h.asynqClient.EnqueueImportJob(createdAsset.ID, forMonth); err != nil {
		logger := observability.GetLogger()
		logger.Error("failed to enqueue import job", "error", err, "asset_id", createdAsset.ID)
	}

	// Emit audit event for file import (non-blocking)
	if h.auditService != nil {
		go func() {
			_ = h.auditService.LogFileImport(c.Request.Context(), "flexpay_employees", totalRows, filename)
		}()
	}

	resp := dto.ImportJobResponse{
		ID:        createdAsset.ID,
		Status:    "pending",
		Message:   "File đã được nhận và đang xử lý",
		CreatedAt: createdAsset.CreatedAt.Format(time.RFC3339),
	}

	response.SuccessCreated(c, resp, "File đã được nhận, đang xử lý")
}

// ensureAssetFileExists checks if the asset's file exists on disk.
// If missing, it stores a new copy and updates the asset's FilePath.
// Returns false if an error response was already sent.
func (h *AdvancePaymentHandler) ensureAssetFileExists(c *gin.Context, asset *domain.Asset, fileContent []byte, filename string) bool {
	if asset.FilePath != "" {
		fullPath := h.fileStorage.GetFilePath(asset.FilePath)
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}

	logger := observability.GetLogger()
	logger.Info("asset file missing on disk, saving new copy",
		"asset_id", asset.ID, "current_path", asset.FilePath)

	storedFile, err := h.fileStorage.StoreBytes(fileContent, filename, domain.UploadTypeFlexPayImport)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("Không thể lưu file: %v", err))
		return false
	}

	asset.FilePath = storedFile.FilePath
	if err := h.assetRepo.Update(c.Request.Context(), asset); err != nil {
		logger.Warn("failed to update asset file path after re-saving missing file", "error", err)
	}
	return true
}

// ExportReconciliation handles FlexPay reconciliation export
// GET /api/v1/advance-payments/reconciliation/export
func (h *AdvancePaymentHandler) ExportReconciliation(c *gin.Context) {
	var req dto.FlexPayReconciliationExportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestParametersVN)
		return
	}

	if _, err := time.Parse("2006-01", req.ForMonth); err != nil {
		response.BadRequest(c, constants.MsgInvalidMonthFormatVN)
		return
	}

	ctx := c.Request.Context()
	logger := observability.GetLogger()

	logger.Info("Starting FlexPay reconciliation export",
		"forMonth", req.ForMonth,
		"user_id", c.GetString("user_id"))

	// Get completed requests for the month
	reportData, err := h.flexPayReconciliationService.GetCompletedRequestsByMonth(ctx, req.ForMonth)
	if err != nil {
		logger.Error("Failed to get completed requests", "error", err)
		response.InternalServerError(c, constants.MsgFailedToGenerateReconciliationVN)
		return
	}

	if len(reportData) == 0 {
		logger.Info("No completed requests found for the month", "forMonth", req.ForMonth)
		response.Success(c, nil, "Không có dữ liệu thanh toán ứng lương cho tháng được chọn")
		return
	}

	// Parse the month for due date calculation
	parsedMonth, _ := time.Parse("2006-01", req.ForMonth)
	atDate := time.Date(parsedMonth.Year(), parsedMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Generate Excel file
	excelBytes, summary, err := h.flexPayReconciliationExporter.GenerateExcel(c.Request.Context(), reportData, atDate)
	if err != nil {
		logger.Error("Failed to generate Excel", "error", err)
		response.InternalServerError(c, constants.MsgFailedToGenerateReconciliationFileVN)
		return
	}

	// Cancel all pending requests for the month
	cancelledCount, _, err := h.flexPayReconciliationService.CancelAllPendingRequests(ctx, req.ForMonth)
	if err != nil {
		logger.Error("Failed to cancel pending requests", "error", err, "cancelled_count", cancelledCount)
		// Don't fail the export if cancellation fails, just log it
	}

	logger.Info("FlexPay reconciliation export completed",
		"forMonth", req.ForMonth,
		"totalProjects", len(reportData),
		"totalAmount", summary.TotalAmount,
		"cancelledCount", cancelledCount,
		"fileSize", len(excelBytes))

	// Generate filename
	filename := fmt.Sprintf("sao_ke_tt_%s.xlsx", req.ForMonth)

	saoKeAsset := h.saveSaoKeExportAssetWithResult(c, excelBytes, filename)

	// Create notification record for sao ke history dialog
	if saoKeAsset != nil {
		h.createSaoKeExportNotification(c.Request.Context(), saoKeAsset.ID, summary.TotalAmount, req.ForMonth)
	}

	// Emit audit event for file export (non-blocking)
	if h.auditService != nil {
		go func() {
			// Count total records across all projects
			totalRecords := 0
			for _, project := range reportData {
				totalRecords += len(project.EmployeeData)
			}
			_ = h.auditService.LogFileExport(c.Request.Context(), "reconciliation", totalRecords, filename)
		}()
	}

	// Set headers for file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", strconv.Itoa(len(excelBytes)))

	// Write the file
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// UploadAdvancePaymentResult uploads and processes a bank transfer result file
func (h *AdvancePaymentHandler) UploadAdvancePaymentResult(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File là bắt buộc")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể mở file")
		return
	}
	defer func() { _ = file.Close() }()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}

	hash := sha256.Sum256(fileContent)
	checksum := fmt.Sprintf("%x", hash)

	// Check for duplicate first - reuse existing asset if found
	existingAsset, err := h.service.GetConfig().AssetRepo.GetByChecksum(c.Request.Context(), checksum, domain.UploadTypeAdvancePaymentResult)
	if err == nil && existingAsset != nil {
		// File is a duplicate, check if the file exists on disk
		// Handle empty file_path - treat as missing file
		if existingAsset.FilePath != "" {
			fullPath := h.fileStorage.GetFilePath(existingAsset.FilePath)
			if _, err := os.Stat(fullPath); err == nil {
				// File exists on disk, reuse the existing asset
				xlsxFile, processErr := excelkit.OpenReader(bytes.NewReader(fileContent))
				if processErr != nil {
					response.BadRequest(c, "File Excel không hợp lệ")
					return
				}
				defer func() { _ = xlsxFile.Close() }()

				result, processErr := h.service.ProcessBankResult(c.Request.Context(), xlsxFile)
				if processErr != nil {
					response.InternalServerError(c, fmt.Sprintf("Xử lý file kết quả thất bại: %s", processErr.Error()))
					return
				}

				// Create bulk_transfer_files record for this result upload
				h.createBulkTransferFileRecord(c.Request.Context(), existingAsset, result, fileHeader.Filename)

				response.Success(c, map[string]interface{}{
					"items":         result.Data,
					"total_txn":     result.TotalTxn,
					"completed_txn": result.CompletedTxn,
					"failed_txn":    result.FailedTxn,
				}, "File kết quả đã được xử lý thành công")
				return
			}
		}
		// File doesn't exist on disk or file_path is empty, continue to save it
		logger := observability.GetLogger()
		logger.Info("duplicate file detected but file missing on disk or path is empty, will save new copy",
			"asset_id", existingAsset.ID, "current_path", existingAsset.FilePath)
	} else if err != nil && !domain.IsNotFoundError(err) {
		response.InternalServerError(c, "Không thể kiểm tra file")
		return
	}

	// Store the file to disk (only if we don't have a copy or file was missing)
	storedFile, err := h.fileStorage.StoreBytes(fileContent, fileHeader.Filename, domain.UploadTypeAdvancePaymentResult)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("Không thể lưu file: %v", err))
		return
	}

	// If we found an existing asset but file was missing (empty path or file not found), update the file path
	if existingAsset != nil {
		// Update the existing asset's file path
		existingAsset.FilePath = storedFile.FilePath
		if err := h.service.GetConfig().AssetRepo.Update(c.Request.Context(), existingAsset); err != nil {
			logger := observability.GetLogger()
			logger.Warn("failed to update asset file path after re-saving missing file", "error", err)
		}

		// Will create bulk_transfer_files record after processing below

		// Continue processing to return the full result to the frontend
		// Fall through to process the file below
	}

	xlsxFile, err := excelkit.OpenReader(bytes.NewReader(fileContent))
	if err != nil {
		response.BadRequest(c, "File Excel không hợp lệ")
		return
	}
	defer func() { _ = xlsxFile.Close() }()

	result, err := h.service.ProcessBankResult(c.Request.Context(), xlsxFile)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("Xử lý file kết quả thất bại: %s", err.Error()))
		return
	}

	userID, _ := c.Get("user_id")

	// Only create new asset if we didn't find an existing one
	// If existingAsset was found but file was missing, we already updated it above (lines 546-606)
	var asset *domain.Asset
	if existingAsset == nil {
		// Use the actual stored file path from FileStorage (file already saved above)
		newAsset := &domain.Asset{
			Filename:   fileHeader.Filename,
			FilePath:   storedFile.FilePath,
			UploadType: domain.UploadTypeAdvancePaymentResult,
			Checksum:   &checksum,
			UploadedBy: userID.(uint),
		}
		createdAsset, err := h.service.GetConfig().AssetRepo.Create(c.Request.Context(), newAsset)
		if err != nil {
			logger := observability.GetLogger()
			logger.Warn("failed to create asset record for bank result upload", "error", err)
			response.InternalServerError(c, "Không thể tạo record cho file")
			return
		}
		asset = createdAsset
	} else {
		// Reuse the existing asset that we already updated above
		asset = existingAsset
	}

	// Create bulk_transfer_files record for this result upload
	h.createBulkTransferFileRecord(c.Request.Context(), asset, result, fileHeader.Filename)

	// Emit audit event for file import (non-blocking)
	if h.auditService != nil {
		go func() {
			_ = h.auditService.LogFileImport(c.Request.Context(), "bulk_transfer_result", result.TotalTxn, fileHeader.Filename)
		}()
	}

	response.Success(c, map[string]interface{}{
		"items":         result.Data,
		"total_txn":     result.TotalTxn,
		"completed_txn": result.CompletedTxn,
		"failed_txn":    result.FailedTxn,
	}, "Xử lý file kết quả thành công")
}

// createBulkTransferFileRecord creates a bulk_transfer_files record for result upload history
func (h *AdvancePaymentHandler) createBulkTransferFileRecord(ctx context.Context, asset *domain.Asset, result *dto.AdvancePaymentUploadResultResponse, filename string) {
	logger := observability.GetLogger()

	dataJSON, err := json.Marshal(result.Data)
	if err != nil {
		logger.Error("failed to marshal result data for bulk_transfer_file record", "error", err)
		return
	}

	var transferAmount int64
	for _, item := range result.Data {
		if item.PaymentStatus == "paid" {
			amount, _ := strconv.ParseInt(item.Amount, 10, 64)
			transferAmount += amount
		}
	}

	file := &domain.BulkTransferFile{
		Filename:          filename,
		Cycle:             domain.StringPtr("flexible"),
		CreatedBy:         asset.UploadedBy,
		TransactionsCount: result.TotalTxn,
		CompletedCount:    result.CompletedTxn,
		FailedCount:       result.FailedTxn,
		TransferAmount:    transferAmount,
		Data:              string(dataJSON),
		AssetID:           &asset.ID,
	}

	if err := h.service.GetConfig().BulkTransferFileRepo.Create(ctx, file); err != nil {
		logger.Error("failed to create bulk_transfer_file record", "error", err, "asset_id", asset.ID)
	}
}

// ExportAdvancePayments exports pending requests to Excel
func (h *AdvancePaymentHandler) ExportAdvancePayments(c *gin.Context) {
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("2006-01-02", fromDateStr)
		if err != nil {
			response.BadRequest(c, "Định dạng ngày không hợp lệ")
			return
		}
	} else {
		now := h.clock.Now()
		fromDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	}

	if toDateStr != "" {
		toDate, err = time.Parse("2006-01-02", toDateStr)
		if err != nil {
			response.BadRequest(c, "Định dạng ngày không hợp lệ")
			return
		}
	} else {
		now := h.clock.Now()
		toDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	file, err := h.service.ExportToExcel(c.Request.Context(), fromDate, toDate, userID.(uint))
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}
	defer func() { _ = file.Close() }()

	buf, err := file.WriteToBuffer()
	if err != nil {
		response.InternalServerError(c, "Không thể tạo file xuất")
		return
	}

	filename := fmt.Sprintf("ung-luong-thang-%s.xlsx", h.clock.Now().Format("2006-01-02"))

	// Save exported file to file history
	asset := h.saveExportAsset(c, buf.Bytes(), filename)

	// Emit audit event for file export (non-blocking)
	if h.auditService != nil && asset != nil {
		go func() {
			// Get request count from service or estimate from file size
			requestCount := int(buf.Len() / 1024) // Rough estimate in KB
			_ = h.auditService.LogFileExport(c.Request.Context(), "advance_payments", requestCount, filename)
		}()
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_, _ = buf.WriteTo(c.Writer)
}

func (h *AdvancePaymentHandler) saveExportAsset(c *gin.Context, data []byte, filename string) *domain.Asset {
	logger := observability.GetLogger()

	userID, exists := c.Get("user_id")
	if !exists {
		logger.Warn("no user_id in context, skipping asset creation for export")
		return nil
	}

	storedFile, err := h.fileStorage.StoreBytes(data, filename, domain.UploadTypeAdvancePaymentExport)
	if err != nil {
		logger.Error("failed to store export file", "error", err)
		return nil
	}

	asset := &domain.Asset{
		Filename:   filename,
		FilePath:   storedFile.FilePath,
		UploadType: domain.UploadTypeAdvancePaymentExport,
		UploadedBy: userID.(uint),
	}
	createdAsset, err := h.assetRepo.Create(c.Request.Context(), asset)
	if err != nil {
		logger.Error("failed to create asset record for export", "error", err)
		return nil
	}

	return createdAsset
}

func (h *AdvancePaymentHandler) saveSaoKeExportAssetWithResult(c *gin.Context, data []byte, filename string) *domain.Asset {
	logger := observability.GetLogger()

	userID, exists := c.Get("user_id")
	if !exists {
		logger.Warn("no user_id in context, skipping asset creation for sao ke export")
		return nil
	}

	storedFile, err := h.fileStorage.StoreBytes(data, filename, domain.UploadTypeAdvancePaymentSaoKeExport)
	if err != nil {
		logger.Error("failed to store sao ke export file", "error", err)
		return nil
	}

	asset := &domain.Asset{
		Filename:   filename,
		FilePath:   storedFile.FilePath,
		UploadType: domain.UploadTypeAdvancePaymentSaoKeExport,
		UploadedBy: userID.(uint),
	}
	createdAsset, err := h.assetRepo.Create(c.Request.Context(), asset)
	if err != nil {
		logger.Error("failed to create asset record for sao ke export", "error", err)
		return nil
	}

	return createdAsset
}

func (h *AdvancePaymentHandler) createSaoKeExportNotification(ctx context.Context, assetID uint, totalAmount int64, forMonth string) {
	logger := observability.GetLogger()

	metadata := domain.PayrollEmailMetadata{
		TotalAmount:  totalAmount,
		SaoKeAssetID: &assetID,
		ReportAtDate: forMonth,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		logger.Error("Failed to marshal sao ke export notification metadata", "error", err)
		return
	}
	metadataStr := string(metadataJSON)

	title := fmt.Sprintf("Sao kê ứng lương - %s", forMonth)
	notification := &domain.Notification{
		Type:        domain.NotificationTypeAdvancePaymentReport,
		Channel:     domain.NotificationChannelEmail,
		Title:       title,
		Message:     title,
		ContentType: domain.NotificationContentTypePlainText,
		Metadata:    &metadataStr,
	}

	if err := h.notificationRepo.Create(ctx, notification); err != nil {
		logger.Error("Failed to create sao ke export notification", "error", err)
	}
}

// DownloadUploadedFile downloads an uploaded file by asset ID
func (h *AdvancePaymentHandler) DownloadUploadedFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID không hợp lệ")
		return
	}

	asset, err := h.assetRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, "Không tìm thấy file")
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin file")
		return
	}

	// Only allow downloading advance-payment-related files
	allowedTypes := map[string]bool{
		domain.UploadTypeFlexPayImport:             true,
		domain.UploadTypeAdvancePaymentResult:      true,
		domain.UploadTypeAdvancePaymentExport:      true,
		domain.UploadTypeAdvancePaymentSaoKeExport: true,
		domain.UploadTypeAdvancePaymentSaoKeResult: true,
	}
	if !allowedTypes[asset.UploadType] {
		response.Forbidden(c, "Không có quyền tải file này")
		return
	}

	filePath := h.fileStorage.GetFilePath(asset.FilePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		response.NotFound(c, "File không tồn tại trên hệ thống")
		return
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}

	safeFilename := url.QueryEscape(strings.ReplaceAll(asset.Filename, " ", "_"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", safeFilename))
	c.Header("Content-Length", strconv.Itoa(len(fileData)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileData)
}

// DownloadTransferHistoryFile downloads the original uploaded Excel file for a transfer history
func (h *AdvancePaymentHandler) DownloadTransferHistoryFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID không hợp lệ")
		return
	}

	logger := observability.GetLogger()
	logger.Info("downloading transfer history file", "id", id, "user_id", c.GetString("user_id"))

	// Get transfer history with Asset preload
	file, err := h.service.GetConfig().BulkTransferFileRepo.GetByIDWithAsset(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, "Không tìm thấy lịch sử chuyển tiền")
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin lịch sử chuyển tiền")
		return
	}

	var asset *domain.Asset

	// Check if this transfer history has an associated file
	if file.AssetID == nil || file.Asset == nil || file.Asset.FilePath == "" {
		response.NotFound(c, "Không tìm thấy file đính kèm")
		return
	}
	asset = file.Asset

	// Get full file path from file storage (asset.FilePath is relative)
	filePath := h.fileStorage.GetFilePath(asset.FilePath)

	// Check if file exists on disk
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Error("file not found on disk", "id", id, "file_path", filePath)
		response.NotFound(c, "File không tồn tại trên hệ thống")
		return
	}

	// Open the file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("failed to read file", "id", id, "file_path", filePath, "error", err)
		response.InternalServerError(c, "Không thể đọc file")
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	safeFilename := url.QueryEscape(strings.ReplaceAll(asset.Filename, " ", "_"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", safeFilename))
	c.Header("Content-Length", strconv.Itoa(len(fileData)))

	// Write the file
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileData)

	logger.Info("transfer history file downloaded", "id", id, "filename", asset.Filename, "size", len(fileData))
}

// ImportFlexibleEmployeeList imports an Excel file containing employees under flexible payment schedule
func (h *AdvancePaymentHandler) ImportFlexibleEmployeeList(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File là bắt buộc")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể mở file")
		return
	}
	defer func() { _ = file.Close() }()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}

	// Validate Excel file
	xlsxFile, err := excelkit.OpenReader(bytes.NewReader(fileContent))
	if err != nil {
		response.BadRequest(c, "File Excel không hợp lệ")
		return
	}
	defer func() { _ = xlsxFile.Close() }()

	createdBy := userID.(uint)

	result, err := h.service.ImportFlexibleEmployeeList(c.Request.Context(), xlsxFile, createdBy)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("Xử lý file thất bại: %s", err.Error()))
		return
	}

	response.Success(c, result, "Nhập danh sách nhân viên thành công")
}
