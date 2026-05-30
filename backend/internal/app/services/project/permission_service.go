package project

import (
	"context"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ProjectPermissionService handles project access management
type ProjectPermissionService struct {
	projectRepo         domain.ProjectRepository
	projectUserRepo     domain.ProjectUserRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	employeeUserRepo    domain.EmployeeUserRepository
	userRepo            domain.UserRepository
	eventBus            domain.EventBus
}

// NewProjectPermissionService creates a new project permission service
func NewProjectPermissionService(
	projectRepo domain.ProjectRepository,
	projectUserRepo domain.ProjectUserRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeUserRepo domain.EmployeeUserRepository,
	userRepo domain.UserRepository,
	eventBus domain.EventBus,
) *ProjectPermissionService {
	return &ProjectPermissionService{
		projectRepo:         projectRepo,
		projectUserRepo:     projectUserRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		employeeUserRepo:    employeeUserRepo,
		userRepo:            userRepo,
		eventBus:            eventBus,
	}
}

// GrantProjectAccess grants project access to a user
func (s *ProjectPermissionService) GrantProjectAccess(ctx context.Context, projectID, userID, grantedBy uint, granterRole string) error {
	// Verify granter is project creator or admin
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Admins can grant access to any project, partners can only grant access to projects they created
	if granterRole != string(domain.RoleAdmin) && project.CreatedBy != grantedBy {
		return domain.NewForbiddenError(constants.MsgOnlyProjectCreatorOrAdminCanGrantAccessVN)
	}

	// Check if user exists and is a partner
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Only allow granting to partner role users
	if user.Role != domain.RolePartner {
		return domain.NewValidationError(constants.MsgCanOnlyGrantProjectAccessToPartnerVN)
	}

	// Check if access already exists
	existing, err := s.projectUserRepo.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.NewValidationError(constants.MsgUserAlreadyHasAccessToProjectVN)
	}

	// Grant access
	projectUser := &domain.ProjectUser{
		ProjectID: projectID,
		UserID:    userID,
		GrantedBy: grantedBy,
	}

	if err := s.projectUserRepo.Create(ctx, projectUser); err != nil {
		return err
	}

	// Publish permission domain event for audit and notifications
	projectName := strings.TrimSpace(project.Name)
	targetName := strings.TrimSpace(user.Fullname)
	if targetName == "" {
		targetName = strings.TrimSpace(user.Username)
	}

	if s.eventBus != nil {
		// Get the actor's full name for audit message
		actorName := ""
		if actorUser, err := s.userRepo.GetByID(ctx, grantedBy); err == nil && actorUser != nil {
			actorName = actorUser.Fullname
			if actorName == "" {
				actorName = actorUser.Username
			}
		}

		event := domain.NewProjectAccessGrantedEvent(ctx, projectID, projectName, userID, targetName, string(user.Role), grantedBy, actorName)
		_ = s.eventBus.Publish(ctx, event)
	}

	return nil
}

// RevokeProjectAccess revokes project access from a user
func (s *ProjectPermissionService) RevokeProjectAccess(ctx context.Context, projectID, userID, revokedBy uint, revokerRole string) error {
	// Verify revoker is project creator or admin
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Admins can revoke access from any project, partners can only revoke access from projects they created
	if revokerRole != string(domain.RoleAdmin) && project.CreatedBy != revokedBy {
		return domain.NewForbiddenError(constants.MsgOnlyProjectCreatorOrAdminCanRevokeAccessVN)
	}

	// Revoke access (soft delete)
	if err := s.projectUserRepo.Delete(ctx, projectID, userID); err != nil {
		return err
	}

	// Publish permission domain event for audit and notifications
	projectName := strings.TrimSpace(project.Name)

	targetName := ""
	if targetUser, err := s.userRepo.GetByID(ctx, userID); err == nil && targetUser != nil {
		targetName = strings.TrimSpace(targetUser.Fullname)
		if targetName == "" {
			targetName = strings.TrimSpace(targetUser.Username)
		}
	}

	if s.eventBus != nil {
		// Get the actor's full name for audit message
		actorName := ""
		if actorUser, err := s.userRepo.GetByID(ctx, revokedBy); err == nil && actorUser != nil {
			actorName = actorUser.Fullname
			if actorName == "" {
				actorName = actorUser.Username
			}
		}

		event := domain.NewProjectAccessRevokedEvent(ctx, projectID, projectName, userID, targetName, revokedBy, actorName)
		_ = s.eventBus.Publish(ctx, event)
	}

	return nil
}

// GetProjectUsers retrieves all users with access to a project, including the owner
func (s *ProjectPermissionService) GetProjectUsers(ctx context.Context, projectID uint) ([]*domain.ProjectUser, error) {
	// Get the project to identify the owner
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Get users who were explicitly granted access
	explicitUsers, err := s.projectUserRepo.GetProjectUsers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if the owner is already in the explicit users list
	ownerAlreadyIncluded := false
	for _, user := range explicitUsers {
		if user.UserID == project.CreatedBy {
			ownerAlreadyIncluded = true
			break
		}
	}

	// If owner is not in the list, add them
	if !ownerAlreadyIncluded {
		// Get the owner user details
		owner, err := s.userRepo.GetByID(ctx, project.CreatedBy)
		if err != nil {
			return nil, err
		}

		// Create a project user entry for the owner
		ownerProjectUser := &domain.ProjectUser{
			ID:        0, // This will be 0 to indicate it's a virtual entry for the owner
			ProjectID: projectID,
			UserID:    project.CreatedBy,
			GrantedBy: project.CreatedBy, // Owner granted access to themselves
			CreatedAt: project.CreatedAt, // Use project creation date
			User:      *owner,
		}

		// Add owner to the beginning of the list
		allUsers := make([]*domain.ProjectUser, 0, len(explicitUsers)+1)
		allUsers = append(allUsers, ownerProjectUser)
		allUsers = append(allUsers, explicitUsers...)

		return allUsers, nil
	}

	return explicitUsers, nil
}

// CanUserAccessProject checks if a user can access a project
func (s *ProjectPermissionService) CanUserAccessProject(ctx context.Context, projectID, userID uint) (bool, error) {
	// Check if user is creator
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}

	if project.CreatedBy == userID {
		return true, nil
	}

	// Check if user has been granted explicit project access
	hasProjectAccess, err := s.projectUserRepo.HasAccess(ctx, projectID, userID)
	if err != nil {
		return false, err
	}
	if hasProjectAccess {
		return true, nil
	}

	// Check if user has access to any employees assigned to this project
	// Get all employees assigned to the project
	projectEmployees, err := s.projectEmployeeRepo.GetByProject(ctx, projectID)
	if err != nil {
		// If error getting project employees, don't fail - just return no access
		return false, nil
	}

	// Check if user has access to any of these employees
	for _, assignment := range projectEmployees {
		hasEmployeeAccess, err := s.employeeUserRepo.HasAccess(ctx, assignment.EmployeeID, userID)
		if err != nil {
			continue // Skip on error, check next employee
		}
		if hasEmployeeAccess {
			return true, nil // User has access to at least one employee in the project
		}
	}

	return false, nil
}

// CanUserModifyProject checks if a user can modify (update/delete) a project
// Partners can only modify if they are creator or have explicit project access
// Employee-based access grants read-only permission
func (s *ProjectPermissionService) CanUserModifyProject(ctx context.Context, projectID, userID uint) (bool, error) {
	// Check if user is creator
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}

	if project.CreatedBy == userID {
		return true, nil
	}

	// Check if user has been granted explicit project access (not employee-based)
	return s.projectUserRepo.HasAccess(ctx, projectID, userID)
}

// CanUserDeleteProject checks if a user can delete a project (only creator)
func (s *ProjectPermissionService) CanUserDeleteProject(ctx context.Context, projectID, userID uint) (bool, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}

	// Only creator can delete
	return project.CreatedBy == userID, nil
}
