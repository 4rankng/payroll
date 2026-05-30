package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// AssignmentLifecycleManager handles the lifecycle operations for project employee assignments
type AssignmentLifecycleManager interface {
	// TerminateAssignment ends an existing assignment with the given end date and reason
	TerminateAssignment(ctx context.Context, assignment *domain.ProjectEmployee, endDate time.Time, reason string) error

	// ReplaceAssignment replaces an old assignment with a new one (soft delete old, create new)
	ReplaceAssignment(ctx context.Context, oldAssignment, newAssignment *domain.ProjectEmployee) error

	// ValidateTransition validates if a transition from one assignment to another is allowed
	ValidateTransition(ctx context.Context, from, to *domain.ProjectEmployee) error
}

// DefaultAssignmentLifecycleManager implements assignment lifecycle management
type DefaultAssignmentLifecycleManager struct {
	projectEmployeeRepo domain.ProjectEmployeeRepository
}

// NewAssignmentLifecycleManager creates a new assignment lifecycle manager
func NewAssignmentLifecycleManager(projectEmployeeRepo domain.ProjectEmployeeRepository) AssignmentLifecycleManager {
	return &DefaultAssignmentLifecycleManager{
		projectEmployeeRepo: projectEmployeeRepo,
	}
}

// TerminateAssignment ends an existing assignment with the given end date
func (m *DefaultAssignmentLifecycleManager) TerminateAssignment(
	ctx context.Context,
	assignment *domain.ProjectEmployee,
	endDate time.Time,
	reason string) error {

	logger := observability.GetLogger()

	logger.InfoContext(ctx, "Terminating assignment",
		"assignment_id", assignment.ID,
		"project_id", assignment.ProjectID,
		"employee_id", assignment.EmployeeID,
		"end_date", endDate,
		"reason", reason,
	)

	// Validate end date is after start date
	if endDate.Before(assignment.StartDate) || endDate.Equal(assignment.StartDate) {
		return domain.NewValidationError(
			fmt.Sprintf("ngày kết thúc (%s) phải sau ngày bắt đầu (%s)",
				endDate.Format("2006-01-02"), assignment.StartDate.Format("2006-01-02")))
	}

	// Set the last date
	assignment.LastDate = &endDate

	// Update the assignment
	if err := m.projectEmployeeRepo.Update(ctx, assignment); err != nil {
		logger.ErrorContext(ctx, "Failed to terminate assignment",
			"assignment_id", assignment.ID,
			"error", err,
		)
		return fmt.Errorf("failed to terminate assignment: %w", err)
	}

	logger.InfoContext(ctx, "Assignment terminated successfully",
		"assignment_id", assignment.ID,
		"end_date", endDate,
	)

	return nil
}

// ReplaceAssignment replaces an old assignment with a new one
func (m *DefaultAssignmentLifecycleManager) ReplaceAssignment(
	ctx context.Context,
	oldAssignment, newAssignment *domain.ProjectEmployee) error {

	logger := observability.GetLogger()

	logger.InfoContext(ctx, "Replacing assignment",
		"old_assignment_id", oldAssignment.ID,
		"project_id", oldAssignment.ProjectID,
		"employee_id", oldAssignment.EmployeeID,
		"new_start_date", newAssignment.StartDate,
	)

	// Validate the transition is allowed
	if err := m.ValidateTransition(ctx, oldAssignment, newAssignment); err != nil {
		return err
	}

	// Soft delete the old assignment (GORM will set deleted_at)
	if err := m.projectEmployeeRepo.Delete(ctx, oldAssignment.ID); err != nil {
		logger.ErrorContext(ctx, "Failed to soft delete old assignment",
			"assignment_id", oldAssignment.ID,
			"error", err,
		)
		return fmt.Errorf("failed to soft delete old assignment: %w", err)
	}

	// Create the new assignment
	if err := m.projectEmployeeRepo.Create(ctx, newAssignment); err != nil {
		logger.ErrorContext(ctx, "Failed to create new assignment",
			"project_id", newAssignment.ProjectID,
			"employee_id", newAssignment.EmployeeID,
			"error", err,
		)
		return fmt.Errorf("failed to create new assignment: %w", err)
	}

	logger.InfoContext(ctx, "Assignment replaced successfully",
		"old_assignment_id", oldAssignment.ID,
		"new_assignment_id", newAssignment.ID,
	)

	return nil
}

// ValidateTransition validates if a transition from one assignment to another is allowed
func (m *DefaultAssignmentLifecycleManager) ValidateTransition(
	ctx context.Context,
	from, to *domain.ProjectEmployee) error {

	// Same project and employee
	if from.ProjectID != to.ProjectID {
		return domain.NewValidationError("cannot replace assignment for different project")
	}

	if from.EmployeeID != to.EmployeeID {
		return domain.NewValidationError("cannot replace assignment for different employee")
	}

	// Validate new assignment is valid
	if err := to.IsValid(); err != nil {
		return fmt.Errorf("new assignment is not valid: %w", err)
	}

	return nil
}
