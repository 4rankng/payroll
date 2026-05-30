package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CancelMyAdvancePaymentRequest cancels the logged-in employee's own pending request
func (h *AdvancePaymentHandler) CancelMyAdvancePaymentRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID yêu cầu không hợp lệ")
		return
	}

	if err := h.service.CancelMyRequestByUserID(c.Request.Context(), requestID, uint64(userID.(uint))); err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil, "Hủy yêu cầu ứng lương thành công")
}

// CancelAdvancePaymentRequest cancels a pending request (admin only)
func (h *AdvancePaymentHandler) CancelAdvancePaymentRequest(c *gin.Context) {
	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID yêu cầu không hợp lệ")
		return
	}

	if err := h.service.CancelRequest(c.Request.Context(), requestID); err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, nil, "Hủy yêu cầu ứng lương thành công")
}

// GetAdvancePaymentSummary returns a summary of advance payments
func (h *AdvancePaymentHandler) GetAdvancePaymentSummary(c *gin.Context) {
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	var fromDate, toDate *time.Time

	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			fromDate = &t
		}
	}
	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			toDate = &t
		}
	}

	var fromTime, toTime time.Time
	if fromDate != nil {
		fromTime = *fromDate
	}
	if toDate != nil {
		toTime = *toDate
	}

	forMonthStr := c.Query("forMonth")

	summary, err := h.service.GetSummary(c.Request.Context(), fromTime, toTime, forMonthStr)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	resp := dto.AdvancePaymentSummaryResponse{
		TotalRequests:           summary.TotalRequests,
		TotalPending:            summary.TotalPending,
		TotalApproved:           summary.TotalApproved,
		TotalCancelled:          summary.TotalCancelled,
		TotalFailed:             summary.TotalFailed,
		TotalPaid:               summary.TotalPaid,
		TotalAmount:             summary.TotalAmount,
		TotalPaidAmount:         summary.TotalPaidAmount,
		TotalPendingAmount:      summary.TotalPendingAmount,
		TotalFailedAmount:       summary.TotalFailedAmount,
		TotalCancelledAmount:    summary.TotalCancelledAmount,
		TotalFee:                summary.TotalFee,
		TotalFeeEarned:          summary.TotalFeeEarned,
		TotalFeeEarnedAllTime:   summary.TotalFeeEarnedAllTime,
		TotalNet:                summary.TotalNet,
		TotalProviderFee:        summary.TotalProviderFee,
		TotalProviderFeeAllTime: summary.TotalProviderFeeAllTime,
		AvgProcessingTimeSecs:   summary.AvgProcessingTimeSecs,
		CompletedUnder30s:       summary.CompletedUnder30s,
		FeePercentage:           summary.FeePercentage,
		AvgFeePerRequest:        summary.AvgFeePerRequest,
		AvgFeePerEmployee:       summary.AvgFeePerEmployee,
		SuccessRate:             summary.SuccessRate,
		DisbursementPercentage:  summary.DisbursementPercentage,
		FromDate:                summary.FromDate,
		ToDate:                  summary.ToDate,
	}

	response.Success(c, resp, "Lấy tổng hợp ứng lương thành công")
}
