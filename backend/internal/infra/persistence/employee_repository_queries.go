package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
)

func (r *EmployeeRepository) List(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.queryBuilder.BuildListQuery(filters)

	err := query.Find(&employees).Error
	return employees, err
}

// ListWithProjects returns employees with their current active project information
func (r *EmployeeRepository) ListWithProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProject, error) {
	// Get employees using the query builder
	employeesQuery := r.projectQueryBuilder.BuildListWithProjectsQuery(filters)
	var employees []*domain.Employee
	err := employeesQuery.Find(&employees).Error
	if err != nil {
		return nil, err
	}

	if len(employees) == 0 {
		return []*domain.EmployeeWithProject{}, nil
	}

	// Extract employee IDs for project lookup
	employeeIDs := make([]uint, len(employees))
	for i, emp := range employees {
		employeeIDs[i] = emp.ID
	}

	// Get project assignments
	assignments, err := r.projectQueryBuilder.FetchProjectAssignments(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}

	// Map to EmployeeWithProject
	return r.projectQueryBuilder.MapToEmployeeWithProject(employees, assignments), nil
}

// ListWithAllProjects returns employees with all their current active projects
func (r *EmployeeRepository) ListWithAllProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	// Get employees using the query builder
	employeesQuery := r.projectQueryBuilder.BuildListWithAllProjectsQuery(filters)
	var employees []*domain.Employee
	err := employeesQuery.Find(&employees).Error
	if err != nil {
		return nil, err
	}

	if len(employees) == 0 {
		return []*domain.EmployeeWithProjects{}, nil
	}

	// Extract employee IDs for project lookup
	employeeIDs := make([]uint, len(employees))
	for i, emp := range employees {
		employeeIDs[i] = emp.ID
	}

	// Get project assignments with payment info
	assignments, err := r.projectQueryBuilder.FetchProjectAssignmentsWithPayment(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}

	// Map to EmployeeWithProjects
	return r.projectQueryBuilder.MapToEmployeeWithProjects(employees, assignments), nil
}

// GetEmployeesWithMissingBankDetails returns employees with missing banking information
func (r *EmployeeRepository) GetEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	// Get employees with missing bank details
	employeesQuery := r.projectQueryBuilder.BuildMissingBankDetailsQuery(filters)
	var employees []*domain.Employee
	err := employeesQuery.Find(&employees).Error
	if err != nil {
		return nil, err
	}

	if len(employees) == 0 {
		return []*domain.EmployeeWithProjects{}, nil
	}

	// Extract employee IDs for project lookup
	employeeIDs := make([]uint, len(employees))
	for i, emp := range employees {
		employeeIDs[i] = emp.ID
	}

	// Get project assignments
	assignments, err := r.projectQueryBuilder.FetchProjectAssignments(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}

	// Convert to EmployeeWithProjects format
	projectsByEmployee := r.projectQueryBuilder.GroupAssignmentsByEmployee(assignments)

	var empWithProjects []*domain.EmployeeWithProjects
	for _, emp := range employees {
		empWithAllProjects := &domain.EmployeeWithProjects{
			Employee:        *emp,
			CurrentProjects: []domain.CurrentProject{},
		}

		// Add all projects for this employee
		if projects, exists := projectsByEmployee[emp.ID]; exists {
			for _, assignment := range projects {
				project := domain.CurrentProject{
					ProjectID:  assignment.ProjectID,
					Name:       assignment.ProjectName,
					Code:       assignment.ProjectCode,
					ClientName: assignment.ProjectClientName,
					Position:   assignment.Position,
					StartDate:  assignment.StartDate,
					LastDate:   assignment.LastDate,
				}
				empWithAllProjects.CurrentProjects = append(empWithAllProjects.CurrentProjects, project)
			}
		}

		empWithProjects = append(empWithProjects, empWithAllProjects)
	}

	return empWithProjects, nil
}

// CountEmployeesWithMissingBankDetails returns count of employees with missing banking information
func (r *EmployeeRepository) CountEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	var count int64
	err := r.projectQueryBuilder.BuildMissingBankDetailsCountQuery(filters).Count(&count).Error
	return count, err
}

func (r *EmployeeRepository) Count(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	// If AccessibleBy filter is present, use two separate queries approach for accurate count
	if filters.AccessibleBy != nil {
		return r.countWithTwoQueries(ctx, filters)
	}

	// For other filters, use the statistics builder
	var count int64
	query := r.statisticsBuilder.BuildCountQuery(filters)
	err := query.Count(&count).Error
	return count, err
}

// countWithTwoQueries performs separate count queries for better accuracy with AccessibleBy filter
func (r *EmployeeRepository) countWithTwoQueries(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	userID := *filters.AccessibleBy

	// First, get shared employee IDs from employee_users table
	var sharedEmployeeIDs []uint
	err := r.DB.Model(&domain.EmployeeUser{}).
		Select("employee_id").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Pluck("employee_id", &sharedEmployeeIDs).Error

	if err != nil {
		sharedEmployeeIDs = []uint{} // Continue with empty shared employees
	}

	// Get assigned employee IDs from project_employees table
	var assignedEmployeeIDs []uint
	err = r.DB.Model(&domain.ProjectEmployee{}).
		Select("DISTINCT employee_id").
		Where("created_by = ? AND deleted_at IS NULL", userID).
		Pluck("employee_id", &assignedEmployeeIDs).Error

	if err != nil {
		assignedEmployeeIDs = []uint{} // Continue with empty assigned employees
	}

	// Track unique employee IDs to avoid double counting
	uniqueEmployeeIDs := make(map[uint]bool)

	// Count 1: Employees created by the user
	ownedFilters := filters
	ownedFilters.CreatedBy = &userID
	ownedFilters.AccessibleBy = nil // Remove AccessibleBy to prevent recursion
	ownedFilters.SkipAccessibilityFilter = true

	ownedQuery := r.queryBuilder.BuildListQuery(ownedFilters)

	// Get owned employee IDs that match filters
	var ownedEmployeeIDs []uint
	err = ownedQuery.Pluck("id", &ownedEmployeeIDs).Error
	if err != nil {
		return 0, err
	}

	// Add owned employees to unique set
	for _, id := range ownedEmployeeIDs {
		uniqueEmployeeIDs[id] = true
	}

	// Count 2: Employees shared with the user (only if there are shared employees)
	if len(sharedEmployeeIDs) > 0 {
		sharedFilters := filters
		sharedFilters.AccessibleBy = nil // Remove AccessibleBy to prevent recursion
		sharedFilters.CreatedBy = nil    // Remove CreatedBy filter
		sharedFilters.SkipAccessibilityFilter = true

		sharedQuery := r.DB.WithContext(ctx).Model(&domain.Employee{}).
			Where("id IN (?)", sharedEmployeeIDs)
		sharedQuery = r.queryBuilder.ApplyFilters(sharedQuery, sharedFilters)

		// Get shared employee IDs that match filters
		var filteredSharedEmployeeIDs []uint
		err = sharedQuery.Pluck("id", &filteredSharedEmployeeIDs).Error
		if err != nil {
			return 0, err
		}

		// Add shared employees to unique set (avoiding duplicates)
		for _, id := range filteredSharedEmployeeIDs {
			uniqueEmployeeIDs[id] = true
		}
	}

	// Count 3: Employees assigned by the user to projects (only if there are assigned employees)
	if len(assignedEmployeeIDs) > 0 {
		assignedFilters := filters
		assignedFilters.AccessibleBy = nil // Remove AccessibleBy to prevent recursion
		assignedFilters.CreatedBy = nil    // Remove CreatedBy filter
		assignedFilters.SkipAccessibilityFilter = true

		assignedQuery := r.DB.WithContext(ctx).Model(&domain.Employee{}).
			Where("id IN (?)", assignedEmployeeIDs)
		assignedQuery = r.queryBuilder.ApplyFilters(assignedQuery, assignedFilters)

		// Get assigned employee IDs that match filters
		var filteredAssignedEmployeeIDs []uint
		err = assignedQuery.Pluck("id", &filteredAssignedEmployeeIDs).Error
		if err != nil {
			return 0, err
		}

		// Add assigned employees to unique set (avoiding duplicates)
		for _, id := range filteredAssignedEmployeeIDs {
			uniqueEmployeeIDs[id] = true
		}
	}

	// Count 4: Employees in projects the user has access to via project_users table
	projectBasedFilters := filters
	projectBasedFilters.AccessibleBy = nil
	projectBasedFilters.CreatedBy = nil
	projectBasedFilters.SkipAccessibilityFilter = true

	// Get employees from projects user has access to
	projectBasedQuery := r.DB.WithContext(ctx).Model(&domain.Employee{}).
		Where(`EXISTS (
			SELECT 1 FROM project_employees pe
			INNER JOIN project_users pu ON pe.project_id = pu.project_id
			WHERE pe.employee_id = employees.id
			AND pu.user_id = ?
			AND pe.deleted_at IS NULL
			AND pu.deleted_at IS NULL
			AND pe.start_date <= NOW()
			AND (pe.last_date IS NULL OR pe.last_date > NOW())
		)`, userID)

	projectBasedQuery = r.queryBuilder.ApplyFilters(projectBasedQuery, projectBasedFilters)

	// Get project-based employee IDs that match filters
	var projectBasedEmployeeIDs []uint
	err = projectBasedQuery.Pluck("id", &projectBasedEmployeeIDs).Error
	if err != nil {
		return 0, err
	}

	// Add project-based employees to unique set (avoiding duplicates)
	for _, id := range projectBasedEmployeeIDs {
		uniqueEmployeeIDs[id] = true
	}

	totalCount := int64(len(uniqueEmployeeIDs))
	return totalCount, nil
}

func (r *EmployeeRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.Employee, error) {
	return r.GetByProjectWithCreatorFilter(ctx, projectID, nil)
}

func (r *EmployeeRepository) GetByProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.projectQueryBuilder.BuildGetByProjectQuery(projectID, createdBy)
	err := query.Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetActiveEmployees(ctx context.Context) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Find(&employees).Error

	return employees, err
}

func (r *EmployeeRepository) GetByCreator(ctx context.Context, creatorID uint) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.queryBuilder.BuildGetByCreatorQuery(creatorID)
	err := query.Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.queryBuilder.BuildUnassignedAtDateQuery(atDate, filters)

	err := query.Find(&employees).Error
	return employees, err
}

// SearchEmployees searches for employees by fullname, email, CCCD, or mobile
func (r *EmployeeRepository) SearchEmployees(ctx context.Context, search string, limit int) ([]*domain.EmployeeWithProject, error) {
	var empWithProjects []*domain.EmployeeWithProject

	query := r.queryBuilder.BuildSearchQuery(search, limit)

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var emp domain.EmployeeWithProject
		var bankID, projectID *uint
		var bankBranchName, projectName, projectCode, projectClientName, position *string

		err := rows.Scan(
			&emp.ID, &emp.Fullname, &emp.Email, &emp.CCCD, &emp.Address, &emp.Mobile,
			&emp.BankID, &emp.BankAccountNumber, &emp.BankAccountName, &emp.DateOfBirth,
			&emp.DeletedAt, &emp.CreatedBy, &emp.CreatedAt, &emp.UpdatedAt,
			&projectID, &projectName, &projectCode, &projectClientName, &position,
			&bankID, &bankBranchName,
		)
		if err != nil {
			return nil, err
		}

		// Set bank information
		if bankID != nil {
			emp.Bank = &domain.Bank{
				ID:         *bankID,
				BranchName: *bankBranchName,
			}
		}

		// Set project information
		emp.ProjectID = projectID
		emp.ProjectName = projectName
		emp.ProjectCode = projectCode
		emp.ProjectClientName = projectClientName
		emp.Position = position

		empWithProjects = append(empWithProjects, &emp)
	}

	return empWithProjects, nil
}
