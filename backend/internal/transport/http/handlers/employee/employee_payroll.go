package employee

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetEmployeePayroll retrieves payroll history for a specific employee
// @Summary Get employee payroll history
// @Description Get payroll payment history for a specific employee with filtering options
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(10)
// @Param sortBy query string false "Field to sort by (created_at, date, amount, payment_status)" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc" default(desc)
// @Param status query string false "Filter by payment status (pending, paid, failed, cancelled)"
// @Param fromDate query string false "Start date for filtering (YYYY-MM-DD)"
// @Param toDate query string false "End date for filtering (YYYY-MM-DD)"
// @Success 200 {object} dto.EmployeePayrollHistoryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/payroll [get]
func (h *Handler) GetEmployeePayroll(c *gin.Context) {
	// 1. Validate ID parameter
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidEmployeeIDVN)
		return
	}

	// 2. Check if employee exists first
	_, err = h.employeeService.GetEmployee(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 3. Check if timesheet service is available
	if h.timesheetService == nil {
		response.InternalServerError(c, "Timesheet service not available")
		return
	}

	// 4. Initialize filters
	employeeID := uint(id)
	filters := domain.TimesheetFilters{
		EmployeeID: &employeeID,
		SortBy:     "created_at",
		SortOrder:  "desc",
		Limit:      10,
		Offset:     0,
	}

	// 5. Handle pagination
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 10
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	// 6. Handle sorting with validation
	if sortBy := c.Query("sortBy"); sortBy != "" {
		allowedSortFields := []string{"created_at", "date", "amount", "payment_status", "updated_at"}
		validSortBy := false
		for _, field := range allowedSortFields {
			if sortBy == field {
				validSortBy = true
				break
			}
		}
		if validSortBy {
			filters.SortBy = sortBy
		}
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		if sortOrder == "asc" || sortOrder == "desc" || sortOrder == "ASC" || sortOrder == "DESC" {
			filters.SortOrder = sortOrder
		}
	}

	// 7. Handle status filtering
	if status := c.Query("status"); status != "" {
		validStatuses := map[string]domain.PaymentStatus{
			"pending":   domain.PaymentStatusPending,
			"paid":      domain.PaymentStatusPaid,
			"failed":    domain.PaymentStatusFailed,
			"cancelled": domain.PaymentStatusCancelled,
		}
		if paymentStatus, ok := validStatuses[status]; ok {
			filters.PaymentStatus = []domain.PaymentStatus{paymentStatus}
		} else {
			response.BadRequest(c, constants.MsgInvalidPayrollStatusVN)
			return
		}
	}

	// 8. Handle date range filtering
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if parsed, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if parsed, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
	}

	// 9. Fetch timesheets with payroll information
	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrievePayrollHistoryVN)
		return
	}

	// 10. Get total count
	total, err := h.timesheetService.CountTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	// 11. Convert to response DTOs
	var payrollItems []dto.EmployeePayrollHistoryItem
	for _, timesheet := range timesheets {
		// Get project name from preloaded relationship
		projectName := ""
		if timesheet.Project.ID != 0 {
			projectName = timesheet.Project.Name
		}

		// Format payment date if exists
		var paymentDateStr *string
		if timesheet.PaymentDate != nil {
			dateStr := timesheet.PaymentDate.Format("2006-01-02")
			paymentDateStr = &dateStr
		}

		payrollItem := dto.EmployeePayrollHistoryItem{
			ID:               timesheet.ID,
			TimesheetID:      timesheet.ID,
			ProjectID:        timesheet.ProjectID,
			ProjectName:      projectName,
			Date:             timesheet.Date.Format("2006-01-02"),
			HoursWorked:      utils.RoundToTwoDecimals(timesheet.HoursWorked),
			PayType:          timesheet.PayType,
			PayRate:          utils.RoundToTwoDecimals(float64(timesheet.PayRate)),
			Amount:           utils.RoundToTwoDecimals(float64(timesheet.Amount)),
			PaymentStatus:    string(timesheet.PaymentStatus),
			PaymentReference: timesheet.PaymentReference,
			PaymentDate:      paymentDateStr,
			CreatedAt:        timesheet.CreatedAt,
			PaidAmount:       utils.RoundToTwoDecimals(float64(timesheet.PaidAmount)),
			PaidAt:           timesheet.PaidAt,
		}

		payrollItems = append(payrollItems, payrollItem)
	}

	// 12. Calculate pagination
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	pagination := dto.PaginationResponse{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: total,
	}

	// 13. Return response
	response.SuccessWithPagination(c, payrollItems, "Employee payroll history retrieved successfully", response.Pagination{
		Page:         pagination.Page,
		PageSize:     pagination.PageSize,
		TotalPages:   pagination.TotalPages,
		TotalRecords: int(pagination.TotalRecords),
	})
}
