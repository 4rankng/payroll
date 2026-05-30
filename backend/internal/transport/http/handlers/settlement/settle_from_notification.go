package settlement

import (
	"encoding/json"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// SettleFromNotification handles POST /email/history/:id/settle.
// It processes settlement for a payroll or advance payment notification using stored metadata.
func (h *Handler) SettleFromNotification(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	idStr := c.Param("id")
	id, err := parseID(idStr)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

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

	if meta.SettledAt != nil {
		response.HandleDomainError(c, domain.NewValidationError(constants.MsgStatementEmailAlreadyReconciledVN))
		return
	}

	if isPayrollReport {
		if len(meta.TimesheetIDs) == 0 {
			response.HandleDomainError(c, domain.NewValidationError(constants.MsgStatementEmailNoTimesheetsVN))
			return
		}

		if meta.SaoKeAssetID == nil || *meta.SaoKeAssetID == 0 {
			response.HandleDomainError(c, domain.NewValidationError(constants.MsgStatementEmailNoAttachmentsVN))
			return
		}

		if err := h.settlementUploadSvc.SettleFromMetadata(c.Request.Context(), meta.TimesheetIDs, meta.TotalWithFee, *meta.SaoKeAssetID); err != nil {
			response.HandleDomainError(c, err)
			return
		}
	}

	now := h.clock.Now()
	meta.SettledAt = &now
	updatedJSON, err := json.Marshal(meta)
	if err != nil {
		response.HandleDomainError(c, domain.NewInternalError(constants.MsgFailedToSerializeUpdatedMetadataVN, err))
		return
	}

	if err := h.notificationRepo.UpdateMetadata(c.Request.Context(), id, string(updatedJSON)); err != nil {
		response.HandleDomainError(c, domain.NewInternalError(constants.MsgFailedToSerializeUpdatedMetadataVN, err))
		return
	}

	response.Success(c, nil, "Đã xác nhận nhận tiền và xử lý đối soát thành công")
}
