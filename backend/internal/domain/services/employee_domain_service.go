package services

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

// EmployeeDomainService handles business logic for employee operations
type EmployeeDomainService struct {
	employeeRepo        domain.EmployeeRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	timesheetRepo       domain.TimesheetRepository
}

// NewEmployeeDomainService creates a new employee domain service
func NewEmployeeDomainService(
	employeeRepo domain.EmployeeRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	timesheetRepo domain.TimesheetRepository,
) *EmployeeDomainService {
	return &EmployeeDomainService{
		employeeRepo:        employeeRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		timesheetRepo:       timesheetRepo,
	}
}

// GetEmployeeStatistics calculates employee statistics using business rules
func (s *EmployeeDomainService) GetEmployeeStatistics(ctx context.Context) (*domain.EmployeeStatistics, error) {
	// Get basic employee count using list with no filters
	employees, err := s.employeeRepo.List(ctx, domain.EmployeeFilters{})
	if err != nil {
		return nil, domain.NewInternalError("Lỗi lấy danh sách nhân viên", err)
	}

	stats := &domain.EmployeeStatistics{
		TotalEmployees: int64(len(employees)),
	}

	// Count employees with banking info
	var withBankingInfo int64
	for _, emp := range employees {
		if emp.BankAccountNumber != "" && emp.BankAccountName != "" {
			withBankingInfo++
		}
	}
	stats.WithBankingInfo = withBankingInfo

	// Count employees assigned to projects
	var assignedCount int64
	for _, emp := range employees {
		hasActiveAssignment, err := s.hasActiveProjectAssignment(ctx, emp.ID)
		if err != nil {
			return nil, domain.NewInternalError("Lỗi kiểm tra phân công dự án", err)
		}
		if hasActiveAssignment {
			assignedCount++
		}
	}
	stats.AssignedToProjects = assignedCount
	stats.UnassignedEmployees = stats.TotalEmployees - assignedCount

	return stats, nil
}

// GetEmployeesSummary generates comprehensive employee summary with business metrics
func (s *EmployeeDomainService) GetEmployeesSummary(ctx context.Context) (*domain.EmployeesSummary, error) {
	// Use repository method for efficient summary calculation
	return s.employeeRepo.GetEmployeesSummary(ctx)
}

// GetEmployeesSummaryForCreator gets summary for employees accessible to a specific user
// Includes employees created by the user, shared with the user, or assigned to projects by the user
func (s *EmployeeDomainService) GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*domain.EmployeesSummary, error) {
	// Use repository method with accessibility filtering
	return s.employeeRepo.GetEmployeesSummaryForCreator(ctx, createdBy)
}

// SearchEmployees performs normalized Vietnamese search
func (s *EmployeeDomainService) SearchEmployees(ctx context.Context, query string, limit int) ([]*domain.EmployeeWithProject, error) {
	// Normalize search query for Vietnamese compatibility
	normalizedQuery := utils.NormalizeVietnameseForSearch(query)

	// Delegate to repository for data retrieval but keep search logic here
	return s.employeeRepo.SearchEmployees(ctx, normalizedQuery, limit)
}

// CanDeleteEmployee validates if an employee can be deleted and returns project assignments to be deleted
func (s *EmployeeDomainService) CanDeleteEmployee(ctx context.Context, employeeID uint) (bool, []*domain.ProjectEmployee, error) {
	// Check if employee has any timesheets
	timesheetCount, err := s.timesheetRepo.CountTimesheetsByEmployeeID(ctx, employeeID)
	if err != nil {
		return false, nil, domain.NewInternalError("Lỗi kiểm tra bảng chấm công của nhân viên", err)
	}

	if timesheetCount > 0 {
		return false, nil, domain.NewValidationError(constants.MsgCannotDeleteEmployeeHasTimesheetsVN)
	}

	// Get all project assignments for this employee
	filters := domain.ProjectEmployeeFilters{
		EmployeeID: &employeeID,
	}
	assignments, err := s.projectEmployeeRepo.List(ctx, filters)
	if err != nil {
		return false, nil, domain.NewInternalError("Lỗi lấy danh sách phân công dự án", err)
	}

	return true, assignments, nil
}

// Private helper methods

func (s *EmployeeDomainService) hasActiveProjectAssignment(ctx context.Context, employeeID uint) (bool, error) {
	filters := domain.ProjectEmployeeFilters{
		EmployeeID: &employeeID,
		ActiveOnly: true,
	}

	assignments, err := s.projectEmployeeRepo.List(ctx, filters)
	if err != nil {
		return false, domain.NewInternalError("Lỗi lấy danh sách phân công", err)
	}

	return len(assignments) > 0, nil
}

// GetUnassignedEmployeesAtDate gets employees who are not assigned to any project at the specified date
func (s *EmployeeDomainService) GetUnassignedEmployeesAtDate(ctx context.Context, date time.Time, page, pageSize int) ([]*domain.Employee, error) {
	// Get all active project assignments (active means LastDate is null or after the given date)
	assignmentFilters := domain.ProjectEmployeeFilters{
		ActiveOnly: true,
	}

	assignments, err := s.projectEmployeeRepo.List(ctx, assignmentFilters)
	if err != nil {
		return nil, domain.NewInternalError("Lỗi lấy danh sách phân công dự án", err)
	}

	// Filter assignments that are active on the given date
	assignedEmployeeIDs := make(map[uint]bool)
	for _, assignment := range assignments {
		// Assignment is active if start date <= given date and (end date is null or end date >= given date)
		if assignment.StartDate.Before(date) || assignment.StartDate.Equal(date) {
			if assignment.LastDate == nil || assignment.LastDate.After(date) || assignment.LastDate.Equal(date) {
				assignedEmployeeIDs[assignment.EmployeeID] = true
			}
		}
	}

	// Get all employees and filter out assigned ones
	filters := domain.EmployeeFilters{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	allEmployees, err := s.employeeRepo.List(ctx, filters)
	if err != nil {
		return nil, domain.NewInternalError("Lỗi lấy danh sách nhân viên", err)
	}

	// Filter out assigned employees
	var unassignedEmployees []*domain.Employee
	for _, employee := range allEmployees {
		if !assignedEmployeeIDs[employee.ID] {
			unassignedEmployees = append(unassignedEmployees, employee)
		}
	}

	return unassignedEmployees, nil
}

// CountUnassignedEmployeesAtDate counts employees who are not assigned to any project at the specified date
func (s *EmployeeDomainService) CountUnassignedEmployeesAtDate(ctx context.Context, date time.Time) (int64, error) {
	// Get all active project assignments (active means LastDate is null or after the given date)
	assignmentFilters := domain.ProjectEmployeeFilters{
		ActiveOnly: true,
	}

	assignments, err := s.projectEmployeeRepo.List(ctx, assignmentFilters)
	if err != nil {
		return 0, domain.NewInternalError("Lỗi lấy danh sách phân công dự án", err)
	}

	// Filter assignments that are active on the given date
	assignedEmployeeIDs := make(map[uint]bool)
	for _, assignment := range assignments {
		// Assignment is active if start date <= given date and (end date is null or end date >= given date)
		if assignment.StartDate.Before(date) || assignment.StartDate.Equal(date) {
			if assignment.LastDate == nil || assignment.LastDate.After(date) || assignment.LastDate.Equal(date) {
				assignedEmployeeIDs[assignment.EmployeeID] = true
			}
		}
	}

	// Get total employee count
	totalEmployees, err := s.employeeRepo.Count(ctx, domain.EmployeeFilters{})
	if err != nil {
		return 0, domain.NewInternalError("Lỗi đếm tổng số nhân viên", err)
	}

	// Calculate unassigned count
	unassignedCount := totalEmployees - int64(len(assignedEmployeeIDs))
	if unassignedCount < 0 {
		unassignedCount = 0
	}

	return unassignedCount, nil
}
