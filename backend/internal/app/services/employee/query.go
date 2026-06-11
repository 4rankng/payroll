package employee

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// GetUnassignedEmployeesAtDate orchestrates retrieval of unassigned employees at a specific date
func (s *EmployeeService) GetUnassignedEmployeesAtDate(ctx context.Context, date time.Time, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	// Extract pagination from filters and delegate to domain service for business logic
	page := (filters.Offset / filters.Limit) + 1
	pageSize := filters.Limit
	return s.EmployeeDomainService.GetUnassignedEmployeesAtDate(ctx, date, page, pageSize)
}

// CountUnassignedEmployeesAtDate orchestrates counting of unassigned employees at a specific date
func (s *EmployeeService) CountUnassignedEmployeesAtDate(ctx context.Context, date time.Time, filters domain.EmployeeFilters) (int64, error) {
	// Delegate to domain service for business logic (filters not needed for count)
	return s.EmployeeDomainService.CountUnassignedEmployeesAtDate(ctx, date)
}

// GetEmployeesWithMissingBankDetails returns employees with missing banking information
func (s *EmployeeService) GetEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	return s.EmployeeRepo.GetEmployeesWithMissingBankDetails(ctx, filters)
}

// CountEmployeesWithMissingBankDetails returns count of employees with missing banking information
func (s *EmployeeService) CountEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	return s.EmployeeRepo.CountEmployeesWithMissingBankDetails(ctx, filters)
}

// ChangeEmployeePassword changes the password for an employee's user account
// RBAC: ADMIN can change any employee password
// RBAC: PARTNER can change passwords for employees they have access to:
// (1) Partner created the employee OR
// (2) Partner is directly shared the employee (employee_users) OR
// (3) Partner is shared a project that the employee is assigned to
func (s *EmployeeService) ChangeEmployeePassword(ctx context.Context, employeeID uint, newPassword string, actorUserID uint, actorRole domain.UserRole) error {
	logger := observability.GetLogger()
	logger.Info("Changing employee password", "employee_id", employeeID, "actor_user_id", actorUserID)

	// 1. Get the employee
	employee, err := s.EmployeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		logger.Error("Employee not found", "employee_id", employeeID, "error", err)
		return err
	}

	// 2. Verify employee has a linked user account
	if employee.UserID == nil {
		logger.Error("Employee has no linked user account", "employee_id", employeeID)
		return domain.NewValidationError(constants.MsgEmployeeNoUserAccountVN)
	}

	// 3. RBAC: PARTNER can only change passwords for employees they have access to
	// Access is granted if ANY of these conditions are met:
	// (1) Partner created the employee
	// (2) Partner is directly shared the employee (via employee_users table)
	// (3) Partner is shared a project that the employee is assigned to
	if actorRole == domain.RolePartner {
		// Check if partner created the employee
		isCreator := employee.CreatedBy == actorUserID

		// Check direct employee sharing
		hasDirectAccess, err := s.EmployeeUserRepo.HasAccess(ctx, employeeID, actorUserID)
		if err != nil {
			logger.Error("Failed to check direct employee access", "employee_id", employeeID, "actor_user_id", actorUserID, "error", err)
			return domain.NewInternalError(constants.MsgFailedToCheckAccessVN, err)
		}

		// Check project-based access if not creator and no direct access
		if !isCreator && !hasDirectAccess {
			hasProjectAccess, err := s.ProjectEmployeeRepo.HasAccessViaProject(ctx, employeeID, actorUserID)
			if err != nil {
				logger.Error("Failed to check employee access via project", "employee_id", employeeID, "actor_user_id", actorUserID, "error", err)
				return domain.NewInternalError(constants.MsgFailedToCheckAccessVN, err)
			}
			if !hasProjectAccess {
				logger.Error("Partner attempting to change password for employee not accessible",
					"employee_id", employeeID,
					"actor_user_id", actorUserID)
				return domain.NewForbiddenError(constants.MsgForbiddenVN)
			}
		}
	}

	// 4. Delegate password reset to UserService which handles validation, hashing, and persistence
	if err := s.UserService.ResetUserPassword(ctx, *employee.UserID, newPassword); err != nil {
		logger.Error("Failed to reset user password", "employee_id", employeeID, "user_id", *employee.UserID, "error", err)
		return err
	}

	logger.Info("Employee password changed successfully", "employee_id", employeeID, "user_id", *employee.UserID)
	return nil
}
