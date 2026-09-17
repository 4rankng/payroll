package employee

import (
	"context"
	"time"

	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/pkg/clock"
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

// SuggestAssignmentStart returns the default start date for a new assignment:
// day after the employee's last recorded timesheet (any project), else the 1st
// of the current month. Callers that received an explicit start date must keep it.
func (s *EmployeeService) SuggestAssignmentStart(ctx context.Context, employeeID uint) (time.Time, error) {
	latest, err := s.TimesheetRepo.GetLatestTimesheetDateByEmployeeID(ctx, employeeID)
	if err != nil {
		return time.Time{}, err
	}
	return domainServices.SuggestAssignmentStart(latest, clock.Now()), nil
}

// UpdateUserLink sets the user_id on an employee record (targeted update).
func (s *EmployeeService) UpdateUserLink(ctx context.Context, employeeID uint, userID uint) error {
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, map[string]any{"user_id": userID})
}
