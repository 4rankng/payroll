package repositories

import (
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
	"context"
	"database/sql"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/infra/persistence/query_builders"

	"gorm.io/gorm"
)

// TimesheetAnalyticsRepository handles analytics and reporting operations for timesheets
type TimesheetAnalyticsRepository struct {
	db           *gorm.DB
	queryBuilder *query_builders.TimesheetQueryBuilder
	errorHandler *common.RepoErrorHandler
}

// NewTimesheetAnalyticsRepository creates a new analytics repository
func NewTimesheetAnalyticsRepository(db *gorm.DB) *TimesheetAnalyticsRepository {
	return &TimesheetAnalyticsRepository{
		db:           db,
		queryBuilder: query_builders.NewTimesheetQueryBuilder(db),
		errorHandler: common.NewRepoErrorHandler(),
	}
}

// GetSummaryStats retrieves summary statistics for timesheets matching filters.
// Collapses the original 7 separate queries into 2 round-trips:
//   - Query 1: all counts, paid totals, employee counts, and last_updated in one pass.
//   - Query 2: pending payment amount + employee count (requires different status filters).
func (r *TimesheetAnalyticsRepository) GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	var stats domain.TimesheetSummaryStats

	// Query 1: aggregate everything that shares the same base filter set.
	type mainResult struct {
		PendingApproval int       `gorm:"column:pending_approval"`
		ApprovedEntries int       `gorm:"column:approved_entries"`
		RejectedEntries int       `gorm:"column:rejected_entries"`
		PaidEntries     int       `gorm:"column:paid_entries"`
		PaidAmount      int64     `gorm:"column:paid_amount"`
		PaidEmployees   int       `gorm:"column:paid_employees"`
		TotalEmployees  int       `gorm:"column:total_employees"`
		LastUpdated     time.Time `gorm:"column:last_updated"`
	}
	var main mainResult
	err := r.queryBuilder.BuildSummaryQuery(filters).
		Select(`
			COALESCE(SUM(timesheet_status = 'pending_approval'), 0)                                                        AS pending_approval,
			COALESCE(SUM(timesheet_status = 'approved'), 0)                                                               AS approved_entries,
			COALESCE(SUM(timesheet_status = 'rejected'), 0)                                                                AS rejected_entries,
			COALESCE(SUM(payment_status = 'paid'), 0)                                                                      AS paid_entries,
			COALESCE(SUM(CASE WHEN payment_status = 'paid' THEN paid_amount ELSE 0 END), 0)                                AS paid_amount,
			COUNT(DISTINCT CASE WHEN payment_status = 'paid' THEN timesheets.employee_id END)                              AS paid_employees,
			COUNT(DISTINCT timesheets.employee_id)                                                                         AS total_employees,
			MAX(timesheets.updated_at)                                                                                     AS last_updated
		`).
		Scan(&main).Error
	if err != nil {
		return nil, err
	}

	stats.PendingApproval = main.PendingApproval
	stats.ApprovedEntries = main.ApprovedEntries
	stats.RejectedEntries = main.RejectedEntries
	stats.TotalEntries = main.PendingApproval + main.ApprovedEntries + main.RejectedEntries
	stats.PaidEntries = main.PaidEntries
	stats.PaidAmount = main.PaidAmount
	stats.PaidEmployees = main.PaidEmployees
	stats.TotalEmployees = main.TotalEmployees
	if main.LastUpdated.IsZero() {
		stats.LastUpdated = clock.Now()
	} else {
		stats.LastUpdated = main.LastUpdated
	}

	// Query 2: pending payment amount + employee count. Keep this cohort aligned
	// with payroll export: approved entries whose payment is pending or failed.
	type pendingPaymentResult struct {
		PendingPaymentAmount    int64 `gorm:"column:pending_payment_amount"`
		PendingPaymentEmployees int   `gorm:"column:pending_payment_employees"`
	}
	var pending pendingPaymentResult
	pendingPaymentFilters := domain.NewPendingPaymentTimesheetFilters()
	err = r.queryBuilder.BuildSummaryQuery(filters).
		Where("timesheet_status IN ?", pendingPaymentFilters.TimesheetStatus).
		Where("payment_status IN ?", pendingPaymentFilters.PaymentStatus).
		Select(`
			COALESCE(SUM(amount), 0)                       AS pending_payment_amount,
			COUNT(DISTINCT timesheets.employee_id)         AS pending_payment_employees
		`).
		Scan(&pending).Error
	if err != nil {
		return nil, err
	}
	stats.PendingPaymentAmount = pending.PendingPaymentAmount
	stats.PendingEmployees = pending.PendingPaymentEmployees

	return &stats, nil
}

// GetAccrualCohort returns row-level point-in-time facts for the cash-readiness
// forecast. The service reconstructs each historical cycle as it was at an
// analogous forecast timestamp using created_at and approved_at.
//
// Scoping (partner access, project, employee) is applied via BuildSummaryQuery.
// The resulting cohort is independent from the "Chờ thanh toán" summary: that
// outstanding-payment backlog is not an input to the target-Ky forecast.
//
// Payment status is intentionally NOT filtered: a paid row is still valid
// historical demand. Rejected and pending rows are included because they are
// required to estimate pending-to-approved conversion without importing the
// global outstanding-payment backlog.
func (r *TimesheetAnalyticsRepository) GetAccrualCohort(ctx context.Context, filters domain.TimesheetFilters) ([]domain.TimesheetAccrualDailyRow, error) {
	var rows []domain.TimesheetAccrualDailyRow
	activeOn := timeutil.StartOfDay(clock.NowUTC())
	err := r.queryBuilder.BuildSummaryQuery(filters).
		Where(`(
			SELECT forecast_pe.payment_schedule
			FROM project_employees forecast_pe
			WHERE forecast_pe.project_id = timesheets.project_id
			  AND forecast_pe.employee_id = timesheets.employee_id
			  AND forecast_pe.deleted_at IS NULL
			  AND (forecast_pe.last_date IS NULL OR forecast_pe.last_date >= ?)
			ORDER BY forecast_pe.created_at DESC, forecast_pe.id DESC
			LIMIT 1
		) = ?`, activeOn, domain.PaymentScheduleWeekly).
		Select(`
			DATE(timesheets.date) AS work_date,
			timesheets.created_at AS created_at,
			timesheets.approved_at AS approved_at,
			timesheets.timesheet_status AS timesheet_status,
			timesheets.employee_id AS employee_id,
			timesheets.project_id AS project_id,
			timesheets.amount AS amount
		`).
		Order("timesheets.date ASC, timesheets.created_at ASC, timesheets.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetSummaryByProject retrieves summary statistics for a specific project
func (r *TimesheetAnalyticsRepository) GetSummaryByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*domain.TimesheetSummary, error) {
	var summary domain.TimesheetSummary

	// Get totals
	var result struct {
		TotalHours    float64
		TotalAmount   float64
		EntryCount    int
		EmployeeCount int
	}

	// Get all summary statistics in a single query with multiple aggregations
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select(`
			COALESCE(SUM(hours_worked), 0) as total_hours,
			COALESCE(SUM(amount), 0) as total_amount,
			COUNT(*) as entry_count,
			COUNT(DISTINCT employee_id) as employee_count
		`).
		Where("project_id = ? AND date BETWEEN ? AND ? AND timesheet_status = ?",
			projectID, fromDate, toDate, domain.TimesheetStatusApproved).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	summary.TotalHours = result.TotalHours
	summary.TotalAmount = int64(result.TotalAmount)
	summary.EntryCount = result.EntryCount
	summary.EmployeeCount = result.EmployeeCount

	// Get hours and amount by pay type
	type PayTypeResult struct {
		PayType string  `json:"paytype"`
		Hours   float64 `json:"hours"`
		Amount  float64 `json:"amount"`
	}

	var payTypeResults []PayTypeResult
	err = r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("paytype, COALESCE(SUM(hours_worked), 0) as hours, COALESCE(SUM(amount), 0) as amount").
		Where("project_id = ? AND date BETWEEN ? AND ? AND timesheet_status = ?",
			projectID, fromDate, toDate, domain.TimesheetStatusApproved).
		Group("paytype").
		Scan(&payTypeResults).Error
	if err != nil {
		return nil, err
	}

	summary.HoursByPayType = make(map[string]float64)
	summary.AmountByPayType = make(map[string]int64)

	for _, result := range payTypeResults {
		payType := result.PayType
		summary.HoursByPayType[payType] = result.Hours
		summary.AmountByPayType[payType] = int64(result.Amount)
	}

	return &summary, nil
}

func (r *TimesheetAnalyticsRepository) GetPendingApprovalCount(ctx context.Context, startDate, endDate time.Time) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ?", domain.TimesheetStatusPendingApproval, startDate, endDate).
		Count(&count).Error

	return int(count), err
}

// GetApprovedSalaryTotal returns total approved salary amount within date range
func (r *TimesheetAnalyticsRepository) GetApprovedSalaryTotal(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ?", domain.TimesheetStatusApproved, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetTotalPaidSalary returns the total paid salary of all time
func (r *TimesheetAnalyticsRepository) GetTotalPaidSalary(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ? AND deleted_at IS NULL", domain.PaymentStatusPaid).
		Select("COALESCE(SUM(paid_amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetPendingSalaryForMonth returns pending salary for the current month
func (r *TimesheetAnalyticsRepository) GetPendingSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("date BETWEEN ?", startDate, endDate).
		Where("timesheet_status IN ?", []domain.TimesheetStatus{
			domain.TimesheetStatusApproved,
			domain.TimesheetStatusPendingApproval,
		}).
		Where("payment_status != ?", domain.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetPaidSalaryForMonth returns paid salary for timesheets with work date in the given month
func (r *TimesheetAnalyticsRepository) GetPaidSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("date BETWEEN ?", startDate, endDate).
		Select("COALESCE(SUM(paid_amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetFirstPaidTimesheetDate returns the earliest paid_at date
func (r *TimesheetAnalyticsRepository) GetFirstPaidTimesheetDate(ctx context.Context) (*time.Time, error) {
	var firstDate time.Time
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Order("paid_at ASC").
		Limit(1).
		Select("paid_at").
		Row().
		Scan(&firstDate)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &firstDate, nil
}

// GetPaidSalaryByEmployee returns total paid salary for each employee in the given period
func (r *TimesheetAnalyticsRepository) GetPaidSalaryByEmployee(ctx context.Context, startDate, endDate time.Time) (map[uint]int64, error) {
	var results []struct {
		EmployeeID uint
		TotalPaid  int64
	}

	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("paid_at BETWEEN ?", startDate, endDate).
		Select("employee_id, SUM(paid_amount) as total_paid").
		Group("employee_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	salaryMap := make(map[uint]int64)
	for _, r := range results {
		salaryMap[r.EmployeeID] = r.TotalPaid
	}
	return salaryMap, nil
}

// WeeklyPayAggRow holds a single week's aggregated pay data from SQL.
type WeeklyPayAggRow struct {
	WeekStart     string  `gorm:"column:week_start"`
	TotalPay      float64 `gorm:"column:total_pay"`
	PaidEmployees int     `gorm:"column:paid_employees"`
}

// GetWeeklyPayAggregated returns pre-aggregated weekly pay data from SQL.
// Groups by ISO week (Monday), sums paid_amount, counts distinct employees.
// Returns ~12 rows instead of fetching thousands of individual timesheets.
func (r *TimesheetAnalyticsRepository) GetWeeklyPayAggregated(ctx context.Context, startDate, endDate time.Time) ([]WeeklyPayAggRow, error) {
	var rows []WeeklyPayAggRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			DATE_FORMAT(DATE_SUB(t.paid_at, INTERVAL WEEKDAY(t.paid_at) DAY), '%Y-%m-%d') as week_start,
			COALESCE(SUM(t.paid_amount), 0) as total_pay,
			COUNT(DISTINCT t.employee_id) as paid_employees
		FROM timesheets t
		WHERE t.payment_status = 'paid'
		  AND t.paid_amount > 0
		  AND t.paid_at IS NOT NULL
		  AND t.paid_at >= ?
		  AND t.paid_at < ?
		  AND t.deleted_at IS NULL
		GROUP BY week_start
		ORDER BY week_start
	`, startDate, endDate.Add(24*time.Hour)).Scan(&rows).Error
	return rows, err
}

// DailyProjectProfitRow holds one day's profit data for a single project.
type DailyProjectProfitRow struct {
	ProjectID   uint    `gorm:"column:project_id"`
	ProjectName string  `gorm:"column:project_name"`
	DayLabel    string  `gorm:"column:day_label"`
	TotalProfit float64 `gorm:"column:total_profit"`
}

// GetDailyProfitByProject returns daily profit (revenue_receivable - paid_amount) per project.
// Groups by the timesheet work date so every disbursement day appears in the result.
func (r *TimesheetAnalyticsRepository) GetDailyProfitByProject(ctx context.Context, startDate time.Time) ([]DailyProjectProfitRow, error) {
	var rows []DailyProjectProfitRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.project_id,
			p.name AS project_name,
			DATE_FORMAT(t.date, '%Y-%m-%d') AS day_label,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS total_profit
		FROM timesheets t
		JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		  AND CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED) > 0
		  AND t.date >= ?
		GROUP BY t.project_id, p.name, day_label
		ORDER BY day_label ASC, t.project_id ASC
	`, startDate).Scan(&rows).Error
	return rows, err
}

// ProjectProfitRow holds all-time aggregated profit data for a single project.
type ProjectProfitRow struct {
	ProjectID    uint    `gorm:"column:project_id"`
	ProjectName  string  `gorm:"column:project_name"`
	ClientName   string  `gorm:"column:client_name"`
	TotalPayout  float64 `gorm:"column:total_payout"`
	TotalRevenue float64 `gorm:"column:total_revenue"`
	TotalProfit  float64 `gorm:"column:total_profit"`
}

// GetAllTimeProfitByProject aggregates revenue_receivable and paid_amount from timesheets
// grouped by project, giving accurate all-time profit figures.
func (r *TimesheetAnalyticsRepository) GetAllTimeProfitByProject(ctx context.Context) ([]ProjectProfitRow, error) {
	var rows []ProjectProfitRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.project_id,
			p.name  AS project_name,
			p.client_name AS client_name,
			COALESCE(SUM(t.paid_amount), 0)         AS total_payout,
			COALESCE(SUM(t.revenue_receivable), 0)  AS total_revenue,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS total_profit
		FROM timesheets t
		JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		GROUP BY t.project_id, p.name, p.client_name
		ORDER BY total_profit DESC
	`).Scan(&rows).Error
	return rows, err
}

// GetProfitByProjectSince aggregates timesheet profit data from a given start date.
func (r *TimesheetAnalyticsRepository) GetProfitByProjectSince(ctx context.Context, since time.Time) ([]ProjectProfitRow, error) {
	var rows []ProjectProfitRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.project_id,
			p.name        AS project_name,
			p.client_name AS client_name,
			COALESCE(SUM(t.paid_amount), 0)        AS total_payout,
			COALESCE(SUM(t.revenue_receivable), 0) AS total_revenue,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS total_profit
		FROM timesheets t
		JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		  AND CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED) > 0
		  AND t.date >= ?
		GROUP BY t.project_id, p.name, p.client_name
		ORDER BY total_profit DESC
	`, since).Scan(&rows).Error
	return rows, err
}

// MonthlyFinancialRow holds aggregated financial data for a single calendar month.
type MonthlyFinancialRow struct {
	Month     string  `gorm:"column:month"`
	PaidOut   float64 `gorm:"column:paid_out"`
	Billed    float64 `gorm:"column:billed"`
	FeeEarned float64 `gorm:"column:fee_earned"`
}

// GetMonthlyFinancials returns per-month paid_out, billed (revenue_receivable), and fee_earned
// for timesheets where revenue_receivable > paid_amount, grouped by calendar month of the work date.
func (r *TimesheetAnalyticsRepository) GetMonthlyFinancials(ctx context.Context, startDate, endDate time.Time) ([]MonthlyFinancialRow, error) {
	var rows []MonthlyFinancialRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			DATE_FORMAT(t.date, '%Y-%m') AS month,
			COALESCE(SUM(t.paid_amount), 0) AS paid_out,
			COALESCE(SUM(t.revenue_receivable), 0) AS billed,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS fee_earned
		FROM timesheets t
		WHERE t.deleted_at IS NULL
		  AND t.date >= ?
		  AND t.date < ?
		  AND CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED) > 0
		GROUP BY month
		ORDER BY month ASC
	`, startDate, endDate).Scan(&rows).Error
	return rows, err
}

// TimesheetProfitSummary holds aggregated profit data from timesheets for a date range.
type TimesheetProfitSummary struct {
	TotalRevenue int64
	TotalPayout  int64
	TotalProfit  int64
}

// GetProfitSummaryFromTimesheets returns revenue_receivable, paid_amount, and profit
// for timesheets where revenue_receivable - paid_amount > 0, within the given date range.
// Uses work date (t.date) for grouping, not payment date.
func (r *TimesheetAnalyticsRepository) GetProfitSummaryFromTimesheets(ctx context.Context, startDate, endDate time.Time) (*TimesheetProfitSummary, error) {
	type result struct {
		TotalRevenue int64 `gorm:"column:total_revenue"`
		TotalPayout  int64 `gorm:"column:total_payout"`
		TotalProfit  int64 `gorm:"column:total_profit"`
	}
	var row result
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(t.revenue_receivable), 0) AS total_revenue,
			COALESCE(SUM(t.paid_amount), 0)        AS total_payout,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS total_profit
		FROM timesheets t
		WHERE t.deleted_at IS NULL
		  AND t.date >= ?
		  AND t.date < ?
		  AND CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED) > 0
	`, startDate, endDate).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &TimesheetProfitSummary{
		TotalRevenue: row.TotalRevenue,
		TotalPayout:  row.TotalPayout,
		TotalProfit:  row.TotalProfit,
	}, nil
}

// GetAllTimeProfitSummaryFromTimesheets returns all-time revenue_receivable, paid_amount, and profit
// for timesheets where revenue_receivable - paid_amount > 0.
func (r *TimesheetAnalyticsRepository) GetAllTimeProfitSummaryFromTimesheets(ctx context.Context) (*TimesheetProfitSummary, error) {
	type result struct {
		TotalRevenue int64 `gorm:"column:total_revenue"`
		TotalPayout  int64 `gorm:"column:total_payout"`
		TotalProfit  int64 `gorm:"column:total_profit"`
	}
	var row result
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(t.revenue_receivable), 0) AS total_revenue,
			COALESCE(SUM(t.paid_amount), 0)        AS total_payout,
			COALESCE(SUM(CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED)), 0) AS total_profit
		FROM timesheets t
		WHERE t.deleted_at IS NULL
		  AND CAST(t.revenue_receivable AS SIGNED) - CAST(t.paid_amount AS SIGNED) > 0
	`).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &TimesheetProfitSummary{
		TotalRevenue: row.TotalRevenue,
		TotalPayout:  row.TotalPayout,
		TotalProfit:  row.TotalProfit,
	}, nil
}

// TopPaidEmployeeRow holds aggregated paid data for a single employee.
type TopPaidEmployeeRow struct {
	EmployeeID   uint    `gorm:"column:employee_id"`
	EmployeeName string  `gorm:"column:employee_name"`
	TotalPaid    float64 `gorm:"column:total_paid"`
	// LastTimesheetDate is the most recent timesheet date for this employee (used for active check)
	LastTimesheetDate *string `gorm:"column:last_timesheet_date"`
}

// GetTopPaidEmployees returns the top N employees by total paid amount.
// If startDate/endDate are provided, filters by timesheet work date; otherwise all-time.
func (r *TimesheetAnalyticsRepository) GetTopPaidEmployees(ctx context.Context, startDate, endDate *time.Time, limit int) ([]TopPaidEmployeeRow, error) {
	if limit <= 0 {
		limit = 10
	}

	q := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select(`
			timesheets.employee_id,
			employees.fullname AS employee_name,
			COALESCE(SUM(timesheets.paid_amount), 0) AS total_paid,
			DATE_FORMAT(MAX(timesheets.date), '%Y-%m-%d') AS last_timesheet_date
		`).
		Joins("JOIN employees ON employees.id = timesheets.employee_id AND employees.deleted_at IS NULL").
		Where("timesheets.payment_status = ?", domain.PaymentStatusPaid)

	if startDate != nil && endDate != nil {
		q = q.Where("timesheets.date >= ? AND timesheets.date < ?", *startDate, *endDate)
	}

	var rows []TopPaidEmployeeRow
	err := q.
		Group("timesheets.employee_id, employees.fullname").
		Order("total_paid DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// PartnerEmployeeStatsRow holds per-partner employee stats.
type PartnerEmployeeStatsRow struct {
	PartnerID        uint    `gorm:"column:partner_id"`
	ActiveEmployees  int     `gorm:"column:active_employees"`
	DroppedEmployees int     `gorm:"column:dropped_employees"`
	PaidEmployees    int     `gorm:"column:paid_employees"`
	TotalPaidAmount  float64 `gorm:"column:total_paid_amount"`
}

// partnerAccessCondition returns a SQL WHERE fragment + 4 args that filters
// timesheets to those whose employee or project is accessible to the given
// partner user. Mirrors ApplyPartnerAccessControl in
// query_builders/timesheet_query_builder.go (the broadest variant — used when
// both EmployeeCreatedBy and EmployeeAssignedByPartner filters are active).
//
// An employee/timesheet is accessible to the partner when ANY of:
//  1. e.created_by = partnerID                              (partner created the employee record)
//  2. EXISTS project_employees row for (employee, partner)  (partner added employee to a project they manage)
//  3. EXISTS employee_users row for (employee, partner)     (partner is directly assigned to the employee)
//  4. EXISTS project_users row for (timesheet.project, partner)  (partner has project-level access)
//
// The fragment requires the outer query to alias the timesheets table as `t`
// and the employees table as `e`. Returns 4 ? placeholders and the partnerID
// repeated 4 times.
//
// IMPORTANT: aliases used inside the EXISTS subqueries are suffixed with
// "_pacc" so they don't collide with any outer-query JOIN aliases like `pe`
// (project_employees) or `pu` (project_users) that some callers already use.
func partnerAccessCondition() (string, func(partnerID uint) []interface{}) {
	cond := `(
        e.created_by = ?
        OR EXISTS (
            SELECT 1 FROM project_employees pe_pacc
            WHERE pe_pacc.employee_id = e.id
              AND pe_pacc.created_by = ?
              AND pe_pacc.deleted_at IS NULL
        )
        OR EXISTS (
            SELECT 1 FROM employee_users eu_pacc
            WHERE eu_pacc.employee_id = e.id
              AND eu_pacc.user_id = ?
              AND eu_pacc.deleted_at IS NULL
        )
        OR EXISTS (
            SELECT 1 FROM project_users pu_pacc
            WHERE pu_pacc.project_id = t.project_id
              AND pu_pacc.user_id = ?
              AND pu_pacc.deleted_at IS NULL
        )
    )`
	args := func(partnerID uint) []interface{} {
		return []interface{}{partnerID, partnerID, partnerID, partnerID}
	}
	return cond, args
}

// GetPartnerEmployeeStats returns active, dropped, and paid employee counts for a given partner.
// Active = has a timesheet in the last 14 days.
// Dropped = had a timesheet before 14 days ago but none in the last 14 days.
//
// Scope: covers ALL employees accessible to the partner — not just those
// whose timesheets the partner created. See partnerAccessCondition() for the
// exact access semantics (matches /partner/employees and /partner/timesheets
// pages). Previously only `t.created_by = partnerID` was used, which
// under-counted whenever an admin or other user created timesheets on behalf
// of the partner's employees.
func (r *TimesheetAnalyticsRepository) GetPartnerEmployeeStats(ctx context.Context, partnerID uint, startDate, endDate *time.Time) (*PartnerEmployeeStatsRow, error) {
	cutoff := clock.Now().AddDate(0, 0, -14)

	accessCond, accessArgs := partnerAccessCondition()

	paidDateFilter := ""
	paidArgs := []interface{}{}
	if startDate != nil && endDate != nil {
		paidDateFilter = " AND t.date >= ? AND t.date < ?"
		paidArgs = append(paidArgs, *startDate, *endDate)
	}

	var result struct {
		ActiveEmployees  int64   `gorm:"column:active_employees"`
		DroppedEmployees int64   `gorm:"column:dropped_employees"`
		PaidEmployees    int64   `gorm:"column:paid_employees"`
		TotalPaidAmount  float64 `gorm:"column:total_paid_amount"`
	}

	// Single query using a CTE that materialises the "timesheets visible to
	// this partner" set once, then aggregates 4 different metrics from it.
	// Each metric only needs its own date/payment-status filter on top.
	query := `
		WITH visible_t AS (
			SELECT t.id, t.employee_id, t.date, t.payment_status, t.paid_amount, t.project_id
			FROM timesheets t
			JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
			WHERE t.deleted_at IS NULL
			  AND ` + accessCond + `
		)
		SELECT
			(SELECT COUNT(DISTINCT vt.employee_id)
			 FROM visible_t vt
			 WHERE vt.date >= ?
			) AS active_employees,
			(SELECT COUNT(DISTINCT vt.employee_id)
			 FROM visible_t vt
			 WHERE vt.date < ?
			   AND NOT EXISTS (
			     SELECT 1 FROM visible_t vt2
			     WHERE vt2.employee_id = vt.employee_id
			       AND vt2.date >= ?
			   )
			) AS dropped_employees,
			COUNT(DISTINCT CASE WHEN vt.payment_status = 'paid' THEN vt.employee_id END) AS paid_employees,
			COALESCE(SUM(CASE WHEN vt.payment_status = 'paid' THEN vt.paid_amount ELSE 0 END), 0) AS total_paid_amount
		FROM visible_t vt
		WHERE 1 = 1` + strings.ReplaceAll(paidDateFilter, "t.date", "vt.date")

	args := append([]interface{}{}, accessArgs(partnerID)...)
	args = append(args, cutoff, cutoff, cutoff)
	args = append(args, paidArgs...)
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error; err != nil {
		return nil, err
	}

	return &PartnerEmployeeStatsRow{
		PartnerID:        partnerID,
		ActiveEmployees:  int(result.ActiveEmployees),
		DroppedEmployees: int(result.DroppedEmployees),
		PaidEmployees:    int(result.PaidEmployees),
		TotalPaidAmount:  result.TotalPaidAmount,
	}, nil
}

// PartnerTopPaidEmployeeRow holds top paid employee data for a partner.
type PartnerTopPaidEmployeeRow struct {
	EmployeeID        uint    `gorm:"column:employee_id"`
	EmployeeName      string  `gorm:"column:employee_name"`
	TotalPaid         float64 `gorm:"column:total_paid"`
	LastTimesheetDate *string `gorm:"column:last_timesheet_date"`
}

// GetPartnerTopPaidEmployees returns top N employees by paid amount for a given partner.
func (r *TimesheetAnalyticsRepository) GetPartnerTopPaidEmployees(ctx context.Context, partnerID uint, startDate, endDate *time.Time, limit int) ([]PartnerTopPaidEmployeeRow, error) {
	if limit <= 0 {
		limit = 10
	}

	// Use the full partner-access scope. The access condition references aliases
	// `t` and `e`, so we alias the tables explicitly in the GORM query.
	accessCond, accessArgs := partnerAccessCondition()

	q := r.db.WithContext(ctx).
		Table("timesheets t").
		Select(`
			t.employee_id,
			e.fullname AS employee_name,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			DATE_FORMAT(MAX(t.date), '%Y-%m-%d') AS last_timesheet_date
		`).
		Joins("JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL").
		Where("t.deleted_at IS NULL AND t.payment_status = ?", domain.PaymentStatusPaid).
		Where(accessCond, accessArgs(partnerID)...)

	if startDate != nil && endDate != nil {
		q = q.Where("t.date >= ? AND t.date < ?", *startDate, *endDate)
	}

	var rows []PartnerTopPaidEmployeeRow
	err := q.
		Group("t.employee_id, e.fullname").
		Order("total_paid DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// PartnerWeeklyPaidStats holds weekly paid employee count and amount for WoW comparison.
type PartnerWeeklyPaidStats struct {
	WeekStart     string  `gorm:"column:week_start"`
	PaidEmployees int     `gorm:"column:paid_employees"`
	TotalPaid     float64 `gorm:"column:total_paid"`
}

// GetPartnerWeeklyPaidStats returns weekly paid stats for the last N weeks for a partner.
// Scope: all timesheets accessible to the partner (see partnerAccessCondition).
func (r *TimesheetAnalyticsRepository) GetPartnerWeeklyPaidStats(ctx context.Context, partnerID uint, weeks int) ([]PartnerWeeklyPaidStats, error) {
	if weeks <= 0 {
		weeks = 8
	}
	startDate := clock.Now().AddDate(0, 0, -weeks*7)
	accessCond, accessArgs := partnerAccessCondition()

	var rows []PartnerWeeklyPaidStats
	err := r.db.WithContext(ctx).
		Table("timesheets t").
		Select(`
			DATE_FORMAT(DATE_SUB(t.date, INTERVAL WEEKDAY(t.date) DAY), '%Y-%m-%d') AS week_start,
			COUNT(DISTINCT t.employee_id) AS paid_employees,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid
		`).
		Joins("JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL").
		Where("t.deleted_at IS NULL AND t.payment_status = ? AND t.date >= ?",
			domain.PaymentStatusPaid, startDate).
		Where(accessCond, accessArgs(partnerID)...).
		Group("week_start").
		Order("week_start ASC").
		Scan(&rows).Error
	return rows, err
}

// PartnerMonthlyPaidStats holds monthly paid employee count and amount for MoM comparison.
type PartnerMonthlyPaidStats struct {
	Month         string  `gorm:"column:month"`
	PaidEmployees int     `gorm:"column:paid_employees"`
	TotalPaid     float64 `gorm:"column:total_paid"`
}

// GetPartnerMonthlyPaidStats returns monthly paid stats for the last N months for a partner.
// Scope: all timesheets accessible to the partner (see partnerAccessCondition).
func (r *TimesheetAnalyticsRepository) GetPartnerMonthlyPaidStats(ctx context.Context, partnerID uint, months int) ([]PartnerMonthlyPaidStats, error) {
	if months <= 0 {
		months = 6
	}
	startDate := clock.Now().AddDate(0, -months, 0)
	accessCond, accessArgs := partnerAccessCondition()

	var rows []PartnerMonthlyPaidStats
	err := r.db.WithContext(ctx).
		Table("timesheets t").
		Select(`
			DATE_FORMAT(t.date, '%Y-%m') AS month,
			COUNT(DISTINCT t.employee_id) AS paid_employees,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid
		`).
		Joins("JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL").
		Where("t.deleted_at IS NULL AND t.payment_status = ? AND t.date >= ?",
			domain.PaymentStatusPaid, startDate).
		Where(accessCond, accessArgs(partnerID)...).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	return rows, err
}

// BankUsageRow holds bank usage stats for a single bank.
type BankUsageRow struct {
	BankName      string  `gorm:"column:bank_name"`
	EmployeeCount int     `gorm:"column:employee_count"`
	TotalPaid     float64 `gorm:"column:total_paid"`
	TransferCount int     `gorm:"column:transfer_count"`
}

// GetBankUsageOverall returns bank usage stats across all employees (active project assignments).
// Counts distinct employees per bank and their total paid amount from timesheets.
// Uses JOIN instead of EXISTS subquery for better performance.
func (r *TimesheetAnalyticsRepository) GetBankUsageOverall(ctx context.Context) ([]BankUsageRow, error) {
	var rows []BankUsageRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(b.branch_name, 'Không có ngân hàng') AS bank_name,
			COUNT(DISTINCT e.id) AS employee_count,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			COUNT(t.id) AS transfer_count
		FROM employees e
		JOIN project_employees pe ON pe.employee_id = e.id
			AND pe.deleted_at IS NULL
			AND pe.last_date IS NULL
		LEFT JOIN banks b ON b.id = e.bank_id
		LEFT JOIN timesheets t ON t.employee_id = e.id
			AND t.deleted_at IS NULL
			AND t.payment_status = 'paid'
		WHERE e.deleted_at IS NULL
		GROUP BY bank_name
		ORDER BY employee_count DESC
	`).Scan(&rows).Error
	return rows, err
}

// GetBankUsageByProject returns bank usage stats for employees in a specific project.
func (r *TimesheetAnalyticsRepository) GetBankUsageByProject(ctx context.Context, projectID uint) ([]BankUsageRow, error) {
	var rows []BankUsageRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(b.branch_name, 'Không có ngân hàng') AS bank_name,
			COUNT(DISTINCT e.id) AS employee_count,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			COUNT(t.id) AS transfer_count
		FROM project_employees pe
		JOIN employees e ON e.id = pe.employee_id AND e.deleted_at IS NULL
		LEFT JOIN banks b ON b.id = e.bank_id
		LEFT JOIN timesheets t ON t.employee_id = e.id
			AND t.project_id = pe.project_id
			AND t.deleted_at IS NULL
			AND t.payment_status = 'paid'
		WHERE pe.project_id = ?
		  AND pe.deleted_at IS NULL
		  AND pe.last_date IS NULL
		GROUP BY bank_name
		ORDER BY employee_count DESC
	`, projectID).Scan(&rows).Error
	return rows, err
}

// ProjectBankUsageRow holds bank usage for a single project.
type ProjectBankUsageRow struct {
	ProjectID     uint    `gorm:"column:project_id"`
	ProjectName   string  `gorm:"column:project_name"`
	BankName      string  `gorm:"column:bank_name"`
	EmployeeCount int     `gorm:"column:employee_count"`
	TotalPaid     float64 `gorm:"column:total_paid"`
	TransferCount int     `gorm:"column:transfer_count"`
}

// GetBankUsageAllProjects returns bank usage grouped by project and bank for all active projects.
func (r *TimesheetAnalyticsRepository) GetBankUsageAllProjects(ctx context.Context) ([]ProjectBankUsageRow, error) {
	var rows []ProjectBankUsageRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			p.id AS project_id,
			p.name AS project_name,
			COALESCE(b.branch_name, 'Không có ngân hàng') AS bank_name,
			COUNT(DISTINCT e.id) AS employee_count,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			COUNT(t.id) AS transfer_count
		FROM project_employees pe
		JOIN projects p ON p.id = pe.project_id AND p.deleted_at IS NULL
		JOIN employees e ON e.id = pe.employee_id AND e.deleted_at IS NULL
		LEFT JOIN banks b ON b.id = e.bank_id
		LEFT JOIN timesheets t ON t.employee_id = e.id
			AND t.project_id = pe.project_id
			AND t.deleted_at IS NULL
			AND t.payment_status = 'paid'
		WHERE pe.deleted_at IS NULL
		  AND pe.last_date IS NULL
		GROUP BY p.id, p.name, bank_name
		ORDER BY p.name ASC, employee_count DESC
	`).Scan(&rows).Error
	return rows, err
}

// PartnerEmployeeDetailRow holds contact-level detail for a single employee.
type PartnerEmployeeDetailRow struct {
	EmployeeID        uint    `gorm:"column:employee_id"`
	EmployeeName      string  `gorm:"column:employee_name"`
	Mobile            string  `gorm:"column:mobile"`
	CCCD              string  `gorm:"column:cccd"`
	ProjectName       string  `gorm:"column:project_name"`
	LastTimesheetDate *string `gorm:"column:last_timesheet_date"`
	TotalPaid         float64 `gorm:"column:total_paid"`
	LastPaidVND       float64 `gorm:"column:last_paid_vnd"`
	LastPaidDate      *string `gorm:"column:last_paid_date"`
}

// partnerLastPaymentCTE builds two CTEs scoped to timesheets accessible to the
// partner (see partnerAccessCondition):
//   - partner_last_paid_date: the most recent payment_date per employee
//   - global_last_paid:       sum of paid amounts on that most recent date
//
// Both CTEs originally filtered by `created_by = partnerID`, which under-
// counted whenever someone else recorded the payment. They now use the full
// partner access scope so the values match what /partner/employees shows.
//
// Returns the SQL fragment + a func that yields the 8 args (4 for each CTE's
// access condition). Callers must pass the args in this exact order at the
// start of their own arg slice.
func partnerLastPaymentCTE(partnerID uint) (string, []interface{}) {
	accessCond, accessArgs := partnerAccessCondition()
	sql := `WITH partner_last_paid_date AS (
		SELECT t.employee_id, DATE_FORMAT(MAX(t.payment_date), '%Y-%m-%d') AS last_paid_date
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		  AND t.payment_status = 'paid'
		  AND t.payment_date IS NOT NULL
		  AND ` + accessCond + `
		GROUP BY t.employee_id
	),
	global_last_paid AS (
		SELECT t.employee_id, COALESCE(SUM(t.paid_amount), 0) AS last_paid_vnd
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		JOIN (
			SELECT employee_id, MAX(payment_date) AS max_pd
			FROM timesheets
			WHERE deleted_at IS NULL AND payment_status = 'paid' AND payment_date IS NOT NULL
			GROUP BY employee_id
		) gmax ON gmax.employee_id = t.employee_id AND t.payment_date = gmax.max_pd
		WHERE t.deleted_at IS NULL
		  AND t.payment_status = 'paid'
		  AND ` + accessCond + `
		GROUP BY t.employee_id
	)`
	args := append([]interface{}{}, accessArgs(partnerID)...)
	args = append(args, accessArgs(partnerID)...)
	return sql, args
}

// GetPartnerActiveEmployees returns employees with a timesheet in the last 14 days for a partner.
// Scope: full partner access (see partnerAccessCondition).
func (r *TimesheetAnalyticsRepository) GetPartnerActiveEmployees(ctx context.Context, partnerID uint) ([]PartnerEmployeeDetailRow, error) {
	cutoff := clock.Now().AddDate(0, 0, -14)
	cteSQL, cteArgs := partnerLastPaymentCTE(partnerID)
	accessCond, accessArgs := partnerAccessCondition()

	query := cteSQL + `
		SELECT
			e.id AS employee_id,
			e.fullname AS employee_name,
			COALESCE(e.mobile, '') AS mobile,
			COALESCE(e.cccd, '') AS cccd,
			COALESCE(p.name, '') AS project_name,
			DATE_FORMAT(MAX(t.date), '%Y-%m-%d') AS last_timesheet_date,
			COALESCE(SUM(CASE WHEN t.payment_status = 'paid' THEN t.paid_amount ELSE 0 END), 0) AS total_paid,
			COALESCE(glps.last_paid_vnd, 0) AS last_paid_vnd,
			plpd.last_paid_date
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		LEFT JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL AND pe.last_date IS NULL
		LEFT JOIN projects p ON p.id = pe.project_id AND p.deleted_at IS NULL
		LEFT JOIN partner_last_paid_date plpd ON plpd.employee_id = e.id
		LEFT JOIN global_last_paid glps ON glps.employee_id = e.id
		WHERE t.deleted_at IS NULL
		  AND t.date >= ?
		  AND ` + accessCond + `
		GROUP BY e.id, e.fullname, e.mobile, e.cccd, p.name, plpd.last_paid_date, glps.last_paid_vnd
		ORDER BY e.fullname ASC
	`

	args := append([]interface{}{}, cteArgs...)
	args = append(args, cutoff)
	args = append(args, accessArgs(partnerID)...)

	var rows []PartnerEmployeeDetailRow
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

// GetPartnerDroppedEmployees returns employees who had timesheets before 14 days ago but none since.
// Scope: full partner access (see partnerAccessCondition).
func (r *TimesheetAnalyticsRepository) GetPartnerDroppedEmployees(ctx context.Context, partnerID uint) ([]PartnerEmployeeDetailRow, error) {
	cutoff := clock.Now().AddDate(0, 0, -14)
	cteSQL, cteArgs := partnerLastPaymentCTE(partnerID)
	accessCond, accessArgs := partnerAccessCondition()

	// The "no recent timesheet" check needs the same access scope — otherwise
	// we'd exclude an employee whose recent timesheet was created by someone
	// other than the partner, which would falsely include them as "dropped".
	// We wrap the recent-activity subquery in a derived table that itself
	// applies the access condition.
	query := cteSQL + `
		SELECT
			e.id AS employee_id,
			e.fullname AS employee_name,
			COALESCE(e.mobile, '') AS mobile,
			COALESCE(e.cccd, '') AS cccd,
			COALESCE(p.name, '') AS project_name,
			DATE_FORMAT(MAX(t.date), '%Y-%m-%d') AS last_timesheet_date,
			COALESCE(SUM(CASE WHEN t.payment_status = 'paid' THEN t.paid_amount ELSE 0 END), 0) AS total_paid,
			COALESCE(glps.last_paid_vnd, 0) AS last_paid_vnd,
			plpd.last_paid_date
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		LEFT JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL AND pe.last_date IS NULL
		LEFT JOIN projects p ON p.id = pe.project_id AND p.deleted_at IS NULL
		LEFT JOIN partner_last_paid_date plpd ON plpd.employee_id = e.id
		LEFT JOIN global_last_paid glps ON glps.employee_id = e.id
		WHERE t.deleted_at IS NULL
		  AND t.date < ?
		  AND ` + accessCond + `
		  AND NOT EXISTS (
		    SELECT 1 FROM timesheets t2
		    JOIN employees e2 ON e2.id = t2.employee_id AND e2.deleted_at IS NULL
		    WHERE t2.deleted_at IS NULL
		      AND t2.employee_id = e.id
		      AND t2.date >= ?
		      AND (
		        e2.created_by = ?
		        OR EXISTS (SELECT 1 FROM project_employees pe2 WHERE pe2.employee_id = e2.id AND pe2.created_by = ? AND pe2.deleted_at IS NULL)
		        OR EXISTS (SELECT 1 FROM employee_users eu2 WHERE eu2.employee_id = e2.id AND eu2.user_id = ? AND eu2.deleted_at IS NULL)
		        OR EXISTS (SELECT 1 FROM project_users pu2 WHERE pu2.project_id = t2.project_id AND pu2.user_id = ? AND pu2.deleted_at IS NULL)
		      )
		  )
		GROUP BY e.id, e.fullname, e.mobile, e.cccd, p.name, plpd.last_paid_date, glps.last_paid_vnd
		ORDER BY last_timesheet_date DESC
	`

	args := append([]interface{}{}, cteArgs...)
	args = append(args, cutoff)
	args = append(args, accessArgs(partnerID)...)
	args = append(args, cutoff, partnerID, partnerID, partnerID, partnerID)

	var rows []PartnerEmployeeDetailRow
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

// GetPartnerPaidEmployees returns employees who were paid in the given period for a partner.
// Scope: full partner access (see partnerAccessCondition).
func (r *TimesheetAnalyticsRepository) GetPartnerPaidEmployees(ctx context.Context, partnerID uint, startDate, endDate *time.Time) ([]PartnerEmployeeDetailRow, error) {
	cteSQL, cteArgs := partnerLastPaymentCTE(partnerID)
	accessCond, accessArgs := partnerAccessCondition()

	query := cteSQL + `
		SELECT
			e.id AS employee_id,
			e.fullname AS employee_name,
			COALESCE(e.mobile, '') AS mobile,
			COALESCE(e.cccd, '') AS cccd,
			COALESCE(p.name, '') AS project_name,
			DATE_FORMAT(MAX(t.date), '%Y-%m-%d') AS last_timesheet_date,
			COALESCE(SUM(t.paid_amount), 0) AS total_paid,
			COALESCE(glps.last_paid_vnd, 0) AS last_paid_vnd,
			plpd.last_paid_date
		FROM timesheets t
		JOIN employees e ON e.id = t.employee_id AND e.deleted_at IS NULL
		LEFT JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL AND pe.last_date IS NULL
		LEFT JOIN projects p ON p.id = pe.project_id AND p.deleted_at IS NULL
		LEFT JOIN partner_last_paid_date plpd ON plpd.employee_id = e.id
		LEFT JOIN global_last_paid glps ON glps.employee_id = e.id
		WHERE t.deleted_at IS NULL
		  AND t.payment_status = 'paid'
		  AND ` + accessCond
	args := append([]interface{}{}, cteArgs...)
	args = append(args, accessArgs(partnerID)...)
	if startDate != nil && endDate != nil {
		query += " AND t.date >= ? AND t.date < ?"
		args = append(args, *startDate, *endDate)
	}
	query += `
		GROUP BY e.id, e.fullname, e.mobile, e.cccd, p.name, plpd.last_paid_date, glps.last_paid_vnd
		ORDER BY total_paid DESC
	`
	var rows []PartnerEmployeeDetailRow
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}
