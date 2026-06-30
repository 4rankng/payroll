package persistence

import (
	"context"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type AdvancePaymentRepository struct {
	*BaseRepository
	filterBuilder *common.FilterBuilder
	errorHandler  *common.RepoErrorHandler
}

func NewAdvancePaymentRepository(db *Database) domain.AdvancePaymentRepository {
	return &AdvancePaymentRepository{
		BaseRepository: NewBaseRepository(db),
		filterBuilder:  common.NewFilterBuilder(db.DB),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

func (r *AdvancePaymentRepository) Create(ctx context.Context, ap *domain.AdvancePayment) error {
	return r.getDB(ctx).Create(ap).Error
}

func (r *AdvancePaymentRepository) Upsert(ctx context.Context, ap *domain.AdvancePayment) error {
	return r.DB.WithContext(ctx).
		Exec(`
			INSERT INTO advance_payments (project_id, employee_id, for_month, upload_date, max_adv_amount, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE
				max_adv_amount = VALUES(max_adv_amount),
				updated_at = NOW()
		`, ap.ProjectID, ap.EmployeeID, ap.ForMonth, ap.UploadDate, ap.MaxAdvAmount).Error
}

func (r *AdvancePaymentRepository) GetByID(ctx context.Context, id uint64) (*domain.AdvancePayment, error) {
	var ap domain.AdvancePayment
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("Employee").
		First(&ap, id).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "advance_payment", id)
	}

	return &ap, nil
}

func (r *AdvancePaymentRepository) GetByEmployeeAndMonth(ctx context.Context, employeeID uint64, forMonth string) ([]*domain.AdvancePayment, error) {
	var aps []*domain.AdvancePayment
	err := r.getDB(ctx).
		Where("employee_id = ? AND for_month = ?", employeeID, forMonth).
		Preload("Project").
		Find(&aps).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment")
	}

	return aps, nil
}

func (r *AdvancePaymentRepository) GetMonthsByEmployee(ctx context.Context, employeeID uint64) ([]string, error) {
	var months []string
	err := r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Distinct("for_month").
		Where("employee_id = ? AND max_adv_amount > 0", employeeID).
		Order("for_month ASC").
		Pluck("for_month", &months).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment")
	}
	return months, nil
}

func (r *AdvancePaymentRepository) SumMaxAdvByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error) {
	var result struct {
		Total uint64
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePayment{}).
		Select("COALESCE(SUM(max_adv_amount), 0) as total").
		Where("employee_id = ? AND for_month = ?", employeeID, forMonth).
		Scan(&result).Error

	return result.Total, err
}

// SumSalaryAndMaxAdvByEmployeeMonth returns the salary (100% earned) and max_adv_amount
// totals for an employee's month in a single query. Used by the polled self-check-in
// advance path so salary + cap are fetched in one round-trip instead of two.
func (r *AdvancePaymentRepository) SumSalaryAndMaxAdvByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, uint64, error) {
	var result struct {
		Salary uint64
		MaxAdv uint64
	}
	err := r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Select("COALESCE(SUM(salary), 0) as salary, COALESCE(SUM(max_adv_amount), 0) as max_adv").
		Where("employee_id = ? AND for_month = ?", employeeID, forMonth).
		Scan(&result).Error

	return result.Salary, result.MaxAdv, err
}

// SumSalaryAndMaxAdvForMonth returns the total salary (100% earned) and
// max_adv_amount across all advance_payments rows for the given month. Used by
// the health dashboard's throughput tile (B4).
func (r *AdvancePaymentRepository) SumSalaryAndMaxAdvForMonth(ctx context.Context, forMonth string) (uint64, uint64, error) {
	var result struct {
		Salary uint64
		MaxAdv uint64
	}
	err := r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Select("COALESCE(SUM(salary), 0) as salary, COALESCE(SUM(max_adv_amount), 0) as max_adv").
		Where("for_month = ?", forMonth).
		Where(`EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = advance_payments.employee_id AND pe.project_id = advance_payments.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL AND pe.check_in_enabled = 1)`).
		Scan(&result).Error
	return result.Salary, result.MaxAdv, err
}

func (r *AdvancePaymentRepository) BatchCreate(ctx context.Context, aps []*domain.AdvancePayment) error {
	if len(aps) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).CreateInBatches(aps, 100).Error
}

// BatchUpsert writes one row per (employee, project, for_month) using
// last-write-wins semantics. The flexpay file has multiple sheets that map
// to different for_months (UL → current calendar month, UL (2) → previous
// calendar month), so the same employee can appear in multiple rows but
// each lives in a distinct period. Re-uploading the same file overwrites
// rows with identical values (idempotent in effect). A different file
// targeting the same period overwrites — that's the documented behavior
// for flexpay corrections.
//
// last_applied_asset_id is stamped on every row as an informational
// "which file most recently touched this row" pointer; it is not part
// of the idempotency mechanism.
func (r *AdvancePaymentRepository) BatchUpsert(ctx context.Context, aps []*domain.AdvancePayment) error {
	if len(aps) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).
		Exec(`INSERT INTO advance_payments (project_id, employee_id, for_month, upload_date, max_adv_amount, last_applied_asset_id, created_at, updated_at)
			SELECT v.project_id, v.employee_id, v.for_month, v.upload_date, v.max_adv_amount, v.last_applied_asset_id, NOW(), NOW()
			FROM (
				SELECT ? as project_id, ? as employee_id, ? as for_month, ? as upload_date, ? as max_adv_amount, ? as last_applied_asset_id`+
			strings.Repeat(" UNION ALL SELECT ?, ?, ?, ?, ?, ?", len(aps)-1)+
			`) v
			ON DUPLICATE KEY UPDATE
				max_adv_amount = VALUES(max_adv_amount),
				last_applied_asset_id = VALUES(last_applied_asset_id),
				updated_at = NOW()`,
			flattenAdvancePayments(aps)...).Error
}

func flattenAdvancePayments(aps []*domain.AdvancePayment) []any {
	args := make([]any, 0, len(aps)*6)
	for _, ap := range aps {
		args = append(args, ap.ProjectID, ap.EmployeeID, ap.ForMonth, ap.UploadDate, ap.MaxAdvAmount, ap.LastAppliedAssetID)
	}
	return args
}

func (r *AdvancePaymentRepository) Update(ctx context.Context, ap *domain.AdvancePayment) error {
	return r.getDB(ctx).Save(ap).Error
}

// getDB returns the transaction-aware DB session.
func (r *AdvancePaymentRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

// AccumulateSalary atomically adds earning to salary AND recomputes max_adv_amount as
// floor(salary * SelfCheckInAdvanceablePercent / 100) for the self-check-in flow.
// Both expressions reference (old salary + earning) so the result is independent of
// MySQL's left-to-right SET evaluation order. earning is always >= 0 (checkout earnings).
func (r *AdvancePaymentRepository) AccumulateSalary(ctx context.Context, id uint64, earning int64) error {
	return r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"salary":         gorm.Expr("salary + ?", earning),
			"max_adv_amount": gorm.Expr("(salary + ?) * ? / 100", earning, domain.SelfCheckInAdvanceablePercent),
		}).Error
}

// ZeroOutQuota sets max_adv_amount AND salary to 0 for all advance payments of a project-employee
// pair from the given month onward. Used when disabling check-in to prevent future advances.
// Zeroing BOTH preserves the invariant max_adv_amount = floor(ratio * salary) so that a
// disable→re-enable within the same period does not leave a stale max_adv_amount=0 against a
// non-zero salary (which would silently lock the employee out of advances until the next checkout).
func (r *AdvancePaymentRepository) ZeroOutQuota(ctx context.Context, projectID, employeeID uint, currentMonth string) error {
	return r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Where("project_id = ? AND employee_id = ? AND for_month >= ?", projectID, employeeID, currentMonth).
		Updates(map[string]any{
			"max_adv_amount": 0,
			"salary":         0,
		}).Error
}

// BatchZeroOutQuota sets max_adv_amount AND salary to 0 for multiple employees in a single
// query. Used by BulkToggleCheckInEnabled to avoid N individual UPDATEs. Mirrors ZeroOutQuota:
// zeroing both fields preserves the invariant max_adv_amount = floor(ratio * salary) on
// disable→re-enable within the same period (see ZeroOutQuota for the full rationale).
func (r *AdvancePaymentRepository) BatchZeroOutQuota(ctx context.Context, projectID uint, employeeIDs []uint, currentMonth string) error {
	if len(employeeIDs) == 0 {
		return nil
	}
	return r.getDB(ctx).
		Model(&domain.AdvancePayment{}).
		Where("project_id = ? AND employee_id IN ? AND for_month >= ?", projectID, employeeIDs, currentMonth).
		Updates(map[string]any{
			"max_adv_amount": 0,
			"salary":         0,
		}).Error
}

// GetLatestForMonth returns the latest for_month derived from flexible employees' created_at
func (r *AdvancePaymentRepository) GetLatestForMonth(ctx context.Context) (string, error) {
	var result struct {
		ForMonth string `gorm:"column:for_month"`
	}

	err := r.DB.WithContext(ctx).
		Table("advance_payments").
		Select("for_month").
		Order("for_month DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		return "", r.errorHandler.HandleGetError(err, "advance_payments", "latest_for_month")
	}

	if result.ForMonth == "" {
		return "", domain.NewNotFoundError("flex pay data not found")
	}

	return result.ForMonth, nil
}

// GetEmployeeAdvanceStats returns employees with their advance payment statistics
// Uses project_employees as base table for flexible payment schedule employees
func (r *AdvancePaymentRepository) GetEmployeeAdvanceStats(ctx context.Context, filters domain.EmployeeAdvanceStatsFilters) ([]*domain.EmployeeAdvanceStats, int64, error) {
	var results []*domain.EmployeeAdvanceStats
	var total int64

	// Build base query using project_employees as the base table for flexible payment schedule.
	// Deduplicate via subquery to avoid row multiplication from duplicate entries.
	query := r.DB.WithContext(ctx).
		Table("(SELECT employee_id, MIN(project_id) as project_id, MAX(id) as project_employee_id FROM project_employees WHERE deleted_at IS NULL AND payment_schedule = ? GROUP BY employee_id) AS pe", domain.PaymentScheduleFlexible).
		Select(`
			e.id as employee_id,
			e.fullname,
			e.cccd,
			COALESCE(u.username, '') as username,
			e.email,
			e.mobile,
			e.bank_id,
			b.branch_name as bank_name,
			e.bank_account_number,
			e.bank_account_name,
			p.id as project_id,
			p.name as project_name,
			p.code as project_code,
			ap.for_month,
			COALESCE(ap.max_adv_amount, 0) as max_advance_amount,
			COALESCE(completed_stats.utilized_amount, 0) as utilized_amount,
			COALESCE(completed_stats.total_fee, 0) as total_fee_generated,
			COALESCE(pending_stats.pending_amount, 0) as pending_amount,
			COALESCE(completed_stats.completed_count, 0) as completed_requests_count,
			COALESCE(pending_stats.pending_count, 0) as pending_requests_count,
			e.created_at,
			pe_detail.id as project_employee_id,
			pe_detail.check_in_enabled
		`).
		Joins("JOIN employees e ON pe.employee_id = e.id").
		Joins("JOIN project_employees pe_detail ON pe_detail.id = pe.project_employee_id").
		Joins("JOIN projects p ON pe.project_id = p.id").
		Joins("LEFT JOIN banks b ON e.bank_id = b.id").
		Joins("LEFT JOIN users u ON e.user_id = u.id").
		Joins(`LEFT JOIN (
			SELECT employee_id, project_id, for_month,
				SUM(max_adv_amount) AS max_adv_amount
			FROM advance_payments
			GROUP BY employee_id, project_id, for_month
		) ap ON ap.employee_id = pe.employee_id
			AND ap.project_id = pe.project_id
			AND ap.for_month = ?`, *filters.ForMonth).
		Joins(`LEFT JOIN (
			SELECT
				apr.employee_id,
				ap2.for_month,
				apr.project_id,
				SUM(apr.request_amount) as utilized_amount,
				SUM(apr.fee) as total_fee,
				COUNT(*) as completed_count
			FROM advance_payment_requests apr
			JOIN advance_payments ap2 ON apr.adv_pay_id = ap2.id
			WHERE apr.status = ?
			GROUP BY apr.employee_id, ap2.for_month, apr.project_id
		) completed_stats ON completed_stats.employee_id = pe.employee_id
			AND completed_stats.project_id = pe.project_id
			AND completed_stats.for_month = ?`, domain.AdvancePaymentStatusCompleted, *filters.ForMonth).
		Joins(`LEFT JOIN (
			SELECT
				apr.employee_id,
				ap2.for_month,
				apr.project_id,
				SUM(apr.request_amount) as pending_amount,
				COUNT(*) as pending_count
			FROM advance_payment_requests apr
			JOIN advance_payments ap2 ON apr.adv_pay_id = ap2.id
			WHERE apr.status = ?
			GROUP BY apr.employee_id, ap2.for_month, apr.project_id
		) pending_stats ON pending_stats.employee_id = pe.employee_id
			AND pending_stats.project_id = pe.project_id
			AND pending_stats.for_month = ?`, domain.AdvancePaymentStatusPending, *filters.ForMonth)
	// Apply filters
	if filters.ForMonth != nil {
		query = query.Where("ap.for_month IS NOT NULL")
	}
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		query = query.Where("e.fullname LIKE ? OR e.cccd LIKE ?", searchTerm, searchTerm)
	}

	// Count total using a separate query
	countQuery := r.DB.WithContext(ctx).
		Table("(SELECT employee_id, MIN(project_id) as project_id, MAX(id) as project_employee_id FROM project_employees WHERE deleted_at IS NULL AND payment_schedule = ? GROUP BY employee_id) AS pe", domain.PaymentScheduleFlexible).
		Joins("JOIN employees e ON pe.employee_id = e.id").
		Joins("JOIN project_employees pe_detail ON pe_detail.id = pe.project_employee_id").
		Joins("JOIN projects p ON pe.project_id = p.id")
	if filters.ForMonth != nil {
		countQuery = countQuery.Joins(`JOIN (
			SELECT employee_id, project_id FROM advance_payments WHERE for_month = ? GROUP BY employee_id, project_id
		) ap_filter ON ap_filter.employee_id = pe.employee_id AND ap_filter.project_id = pe.project_id`, *filters.ForMonth)
	}
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		countQuery = countQuery.Where("e.fullname LIKE ? OR e.cccd LIKE ?", searchTerm, searchTerm)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "project_employees")
	}

	// Apply sorting, then add a deterministic tiebreaker for stable pagination.
	// "available_amount" maps to a trusted SQL expression, which cannot go through
	// ApplySorting (that helper enforces an identifier shape to prevent SQL
	// injection), so apply it directly with a sanitized direction. All other
	// sorts use the sanitized ApplySorting helper.
	if filters.SortBy == "available_amount" {
		expr := "GREATEST(0, CAST(COALESCE(ap.max_adv_amount, 0) AS SIGNED) - CAST(COALESCE(completed_stats.utilized_amount, 0) AS SIGNED) - CAST(COALESCE(pending_stats.pending_amount, 0) AS SIGNED))"
		query = query.Order(expr + " " + common.SanitizeSortOrder(filters.SortOrder, "DESC"))
	} else {
		query = r.filterBuilder.ApplySorting(query, filters.SortBy, filters.SortOrder, "e.fullname")
	}
	query = query.Order("e.id ASC")
	query = r.filterBuilder.ApplyPagination(query, filters.Limit, filters.Offset)

	err := query.Scan(&results).Error
	if err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "project_employees")
	}

	return results, total, nil
}

// GetAvailableMonths returns all months with flexible employees, sorted descending
func (r *AdvancePaymentRepository) GetAvailableMonths(ctx context.Context) ([]*domain.AvailableMonth, error) {
	var results []*domain.AvailableMonth

	err := r.DB.WithContext(ctx).
		Table("advance_payments").
		Select("for_month, COUNT(DISTINCT employee_id) as employee_count").
		Group("for_month").
		Order("for_month DESC").
		Scan(&results).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payments")
	}

	return results, nil
}

// GetEmployeeByID returns an employee by ID
func (r *AdvancePaymentRepository) GetEmployeeByID(ctx context.Context, employeeID uint64) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.DB.WithContext(ctx).First(&employee, employeeID).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "employee", employeeID)
	}

	return &employee, nil
}

func (r *AdvancePaymentRepository) HasDataForMonth(ctx context.Context, forMonth string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePayment{}).
		Where("for_month = ?", forMonth).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, r.errorHandler.HandleGetError(err, "advance_payment", "has_data_for_month")
	}
	return count > 0, nil
}

// quotaAnomalySQL returns the Raw SQL query and bind args for each anomaly type.
// The queries reuse the predicates documented on
// AdvancePaymentRepository.GetQuotaAnomalies in the domain interface.
func (r *AdvancePaymentRepository) quotaAnomalySQL(forMonth, anomalyType string, startOfMonth, endOfNextMonth time.Time) (string, []any) {
	switch anomalyType {
	case "drift":
		// max_adv_amount invariant: must equal floor(salary * 70 / 100) when salary>0.
		return `
			SELECT ap.employee_id, ap.project_id, ap.for_month, ap.salary, ap.max_adv_amount, 'drift' as reason
			FROM advance_payments ap
			WHERE ap.for_month = ? AND ap.salary > 0
			  AND ap.max_adv_amount != FLOOR(ap.salary * ? / 100)
				  AND EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = ap.employee_id AND pe.project_id = ap.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL AND pe.check_in_enabled = 1)
		`, []any{forMonth, domain.SelfCheckInAdvanceablePercent}
	case "missing":
		// Earning>0 attendance this month but no advance_payments row for the pair.
		return `
			SELECT a.employee_id, a.project_id, ? as for_month, 0 as salary, 0 as max_adv_amount, 'missing' as reason
			FROM (
				SELECT DISTINCT employee_id, project_id
				FROM attendances
				WHERE check_out_time IS NOT NULL AND earning_amount > 0
				  AND check_in_time >= ? AND check_in_time < ?
					  AND EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = attendances.employee_id AND pe.project_id = attendances.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL AND pe.check_in_enabled = 1)
			) a
				LEFT JOIN advance_payments ap
			  ON ap.project_id = a.project_id AND ap.employee_id = a.employee_id AND ap.for_month = ?
			WHERE ap.id IS NULL
		`, []any{forMonth, startOfMonth, endOfNextMonth, forMonth}
	case "stale":
		// salary>0 but the assignment has check_in_enabled=false (ZeroOutQuota didn't fire on toggle-off).
		return `
			SELECT ap.employee_id, ap.project_id, ap.for_month, ap.salary, ap.max_adv_amount, 'stale' as reason
			FROM advance_payments ap
			JOIN project_employees pe ON pe.employee_id = ap.employee_id AND pe.project_id = ap.project_id
			WHERE ap.for_month = ? AND ap.salary > 0 AND pe.check_in_enabled = false
			  AND pe.deleted_at IS NULL AND pe.last_date IS NULL
		`, []any{forMonth}
	default:
		return "", nil
	}
}

// GetQuotaAnomalies returns quota rows violating the named invariant. See
// domain.AdvancePaymentRepository.GetQuotaAnomalies for the per-type contract.
func (r *AdvancePaymentRepository) GetQuotaAnomalies(ctx context.Context, forMonth, anomalyType string) ([]domain.QuotaAnomaly, error) {
	startOfMonth, endOfNextMonth, err := quotaAnomalyMonthBounds(forMonth)
	if err != nil {
		return nil, domain.NewValidationError("định dạng tháng không hợp lệ (YYYY-MM)")
	}
	sql, args := r.quotaAnomalySQL(forMonth, anomalyType, startOfMonth, endOfNextMonth)
	if sql == "" {
		return nil, domain.NewValidationError("loại bất thường không hợp lệ (drift|missing|stale)")
	}

	var rows []domain.QuotaAnomaly
	err = r.DB.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment_quota_anomalies")
	}
	return rows, nil
}

// CountQuotaAnomalies returns the count of rows violating the named invariant.
func (r *AdvancePaymentRepository) CountQuotaAnomalies(ctx context.Context, forMonth, anomalyType string) (int, error) {
	startOfMonth, endOfNextMonth, err := quotaAnomalyMonthBounds(forMonth)
	if err != nil {
		return 0, domain.NewValidationError("định dạng tháng không hợp lệ (YYYY-MM)")
	}
	sql, args := r.quotaAnomalySQL(forMonth, anomalyType, startOfMonth, endOfNextMonth)
	if sql == "" {
		return 0, domain.NewValidationError("loại bất thường không hợp lệ (drift|missing|stale)")
	}

	var count int64
	err = r.DB.WithContext(ctx).Raw("SELECT COUNT(*) FROM ("+sql+") AS sub", args...).Scan(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleGetError(err, "advance_payment_quota_anomalies", "count")
	}
	return int(count), nil
}

// quotaAnomalyMonthBounds converts a "YYYY-MM" string into the [start, end) window
// covering that calendar month. The "missing" anomaly query uses this to bound
// attendance lookups. The end bound is exclusive at the start of the next month.
// Returns an error if forMonth is not a valid "YYYY-MM" value.
func quotaAnomalyMonthBounds(forMonth string) (time.Time, time.Time, error) {
	t, err := time.ParseInLocation("2006-01", forMonth, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	loc := t.Location()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}
