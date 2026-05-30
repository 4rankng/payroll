package employee

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
	"gorm.io/gorm"
)

// EmployeeSyncServiceInterface defines the interface for employee sync service
type EmployeeSyncServiceInterface interface {
	SyncEmployeeNameToActiveAssignments(ctx context.Context, employeeID uint, newFullName string) (*SyncResult, error)
	SyncAllInconsistentData(ctx context.Context) (*BulkSyncResult, error)
	ValidateEmployeeNameConsistency(ctx context.Context, employeeID uint) (*ConsistencyReport, error)
}

// EmployeeSyncService handles synchronizing employee data between employees and project_employees tables
type EmployeeSyncService struct {
	projectEmployeeRepo domain.ProjectEmployeeRepository
	employeeRepo        domain.EmployeeRepository
	transactionManager  *infrastructure.TransactionManager
	eventBus            domain.EventBus
}

// NewEmployeeSyncService creates a new employee sync service
func NewEmployeeSyncService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	transactionManager *infrastructure.TransactionManager,
	eventBus domain.EventBus,
) *EmployeeSyncService {
	return &EmployeeSyncService{
		projectEmployeeRepo: projectEmployeeRepo,
		employeeRepo:        employeeRepo,
		transactionManager:  transactionManager,
		eventBus:            eventBus,
	}
}

// SyncEmployeeNameToActiveAssignments synchronizes employee name to all active and upcoming project assignments
func (s *EmployeeSyncService) SyncEmployeeNameToActiveAssignments(ctx context.Context, employeeID uint, newFullName string) (*SyncResult, error) {
	// Get the employee to verify they exist and get the old name for audit
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}

	oldFullName := employee.Fullname
	if oldFullName == newFullName {
		// No change needed
		return &SyncResult{
			EmployeeID:   employeeID,
			OldFullName:  oldFullName,
			NewFullName:  newFullName,
			UpdatedCount: 0,
			SyncedAt:     clock.Now(),
		}, nil
	}

	var updatedCount int64
	var syncError error

	// Execute sync within a transaction
	err = s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Get all assignments for this employee
		assignments, err := s.projectEmployeeRepo.GetByEmployee(ctx, employeeID)
		if err != nil {
			return fmt.Errorf("failed to get employee assignments: %w", err)
		}

		// Filter for active and upcoming assignments
		now := clock.Now()
		for _, assignment := range assignments {
			// Skip assignments that ended in the past
			if assignment.LastDate != nil && assignment.LastDate.Before(now) {
				continue
			}

			// Update each assignment with the new employee name
			assignment.EmployeeName = newFullName
			if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
				return fmt.Errorf("failed to update assignment %d: %w", assignment.ID, err)
			}
			updatedCount++
		}

		return nil
	})

	if err != nil {
		syncError = fmt.Errorf("failed to sync employee name: %w", err)
		updatedCount = 0
	}

	// Create sync result
	result := &SyncResult{
		EmployeeID:   employeeID,
		OldFullName:  oldFullName,
		NewFullName:  newFullName,
		UpdatedCount: int(updatedCount),
		SyncedAt:     clock.Now(),
		Error:        syncError,
	}

	// Publish sync completion event (even on failure, for audit purposes)
	if s.eventBus != nil {
		actorFullName := auditctx.GetFullName(ctx)
		if actorFullName == "" {
			actorFullName = "System"
		}

		event := domain.NewProjectEmployeesSyncedEvent(
			ctx,
			employeeID,
			oldFullName,
			newFullName,
			int(updatedCount),
			auditctx.GetUserIDOrZero(ctx),
			actorFullName,
		)

		if err := s.eventBus.Publish(ctx, event); err != nil {
			// Log but don't fail the operation if event publishing fails
			slog.Default().Warn("failed to publish ProjectEmployeesSyncedEvent", "error", err)
		}
	}

	return result, syncError
}

// SyncAllInconsistentData performs a one-time sync of all inconsistent employee names
// This should be called as a migration script or administrative function
func (s *EmployeeSyncService) SyncAllInconsistentData(ctx context.Context) (*BulkSyncResult, error) {
	// Get all employees with their current assignments
	employees, err := s.employeeRepo.List(ctx, domain.EmployeeFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}

	result := &BulkSyncResult{
		TotalEmployees:  len(employees),
		SyncedEmployees: 0,
		TotalUpdates:    0,
		Errors:          []string{},
		SyncedAt:        clock.Now(),
	}

	for _, employee := range employees {
		syncResult, err := s.SyncEmployeeNameToActiveAssignments(ctx, employee.ID, employee.Fullname)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Employee %d (%s): %v", employee.ID, employee.Fullname, err))
			continue
		}

		if syncResult.UpdatedCount > 0 {
			result.SyncedEmployees++
			result.TotalUpdates += syncResult.UpdatedCount
		}
	}

	return result, nil
}

// ValidateEmployeeNameConsistency checks if employee names are consistent between tables
func (s *EmployeeSyncService) ValidateEmployeeNameConsistency(ctx context.Context, employeeID uint) (*ConsistencyReport, error) {
	// Get employee from employees table
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}

	// Get all assignments for this employee
	assignments, err := s.projectEmployeeRepo.GetByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee assignments: %w", err)
	}

	report := &ConsistencyReport{
		EmployeeID:              employeeID,
		CanonicalName:           employee.Fullname,
		TotalAssignments:        len(assignments),
		InconsistentCount:       0,
		InconsistentAssignments: []InconsistentAssignment{},
		CheckedAt:               clock.Now(),
	}

	for _, assignment := range assignments {
		if assignment.EmployeeName != employee.Fullname {
			report.InconsistentCount++
			report.InconsistentAssignments = append(report.InconsistentAssignments, InconsistentAssignment{
				AssignmentID: assignment.ID,
				ProjectID:    assignment.ProjectID,
				CurrentName:  assignment.EmployeeName,
				CorrectName:  employee.Fullname,
				LastDate:     assignment.LastDate,
			})
		}
	}

	return report, nil
}

// SyncResult represents the result of a single employee name sync operation
type SyncResult struct {
	EmployeeID   uint
	OldFullName  string
	NewFullName  string
	UpdatedCount int
	SyncedAt     time.Time
	Error        error
}

// IsSuccess returns true if the sync was successful
func (r *SyncResult) IsSuccess() bool {
	return r.Error == nil
}

// BulkSyncResult represents the result of bulk sync operations
type BulkSyncResult struct {
	TotalEmployees  int
	SyncedEmployees int
	TotalUpdates    int
	Errors          []string
	SyncedAt        time.Time
}

// IsSuccess returns true if all operations were successful
func (r *BulkSyncResult) IsSuccess() bool {
	return len(r.Errors) == 0
}

// ConsistencyReport represents a consistency check report for an employee
type ConsistencyReport struct {
	EmployeeID              uint
	CanonicalName           string
	TotalAssignments        int
	InconsistentCount       int
	InconsistentAssignments []InconsistentAssignment
	CheckedAt               time.Time
}

// IsConsistent returns true if all assignments have consistent names
func (r *ConsistencyReport) IsConsistent() bool {
	return r.InconsistentCount == 0
}

// InconsistentAssignment represents an assignment with inconsistent employee name
type InconsistentAssignment struct {
	AssignmentID uint
	ProjectID    uint
	CurrentName  string
	CorrectName  string
	LastDate     *time.Time
}
