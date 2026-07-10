package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetMyAdvancePaymentHistory returns the request history for the logged-in employee
func (h *AdvancePaymentHandler) GetMyAdvancePaymentHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Optional salary-period scope (forMonth=YYYY-MM). Preferred for employee
	// history: an advance is charged to a salary period, which can differ from
	// the calendar month it was submitted in (days 1–8 charge to the previous
	// period). Omitted = all-time history.
	var forMonth *string
	if fm := c.Query("forMonth"); fm != "" {
		forMonth = &fm
	}

	// Optional created_at date-range fallback (inclusive bounds, matching the
	// admin ListAdvancePayments convention).
	var fromDate, toDate *time.Time
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		if t, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			fromDate = &t
		}
	}
	if toDateStr := c.Query("toDate"); toDateStr != "" {
		if t, err := time.Parse("2006-01-02", toDateStr); err == nil {
			toDate = &t
		}
	}

	items, total, err := h.service.GetRequestHistoryByUserID(c.Request.Context(), uint64(userID.(uint)), pageSize, offset, fromDate, toDate, forMonth)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	// Convert to DTO items
	data := make([]dto.AdvancePaymentHistoryItem, len(items))
	for i, item := range items {
		data[i] = dto.AdvancePaymentHistoryItem{
			ID:            item.ID,
			RequestAmount: item.RequestAmount,
			Fee:           item.Fee,
			NetAmount:     item.NetAmount,
			Status:        string(item.Status),
			CreatedAt:     item.CreatedAt,
			PaidAt:        item.PaidAt,
			ProjectName:   item.ProjectName,
			ProjectCode:   item.ProjectCode,
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, data, "Lấy lịch sử ứng lương thành công",
		response.Pagination{Page: page, PageSize: pageSize, TotalPages: totalPages, TotalRecords: int(total)})
}

// CalculateFeePreview calculates the fee for a given amount
func (h *AdvancePaymentHandler) CalculateFeePreview(c *gin.Context) {
	var req dto.CalculateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	fee, netAmount := h.service.CalculateFeePreview(c.Request.Context(), req.Amount)

	resp := dto.CalculateFeeResponse{
		RequestAmount: req.Amount,
		Fee:           fee,
		NetAmount:     netAmount,
	}

	response.Success(c, resp, "Tính phí thành công")
}
