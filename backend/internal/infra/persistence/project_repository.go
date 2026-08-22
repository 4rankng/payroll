package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	pkgConstants "api-server/internal/pkg/constants"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

type ProjectRepository struct {
	*BaseRepository
	filterBuilder *common.FilterBuilder
	errorHandler  *common.RepoErrorHandler
}

func NewProjectRepository(db *Database) domain.ProjectRepository {
	return &ProjectRepository{
		BaseRepository: NewBaseRepository(db),
		filterBuilder:  common.NewFilterBuilder(db.DB),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	return r.DB.WithContext(ctx).Create(project).Error
}

func (r *ProjectRepository) GetByID(ctx context.Context, id uint) (*domain.Project, error) {
	var project domain.Project
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		First(&project, id).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "project", id)
	}

	return &project, nil
}

func (r *ProjectRepository) GetByIDs(ctx context.Context, ids []uint) (map[uint]*domain.Project, error) {
	if len(ids) == 0 {
		return make(map[uint]*domain.Project), nil
	}
	projects, err := common.Chunk(ctx, r.DB.WithContext(ctx), ids, common.DefaultChunkSize,
		func(tx *gorm.DB, batch []uint) ([]*domain.Project, error) {
			var batchProjects []*domain.Project
			if err := tx.Where("id IN ?", batch).Find(&batchProjects).Error; err != nil {
				return nil, err
			}
			return batchProjects, nil
		})
	if err != nil {
		return nil, err
	}
	result := make(map[uint]*domain.Project, len(projects))
	for _, p := range projects {
		result[p.ID] = p
	}
	return result, nil
}

func (r *ProjectRepository) GetByCode(ctx context.Context, code string) (*domain.Project, error) {
	var project domain.Project
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("code = ?", code).
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgProjectNotFoundVN)
		}
		return nil, err
	}

	return &project, nil
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	return r.DB.WithContext(ctx).Save(project).Error
}

func (r *ProjectRepository) Delete(ctx context.Context, id uint) error {
	// Soft delete by setting deleted_at
	return r.DB.WithContext(ctx).Delete(&domain.Project{}, id).Error
}

func (r *ProjectRepository) List(ctx context.Context, filters domain.ProjectFilters) ([]*domain.Project, error) {
	var projects []*domain.Project
	query := r.DB.WithContext(ctx).Preload("Creator")

	// Apply filters using common FilterBuilder
	query = r.applyFilters(query, filters)

	// Apply sorting and pagination using common FilterBuilder
	query = r.filterBuilder.ApplySorting(query, filters.SortBy, filters.SortOrder, "created_at")
	query = r.filterBuilder.ApplyPagination(query, filters.Limit, filters.Offset)

	err := query.Find(&projects).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "projects")
	}
	return projects, nil
}

func (r *ProjectRepository) Count(ctx context.Context, filters domain.ProjectFilters) (int64, error) {
	// If AccessibleBy filter is present, use the same approach as ListWithEmployeeCount
	// Get accessible project IDs first, then count with other filters applied
	if filters.AccessibleBy != nil {
		return r.countAccessible(ctx, filters)
	}

	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.Project{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleListError(err, "projects")
	}
	return count, nil
}

// countAccessible counts projects accessible to a user, matching the logic in listWithEmployeeCountTwoQueries
func (r *ProjectRepository) countAccessible(ctx context.Context, filters domain.ProjectFilters) (int64, error) {
	userID := *filters.AccessibleBy

	// Get all accessible project IDs (owned + shared, no duplicates)
	accessibleProjectIDs, err := r.getAccessibleProjectIDs(ctx, userID)
	if err != nil {
		return 0, err
	}

	if len(accessibleProjectIDs) == 0 {
		return 0, nil
	}

	// Apply other filters (status, date range, search) to count accessible projects
	filterQuery := r.DB.WithContext(ctx).Model(&domain.Project{}).
		Where("projects.id IN (?)", accessibleProjectIDs)

	// Create filters without AccessibleBy to avoid infinite recursion
	countFilters := filters
	countFilters.AccessibleBy = nil
	countFilters.SkipAccessibilityFilter = true

	filterQuery = r.applyFilters(filterQuery, countFilters)

	var count int64
	if err := filterQuery.Count(&count).Error; err != nil {
		return 0, r.errorHandler.HandleListError(err, "projects count")
	}

	return count, nil
}

func (r *ProjectRepository) GetSummary(ctx context.Context) (*domain.ProjectSummary, error) {
	var summary domain.ProjectSummary

	// Count active projects
	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Where("project_status = ?", domain.ProjectStatusRunning).
		Count(&summary.TotalActiveProjects).Error
	if err != nil {
		return nil, err
	}

	// Sum financial fields
	var result struct {
		TotalReceived          float64
		TotalPayout            float64
		TotalPendingPayable    float64
		TotalPendingReceivable float64
	}

	err = r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			COALESCE(SUM(total_received_vnd), 0) as total_received,
			COALESCE(SUM(total_payout_vnd), 0) as total_payout,
			COALESCE(SUM(pending_payable_vnd), 0) as total_pending_payable,
			COALESCE(SUM(pending_receivable_vnd), 0) as total_pending_receivable
		`).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	summary.TotalReceivedVND = result.TotalReceived
	summary.TotalPayoutVND = result.TotalPayout
	summary.TotalPendingPayableVND = result.TotalPendingPayable
	summary.TotalPendingReceivableVND = result.TotalPendingReceivable

	return &summary, nil
}

func (r *ProjectRepository) applyFilters(query *gorm.DB, filters domain.ProjectFilters) *gorm.DB {
	// Use common FilterBuilder for standard filters
	query = r.filterBuilder.ApplyStatus(query, &filters)
	query = r.filterBuilder.ApplyDateRange(query, &filters)

	// Apply creator filter only if not using AccessibleBy filter
	// (AccessibleBy will handle the creator check as part of its logic)
	if filters.CreatedBy != nil && filters.AccessibleBy == nil && filters.ModifiableBy == nil {
		query = r.filterBuilder.ApplyCreator(query, &filters)
	}

	if filters.ModifiableBy != nil {
		query = query.Where(`
			projects.created_by = ? OR EXISTS (
				SELECT 1 FROM project_users
				WHERE project_users.project_id = projects.id
					AND project_users.user_id = ?
					AND project_users.deleted_at IS NULL
			)`, *filters.ModifiableBy, *filters.ModifiableBy)
	}

	// Filter for accessible projects (owned, shared, or with assigned employees)
	if filters.AccessibleBy != nil && !filters.SkipAccessibilityFilter {
		query = r.applyAccessibilityFilter(query, *filters.AccessibleBy)
	}

	// Apply Vietnamese search filter
	query = r.filterBuilder.ApplyVietnameseSearch(query, &filters)

	return query
}

// applyAccessibilityFilter applies the access control filter for projects
// This is project-specific logic that uses optimized LEFT JOINs
// Access paths: project creator, explicit project_users, created employee assignments, employee access via employee_users
func (r *ProjectRepository) applyAccessibilityFilter(query *gorm.DB, userID uint) *gorm.DB {
	// Use optimized LEFT JOIN approach instead of EXISTS subqueries
	query = query.Joins(`
		LEFT JOIN project_users pu ON pu.project_id = projects.id
			AND pu.user_id = ?
			AND pu.deleted_at IS NULL
		LEFT JOIN project_employees pe ON pe.project_id = projects.id
			AND pe.deleted_at IS NULL
		LEFT JOIN employee_users eu ON eu.employee_id = pe.employee_id
			AND eu.user_id = ?
			AND eu.deleted_at IS NULL`,
		userID, userID)

	query = query.Where("projects.created_by = ? OR pu.id IS NOT NULL OR pe.created_by = ? OR eu.id IS NOT NULL", userID, userID)
	return query
}

// Additional helper methods

func (r *ProjectRepository) GetByCreator(ctx context.Context, creatorID uint) ([]*domain.Project, error) {
	var projects []*domain.Project
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("created_by = ?", creatorID).
		Limit(common.DefaultMaxResults).
		Find(&projects).Error

	return projects, err
}

func (r *ProjectRepository) GetActiveProjects(ctx context.Context) ([]*domain.Project, error) {
	var projects []*domain.Project
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("project_status = ?", domain.ProjectStatusRunning).
		Limit(common.DefaultMaxResults).
		Find(&projects).Error

	return projects, err
}

func (r *ProjectRepository) GetByIDWithEmployeeCount(ctx context.Context, id uint) (*domain.ProjectWithEmployeeCount, error) {
	var result domain.ProjectWithEmployeeCount

	employeeCountSubQuery := r.createEmployeeCountSubQuery()

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(pe.employee_count, 0) as employee_count,
			COALESCE(pe.weekly_count, 0) as weekly_salary_employee_count,
			COALESCE(pe.monthly_count, 0) as monthly_salary_employee_count
		`).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (?) pe ON projects.id = pe.project_id", employeeCountSubQuery).
		Where("projects.id = ?", id).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgProjectNotFoundVN)
		}
		return nil, err
	}

	return &result, nil
}

// getAccessibleProjectIDs returns all project IDs accessible to the user (owned + shared + with employee assignments)
// This matches the logic in applyAccessibilityFilter to ensure consistency
// Access paths: project creator, explicit project_users, created employee assignments, employee access via employee_users
func (r *ProjectRepository) getAccessibleProjectIDs(ctx context.Context, userID uint) ([]uint, error) {
	// Use a single optimized query to get all accessible project IDs
	// Matching the logic in applyAccessibilityFilter:
	// 1. Projects created by the user
	// 2. Projects shared via project_users
	// 3. Projects where the user created an employee assignment
	// 4. Projects where the user has access to an assigned employee via employee_users
	var accessibleIDs []uint

	query := r.DB.WithContext(ctx).Model(&domain.Project{}).
		Select("DISTINCT projects.id").
		Joins("LEFT JOIN project_users pu ON pu.project_id = projects.id AND pu.user_id = ? AND pu.deleted_at IS NULL", userID).
		Joins("LEFT JOIN project_employees pe ON pe.project_id = projects.id AND pe.deleted_at IS NULL").
		Joins("LEFT JOIN employee_users eu ON eu.employee_id = pe.employee_id AND eu.user_id = ? AND eu.deleted_at IS NULL", userID).
		Where("projects.created_by = ? OR pu.id IS NOT NULL OR pe.created_by = ? OR eu.id IS NOT NULL", userID, userID).
		Where("projects.deleted_at IS NULL")

	err := query.Pluck("projects.id", &accessibleIDs).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "accessible project IDs")
	}

	return accessibleIDs, nil
}

func (r *ProjectRepository) ListWithEmployeeCount(ctx context.Context, filters domain.ProjectFilters) ([]*domain.ProjectWithEmployeeCount, error) {

	// If AccessibleBy filter is present, use two separate queries approach
	if filters.AccessibleBy != nil {
		return r.listWithEmployeeCountTwoQueries(ctx, filters)
	}

	// Use optimized single aggregation query instead of multiple subqueries
	var results []*domain.ProjectWithEmployeeCount

	query := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(SUM(CASE WHEN pe.last_date IS NULL THEN 1 ELSE 0 END), 0) as employee_count,
			COALESCE(SUM(CASE WHEN pe.last_date IS NULL AND pe.payment_schedule = ? THEN 1 ELSE 0 END), 0) as weekly_salary_employee_count,
			COALESCE(SUM(CASE WHEN pe.last_date IS NULL AND pe.payment_schedule = ? THEN 1 ELSE 0 END), 0) as monthly_salary_employee_count
		`, pkgConstants.PaymentScheduleWeekly, pkgConstants.PaymentScheduleMonthly).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN project_employees pe ON projects.id = pe.project_id AND pe.deleted_at IS NULL").
		Where("projects.deleted_at IS NULL").
		Group("projects.id, u.fullname")

	// Apply additional filters
	query = r.applyFilters(query, filters)

	// Apply sorting
	sortBy := "projects.created_at"
	if filters.SortBy != "" {
		sortBy = "projects." + filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "projects.created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Scan(&results).Error
	return results, err
}

// listWithEmployeeCountTwoQueries performs separate queries for better performance with AccessibleBy filter
// Access paths: 1) owned, 2) shared via project_users, 3) employee assignments created by user, 4) via employee_users
func (r *ProjectRepository) listWithEmployeeCountTwoQueries(ctx context.Context, filters domain.ProjectFilters) ([]*domain.ProjectWithEmployeeCount, error) {
	userID := *filters.AccessibleBy

	// Get shared project IDs
	sharedProjectIDs, err := r.getSharedProjectIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get employee-accessible project IDs (via employee_users or created employee assignments)
	employeeProjectIDs, err := r.getEmployeeAccessibleProjectIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Fetch projects from each access path
	ownedProjects, err := r.fetchOwnedProjects(ctx, filters, userID)
	if err != nil {
		return nil, err
	}

	sharedProjects, err := r.fetchSharedProjects(ctx, filters, sharedProjectIDs)
	if err != nil {
		return nil, err
	}

	employeeProjects, err := r.fetchEmployeeAccessibleProjects(ctx, filters, employeeProjectIDs, userID, sharedProjectIDs)
	if err != nil {
		return nil, err
	}

	// Merge and paginate
	return r.mergeAndPaginateProjects(ownedProjects, sharedProjects, employeeProjects, filters.Offset, filters.Limit)
}

// getSharedProjectIDs retrieves project IDs shared with the given user via project_users
func (r *ProjectRepository) getSharedProjectIDs(ctx context.Context, userID uint) ([]uint, error) {
	var sharedProjectIDs []uint
	err := r.DB.WithContext(ctx).Model(&domain.ProjectUser{}).
		Select("project_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Pluck("project_id", &sharedProjectIDs).Error

	if err != nil {
		sharedProjectIDs = []uint{} // Continue with empty shared projects
	}

	return sharedProjectIDs, nil
}

// getEmployeeAccessibleProjectIDs retrieves project IDs accessible via employee access
// Includes: 1) projects where user created employee assignment, 2) projects where user has access to assigned employee via employee_users
func (r *ProjectRepository) getEmployeeAccessibleProjectIDs(ctx context.Context, userID uint) ([]uint, error) {
	var employeeProjectIDs []uint

	err := r.DB.WithContext(ctx).Model(&domain.Project{}).
		Select("DISTINCT projects.id").
		Joins("LEFT JOIN project_employees pe ON pe.project_id = projects.id AND pe.deleted_at IS NULL").
		Joins("LEFT JOIN employee_users eu ON eu.employee_id = pe.employee_id AND eu.user_id = ? AND eu.deleted_at IS NULL", userID).
		Where("(pe.created_by = ? OR eu.id IS NOT NULL) AND projects.deleted_at IS NULL", userID).
		Pluck("projects.id", &employeeProjectIDs).Error

	if err != nil {
		employeeProjectIDs = []uint{} // Continue with empty employee projects
	}

	return employeeProjectIDs, nil
}

// createEmployeeCountSubQuery creates an optimized subquery for employee counts
func (r *ProjectRepository) createEmployeeCountSubQuery() *gorm.DB {
	return r.DB.Model(&domain.ProjectEmployee{}).
		Select(`
			project_id,
			COUNT(*) as employee_count,
			SUM(CASE WHEN payment_schedule = ? THEN 1 ELSE 0 END) as weekly_count,
			SUM(CASE WHEN payment_schedule = ? THEN 1 ELSE 0 END) as monthly_count
		`, pkgConstants.PaymentScheduleWeekly, pkgConstants.PaymentScheduleMonthly).
		Where("last_date IS NULL").
		Group("project_id")
}

// fetchOwnedProjects retrieves projects created by the user with employee counts
func (r *ProjectRepository) fetchOwnedProjects(ctx context.Context, filters domain.ProjectFilters, userID uint) ([]*domain.ProjectWithEmployeeCount, error) {
	ownedFilters := filters
	ownedFilters.CreatedBy = &userID
	ownedFilters.AccessibleBy = nil
	ownedFilters.SkipAccessibilityFilter = true

	employeeCountSubQuery := r.createEmployeeCountSubQuery()

	query := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(pe.employee_count, 0) as employee_count,
			COALESCE(pe.weekly_count, 0) as weekly_salary_employee_count,
			COALESCE(pe.monthly_count, 0) as monthly_salary_employee_count
		`).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (?) pe ON projects.id = pe.project_id", employeeCountSubQuery)

	query = r.applyFilters(query, ownedFilters)

	// Apply sorting
	sortBy := "projects.created_at"
	if ownedFilters.SortBy != "" {
		sortBy = "projects." + ownedFilters.SortBy
	}
	sortOrder := "DESC"
	if ownedFilters.SortOrder != "" {
		sortOrder = ownedFilters.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "projects.created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	var results []*domain.ProjectWithEmployeeCount
	err := query.Scan(&results).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "owned projects")
	}

	return results, nil
}

// fetchSharedProjects retrieves projects shared with the user with employee counts
func (r *ProjectRepository) fetchSharedProjects(ctx context.Context, filters domain.ProjectFilters, projectIDs []uint) ([]*domain.ProjectWithEmployeeCount, error) {
	if len(projectIDs) == 0 {
		return []*domain.ProjectWithEmployeeCount{}, nil
	}

	sharedFilters := filters
	sharedFilters.AccessibleBy = nil
	sharedFilters.CreatedBy = nil
	sharedFilters.SkipAccessibilityFilter = true

	employeeCountSubQuery := r.createEmployeeCountSubQuery()

	query := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(pe.employee_count, 0) as employee_count,
			COALESCE(pe.weekly_count, 0) as weekly_salary_employee_count,
			COALESCE(pe.monthly_count, 0) as monthly_salary_employee_count
		`).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (?) pe ON projects.id = pe.project_id", employeeCountSubQuery).
		Where("projects.id IN (?)", projectIDs)

	query = r.applyFilters(query, sharedFilters)

	// Apply sorting
	sortBy := "projects.created_at"
	if sharedFilters.SortBy != "" {
		sortBy = "projects." + sharedFilters.SortBy
	}
	sortOrder := "DESC"
	if sharedFilters.SortOrder != "" {
		sortOrder = sharedFilters.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "projects.created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	var results []*domain.ProjectWithEmployeeCount
	err := query.Scan(&results).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "shared projects")
	}

	return results, nil
}

// fetchEmployeeAccessibleProjects retrieves projects accessible via employee access with employee counts
// Excludes projects already fetched as owned or shared to avoid duplicates
func (r *ProjectRepository) fetchEmployeeAccessibleProjects(ctx context.Context, filters domain.ProjectFilters, projectIDs []uint, userID uint, sharedProjectIDs []uint) ([]*domain.ProjectWithEmployeeCount, error) {
	if len(projectIDs) == 0 {
		return []*domain.ProjectWithEmployeeCount{}, nil
	}

	employeeFilters := filters
	employeeFilters.AccessibleBy = nil
	employeeFilters.CreatedBy = nil
	employeeFilters.SkipAccessibilityFilter = true

	employeeCountSubQuery := r.createEmployeeCountSubQuery()

	query := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(pe.employee_count, 0) as employee_count,
			COALESCE(pe.weekly_count, 0) as weekly_salary_employee_count,
			COALESCE(pe.monthly_count, 0) as monthly_salary_employee_count
		`).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (?) pe ON projects.id = pe.project_id", employeeCountSubQuery).
		Where("projects.id IN (?)", projectIDs).
		Where("projects.created_by != ?", userID) // Exclude owned projects

	// Exclude shared projects
	if len(sharedProjectIDs) > 0 {
		query = query.Where("projects.id NOT IN (?)", sharedProjectIDs)
	}

	query = r.applyFilters(query, employeeFilters)

	// Apply sorting
	sortBy := "projects.created_at"
	if employeeFilters.SortBy != "" {
		sortBy = "projects." + employeeFilters.SortBy
	}
	sortOrder := "DESC"
	if employeeFilters.SortOrder != "" {
		sortOrder = employeeFilters.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "projects.created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	var results []*domain.ProjectWithEmployeeCount
	err := query.Scan(&results).Error
	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "employee-accessible projects")
	}

	return results, nil
}

// mergeAndPaginateProjects merges owned, shared, and employee-accessible projects, removes duplicates, sorts, and applies pagination
func (r *ProjectRepository) mergeAndPaginateProjects(owned, shared, employee []*domain.ProjectWithEmployeeCount, offset, limit int) ([]*domain.ProjectWithEmployeeCount, error) {
	// Merge results, avoiding duplicates
	projectIDMap := make(map[uint]bool)
	var allResults []*domain.ProjectWithEmployeeCount

	// Add owned projects first
	for _, project := range owned {
		projectIDMap[project.ID] = true
		allResults = append(allResults, project)
	}

	// Add shared projects that aren't already in the list
	for _, project := range shared {
		if !projectIDMap[project.ID] {
			projectIDMap[project.ID] = true
			allResults = append(allResults, project)
		}
	}

	// Add employee-accessible projects that aren't already in the list
	for _, project := range employee {
		if !projectIDMap[project.ID] {
			projectIDMap[project.ID] = true
			allResults = append(allResults, project)
		}
	}

	// Sort by created_at DESC (default) to maintain consistent ordering
	// Note: Each individual fetch is already sorted, so this maintains the overall sort order
	// by preserving created_at ordering within each group

	// Apply pagination
	totalResults := len(allResults)
	start := offset
	if start >= totalResults {
		return []*domain.ProjectWithEmployeeCount{}, nil
	}

	end := start + limit
	if limit <= 0 || end > totalResults {
		end = totalResults
	}

	return allResults[start:end], nil
}

func (r *ProjectRepository) GetProjectsWithEmployeeCount(ctx context.Context) ([]*domain.ProjectWithEmployeeCount, error) {
	var results []*domain.ProjectWithEmployeeCount

	employeeCountSubQuery := r.createEmployeeCountSubQuery()

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select(`
			projects.*,
			u.fullname as creator_fullname,
			COALESCE(pe.employee_count, 0) as employee_count,
			COALESCE(pe.weekly_count, 0) as weekly_salary_employee_count,
			COALESCE(pe.monthly_count, 0) as monthly_salary_employee_count
		`).
		Joins("LEFT JOIN users u ON projects.created_by = u.id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (?) pe ON projects.id = pe.project_id", employeeCountSubQuery).
		Limit(common.DefaultMaxResults).
		Scan(&results).Error

	return results, err
}

func (r *ProjectRepository) GetFinancialSummaryByDateRange(ctx context.Context, startDate, endDate time.Time) (*domain.ProjectFinancialSummary, error) {
	var summary domain.ProjectFinancialSummary

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Select(`
			COUNT(*) as total_projects,
			COALESCE(SUM(total_received_vnd), 0) as total_revenue,
			COALESCE(SUM(total_payout_vnd), 0) as total_expenses,
			COALESCE(SUM(pending_receivable_vnd), 0) as pending_revenue,
			COALESCE(SUM(pending_payable_vnd), 0) as pending_expenses
		`).
		Scan(&summary).Error

	if err != nil {
		return nil, err
	}

	summary.NetProfit = summary.TotalRevenue - summary.TotalExpenses
	summary.StartDate = startDate
	summary.EndDate = endDate

	return &summary, nil
}

// GetActiveCountAtDate returns count of active projects at a specific date
func (r *ProjectRepository) GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Where("project_status = ? AND created_at <= ?", domain.ProjectStatusRunning, date).
		Count(&count).Error

	return int(count), err
}

// GetStatusCounts returns count of projects by status
func (r *ProjectRepository) GetStatusCounts(ctx context.Context) (map[string]int, error) {
	type StatusCount struct {
		Status string
		Count  int64
	}

	var results []StatusCount
	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("project_status as status, COUNT(*) as count").
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	statusCounts := make(map[string]int)
	for _, result := range results {
		statusCounts[result.Status] = int(result.Count)
	}

	// Ensure all statuses are represented
	if _, exists := statusCounts["draft"]; !exists {
		statusCounts["draft"] = 0
	}
	if _, exists := statusCounts["active"]; !exists {
		statusCounts["active"] = 0
	}
	if _, exists := statusCounts["completed"]; !exists {
		statusCounts["completed"] = 0
	}
	if _, exists := statusCounts["cancelled"]; !exists {
		statusCounts["cancelled"] = 0
	}
	if _, exists := statusCounts["inactive"]; !exists {
		statusCounts["inactive"] = 0
	}

	return statusCounts, nil
}

// GetStatusCountsForCreator gets project status counts for a specific creator
func (r *ProjectRepository) GetStatusCountsForCreator(ctx context.Context, createdBy uint) (map[string]int, error) {
	type StatusCount struct {
		Status string
		Count  int64
	}

	var results []StatusCount
	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("project_status as status, COUNT(*) as count").
		Where("created_by = ?", createdBy).
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	statusCounts := make(map[string]int)
	for _, result := range results {
		statusCounts[result.Status] = int(result.Count)
	}

	// Ensure all statuses are represented
	if _, exists := statusCounts["draft"]; !exists {
		statusCounts["draft"] = 0
	}
	if _, exists := statusCounts["active"]; !exists {
		statusCounts["active"] = 0
	}
	if _, exists := statusCounts["paused"]; !exists {
		statusCounts["paused"] = 0
	}
	if _, exists := statusCounts["completed"]; !exists {
		statusCounts["completed"] = 0
	}
	if _, exists := statusCounts["cancelled"]; !exists {
		statusCounts["cancelled"] = 0
	}
	if _, exists := statusCounts["inactive"]; !exists {
		statusCounts["inactive"] = 0
	}

	return statusCounts, nil
}

// SearchProjects searches for projects using Vietnamese text normalization
func (r *ProjectRepository) SearchProjects(ctx context.Context, search string, limit int) ([]*domain.Project, error) {
	if search == "" {
		return nil, domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}

	if len(strings.TrimSpace(search)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	if limit <= 0 || limit > 100 {
		limit = 100 // default limit - return all results up to 100
	}

	var projects []*domain.Project

	normalizedSearch := utils.NormalizeVietnameseForSearch(search)

	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where(
			"name LIKE ? OR code LIKE ? OR client_name LIKE ? OR description LIKE ?",
			normalizedSearch, normalizedSearch, normalizedSearch, normalizedSearch,
		).
		Order("name ASC").
		Limit(limit).
		Find(&projects).Error

	return projects, err
}

// GetPendingActivatedProjects returns draft projects that should be activated (start_date <= given date)
func (r *ProjectRepository) GetPendingActivatedProjects(ctx context.Context, date time.Time) ([]*domain.Project, error) {
	var projects []*domain.Project

	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("project_status = ? AND start_date IS NOT NULL AND start_date <= ?",
			domain.ProjectStatusDraft, date).
		Limit(common.DefaultMaxResults).
		Find(&projects).Error

	return projects, err
}

// GetPendingCompletedProjects returns draft or active projects that should be completed (end_date <= given date)
func (r *ProjectRepository) GetPendingCompletedProjects(ctx context.Context, date time.Time) ([]*domain.Project, error) {
	var projects []*domain.Project

	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("project_status IN (?, ?) AND end_date IS NOT NULL AND end_date <= ?",
			domain.ProjectStatusDraft, domain.ProjectStatusRunning, date).
		Limit(common.DefaultMaxResults).
		Find(&projects).Error

	return projects, err
}

// CodeExistsIncludingDeleted checks if a project code exists, including soft-deleted records
// This is used during project code generation to avoid conflicts with soft-deleted project codes
func (r *ProjectRepository) CodeExistsIncludingDeleted(ctx context.Context, code string) bool {
	var count int64
	err := r.DB.WithContext(ctx).
		Unscoped(). // Include soft-deleted records
		Model(&domain.Project{}).
		Where("code = ?", code).
		Count(&count).Error

	if err != nil {
		// Log error but return true to be safe (assume code exists)
		return true
	}

	return count > 0
}
