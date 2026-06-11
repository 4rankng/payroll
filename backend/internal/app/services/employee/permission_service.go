package employee

import (
	"context"
	"strings"

	"api-server/internal/app/services/audit"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// EmployeePermissionService handles employee access management
type EmployeePermissionService struct {
	employeeRepo        domain.EmployeeRepository
	employeeUserRepo    domain.EmployeeUserRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	userRepo            domain.UserRepository
	eventBus            domain.EventBus
}

// NewEmployeePermissionService creates a new employee permission service
func NewEmployeePermissionService(
	employeeRepo domain.EmployeeRepository,
	employeeUserRepo domain.EmployeeUserRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	userRepo domain.UserRepository,
	eventBus domain.EventBus,
) *EmployeePermissionService {
	return &EmployeePermissionService{
		employeeRepo:        employeeRepo,
		employeeUserRepo:    employeeUserRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		userRepo:            userRepo,
		eventBus:            eventBus,
	}
}

// GrantEmployeeAccess grants employee access to a user
func (s *EmployeePermissionService) GrantEmployeeAccess(ctx context.Context, employeeID, userID, grantedBy uint, granterRole string) error {
	// Verify granter is employee creator or admin
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}

	// Admins can grant access to any employee, partners can only grant access to employees they created
	if granterRole != string(domain.RoleAdmin) && employee.CreatedBy != grantedBy {
		return domain.NewForbiddenError(constants.MsgOnlyEmployeeCreatorOrAdminCanGrantAccessVN)
	}

	// Check if user exists and is a partner
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Only allow granting to partner role users
	if user.Role != domain.RolePartner {
		return domain.NewValidationError(constants.MsgCanOnlyGrantEmployeeAccessToPartnerVN)
	}

	// Prevent granting access to yourself
	if userID == employee.CreatedBy {
		return domain.NewValidationError(constants.MsgCannotGrantAccessToEmployeeCreatorVN)
	}

	// Check if access already exists
	existing, err := s.employeeUserRepo.GetByEmployeeAndUser(ctx, employeeID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.NewValidationError(constants.MsgUserAlreadyHasAccessToEmployeeVN)
	}

	// Grant access
	employeeUser := &domain.EmployeeUser{
		EmployeeID: employeeID,
		UserID:     userID,
		GrantedBy:  grantedBy,
	}

	if err := s.employeeUserRepo.Create(ctx, employeeUser); err != nil {
		return err
	}

	// Publish permission domain event for audit and notifications
	employeeName := strings.TrimSpace(employee.Fullname)

	if s.eventBus != nil {
		actorFullName := audit.GetActorFullName(ctx, s.userRepo, grantedBy)
		event := domain.NewEmployeeAccessGrantedEvent(ctx, employeeID, employeeName, "", 0, string(user.Role), grantedBy, actorFullName)
		if err := s.eventBus.Publish(ctx, event); err != nil {
			observability.GetLogger().Error("failed to publish EmployeeAccessGrantedEvent", "error", err)
		}
	}

	return nil
}

// RevokeEmployeeAccess revokes employee access from a user
func (s *EmployeePermissionService) RevokeEmployeeAccess(ctx context.Context, employeeID, userID, revokedBy uint, revokerRole string) error {
	// Verify revoker is employee creator or admin
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}

	// Admins can revoke access from any employee, partners can only revoke access from employees they created
	if revokerRole != string(domain.RoleAdmin) && employee.CreatedBy != revokedBy {
		return domain.NewForbiddenError(constants.MsgOnlyEmployeeCreatorOrAdminCanRevokeAccessVN)
	}

	// Revoke access (soft delete)
	if err := s.employeeUserRepo.Delete(ctx, employeeID, userID); err != nil {
		return err
	}

	// Publish permission domain event for audit and notifications
	employeeName := strings.TrimSpace(employee.Fullname)

	if s.eventBus != nil {
		actorFullName := audit.GetActorFullName(ctx, s.userRepo, revokedBy)
		event := domain.NewEmployeeAccessRevokedEvent(ctx, employeeID, employeeName, "", 0, revokedBy, actorFullName)
		if err := s.eventBus.Publish(ctx, event); err != nil {
			observability.GetLogger().Error("failed to publish EmployeeAccessRevokedEvent", "error", err)
		}
	}

	return nil
}

// GetEmployeeUsers retrieves all users with access to an employee, including the owner
func (s *EmployeePermissionService) GetEmployeeUsers(ctx context.Context, employeeID uint) ([]*domain.EmployeeUser, error) {
	// Get the employee to identify the owner
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// Get users who were explicitly granted access
	explicitUsers, err := s.employeeUserRepo.GetEmployeeUsers(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// Check if the owner is already in the explicit users list
	ownerAlreadyIncluded := false
	for _, user := range explicitUsers {
		if user.UserID == employee.CreatedBy {
			ownerAlreadyIncluded = true
			break
		}
	}

	// If owner is not in the list, add them
	if !ownerAlreadyIncluded {
		// Get the owner user details
		owner, err := s.userRepo.GetByID(ctx, employee.CreatedBy)
		if err != nil {
			return nil, err
		}

		// Create an employee user entry for the owner
		ownerEmployeeUser := &domain.EmployeeUser{
			ID:         0, // This will be 0 to indicate it's a virtual entry for the owner
			EmployeeID: employeeID,
			UserID:     employee.CreatedBy,
			GrantedBy:  employee.CreatedBy, // Owner granted access to themselves
			CreatedAt:  employee.CreatedAt, // Use employee creation date
			User:       *owner,
		}

		// Add owner to the beginning of the list
		allUsers := make([]*domain.EmployeeUser, 0, len(explicitUsers)+1)
		allUsers = append(allUsers, ownerEmployeeUser)
		allUsers = append(allUsers, explicitUsers...)

		return allUsers, nil
	}

	return explicitUsers, nil
}

// CanUserAccessEmployee checks if a user can access an employee
func (s *EmployeePermissionService) CanUserAccessEmployee(ctx context.Context, employeeID, userID uint) (bool, error) {
	// Check if user is creator
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return false, err
	}

	if employee.CreatedBy == userID {
		return true, nil
	}

	// Check if user has been granted explicit access
	hasExplicitAccess, err := s.employeeUserRepo.HasAccess(ctx, employeeID, userID)
	if err != nil {
		return false, err
	}
	if hasExplicitAccess {
		return true, nil
	}

	// Check if employee is assigned to a project the user has access to
	return s.projectEmployeeRepo.HasAccessViaProject(ctx, employeeID, userID)
}

// CanUserDeleteEmployee checks if a user can delete an employee (only creator)
func (s *EmployeePermissionService) CanUserDeleteEmployee(ctx context.Context, employeeID, userID uint) (bool, error) {
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return false, err
	}

	// Only creator can delete
	return employee.CreatedBy == userID, nil
}

// GetAccessibleEmployeeIDs returns all employee IDs that a user has access to
// This includes employees they created and employees they've been granted access to
func (s *EmployeePermissionService) GetAccessibleEmployeeIDs(ctx context.Context, userID uint) ([]uint, error) {
	// Get employee IDs user has been granted access to via employee_users table
	grantedEmployeeIDs, err := s.employeeUserRepo.GetUserEmployees(ctx, userID)
	if err != nil {
		return nil, err
	}

	return grantedEmployeeIDs, nil
}
