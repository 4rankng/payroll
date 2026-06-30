package dto

import (
	"time"
)

// FinancialOverviewResponse represents financial overview data for charts
type FinancialOverviewResponse struct {
	Period        string                 `json:"period"`
	Year          int                    `json:"year"`
	CurrentMonth  CurrentMonthFinancial  `json:"current_month"`
	PreviousMonth CurrentMonthFinancial  `json:"previous_month"`
	Growth        GrowthMetrics          `json:"growth"`
	ChartData     []MonthlyFinancialData `json:"chart_data"`
}

// CurrentMonthFinancial represents financial data for a specific month
type CurrentMonthFinancial struct {
	Month                  string  `json:"month"`
	TotalRevenueVND        int64   `json:"total_revenue_vnd"`
	TotalExpensesVND       int64   `json:"total_expenses_vnd"`
	NetProfitVND           int64   `json:"net_profit_vnd"`
	ProfitMarginPercentage float64 `json:"profit_margin_percentage"`
}

// GrowthMetrics represents growth percentages
type GrowthMetrics struct {
	RevenueChangePercentage float64 `json:"revenue_change_percentage"`
	ExpenseChangePercentage float64 `json:"expense_change_percentage"`
	ProfitChangePercentage  float64 `json:"profit_change_percentage"`
}

// MonthlyFinancialData represents chart data for a single month
type MonthlyFinancialData struct {
	Month       string `json:"month"`
	RevenueVND  int64  `json:"revenue_vnd"`
	ExpensesVND int64  `json:"expenses_vnd"`
}

// RecentActivitiesResponse represents recent system activities
type RecentActivitiesResponse []ActivityItem

// ActivityItem represents a single system activity in the dashboard feed
type ActivityItem struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// SystemNotificationsResponse represents system notifications
type SystemNotificationsResponse struct {
	Notifications []NotificationItem `json:"notifications"`
	UnreadCount   int                `json:"unread_count"`
}

// NotificationItem represents a single system notification
type NotificationItem struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	ActionURL *string   `json:"action_url"`
	CreatedAt time.Time `json:"created_at"`
	Icon      string    `json:"icon"`
}

// NewEmployeesResponse represents recently added employees
type NewEmployeesResponse []NewEmployeeItem

// NewEmployeeItem represents a recently added employee
type NewEmployeeItem struct {
	ID             int          `json:"id"`
	Fullname       string       `json:"fullname"`
	Email          *string      `json:"email"`
	CCCD           string       `json:"cccd"`
	DateOfBirth    string       `json:"date_of_birth"`
	CurrentProject *ProjectInfo `json:"current_project"`
	CreatedAt      time.Time    `json:"created_at"`
	CreatedByName  string       `json:"created_by_name"`
	Avatar         *string      `json:"avatar"`
}

// ProjectInfo represents basic project information
type ProjectInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// FinancialOverviewRequest represents query parameters for financial overview
type FinancialOverviewRequest struct {
	Period string `form:"period"`
	Months int    `form:"months"`
	Year   int    `form:"year"`
}

// RecentActivitiesRequest represents query parameters for activities
type RecentActivitiesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"pageSize,default=10" binding:"min=1,max=100"`
	SortBy    string `form:"sortBy,default=created_at"`
	SortOrder string `form:"sortOrder,default=desc"`
}

// SystemNotificationsRequest represents query parameters for notifications
type SystemNotificationsRequest struct {
	Page        int  `form:"page,default=1" binding:"min=1"`
	PageSize    int  `form:"pageSize,default=5" binding:"min=1,max=20"`
	IncludeRead bool `form:"include_read,default=false"`
}

// NewEmployeesRequest represents query parameters for new employees
type NewEmployeesRequest struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"pageSize,default=5" binding:"min=1,max=20"`
	Days     int `form:"days,default=30" binding:"min=1"`
}

// FinancialChartRequest represents query parameters for financial chart
type FinancialChartRequest struct {
	Period string `form:"period,default=month" binding:"oneof=day week month quarter year"`
}

// FinancialChartResponse represents financial chart data
type FinancialChartResponse []FinancialDataPoint

// FinancialDataPoint represents a single data point in the financial chart
type FinancialDataPoint struct {
	Date            string `json:"date"`
	CashVND         int64  `json:"cash_vnd"`
	ReceivableVND   int64  `json:"receivable_vnd"`
	PayableVND      int64  `json:"payable_vnd"`
	RevenueVND      int64  `json:"revenue_vnd"`
	ExpensesVND     int64  `json:"expenses_vnd"`
	ProfitVND       int64  `json:"profit_vnd"`
	ActiveEmployees int    `json:"active_employees"`
}

// DashboardSummaryRequest represents query parameters for dashboard summary
type DashboardSummaryRequest struct {
	Month *string `form:"month" binding:"omitempty" time_format:"2006-01"` // Format: YYYY-MM (e.g., "2025-04")
}

// DashboardSummaryResponse represents the dashboard summary data
type DashboardSummaryResponse struct {
	// TotalEmployees is the count of employees currently assigned to projects
	// (excludes employees without active project assignments)
	TotalEmployees int `json:"total_employees"`
	// TotalWorkingEmployees is the count of distinct employees paid in the most recent weekly period
	// (matches historical endpoint's weekly_pay.paid_employees for the latest period)
	TotalWorkingEmployees int `json:"total_working_employees"`
	// EmployeesHiredThisMonth counts distinct employees whose project assignment
	// start_date falls within the current month (based on project_employees),
	// not employees created this month.
	EmployeesHiredThisMonth     int   `json:"employees_hired_this_month"`
	TotalPaidSalary             int64 `json:"total_paid_salary"`
	PendingSalaryThisMonth      int64 `json:"pending_salary_this_month"`
	PaidSalaryThisMonth         int64 `json:"paid_salary_this_month"`
	TotalRevenue                int64 `json:"total_revenue"`
	TotalRevenueThisMonth       int64 `json:"total_revenue_this_month"`
	TotalProfit                 int64 `json:"total_profit"`
	TotalProfitThisMonth        int64 `json:"total_profit_this_month"`
	TotalWeeklySalaryEmployees  int   `json:"total_weekly_salary_employees"`
	TotalMonthlySalaryEmployees int   `json:"total_monthly_salary_employees"`
	AvgWeeklySalary             int64 `json:"avg_weekly_salary"`
	AvgMonthlySalary            int64 `json:"avg_monthly_salary"`
}

// EmployeeActivityStatsResponse represents employee login activity grouped by payment schedule
type EmployeeActivityStatsResponse struct {
	ActiveWeekly   int `json:"active_weekly"`
	ActiveMonthly  int `json:"active_monthly"`
	ActiveFlexible int `json:"active_flexible"`
	ActiveTotal    int `json:"active_total"`
}

// ActiveEmployeeUserResponse represents a user in the activity list
type ActiveEmployeeUserResponse struct {
	UserID     uint    `json:"user_id"`
	EmployeeID uint    `json:"employee_id"`
	Fullname   string  `json:"fullname"`
	Username   string  `json:"username"`
	LastLogin  *string `json:"last_login"`
}

// SalaryDistributionCycleData holds distribution data for one payment cycle type
type SalaryDistributionCycleData struct {
	Values  []int64                   `json:"values"`
	Summary SalaryDistributionSummary `json:"summary"`
}

// SalaryDistributionResponse represents salary distribution data grouped by cycle
type SalaryDistributionResponse struct {
	Weekly   *SalaryDistributionCycleData `json:"weekly,omitempty"`
	Monthly  *SalaryDistributionCycleData `json:"monthly,omitempty"`
	Flexible *SalaryDistributionCycleData `json:"flexible,omitempty"`
}

// SalaryDistributionRequest represents query parameters for salary distribution endpoint
type SalaryDistributionRequest struct {
	Month    *string `form:"month"`
	FromDate *string `form:"fromDate"`
	ToDate   *string `form:"toDate"`
}

// SalaryDistributionSummary represents statistical summary of salary distribution
type SalaryDistributionSummary struct {
	Count  int   `json:"count"`
	Mean   int64 `json:"mean"`
	Median int64 `json:"median"`
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	StdDev int64 `json:"std_dev"`
}

// SalaryDistributionMetaResponse represents metadata for salary distribution
type SalaryDistributionMetaResponse struct {
	Period SalaryDistributionPeriod `json:"period"`
}

// SalaryDistributionPeriod represents the time period for salary data
type SalaryDistributionPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ProjectionResponse represents the complete projection data
type ProjectionResponse struct {
	AsOf            string         `json:"as_of"`
	HistoricalWeeks int            `json:"historical_weeks"`
	HistoricalData  HistoricalData `json:"historical_data"`
	Projections     Projections    `json:"projections"`
	Notes           string         `json:"notes"`
}

// HistoricalData contains all historical data points
type HistoricalData struct {
	WorkingEmployees []WorkingEmployeeData `json:"working_employees"`
	SalaryData       []SalaryData          `json:"salary_data"`
	WeeklyPayData    []WeeklyPayData       `json:"weekly_pay_data"`
	RevenueData      []RevenueData         `json:"revenue_data"`
	CapitalData      []CapitalData         `json:"capital_data"`
}

// WorkingEmployeeData represents weekly working employee count
type WorkingEmployeeData struct {
	Date             string `json:"date"`
	WorkingEmployees int    `json:"working_employees"`
	PeriodStart      string `json:"period_start"`
}

// SalaryData represents bi-weekly salary data
type SalaryData struct {
	Date          string  `json:"date"`
	TotalSalary   float64 `json:"total_salary"`
	AverageSalary float64 `json:"average_salary"`
}

// WeeklyPayData represents weekly payment data
type WeeklyPayData struct {
	Date              string  `json:"date"`
	TotalWeeklyPay    float64 `json:"total_weekly_pay"`
	PaidEmployees     int     `json:"paid_employees"`
	AvgPayPerEmployee float64 `json:"avg_pay_per_employee"`
}

// RevenueData represents weekly revenue and expense data
type RevenueData struct {
	Date          string  `json:"date"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalExpenses float64 `json:"total_expenses"`
	Profit        float64 `json:"profit"`
}

// CapitalData represents capital infusion data
type CapitalData struct {
	Date         string  `json:"date"`
	TotalCapital float64 `json:"total_capital"`
}

// Projections contains all projection data
type Projections struct {
	WorkingEmployees    WorkingEmployeesProjections    `json:"working_employees"`
	WeeklyPay           WeeklyPayProjections           `json:"weekly_pay"`
	CapitalRequirements CapitalRequirementsProjections `json:"capital_requirements"`
	Profit              ProfitProjections              `json:"profit"`
	RevenueProfit       RevenueProfitProjections       `json:"revenue_profit"`
}

// WorkingEmployeesProjections contains working employee projections
type WorkingEmployeesProjections struct {
	OneWeek     ProjectionDetail `json:"1w"`
	FourWeeks   ProjectionDetail `json:"4w"`
	TwelveWeeks ProjectionDetail `json:"12w"`
}

// WeeklyPayProjections contains weekly pay projections
type WeeklyPayProjections struct {
	OneWeek     ProjectionDetail `json:"1w"`
	FourWeeks   ProjectionDetail `json:"4w"`
	TwelveWeeks ProjectionDetail `json:"12w"`
}

// CapitalRequirementsProjections contains capital requirement projections
type CapitalRequirementsProjections struct {
	OneMonth    CapitalRequirementProjection `json:"1m"`
	ThreeMonths CapitalRequirementProjection `json:"3m"`
	SixMonths   CapitalRequirementProjection `json:"6m"`
}

// ProfitProjections contains profit projections
type ProfitProjections struct {
	OneMonth    ProfitProjection `json:"1m"`
	ThreeMonths ProfitProjection `json:"3m"`
	SixMonths   ProfitProjection `json:"6m"`
}

// RevenueProfitProjections contains revenue and profit projections
type RevenueProfitProjections struct {
	OneWeek     RevenueProfitProjection `json:"1w"`
	FourWeeks   RevenueProfitProjection `json:"4w"`
	TwelveWeeks RevenueProfitProjection `json:"12w"`
}

// ProjectionDetail represents a basic projection
type ProjectionDetail struct {
	Method           string `json:"method"`
	WorkingEmployees int    `json:"working_employees,omitempty"`
	WeeklyPay        int64  `json:"weekly_pay,omitempty"`
}

// CapitalRequirementProjection represents capital requirement projection
type CapitalRequirementProjection struct {
	CapitalNeeded         int64 `json:"capital_needed"`
	AvgMonthlyPayroll     int64 `json:"avg_monthly_payroll"`
	AvgMonthlyPerEmployee int64 `json:"avg_monthly_per_employee"`
	ProjectedEmployees    int   `json:"projected_employees"`
}

// ProfitProjection represents profit projection
type ProfitProjection struct {
	EstimatedProfit  int64   `json:"estimated_profit"`
	ProfitMargin     float64 `json:"profit_margin"`
	AvgMonthlyProfit int64   `json:"avg_monthly_profit"`
	GrowthFactor     float64 `json:"growth_factor"`
}

// RevenueProfitProjection represents revenue and profit projection with metrics
type RevenueProfitProjection struct {
	Method       string            `json:"method"`
	Revenue      int64             `json:"revenue"`
	Expenses     int64             `json:"expenses"`
	Profit       int64             `json:"profit"`
	ProfitMargin float64           `json:"profit_margin"`
	Metrics      ProjectionMetrics `json:"metrics"`
}

// ProjectionMetrics represents projection accuracy metrics
type ProjectionMetrics struct {
	RevenueMAE   float64 `json:"revenue_mae"`
	RevenueRMSE  float64 `json:"revenue_rmse"`
	RevenueMAPE  float64 `json:"revenue_mape"`
	ExpensesMAE  float64 `json:"expenses_mae"`
	ExpensesRMSE float64 `json:"expenses_rmse"`
	ExpensesMAPE float64 `json:"expenses_mape"`
}

// NewProjectionResponse represents the simplified projection response format
type NewProjectionResponse struct {
	Status  string            `json:"status"`
	Data    NewProjectionData `json:"data"`
	Message string            `json:"message"`
}

// NewProjectionData contains the main projection data
type NewProjectionData struct {
	Date        string            `json:"date"`
	Historical  NewHistoricalData `json:"historical"`
	Projections NewProjections    `json:"projections"`
}

// NewHistoricalData contains simplified historical data
type NewHistoricalData struct {
	WeeklyPay []WeeklyPayHistorical `json:"weekly_pay"`
	Revenue   []RevenueHistorical   `json:"revenue"`
	Capital   []CapitalHistorical   `json:"capital"`
}

// WeeklyPayHistorical matches Python output format
type WeeklyPayHistorical struct {
	Date              string  `json:"date"`
	PaidAmount        int64   `json:"paid_amount"`
	PaidEmployees     int     `json:"paid_employees"`
	AvgPayPerEmployee float64 `json:"avg_pay_per_employee"`
}

// RevenueHistorical with running totals
type RevenueHistorical struct {
	Date          string `json:"date"`
	TotalRevenue  int64  `json:"total_revenue"`
	TotalExpenses int64  `json:"total_expenses"`
	Profit        int64  `json:"profit"`
}

// CapitalHistorical with cumulative totals
type CapitalHistorical struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

// NewProjections contains simplified projection data
type NewProjections struct {
	PaidEmployees DataProjection `json:"paid_employees"`
	WeeklyPay     DataProjection `json:"weekly_pay"`
	Capital       DataProjection `json:"capital"`
	Profit        DataProjection `json:"profit"`
	Revenue       DataProjection `json:"revenue"`
}

// DataProjection represents daily projection data (30, 60, 90 days)
type DataProjection struct {
	ThirtyDays int64 `json:"30d"`
	SixtyDays  int64 `json:"60d"`
	NinetyDays int64 `json:"90d"`
}

// HistoricalRequest represents query parameters for historical data
type HistoricalRequest struct {
	StartDate *string `form:"start_date" binding:"omitempty" time_format:"2006-01-02"`
	EndDate   *string `form:"end_date" binding:"omitempty" time_format:"2006-01-02"`
}

// MonthlyFinancialsRequest represents query parameters for monthly financial summary
type MonthlyFinancialsRequest struct {
	// Period: "3m" (last 3 months), "6m" (last 6 months), "1y" (last 12 months). Default: "3m"
	Period string `form:"period,default=3m" binding:"omitempty,oneof=3m 6m 1y"`
}

// MonthlyFinancialRow represents financial data for a single calendar month
type MonthlyFinancialRow struct {
	Month     string `json:"month"`      // e.g. "2026-01"
	PaidOut   int64  `json:"paid_out"`   // expenses
	Billed    int64  `json:"billed"`     // revenue
	FeeEarned int64  `json:"fee_earned"` // profit = billed - paid_out
}

// MonthlyFinancialsResponse is the response for the monthly financial summary endpoint
type MonthlyFinancialsResponse struct {
	Period string                `json:"period"`
	Months []MonthlyFinancialRow `json:"months"`
	Total  MonthlyFinancialRow   `json:"total"`
}

// HistoricalResponse represents historical dashboard data
type HistoricalResponse struct {
	PaidEmployees []DailyCount   `json:"paid_employees"`
	WeeklyPay     []PeriodPay    `json:"weekly_pay"`
	Revenue       []RunningTotal `json:"revenue"`
	Profit        []RunningTotal `json:"profit"`
	Capital       []RunningTotal `json:"capital"`
	Cash          []DailyAmount  `json:"cash"`
	Expenses      []DailyAmount  `json:"expenses"`
}

// HistoricalDataResponse represents the expected response format for historical data
type HistoricalDataResponse struct {
	Date       string              `json:"date"`
	Historical HistoricalDataItems `json:"historical"`
}

// HistoricalDataItems contains all historical data items
type HistoricalDataItems struct {
	WeeklyPay []WeeklyPayHistorical `json:"weekly_pay"`
	Revenue   []RevenueHistorical   `json:"revenue"`
	Capital   []CapitalHistorical   `json:"capital"`
}

// DailyCount represents a daily count value
type DailyCount struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// PeriodPay represents payroll period payment data
type PeriodPay struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// RunningTotal represents a running total value
type RunningTotal struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// DailyAmount represents a daily amount value
type DailyAmount struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// SalaryDistributionWithMetaResponse combines distribution data and its metadata
// into a single flat response object, avoiding double-nesting.
type SalaryDistributionWithMetaResponse struct {
	Weekly   *SalaryDistributionCycleData `json:"weekly,omitempty"`
	Monthly  *SalaryDistributionCycleData `json:"monthly,omitempty"`
	Flexible *SalaryDistributionCycleData `json:"flexible,omitempty"`
	Period   SalaryDistributionPeriod     `json:"period"`
}

// ProjectProfitabilityItem represents a single project's profitability data
type ProjectProfitabilityItem struct {
	Rank                uint    `json:"rank"`
	ProjectID           uint    `json:"project_id"`
	ProjectName         string  `json:"project_name"`
	ClientName          string  `json:"client_name"`
	EmployeeCount       int     `json:"employee_count"`
	TotalReceivedVND    float64 `json:"total_received_vnd"`
	TotalPayoutVND      float64 `json:"total_payout_vnd"`
	NetProfitVND        float64 `json:"net_profit_vnd"`
	ProfitMarginPercent float64 `json:"profit_margin_percent"`
	Status              string  `json:"status"`
}

// ProjectProfitabilityResponse represents the project profitability ranking response
type ProjectProfitabilityResponse struct {
	Projects   []ProjectProfitabilityItem `json:"projects"`
	TotalCount int                        `json:"total_count"`
}

// ProjectWeeklySeries holds daily cumulative profit data for a single project
type ProjectWeeklySeries struct {
	ProjectID   uint      `json:"project_id"`
	ProjectName string    `json:"project_name"`
	TotalProfit float64   `json:"total_profit"`
	Data        []float64 `json:"data"`
}

// ProjectWeeklyProfitResponse holds the daily profit chart data for all projects
type ProjectWeeklyProfitResponse struct {
	Days   []string              `json:"days"`
	Series []ProjectWeeklySeries `json:"series"`
}

// TopPaidEmployeeItem represents a single employee in the top paid list
type TopPaidEmployeeItem struct {
	Rank              int    `json:"rank"`
	EmployeeID        uint   `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	TotalPaidVND      int64  `json:"total_paid_vnd"`
	IsActive          bool   `json:"is_active"` // true if has timesheet in last 14 days
	LastTimesheetDate string `json:"last_timesheet_date,omitempty"`
}

// TopPaidEmployeesRequest represents query parameters for top paid employees
type TopPaidEmployeesRequest struct {
	Month string `form:"month"` // YYYY-MM, optional; if empty returns all-time
	Limit int    `form:"limit,default=10" binding:"min=1,max=50"`
}

// TopPaidEmployeesResponse represents the top paid employees response
type TopPaidEmployeesResponse struct {
	Employees []TopPaidEmployeeItem `json:"employees"`
	Period    string                `json:"period"` // "all_time" or "YYYY-MM"
}

// PartnerDashboardRequest represents query parameters for partner dashboard
type PartnerDashboardRequest struct {
	Month string `form:"month"` // YYYY-MM, optional; if empty uses current month for paid stats
}

// PartnerDashboardWoWChange represents week-over-week change
type PartnerDashboardWoWChange struct {
	CurrentWeek  int64   `json:"current_week"`
	PreviousWeek int64   `json:"previous_week"`
	ChangeAmount int64   `json:"change_amount"`
	ChangePct    float64 `json:"change_pct"`
}

// PartnerDashboardMoMChange represents month-over-month change
type PartnerDashboardMoMChange struct {
	CurrentMonth  int64   `json:"current_month"`
	PreviousMonth int64   `json:"previous_month"`
	ChangeAmount  int64   `json:"change_amount"`
	ChangePct     float64 `json:"change_pct"`
}

// PartnerDashboardResponse represents the partner dashboard overview
type PartnerDashboardResponse struct {
	ActiveEmployees  int                       `json:"active_employees"`   // has timesheet in last 14 days
	DroppedEmployees int                       `json:"dropped_employees"`  // had timesheet before 14 days ago, none in last 14 days
	PaidEmployees    int                       `json:"paid_employees"`     // distinct employees paid in selected period
	TotalPaidVND     int64                     `json:"total_paid_vnd"`     // total paid amount in selected period
	WoWPaidEmployees PartnerDashboardWoWChange `json:"wow_paid_employees"` // week-over-week paid employee count
	WoWPaidAmount    PartnerDashboardWoWChange `json:"wow_paid_amount"`    // week-over-week paid amount
	MoMPaidEmployees PartnerDashboardMoMChange `json:"mom_paid_employees"` // month-over-month paid employee count
	MoMPaidAmount    PartnerDashboardMoMChange `json:"mom_paid_amount"`    // month-over-month paid amount
	TopPaidEmployees []TopPaidEmployeeItem     `json:"top_paid_employees"` // top 10 by paid amount
	Period           string                    `json:"period"`             // selected month or "all_time"
}

// BankUsageItem represents bank usage stats for a single bank
type BankUsageItem struct {
	BankName      string  `json:"bank_name"`
	EmployeeCount int     `json:"employee_count"`
	TotalPaidVND  int64   `json:"total_paid_vnd"`
	TransferCount int     `json:"transfer_count"`
	Percentage    float64 `json:"percentage"` // % of total employees
}

// BankUsageRequest represents query parameters for bank usage endpoint
type BankUsageRequest struct {
	ProjectID *uint `form:"project_id"` // optional; if omitted returns overall
}

// BankUsageResponse represents the bank usage breakdown response
type BankUsageResponse struct {
	Banks          []BankUsageItem `json:"banks"`
	TotalEmployees int             `json:"total_employees"`
	TotalPaidVND   int64           `json:"total_paid_vnd"`
	TotalTransfers int             `json:"total_transfers"`
	ProjectID      *uint           `json:"project_id,omitempty"`
	ProjectName    string          `json:"project_name,omitempty"`
}

// ProjectBankUsageItem represents bank usage for a single project
type ProjectBankUsageItem struct {
	ProjectID      uint            `json:"project_id"`
	ProjectName    string          `json:"project_name"`
	Banks          []BankUsageItem `json:"banks"`
	TotalEmployees int             `json:"total_employees"`
}

// BankUsageAllProjectsResponse represents bank usage broken down by all projects
type BankUsageAllProjectsResponse struct {
	Projects []ProjectBankUsageItem `json:"projects"`
	Overall  BankUsageResponse      `json:"overall"`
}

// PartnerEmployeeDetailItem represents a single employee with contact info for the partner dashboard
type PartnerEmployeeDetailItem struct {
	EmployeeID        uint   `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	Mobile            string `json:"mobile"`
	CCCD              string `json:"cccd"`
	ProjectName       string `json:"project_name"`
	LastTimesheetDate string `json:"last_timesheet_date,omitempty"`
	TotalPaidVND      int64  `json:"total_paid_vnd"`
	LastPaidVND       int64  `json:"last_paid_vnd"`
	LastPaidDate      string `json:"last_paid_date,omitempty"`
	IsActive          bool   `json:"is_active"`
}

// PartnerEmployeeListRequest represents query parameters for the partner employee list endpoint
type PartnerEmployeeListRequest struct {
	// Type: "active" | "dropped" | "paid"
	Type  string `form:"type" binding:"required,oneof=active dropped paid"`
	Month string `form:"month"` // YYYY-MM, used for "paid" type
}

// PartnerEmployeeListResponse represents the list of employees for a given category
type PartnerEmployeeListResponse struct {
	Employees []PartnerEmployeeDetailItem `json:"employees"`
	Type      string                      `json:"type"`
	Total     int                         `json:"total"`
}

// CheckInHealthResponse holds anomaly counts across the check-in → quota → request pipeline.
type CheckInHealthResponse struct {
	// A. Check-in / Check-out
	FailedAttemptsByCategory  []FailedAttemptCategoryCount `json:"failed_attempts_by_category"`
	FailedCheckInToday        int                          `json:"failed_check_in_today"`
	FailedCheckOutToday       int                          `json:"failed_check_out_today"`
	OpenCheckedIn             int                          `json:"open_checked_in"`
	Orphaned                  int                          `json:"orphaned"`
	AutoRejectedToday         int                          `json:"auto_rejected_today"`
	CompletedZeroEarningToday int                          `json:"completed_zero_earning_today"`
	SuccessfulCheckoutsToday  int                          `json:"successful_checkouts_today"`
	// B. Quota
	QuotaInvariantDrift    int   `json:"quota_invariant_drift"`
	MissingQuotaRows       int   `json:"missing_quota_rows"`
	StaleQuotaAfterDisable int   `json:"stale_quota_after_disable"`
	QuotaSalaryThisMonth   int64 `json:"quota_salary_this_month"`
	QuotaMaxAdvThisMonth   int64 `json:"quota_max_adv_this_month"`
	// C. Advance requests
	RequestsStuckPending   int `json:"requests_stuck_pending"`
	RequestsFailedToday    int `json:"requests_failed_today"`
	RequestsCompletedToday int `json:"requests_completed_today"`
	RequestsTotalToday     int `json:"requests_total_today"`
}

// FailedAttemptCategoryCount mirrors domain.FailedAttemptCategoryCount for the DTO layer.
type FailedAttemptCategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// QuotaAnomalyRow represents a single quota anomaly record for drill-down.
type QuotaAnomalyRow struct {
	EmployeeID   uint   `json:"employee_id"`
	ProjectID    uint   `json:"project_id"`
	ForMonth     string `json:"for_month"`
	Salary       int64  `json:"salary"`
	MaxAdvAmount int64  `json:"max_adv_amount"`
	ExpectedMax  int64  `json:"expected_max"`
	Reason       string `json:"reason"` // "drift" | "missing" | "stale"
	EmployeeName string `json:"employee_name,omitempty"`
	ProjectName  string `json:"project_name,omitempty"`
}
