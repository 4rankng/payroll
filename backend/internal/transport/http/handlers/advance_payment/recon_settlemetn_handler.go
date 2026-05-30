package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/response"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// UploadReconciliationSettlement handles uploading reconciliation file to settle receivables
// POST /api/v1/advance-payments/reconciliation/settle
func (h *AdvancePaymentHandler) UploadReconciliationSettlement(c *gin.Context) {
	logger := observability.GetLogger()
	ctx := c.Request.Context()

	// Get file from form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File là bắt buộc")
		return
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Không thể mở file")
		return
	}
	defer func() { _ = file.Close() }()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		response.InternalServerError(c, "Không thể đọc file")
		return
	}

	// Parse Excel file
	xlsxFile, err := excelize.OpenReader(bytes.NewReader(fileContent))
	if err != nil {
		response.BadRequest(c, "File Excel không hợp lệ")
		return
	}
	defer func() { _ = xlsxFile.Close() }()

	logger.Info("Processing reconciliation settlement upload",
		"filename", fileHeader.Filename,
		"size", len(fileContent))

	// Process settlement using the service
	result, err := h.flexPaySettlementService.ProcessSettlementFile(ctx, xlsxFile)
	if err != nil {
		logger.Error("Failed to process settlement file", "error", err)
		response.BadRequest(c, err.Error())
		return
	}

	logger.Info("Reconciliation settlement completed",
		"settled_count", result.SettledCount,
		"settled_at", result.SettledAt)

	h.saveSaoKeResultAsset(c, fileContent, fileHeader.Filename)

	response.Success(c, dto.ReconciliationSettlementResult{
		SettledCount: result.SettledCount,
		RequestIDs:   result.RequestIDs,
		SettledAt:    result.SettledAt,
	}, result.Message)
}

func (h *AdvancePaymentHandler) saveSaoKeResultAsset(c *gin.Context, fileContent []byte, filename string) {
	logger := observability.GetLogger()

	userID, exists := c.Get("user_id")
	if !exists {
		logger.Warn("no user_id in context, skipping asset creation for sao ke result")
		return
	}

	hash := sha256.Sum256(fileContent)
	checksum := fmt.Sprintf("%x", hash)

	storedFile, err := h.fileStorage.StoreBytes(fileContent, filename, domain.UploadTypeAdvancePaymentSaoKeResult)
	if err != nil {
		logger.Error("failed to store sao ke result file", "error", err)
		return
	}

	asset := &domain.Asset{
		Filename:   filename,
		FilePath:   storedFile.FilePath,
		UploadType: domain.UploadTypeAdvancePaymentSaoKeResult,
		Checksum:   &checksum,
		UploadedBy: userID.(uint),
	}
	if _, err := h.assetRepo.Create(c.Request.Context(), asset); err != nil {
		logger.Error("failed to create asset record for sao ke result", "error", err)
	}
}
