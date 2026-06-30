package handlers

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/dashboard"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboardService *dashboard.Service
}

func NewDashboardHandler(dashboardService *dashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetFinancialOverview retrieves financial data for dashboard charts
// @Summary Get financial overview
// @Description Get revenue and expense data for the financial chart with month-over-month comparison
// @Tags dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period for analysis - month, quarter, year (default: month)"
// @Param months query int false "Number of months to include in chart data (default: 12, max: 24)"
// @Param year query int false "Year to analyze (default: current year)"
// @Success 200 {object} dto.FinancialOverviewResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/financial-overview [get]
func (h *DashboardHandler) GetFinancialOverview(c *gin.Context) {
	var req dto.FinancialOverviewRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	overview, err := h.dashboardService.GetFinancialOverview(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, overview, constants.MsgFinancialOverviewRetrievedVN)
}

// GetRecentActivities retrieves recent system activities
// @Summary Get recent activities
// @Description Get a feed of recent system activities for the dashboard activity section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of activities per page (default: 10, max: 100)"
// @Param sortBy query string false "Sort field (default: created_at)"
// @Param sortOrder query string false "Sort order - asc or desc (default: desc)"
// @Success 200 {object} dto.RecentActivitiesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/recent-activities [get]
func (h *DashboardHandler) GetRecentActivities(c *gin.Context) {
	var req dto.RecentActivitiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	activities, pagination, err := h.dashboardService.GetRecentActivities(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, activities, constants.MsgRecentActivitiesRetrievedVN, *pagination)
}

// GetSystemNotifications retrieves system notifications
// @Summary Get system notifications
// @Description Get system notifications with unread count for the dashboard notifications section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of notifications per page (default: 5, max: 20)"
// @Param include_read query bool false "Include read notifications (default: false)"
// @Success 200 {object} dto.SystemNotificationsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/notifications [get]
func (h *DashboardHandler) GetSystemNotifications(c *gin.Context) {
	var req dto.SystemNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	notifications, pagination, err := h.dashboardService.GetSystemNotifications(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, notifications, constants.MsgSystemNotificationsRetrievedVN, *pagination)
}

// GetNewEmployees retrieves recently added employees
// @Summary Get new employees
// @Description Get recently added employees for the dashboard new employees section
// @Tags dashboard
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of employees per page (default: 5, max: 20)"
// @Param days query int false "Number of days to look back (default: 30)"
// @Success 200 {object} dto.NewEmployeesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/new-employees [get]
func (h *DashboardHandler) GetNewEmployees(c *gin.Context) {
	var req dto.NewEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	employees, pagination, err := h.dashboardService.GetNewEmployees(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessWithPagination(c, employees, constants.MsgNewEmployeesRetrievedVN, *pagination)
}

// GetFinancialChartData retrieves financial data for dashboard charts
// @Summary Get financial chart data
// @Description Get financial data aggregated by time period for dashboard charts
// @Tags dashboard
// @Accept json
// @Produce json
// @Param period query string false "Period for aggregation - day, week, month, quarter, year (default: month)"
// @Success 200 {object} dto.FinancialChartResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/financial [get]
func (h *DashboardHandler) GetFinancialChartData(c *gin.Context) {
	var req dto.FinancialChartRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	data, err := h.dashboardService.GetFinancialChartData(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, constants.MsgFinancialOverviewRetrievedVN)
}

// GetDashboardSummary retrieves dashboard summary statistics
// ADMIN only - This endpoint provides sensitive financial and employee metrics
// @Summary Get dashboard summary
// @Description Get dashboard summary statistics including employee counts, salary information, and profit metrics (ADMIN only)
// @Tags dashboard
// @Accept json
// @Produce json
// @Param month query string false "Month in YYYY-MM format (e.g., 2025-04). Defaults to current month."
// @Success 200 {object} dto.DashboardSummaryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/summary [get]
func (h *DashboardHandler) GetDashboardSummary(c *gin.Context) {
	var req dto.DashboardSummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	summary, err := h.dashboardService.GetDashboardSummary(c.Request.Context(), req.Month)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, summary, constants.MsgDashboardSummaryRetrievedVN)
}

// GetSalaryDistribution retrieves salary distribution data for dashboard charts
// ADMIN only - This endpoint provides sensitive salary distribution information
// @Summary Get salary distribution
// @Description Get salary distribution data grouped by payment cycle (weekly, monthly, flexible) with optional date range filtering (ADMIN only)
// @Tags dashboard
// @Accept json
// @Produce json
// @Param fromDate query string false "Start date in YYYY-MM-DD format"
// @Param toDate query string false "End date in YYYY-MM-DD format"
// @Param month query string false "Month in YYYY-MM format (alternative to fromDate/toDate). Defaults to current month."
// @Success 200 {object} map[string]interface{}{ "success": bool, "data": dto.SalaryDistributionResponse, "meta": dto.SalaryDistributionMetaResponse }
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/salary-distribution [get]
func (h *DashboardHandler) GetSalaryDistribution(c *gin.Context) {
	var req dto.SalaryDistributionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	data, meta, err := h.dashboardService.GetSalaryDistribution(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, dto.SalaryDistributionWithMetaResponse{
		Weekly:   data.Weekly,
		Monthly:  data.Monthly,
		Flexible: data.Flexible,
		Period:   meta.Period,
	}, "Lấy phân phối lương thành công")
}

// GetHistoricalData retrieves historical dashboard data
// @Summary Get historical dashboard data
// @Description Get historical data for the dashboard including paid employees, weekly pay, revenue, profit, capital, and expenses
// @Tags dashboard
// @Accept json
// @Produce json
// @Param start_date query string false "Start date in YYYY-MM-DD format (optional, defaults to 14 days ago)"
// @Param end_date query string false "End date in YYYY-MM-DD format (optional, defaults to today)"
// @Success 200 {object} dto.HistoricalResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/historical [get]
func (h *DashboardHandler) GetHistoricalData(c *gin.Context) {
	var req dto.HistoricalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	data, err := h.dashboardService.GetHistoricalData(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Lấy dữ liệu thành công")
}

// GetEmployeeActivityStats retrieves employee login activity grouped by payment schedule
// @Summary Get employee activity stats
// @Description Get count of employees who logged in within the selected month, grouped by payment schedule (weekly, monthly, flexible)
// @Tags dashboard
// @Accept json
// @Produce json
// @Param month query string false "Month in yyyy-MM format (defaults to current month)"
// @Success 200 {object} dto.EmployeeActivityStatsResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/employee-activity [get]
func (h *DashboardHandler) GetEmployeeActivityStats(c *gin.Context) {
	month := c.Query("month")
	stats, err := h.dashboardService.GetEmployeeActivityStats(c.Request.Context(), month)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, stats, "Lấy thống kê hoạt động nhân viên thành công")
}

// GetActiveEmployeesBySchedule retrieves users who logged in within the selected month for a given schedule type
func (h *DashboardHandler) GetActiveEmployeesBySchedule(c *gin.Context) {
	schedule := c.Query("schedule")
	if schedule != "weekly" && schedule != "monthly" && schedule != "flexible" {
		response.BadRequest(c, "schedule phải là weekly, monthly hoặc flexible")
		return
	}
	month := c.Query("month")

	users, err := h.dashboardService.GetActiveEmployeesBySchedule(c.Request.Context(), month, schedule)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, users, "Lấy danh sách nhân viên thành công")
}

// GetProjectProfitability returns all-time profitability ranking for each project
// @Summary Get project profitability ranking
// @Description Get all projects ranked by net profit (revenue - expenses)
// @Tags dashboard
// @Produce json
// @Success 200 {object} dto.ProjectProfitabilityResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/project-profitability [get]
func (h *DashboardHandler) GetProjectProfitability(c *gin.Context) {
	data, err := h.dashboardService.GetProjectProfitability(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy xếp hạng lợi nhuận dự án thành công")
}

// GetMonthlyFinancials returns per-month revenue, expenses, and profit
// @Summary Get monthly financial summary
// @Description Get revenue, expenses, and profit aggregated by calendar month for the selected period
// @Tags dashboard
// @Produce json
// @Param period query string false "Period: 3m (last 3 months), 6m (last 6 months), 1y (last 12 months). Default: 3m"
// @Success 200 {object} dto.MonthlyFinancialsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/monthly-financials [get]
func (h *DashboardHandler) GetMonthlyFinancials(c *gin.Context) {
	var req dto.MonthlyFinancialsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}
	if req.Period == "" {
		req.Period = "3m"
	}

	data, err := h.dashboardService.GetMonthlyFinancials(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy tổng kết tài chính theo tháng thành công")
}

// GetProjectWeeklyProfit returns daily cumulative profit per project for the last N days
// @Summary Get daily cumulative profit by project
// @Description Get cumulative daily profit (revenue_receivable - paid_amount) per project
// @Tags dashboard
// @Produce json
// @Param days query int false "Number of days to look back (default: 84, max: 365)"
// @Success 200 {object} dto.ProjectWeeklyProfitResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/project-weekly-profit [get]
func (h *DashboardHandler) GetProjectWeeklyProfit(c *gin.Context) {
	days := 84 // default ~12 weeks
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	} else if w := c.Query("weeks"); w != "" {
		// backward-compat: if old client sends ?weeks=N, convert to days
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			days = parsed * 7
		}
	}
	data, err := h.dashboardService.GetProjectWeeklyProfit(c.Request.Context(), days)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy lợi nhuận theo dự án thành công")
}

// GetTopPaidEmployees returns the top N employees by total paid amount
// @Summary Get top paid employees
// @Description Get top employees ranked by total paid amount for a given month or all-time
// @Tags dashboard
// @Produce json
// @Param month query string false "Month in YYYY-MM format (optional; omit for all-time)"
// @Param limit query int false "Number of employees to return (default: 10, max: 50)"
// @Success 200 {object} dto.TopPaidEmployeesResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/top-paid-employees [get]
func (h *DashboardHandler) GetTopPaidEmployees(c *gin.Context) {
	var req dto.TopPaidEmployeesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	data, err := h.dashboardService.GetTopPaidEmployees(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy danh sách nhân viên được trả lương cao nhất thành công")
}

// GetPartnerDashboard returns the partner dashboard overview
// @Summary Get partner dashboard
// @Description Get partner overview: active employees, dropped employees, paid stats, WoW/MoM changes, top paid employees
// @Tags dashboard
// @Produce json
// @Param month query string false "Month in YYYY-MM format (defaults to current month)"
// @Success 200 {object} dto.PartnerDashboardResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/partner [get]
func (h *DashboardHandler) GetPartnerDashboard(c *gin.Context) {
	var req dto.PartnerDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	// Get the authenticated user's ID from context
	userIDVal, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}

	data, err := h.dashboardService.GetPartnerDashboard(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy tổng quan đối tác thành công")
}

// GetBankUsage returns bank usage breakdown by employee count and paid amount
// @Summary Get bank usage breakdown
// @Description Get which banks employees use, with employee count, transfer count, and paid amount. Filter by project_id for per-project view.
// @Tags dashboard
// @Produce json
// @Param project_id query int false "Filter by project ID (optional; omit for overall)"
// @Success 200 {object} dto.BankUsageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/bank-usage [get]
func (h *DashboardHandler) GetBankUsage(c *gin.Context) {
	var req dto.BankUsageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	data, err := h.dashboardService.GetBankUsage(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy thống kê ngân hàng thành công")
}

// GetBankUsageAllProjects returns bank usage broken down by all active projects plus overall
// @Summary Get bank usage for all projects
// @Description Get bank usage breakdown for every active project plus an overall summary
// @Tags dashboard
// @Produce json
// @Success 200 {object} dto.BankUsageAllProjectsResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/bank-usage/projects [get]
func (h *DashboardHandler) GetBankUsageAllProjects(c *gin.Context) {
	data, err := h.dashboardService.GetBankUsageAllProjects(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy thống kê ngân hàng theo dự án thành công")
}

// GetPartnerEmployeeList returns the employee list for a given category (active/dropped/paid)
// @Summary Get partner employee list by category
// @Description Returns employees for the authenticated partner filtered by type: active (timesheet in last 14 days), dropped (no timesheet in last 14 days), or paid (paid in selected month)
// @Tags dashboard
// @Produce json
// @Param type query string true "Employee category: active, dropped, or paid"
// @Param month query string false "Month in YYYY-MM format (used for 'paid' type)"
// @Success 200 {object} dto.PartnerEmployeeListResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/partner/employees [get]
func (h *DashboardHandler) GetPartnerEmployeeList(c *gin.Context) {
	var req dto.PartnerEmployeeListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	userIDVal, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}

	data, err := h.dashboardService.GetPartnerEmployeeList(c.Request.Context(), userID, &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Lấy danh sách nhân viên thành công")
}

// GetCheckInHealth retrieves anomaly/throughput counts for the check-in → quota →
// request pipeline. Drives the admin health summary strip.
// @Summary Get check-in health metrics
// @Description Get anomaly and throughput counts across check-in/out, quota, and advance requests
// @Tags dashboard
// @Accept json
// @Produce json
// @Param month query string false "Month in yyyy-MM format (defaults to current month)"
// @Success 200 {object} dto.CheckInHealthResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/check-in-health [get]
func (h *DashboardHandler) GetCheckInHealth(c *gin.Context) {
	month := c.Query("month")
	stats, err := h.dashboardService.GetCheckInHealth(c.Request.Context(), month)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, stats, "Lấy chỉ số sức khoẻ check-in thành công")
}

// GetQuotaAnomalies returns the drill-down rows behind a quota anomaly tile.
// @Summary Get quota anomaly drill-down
// @Description Get the specific quota rows violating the named invariant (drift|missing|stale)
// @Tags dashboard
// @Accept json
// @Produce json
// @Param type query string true "Anomaly type: drift | missing | stale"
// @Param month query string false "Month in yyyy-MM format (defaults to current month)"
// @Success 200 {array} dto.QuotaAnomalyRow
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /dashboard/quota-anomalies [get]
func (h *DashboardHandler) GetQuotaAnomalies(c *gin.Context) {
	anomalyType := c.Query("type")
	if anomalyType != "drift" && anomalyType != "missing" && anomalyType != "stale" {
		response.BadRequest(c, "type phải là drift, missing hoặc stale")
		return
	}
	month := c.Query("month")

	rows, err := h.dashboardService.GetQuotaAnomalies(c.Request.Context(), month, anomalyType)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, rows, "Lấy danh sách bất thường hạn mức thành công")
}
