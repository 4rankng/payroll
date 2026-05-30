package query_builders

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// EmployeeProjectQueryBuilder handles complex query building for employee-project operations
type EmployeeProjectQueryBuilder struct {
	db *gorm.DB
}

// NewEmployeeProjectQueryBuilder creates a new employee project query builder
func NewEmployeeProjectQueryBuilder(db *gorm.DB) *EmployeeProjectQueryBuilder {
	return &EmployeeProjectQueryBuilder{
		db: db,
	}
}

// ProjectAssignment represents a project assignment for an employee
type ProjectAssignment struct {
	EmployeeID        uint       `gorm:"column:employee_id"`
	ProjectID         uint       `gorm:"column:project_id"`
	ProjectName       string     `gorm:"column:project_name"`
	ProjectCode       string     `gorm:"column:project_code"`
	ProjectClientName string     `gorm:"column:project_client_name"`
	Position          string     `gorm:"column:position"`
	StartDate         time.Time  `gorm:"column:start_date"`
	LastDate          *time.Time `gorm:"column:last_date"`
}

// ProjectAssignmentWithPayment represents a project assignment with payment schedule info
type ProjectAssignmentWithPayment struct {
	EmployeeID             uint       `gorm:"column:employee_id"`
	ProjectEmployeeID      uint       `gorm:"column:project_employee_id"`
	ProjectID              uint       `gorm:"column:project_id"`
	ProjectName            string     `gorm:"column:project_name"`
	ProjectCode            string     `gorm:"column:project_code"`
	ProjectClientName      string     `gorm:"column:project_client_name"`
	Position               string     `gorm:"column:position"`
	StartDate              time.Time  `gorm:"column:start_date"`
	LastDate               *time.Time `gorm:"column:last_date"`
	PaymentSchedule        string     `gorm:"column:payment_schedule"`
	PendingPaymentSchedule *string    `gorm:"column:pending_payment_schedule"`
	ScheduleEffectiveFrom  *time.Time `gorm:"column:schedule_effective_from"`
	IsFlexible             bool       `gorm:"column:is_flexible"`
	CheckInEnabled         bool       `gorm:"column:check_in_enabled"`
}

// BuildListWithProjectsQuery builds the employee query for ListWithProjects
func (b *EmployeeProjectQueryBuilder) BuildListWithProjectsQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).Preload("Bank").Preload("User")

	// Apply filters for employee selection
	query = b.applyFilters(query, filters)

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

// BuildListWithAllProjectsQuery builds the employee query for ListWithAllProjects
func (b *EmployeeProjectQueryBuilder) BuildListWithAllProjectsQuery(filters domain.EmployeeFilters) *gorm.DB {
	return b.BuildListWithProjectsQuery(filters)
}

// BuildProjectAssignmentsQuery builds a query to fetch project assignments for employees
func (b *EmployeeProjectQueryBuilder) BuildProjectAssignmentsQuery(ctx context.Context, employeeIDs []uint) *gorm.DB {
	return b.db.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`project_employees.employee_id, project_employees.project_id, p.name as project_name, p.code as project_code,
			p.client_name as project_client_name, project_employees.position, project_employees.start_date, project_employees.last_date`).
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.employee_id IN ? AND project_employees.last_date IS NULL AND project_employees.start_date <= CURDATE()", employeeIDs)
}

// BuildProjectAssignmentsWithPaymentQuery builds a query to fetch project assignments with payment info
func (b *EmployeeProjectQueryBuilder) BuildProjectAssignmentsWithPaymentQuery(ctx context.Context, employeeIDs []uint) *gorm.DB {
	return b.db.WithContext(ctx).
		Model(&domain.ProjectEmployee{}).
		Select(`project_employees.id as project_employee_id, project_employees.employee_id, project_employees.project_id, p.name as project_name, p.code as project_code,
			p.client_name as project_client_name, project_employees.position, project_employees.start_date, project_employees.last_date,
			project_employees.payment_schedule, project_employees.pending_payment_schedule, project_employees.schedule_effective_from,
			p.is_flexible, project_employees.check_in_enabled`).
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.employee_id IN ? AND project_employees.last_date IS NULL AND project_employees.start_date <= CURDATE()", employeeIDs)
}

// FetchProjectAssignments fetches project assignments for the given employee IDs
func (b *EmployeeProjectQueryBuilder) FetchProjectAssignments(ctx context.Context, employeeIDs []uint) ([]ProjectAssignment, error) {
	var assignments []ProjectAssignment
	err := b.BuildProjectAssignmentsQuery(ctx, employeeIDs).Find(&assignments).Error
	return assignments, err
}

// FetchProjectAssignmentsWithPayment fetches project assignments with payment info for the given employee IDs
func (b *EmployeeProjectQueryBuilder) FetchProjectAssignmentsWithPayment(ctx context.Context, employeeIDs []uint) ([]ProjectAssignmentWithPayment, error) {
	var assignments []ProjectAssignmentWithPayment
	err := b.BuildProjectAssignmentsWithPaymentQuery(ctx, employeeIDs).Find(&assignments).Error
	return assignments, err
}

// GroupAssignmentsByEmployee groups project assignments by employee ID
func (b *EmployeeProjectQueryBuilder) GroupAssignmentsByEmployee(assignments []ProjectAssignment) map[uint][]ProjectAssignment {
	grouped := make(map[uint][]ProjectAssignment)
	for _, assignment := range assignments {
		grouped[assignment.EmployeeID] = append(grouped[assignment.EmployeeID], assignment)
	}
	return grouped
}

// GroupAssignmentsWithPaymentByEmployee groups project assignments with payment info by employee ID
func (b *EmployeeProjectQueryBuilder) GroupAssignmentsWithPaymentByEmployee(assignments []ProjectAssignmentWithPayment) map[uint][]ProjectAssignmentWithPayment {
	grouped := make(map[uint][]ProjectAssignmentWithPayment)
	for _, assignment := range assignments {
		grouped[assignment.EmployeeID] = append(grouped[assignment.EmployeeID], assignment)
	}
	return grouped
}

// MapToEmployeeWithProject maps employees and assignments to EmployeeWithProject slice
func (b *EmployeeProjectQueryBuilder) MapToEmployeeWithProject(employees []*domain.Employee, assignments []ProjectAssignment) []*domain.EmployeeWithProject {
	projectsByEmployee := b.GroupAssignmentsByEmployee(assignments)

	var empWithProjects []*domain.EmployeeWithProject
	for _, emp := range employees {
		empWithProject := &domain.EmployeeWithProject{
			Employee: *emp,
		}

		if projects, exists := projectsByEmployee[emp.ID]; exists && len(projects) > 0 {
			firstProject := projects[0]
			empWithProject.ProjectID = &firstProject.ProjectID
			empWithProject.ProjectName = &firstProject.ProjectName
			empWithProject.ProjectCode = &firstProject.ProjectCode
			empWithProject.ProjectClientName = &firstProject.ProjectClientName
			empWithProject.Position = &firstProject.Position
			empWithProject.StartDate = &firstProject.StartDate
			empWithProject.LastDate = firstProject.LastDate
		}

		empWithProjects = append(empWithProjects, empWithProject)
	}

	return empWithProjects
}

// MapToEmployeeWithProjects maps employees and assignments to EmployeeWithProjects slice
func (b *EmployeeProjectQueryBuilder) MapToEmployeeWithProjects(employees []*domain.Employee, assignments []ProjectAssignmentWithPayment) []*domain.EmployeeWithProjects {
	projectsByEmployee := b.GroupAssignmentsWithPaymentByEmployee(assignments)

	var empWithProjects []*domain.EmployeeWithProjects
	for _, emp := range employees {
		empWithAllProjects := &domain.EmployeeWithProjects{
			Employee:        *emp,
			CurrentProjects: []domain.CurrentProject{},
		}

		if assignments, exists := projectsByEmployee[emp.ID]; exists {
			for _, assignment := range assignments {
				project := domain.CurrentProject{
					ProjectID:              assignment.ProjectID,
					ProjectEmployeeID:      assignment.ProjectEmployeeID,
					Name:                   assignment.ProjectName,
					Code:                   assignment.ProjectCode,
					ClientName:             assignment.ProjectClientName,
					Position:               assignment.Position,
					StartDate:              assignment.StartDate,
					LastDate:               assignment.LastDate,
					PaymentSchedule:        assignment.PaymentSchedule,
					PendingPaymentSchedule: assignment.PendingPaymentSchedule,
					ScheduleEffectiveFrom:  assignment.ScheduleEffectiveFrom,
					IsFlexible:             assignment.IsFlexible,
					CheckInEnabled:         assignment.CheckInEnabled,
				}
				empWithAllProjects.CurrentProjects = append(empWithAllProjects.CurrentProjects, project)
			}
		}

		empWithProjects = append(empWithProjects, empWithAllProjects)
	}

	return empWithProjects
}

// BuildGetByProjectQuery builds a query for getting employees by project
func (b *EmployeeProjectQueryBuilder) BuildGetByProjectQuery(projectID uint, createdBy *uint) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).
		Select("employees.*").
		Joins("INNER JOIN project_employees pe ON employees.id = pe.employee_id AND pe.deleted_at IS NULL").
		Where("pe.project_id = ? AND pe.last_date IS NULL", projectID)

	// Apply creator filter if provided
	if createdBy != nil {
		query = query.Where("employees.created_by = ?", *createdBy)
	}

	return query.Preload("Creator")
}

// buildMissingBankDetailsBaseQuery builds the shared base query for missing bank details queries.
// Filters employees who:
//   - Have missing banking information (no bank or empty account details)
//   - Are currently assigned to active projects
//   - Have at least one pending timesheet or pending advance payment request
func (b *EmployeeProjectQueryBuilder) buildMissingBankDetailsBaseQuery(filters domain.EmployeeFilters) *gorm.DB {
	activeProjectSubquery := b.db.Model(&domain.ProjectEmployee{}).
		Select("DISTINCT project_employees.employee_id").
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.last_date IS NULL AND project_employees.start_date <= CURDATE()")

	query := b.db.Model(&domain.Employee{})

	query = b.applyFilters(query, filters)

	// Missing banking information
	query = query.Where("employees.bank_id IS NULL OR (employees.bank_account_number = '' AND employees.bank_account_name = '')")

	// Currently assigned to active projects
	query = query.Where("employees.id IN (?)", activeProjectSubquery)

	// Has pending timesheet or pending advance payment request
	query = query.Where(`(
		EXISTS (SELECT 1 FROM timesheets t WHERE t.employee_id = employees.id AND t.timesheet_status = 'pending_approval')
		OR
		EXISTS (SELECT 1 FROM advance_payment_requests apr WHERE apr.employee_id = employees.id AND apr.status = 'PENDING')
	)`)

	return query
}

// BuildMissingBankDetailsQuery builds a query for getting employees with missing banking details
func (b *EmployeeProjectQueryBuilder) BuildMissingBankDetailsQuery(filters domain.EmployeeFilters) *gorm.DB {
	query := b.buildMissingBankDetailsBaseQuery(filters).Preload("Bank")

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

// BuildMissingBankDetailsCountQuery builds a query for counting employees with missing banking details
func (b *EmployeeProjectQueryBuilder) BuildMissingBankDetailsCountQuery(filters domain.EmployeeFilters) *gorm.DB {
	return b.buildMissingBankDetailsBaseQuery(filters)
}

// BuildUnassignedAtDateQuery builds a query for getting unassigned employees at a specific date
func (b *EmployeeProjectQueryBuilder) BuildUnassignedAtDateQuery(atDate time.Time, filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{}).Preload("Creator")

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

// BuildUnassignedAtDateCountQuery builds a query for counting unassigned employees at a specific date
func (b *EmployeeProjectQueryBuilder) BuildUnassignedAtDateCountQuery(atDate time.Time, filters domain.EmployeeFilters) *gorm.DB {
	query := b.db.Model(&domain.Employee{})

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

	return query
}

// applyFilters applies filters to a project query
func (b *EmployeeProjectQueryBuilder) applyFilters(query *gorm.DB, filters domain.EmployeeFilters) *gorm.DB {
	// Filter by creator
	if filters.CreatedBy != nil {
		query = query.Where("employees.created_by = ?", *filters.CreatedBy)
	}

	// Filter for accessible employees (owned, shared, or assigned to projects)
	if filters.AccessibleBy != nil {
		if !filters.SkipAccessibilityFilter {
			userID := *filters.AccessibleBy
			// Get shared employee IDs for the user via employee_users table
			var sharedEmployeeIDs []uint
			err := b.db.Model(&domain.EmployeeUser{}).
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
		}
	}

	// Filter by project (requires join)
	if filters.ProjectID != nil {
		subQuery := b.db.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id = ? AND last_date IS NULL AND deleted_at IS NULL", *filters.ProjectID)
		query = query.Where("employees.id IN (?)", subQuery)
	}

	// Filter by multiple projects
	if len(filters.ProjectIDs) > 0 {
		subQuery := b.db.Model(&domain.ProjectEmployee{}).
			Select("employee_id").
			Where("project_id IN ? AND last_date IS NULL AND deleted_at IS NULL", filters.ProjectIDs)
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
		endOfDay := filters.ToDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		query = query.Where("employees.created_at <= ?", endOfDay)
	}

	// Search functionality with Vietnamese normalization using virtual column
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("employees.search_normalized LIKE ?", normalizedSearch)
	}

	return query
}
