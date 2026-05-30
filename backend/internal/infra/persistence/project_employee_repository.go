package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"gorm.io/gorm"
)

type ProjectEmployeeRepository struct {
	*BaseRepository
}

func NewProjectEmployeeRepository(db *Database) domain.ProjectEmployeeRepository {
	return &ProjectEmployeeRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// getDB returns the transaction-aware DB session. If a transaction is active in
// the context (set via domain.WithTransactionContext), it uses that TX; otherwise
// it falls back to the repository's own DB connection.
func (r *ProjectEmployeeRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

// applyCommonPreloads centralizes the loading of common related entities
// Always excludes deleted projects and employees
func (r *ProjectEmployeeRepository) applyCommonPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Project", "deleted_at IS NULL").
		Preload("Employee", "deleted_at IS NULL").
		Preload("Creator")
}

func (r *ProjectEmployeeRepository) Create(ctx context.Context, assignment *domain.ProjectEmployee) error {
	return r.SafeCreate(ctx, assignment)
}

func (r *ProjectEmployeeRepository) GetByID(ctx context.Context, id uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	query := r.DB.WithContext(ctx)
	err := r.applyCommonPreloads(query).First(&assignment, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

func (r *ProjectEmployeeRepository) GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("project_id = ? AND employee_id = ?", projectID, employeeID)
	err := r.applyCommonPreloads(query).First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

func (r *ProjectEmployeeRepository) GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	// Check for active assignment: no end date OR end date is today or in the future
	today := clock.NowUTC().Truncate(24 * time.Hour)

	err := r.getDB(ctx).
		Where("project_id = ? AND employee_id = ? AND (last_date IS NULL OR last_date >= ?)",
			projectID, employeeID, today).
		Order("created_at DESC, id DESC").
		First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgActiveAssignmentNotFoundVN)
		}
		return nil, err
	}

	return &assignment, nil
}

// GetActiveAssignmentsByProjectsAndEmployees batch fetches active assignments for multiple project-employee pairs
func (r *ProjectEmployeeRepository) GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*domain.ProjectEmployee, error) {
	if len(projectIDs) == 0 || len(employeeIDs) == 0 {
		return []*domain.ProjectEmployee{}, nil
	}

	var assignments []*domain.ProjectEmployee
	today := clock.NowUTC().Truncate(24 * time.Hour)

	// Fetch all active assignments for the given projects and employees, excluding deleted projects and employees
	// This uses the composite index we created: idx_project_employees_project_employee
	query := r.DB.WithContext(ctx).
		Joins("INNER JOIN projects ON project_employees.project_id = projects.id AND projects.deleted_at IS NULL").
		Joins("INNER JOIN employees ON project_employees.employee_id = employees.id AND employees.deleted_at IS NULL").
		Where("project_employees.project_id IN ? AND project_employees.employee_id IN ? AND (project_employees.last_date IS NULL OR project_employees.last_date >= ?)",
			projectIDs, employeeIDs, today)

	err := query.Find(&assignments).Error

	if err != nil {
		return nil, err
	}

	return assignments, nil
}

func (r *ProjectEmployeeRepository) Update(ctx context.Context, assignment *domain.ProjectEmployee) error {
	// Omit relationship structs to prevent Save() from overwriting foreign keys
	// with zero values when the assignment was fetched without preloads
	// (e.g., via GetActiveAssignmentByProjectAndEmployee used by ToggleCheckInEnabled).
	db := r.getDB(ctx).Omit("Project", "Employee", "Creator")
	if err := db.Save(assignment).Error; err != nil {
		return fmt.Errorf("failed to update project employee: %w", err)
	}
	return nil
}

func (r *ProjectEmployeeRepository) Delete(ctx context.Context, id uint) error {
	return r.SafeDelete(ctx, &domain.ProjectEmployee{}, id)
}

func (r *ProjectEmployeeRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return r.DB.WithContext(ctx).
		Where("project_id = ?", projectID).
		Delete(&domain.ProjectEmployee{}).Error
}

func (r *ProjectEmployeeRepository) BulkUpdateLastDate(ctx context.Context, assignmentIDs []uint, lastDate time.Time) error {
	if len(assignmentIDs) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id IN ? AND last_date IS NULL", assignmentIDs).
		Updates(map[string]any{
			"last_date":  lastDate,
			"updated_at": clock.Now(),
		}).Error
}

func (r *ProjectEmployeeRepository) List(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee

	// Special handling for project employee listing to avoid duplicates
	if filters.ProjectID != nil {
		return r.getDistinctProjectEmployees(ctx, filters)
	}

	query := r.DB.WithContext(ctx)
	query = r.applyCommonPreloads(query)

	// Apply filters
	query = r.applyFilters(query, filters)

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

	err := query.Find(&assignments).Error
	return assignments, err
}

func (r *ProjectEmployeeRepository) Count(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	var count int64

	// Special handling for project employee counting to avoid duplicates
	if filters.ProjectID != nil {
		return r.countDistinctProjectEmployees(ctx, filters)
	}

	query := r.DB.WithContext(ctx).Model(&domain.ProjectEmployee{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *ProjectEmployeeRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("project_id = ?", projectID)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) GetByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("employee_id = ?", employeeID)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) GetActiveAssignments(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	today := clock.NowUTC().Truncate(24 * time.Hour)

	query := r.DB.WithContext(ctx).Where("project_id = ? AND (last_date IS NULL OR last_date >= ?)", projectID, today)
	err := r.applyCommonPreloads(query).Find(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) EndAssignment(ctx context.Context, id uint, endDate time.Time) error {
	// First check if the assignment exists and get the current record
	var assignment domain.ProjectEmployee
	err := r.DB.WithContext(ctx).First(&assignment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
		}
		return err
	}

	// Validate the assignment has valid data
	if assignment.EmployeeID == 0 {
		return domain.NewValidationError("Dữ liệu phân công không hợp lệ")
	}

	// Use GORM's Update method which automatically handles updated_at
	return r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id = ?", id).
		Update("last_date", endDate).Error
}

func (r *ProjectEmployeeRepository) applyFilters(query *gorm.DB, filters domain.ProjectEmployeeFilters) *gorm.DB {
	// Filter by project
	if filters.ProjectID != nil {
		query = query.Where("project_id = ?", *filters.ProjectID)

		today := clock.NowUTC().Truncate(24 * time.Hour)
		// Filter by status
		switch filters.Status {
		case "current":
			query = query.Where("start_date <= ? AND (last_date IS NULL OR last_date >= ?)", today, today)
		case "upcoming":
			query = query.Where("start_date > ?", today)
		default:
			// Default case: return both current and upcoming, so exclude past employees
			query = query.Where("last_date IS NULL OR last_date >= ?", today)
		}
	}

	// Filter by employee
	if filters.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filters.EmployeeID)
	}

	// Filter by creator
	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}

	// Filter by date range
	if filters.FromDate != nil {
		query = query.Where("start_date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("start_date <= ?", *filters.ToDate)
	}

	// Filter active only (no end date)
	if filters.ActiveOnly {
		query = query.Where("last_date IS NULL")
	}

	// Filter by payment schedule
	if filters.PaymentSchedule != nil {
		query = query.Where("payment_schedule = ?", string(*filters.PaymentSchedule))
	}

	return query
}

// Additional helper methods

func (r *ProjectEmployeeRepository) GetActiveEmployeesForProject(ctx context.Context, projectID uint) ([]*domain.Employee, error) {
	return r.GetActiveEmployeesForProjectWithCreatorFilter(ctx, projectID, nil)
}

func (r *ProjectEmployeeRepository) GetActiveEmployeesForProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*domain.Employee, error) {
	var employees []*domain.Employee

	query := r.DB.WithContext(ctx).
		Model(&domain.Employee{}).
		Select("employees.*").
		Joins("INNER JOIN project_employees pe ON employees.id = pe.employee_id AND pe.deleted_at IS NULL").
		Where("pe.project_id = ? AND pe.last_date IS NULL", projectID)

	// Apply creator filter if provided
	if createdBy != nil {
		query = query.Where("employees.created_by = ?", *createdBy)
	}

	err := query.Find(&employees).Error
	return employees, err
}

func (r *ProjectEmployeeRepository) GetActiveProjectsForEmployee(ctx context.Context, employeeID uint) ([]*domain.Project, error) {
	var projects []*domain.Project

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("projects.*").
		Joins("INNER JOIN project_employees pe ON projects.id = pe.project_id AND pe.deleted_at IS NULL").
		Where("pe.employee_id = ? AND pe.last_date IS NULL AND projects.project_status = ?",
			employeeID, domain.ProjectStatusRunning).
		Find(&projects).Error

	return projects, err
}

func (r *ProjectEmployeeRepository) GetAssignmentHistory(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployeeWithDetails, error) {
	var assignments []*domain.ProjectEmployeeWithDetails

	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`
			project_employees.*,
			p.name as project_name,
			p.code as project_code,
			p.status as project_status,
			p.client_name,
			e.fullname as employee_fullname,
			e.cccd as employee_cccd,
			e.email as employee_email,
			e.status as employee_status
		`).
		Joins("LEFT JOIN projects p ON project_employees.project_id = p.id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN employees e ON project_employees.employee_id = e.id AND e.deleted_at IS NULL").
		Where("project_employees.employee_id = ?", employeeID).
		Order("project_employees.start_date DESC").
		Scan(&assignments).Error

	return assignments, err
}

func (r *ProjectEmployeeRepository) GetProjectAssignmentSummary(ctx context.Context, projectID uint) (*domain.ProjectAssignmentSummary, error) {
	var summary domain.ProjectAssignmentSummary

	// Single query with conditional aggregation to calculate all counts at once
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`
			COUNT(*) AS total_assignments,
			COUNT(CASE WHEN last_date IS NULL THEN 1 END) AS active_assignments,
			COUNT(CASE WHEN last_date IS NOT NULL THEN 1 END) AS ended_assignments
		`).
		Where("project_id = ?", projectID).
		Scan(&summary).Error

	if err != nil {
		return nil, err
	}

	summary.ProjectID = projectID
	return &summary, nil
}

func (r *ProjectEmployeeRepository) BulkEndAssignments(ctx context.Context, projectID uint, endDate time.Time, updatedBy uint) error {
	return r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("project_id = ? AND last_date IS NULL", projectID).
		Updates(map[string]any{
			"last_date":  endDate,
			"updated_at": clock.Now(),
		}).Error
}

func (r *ProjectEmployeeRepository) GetEmployeeCodeForProject(ctx context.Context, employeeID, projectID uint) (string, error) {
	var employeeCode string
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select("employee_code").
		Where("employee_id = ? AND project_id = ? AND last_date IS NULL",
			employeeID, projectID).
		Scan(&employeeCode).Error

	return employeeCode, err
}

// GetOverlappingAssignments returns assignments that overlap with given date range (pure data access)
func (r *ProjectEmployeeRepository) GetOverlappingAssignments(ctx context.Context, employeeID uint, startDate, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).Where("employee_id = ?", employeeID)
	query = r.applyCommonPreloads(query)

	if endDate != nil {
		query = query.Where("start_date <= ? AND (last_date IS NULL OR last_date > ?)", *endDate, startDate)
	} else {
		query = query.Where("(start_date <= ? AND last_date IS NULL) OR (start_date <= ? AND last_date > ?)", startDate, startDate, startDate)
	}

	err := query.Find(&assignments).Error
	return assignments, err
}

// GetCurrentProjectForEmployee returns the current active project for an employee
func (r *ProjectEmployeeRepository) GetCurrentProjectForEmployee(ctx context.Context, employeeID uint) (*domain.Project, error) {
	var project domain.Project

	err := r.DB.WithContext(ctx).
		Model(&domain.Project{}).
		Select("projects.*").
		Joins("INNER JOIN project_employees pe ON projects.id = pe.project_id AND pe.deleted_at IS NULL").
		Where("pe.employee_id = ? AND pe.last_date IS NULL", employeeID).
		Preload("Creator").
		First(&project).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No current project
		}
		return nil, err
	}

	return &project, nil
}

// GetCurrentProjectsForEmployees retrieves current projects for multiple employees in batch
func (r *ProjectEmployeeRepository) GetCurrentProjectsForEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*domain.Project, error) {
	if len(employeeIDs) == 0 {
		return make(map[uint]*domain.Project), nil
	}

	type ProjectEmployeeMapping struct {
		EmployeeID uint
		ProjectID  uint
	}

	var mappings []ProjectEmployeeMapping
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select("employee_id, project_id").
		Where("employee_id IN ? AND last_date IS NULL", employeeIDs).
		Scan(&mappings).Error

	if err != nil {
		return nil, err
	}

	if len(mappings) == 0 {
		return make(map[uint]*domain.Project), nil
	}

	// Extract unique project IDs
	projectIDs := make([]uint, 0, len(mappings))
	projectIDSet := make(map[uint]bool)
	for _, m := range mappings {
		if !projectIDSet[m.ProjectID] {
			projectIDs = append(projectIDs, m.ProjectID)
			projectIDSet[m.ProjectID] = true
		}
	}

	// Fetch all projects at once
	var projects []*domain.Project
	err = r.DB.WithContext(ctx).
		Where("id IN ?", projectIDs).
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	// Create project lookup map
	projectMap := make(map[uint]*domain.Project)
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	// Map employees to their projects
	result := make(map[uint]*domain.Project)
	for _, m := range mappings {
		if project, exists := projectMap[m.ProjectID]; exists {
			result[m.EmployeeID] = project
		}
	}

	return result, nil
}

// getDistinctProjectEmployees returns unique employees for a project using window function
func (r *ProjectEmployeeRepository) getDistinctProjectEmployees(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	today := clock.NowUTC().Truncate(24 * time.Hour)

	// Subquery with window function to rank assignments per employee
	subQuery := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`id, ROW_NUMBER() OVER(
			PARTITION BY employee_id
			ORDER BY
				CASE WHEN start_date <= ? AND (last_date IS NULL OR last_date > ?) THEN 1 ELSE 2 END,
				start_date DESC
		) as rn`, today, today).
		Where("project_id = ?", *filters.ProjectID)

	// Apply status filters to the subquery
	switch filters.Status {
	case "current":
		subQuery = subQuery.Where("start_date <= ? AND (last_date IS NULL OR last_date > ?)", today, today)
	case "upcoming":
		subQuery = subQuery.Where("start_date > ?", today)
	default:
		// Default: exclude ended assignments
		subQuery = subQuery.Where("last_date IS NULL OR last_date > ?", today)
	}

	// Main query joins with the subquery to get only the top-ranked assignment per employee
	var assignments []*domain.ProjectEmployee
	query := r.DB.WithContext(ctx).
		Joins("JOIN (?) AS ranked_assignments ON project_employees.id = ranked_assignments.id AND ranked_assignments.rn = 1", subQuery).
		// Left join to get the most recent timesheet date per employee for this project
		Joins(`LEFT JOIN (
			SELECT employee_id, MAX(date) AS last_timesheet_date
			FROM timesheets
			WHERE project_id = ? AND deleted_at IS NULL
			GROUP BY employee_id
		) AS ts_summary ON ts_summary.employee_id = project_employees.employee_id`, *filters.ProjectID)

	// Apply creator filter if provided
	if filters.CreatedBy != nil {
		query = query.Joins("INNER JOIN employees e ON project_employees.employee_id = e.id").
			Where("e.created_by = ?", *filters.CreatedBy)
	}

	query = r.applyCommonPreloads(query)

	// Sort by the most recent of: assignment created_at vs last timesheet date.
	// This puts recently active employees first regardless of when they were assigned.
	query = query.Order("GREATEST(project_employees.created_at, COALESCE(ts_summary.last_timesheet_date, '0001-01-01')) DESC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&assignments).Error
	return assignments, err
}

// countDistinctProjectEmployees returns count of unique employees for a project
func (r *ProjectEmployeeRepository) countDistinctProjectEmployees(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	today := clock.NowUTC().Truncate(24 * time.Hour)

	query := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select("COUNT(DISTINCT project_employees.employee_id)").
		Where("project_employees.project_id = ?", *filters.ProjectID)

	// Apply creator filter if provided
	if filters.CreatedBy != nil {
		query = query.Joins("INNER JOIN employees e ON project_employees.employee_id = e.id").
			Where("e.created_by = ?", *filters.CreatedBy)
	}

	// Apply status filters
	switch filters.Status {
	case "current":
		query = query.Where("start_date <= ? AND (last_date IS NULL OR last_date > ?)", today, today)
	case "upcoming":
		query = query.Where("start_date > ?", today)
	default:
		// Default: exclude ended assignments (last_date <= today)
		query = query.Where("last_date IS NULL OR last_date > ?", today)
	}

	var count int64
	err := query.Scan(&count).Error
	return count, err
}

// DeleteAssignmentsByEmployeeID soft deletes all project assignments for an employee
func (r *ProjectEmployeeRepository) DeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error {
	return r.DB.WithContext(ctx).
		Where("employee_id = ?", employeeID).
		Delete(&domain.ProjectEmployee{}).Error
}

// CountWorkingEmployees returns the count of distinct employees with active assignments
func (r *ProjectEmployeeRepository) CountWorkingEmployees(ctx context.Context) (int, error) {
	// Use DB/local timezone and include only assignments that have started
	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("(last_date IS NULL OR last_date >= ?)", today).
		Where("start_date <= ?", today).
		Distinct("employee_id").
		Count(&count).Error

	return int(count), err
}

// GetEmployeesByPaymentSchedule retrieves all employees with a specific payment schedule
func (r *ProjectEmployeeRepository) GetEmployeesByPaymentSchedule(ctx context.Context, schedule domain.PaymentSchedule) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee

	query := r.DB.WithContext(ctx).
		Where("payment_schedule = ?", string(schedule))

	err := r.applyCommonPreloads(query).Find(&assignments).Error
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

// CountActiveEmployeesByPaymentSchedule counts distinct current employees assigned to a schedule
func (r *ProjectEmployeeRepository) CountActiveEmployeesByPaymentSchedule(ctx context.Context, schedule domain.PaymentSchedule) (int, error) {
	var count int64
	now := clock.Now()

	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("payment_schedule = ?", string(schedule)).
		Where("start_date <= ?", now).
		Where("last_date IS NULL OR last_date >= ?", now).
		Distinct("employee_id").
		Count(&count).Error

	return int(count), err
}

// GetEmployeesWithPendingScheduleChanges retrieves employees whose pending schedule changes should be applied
func (r *ProjectEmployeeRepository) GetEmployeesWithPendingScheduleChanges(ctx context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee

	query := r.DB.WithContext(ctx).
		Where("pending_payment_schedule IS NOT NULL").
		Where("schedule_effective_from IS NOT NULL").
		Where("schedule_effective_from <= ?", effectiveDate)

	err := r.applyCommonPreloads(query).Find(&assignments).Error
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

// ApplyScheduleChanges applies pending payment schedule changes for specified employees
func (r *ProjectEmployeeRepository) ApplyScheduleChanges(ctx context.Context, employeeIDs []uint) error {
	if len(employeeIDs) == 0 {
		return nil
	}

	// Update payment_schedule from pending_payment_schedule and clear pending fields
	return r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Where("id IN ?", employeeIDs).
		Where("pending_payment_schedule IS NOT NULL").
		Updates(map[string]interface{}{
			"payment_schedule":         gorm.Expr("pending_payment_schedule"),
			"pending_payment_schedule": nil,
			"schedule_effective_from":  nil,
		}).Error
}

// HasAccessViaProject checks if user can access employee through project assignments
func (r *ProjectEmployeeRepository) HasAccessViaProject(ctx context.Context, employeeID, userID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Joins("INNER JOIN project_users ON project_employees.project_id = project_users.project_id").
		Where("project_employees.employee_id = ?", employeeID).
		Where("project_users.user_id = ?", userID).
		Where("project_employees.deleted_at IS NULL").
		Where("project_users.deleted_at IS NULL").
		Where("project_employees.start_date <= NOW()").
		Where("project_employees.last_date IS NULL OR project_employees.last_date > NOW()").
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}
