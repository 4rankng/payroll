package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ListAdvancePayments lists all advance payment requests with filters
func (h *AdvancePaymentHandler) ListAdvancePayments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	search := c.Query("search")
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	if page < 1 {
		page = 1
	}
	// Clamp instead of silently resetting: a client asking for 150 gets the
	// maximum page (100), not a surprise 20.
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filters := domain.AdvancePaymentRequestFilters{
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Search:    search,
	}

	if status != "" {
		filters.Status = &status
	}

	if projectIDStr := c.Query("projectId"); projectIDStr != "" {
		if pid, err := strconv.ParseUint(projectIDStr, 10, 64); err == nil {
			filters.ProjectID = &pid
		}
	}

	if employeeIDStr := c.Query("employeeId"); employeeIDStr != "" {
		if eid, err := strconv.ParseUint(employeeIDStr, 10, 64); err == nil {
			filters.EmployeeID = &eid
		}
	}

	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			filters.FromDate = &t
		}
	}
	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			filters.ToDate = &t
		}
	}

	if forMonthStr := c.Query("forMonth"); forMonthStr != "" {
		filters.ForMonth = &forMonthStr
	}

	requests, total, err := h.service.ListRequests(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	items := make([]dto.AdvancePaymentRequestItem, len(requests))
	for i, r := range requests {
		item := dto.AdvancePaymentRequestItem{
			ID:                      r.ID,
			EmployeeID:              r.EmployeeID,
			ProjectID:               r.ProjectID,
			RequestAmount:           r.RequestAmount,
			Fee:                     r.Fee,
			NetAmount:               r.NetAmount,
			Status:                  string(r.Status),
			CreatedAt:               r.CreatedAt,
			PaidAt:                  r.PaidAt,
			SettlementTransactionID: r.SettlementTransactionID,
			ProviderFee:             r.ProviderFee,
		}

		if r.Employee.ID != 0 {
			item.EmployeeName = r.Employee.Fullname
			item.EmployeeCCCD = r.Employee.CCCD
		}
		if r.Project.ID != 0 {
			item.ProjectCode = r.Project.Code
			item.ProjectName = r.Project.Name
		}
		if r.PaymentReference != nil {
			item.PaymentReference = r.PaymentReference
		}

		items[i] = item
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, items, "Lấy danh sách yêu cầu ứng lương thành công",
		response.Pagination{Page: page, PageSize: pageSize, TotalPages: totalPages, TotalRecords: int(total)})
}
