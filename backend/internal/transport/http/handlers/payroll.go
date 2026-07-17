package handlers

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/storage"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/payroll/bulktransfer"
)

type PayrollHandler struct {
	payrollService       *payroll.PayrollService
	autoBulkTransferSvc  bulktransfer.AutoBulkTransferService
	cacheService         domain.CacheServiceUseCase
	bulkTransferFileRepo domain.BulkTransferFileRepository
	assetRepo            domain.AssetRepository
	fileStorage          storage.FileStorage
	eventBus             domain.EventBus
	logger               *slog.Logger
}

func NewPayrollHandler(
	payrollService *payroll.PayrollService,
	autoBulkTransferSvc bulktransfer.AutoBulkTransferService,
	cacheService domain.CacheServiceUseCase,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	assetRepo domain.AssetRepository,
	fileStorage storage.FileStorage,
	eventBus domain.EventBus,
) *PayrollHandler {
	return &PayrollHandler{
		payrollService:       payrollService,
		autoBulkTransferSvc:  autoBulkTransferSvc,
		cacheService:         cacheService,
		bulkTransferFileRepo: bulkTransferFileRepo,
		assetRepo:            assetRepo,
		fileStorage:          fileStorage,
		eventBus:             eventBus,
		logger:               slog.Default(),
	}
}

func validateExportBulkTransferRequest(req *dto.ExportBulkTransferRequest) error {
	hasMonth := strings.TrimSpace(req.ForMonth) != ""
	hasRange := strings.TrimSpace(req.FromDate) != "" || strings.TrimSpace(req.ToDate) != ""

	if hasMonth && hasRange {
		return fmt.Errorf("không thể gửi đồng thời for_month và from_date/to_date")
	}

	if hasMonth {
		return nil
	}

	if strings.TrimSpace(req.FromDate) == "" || strings.TrimSpace(req.ToDate) == "" {
		return fmt.Errorf("from_date và to_date là bắt buộc khi không sử dụng for_month")
	}

	return nil
}

// GetPayrollTemplate serves the bank bulk transfer template file
func (h *PayrollHandler) GetPayrollTemplate(c *gin.Context) {
	templateBytes, err := h.payrollService.GetBulkTransferTemplate(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetTemplateFileVN)
		return
	}

	// Set headers for XLSX file download (MBank template)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=MBank-bulk-transfer-template.xlsx")
	c.Header("Content-Length", strconv.Itoa(len(templateBytes)))

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", templateBytes)
}

// ExportBulkTransfer exports timesheets data to bank bulk transfer Excel format
func (h *PayrollHandler) ExportBulkTransfer(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		response.InternalServerError(c, constants.MsgInvalidUserIDVN)
		return
	}

	var req dto.ExportBulkTransferRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	if err := validateExportBulkTransferRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.CreatedBy = userIDUint
	exportResponse, err := h.payrollService.ExportBulkTransfer(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("ExportBulkTransfer error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	contentType := exportResponse.ContentType
	if contentType == "" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	fileExtension := exportResponse.FileExtension
	if fileExtension == "" {
		fileExtension = ".xlsx"
	}

	// Use the filename that was already generated and saved to database.
	downloadFilename := fmt.Sprintf("%s%s", exportResponse.Filename, fileExtension)
	contentDisposition := mime.FormatMediaType("attachment", map[string]string{"filename": downloadFilename})

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition)
	c.Header("Content-Length", strconv.Itoa(len(exportResponse.Data)))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")

	c.Data(http.StatusOK, contentType, exportResponse.Data)
}

// InitiateAutoBulkTransfer initiates an auto bulk transfer for approved timesheets
func (h *PayrollHandler) InitiateAutoBulkTransfer(c *gin.Context) {
	if h.autoBulkTransferSvc == nil {
		response.Forbidden(c, "Auto bulk transfer is not enabled")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		response.InternalServerError(c, constants.MsgInvalidUserIDVN)
		return
	}

	var req dto.ExportBulkTransferRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	if err := validateExportBulkTransferRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.CreatedBy = userIDUint

	resp, err := h.autoBulkTransferSvc.InitiateBulkTransfer(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("InitiateAutoBulkTransfer error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessCreated(c, resp, "Auto bulk transfer initiated")
}

// GetAutoBulkTransferConfig returns whether auto bulk transfer is enabled.
func (h *PayrollHandler) GetAutoBulkTransferConfig(c *gin.Context) {
	response.Success(c, gin.H{"enabled": h.autoBulkTransferSvc != nil}, "")
}

// GetAutoBulkTransferStatus returns the status of an auto bulk transfer batch
func (h *PayrollHandler) GetAutoBulkTransferStatus(c *gin.Context) {
	if h.autoBulkTransferSvc == nil {
		response.Forbidden(c, "Auto bulk transfer is not enabled")
		return
	}

	batchID := c.Param("batch_id")
	if batchID == "" {
		response.BadRequest(c, "batch_id is required")
		return
	}

	resp, err := h.autoBulkTransferSvc.GetBatchStatus(c.Request.Context(), batchID)
	if err != nil {
		h.logger.Error("GetAutoBulkTransferStatus error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, resp, "")
}

// ImportBulkTransferResult imports bulk transfer results from an Excel file
func (h *PayrollHandler) ImportBulkTransferResult(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, constants.MsgNoFileUploadedVN)
		return
	}

	// Validate file extension
	if file.Filename == "" {
		response.BadRequest(c, constants.MsgFileNameEmptyVN)
		return
	}

	// Check file size (limit to 10MB)
	const maxFileSize = 10 << 20 // 10MB
	if file.Size > maxFileSize {
		response.BadRequest(c, constants.MsgFileSizeExceedsLimitVN)
		return
	}

	// Validate file format (should be .xlsx or .xls)
	filename := file.Filename
	lowerFilename := strings.ToLower(filename)
	if !strings.HasSuffix(lowerFilename, ".xlsx") && !strings.HasSuffix(lowerFilename, ".xls") {
		response.BadRequest(c, constants.MsgFileNotExcelFormatVN)
		return
	}

	// Process the bulk transfer result
	result, err := h.payrollService.ProcessBulkTransferResult(c.Request.Context(), file, userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Return success response with processing details
	message := fmt.Sprintf("Bulk transfer result processed successfully. %d records processed, %d succeeded, %d failed",
		result.TotalTxn, result.CompletedTxn, result.FailedTxn)

	response.Success(c, map[string]interface{}{
		"items":         result.Data,
		"total_txn":     result.TotalTxn,
		"completed_txn": result.CompletedTxn,
		"failed_txn":    result.FailedTxn,
	}, message)
}

// GetBulkTransferUploadHistories retrieves all bulk transfer upload histories with pagination
func (h *PayrollHandler) GetBulkTransferUploadHistories(c *gin.Context) {
	var req dto.ListBulkTransferHistoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	result, err := h.payrollService.GetBulkTransferUploadHistories(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Use SuccessWithPagination to avoid nested data structure
	pagination := response.Pagination{
		Page:         result.Pagination.Page,
		PageSize:     result.Pagination.PageSize,
		TotalPages:   result.Pagination.TotalPages,
		TotalRecords: int(result.Pagination.TotalRecords),
	}
	response.SuccessWithPagination(c, result.Data, constants.MsgDataRetrievedSuccessfullyVN, pagination)
}

// GetBulkTransferUploadHistoryByID retrieves a single bulk transfer upload history by ID
func (h *PayrollHandler) GetBulkTransferUploadHistoryByID(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidIDFormatVN)
	if !ok {
		return
	}

	history, err := h.payrollService.GetBulkTransferUploadHistoryByID(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{
		"items":         history.Data,
		"total_txn":     history.TotalTxn,
		"completed_txn": history.CompletedTxn,
		"failed_txn":    history.FailedTxn,
	}, constants.MsgDataRetrievedSuccessfullyVN)
}

// GetPayrollHistories retrieves payment histories with pagination and access control
func (h *PayrollHandler) GetPayrollHistories(c *gin.Context) {
	// Get user info from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole, exists := c.Get(constants.CtxUserRole)
	if !exists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	// Bind query parameters using DTO
	var req dto.ListPayrollHistoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	// Call service
	result, err := h.payrollService.GetPayrollHistories(c.Request.Context(), &req, userID.(uint), userRole.(string))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Use message constant
	message := constants.PayrollHistoriesFetched

	// Use SuccessWithPagination to avoid nested data structure
	pagination := response.Pagination{
		Page:         result.Pagination.Page,
		PageSize:     result.Pagination.PageSize,
		TotalPages:   result.Pagination.TotalPages,
		TotalRecords: int(result.Pagination.TotalRecords),
	}
	response.SuccessWithPagination(c, result.Data, message, pagination)
}

// ExportPayrollHistories exports payroll histories to Excel with multiple sheets
func (h *PayrollHandler) ExportPayrollHistories(c *gin.Context) {
	// Get user info from context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole, exists := c.Get(constants.CtxUserRole)
	if !exists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	// Parse JSON body
	var req dto.ExportPayrollHistoriesRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Validate date parameters
	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Call service layer
	excelBytes, filename, err := h.payrollService.ExportPayrollHistories(
		c.Request.Context(),
		&req,
		userID.(uint),
		userRole.(string),
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Set download headers
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", strconv.Itoa(len(excelBytes)))

	// Audit emit (best-effort): admin exported cross-employee payroll histories.
	if h.eventBus != nil {
		if uid, ok := userID.(uint); ok {
			event := domain.NewPayrollHistoriesExportedEvent(c.Request.Context(), uid, req.FromDate, req.ToDate, filename)
			if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
				h.logger.Warn("Failed to publish PayrollHistoriesExported audit event", "error", err)
			}
		}
	}

	// Return Excel file
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// DownloadBulkTransferHistoryFile downloads the original uploaded Excel file for a bulk transfer history
func (h *PayrollHandler) DownloadBulkTransferHistoryFile(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID không hợp lệ")
	if !ok {
		return
	}

	h.logger.Info("downloading bulk transfer history file", "id", id, "user_id", c.GetString("user_id"))

	// Fetch the bulk transfer file record first
	file, err := h.bulkTransferFileRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, "Không tìm thấy lịch sử chuyển tiền")
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin lịch sử chuyển tiền")
		return
	}

	if file.AssetID == nil {
		response.NotFound(c, "Không tìm thấy lịch sử chuyển tiền")
		return
	}

	// Fetch the asset using the bulk transfer file's asset_id
	asset, err := h.assetRepo.GetByID(c.Request.Context(), *file.AssetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.BadRequest(c, "File đính kèm không tồn tại")
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin file đính kèm")
		return
	}

	// Get full file path from file storage (asset.FilePath is relative)
	filePath := h.fileStorage.GetFilePath(asset.FilePath)

	// Check if file exists on disk
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.logger.Error("file not found on disk", "id", id, "asset_id", *file.AssetID, "relative_path", asset.FilePath, "full_path", filePath)
		response.BadRequest(c, "File không tồn tại trên hệ thống")
		return
	}

	// Read the file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		h.logger.Error("failed to read file", "id", id, "file_path", filePath, "error", err)
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

	h.logger.Info("bulk transfer history file downloaded", "id", id, "filename", asset.Filename, "size", len(fileData))

	// Audit emit (best-effort): admin re-downloaded a bulk transfer file.
	if h.eventBus != nil {
		userIDRaw, _ := c.Get("user_id")
		if uid, ok := userIDRaw.(uint); ok && uid != 0 {
			event := domain.NewBulkTransferFileDownloadedEvent(c.Request.Context(), uid, file.ID, *file.AssetID, asset.Filename)
			if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
				h.logger.Warn("Failed to publish BulkTransferFileDownloaded audit event", "error", err)
			}
		}
	}
}

// MarkExternallyPaid marks timesheets as paid via external transfer (outside the app).
func (h *PayrollHandler) MarkExternallyPaid(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	var req dto.MarkExternallyPaidRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	resp, err := h.payrollService.MarkExternallyPaid(c.Request.Context(), &req, userID.(uint))
	if err != nil {
		h.logger.Error("MarkExternallyPaid error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	if h.eventBus != nil {
		event := domain.NewTimesheetBulkExternallyPaidEvent(c.Request.Context(), userID.(uint), resp.MarkedCount, req.Reference)
		if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
			h.logger.Warn("Failed to publish TimesheetBulkExternallyPaid audit event", "error", err)
		}
	}

	response.Success(c, resp, fmt.Sprintf("Đã đánh dấu %d bảng chấm công là đã trả ngoài ứng dụng", resp.MarkedCount))
}

// GetPendingUploads returns bulk transfer exports awaiting result upload for the current user.
func (h *PayrollHandler) GetPendingUploads(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	files, err := h.bulkTransferFileRepo.GetPendingUploadsByUser(c.Request.Context(), userID.(uint))
	if err != nil {
		h.logger.Error("GetPendingUploads error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	type pendingUpload struct {
		ID                uint       `json:"id"`
		Filename          string     `json:"filename"`
		Cycle             string     `json:"cycle,omitempty"`
		TransactionsCount int        `json:"transactions_count"`
		TransferAmount    int64      `json:"transfer_amount"`
		CreatedAt         time.Time  `json:"created_at"`
		FromDate          *time.Time `json:"from_date,omitempty"`
		ToDate            *time.Time `json:"to_date,omitempty"`
	}

	items := make([]pendingUpload, len(files))
	for i, f := range files {
		items[i] = pendingUpload{
			ID:                f.ID,
			Filename:          f.Filename,
			TransactionsCount: f.TransactionsCount,
			TransferAmount:    f.TransferAmount,
			CreatedAt:         f.CreatedAt,
			FromDate:          f.FromDate,
			ToDate:            f.ToDate,
		}
		if f.Cycle != nil {
			items[i].Cycle = *f.Cycle
		}
	}

	response.Success(c, items, "")
}

// EstimateFee returns the estimated disbursement fee for a potential bulk transfer.
func (h *PayrollHandler) EstimateFee(c *gin.Context) {
	if h.autoBulkTransferSvc == nil {
		response.Forbidden(c, "Auto bulk transfer is not enabled")
		return
	}

	var req dto.ExportBulkTransferRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	if err := validateExportBulkTransferRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.autoBulkTransferSvc.EstimateFee(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("EstimateFee error", "error", err)
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, resp, "")
}
