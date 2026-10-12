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

// GetEmployeesWithMissingMobile returns active employees with no mobile number
// on file — the partner-facing data-quality nudge list. The filters carry the
// caller's accessibility scope (parseEmployeeFilters sets AccessibleBy for
// partners), so a partner never sees another partner's roster.
func (s *EmployeeService) GetEmployeesWithMissingMobile(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	return s.EmployeeRepo.GetEmployeesWithMissingMobile(ctx, filters)
}

// CountEmployeesWithMissingMobile counts the GetEmployeesWithMissingMobile set.
func (s *EmployeeService) CountEmployeesWithMissingMobile(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	return s.EmployeeRepo.CountEmployeesWithMissingMobile(ctx, filters)
}

// CountEmployeesWithMissingBankDetails returns count of employees with missing banking information
func (s *EmployeeService) CountEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	return s.EmployeeRepo.CountEmployeesWithMissingBankDetails(ctx, filters)
}

// ListEmployeesPaidWithoutMobile returns employees who received salary (paid
// timesheets) or a completed FlexPay advance within [from, to] but have no
// mobile number on file. Read-only pass-through to the repository read model.
func (s *EmployeeService) ListEmployeesPaidWithoutMobile(ctx context.Context, from, to time.Time) ([]*domain.EmployeePaidActivity, error) {
	return s.EmployeeRepo.GetPaidWithoutMobile(ctx, from, to)
}

// GetDuplicateIdentities returns phone numbers and CCCDs carried by more than one
// employee record. These are the accounts that cannot log in via mobile, because
// identity.Resolver fails closed rather than picking one of the colliding
// accounts — so resetting a password for one of them appears to do nothing.
// Read-only: it reports, it never merges.
func (s *EmployeeService) GetDuplicateIdentities(ctx context.Context) ([]*domain.DuplicateIdentityGroup, error) {
	return s.EmployeeRepo.GetDuplicateIdentities(ctx)
}

// DuplicateMobileMember is one visible holder of a duplicated mobile number.
type DuplicateMobileMember struct {
	ID       uint
	Fullname string
}

// DuplicateMobileGroupView is one phone number carried by more than one active
// employee, with only the members the caller may see. HiddenCount is the number
// of holders outside the caller's scope — the collision still breaks phone
// login for every holder, so the group is reported even when some names must
// stay hidden. This is the "which number must change" list behind the partner
// dashboard nudge; it reports, it never picks a winner.
type DuplicateMobileGroupView struct {
	Mobile      string
	Employees   []DuplicateMobileMember
	HiddenCount int
}

// GetDuplicateMobileGroups lists mobile collision groups. scopeIDs restricts
// the visible members (nil = unscoped, for admins and internal callers); a
// group with no visible member is dropped because the caller cannot act on it.
func (s *EmployeeService) GetDuplicateMobileGroups(ctx context.Context, scopeIDs []uint) ([]*DuplicateMobileGroupView, error) {
	groups, err := s.EmployeeRepo.GetDuplicateIdentities(ctx)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(groups))
	for _, g := range groups {
		if g.Field == "mobile" {
			values = append(values, g.Value)
		}
	}
	if len(values) == 0 {
		return []*DuplicateMobileGroupView{}, nil
	}

	holders, err := s.EmployeeRepo.ListByMobiles(ctx, values)
	if err != nil {
		return nil, err
	}
	byMobile := make(map[string][]*domain.Employee, len(values))
	for _, holder := range holders {
		byMobile[holder.Mobile] = append(byMobile[holder.Mobile], holder)
	}

	var scope map[uint]bool
	if scopeIDs != nil {
		scope = make(map[uint]bool, len(scopeIDs))
		for _, id := range scopeIDs {
			scope[id] = true
		}
	}

	views := make([]*DuplicateMobileGroupView, 0, len(values))
	for _, value := range values {
		view := &DuplicateMobileGroupView{Mobile: value, Employees: []DuplicateMobileMember{}}
		for _, holder := range byMobile[value] {
			if scope == nil || scope[holder.ID] {
				view.Employees = append(view.Employees, DuplicateMobileMember{ID: holder.ID, Fullname: holder.Fullname})
			} else {
				view.HiddenCount++
			}
		}
		if len(view.Employees) == 0 {
			continue
		}
		views = append(views, view)
	}
	return views, nil
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
