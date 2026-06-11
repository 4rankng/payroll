package employee

import (
	"context"

	"api-server/internal/domain"
)

// GetActiveAssignment returns the active project assignment for an employee, if any.
func (s *EmployeeService) GetActiveAssignment(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	return s.ProjectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
}

// GetActiveAssignments returns all active assignments for a project.
func (s *EmployeeService) GetActiveAssignments(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	return s.ProjectEmployeeRepo.GetActiveAssignments(ctx, projectID)
}

// CreateAssignment creates a new project-employee assignment.
func (s *EmployeeService) CreateAssignment(ctx context.Context, assignment *domain.ProjectEmployee) error {
	return s.ProjectEmployeeRepo.Create(ctx, assignment)
}

// UpdateUserLink sets the user_id on an employee record (targeted update).
func (s *EmployeeService) UpdateUserLink(ctx context.Context, employeeID uint, userID uint) error {
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, map[string]any{"user_id": userID})
}
