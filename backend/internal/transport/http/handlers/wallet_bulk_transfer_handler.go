package handlers

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"api-server/internal/app/services/wallet_bulk"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// MaxBulkUploadBytes is the hard cap on uploaded .xlsx size. 10 MiB is
// generous for a 5,000-row workbook (typical is <100KB) but small enough
// that a malicious upload can't OOM us. Defends against zip bombs alongside
// the parser's UnzipSizeLimit.
const MaxBulkUploadBytes = 10 << 20 // 10 MiB

// WalletBulkTransferHandler exposes the wallet bulk transfer endpoints.
type WalletBulkTransferHandler struct {
	svc    *wallet_bulk.WalletBulkTransferService
	logger *slog.Logger
}

// NewWalletBulkTransferHandler wires the handler.
func NewWalletBulkTransferHandler(svc *wallet_bulk.WalletBulkTransferService, logger *slog.Logger) *WalletBulkTransferHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WalletBulkTransferHandler{svc: svc, logger: logger}
}

// UploadBulkTransfer handles POST /wallet/bulk-transfer/upload.
//
// Defense-in-depth on the upload size + format:
//  1. http.MaxBytesReader BEFORE c.FormFile (limits total request body).
//  2. fileHeader.Size check (catches oversized files even if the stream lies).
//  3. ZIP magic sniff (50 4B 03/05/07 04/06/08) — rejects non-xlsx content
//     even if the Content-Type or extension claims otherwise.
//  4. io.LimitReader on the actual read (defense-in-depth).
//
// All hard limits are enforced BEFORE the parser runs, so a zip bomb can't
// even reach excelize's UnzipSizeLimit.
func (h *WalletBulkTransferHandler) UploadBulkTransfer(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}
	if h.svc == nil {
		response.InternalServerError(c, "wallet bulk transfer service not configured")
		return
	}

	// Step 1: cap the request body BEFORE c.FormFile reads it.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBulkUploadBytes+512)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "http: request body too large") {
			response.BadRequest(c, "File vượt quá 10MB")
			return
		}
		response.BadRequest(c, "File không hợp lệ")
		return
	}

	// Step 2: fileHeader size check.
	if fileHeader.Size > MaxBulkUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "file_too_large",
			"max_bytes": MaxBulkUploadBytes,
		})
		return
	}

	// Step 3: open + ZIP magic sniff.
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}
	defer func() { _ = file.Close() }()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	head = head[:n]
	if _, seekErr := file.Seek(0, 0); seekErr != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}
	if !isZipMagic(head) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_file_type",
			"message": "File phải là .xlsx hợp lệ",
		})
		return
	}

	// Step 4: read all bytes with a hard cap.
	fileBytes, err := io.ReadAll(io.LimitReader(file, MaxBulkUploadBytes+1))
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}
	if len(fileBytes) > MaxBulkUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "file_too_large",
			"max_bytes": MaxBulkUploadBytes,
		})
		return
	}

	// Resolve the actor.
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

	// Step 5: run the pipeline.
	result, err := h.svc.Upload(c.Request.Context(), fileBytes, fileHeader.Filename, uint64(userIDUint))
	if err != nil {
		h.mapUploadError(c, err)
		return
	}

	// Wrap in the standard {status, data, message} envelope so the
	// frontend's apiClient.upload<T>() → res.data unwrapping works (it
	// expects ApiResponse<T>, not a bare struct). All other upload handlers
	// (e.g. employee/import_handler.go) wrap via response.Success; we keep
	// HTTP 202 Accepted here because the batch is processed asynchronously
	// per-row — the upload is "accepted", not yet "completed".
	c.JSON(http.StatusAccepted, response.SuccessResponse{
		Status:  "success",
		Data:    result,
		Message: "Đã nhận file — đang xử lý hàng loạt",
	})
}

// GetBatch handles GET /wallet/bulk-transfer/batches/:id.
func (h *WalletBulkTransferHandler) GetBatch(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequest(c, "Invalid batch id")
		return
	}
	detail, err := h.svc.GetBatch(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBulkTransferBatchNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "batch_not_found"})
			return
		}
		h.logger.Error("wallet_bulk: get batch", "id", id, "error", err)
		response.InternalServerError(c, "Không thể tải lô chuyển tiền")
		return
	}
	c.JSON(http.StatusOK, response.SuccessResponse{
		Status:  "success",
		Data:    detail,
		Message: "",
	})
}

// ListBatches handles GET /wallet/bulk-transfer/batches.
func (h *WalletBulkTransferHandler) ListBatches(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.svc.ListBatches(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("wallet_bulk: list batches", "error", err)
		response.InternalServerError(c, "Không thể tải danh sách lô")
		return
	}
	c.JSON(http.StatusOK, response.SuccessResponse{
		Status:  "success",
		Data:    result,
		Message: "",
	})
}

// DownloadKQ handles GET /wallet/bulk-transfer/batches/:id/kq.
// Returns the .xlsx as a blob with Content-Disposition. Works at any status.
func (h *WalletBulkTransferHandler) DownloadKQ(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.BadRequest(c, "Invalid batch id")
		return
	}
	scope := c.DefaultQuery("scope", "all")
	if scope != "all" && scope != "successful" {
		response.BadRequest(c, "scope must be all or successful")
		return
	}
	bytes, filename, err := h.svc.DownloadKQScoped(c.Request.Context(), id, scope == "successful")
	if err != nil {
		if errors.Is(err, domain.ErrBulkTransferBatchNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "batch_not_found"})
			return
		}
		h.logger.Error("wallet_bulk: download kq", "id", id, "error", err)
		response.InternalServerError(c, "Không thể tạo file KQ")
		return
	}
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Content-Length", strconv.Itoa(len(bytes)))
	c.Header("Cache-Control", "no-store")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Data(http.StatusOK, contentType, bytes)
}

// mapUploadError translates service-layer errors into HTTP responses
// matching the error table in phase-04-kq-excel-routes.md.
func (h *WalletBulkTransferHandler) mapUploadError(c *gin.Context, err error) {
	// Duplicate VFIC — 409 with conflict list.
	var dupVFIC *wallet_bulk.DuplicateVFICError
	if errors.As(err, &dupVFIC) {
		c.JSON(http.StatusConflict, gin.H{
			"error":     "duplicate_vfic",
			"conflicts": dupVFIC.Conflicts,
		})
		return
	}

	// Duplicate content-hash — 409.
	if errors.Is(err, wallet_bulk.ErrDuplicateBatch) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "duplicate_batch",
			"message": "File này đã được tải lên trước đó",
		})
		return
	}

	// Format errors — 400 with helpful messages.
	switch {
	case errors.Is(err, wallet_bulk.ErrMissingSwiftColumn):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_swift_column",
			"message": "File thiếu cột SWIFT. Vui lòng dùng nút 'Chuyển OnePay' trên trang Bảng công để xuất file.",
		})
	case errors.Is(err, wallet_bulk.ErrInvalidSwift):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_swift", "message": err.Error()})
	case errors.Is(err, wallet_bulk.ErrMissingVFICCode):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_row", "message": err.Error()})
	case errors.Is(err, wallet_bulk.ErrInvalidAmount):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_row", "message": err.Error()})
	case errors.Is(err, wallet_bulk.ErrUnknownExcelHeader):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_excel_format", "message": err.Error()})
	case errors.Is(err, wallet_bulk.ErrInvalidSheet):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_excel_format", "message": err.Error()})
	case errors.Is(err, wallet_bulk.ErrEmptyFile):
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty_file", "message": "File không có giao dịch nào"})
	case errors.Is(err, wallet_bulk.ErrTooManyRows):
		c.JSON(http.StatusBadRequest, gin.H{"error": "too_many_rows", "max": wallet_bulk.MaxRows})
	default:
		h.logger.Error("wallet_bulk: upload failed", "error", err)
		response.InternalServerError(c, "Không thể xử lý file")
	}
}

// isZipMagic reports whether the head bytes match a ZIP local file header
// (PK\x03\x04 / PK\x05\x06 / PK\x07\x08). Used to reject non-xlsx uploads
// before the parser runs.
func isZipMagic(head []byte) bool {
	if len(head) < 4 {
		return false
	}
	return head[0] == 0x50 && head[1] == 0x4B &&
		(head[2] == 0x03 || head[2] == 0x05 || head[2] == 0x07) &&
		(head[3] == 0x04 || head[3] == 0x06 || head[3] == 0x08)
}

// parseUintParam reads :param as uint64.
func parseUintParam(c *gin.Context, name string) (uint64, error) {
	raw := c.Param(name)
	if raw == "" {
		return 0, errors.New("missing param")
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
