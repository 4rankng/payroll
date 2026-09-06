package settlement

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"

	"github.com/gin-gonic/gin"
)

// UploadSettlement handles POST /email/history/:id/upload-settlement.
// It uploads a settlement Excel file to a notification record and processes it with dedup.
func (h *Handler) UploadSettlement(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	// Parse notification ID
	idStr := c.Param("id")
	id, err := parseID(idStr)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	// Fetch and validate the notification record
	notification, err := h.notificationRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, domain.NewNotFoundError(constants.MsgEmailHistoryRecordNotFoundVN))
		return
	}

	isPayrollReport := notification.Type == domain.NotificationTypePayrollReport
	isAdvanceReport := notification.Type == domain.NotificationTypeAdvancePaymentReport
	if !isPayrollReport && !isAdvanceReport {
		response.HandleDomainError(c, domain.NewValidationError(constants.MsgCanOnlyReconcileFromStatementEmailVN))
		return
	}

	if notification.Metadata == nil || *notification.Metadata == "" {
		response.HandleDomainError(c, domain.NewValidationError(constants.MsgStatementEmailNoFinancialDataVN))
		return
	}

	var meta domain.PayrollEmailMetadata
	if err := json.Unmarshal([]byte(*notification.Metadata), &meta); err != nil {
		response.HandleDomainError(c, domain.NewInternalError(constants.MsgFailedToParseEmailMetadataVN, err))
		return
	}

	// Guarded multipart parse: body cap, size cap, xlsx magic sniff.
	header, ok := uploadguard.Validate(c, false)
	if !ok {
		return
	}

	// Get user ID from context
	userIDVal, _ := c.Get(constants.CtxUserID)
	userID, _ := userIDVal.(uint)

	// Process with dedup
	result, err := h.settlementUploadSvc.ProcessSettlementFileWithDedup(
		c.Request.Context(),
		header,
		userID,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Mark notification as settled
	if result.ProcessedTimesheets > 0 {
		meta.SettledAt = &time.Time{}
		*meta.SettledAt = h.clock.Now()
		if updatedJSON, jsonErr := json.Marshal(meta); jsonErr == nil {
			_ = h.notificationRepo.UpdateMetadata(c.Request.Context(), id, string(updatedJSON))
		}
	}

	resp := dto.UploadSettlementResponse{
		ProcessedTimesheets: result.ProcessedTimesheets,
		SkippedTimesheets:   result.SkippedTimesheets,
		SettlementsCreated:  result.SettlementsCreated,
		SettlementAmount:    result.SettlementAmount,
	}

	if result.SkippedTimesheets > 0 && result.ProcessedTimesheets > 0 {
		response.Success(c, resp, fmt.Sprintf(
			"Đã xử lý %d timesheets mới, bỏ qua %d timesheets đã thanh toán trước đó",
			result.ProcessedTimesheets, result.SkippedTimesheets))
	} else if result.SkippedTimesheets > 0 {
		response.Success(c, resp, fmt.Sprintf(
			"Tất cả %d timesheets đã được thanh toán trước đó, không cần xử lý thêm",
			result.SkippedTimesheets))
	} else {
		response.Success(c, resp, fmt.Sprintf(
			"Đã xử lý file đối soát thành công. %d timesheets đã được thanh toán, %d settlements được tạo",
			result.ProcessedTimesheets, result.SettlementsCreated))
	}
}

func isAdmin(c *gin.Context) bool {
	role := c.GetString(constants.CtxUserRole)
	return role == string(domain.RoleAdmin)
}

func parseID(idStr string) (uint, error) {
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return uint(id), nil
}
