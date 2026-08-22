package project

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	auditctx "api-server/internal/pkg/context"

	"gorm.io/gorm"
)

// CacheDeleter is a minimal interface for cache invalidation
type CacheDeleter interface {
	Delete(ctx context.Context, key string) error
}

// TimesheetRecalculator defines the interface for recalculating timesheets
type TimesheetRecalculator interface {
	RecalculateTimesheetsForAssignment(ctx context.Context, assignment *domain.ProjectEmployee, updatedBy uint) error
}

// ProjectEmployeeService demonstrates orchestration-only pattern
type ProjectEmployeeService struct {
	// Repositories for data access
	projectEmployeeRepo      domain.ProjectEmployeeRepository
	employeeRepo             domain.EmployeeRepository
	employeeUserRepo         domain.EmployeeUserRepository
	projectRepo              domain.ProjectRepository
	timesheetAssignmentCheck domain.TimesheetAssignmentChecker
	timesheetReader          domain.TimesheetReader
	auditLogRepo             domain.AuditLogRepository

	// Domain services for business logic
	assignmentService *domainServices.EmployeeAssignmentService

	// Application services
	timesheetRecalculator TimesheetRecalculator

	// Infrastructure services
	transactionManager     *infrastructure.TransactionManager
	advancePaymentRepo     domain.AdvancePaymentRepository
	payrateRepo            domain.PayrateRepository
	eventBus               domain.EventBus
	notification           infraports.NotificationPort
	payCycleEventPublisher domain.PayCycleEventPublisher
	cache                  CacheDeleter
}

// NewProjectEmployeeService creates a new refactored assignment service
func NewProjectEmployeeService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	employeeUserRepo domain.EmployeeUserRepository,
	projectRepo domain.ProjectRepository,
	timesheetRepo domain.TimesheetRepository,
	auditLogRepo domain.AuditLogRepository,
	transactionManager *infrastructure.TransactionManager,
	advancePaymentRepo domain.AdvancePaymentRepository,
	payrateRepo domain.PayrateRepository,
	notification infraports.NotificationPort,
	eventBus domain.EventBus,
	payCycleEventPublisher domain.PayCycleEventPublisher,
	timesheetRecalculator TimesheetRecalculator,
	cache CacheDeleter,
) *ProjectEmployeeService {
	// Initialize domain services
	assignmentService := domainServices.NewEmployeeAssignmentService(projectEmployeeRepo, employeeRepo, projectRepo, timesheetRepo)

	return &ProjectEmployeeService{
		projectEmployeeRepo:      projectEmployeeRepo,
		employeeRepo:             employeeRepo,
		employeeUserRepo:         employeeUserRepo,
		projectRepo:              projectRepo,
		timesheetAssignmentCheck: timesheetRepo,
		timesheetReader:          timesheetRepo,
		auditLogRepo:             auditLogRepo,
		assignmentService:        assignmentService,
		timesheetRecalculator:    timesheetRecalculator,
		transactionManager:       transactionManager,
		advancePaymentRepo:       advancePaymentRepo,
		payrateRepo:              payrateRepo,
		eventBus:                 eventBus,
		notification:             notification,
		payCycleEventPublisher:   payCycleEventPublisher,
	}
}

// AssignEmployee orchestrates employee assignment (no business logic here)
func (s *ProjectEmployeeService) AssignEmployee(ctx context.Context, assignment *domain.ProjectEmployee, assignedBy uint) (*domain.ProjectEmployee, error) {
	var result *domain.ProjectEmployee

	// Preload employee and project for event publishing
	employee, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
	if err != nil {
		return nil, err
	}
	project, err := s.projectRepo.GetByID(ctx, assignment.ProjectID)
	if err != nil {
		return nil, err
	}

	// Orchestrate the operation within a transaction
	err = s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Validate assignment using domain service
		if err := s.assignmentService.ValidateAssignment(ctx, assignment); err != nil {
			return err
		}

		// 2. Set audit fields
		assignment.CreatedBy = assignedBy

		// 3. Create assignment using repository
		if err := s.projectEmployeeRepo.Create(ctx, assignment); err != nil {
			return err
		}

		// 4. Populate employee data explicitly (replaces GORM hooks)
		assignment.EmployeeName = employee.Fullname
		assignment.EmployeeCCCD = employee.CCCD
		if assignment.EmployeeCode == "" {
			assignment.EmployeeCode = employee.CCCD
		}

		// 5. Update the assignment with populated employee data
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 6. Publish ProjectEmployeeCreatedEvent for audit and notifications
		if s.eventBus != nil {
			// Need to get a user repository for actor lookup - for now, use a simple context approach
			actorFullName := auditctx.GetFullName(ctx)
			if actorFullName == "" {
				actorFullName = "Unknown" // fallback when no context available
			}
			event := domain.NewProjectEmployeeCreatedEvent(ctx, assignment, project.Name, employee.Fullname, auditctx.GetUserIDOrZero(ctx), actorFullName)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish ProjectEmployeeCreatedEvent", "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	result = assignment
	return result, nil
}

// UpdateAssignment orchestrates assignment updates
func (s *ProjectEmployeeService) UpdateAssignment(ctx context.Context, assignment *domain.ProjectEmployee, updatedBy uint) error {
	var existing *domain.ProjectEmployee

	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get existing assignment for audit
		var err error
		existing, err = s.projectEmployeeRepo.GetByID(ctx, assignment.ID)
		if err != nil {
			return err
		}

		// 2. Validate updated assignment using domain service (use update-specific validation)
		if err := s.assignmentService.ValidateAssignmentForUpdate(ctx, assignment); err != nil {
			return err
		}

		// 3. Update assignment using repository
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 4. Publish ProjectEmployeeUpdatedEvent
		if s.eventBus != nil {
			// Ensure related entities are available for the event
			employee, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if err == nil {
				assignment.Employee = *employee
			}
			project, err := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if err == nil {
				assignment.Project = *project
			}

			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (update)", "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 5. Invalidate assignment cache so subsequent validation sees updated data
	if s.cache != nil {
		cacheKey := fmt.Sprintf("assignment:%d:%d", assignment.ProjectID, assignment.EmployeeID)
		if err := s.cache.Delete(ctx, cacheKey); err != nil {
			observability.GetLogger().Warn("failed to invalidate assignment cache after update", "error", err)
		}
	}

	// 6. After successful transaction commit, recalculate editable timesheets if position or start date changed
	if s.timesheetRecalculator != nil && (existing.Position != assignment.Position || !existing.StartDate.Equal(assignment.StartDate)) {
		recalculator := s.timesheetRecalculator
		assignmentCopy := *assignment
		domain.RegisterAfterCommit(ctx, func() {
			if err := recalculator.RecalculateTimesheetsForAssignment(context.Background(), &assignmentCopy, updatedBy); err != nil {
				observability.GetLogger().Warn("failed to recalculate timesheets for assignment", "assignmentID", assignmentCopy.ID, "error", err)
			}
		})
	}

	return nil
}

// UpdateAssignmentPosition updates only the position field of an existing assignment,
// with full cache invalidation, event emission, and conditional timesheet recalculation.
// Uses targeted column update to avoid full-row Save() overwriting concurrent changes.
func (s *ProjectEmployeeService) UpdateAssignmentPosition(ctx context.Context, assignmentID uint, position string, updatedBy uint) error {
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.projectEmployeeRepo.GetByID(txCtx, assignmentID)
		if err != nil {
			return err
		}
		return s.updateAssignmentPosition(txCtx, existing, existing.Position, position, updatedBy)
	})
}

// UpdateAssignmentPositionIfCurrent updates a position only when it still
// matches the caller's snapshot. When ctx already carries a transaction, the
// compare-and-swap and all downstream writes join that transaction.
func (s *ProjectEmployeeService) UpdateAssignmentPositionIfCurrent(
	ctx context.Context,
	assignmentID uint,
	currentPosition string,
	newPosition string,
	updatedBy uint,
) error {
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.projectEmployeeRepo.GetByID(txCtx, assignmentID)
		if err != nil {
			return err
		}
		if existing.Position != currentPosition {
			return domain.NewConflictError("vị trí nhân viên đã thay đổi trong lúc nhập BCC")
		}
		return s.updateAssignmentPosition(txCtx, existing, currentPosition, newPosition, updatedBy)
	})
}

func (s *ProjectEmployeeService) updateAssignmentPosition(
	ctx context.Context,
	existing *domain.ProjectEmployee,
	currentPosition string,
	newPosition string,
	updatedBy uint,
) error {
	if currentPosition == newPosition {
		return nil
	}
	if err := s.projectEmployeeRepo.UpdatePositionIfCurrent(ctx, existing.ID, currentPosition, newPosition); err != nil {
		return err
	}

	existing.Position = newPosition
	assignmentCopy := *existing
	afterCommitCtx := domain.WithoutTransactionContext(context.WithoutCancel(ctx))
	domain.RegisterAfterCommit(ctx, func() {
		if s.cache != nil {
			cacheKey := fmt.Sprintf("assignment:%d:%d", assignmentCopy.ProjectID, assignmentCopy.EmployeeID)
			if err := s.cache.Delete(afterCommitCtx, cacheKey); err != nil {
				observability.GetLogger().Warn("failed to invalidate assignment cache after position update", "error", err)
			}
		}

		if s.eventBus != nil {
			if employee, err := s.employeeRepo.GetByID(afterCommitCtx, assignmentCopy.EmployeeID); err == nil {
				assignmentCopy.Employee = *employee
			}
			if project, err := s.projectRepo.GetByID(afterCommitCtx, assignmentCopy.ProjectID); err == nil {
				assignmentCopy.Project = *project
			}
			event := domain.NewProjectEmployeeUpdatedEvent(afterCommitCtx, &assignmentCopy)
			if err := s.eventBus.Publish(afterCommitCtx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (position update)", "error", err)
			}
		}

		if s.timesheetRecalculator != nil {
			if err := s.timesheetRecalculator.RecalculateTimesheetsForAssignment(afterCommitCtx, &assignmentCopy, updatedBy); err != nil {
				observability.GetLogger().Warn("failed to recalculate timesheets after position update", "assignmentID", assignmentCopy.ID, "error", err)
			}
		}
	})

	return nil
}

// EndAssignment orchestrates assignment termination
func (s *ProjectEmployeeService) EndAssignment(ctx context.Context, assignmentID uint, endDate time.Time, endedBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get existing assignment
		assignment, err := s.projectEmployeeRepo.GetByID(ctx, assignmentID)
		if err != nil {
			return err
		}

		// 2. Check if there are timesheets after the end date (business rule)
		hasTimesheets, err := s.timesheetAssignmentCheck.HasTimesheetsAfterDate(ctx, assignment.ProjectID, assignment.EmployeeID, endDate)
		if err != nil {
			return err
		}
		if hasTimesheets {
			return domain.NewValidationError(constants.MsgCannotEndAssignmentBeforeTimesheetDatesVN)
		}

		// 3. End assignment using repository
		if err := s.projectEmployeeRepo.EndAssignment(ctx, assignmentID, endDate); err != nil {
			return err
		}

		// 4. Publish ProjectEmployeeUpdatedEvent to reflect end of assignment
		if s.eventBus != nil {
			updated, err := s.projectEmployeeRepo.GetByID(ctx, assignmentID)
			if err == nil {
				// Load related entities for richer event payload
				employee, errEmp := s.employeeRepo.GetByID(ctx, updated.EmployeeID)
				if errEmp == nil {
					updated.Employee = *employee
				}
				project, errProj := s.projectRepo.GetByID(ctx, updated.ProjectID)
				if errProj == nil {
					updated.Project = *project
				}

				event := domain.NewProjectEmployeeUpdatedEvent(ctx, updated)
				if err := s.eventBus.Publish(ctx, event); err != nil {
					observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (end)", "error", err)
				}
			}
		}

		return nil
	})
}

// GetConflictingAssignments orchestrates conflict detection
func (s *ProjectEmployeeService) GetConflictingAssignments(ctx context.Context, employeeID uint, startDate time.Time, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to domain service - no transaction needed for read operation
	return s.assignmentService.GetConflictingAssignments(ctx, employeeID, startDate, endDate)
}

// GetAssignmentsByProject orchestrates data retrieval
func (s *ProjectEmployeeService) GetAssignmentsByProject(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByProject(ctx, projectID)
}

// GetAssignmentsByEmployee orchestrates data retrieval
func (s *ProjectEmployeeService) GetAssignmentsByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByEmployee(ctx, employeeID)
}

// ListAssignments orchestrates assignment listing with pagination
func (s *ProjectEmployeeService) ListAssignments(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.List(ctx, filters)
}

// CountAssignments orchestrates assignment counting
func (s *ProjectEmployeeService) CountAssignments(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.Count(ctx, filters)
}

// GetAssignmentByProjectAndEmployee orchestrates specific assignment retrieval
func (s *ProjectEmployeeService) GetAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetByProjectAndEmployee(ctx, projectID, employeeID)
}

// AssignEmployeesBatch orchestrates batch employee assignment
func (s *ProjectEmployeeService) AssignEmployeesBatch(ctx context.Context, assignments []*domain.ProjectEmployee, assignedBy uint) ([]*domain.ProjectEmployee, error) {
	var results []*domain.ProjectEmployee

	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, assignment := range assignments {
			// Delegate to single assignment method
			result, err := s.AssignEmployee(ctx, assignment, assignedBy)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})

	return results, err
}

// RemoveEmployeesBatch orchestrates batch employee removal
func (s *ProjectEmployeeService) RemoveEmployeesBatch(ctx context.Context, projectID uint, requests []RemoveEmployeeRequest, removedBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, request := range requests {
			// Find the active assignment by project and employee
			assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, request.EmployeeID)
			if err != nil {
				// Return the error directly - it should already be a proper domain error with Vietnamese message
				return err
			}

			// Validate the retrieved assignment has proper data
			if assignment.EmployeeID == 0 {
				return domain.NewValidationError(constants.MsgInvalidAssignmentDataVN)
			}

			// Determine end date
			var endDate time.Time
			if request.LastDate != nil {
				endDate = *request.LastDate
			} else {
				endDate = clock.Now()
			}

			// Always soft-delete by setting last_date to preserve historical data
			// (advance payments, audit trail) even when no timesheets exist.
			if err := s.projectEmployeeRepo.EndAssignment(ctx, assignment.ID, endDate); err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveEmployeeRequest represents a request to remove an employee from a project
type RemoveEmployeeRequest struct {
	EmployeeID uint       `json:"employee_id"`
	LastDate   *time.Time `json:"last_date"`
}

// CleanupDuplicateAssignments removes duplicate assignment records for the same employee-project combination
// Keeps only the active assignment (last_date = NULL) or the most recent one if all are ended
func (s *ProjectEmployeeService) CleanupDuplicateAssignments(ctx context.Context) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Find all duplicate assignments (same project_id + employee_id combination)
		var duplicates []struct {
			ProjectID  uint `json:"project_id"`
			EmployeeID uint `json:"employee_id"`
			Count      int  `json:"count"`
		}

		err := tx.WithContext(ctx).
			Model(&domain.ProjectEmployee{}).
			Select("project_id, employee_id, COUNT(*) as count").
			Group("project_id, employee_id").
			Having("COUNT(*) > 1").
			Scan(&duplicates).Error

		if err != nil {
			return err
		}

		for _, dup := range duplicates {
			// Get all assignments for this project-employee combination
			assignments, err := s.projectEmployeeRepo.GetByEmployee(ctx, dup.EmployeeID)
			if err != nil {
				continue // Skip this one and continue with others
			}

			// Filter to only this project
			var projectAssignments []*domain.ProjectEmployee
			for _, assignment := range assignments {
				if assignment.ProjectID == dup.ProjectID {
					projectAssignments = append(projectAssignments, assignment)
				}
			}

			if len(projectAssignments) <= 1 {
				continue // No duplicates for this combination
			}

			// Find the assignment to keep (active one or most recent)
			var keepAssignment *domain.ProjectEmployee
			for _, assignment := range projectAssignments {
				if assignment.LastDate == nil {
					keepAssignment = assignment
					break
				}
			}

			// If no active assignment, keep the most recent one
			if keepAssignment == nil {
				for _, assignment := range projectAssignments {
					if keepAssignment == nil || (assignment.UpdatedAt.After(keepAssignment.UpdatedAt)) {
						keepAssignment = assignment
					}
				}
			}

			// Delete all others
			for _, assignment := range projectAssignments {
				if assignment.ID != keepAssignment.ID {
					// Check if this assignment has timesheets
					hasTimesheets, err := s.timesheetAssignmentCheck.HasTimesheetsForAssignment(ctx, assignment.ProjectID, assignment.EmployeeID)
					if err != nil {
						continue // Skip on error
					}

					if !hasTimesheets {
						// Safe to delete
						if err := s.projectEmployeeRepo.Delete(ctx, assignment.ID); err != nil {
							continue // Skip on error, continue with others
						}
					}
				}
			}
		}

		return nil
	})
}

// RequestPaymentScheduleChange orchestrates payment schedule change request
func (s *ProjectEmployeeService) RequestPaymentScheduleChange(ctx context.Context, assignmentID uint, newSchedule domain.PaymentSchedule, effectiveDate time.Time, requestedBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get existing assignment
		assignment, err := s.projectEmployeeRepo.GetByID(ctx, assignmentID)
		if err != nil {
			return err
		}

		// Store old schedule for event
		oldSchedule := assignment.PaymentSchedule

		// 2. Get old state for audit
		if err != nil {
			return fmt.Errorf("failed to marshal existing assignment for audit: %w", err)
		}

		hasPaidThisMonth, err := s.hasPaidTimesheetsThisMonth(ctx, assignment)
		if err != nil {
			return err
		}

		applyImmediately := false
		switch newSchedule {
		case domain.PaymentScheduleMonthly:
			// Weekly -> Monthly: only check this employee
			applyImmediately = !hasPaidThisMonth
		case domain.PaymentScheduleWeekly:
			// Monthly -> Weekly: ensure no one in the project has been paid this month
			projectHasPaid, err := s.projectHasPaidTimesheetsThisMonth(ctx, assignment.ProjectID)
			if err != nil {
				return err
			}
			applyImmediately = !projectHasPaid
		default:
			// PaymentScheduleFlexible (and any unrecognized value) lands here:
			// apply immediately unless THIS employee has been paid this month.
			// NOTE: the Weekly branch above additionally defers when ANY project
			// member has been paid this month (projectHasPaidTimesheetsThisMonth),
			// because Monthly→Weekly is a finer-grained transition that can
			// corrupt a mid-month cycle. Monthly→Flexible is the same kind of
			// finer-grained transition — confirm whether Flexible should defer
			// project-wide too (i.e. share the Weekly branch) before relying on
			// this per-employee default. Tracked as a business-rule decision.
			applyImmediately = !hasPaidThisMonth
		}

		if applyImmediately {
			if err := assignment.ApplyScheduleChangeImmediately(newSchedule); err != nil {
				return err
			}
		} else {
			nextMonthEffectiveDate := firstDayOfNextMonth(clock.Now())
			effectiveDate = nextMonthEffectiveDate
			if err := assignment.RequestScheduleChange(newSchedule, effectiveDate); err != nil {
				return err
			}
		}

		// 4. Update assignment in repository
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 5. Publish ProjectEmployeeUpdatedEvent
		if s.eventBus != nil {
			emp, errEmp := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if errEmp == nil && emp != nil {
				assignment.Employee = *emp
			}
			proj, errProj := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if errProj == nil && proj != nil {
				assignment.Project = *proj
			}

			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (schedule change)", "error", err)
			}
		}

		// 6. Publish payment cycle changed event
		if s.payCycleEventPublisher != nil {
			// Get employee and project details for event context
			employee, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if err != nil {
				return fmt.Errorf("failed to get employee for event: %w", err)
			}

			project, err := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to get project for event: %w", err)
			}

			effectiveFromDate := clock.Now()
			if !applyImmediately {
				effectiveFromDate = effectiveDate
			}

			event := domain.NewPayCycleChangedEvent(
				assignment.ProjectID,
				assignment.EmployeeID,
				string(oldSchedule),
				string(newSchedule),
				effectiveFromDate,
				applyImmediately,
				&requestedBy,
				project.Name,
				employee.Fullname,
			)

			if err := s.payCycleEventPublisher.PublishPayCycleChanged(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				// The event is for notifications and audit, not critical business logic
				observability.GetLogger().Warn("failed to publish payment cycle changed event", "error", err)
			}
		}

		return nil
	})
}

// CancelPaymentScheduleChange orchestrates canceling a pending payment schedule change
func (s *ProjectEmployeeService) CancelPaymentScheduleChange(ctx context.Context, assignmentID uint, canceledBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get existing assignment
		assignment, err := s.projectEmployeeRepo.GetByID(ctx, assignmentID)
		if err != nil {
			return err
		}

		// 2. Get old state for audit
		if err != nil {
			return fmt.Errorf("failed to marshal existing assignment for audit: %w", err)
		}

		// 3. Cancel schedule change using domain method
		if err := assignment.CancelPendingScheduleChange(); err != nil {
			return err
		}

		// 4. Update assignment in repository
		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		// 5. Publish ProjectEmployeeUpdatedEvent to reflect cancellation
		if s.eventBus != nil {
			emp, errEmp := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if errEmp == nil && emp != nil {
				assignment.Employee = *emp
			}
			proj, errProj := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if errProj == nil && proj != nil {
				assignment.Project = *proj
			}

			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (cancel schedule change)", "error", err)
			}
		}

		return nil
	})
}

// CancelPendingCheckInEnable cancels a pending (not yet activated) check-in enable.
func (s *ProjectEmployeeService) CancelPendingCheckInEnable(ctx context.Context, assignmentID uint, canceledBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		assignment, err := s.projectEmployeeRepo.GetByID(ctx, assignmentID)
		if err != nil {
			return err
		}

		if err := assignment.CancelPendingCheckInEnable(); err != nil {
			return err
		}

		if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
			return err
		}

		if s.eventBus != nil {
			emp, errEmp := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
			if errEmp == nil && emp != nil {
				assignment.Employee = *emp
			}
			proj, errProj := s.projectRepo.GetByID(ctx, assignment.ProjectID)
			if errProj == nil && proj != nil {
				assignment.Project = *proj
			}

			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.eventBus.Publish(ctx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (cancel pending check-in)", "error", err)
			}
		}

		return nil
	})
}

// ApplyPendingCheckInEnables activates all pending check-in enables whose
// effective date (day 1 of the month) has arrived. Self-healing: the query is
// date-driven (`effective <= today`), so a missed run applies on the next one.
func (s *ProjectEmployeeService) ApplyPendingCheckInEnables(ctx context.Context) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		now := clock.Now()
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		assignments, err := s.projectEmployeeRepo.GetEmployeesWithPendingCheckInEnable(ctx, startOfToday)
		if err != nil {
			return err
		}

		for _, assignment := range assignments {
			if !assignment.ApplyPendingCheckIn() {
				continue
			}

			if err := s.projectEmployeeRepo.Update(ctx, assignment); err != nil {
				observability.GetLogger().Warn("failed to apply pending check-in enable", "assignmentID", assignment.ID, "error", err)
				continue
			}

			if s.eventBus != nil {
				emp, errEmp := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
				if errEmp == nil && emp != nil {
					assignment.Employee = *emp
				}
				proj, errProj := s.projectRepo.GetByID(ctx, assignment.ProjectID)
				if errProj == nil && proj != nil {
					assignment.Project = *proj
				}

				event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
				if err := s.eventBus.Publish(ctx, event); err != nil {
					observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (apply pending check-in)", "error", err)
				}
			}
		}

		return nil
	})
}

// This method can be called by a background job or manually

func (s *ProjectEmployeeService) ApplyPendingScheduleChanges(ctx context.Context) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Get all employees with pending schedule changes that should be applied today or earlier
		now := clock.Now()
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		employees, err := s.projectEmployeeRepo.GetEmployeesWithPendingScheduleChanges(ctx, startOfToday)
		if err != nil {
			return err
		}

		// Apply each pending change
		for _, employee := range employees {
			// Get old state for audit
			if err != nil {
				continue // Skip this one, continue with others
			}

			// Store old schedule and pending schedule for event
			oldSchedule := employee.PaymentSchedule
			pendingSchedule := employee.PendingPaymentSchedule

			// Apply pending schedule change using domain method
			if employee.ApplyPendingSchedule() {
				// Update in repository
				if err := s.projectEmployeeRepo.Update(ctx, employee); err != nil {
					continue // Skip this one, continue with others
				}

				// Publish ProjectEmployeeUpdatedEvent for applied schedule change
				if s.eventBus != nil {
					emp, errEmp := s.employeeRepo.GetByID(ctx, employee.EmployeeID)
					if errEmp == nil && emp != nil {
						employee.Employee = *emp
					}
					proj, errProj := s.projectRepo.GetByID(ctx, employee.ProjectID)
					if errProj == nil && proj != nil {
						employee.Project = *proj
					}

					event := domain.NewProjectEmployeeUpdatedEvent(ctx, employee)
					if err := s.eventBus.Publish(ctx, event); err != nil {
						observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (apply pending schedule)", "error", err)
					}
				}

				// Publish payment cycle changed event
				if s.payCycleEventPublisher != nil && pendingSchedule != nil {
					// Get employee and project details for event context
					emp, err := s.employeeRepo.GetByID(ctx, employee.EmployeeID)
					if err != nil {
						continue
					}

					project, err := s.projectRepo.GetByID(ctx, employee.ProjectID)
					if err != nil {
						continue
					}

					event := domain.NewPayCycleChangedEvent(
						employee.ProjectID,
						employee.EmployeeID,
						string(oldSchedule),
						string(*pendingSchedule),
						now,
						false, // This is a deferred change being applied
						nil,   // System initiated
						project.Name,
						emp.Fullname,
					)

					_ = s.payCycleEventPublisher.PublishPayCycleChanged(ctx, event)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// ToggleCheckInEnabled toggles the check-in enabled status for an employee in a project.
// Enabling is DEFERRED: it activates on day 1 of the next month (strict — enabling
// on the 1st still defers to the following month). Disabling stays instant and
// zeroes out quota for the current month onward; disabling a pending (not yet
// active) enable just cancels the pending request without touching quota.
func (s *ProjectEmployeeService) ToggleCheckInEnabled(ctx context.Context, projectID, employeeID uint, enabled bool, updatedBy uint) error {
	// Require an active payrate before enabling check-in — without it attendance
	// records cannot be priced and timesheets would fail to process.
	if enabled {
		rates, err := s.payrateRepo.GetActiveByProject(ctx, projectID, clock.Now())
		if err != nil {
			return fmt.Errorf("failed to check payrate configuration: %w", err)
		}
		if len(rates) == 0 {
			return domain.NewValidationError("Dự án chưa có bảng lương. Vui lòng cấu hình bảng lương trước khi bật điểm danh.")
		}
	}

	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(txCtx, projectID, employeeID)
		if err != nil {
			return err
		}

		if enabled {
			if assignment.CheckInEnabled || assignment.HasPendingCheckInEnable() {
				return nil // Already active or already pending — no change needed
			}
			if err := assignment.RequestCheckInEnable(firstDayOfNextMonth(clock.Now())); err != nil {
				return err
			}
		} else {
			if assignment.HasPendingCheckInEnable() {
				// Cancel the pending enable: check-in was never active, so there is
				// no quota to zero out.
				assignment.PendingCheckInEnabled = nil
				assignment.CheckInEffectiveFrom = nil
			} else if assignment.CheckInEnabled {
				assignment.CheckInEnabled = false
				// Zero out quota for current month onward (as per spec)
				currentMonth := clock.Now().Format("2006-01")
				if err := s.advancePaymentRepo.ZeroOutQuota(txCtx, projectID, employeeID, currentMonth); err != nil {
					return err
				}
			} else {
				return nil // Already disabled with nothing pending — no change needed
			}
		}

		if err := s.projectEmployeeRepo.Update(txCtx, assignment); err != nil {
			return err
		}

		// Publish domain event
		if s.eventBus != nil {
			event := domain.NewProjectEmployeeUpdatedEvent(txCtx, assignment)
			if err := s.eventBus.Publish(txCtx, event); err != nil {
				observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (toggle check-in)", "error", err)
			}
		}

		return nil
	})
}

// BulkToggleCheckInEnabled toggles the check-in enabled status for multiple employees in a project.
// The toggle logic is inlined within a single transaction to avoid nested transactions and ensure atomicity.
// Enabling is deferred to day 1 of the next month (same rule as the single toggle);
// disabling cancels any pending enable first (no quota zeroing for never-active rows).
func (s *ProjectEmployeeService) BulkToggleCheckInEnabled(ctx context.Context, projectID uint, employeeIDs []uint, enabled bool, updatedBy uint) error {
	// Require an active payrate before enabling check-in — without it attendance
	// records cannot be priced and timesheets would fail to process.
	if enabled {
		rates, err := s.payrateRepo.GetActiveByProject(ctx, projectID, clock.Now())
		if err != nil {
			return fmt.Errorf("failed to check payrate configuration: %w", err)
		}
		if len(rates) == 0 {
			return domain.NewValidationError("Dự án chưa có bảng lương. Vui lòng cấu hình bảng lương trước khi bật điểm danh.")
		}
	}

	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		var disabledIDs []uint
		for _, empID := range employeeIDs {
			assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(txCtx, projectID, empID)
			if err != nil {
				return err
			}

			if enabled {
				if assignment.CheckInEnabled || assignment.HasPendingCheckInEnable() {
					continue
				}
				if err := assignment.RequestCheckInEnable(firstDayOfNextMonth(clock.Now())); err != nil {
					return err
				}
			} else {
				if assignment.HasPendingCheckInEnable() {
					// Cancel pending enable — never active, no quota to zero.
					assignment.PendingCheckInEnabled = nil
					assignment.CheckInEffectiveFrom = nil
				} else if assignment.CheckInEnabled {
					assignment.CheckInEnabled = false
					disabledIDs = append(disabledIDs, empID)
				} else {
					continue
				}
			}

			if err := s.projectEmployeeRepo.Update(txCtx, assignment); err != nil {
				return err
			}

			if s.eventBus != nil {
				event := domain.NewProjectEmployeeUpdatedEvent(txCtx, assignment)
				if err := s.eventBus.Publish(txCtx, event); err != nil {
					observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (bulk toggle)", "error", err)
				}
			}
		}

		// Batch zero-out quotas for all disabled employees in a single query.
		if !enabled && len(disabledIDs) > 0 {
			currentMonth := clock.Now().Format("2006-01")
			if err := s.advancePaymentRepo.BatchZeroOutQuota(txCtx, projectID, disabledIDs, currentMonth); err != nil {
				return err
			}
		}
		return nil
	})
}

func currentCheckInMonthWindow() (time.Time, time.Time, time.Time) {
	now := clock.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)
	asOfDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return monthStart, monthEnd, asOfDate
}

func (s *ProjectEmployeeService) GetCheckInConfiguration(
	ctx context.Context,
	projectID uint,
	status domain.CheckInConfigurationStatus,
	search string,
	page int,
	pageSize int,
) (*domain.CheckInConfigurationResult, time.Time, time.Time, error) {
	monthStart, monthEnd, asOfDate := currentCheckInMonthWindow()
	result, err := s.projectEmployeeRepo.GetCheckInConfiguration(ctx, domain.CheckInConfigurationQuery{
		ProjectID:  projectID,
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
		AsOfDate:   asOfDate,
		Status:     status,
		Search:     search,
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	})
	return result, monthStart, monthEnd, err
}

func (s *ProjectEmployeeService) DisableInactiveCheckInEmployees(
	ctx context.Context,
	projectID uint,
	updatedBy uint,
) (int, error) {
	disabledCount := 0
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		monthStart, monthEnd, asOfDate := currentCheckInMonthWindow()
		assignments, err := s.projectEmployeeRepo.GetCurrentAssignmentsForProjectForUpdate(txCtx, projectID, asOfDate)
		if err != nil {
			return err
		}
		result, err := s.projectEmployeeRepo.GetCheckInConfiguration(txCtx, domain.CheckInConfigurationQuery{
			ProjectID:  projectID,
			MonthStart: monthStart,
			MonthEnd:   monthEnd,
			AsOfDate:   asOfDate,
			Status:     domain.CheckInConfigurationStatusInactive,
		})
		if err != nil {
			return err
		}
		if len(result.Employees) == 0 {
			return nil
		}

		inactiveEmployeeIDs := make(map[uint]struct{}, len(result.Employees))
		employeeIDs := make([]uint, 0, len(result.Employees))
		for _, employee := range result.Employees {
			if _, exists := inactiveEmployeeIDs[employee.EmployeeID]; exists {
				continue
			}
			inactiveEmployeeIDs[employee.EmployeeID] = struct{}{}
			employeeIDs = append(employeeIDs, employee.EmployeeID)
		}

		for _, assignment := range assignments {
			if _, shouldDisable := inactiveEmployeeIDs[assignment.EmployeeID]; !shouldDisable {
				continue
			}
			assignment.CheckInEnabled = false
			assignment.PendingCheckInEnabled = nil
			assignment.CheckInEffectiveFrom = nil
			if err := s.projectEmployeeRepo.Update(txCtx, assignment); err != nil {
				return err
			}
			if s.eventBus != nil {
				event := domain.NewProjectEmployeeUpdatedEvent(txCtx, assignment)
				if err := s.eventBus.Publish(txCtx, event); err != nil {
					observability.GetLogger().Warn("failed to publish ProjectEmployeeUpdatedEvent (disable inactive)", "error", err)
				}
			}
		}
		if err := s.advancePaymentRepo.BatchZeroOutQuota(txCtx, projectID, employeeIDs, clock.Now().Format("2006-01")); err != nil {
			return err
		}
		disabledCount = len(employeeIDs)
		return nil
	})
	return disabledCount, err
}

// GetEmployeesByPaymentSchedule retrieves all employees with a specific payment schedule
func (s *ProjectEmployeeService) GetEmployeesByPaymentSchedule(ctx context.Context, schedule domain.PaymentSchedule) ([]*domain.ProjectEmployee, error) {
	// Simple delegation to repository - no business logic needed
	return s.projectEmployeeRepo.GetEmployeesByPaymentSchedule(ctx, schedule)
}

// GetEmployeesWithPendingScheduleChanges retrieves all employees with pending schedule changes
func (s *ProjectEmployeeService) GetEmployeesWithPendingScheduleChanges(ctx context.Context) ([]*domain.ProjectEmployee, error) {
	// Get all employees with pending schedule changes (no time limit)
	return s.projectEmployeeRepo.GetEmployeesWithPendingScheduleChanges(ctx, time.Time{})
}

func (s *ProjectEmployeeService) hasPaidTimesheetsThisMonth(ctx context.Context, assignment *domain.ProjectEmployee) (bool, error) {
	now := clock.Now()
	start := startOfMonth(now)
	end := firstDayOfNextMonth(now).AddDate(0, 0, -1)

	employeeID := assignment.EmployeeID
	filters := domain.TimesheetFilters{
		ProjectIDs:    []uint{assignment.ProjectID},
		EmployeeID:    &employeeID,
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:      &start,
		ToDate:        &end,
	}

	count, err := s.timesheetReader.Count(ctx, filters)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *ProjectEmployeeService) projectHasPaidTimesheetsThisMonth(ctx context.Context, projectID uint) (bool, error) {
	now := clock.Now()
	start := startOfMonth(now)
	end := firstDayOfNextMonth(now).AddDate(0, 0, -1)

	filters := domain.TimesheetFilters{
		ProjectIDs:    []uint{projectID},
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:      &start,
		ToDate:        &end,
	}

	count, err := s.timesheetReader.Count(ctx, filters)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func firstDayOfNextMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
}
