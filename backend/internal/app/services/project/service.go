package project

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

type ProjectService struct {
	ProjectRepo         domain.ProjectRepository
	EmployeeRepo        domain.EmployeeRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	TimesheetRepo       domain.TimesheetRepository
	events              domain.EventBus
	cache               domain.CacheServiceUseCase
	logger              *slog.Logger
}

func NewProjectService(projectRepo domain.ProjectRepository, employeeRepo domain.EmployeeRepository, projectEmployeeRepo domain.ProjectEmployeeRepository, timesheetRepo domain.TimesheetRepository, events domain.EventBus, cache domain.CacheServiceUseCase, logger *slog.Logger) *ProjectService {
	return &ProjectService{
		ProjectRepo:         projectRepo,
		EmployeeRepo:        employeeRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		TimesheetRepo:       timesheetRepo,
		events:              events,
		cache:               cache,
		logger:              logger,
	}
}

func (s *ProjectService) CreateProject(ctx context.Context, project *domain.Project, createdBy uint) (*domain.Project, error) {
	s.logger.Info("Creating project", "name", project.Name, "client", project.ClientName, "created_by", createdBy)

	// Auto-generate code if not provided
	if project.Code == "" {
		project.Code = s.generateProjectCode(project.ClientName)
		s.logger.Info("Auto-generated project code", "code", project.Code)
	}

	// Check if project code already exists
	if existingProject, err := s.ProjectRepo.GetByCode(ctx, project.Code); err == nil {
		s.logger.Error("Project with this code already exists", "code", project.Code, "existing_project_id", existingProject.ID)
		return nil, domain.NewConflictError(fmt.Sprintf("project with code '%s' already exists", project.Code))
	}

	// Validate project before creation
	if err := project.IsValid(); err != nil {
		s.logger.Error("Project validation failed", "error", err, "project", project.Name)
		return nil, err
	}

	if err := s.ProjectRepo.Create(ctx, project); err != nil {
		s.logger.Error("Failed to create project in database", "error", err, "project", project.Name, "created_by", createdBy)
		// Check if it's a database constraint violation (additional safety net)
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			return nil, domain.NewConflictError(fmt.Sprintf("project with code '%s' already exists", project.Code))
		}
		return nil, domain.NewInternalError(constants.MsgFailedToCreateProjectVN, err)
	}

	s.logger.Info("Project created successfully", "id", project.ID, "name", project.Name, "code", project.Code)

	// Publish domain event
	actorFullName := auditctx.GetFullName(ctx)
	if actorFullName == "" {
		actorFullName = "Unknown"
	}
	event := domain.NewProjectCreatedEvent(ctx, project, auditctx.GetUserIDOrZero(ctx), actorFullName)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish ProjectCreatedEvent", "project_id", project.ID, "error", err)
	}

	return project, nil
}

func (s *ProjectService) GetProject(ctx context.Context, id uint) (*domain.Project, error) {
	return s.ProjectRepo.GetByID(ctx, id)
}

func (s *ProjectService) GetProjectWithEmployeeCount(ctx context.Context, id uint) (*domain.ProjectWithEmployeeCount, error) {
	return s.ProjectRepo.GetByIDWithEmployeeCount(ctx, id)
}

func (s *ProjectService) UpdateProject(ctx context.Context, project *domain.Project, updatedBy uint) error {
	// Get the original project to check for status changes
	originalProject, err := s.ProjectRepo.GetByID(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to get original project: %w", err)
	}

	// Check if payment schedule is being changed
	// Currently no IsWeekly or IsMonthly fields exist, so we skip this check
	// When those fields are added back in the future, this check should be re-enabled

	// Log salary period changes with unsettled paid timesheets (allowed, settlement dedup prevents issues)
	salaryPeriodChanged := project.SalaryPeriodFrom != originalProject.SalaryPeriodFrom ||
		project.SalaryPeriodTo != originalProject.SalaryPeriodTo
	if salaryPeriodChanged {
		count, err := s.TimesheetRepo.CountUnsettledPaidTimesheets(ctx, project.ID)
		if err != nil {
			return fmt.Errorf("failed to check unsettled timesheets: %w", err)
		}
		if count > 0 {
			s.logger.Warn("Salary period change with unsettled paid timesheets",
				"project_id", project.ID,
				"unsettled_count", count)
		}
	}

	// Check if project is being closed (completed or cancelled)
	isBeingClosed := (project.IsCompleted() || project.IsCancelled()) &&
		(!originalProject.IsCompleted() && !originalProject.IsCancelled())

	if err := s.ProjectRepo.Update(ctx, project); err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Auto-terminate employees if project is being closed
	if isBeingClosed {
		if err := s.autoTerminateEmployees(ctx, project, updatedBy); err != nil {
			s.logger.Error("Failed to auto-terminate employees for closed project",
				"project_id", project.ID,
				"project_status", project.ProjectStatus,
				"error", err)
			// Don't fail the update, just log the error
		}
	}

	// Publish domain event
	actorFullName := auditctx.GetFullName(ctx)
	if actorFullName == "" {
		actorFullName = "Unknown"
	}
	event := domain.NewProjectUpdatedEvent(ctx, project, auditctx.GetUserIDOrZero(ctx), actorFullName, originalProject)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish ProjectUpdatedEvent", "project_id", project.ID, "error", err)
	}

	return nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uint, deletedBy uint) error {
	project, err := s.ProjectRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if project has approved timesheets - if so, prevent deletion
	if err := s.validateProjectCanBeDeleted(ctx, id); err != nil {
		s.logger.Warn("Project deletion blocked due to validation failure",
			"project_id", id, "project_name", project.Name, "error", err)
		return err
	}

	s.logger.Info("Starting project deletion with cascade cleanup",
		"project_id", id, "project_name", project.Name, "deleted_by", deletedBy)

	// Perform cascade deletion of related entities
	if err := s.cascadeDeleteProjectRelations(ctx, id); err != nil {
		s.logger.Error("Failed to cascade delete project relations",
			"project_id", id, "error", err)
		return fmt.Errorf("failed to delete related project data: %w", err)
	}

	// Delete the project itself
	if err := s.ProjectRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.Info("Project successfully deleted", "project_id", id, "project_name", project.Name)

	// Publish domain event
	actorFullName := auditctx.GetFullName(ctx)
	if actorFullName == "" {
		actorFullName = "Unknown"
	}
	event := domain.NewProjectDeletedEvent(ctx, project.ID, project.Name, project.Code, deletedBy, actorFullName)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish ProjectDeletedEvent", "project_id", id, "error", err)
	}

	return nil
}

func (s *ProjectService) ListProjects(ctx context.Context, filters domain.ProjectFilters) ([]*domain.Project, error) {
	return s.ProjectRepo.List(ctx, filters)
}

func (s *ProjectService) ListProjectsWithEmployeeCount(ctx context.Context, filters domain.ProjectFilters) ([]*domain.ProjectWithEmployeeCount, error) {
	// Apply a short-lived microcache for high-traffic project list endpoints.
	if filters.Limit > 0 {
		cacheKey := s.generateProjectListCacheKey(filters)

		var cached []*domain.ProjectWithEmployeeCount
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}

		result, err := s.ProjectRepo.ListWithEmployeeCount(ctx, filters)
		if err != nil {
			return nil, err
		}

		if len(result) > 0 {
			_ = s.cache.Set(ctx, cacheKey, result, constants.ProjectListCacheTTL)
		}

		return result, nil
	}

	// Use the repository's ListWithEmployeeCount method which handles the two-query approach correctly
	return s.ProjectRepo.ListWithEmployeeCount(ctx, filters)
}

func (s *ProjectService) CountProjects(ctx context.Context, filters domain.ProjectFilters) (int64, error) {
	return s.ProjectRepo.Count(ctx, filters)
}

func (s *ProjectService) GetProjectSummary(ctx context.Context) (*domain.ProjectSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("project_summary")
	var summary domain.ProjectSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.ProjectRepo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.ProjectListCacheTTL)

	return result, nil
}

func (s *ProjectService) GetPartnerProjectSummary(ctx context.Context, createdBy uint) (*domain.PartnerProjectSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("partner_project_summary", fmt.Sprintf("%d", createdBy))
	var summary domain.PartnerProjectSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	// Get projects count for this partner (filter by creator)
	filters := domain.ProjectFilters{
		CreatedBy: &createdBy,
	}
	projects, err := s.ProjectRepo.List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to list projects for partner summary", "created_by", createdBy, "error", err)
		return nil, domain.NewInternalError(constants.MsgFailedToGetProjectsVN, err)
	}

	// Get status counts for this partner's projects only
	statusCounts, err := s.ProjectRepo.GetStatusCountsForCreator(ctx, createdBy)
	if err != nil {
		s.logger.Error("Failed to get status counts for partner summary", "created_by", createdBy, "error", err)
		return nil, domain.NewInternalError(constants.MsgFailedToGetStatusCountsVN, err)
	}

	// Get total active employees created by this partner
	activeEmployeeCount, err := s.EmployeeRepo.GetActiveCountAtDateForCreator(ctx, clock.Now(), createdBy)
	if err != nil {
		s.logger.Error("Failed to get active employee count for partner summary", "created_by", createdBy, "error", err)
		return nil, domain.NewInternalError(constants.MsgFailedToGetActiveEmployeeCountVN, err)
	}

	// Calculate summary statistics for this partner only
	result := &domain.PartnerProjectSummary{
		TotalProjects:     len(projects),
		ActiveProjects:    statusCounts["active"],
		CompletedProjects: statusCounts["completed"],
		TotalEmployees:    activeEmployeeCount,
	}

	s.logger.Info("Partner project summary calculated",
		"total_projects", result.TotalProjects,
		"active_projects", result.ActiveProjects,
		"completed_projects", result.CompletedProjects,
		"total_employees", result.TotalEmployees,
	)

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.ProjectListCacheTTL)

	return result, nil
}

// generateProjectListCacheKey builds a deterministic cache key for project list queries.
func (s *ProjectService) generateProjectListCacheKey(filters domain.ProjectFilters) string {
	key := "projects:list"

	if len(filters.ProjectStatus) > 0 {
		var statuses []string
		for _, status := range filters.ProjectStatus {
			statuses = append(statuses, string(status))
		}
		key = fmt.Sprintf("%s:status:%s", key, strings.Join(statuses, ","))
	}
	if filters.CreatedBy != nil {
		key = fmt.Sprintf("%s:created_by:%d", key, *filters.CreatedBy)
	}
	if filters.AccessibleBy != nil {
		key = fmt.Sprintf("%s:accessible_by:%d", key, *filters.AccessibleBy)
	}
	if filters.Search != "" {
		key = fmt.Sprintf("%s:search:%s", key, strings.ToLower(strings.TrimSpace(filters.Search)))
	}
	if filters.FromDate != nil {
		key = fmt.Sprintf("%s:from:%s", key, filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		key = fmt.Sprintf("%s:to:%s", key, filters.ToDate.Format("2006-01-02"))
	}
	if filters.Limit > 0 {
		key = fmt.Sprintf("%s:limit:%d", key, filters.Limit)
	}
	if filters.Offset > 0 {
		key = fmt.Sprintf("%s:offset:%d", key, filters.Offset)
	}
	if filters.SortBy != "" {
		key = fmt.Sprintf("%s:sort_by:%s", key, filters.SortBy)
	}
	if filters.SortOrder != "" {
		key = fmt.Sprintf("%s:sort_order:%s", key, filters.SortOrder)
	}

	return key
}

// AutoActivateProjects finds all draft projects whose start date has arrived and activates them
func (s *ProjectService) AutoActivateProjects(ctx context.Context) error {
	s.logger.Info("Starting auto-activation of projects based on start date")

	// Get all draft projects with start dates today or in the past
	today := clock.NowUTC().Truncate(24 * time.Hour)

	draftProjects, err := s.ProjectRepo.GetPendingActivatedProjects(ctx, today)
	if err != nil {
		s.logger.Error("Failed to get draft projects for auto-activation", "error", err)
		return fmt.Errorf("failed to get draft projects: %w", err)
	}

	activatedCount := 0
	for _, project := range draftProjects {
		// Check if project has a start date and it's today or in the past
		if project.StartDate != nil && !project.StartDate.After(today) {
			// Check if transition is allowed
			if !project.CanTransitionTo(domain.ProjectStatusRunning) {
				s.logger.Warn("Cannot transition project to active status",
					"project_id", project.ID,
					"current_status", project.ProjectStatus)
				continue
			}

			// Update project status to active
			originalStatus := project.ProjectStatus
			project.ProjectStatus = domain.ProjectStatusRunning

			if err := s.ProjectRepo.Update(ctx, project); err != nil {
				s.logger.Error("Failed to auto-activate project",
					"project_id", project.ID,
					"project_name", project.Name,
					"start_date", project.StartDate.Format("2006-01-02"),
					"error", err)
				continue
			}

			activatedCount++

			s.logger.Info("Auto-activated project",
				"project_id", project.ID,
				"project_name", project.Name,
				"start_date", project.StartDate.Format("2006-01-02"),
				"previous_status", originalStatus,
				"new_status", project.ProjectStatus)

			// Publish ProjectUpdatedEvent for auto-activation
			event := domain.NewProjectUpdatedEvent(ctx, project, 0, "System", nil)
			if err := s.events.Publish(ctx, event); err != nil {
				s.logger.Warn("Failed to publish ProjectUpdated event for auto-activation",
					"projectID", project.ID,
					"error", err)
			}
		}
	}

	return nil
}

// AutoCompleteProjects finds all draft or active projects whose end date has arrived and completes them
func (s *ProjectService) AutoCompleteProjects(ctx context.Context) error {
	s.logger.Info("Starting auto-completion of projects based on end date")

	// Get all draft or active projects with end dates today or in the past
	today := clock.NowUTC().Truncate(24 * time.Hour)

	projects, err := s.ProjectRepo.GetPendingCompletedProjects(ctx, today)
	if err != nil {
		s.logger.Error("Failed to get projects for auto-completion", "error", err)
		return fmt.Errorf("failed to get projects for completion: %w", err)
	}

	completedCount := 0
	for _, project := range projects {
		// Check if project has an end date and it's today or in the past
		if project.EndDate != nil && !project.EndDate.After(today) {
			// Check if transition is allowed
			if !project.CanTransitionTo(domain.ProjectStatusCompleted) {
				s.logger.Warn("Cannot transition project to completed status",
					"project_id", project.ID,
					"current_status", project.ProjectStatus)
				continue
			}

			// Update project status to completed
			originalStatus := project.ProjectStatus
			project.ProjectStatus = domain.ProjectStatusCompleted

			if err := s.ProjectRepo.Update(ctx, project); err != nil {
				s.logger.Error("Failed to auto-complete project",
					"project_id", project.ID,
					"project_name", project.Name,
					"end_date", project.EndDate.Format("2006-01-02"),
					"error", err)
				continue
			}

			completedCount++

			s.logger.Info("Auto-completed project",
				"project_id", project.ID,
				"project_name", project.Name,
				"end_date", project.EndDate.Format("2006-01-02"),
				"previous_status", originalStatus,
				"new_status", project.ProjectStatus)

			// Publish ProjectUpdatedEvent for auto-completion
			event := domain.NewProjectUpdatedEvent(ctx, project, 0, "System", nil)
			if err := s.events.Publish(ctx, event); err != nil {
				s.logger.Warn("Failed to publish ProjectUpdated event for auto-completion",
					"projectID", project.ID,
					"error", err)
			}
		}
	}

	return nil
}

// generateProjectCode creates a project code from client name initials + sequential number
// Example: "Acme Corporation" -> "AC001", "AC002", etc.
func (s *ProjectService) generateProjectCode(clientName string) string {
	// Extract initials from client name
	initials := s.extractInitials(clientName)

	// Find the next available number for this client
	// Start from 001 and increment until we find an unused code
	for i := 1; i <= 999; i++ {
		code := fmt.Sprintf("%s%03d", initials, i)

		// Check if this code already exists
		// Note: In a production system, you might want to do this in a transaction
		// or use a database sequence to avoid race conditions
		if !s.codeExists(code) {
			return code
		}
	}

	// Fallback to timestamp-based code if we exhaust 999 possibilities
	timestamp := clock.Now().Unix()
	return fmt.Sprintf("%s%d", initials, timestamp%10000)
}

// extractInitials extracts initials from a client name
// Example: "Acme Corporation Ltd" -> "ACL"
func (s *ProjectService) extractInitials(clientName string) string {
	if clientName == "" {
		return "XX"
	}

	words := strings.Fields(strings.TrimSpace(clientName))
	if len(words) == 0 {
		return "XX"
	}

	var initials strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			// Take the first character and convert to uppercase
			char := strings.ToUpper(string(word[0]))
			// Only include alphabetic characters
			if char >= "A" && char <= "Z" {
				initials.WriteString(char)
			}
		}
		// Limit to maximum 3 initials
		if initials.Len() >= 3 {
			break
		}
	}

	// Ensure we have at least 2 characters
	result := initials.String()
	if len(result) < 2 {
		result = result + "X"
		if len(result) < 2 {
			result = "XX"
		}
	}

	return result
}

// codeExists checks if a project code already exists in the database
func (s *ProjectService) codeExists(code string) bool {
	// Use CodeExistsIncludingDeleted to check both active and soft-deleted projects
	// This prevents code conflicts with soft-deleted projects due to unique index constraint
	return s.ProjectRepo.CodeExistsIncludingDeleted(context.Background(), code)
}

// autoTerminateEmployees terminates all active employees for a project when it's closed
func (s *ProjectService) autoTerminateEmployees(ctx context.Context, project *domain.Project, updatedBy uint) error {
	s.logger.Info("Auto-terminating employees for closed project",
		"project_id", project.ID,
		"project_status", project.ProjectStatus)

	// Get all active project employees (those without LastDate)
	filters := domain.ProjectEmployeeFilters{
		ProjectID:  &project.ID,
		ActiveOnly: true,
		Limit:      1000,
		Offset:     0,
	}

	activeAssignments, err := s.ProjectEmployeeRepo.List(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to get active project employees: %w", err)
	}

	if len(activeAssignments) == 0 {
		s.logger.Info("No active employees to terminate for project", "project_id", project.ID)
		return nil
	}

	// Determine termination date
	var terminationDate time.Time
	if project.EndDate != nil {
		terminationDate = *project.EndDate
	} else {
		terminationDate = clock.NowUTC().Truncate(24 * time.Hour)
	}

	// Collect IDs of active assignments
	assignmentIDs := make([]uint, 0, len(activeAssignments))
	for _, a := range activeAssignments {
		if a.LastDate == nil {
			assignmentIDs = append(assignmentIDs, a.ID)
		}
	}

	if len(assignmentIDs) == 0 {
		return nil
	}

	// Batch-update all active assignments in a single query
	if err := s.ProjectEmployeeRepo.BulkUpdateLastDate(ctx, assignmentIDs, terminationDate); err != nil {
		s.logger.Error("Failed to bulk-terminate employee assignments",
			"project_id", project.ID,
			"error", err)
		return fmt.Errorf("failed to terminate employee assignments: %w", err)
	}

	s.logger.Info("Completed auto-termination of employees for project closure",
		"project_id", project.ID,
		"terminated_employees", len(assignmentIDs),
		"termination_date", terminationDate.Format("2006-01-02"))

	// Publish events for each terminated assignment
	for _, assignment := range activeAssignments {
		if assignment.LastDate == nil {
			event := domain.NewProjectEmployeeUpdatedEvent(ctx, assignment)
			if err := s.events.Publish(ctx, event); err != nil {
				s.logger.Warn("Failed to publish ProjectEmployeeUpdated event for auto-termination",
					"projectID", project.ID,
					"assignmentID", assignment.ID,
					"error", err)
			}
		}
	}

	return nil
}

// validateProjectCanBeDeleted checks if a project can be safely deleted
// Projects cannot be deleted if they have approved timesheets
func (s *ProjectService) validateProjectCanBeDeleted(ctx context.Context, projectID uint) error {
	// Check if project has any approved timesheets
	filters := domain.TimesheetFilters{
		ProjectIDs:      []uint{projectID},
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		Limit:           1, // Just need to know if any exist
	}

	timesheets, err := s.getTimesheetRepo().List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to check for approved timesheets", "project_id", projectID, "error", err)
		return domain.NewInternalError(constants.MsgFailedToValidateProjectDeletionVN, err)
	}

	if len(timesheets) > 0 {
		return domain.NewValidationError(constants.MsgCannotDeleteProjectWithApprovedTimesheetsVN)
	}

	return nil
}

// cascadeDeleteProjectRelations deletes all related entities for a project
// This includes timesheets and employee assignments
func (s *ProjectService) cascadeDeleteProjectRelations(ctx context.Context, projectID uint) error {
	s.logger.Info("Starting cascade deletion of project relations", "project_id", projectID)

	// Delete all timesheets for this project
	if err := s.deleteProjectTimesheets(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project timesheets: %w", err)
	}

	// Delete all employee assignments for this project
	if err := s.deleteProjectEmployeeAssignments(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project employee assignments: %w", err)
	}

	s.logger.Info("Completed cascade deletion of project relations", "project_id", projectID)
	return nil
}

// deleteProjectTimesheets removes all timesheets associated with a project
func (s *ProjectService) deleteProjectTimesheets(ctx context.Context, projectID uint) error {
	timesheetRepo := s.getTimesheetRepo()
	if timesheetRepo == nil {
		s.logger.Warn("Timesheet repository not available for cascade deletion", "project_id", projectID)
		return nil
	}

	if err := timesheetRepo.DeleteByProjectID(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project timesheets: %w", err)
	}

	s.logger.Info("Deleted project timesheets", "project_id", projectID)
	return nil
}

// deleteProjectEmployeeAssignments removes all employee assignments for a project
func (s *ProjectService) deleteProjectEmployeeAssignments(ctx context.Context, projectID uint) error {
	if err := s.ProjectEmployeeRepo.DeleteByProjectID(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project employee assignments: %w", err)
	}

	s.logger.Info("Deleted project employee assignments", "project_id", projectID)
	return nil
}

// getTimesheetRepo returns the timesheet repository
func (s *ProjectService) getTimesheetRepo() domain.TimesheetRepository {
	return s.TimesheetRepo
}
