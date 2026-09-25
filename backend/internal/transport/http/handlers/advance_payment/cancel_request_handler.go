package advance_payment

import (
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

	// The service already returns the full response DTO (including derived
	// metrics and FromDate/ToDate); re-copying fields here would silently drop
	// any future field the service adds.
	summary, err := h.service.GetSummary(c.Request.Context(), fromTime, toTime, forMonthStr)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, summary, "Lấy tổng hợp ứng lương thành công")
}
