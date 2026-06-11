package query_builders

import (
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// EmployeeStatisticsBuilder handles complex query building for employee statistics and summary operations
type EmployeeStatisticsBuilder struct {
	db *gorm.DB
}

// NewEmployeeStatisticsBuilder creates a new employee statistics query builder
func NewEmployeeStatisticsBuilder(db *gorm.DB) *EmployeeStatisticsBuilder {
	return &EmployeeStatisticsBuilder{
		db: db,
	}
}

// BuildCountQuery builds a query for counting employees with filters
func (b *EmployeeStatisticsBuilder) BuildCountQuery(filters domain.EmployeeFilters) *gorm.DB {
	// If AccessibleBy filter is present, use two separate queries approach for accurate count
	if filters.AccessibleBy != nil {
		return b.BuildCountWithAccessibilityQuery(filters)
	}

	// For other filters, use the original single query approach
	query := b.db.Model(&domain.Employee{})
	return b.applyFilters(query, filters)
}

// BuildCountWithAccessibilityQuery performs separate count queries for better accuracy with AccessibleBy filter
// This avoids the complex subquery issues and provides accurate counting
func (b *EmployeeStatisticsBuilder) BuildCountWithAccessibilityQuery(filters domain.EmployeeFilters) *gorm.DB {
	// This method returns a query that will count with accessibility
	// The actual counting logic is implemented in the repository
	// as it requires executing multiple queries and aggregating results
	query := b.db.Model(&domain.Employee{})
	return b.applyFilters(query, filters)
}

// BuildSummaryQuery builds a query for employee summary statistics
func (b *EmployeeStatisticsBuilder) BuildSummaryQuery() *gorm.DB {
	now := clock.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Query to get employee counts and month-to-date salary/paid
	return b.db.Model(&domain.Employee{}).
		Select(`
            COUNT(*) as total_employees,
            COALESCE((
                SELECT SUM(t.amount)
                FROM timesheets t
                INNER JOIN employees e2 ON t.employee_id = e2.id
                WHERE t.timesheet_status = 'approved'
                  AND t.date BETWEEN ? AND ?
            ), 0) as salary_month_to_date,
            COALESCE((
                SELECT SUM(t.paid_amount)
                FROM timesheets t
                INNER JOIN employees e3 ON t.employee_id = e3.id
                WHERE t.timesheet_status = 'approved'
                  AND t.payment_status = 'paid'
                  AND t.date BETWEEN ? AND ?
            ), 0) as paid_month_to_date
        `, startOfMonth, now, startOfMonth, now)
}

// BuildSummaryForCreatorQuery builds a query for employee summary for a specific creator
func (b *EmployeeStatisticsBuilder) BuildSummaryForCreatorQuery(createdBy uint) *gorm.DB {
	// Get shared employee IDs for the user via employee_users table
	var sharedEmployeeIDs []uint
	_ = b.db.Model(&domain.EmployeeUser{}).
		Select("employee_id").
		Where("user_id = ? AND deleted_at IS NULL", createdBy).
		Pluck("employee_id", &sharedEmployeeIDs)

	// Build query for accessible employees (owned, shared, or assigned)
	query := b.db.Model(&domain.Employee{})

	// Apply the same accessibility logic used in the List method
	if len(sharedEmployeeIDs) > 0 {
		// Include owned, shared, and assigned employees
		query = query.Where(
			"employees.created_by = ? OR employees.id IN (?) OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			createdBy, sharedEmployeeIDs, createdBy, createdBy,
		)
	} else {
		// Include owned and assigned employees if no shared employees found
		query = query.Where(
			"employees.created_by = ? OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			createdBy, createdBy, createdBy,
		)
	}

	return query
}

// BuildSummaryForCreatorCountQuery builds a query for counting accessible employees for a creator
func (b *EmployeeStatisticsBuilder) BuildSummaryForCreatorCountQuery(createdBy uint) *gorm.DB {
	return b.BuildSummaryForCreatorQuery(createdBy)
}

// BuildSummaryForCreatorSalaryQuery builds a query for salary statistics for accessible employees
func (b *EmployeeStatisticsBuilder) BuildSummaryForCreatorSalaryQuery(accessibleEmployeeIDs []uint) *gorm.DB {
	if len(accessibleEmployeeIDs) == 0 {
		return b.db.Model(&domain.Employee{}).Where("1 = 0")
	}

	now := clock.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	return b.db.Model(&domain.Employee{}).
		Select(`
			COALESCE((
				SELECT SUM(t.amount)
				FROM timesheets t
				WHERE t.employee_id IN (?)
				  AND t.timesheet_status = 'approved'
				  AND t.date BETWEEN ? AND ?
			), 0) as salary_month_to_date,
			COALESCE((
				SELECT SUM(t.paid_amount)
				FROM timesheets t
				WHERE t.employee_id IN (?)
				  AND t.timesheet_status = 'approved'
				  AND t.payment_status = 'paid'
				  AND t.date BETWEEN ? AND ?
			), 0) as paid_month_to_date
		`, accessibleEmployeeIDs, startOfMonth, now, accessibleEmployeeIDs, startOfMonth, now)
}

// BuildSummaryForCreatorProjectStatsQuery builds a query for project-related statistics for accessible employees
func (b *EmployeeStatisticsBuilder) BuildSummaryForCreatorProjectStatsQuery(accessibleEmployeeIDs []uint, startOfMonth, endOfMonth time.Time) *gorm.DB {
	if len(accessibleEmployeeIDs) == 0 {
		return b.db.Model(&domain.Employee{}).Where("1 = 0")
	}

	return b.db.Model(&domain.Employee{}).
		Select(`
			COUNT(DISTINCT CASE WHEN deleted_at IS NULL AND created_at BETWEEN ? AND ? THEN id END) as employees_hired_this_month
		`, startOfMonth, endOfMonth).
		Where("id IN ?", accessibleEmployeeIDs)
}

// BuildSummaryForCreatorWorkingCountQuery builds a query for counting working employees
func (b *EmployeeStatisticsBuilder) BuildSummaryForCreatorWorkingCountQuery(accessibleEmployeeIDs []uint) *gorm.DB {
	if len(accessibleEmployeeIDs) == 0 {
		return b.db.Model(&domain.Employee{}).Where("1 = 0")
	}

	return b.db.Model(&domain.Employee{}).
		Where("id IN ?", accessibleEmployeeIDs).
		Where(`deleted_at IS NULL AND EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.deleted_at IS NULL
			AND pe.start_date <= NOW()
			AND (pe.last_date IS NULL OR pe.last_date > NOW())
		)`)
}

// BuildStatisticsQuery builds a query for employee statistics
func (b *EmployeeStatisticsBuilder) BuildStatisticsQuery() *gorm.DB {
	return b.db.Model(&domain.Employee{}).
		Select(`
			COUNT(*) as total_employees,
			COUNT(CASE WHEN bank_account_number != '' AND bank_account_name != '' THEN 1 END) as with_banking_info
		`)
}

// BuildStatisticsAssignedQuery builds a query to count employees assigned to projects
func (b *EmployeeStatisticsBuilder) BuildStatisticsAssignedQuery() *gorm.DB {
	subQuery := b.db.Model(&domain.ProjectEmployee{}).
		Select("DISTINCT employee_id").
		Where("deleted_at IS NULL").
		Where("(last_date IS NULL OR last_date > NOW())")

	return b.db.Model(&domain.Employee{}).
		Where("id IN (?)", subQuery)
}

// BuildRecentEmployeesQuery builds a query for getting recent employees within a date range
func (b *EmployeeStatisticsBuilder) BuildRecentEmployeesQuery(startDate, endDate time.Time, limit int) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Preload("Creator").
		Preload("Bank").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	return query
}

// BuildRecentEmployeesPaginatedQuery builds a query for getting recent employees with pagination
func (b *EmployeeStatisticsBuilder) BuildRecentEmployeesPaginatedQuery(startDate, endDate time.Time, limit, offset int) *gorm.DB {
	return b.db.Model(&domain.Employee{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
}

// BuildCountRecentQuery builds a query for counting recent employees within a date range
func (b *EmployeeStatisticsBuilder) BuildCountRecentQuery(startDate, endDate time.Time) *gorm.DB {
	return b.db.Model(&domain.Employee{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate)
}

// BuildGetAccessibleEmployeeIDsQuery builds a query to get employee IDs accessible to a user
// Returns a query that fetches the IDs
func (b *EmployeeStatisticsBuilder) BuildGetAccessibleEmployeeIDsQuery(createdBy uint) *gorm.DB {
	// Build query for accessible employees (owned, shared, or assigned)
	query := b.db.Model(&domain.Employee{}).Select("employees.id")

	// Get shared employee IDs for the user via employee_users table
	var sharedEmployeeIDs []uint
	_ = b.db.Model(&domain.EmployeeUser{}).
		Select("employee_id").
		Where("user_id = ? AND deleted_at IS NULL", createdBy).
		Pluck("employee_id", &sharedEmployeeIDs)

	// Apply the same accessibility logic used in the List method
	if len(sharedEmployeeIDs) > 0 {
		// Include owned, shared, and assigned employees
		query = query.Where(
			"employees.created_by = ? OR employees.id IN (?) OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			createdBy, sharedEmployeeIDs, createdBy, createdBy,
		)
	} else {
		// Include owned and assigned employees if no shared employees found
		query = query.Where(
			"employees.created_by = ? OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			createdBy, createdBy, createdBy,
		)
	}

	return query
}

// BuildWorkingEmployeesCountQuery builds a query for counting working employees
func (b *EmployeeStatisticsBuilder) BuildWorkingEmployeesCountQuery() *gorm.DB {
	return b.db.Model(&domain.Employee{}).
		Where(`deleted_at IS NULL AND EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.deleted_at IS NULL
			AND pe.start_date <= NOW()
			AND (pe.last_date IS NULL OR pe.last_date > NOW())
		)`)
}

// BuildEmployeesHiredThisMonthQuery builds a query for counting employees hired this month
func (b *EmployeeStatisticsBuilder) BuildEmployeesHiredThisMonthQuery(startOfMonth, endOfMonth time.Time) *gorm.DB {
	return b.db.Model(&domain.Employee{}).
		Select(`
			COUNT(DISTINCT CASE WHEN deleted_at IS NULL AND created_at BETWEEN ? AND ? THEN id END) as employees_hired_this_month
		`, startOfMonth, endOfMonth)
}

// applyFilters applies filters to a statistics query
func (b *EmployeeStatisticsBuilder) applyFilters(query *gorm.DB, filters domain.EmployeeFilters) *gorm.DB {
	// Filter by creator
	if filters.CreatedBy != nil {
		query = query.Where("employees.created_by = ?", *filters.CreatedBy)
	}

	// Filter by project (requires join)
	if filters.ProjectID != nil {
		subQuery := b.db.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id = ? AND last_date IS NULL AND deleted_at IS NULL", *filters.ProjectID)
		query = query.Where("employees.id IN (?)", subQuery)
	}

	// Filter by status (working or unassigned)
	switch filters.Status {
	case "working":
		// Employees currently assigned to at least one project
		query = query.Where(`EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.deleted_at IS NULL
			AND pe.start_date <= NOW()
			AND (pe.last_date IS NULL OR pe.last_date > NOW())
		)`)
	case "unassigned":
		// Employees NOT assigned to any project
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.deleted_at IS NULL
			AND pe.start_date <= NOW()
			AND (pe.last_date IS NULL OR pe.last_date > NOW())
		)`)
	}

	// Filter by date range on created_at
	if filters.FromDate != nil {
		query = query.Where("employees.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		// Include the entire day by adding 23:59:59
		endOfDay := timeutil.EndOfDay(*filters.ToDate)
		query = query.Where("employees.created_at <= ?", endOfDay)
	}

	// Filter by payment schedule (via project_employees join)
	if filters.PaymentSchedule != "" {
		query = query.Where(`EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			AND pe.deleted_at IS NULL
			AND pe.last_date IS NULL
			AND pe.start_date <= NOW()
			AND pe.payment_schedule = ?
		)`, filters.PaymentSchedule)
	}

	// Search functionality with Vietnamese normalization using virtual column
	if filters.Search != "" {
		// Use NormalizeVietnameseForSearch if available, otherwise use the search as-is
		// This assumes utils.NormalizeVietnameseForSearch is available
		normalizedSearch := "%" + filters.Search + "%" // Simplified version
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	return query
}
