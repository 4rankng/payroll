package project

import (
	"context"
	"strings"

	"api-server/internal/app/services/audit"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
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

// canManageProjectAccess reports whether the actor may change a project's
// access list. Admins manage every project. A partner may manage only their
// own project: as its creator (until an admin revokes that intrinsic owner
// privilege) or as the holder of an explicit project_users grant. Any other
// role has no business granting project management rights (PAYR-30).
func (s *ProjectPermissionService) canManageProjectAccess(ctx context.Context, project *domain.Project, actorID uint, actorRole string) (bool, error) {
	if actorRole == string(domain.RoleAdmin) {
		return true, nil
	}
	if actorRole != string(domain.RolePartner) {
		return false, nil
	}
	if project.CreatedBy == actorID && project.OwnerRevokedAt == nil {
		return true, nil
	}
	return s.projectUserRepo.HasAccess(ctx, project.ID, actorID)
}

// GrantProjectAccess grants project access to a user
func (s *ProjectPermissionService) GrantProjectAccess(ctx context.Context, projectID, userID, grantedBy uint, granterRole string) error {
	// Verify the granter may manage this project's access list
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Admins manage access on any project; partner-admins (unrevoked creator or
	// holder of an explicit grant) only on their own project (PAYR-30).
	canManage, err := s.canManageProjectAccess(ctx, project, grantedBy, granterRole)
	if err != nil {
		return err
	}
	if !canManage {
		return domain.NewForbiddenError(constants.MsgOnlyProjectManagerCanGrantAccessVN)
	}

	// Check if user exists and is a partner
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Only partner-role users may hold project management rights. A partner
	// admin targeting any other role is a company-side elevation attempt
	// (admin/accountant/...) and is forbidden, not a validation slip.
	if user.Role != domain.RolePartner {
		if granterRole != string(domain.RoleAdmin) {
			return domain.NewForbiddenError(constants.MsgCannotGrantProjectAccessToNonPartnerVN)
		}
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
		actorName := audit.GetActorFullName(ctx, s.userRepo, grantedBy)
		event := domain.NewProjectAccessGrantedEvent(ctx, projectID, projectName, userID, targetName, string(user.Role), grantedBy, actorName)
		if err := s.eventBus.Publish(ctx, event); err != nil {
			observability.GetLogger().Error("failed to publish ProjectAccessGrantedEvent", "error", err)
		}
	}

	return nil
}

// RevokeProjectAccess revokes project access from a user
func (s *ProjectPermissionService) RevokeProjectAccess(ctx context.Context, projectID, userID, revokedBy uint, revokerRole string) error {
	// Verify the revoker may manage this project's access list
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Revoking the project creator strips the intrinsic owner privilege and is
	// admin-only: partner-admins may only manage the users they share the
	// project with (PAYR-30).
	if userID == project.CreatedBy && revokerRole != string(domain.RoleAdmin) {
		return domain.NewForbiddenError(constants.MsgOnlyAdminCanRevokeProjectCreatorVN)
	}

	// Admins revoke on any project; partner-admins only on their own (PAYR-30).
	canManage, err := s.canManageProjectAccess(ctx, project, revokedBy, revokerRole)
	if err != nil {
		return err
	}
	if !canManage {
		return domain.NewForbiddenError(constants.MsgOnlyProjectManagerCanRevokeAccessVN)
	}

	// Revoking the creator also kills the owner privilege itself, which no
	// project_users row represents (PAYR-30). Idempotent: a later grant re-adds
	// a plain management row; the intrinsic owner privilege is never restored.
	if userID == project.CreatedBy && project.OwnerRevokedAt == nil {
		now := clock.Now()
		project.OwnerRevokedAt = &now
		if err := s.projectRepo.Update(ctx, project); err != nil {
			return err
		}
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
		actorName := audit.GetActorFullName(ctx, s.userRepo, revokedBy)
		event := domain.NewProjectAccessRevokedEvent(ctx, projectID, projectName, userID, targetName, revokedBy, actorName)
		if err := s.eventBus.Publish(ctx, event); err != nil {
			observability.GetLogger().Error("failed to publish ProjectAccessRevokedEvent", "error", err)
		}
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

	// If owner is not in the list, add them — unless an admin revoked their
	// intrinsic owner access (PAYR-30).
	if !ownerAlreadyIncluded && project.OwnerRevokedAt == nil {
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

	// The creator holds intrinsic owner access until an admin revokes it (PAYR-30).
	if project.CreatedBy == userID && project.OwnerRevokedAt == nil {
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
	projectEmployees, err := s.projectEmployeeRepo.GetByProject(ctx, projectID)
	if err != nil {
		return false, nil
	}

	for _, assignment := range projectEmployees {
		hasEmployeeAccess, err := s.employeeUserRepo.HasAccess(ctx, assignment.EmployeeID, userID)
		if err != nil {
			continue
		}
		if hasEmployeeAccess {
			return true, nil
		}
	}

	return false, nil
}

// CanUserModifyProject checks if a user can modify (update/delete) a project.
// Partners can only modify if they are creator or have explicit project access.
// Employee-based access grants read-only permission.
func (s *ProjectPermissionService) CanUserModifyProject(ctx context.Context, projectID, userID uint) (bool, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}

	if project.CreatedBy == userID && project.OwnerRevokedAt == nil {
		return true, nil
	}

	return s.projectUserRepo.HasAccess(ctx, projectID, userID)
}

// CanUserDeleteProject checks if a user can delete a project (only creator)
func (s *ProjectPermissionService) CanUserDeleteProject(ctx context.Context, projectID, userID uint) (bool, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}

	return project.CreatedBy == userID && project.OwnerRevokedAt == nil, nil
}
