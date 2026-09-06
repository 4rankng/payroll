package timesheet

import (
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"

	"github.com/gin-gonic/gin"
)

// UploadSettlementResult uploads and processes a settlement result file
// @Summary Upload settlement result file
// @Description Upload an Excel file containing settlement results and process revenue settlements
// @Tags timesheets
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file to upload"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/payroll/upload-settlement-result [post]
func (h *Handler) UploadSettlementResult(c *gin.Context) {
	// Get user context
	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Guarded multipart parse: body cap, size cap, xlsx magic sniff.
	header, ok := uploadguard.Validate(c, false)
	if !ok {
		return
	}

	// Process the settlement file using dedup path: re-processes orphaned timesheets
	// (revenue_paid=1 but transaction still pending) instead of blocking with an error.
	result, err := h.settlementUploadService.ProcessSettlementFileWithDedup(
		c.Request.Context(),
		header,
		userID,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Create notification when timesheets were processed
	if result.ProcessedTimesheets > 0 && result.AssetID > 0 {
		h.settlementUploadService.CreateSettlementNotification(
			c.Request.Context(),
			userID,
			result.SettlementAmount,
			result.AssetID,
			result.SkippedIDs,
			false,
		)
	}

	// Build response
	responseData := dto.UploadSettlementResultResponse{
		TotalTimesheets:    result.ProcessedTimesheets + result.SkippedTimesheets,
		SettledTimesheets:  result.ProcessedTimesheets,
		SettlementAmount:   result.SettlementAmount,
		AssetID:            result.AssetID,
		SettlementsCreated: result.SettlementsCreated,
	}

	var message string
	if result.SkippedTimesheets > 0 && result.ProcessedTimesheets > 0 {
		message = fmt.Sprintf("Đã xử lý file đối soát thành công. %d timesheets mới, %d timesheets bỏ qua (đã xử lý trước), %d settlements được tạo",
			result.ProcessedTimesheets, result.SkippedTimesheets, result.SettlementsCreated)
	} else if result.SkippedTimesheets > 0 {
		message = fmt.Sprintf("Tất cả %d timesheets đã được xử lý trước đó", result.SkippedTimesheets)
	} else {
		message = fmt.Sprintf("Đã xử lý file đối soát thành công. %d timesheets đã được đánh dấu thanh toán, %d settlements được tạo",
			result.ProcessedTimesheets, result.SettlementsCreated)
	}

	response.Success(c, responseData, message)
}
