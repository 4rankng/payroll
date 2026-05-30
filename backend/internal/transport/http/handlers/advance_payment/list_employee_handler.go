package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetEmployees returns list of employees from the latest imported flex pay month with their advance payment statistics
func (h *AdvancePaymentHandler) GetEmployees(c *gin.Context) {
	forMonth := c.Query("forMonth")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 2000 {
		pageSize = 20
	}

	sortBy := c.DefaultQuery("sortBy", "max_advance_amount")
	sortOrder := c.DefaultQuery("sortOrder", "DESC")

	filters := domain.EmployeeAdvanceStatsFilters{
		Search:    search,
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	if forMonth != "" {
		filters.ForMonth = &forMonth
	}

	employees, _, total, err := h.service.GetEmployeeAdvanceStats(c.Request.Context(), filters)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.SuccessWithPagination(c, []dto.EmployeeAdvanceItem{}, "Lấy danh sách nhân viên thành công",
				response.Pagination{Page: page, PageSize: pageSize, TotalPages: 0, TotalRecords: 0})
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	// Convert to DTO items
	data := make([]dto.EmployeeAdvanceItem, len(employees))
	for i, emp := range employees {
		// Clamp the subtraction at zero. uint64 underflows wrap around to
		// ~1.8e19 ("18 trillion trillion") if Utilized + Pending ever
		// exceeds Max — which can happen transiently around a downward
		// adjustment of the ceiling. Defensive even after the SUM-aware
		// JOIN fix.
		var available uint64
		drawn := emp.UtilizedAmount + emp.PendingAmount
		if emp.MaxAdvanceAmount > drawn {
			available = emp.MaxAdvanceAmount - drawn
		}
		item := dto.EmployeeAdvanceItem{
			EmployeeID:             emp.EmployeeID,
			Fullname:               emp.Fullname,
			CCCD:                   emp.CCCD,
			Username:               emp.Username,
			Email:                  emp.Email,
			ForMonth:               emp.ForMonth,
			MaxAdvanceAmount:       emp.MaxAdvanceAmount,
			UtilizedAmount:         emp.UtilizedAmount,
			AvailableAmount:        available,
			TotalFeeGenerated:      emp.TotalFeeGenerated,
			PendingAmount:          emp.PendingAmount,
			CompletedRequestsCount: emp.CompletedRequestsCount,
			PendingRequestsCount:   emp.PendingRequestsCount,
			CreatedAt:              emp.CreatedAt,
		}

		// Set bank info if available
		if emp.BankID != nil && emp.BankAccountNumber != "" {
			bankName := ""
			if emp.BankName != nil {
				bankName = *emp.BankName
			}
			item.Bank = &dto.EmployeeAdvanceBankInfo{
				BankID:        *emp.BankID,
				BankName:      bankName,
				AccountNumber: emp.BankAccountNumber,
				AccountName:   emp.BankAccountName,
			}
		}

		// Set project info
		item.Project = &dto.EmployeeAdvanceProjectInfo{
			ID:             emp.ProjectID,
			Name:           emp.ProjectName,
			Code:           emp.ProjectCode,
			AssignmentID:   emp.ProjectEmployeeID,
			CheckInEnabled: emp.CheckInEnabled,
		}

		data[i] = item
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, data, "Lấy danh sách nhân viên thành công",
		response.Pagination{Page: page, PageSize: pageSize, TotalPages: totalPages, TotalRecords: int(total)})
}
