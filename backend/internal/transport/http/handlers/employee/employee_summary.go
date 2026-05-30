package employee

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetEmployeesSummary returns aggregated metrics for all employees
// @Summary Get employees summary
// @Description Returns aggregated metrics for all employees in the system
// @Tags employees
// @Accept json
// @Produce json
// @Success 200 {object} dto.EmployeesSummaryResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/summary [get]
func (h *Handler) GetEmployeesSummary(c *gin.Context) {
	// 1. Validate user context
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole := c.GetString(constants.CtxUserRole)

	// 2. Apply role-based access control
	var summary *domain.EmployeesSummary
	var err error

	if userRole == string(domain.RolePartner) {
		if uid, ok := userID.(uint); ok {
			summary, err = h.employeeService.GetEmployeesSummaryForCreator(c.Request.Context(), uid)
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	} else {
		// Admin users see all employees
		summary, err = h.employeeService.GetEmployeesSummary(c.Request.Context())
	}

	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListEmployeesVN)
		return
	}

	summaryResponse := dto.EmployeesSummaryResponse{
		TotalEmployees:          summary.TotalEmployees,
		TotalWorkingEmployees:   summary.TotalWorkingEmployees,
		EmployeesHiredThisMonth: summary.EmployeesHiredThisMonth,
		SalaryMonthToDate:       summary.SalaryMonthToDate,
		PaidMonthToDate:         summary.PaidMonthToDate,
	}

	response.Success(c, summaryResponse, "Employee summary retrieved successfully")
}

// GetEmployeeSummary retrieves summary data for a specific employee
// @Summary Get employee summary
// @Description Get summary data for a specific employee including payroll payments and earnings
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Success 200 {object} dto.EmployeeSummaryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/summary [get]
func (h *Handler) GetEmployeeSummary(c *gin.Context) {
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

	// 4. Get employee summary
	summary, err := h.employeeService.GetEmployeeSummary(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	var lastPaymentDate string
	if summary.LastPaymentDate != nil {
		lastPaymentDate = summary.LastPaymentDate.Format("2006-01-02")
	}

	summaryResponse := dto.EmployeeSummaryResponse{
		TotalPayrollPayments: summary.TotalPayrollPayments,
		TotalEarningsVND:     summary.TotalEarningsVND,
		LastPaymentDate:      lastPaymentDate,
		AvgWeeklyEarningsVND: summary.AvgWeeklyEarningsVND,
	}

	response.Success(c, summaryResponse, "Employee summary retrieved successfully")
}
