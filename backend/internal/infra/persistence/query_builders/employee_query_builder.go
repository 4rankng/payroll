package query_builders

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// EmployeeQueryBuilder handles complex query building for employee operations
type EmployeeQueryBuilder struct {
	*common.BaseQueryBuilder
}

// NewEmployeeQueryBuilder creates a new employee query builder
func NewEmployeeQueryBuilder(db *gorm.DB) *EmployeeQueryBuilder {
	preloads := []string{"Creator", "Bank", "User"}
	return &EmployeeQueryBuilder{
		BaseQueryBuilder: common.NewBaseQueryBuilder(db, "created_at", "DESC", preloads),
	}
}

// BuildListQuery builds a query for listing employees with filters
func (b *EmployeeQueryBuilder) BuildListQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.BuildBaseQuery(&domain.Employee{})
	query = b.ApplyDefaultPreloads(query)

	query = b.ApplyFilters(query, filters)

	// Apply sorting and pagination using BaseQueryBuilder
	query = b.ApplyListOptions(query, filters.SortBy, filters.SortOrder, filters.Limit, filters.Offset)

	return query
}

// BuildGetByIDQuery builds a query for getting an employee by ID
func (b *EmployeeQueryBuilder) BuildGetByIDQuery(id uint) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Preload("Bank").
		Preload("User").
		Where("id = ?", id)
}

// BuildGetByIDForUpdateQuery builds a query for getting an employee by ID with lock
func (b *EmployeeQueryBuilder) BuildGetByIDForUpdateQuery(id uint) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Where("id = ?", id)
}

// BuildGetByFieldQuery builds a query for getting an employee by a specific field
func (b *EmployeeQueryBuilder) BuildGetByFieldQuery(fieldName string, value interface{}) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Preload("Bank").
		Preload("User").
		Where(fieldName+" = ?", value)
}

// BuildGetByFieldWithoutUserQuery builds a query for getting an employee by a specific field without preloading User
func (b *EmployeeQueryBuilder) BuildGetByFieldWithoutUserQuery(fieldName string, value interface{}) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Preload("Bank").
		Where(fieldName+" = ?", value)
}

// BuildExistsQuery builds a query to check if an employee exists by a specific field
func (b *EmployeeQueryBuilder) BuildExistsQuery(fieldName string, value interface{}) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Where(fieldName+" = ?", value)
}

// BuildSearchQuery builds a query for searching employees
func (b *EmployeeQueryBuilder) BuildSearchQuery(search string, limit int) *gorm.DB {
	query := b.DB.Model(&domain.Employee{}).
		Select(`employees.*,
			projects.id as project_id,
			projects.name as project_name,
			projects.code as project_code,
			projects.client_name as project_client_name,
			pe.position as position,
			banks.id as bank_id,
			banks.branch_name as bank_branch_name`).
		Joins(`LEFT JOIN project_employees pe ON employees.id = pe.employee_id
		   AND pe.deleted_at IS NULL
		   AND pe.last_date IS NULL AND pe.start_date <= CURDATE()`).
		Joins("LEFT JOIN projects ON pe.project_id = projects.id AND projects.deleted_at IS NULL").
		Joins("LEFT JOIN banks ON employees.bank_id = banks.id")

	// Apply search filter using normalized column
	normalizedSearch := utils.NormalizeVietnameseForSearch(search)
	query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)

	// Apply limit - default to 100 for max search results
	if limit > 0 && limit <= 100 {
		query = query.Limit(limit)
	} else {
		query = query.Limit(100)
	}

	// Order by fullname for consistent results
	query = query.Order("employees.fullname ASC")

	return query
}

// ApplyFilters applies employee filters to a query
func (b *EmployeeQueryBuilder) ApplyFilters(query *gorm.DB, filters domain.EmployeeFilters) *gorm.DB {
	// Use common FilterBuilder for date range filter
	query = b.GetFilterBuilder().ApplyDateRange(query, &filters)

	// Apply creator filter only if not using AccessibleBy filter
	if filters.CreatedBy != nil && filters.AccessibleBy == nil {
		query = b.GetFilterBuilder().ApplyCreator(query, &filters)
	}

	// Filter for accessible employees (owned, shared, or assigned to projects)
	if filters.AccessibleBy != nil && !filters.SkipAccessibilityFilter {
		query = b.ApplyAccessControl(query, filters)
	}

	// Filter by project (requires join) - employee-specific logic
	if filters.ProjectID != nil {
		subQuery := b.DB.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id = ? AND last_date IS NULL AND deleted_at IS NULL", *filters.ProjectID)
		query = query.Where("employees.id IN (?)", subQuery)
	}

	// Filter by status (working or unassigned) - employee-specific logic
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

	// Filter by payment schedule (via project_employees join) - employee-specific logic
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

	// Search functionality with Vietnamese normalization using virtual column - employee-specific
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	return query
}

// ApplyAccessControl applies user-based access control to a query
// This handles owned, shared, and project-assigned employees
func (b *EmployeeQueryBuilder) ApplyAccessControl(query *gorm.DB, filters domain.EmployeeFilters) *gorm.DB {
	if filters.AccessibleBy == nil {
		return query
	}

	userID := *filters.AccessibleBy

	// Get shared employee IDs for the user via employee_users table
	var sharedEmployeeIDs []uint
	err := b.DB.Model(&domain.EmployeeUser{}).
		Select("employee_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Pluck("employee_id", &sharedEmployeeIDs).Error

	if err != nil {
		// If shared query fails, just filter by created_by and project assignments to prevent total failure
		query = query.Where(
			"employees.created_by = ? OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			userID, userID, userID,
		)
	} else if len(sharedEmployeeIDs) > 0 {
		// Include owned, shared, and assigned employees
		query = query.Where(
			"employees.created_by = ? OR employees.id IN (?) OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			userID, sharedEmployeeIDs, userID, userID,
		)
	} else {
		// Include owned and assigned employees if no shared employees found
		query = query.Where(
			"employees.created_by = ? OR EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = employees.id AND pe.created_by = ? AND pe.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM project_employees pe2 INNER JOIN project_users pu ON pe2.project_id = pu.project_id WHERE pe2.employee_id = employees.id AND pu.user_id = ? AND pe2.deleted_at IS NULL AND pu.deleted_at IS NULL AND pe2.start_date <= NOW() AND (pe2.last_date IS NULL OR pe2.last_date > NOW()))",
			userID, userID, userID,
		)
	}

	return query
}

// BuildGetByIDsQuery builds a query for getting employees by IDs with batching
func (b *EmployeeQueryBuilder) BuildGetByIDsQuery(ctx context.Context, ids []int64) *gorm.DB {
	query := b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Preload("Bank").
		Preload("User")

	if len(ids) == 0 {
		return query.Where("1 = 0") // Return no results
	}

	const batchSize = 1000 // Safe batch size to stay under MySQL's 32KB query limit

	// For large ID lists, we'll need to fetch in batches
	// This method returns the base query, batching should be handled by the caller
	if len(ids) > batchSize {
		// Return query with first batch
		return query.Where("id IN ?", ids[:batchSize])
	}

	return query.Where("id IN ?", ids)
}

// BuildUnassignedAtDateQuery builds a query for getting unassigned employees at a specific date
func (b *EmployeeQueryBuilder) BuildUnassignedAtDateQuery(atDate time.Time, filters domain.EmployeeFilters) *gorm.DB {
	query := b.DB.Model(&domain.Employee{}).
		Preload("Creator")

	// Apply search filter if provided
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	// Find employees NOT assigned to any active project at the given date
	query = query.Where(`NOT EXISTS (
		SELECT 1 FROM project_employees pe
		WHERE pe.employee_id = employees.id
		AND pe.deleted_at IS NULL
		AND pe.start_date <= ?
		AND (pe.last_date IS NULL OR pe.last_date > ?)
	)`, atDate, atDate)

	// Apply sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}

// BuildGetActiveCountAtDateQuery builds a query for counting active employees at a specific date
func (b *EmployeeQueryBuilder) BuildGetActiveCountAtDateQuery(date time.Time) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Where("created_at <= ?", date)
}

// BuildGetActiveCountAtDateForCreatorQuery builds a query for counting active employees for a creator at a specific date
func (b *EmployeeQueryBuilder) BuildGetActiveCountAtDateForCreatorQuery(date time.Time, createdBy uint) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Where("created_at <= ? AND created_by = ?", date, createdBy)
}

// BuildGetByCreatorQuery builds a query for getting employees by creator
func (b *EmployeeQueryBuilder) BuildGetByCreatorQuery(creatorID uint) *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Where("created_by = ?", creatorID)
}

// BuildGetWithBankingInfoQuery builds a query for getting employees with banking info
func (b *EmployeeQueryBuilder) BuildGetWithBankingInfoQuery() *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Where("bank_account_number != '' AND bank_account_name != ''")
}

// BuildGetByAgeRangeQuery builds a query for getting employees within an age range
func (b *EmployeeQueryBuilder) BuildGetByAgeRangeQuery(minAge, maxAge int) *gorm.DB {
	// Calculate date ranges for age filtering
	now := clock.Now()
	maxBirthDate := now.AddDate(-minAge, 0, 0)
	minBirthDate := now.AddDate(-maxAge-1, 0, 0)

	return b.DB.Model(&domain.Employee{}).
		Preload("Creator").
		Where("date_of_birth BETWEEN ? AND ?", minBirthDate, maxBirthDate)
}

// BuildGetDuplicateCCCDsQuery builds a query for getting duplicate CCCDs
func (b *EmployeeQueryBuilder) BuildGetDuplicateCCCDsQuery() *gorm.DB {
	return b.DB.Model(&domain.Employee{}).
		Select("cccd").
		Group("cccd").
		Having("COUNT(*) > 1")
}
