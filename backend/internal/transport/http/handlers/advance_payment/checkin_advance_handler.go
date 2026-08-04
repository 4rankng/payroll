package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetMyCheckInAdvanceInfo returns the self-check-in advance summary for the logged-in
// employee via the DEDICATED /me/check-in-advance path. Separate from the admin-upload
// /me/advance-payment flow: Admin-configured advanceable cap, calendar-month salary
// period, day-10 request window, dual-wage display + OT/allowance disclaimer.
func (h *AdvancePaymentHandler) GetMyCheckInAdvanceInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	info, err := h.service.GetCheckInAdvanceInfoByUserID(c.Request.Context(), uint64(userID.(uint)))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	// Map into the shared AdvancePaymentInfoResponse so the existing employee
	// components (limit card, request form) are reused; salary/disclaimer/
	// windowOpenDay are populated only on this dedicated check-in path.
	cfg := h.service.GetConfig()
	resp := dto.AdvancePaymentInfoResponse{
		ForMonth:         info.ForMonth,
		MaxAdvanceAmount: info.MaxAdvanceAmount,
		CompletedAmount:  info.CompletedAmount,
		PendingAmount:    info.PendingAmount,
		RemainingAmount:  info.RemainingAmount,
		CanRequest:       info.CanRequest,
		CanRequestTitle:  info.CanRequestTitle,
		CanRequestReason: info.CanRequestReason,
		FeePercentage:    cfg.GetAdvanceCashFeePercentage(c.Request.Context()),
		MinFee:           cfg.GetAdvancePaymentFeeMin(c.Request.Context()),
		HasFlexible:      info.HasFlexible,
		Quotas: []dto.AdvancePaymentQuotaResponse{{
			ForMonth:         info.ForMonth,
			MaxAdvanceAmount: info.MaxAdvanceAmount,
			CompletedAmount:  info.CompletedAmount,
			PendingAmount:    info.PendingAmount,
			RemainingAmount:  info.RemainingAmount,
		}},
		Salary:          info.Salary,
		PendingEarnings: info.PendingEarnings,
		Disclaimer:      info.Disclaimer,
		WindowOpenDay:   info.WindowOpenDay,
	}
	if cfg.GetTransferLimits != nil {
		limits := cfg.GetTransferLimits(c.Request.Context())
		resp.ProviderMinTransferAmount = uint64(limits.MinAmount)
		resp.ProviderMaxTransferAmount = uint64(limits.MaxAmount)
	}

	response.Success(c, resp, "Lấy thông tin ứng lương tự chấm công thành công")
}

// CreateCheckInAdvanceRequest creates an advance request under the self-check-in flow
// via the DEDICATED /me/check-in-advance/request path.
func (h *AdvancePaymentHandler) CreateCheckInAdvanceRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	var req dto.CreateAdvancePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	forMonth := req.ForMonth
	if forMonth == "" {
		forMonth = h.clock.Now().Format("2006-01")
	}

	result, err := h.service.CreateCheckInAdvanceRequestByUserID(c.Request.Context(), uint64(userID.(uint)), req.Amount, forMonth)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.SuccessCreated(c, result, "Yêu cầu ứng lương đã được tạo thành công")
}
