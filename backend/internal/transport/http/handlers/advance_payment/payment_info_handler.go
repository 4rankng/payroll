package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetMyAdvancePaymentInfo returns advance payment info for the logged-in employee
func (h *AdvancePaymentHandler) GetMyAdvancePaymentInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	info, err := h.service.GetEmployeeAdvanceInfoByUserID(c.Request.Context(), uint64(userID.(uint)))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	cfg := h.service.GetConfig()
	feePercentage := cfg.GetAdvanceCashFeePercentage(c.Request.Context())
	minFee := cfg.GetAdvancePaymentFeeMin(c.Request.Context())

	resp := dto.AdvancePaymentInfoResponse{
		ForMonth:         info.CurrentMonth,
		MaxAdvanceAmount: info.TotalMaxAdvance,
		CompletedAmount:  info.CompletedAmount,
		PendingAmount:    info.PendingAmount,
		RemainingAmount:  info.RemainingAmount,
		CanRequest:       info.CanRequest,
		CanRequestTitle:  info.CanRequestTitle,
		CanRequestReason: info.CanRequestReason,
		FeePercentage:    feePercentage,
		MinFee:           minFee,
		HasFlexible:      info.HasFlexibleSchedule,
		Quotas:           make([]dto.AdvancePaymentQuotaResponse, len(info.Quotas)),
	}

	// Populate disbursement provider transfer limits
	if cfg.GetTransferLimits != nil {
		limits := cfg.GetTransferLimits(c.Request.Context())
		resp.ProviderMinTransferAmount = uint64(limits.MinAmount)
		resp.ProviderMaxTransferAmount = uint64(limits.MaxAmount)
	}

	for i, q := range info.Quotas {
		resp.Quotas[i] = dto.AdvancePaymentQuotaResponse{
			ForMonth:         q.ForMonth,
			MaxAdvanceAmount: q.MaxAdvanceAmount,
			CompletedAmount:  q.CompletedAmount,
			PendingAmount:    q.PendingAmount,
			RemainingAmount:  q.RemainingAmount,
		}
	}

	response.Success(c, resp, "Lấy thông tin ứng lương thành công")
}
