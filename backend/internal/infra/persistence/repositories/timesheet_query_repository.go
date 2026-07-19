package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/infra/persistence/query_builders"
	"api-server/internal/infra/persistence/relationship_loaders"

	"gorm.io/gorm"
)

// TimesheetQueryRepository handles read operations for timesheets
type TimesheetQueryRepository struct {
	db             *gorm.DB
	queryBuilder   *query_builders.TimesheetQueryBuilder
	relLoader      *relationship_loaders.TimesheetRelationshipLoader
	batchProcessor *common.BatchProcessor
	errorHandler   *common.RepoErrorHandler
}

// NewTimesheetQueryRepository creates a new query repository
func NewTimesheetQueryRepository(db *gorm.DB) *TimesheetQueryRepository {
	return &TimesheetQueryRepository{
		db:             db,
		queryBuilder:   query_builders.NewTimesheetQueryBuilder(db),
		relLoader:      relationship_loaders.NewTimesheetRelationshipLoader(db),
		batchProcessor: common.NewBatchProcessor(common.DefaultBatchConfig()),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

// GetByID retrieves a timesheet by ID with relationships
func (r *TimesheetQueryRepository) GetByID(ctx context.Context, id uint) (*domain.Timesheet, error) {
	var timesheet domain.Timesheet

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&timesheet).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "timesheet", id)
	}

	// Load relationships for single timesheet
	err = r.relLoader.LoadSingleTimesheetRelationships(ctx, &timesheet)
	if err != nil {
		return nil, err
	}

	return &timesheet, nil
}

// GetByIDs retrieves multiple timesheets by IDs with efficient relationship loading
func (r *TimesheetQueryRepository) GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	return r.getByIDsInternal(ctx, ids, true)
}

// GetByIDsWithoutRelations retrieves multiple timesheets by IDs without loading relationships
// This is optimized for validation-only operations where relationships are not needed
func (r *TimesheetQueryRepository) GetByIDsWithoutRelations(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	return r.getByIDsInternal(ctx, ids, false)
}

// GetTimesheetDatesByIDs returns id → date for the given IDs without loading full
// rows or any relationships. This is the lightest possible lookup for callers that
// only need the timesheet date — e.g. bank-transfer-histories cycle resolution,
// which previously loaded ~5,000 full rows + relations just to read Date.Year/Month/Day.
//
// Uses the BatchProcessor to stay under MySQL's packet limit. Ordering is not
// applied because callers consume the result via map lookup.
func (r *TimesheetQueryRepository) GetTimesheetDatesByIDs(ctx context.Context, ids []uint) (map[uint]time.Time, error) {
	result := make(map[uint]time.Time, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	type idDate struct {
		ID   uint      `gorm:"column:id"`
		Date time.Time `gorm:"column:date"`
	}

	err := r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		var rows []idDate
		if err := r.db.WithContext(ctx).
			Table("timesheets").
			Select("id, date").
			Where("id IN ?", batchIDs).
			Where("deleted_at IS NULL").
			Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			result[row.ID] = row.Date
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// getByIDsInternal is the shared implementation for GetByIDs methods
func (r *TimesheetQueryRepository) getByIDsInternal(ctx context.Context, ids []uint, includeRelations bool) ([]*domain.Timesheet, error) {
	if len(ids) == 0 {
		return []*domain.Timesheet{}, nil
	}

	var timesheets []*domain.Timesheet

	// Use BatchProcessor to fetch timesheets in batches
	err := r.batchProcessor.ProcessInBatches(ctx, ids, func(batch interface{}) error {
		batchIDs := batch.([]uint)
		var batchTimesheets []*domain.Timesheet
		query := r.db.WithContext(ctx).
			Where("id IN ?", batchIDs).
			Order("date DESC")
		// When relations aren't needed (validation-only path), project only the
		// columns the bulk-update validator reads. Avoids SELECT * on the wide
		// timesheets row (multiple text/json/datetime columns). ck:debug 2026-07-04.
		if !includeRelations {
			query = query.Select("id, project_id, employee_id, date, hours_worked, timesheet_status, payment_status, amount, paid_amount, paid_at, approved_by")
		}
		err := query.Find(&batchTimesheets).Error

		if err != nil {
			return err
		}

		timesheets = append(timesheets, batchTimesheets...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Load relationships for all timesheets (if requested)
	err = r.relLoader.LoadTimesheetRelationships(ctx, timesheets, includeRelations)
	if err != nil {
		return nil, err
	}

	return timesheets, nil
}

// List retrieves timesheets based on filters with optional relationship loading
func (r *TimesheetQueryRepository) List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	query := r.queryBuilder.BuildListQuery(filters)

	// Apply sorting with timesheet-specific logic
	sortBy := "timesheets.date"
	if filters.SortBy != "" {
		if filters.SortBy == "date" {
			sortBy = "timesheets.date"
		} else {
			sortBy = "timesheets." + filters.SortBy
		}
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "timesheets.date"), common.SanitizeSortOrder(sortOrder, "DESC")))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&timesheets).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "timesheet")
	}

	// Load relationships if not skipped
	if !filters.SkipRelations {
		err = r.relLoader.LoadTimesheetRelationships(ctx, timesheets, true)
		if err != nil {
			return nil, err
		}
	}

	return timesheets, nil
}

// Count returns the count of timesheets matching filters
func (r *TimesheetQueryRepository) Count(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	var count int64
	query := r.queryBuilder.BuildCountQuery(filters)
	err := query.Count(&count).Error
	return count, err
}

// GetByProjectAndEmployee retrieves timesheets for a specific project and employee within date range
func (r *TimesheetQueryRepository) GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	query := r.queryBuilder.BuildProjectEmployeeRangeQuery(projectID, employeeID, fromDate, toDate).
		Preload("Project").
		Preload("Employee").
		Preload("CreatedUser").
		Preload("ApprovedUser").
		Order("timesheets.date DESC")

	err := query.Find(&timesheets).Error
	return timesheets, err
}

// GetByProject retrieves timesheets for a project within date range
func (r *TimesheetQueryRepository) GetByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	query := r.queryBuilder.BuildProjectRangeQuery(projectID, fromDate, toDate).
		Preload("Project").
		Preload("Employee").
		Preload("CreatedUser").
		Preload("ApprovedUser").
		Order("timesheets.date DESC, timesheets.employee_id ASC")

	err := query.Find(&timesheets).Error
	return timesheets, err
}

// GetByEmployee retrieves timesheets for an employee within date range
func (r *TimesheetQueryRepository) GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	query := r.queryBuilder.BuildEmployeeRangeQuery(employeeID, fromDate, toDate).
		Preload("Project").
		Preload("Employee").
		Preload("CreatedUser").
		Preload("ApprovedUser").
		Order("timesheets.date DESC")

	err := query.Find(&timesheets).Error
	return timesheets, err
}

// GetByEmployeeAndPeriod retrieves timesheets for an employee within a date range (alias for GetByEmployee)
func (r *TimesheetQueryRepository) GetByEmployeeAndPeriod(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return r.GetByEmployee(ctx, employeeID, fromDate, toDate)
}

// GetByProjectEmployeeDatePaytype retrieves a specific timesheet by project, employee, date, and paytype
func (r *TimesheetQueryRepository) GetByProjectEmployeeDatePaytype(ctx context.Context, projectID, employeeID uint, date time.Time, paytype string) (*domain.Timesheet, error) {
	var timesheet domain.Timesheet

	query := r.queryBuilder.BuildProjectEmployeeDateQuery(projectID, employeeID, date, paytype)
	err := query.First(&timesheet).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found is not an error in this case
		}
		return nil, err
	}

	return &timesheet, nil
}

// GetByProjectEmployeeDateHourType retrieves a specific timesheet by project, employee, date, and hour type
func (r *TimesheetQueryRepository) GetByProjectEmployeeDateHourType(ctx context.Context, projectID, employeeID uint, date time.Time, hourType string) (*domain.Timesheet, error) {
	var timesheet domain.Timesheet

	query := r.queryBuilder.BuildProjectEmployeeDateHourTypeQuery(projectID, employeeID, date, hourType)
	err := query.First(&timesheet).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found is not an error in this case
		}
		return nil, err
	}

	return &timesheet, nil
}

// dayBoundsInDateLocation returns the half-open [start, end) 24-hour window for the
// calendar day of `date`, constructed in the same location timesheet dates are
// stored/parsed (time.Local).
//
// Building this window in time.UTC is a recurring bug: under a loc=Local DSN the
// go-sql-driver converts the bound time.Time values to Local before sending, which
// shifts the window +7h on the UTC+7 prod box — so a lookup for day D misses D's
// own rows (stored at local midnight) and instead returns day D+1's. That broke
// bulk upsert/delete silently (GetByEmployeeDateCombos) and made the daily 24-hour
// validation false-reject saves (GetByProjectEmployeeDate): e.g. employee 661,
// 2026-06-21 — UTC bounds returned 2026-06-22's 12.0h, so a 12.5h save read as
// 12.5 + 12.0 = 24.5h and was rejected with "Tổng giờ làm việc ngày ... vượt quá
// 24 giờ". The UTC→Local fallback keeps callers that hand in a UTC time safe.
func dayBoundsInDateLocation(date time.Time) (time.Time, time.Time) {
	loc := date.Location()
	if loc == time.UTC {
		loc = time.Local
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	return startOfDay, startOfDay.Add(24 * time.Hour)
}

// GetByProjectEmployeeDate retrieves all timesheets for a specific project, employee, and date
func (r *TimesheetQueryRepository) GetByProjectEmployeeDate(ctx context.Context, projectID, employeeID uint, date time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	startOfDay, endOfDay := dayBoundsInDateLocation(date)
	query := r.db.WithContext(ctx).Where("project_id = ? AND employee_id = ? AND date >= ? AND date < ?", projectID, employeeID, startOfDay, endOfDay)

	err := query.Find(&timesheets).Error
	if err != nil {
		return nil, err
	}

	return timesheets, nil
}

// GetByEmployeeDateCombos retrieves all timesheets for multiple employee/project/date combinations in a single query
func (r *TimesheetQueryRepository) GetByEmployeeDateCombos(ctx context.Context, combos []domain.EmployeeDateCombo) ([]*domain.Timesheet, error) {
	if len(combos) == 0 {
		return []*domain.Timesheet{}, nil
	}

	var timesheets []*domain.Timesheet

	// Build the query with OR conditions for each combo
	query := r.db.WithContext(ctx).Model(&domain.Timesheet{})

	for i, combo := range combos {
		startOfDay, endOfDay := dayBoundsInDateLocation(combo.Date)
		if i == 0 {
			query = query.Where("(employee_id = ? AND project_id = ? AND date >= ? AND date < ?)",
				combo.EmployeeID, combo.ProjectID, startOfDay, endOfDay)
		} else {
			query = query.Or("(employee_id = ? AND project_id = ? AND date >= ? AND date < ?)",
				combo.EmployeeID, combo.ProjectID, startOfDay, endOfDay)
		}
	}

	err := query.Find(&timesheets).Error
	if err != nil {
		return nil, err
	}

	return timesheets, nil
}

// GetEntryTableData retrieves timesheets for entry table reporting with specific filters
func (r *TimesheetQueryRepository) GetEntryTableData(ctx context.Context, projectID uint, employeeIDs []uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet

	query := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Where("date >= ? AND date < ?", fromDate, toDate.Add(24*time.Hour)).
		Where("hours_worked > 0") // Only include non-zero entries

	// Filter by employee IDs if provided
	if len(employeeIDs) > 0 {
		query = query.Where("employee_id IN ?", employeeIDs)
	}

	// Order by employee_id then date for easier processing
	err := query.Order("employee_id ASC, date ASC").Find(&timesheets).Error
	if err != nil {
		return nil, err
	}

	return timesheets, nil
}

// GetPaidAmountByPeriod retrieves the total paid amount for timesheets within a date range
func (r *TimesheetQueryRepository) GetPaidAmountByPeriod(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var result struct {
		Total int64
	}

	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("COALESCE(SUM(paid_amount), 0) as total").
		Where("payment_status = ? AND date >= ? AND date < ?",
			domain.PaymentStatusPaid,
			startDate,
			endDate.Add(24*time.Hour)).
		Scan(&result).Error

	if err != nil {
		return 0, err
	}

	return result.Total, nil
}

// CountDistinctPaidEmployeesByDate counts unique paid employees on a specific date
func (r *TimesheetQueryRepository) CountDistinctPaidEmployeesByDate(ctx context.Context, date time.Time) (int64, error) {
	var result struct {
		Count int64
	}

	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("COUNT(DISTINCT employee_id) as count").
		Where("payment_status = ? AND paid_at >= ? AND paid_at < ?",
			domain.PaymentStatusPaid,
			date,
			date.Add(24*time.Hour)).
		Scan(&result).Error

	if err != nil {
		return 0, err
	}

	return result.Count, nil
}

// GetPaidEmployeesByDateRange returns daily counts of paid employees within 30-day windows
func (r *TimesheetQueryRepository) GetPaidEmployeesByDateRange(ctx context.Context, startDate, endDate time.Time, windowDays int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// First get the earliest paid timesheet date
	var earliestDate time.Time
	r.db.WithContext(ctx).Model(&domain.Timesheet{}).
		Select("MIN(DATE(date))").
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Scan(&earliestDate)

	// Adjust start date to be the earlier of requested start or earliest data
	if startDate.Before(earliestDate) {
		startDate = earliestDate
	}

	// Get rolling 30-day counts for each date
	query := `
		WITH RECURSIVE date_range AS (
			SELECT ? as check_date
			UNION ALL
			SELECT DATE_ADD(check_date, INTERVAL 1 DAY)
			FROM date_range
			WHERE check_date < ?
		),
		employee_counts AS (
			SELECT
				dr.check_date as day,
				(
					SELECT COUNT(DISTINCT t.employee_id)
					FROM timesheets t
					WHERE t.payment_status = 'paid'
					AND t.date >= DATE_SUB(dr.check_date, INTERVAL ? DAY)
					AND t.date < DATE_ADD(dr.check_date, INTERVAL 1 DAY)
				) as employee_count
			FROM date_range dr
		)
		SELECT day, employee_count as count
		FROM employee_counts
		WHERE employee_count > 0
		ORDER BY day
	`

	err := r.db.WithContext(ctx).Raw(query,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		windowDays-1).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get paid employees by date range: %w", err)
	}

	return results, nil
}

// GetWeeklyPayByPeriod retrieves weekly pay totals grouped by payroll period end dates.
// Uses range-based grouping (FLOOR((DAY(date)-1)/7)) so MySQL can use the date index
// for the WHERE range filter before applying the GROUP BY computation.
func (r *TimesheetQueryRepository) GetWeeklyPayByPeriod(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	query := `
		SELECT
			DATE_FORMAT(date, '%Y-%m-01') + INTERVAL (FLOOR((DAY(date)-1)/7) * 7 + 6) DAY as period_date,
			COALESCE(SUM(paid_amount), 0) as total_amount,
			COUNT(DISTINCT employee_id) as paid_employees
		FROM timesheets
		WHERE payment_status = 'paid'
			AND deleted_at IS NULL
			AND date >= ?
			AND date < DATE_ADD(?, INTERVAL 1 DAY)
		GROUP BY period_date
		ORDER BY period_date
	`

	err := r.db.WithContext(ctx).Raw(query,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02")).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

// ListGroupedByEmployee retrieves timesheets grouped by employee with server-side pagination
// This ensures consistent pagination by employee (not by individual timesheet entries)
// Returns employee groups, all timesheets for those employees, total count, and error.
func (r *TimesheetQueryRepository) ListGroupedByEmployee(ctx context.Context, filters domain.TimesheetFilters) ([]domain.EmployeeGroupResult, []*domain.Timesheet, int64, error) {
	// Build the base query for employee grouping (includes latest employee_code via LEFT JOIN)
	baseQuery := r.buildGroupedQuery(filters)

	// Get total count of distinct employees (reuses the same base query, no duplicated access control)
	var total int64
	countQuery := baseQuery.Session(&gorm.Session{})
	err := countQuery.
		Select("COUNT(DISTINCT e.id) as count").
		Scan(&total).Error
	if err != nil {
		return nil, nil, 0, r.errorHandler.HandleListError(err, "grouped timesheet count")
	}

	// Get distinct employees with pagination
	var employeeResults []domain.EmployeeGroupResult

	// Apply sorting
	sortBy := "e.fullname"
	if filters.SortBy != "" {
		switch filters.SortBy {
		case "employee_name":
			sortBy = "e.fullname"
		case "employee_code":
			sortBy = "employee_code"
		case "total_hours":
			sortBy = "total_hours"
		case "total_amount":
			sortBy = "total_amount"
		case "date":
			sortBy = "MAX(timesheets.date)"
		default:
			sortBy = "e.fullname"
		}
	}

	sortOrder := common.SanitizeSortOrder(filters.SortOrder, "ASC")

	// Apply pagination
	limit := filters.Limit
	if limit == 0 {
		limit = 20 // Default page size
	}
	offset := filters.Offset

	// Execute the grouped query
	groupQuery := baseQuery.
		Select(`
			e.id as employee_id,
			e.fullname as employee_name,
			latest_pe.employee_code as employee_code,
			COALESCE(SUM(timesheets.hours_worked), 0) as total_hours,
			COALESCE(SUM(timesheets.amount), 0) as total_amount,
			COUNT(timesheets.id) as entry_count
		`).
		Group("e.id, e.fullname, latest_pe.employee_code").
		Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Limit(limit).
		Offset(offset)

	err = groupQuery.Scan(&employeeResults).Error
	if err != nil {
		return nil, nil, 0, r.errorHandler.HandleListError(err, "grouped timesheet")
	}

	if len(employeeResults) == 0 {
		return employeeResults, []*domain.Timesheet{}, total, nil
	}

	// Extract employee IDs
	employeeIDs := make([]uint, len(employeeResults))
	for i, result := range employeeResults {
		employeeIDs[i] = result.EmployeeID
	}

	// Get all timesheets for these employees using a simple filtered query.
	// No access control needed — employee IDs already came from the access-controlled grouped query.
	var allTimesheets []*domain.Timesheet
	timesheetQuery := r.db.WithContext(ctx).Model(&domain.Timesheet{}).
		Where("timesheets.employee_id IN ?", employeeIDs)
	timesheetQuery = r.applyDataFilters(timesheetQuery, filters)

	err = timesheetQuery.
		Order("timesheets.employee_id ASC, timesheets.date DESC").
		Find(&allTimesheets).Error

	if err != nil {
		return nil, nil, 0, err
	}

	// Load relationships (skip user details for grouped view)
	err = r.relLoader.LoadTimesheetRelationshipsWithOptions(ctx, allTimesheets, relationship_loaders.RelationshipLoadOptions{
		LoadProject:  true,
		LoadEmployee: true,
		LoadUsers:    false,
	})
	if err != nil {
		return nil, nil, 0, err
	}

	return employeeResults, allTimesheets, total, nil
}

// CountDistinctEmployees returns the count of distinct employees matching filters
func (r *TimesheetQueryRepository) CountDistinctEmployees(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	var count int64
	query := r.buildGroupedQuery(filters).
		Select("COUNT(DISTINCT e.id) as count")

	err := query.Scan(&count).Error
	return count, err
}

// buildGroupedQuery builds the base query for employee grouping
func (r *TimesheetQueryRepository) buildGroupedQuery(filters domain.TimesheetFilters) *gorm.DB {
	// Start with the base query structure from query builder
	query := r.db.Model(&domain.Timesheet{}).
		Joins("INNER JOIN employees e ON e.id = timesheets.employee_id").
		Joins("INNER JOIN projects p ON p.id = timesheets.project_id").
		Joins(`LEFT JOIN project_employees latest_pe ON latest_pe.id = (
			SELECT pe.id FROM project_employees pe
			WHERE pe.employee_id = e.id AND pe.deleted_at IS NULL
			ORDER BY pe.created_at DESC LIMIT 1
		)`)

	// Apply partner access control (inline since employees table is already joined as 'e')
	query = r.applyGroupedAccessControl(query, filters)

	// Apply project filter
	if len(filters.ProjectIDs) > 0 {
		query = query.Where("timesheets.project_id IN ?", filters.ProjectIDs)
	}

	// Apply status filter
	if len(filters.TimesheetStatus) > 0 {
		statuses := make([]string, len(filters.TimesheetStatus))
		for i, status := range filters.TimesheetStatus {
			statuses[i] = string(status)
		}
		query = query.Where("timesheets.timesheet_status IN ?", statuses)
	}

	// Apply payment status filter
	if len(filters.PaymentStatus) > 0 {
		paymentStatuses := make([]string, len(filters.PaymentStatus))
		for i, status := range filters.PaymentStatus {
			paymentStatuses[i] = string(status)
		}
		query = query.Where("timesheets.payment_status IN ?", paymentStatuses)
	}

	// Apply paytype filter
	if len(filters.PayType) > 0 {
		query = query.Where("timesheets.paytype IN ?", filters.PayType)
	}

	// Apply date range filter
	if filters.FromDate != nil {
		query = query.Where("timesheets.date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		nextDay := filters.ToDate.Add(24 * time.Hour)
		query = query.Where("timesheets.date < ?", nextDay)
	}

	// Apply search filter on employee fullname or code
	if filters.Search != "" {
		search := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(e.fullname) LIKE ? OR LOWER(latest_pe.employee_code) LIKE ?", search, search)
	}

	// Apply created by filter
	if filters.CreatedBy != nil {
		query = query.Where("timesheets.created_by = ?", *filters.CreatedBy)
	}

	// Apply approved by filter
	if filters.ApprovedBy != nil {
		query = query.Where("timesheets.approved_by = ?", *filters.ApprovedBy)
	}

	// Apply force payroll filter
	if filters.ForcePayroll != nil {
		query = query.Where("timesheets.force_payroll = ?", *filters.ForcePayroll)
	}

	// Apply allowed edit filter
	if filters.AllowedEdit != nil {
		query = query.Where("timesheets.allowed_edit = ?", *filters.AllowedEdit)
	}

	// Apply request edit ID filter
	if filters.RequestEditID != nil {
		query = query.Where("timesheets.request_edit_id = ?", *filters.RequestEditID)
	}

	// Apply has request edit filter
	if filters.HasRequestEdit != nil {
		if *filters.HasRequestEdit {
			query = query.Where("timesheets.request_edit_id IS NOT NULL")
		} else {
			query = query.Where("timesheets.request_edit_id IS NULL")
		}
	}

	return query
}

// applyGroupedAccessControl applies partner-based access control for grouped queries.
// Delegates to the shared query builder method since employees table is already joined as alias 'e'.
func (r *TimesheetQueryRepository) applyGroupedAccessControl(query *gorm.DB, filters domain.TimesheetFilters) *gorm.DB {
	return r.queryBuilder.ApplyPartnerAccessControl(query, filters)
}

// applyDataFilters applies only data-level filters (no access control) to a timesheet query.
// Used when employee IDs are already validated by access control in the grouped query.
func (r *TimesheetQueryRepository) applyDataFilters(query *gorm.DB, filters domain.TimesheetFilters) *gorm.DB {
	if len(filters.ProjectIDs) > 0 {
		query = query.Where("timesheets.project_id IN ?", filters.ProjectIDs)
	}

	if len(filters.TimesheetStatus) > 0 {
		statuses := make([]string, len(filters.TimesheetStatus))
		for i, status := range filters.TimesheetStatus {
			statuses[i] = string(status)
		}
		query = query.Where("timesheets.timesheet_status IN ?", statuses)
	}

	if len(filters.PaymentStatus) > 0 {
		paymentStatuses := make([]string, len(filters.PaymentStatus))
		for i, status := range filters.PaymentStatus {
			paymentStatuses[i] = string(status)
		}
		query = query.Where("timesheets.payment_status IN ?", paymentStatuses)
	}

	if len(filters.PayType) > 0 {
		query = query.Where("timesheets.paytype IN ?", filters.PayType)
	}

	if filters.FromDate != nil {
		query = query.Where("timesheets.date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		nextDay := filters.ToDate.Add(24 * time.Hour)
		query = query.Where("timesheets.date < ?", nextDay)
	}

	if filters.CreatedBy != nil {
		query = query.Where("timesheets.created_by = ?", *filters.CreatedBy)
	}

	if filters.ApprovedBy != nil {
		query = query.Where("timesheets.approved_by = ?", *filters.ApprovedBy)
	}

	if filters.ForcePayroll != nil {
		query = query.Where("timesheets.force_payroll = ?", *filters.ForcePayroll)
	}

	if filters.AllowedEdit != nil {
		query = query.Where("timesheets.allowed_edit = ?", *filters.AllowedEdit)
	}

	if filters.RequestEditID != nil {
		query = query.Where("timesheets.request_edit_id = ?", *filters.RequestEditID)
	}

	if filters.HasRequestEdit != nil {
		if *filters.HasRequestEdit {
			query = query.Where("timesheets.request_edit_id IS NOT NULL")
		} else {
			query = query.Where("timesheets.request_edit_id IS NULL")
		}
	}

	return query
}

// CountUnsettledPaidTimesheets counts timesheets that are paid but revenue not yet received
func (r *TimesheetQueryRepository) CountUnsettledPaidTimesheets(ctx context.Context, projectID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND payment_status = ? AND revenue_paid = ? AND deleted_at IS NULL",
			projectID, domain.PaymentStatusPaid, false).
		Count(&count).Error
	return count, err
}
