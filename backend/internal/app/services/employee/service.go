package employee

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
)

type EmployeeService struct {
	// Repositories - for data access only
	EmployeeRepo        domain.EmployeeRepository
	TimesheetRepo       domain.TimesheetRepository
	BankRepo            domain.BankRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	EmployeeUserRepo    domain.EmployeeUserRepository
	UserRepo            domain.UserRepository

	// Domain Services - for business logic
	EmployeeDomainService *domainServices.EmployeeDomainService

	// Application Services
	UserService *user.UserService

	// Infrastructure
	TransactionManager domain.TransactionManager
	events             domain.EventBus
	cache              domain.CacheServiceUseCase
}

func NewEmployeeService(cfg *Config) *EmployeeService {
	return &EmployeeService{
		EmployeeRepo:          cfg.EmployeeRepo,
		TimesheetRepo:         cfg.TimesheetRepo,
		BankRepo:              cfg.BankRepo,
		ProjectEmployeeRepo:   cfg.ProjectEmployeeRepo,
		EmployeeUserRepo:      cfg.EmployeeUserRepo,
		UserRepo:              cfg.UserRepo,
		EmployeeDomainService: cfg.EmployeeDomainService,
		UserService:           cfg.UserService,
		TransactionManager:    cfg.TransactionManager,
		events:                cfg.Events,
		cache:                 cfg.Cache,
	}
}

// CreateEmployee orchestrates employee creation using domain services and transaction management
func (s *EmployeeService) CreateEmployee(ctx context.Context, employee *domain.Employee, createdBy uint) (*domain.Employee, error) {
	// Orchestrate employee creation within a transaction
	result, err := s.TransactionManager.WithTransactionResult(ctx, func(txCtx context.Context) (any, error) {
		// Set created by
		employee.CreatedBy = createdBy
		return s.createEmployeeCore(txCtx, employee)
	})

	if err != nil {
		return nil, err
	}

	createdEmployee := result.(*domain.Employee)

	// Publish domain event (outside transaction)
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, createdBy)
	event := domain.NewEmployeeCreatedEvent(ctx, createdEmployee, createdBy, actorFullName)
	if err := s.events.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish EmployeeCreatedEvent", "employee_id", createdEmployee.ID, "error", err)
	}

	return createdEmployee, nil
}

// createEmployeeCore contains the core transactional logic for creating an employee.
// It is shared between the main CreateEmployee flow and any asynchronous handlers that
// may need to participate in the same transaction in the future.
func (s *EmployeeService) createEmployeeCore(ctx context.Context, employee *domain.Employee) (*domain.Employee, error) {
	// Sanitize CCCD and email before persistence
	// Sanitize identifiers and contact fields
	employee.CCCD = strings.TrimSpace(employee.CCCD)
	if employee.Email != nil {
		trimmedEmail := strings.TrimSpace(*employee.Email)
		if trimmedEmail == "" {
			employee.Email = nil
		} else {
			employee.Email = &trimmedEmail
		}
	}

	// Normalize employee fullname to Vietnamese title case
	employee.Fullname = utils.ToVietnameseTitleCase(employee.Fullname)

	// Normalize bank account name if provided
	if employee.BankAccountName != "" {
		employee.BankAccountName = utils.ToVietnameseTitleCase(employee.BankAccountName)
	}

	// 1. Validate employee data (domain entity validation)
	if err := employee.IsValid(); err != nil {
		return nil, err
	}

	// 2. Validate bank reference if provided (infrastructure validation)
	if employee.BankID != nil {
		if _, err := s.BankRepo.GetByID(ctx, *employee.BankID); err != nil {
			if domain.IsNotFoundError(err) {
				return nil, domain.NewValidationError(constants.MsgSelectedBankNotExistVN)
			}
			return nil, fmt.Errorf("%s: %w", constants.MsgFailedToValidateBankVN, err)
		}
	}

	// 3. Check for duplicate employee by CCCD before creating user account
	// This ensures we don't create a user if the employee already exists
	existingEmployee, err := s.EmployeeRepo.GetByCCCD(ctx, employee.CCCD)
	if err == nil && existingEmployee != nil {
		// Employee exists - check if it's soft-deleted
		logger := observability.GetLogger()
		if !existingEmployee.DeletedAt.Valid {
			// Employee is active, this is a true duplicate
			logger.Warn("Employee with CCCD already exists", "cccd", employee.CCCD, "existing_id", existingEmployee.ID)
			return nil, domain.NewConflictError(fmt.Sprintf("Employee with CCCD '%s' already exists", employee.CCCD))
		}
		// Employee is soft-deleted, allow creation (the unique constraint includes deleted_at)
		logger.Info("Employee with CCCD exists but is soft-deleted, allowing creation", "cccd", employee.CCCD, "deleted_id", existingEmployee.ID)
		// Set employee to nil to indicate no active duplicate exists
		existingEmployee = nil
	}

	// 4. Create user account for the employee
	userID, err := s.createUserForEmployee(ctx, employee)
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to create user for employee", "employee_name", employee.Fullname, "error", err)
		// Return error as-is to preserve the actual error message from the user service
		// This avoids hiding the root cause (e.g., duplicate username) behind generic error wrapping
		return nil, err
	}

	// 5. Link user to employee
	employee.UserID = &userID

	// 6. Persist employee (repository operation)
	if err := s.EmployeeRepo.Create(ctx, employee); err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to create employee", "cccd", employee.CCCD, "error", err)
		return nil, fmt.Errorf("%s: %w", constants.MsgFailedToCreateEmployeeVN, err)
	}

	return employee, nil
}

func (s *EmployeeService) GetEmployee(ctx context.Context, id uint) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByID(ctx, id)
}

func (s *EmployeeService) GetEmployeeForUpdate(ctx context.Context, id uint) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByIDForUpdate(ctx, id)
}

// GetUserByID retrieves a user by ID - helper method for employee operations
func (s *EmployeeService) GetUserByID(ctx context.Context, userID uint) (*domain.User, error) {
	return s.UserRepo.GetByID(ctx, userID)
}

// UpdateEmployee orchestrates employee update using domain services and transaction management
func (s *EmployeeService) UpdateEmployee(ctx context.Context, employee *domain.Employee, updatedBy uint) error {
	var originalEmployee *domain.Employee
	var nameChanged bool

	// Orchestrate employee update within a transaction
	err := s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Get original employee data for change detection
		var err error
		originalEmployee, err = s.EmployeeRepo.GetByID(txCtx, employee.ID)
		if err != nil {
			return fmt.Errorf("failed to get original employee data: %w", err)
		}

		// Store original name for comparison after normalization
		originalName := originalEmployee.Fullname

		// Sanitize CCCD and email before persistence
		// Sanitize identifiers and contact fields
		employee.CCCD = strings.TrimSpace(employee.CCCD)
		if employee.Email != nil {
			trimmedEmail := strings.TrimSpace(*employee.Email)
			if trimmedEmail == "" {
				employee.Email = nil
			} else {
				employee.Email = &trimmedEmail
			}
		}

		// Normalize employee fullname to Vietnamese title case
		employee.Fullname = utils.ToVietnameseTitleCase(employee.Fullname)

		// Normalize bank account name if provided
		if employee.BankAccountName != "" {
			employee.BankAccountName = utils.ToVietnameseTitleCase(employee.BankAccountName)
		}

		// Check if name changed after normalization
		nameChanged = originalName != employee.Fullname

		// 1. Validate employee data (domain entity validation)
		if err := employee.IsValid(); err != nil {
			return err
		}

		// 2. Validate bank reference if provided (infrastructure validation)
		if employee.BankID != nil {
			if _, err := s.BankRepo.GetByID(txCtx, *employee.BankID); err != nil {
				if domain.IsNotFoundError(err) {
					return domain.NewValidationError(constants.MsgSelectedBankNotExistVN)
				}
				return fmt.Errorf("%s: %w", constants.MsgFailedToValidateBankVN, err)
			}
		}

		// 3. Update employee (repository operation)
		if err := s.EmployeeRepo.Update(txCtx, employee); err != nil {
			logger := observability.GetLogger()
			logger.Error("Failed to update employee", "employee_id", employee.ID, "error", err)
			return fmt.Errorf("%s: %w", constants.MsgFailedToUpdateEmployeeVN, err)
		}

		// 4. Sync email to linked User account (required for Google OAuth login)
		emailChanged := (employee.Email == nil) != (originalEmployee.Email == nil) ||
			(employee.Email != nil && originalEmployee.Email != nil && *employee.Email != *originalEmployee.Email)
		if emailChanged && employee.UserID != nil {
			linkedUser, err := s.UserRepo.GetByID(txCtx, *employee.UserID)
			if err != nil {
				logger := observability.GetLogger()
				logger.Warn("Failed to fetch linked user for email sync", "user_id", *employee.UserID, "error", err)
			} else {
				linkedUser.Email = employee.Email
				if err := s.UserRepo.Update(txCtx, linkedUser); err != nil {
					logger := observability.GetLogger()
					logger.Error("Failed to sync email to linked user", "user_id", *employee.UserID, "error", err)
					return fmt.Errorf("failed to sync email to user account: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Publish domain events (outside transaction)
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, updatedBy)

	// If name changed, publish EmployeeNameUpdatedEvent first to trigger sync
	if nameChanged {
		nameChangeEvent := domain.NewEmployeeNameUpdatedEvent(
			ctx,
			employee.ID,
			originalEmployee.Fullname,
			employee.Fullname,
			employee.CCCD,
			updatedBy,
			actorFullName,
		)

		if err := s.events.Publish(ctx, nameChangeEvent); err != nil {
			logger := observability.GetLogger()
			logger.Warn("Failed to publish EmployeeNameUpdatedEvent", "employee_id", employee.ID, "error", err)
		}
	}

	// Always publish EmployeeUpdatedEvent for general audit purposes
	event := domain.NewEmployeeUpdatedEvent(ctx, employee, updatedBy, actorFullName, originalEmployee)
	if err := s.events.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish EmployeeUpdatedEvent", "employee_id", employee.ID, "error", err)
	}

	return nil
}

func (s *EmployeeService) DeleteEmployee(ctx context.Context, id uint, deletedBy uint) error {
	var deletedEmployee *domain.Employee

	// Orchestrate employee deletion within a transaction
	err := s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Get employee details before deletion for event
		var err error
		deletedEmployee, err = s.EmployeeRepo.GetByID(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to get employee for deletion: %w", err)
		}

		// 2. Validate if employee can be deleted and get assignments to delete
		canDelete, assignments, err := s.EmployeeDomainService.CanDeleteEmployee(txCtx, id)
		if err != nil {
			return err
		}

		if !canDelete {
			return err // Error is already properly formatted from domain service
		}

		// 3. Delete all project assignments first
		if len(assignments) > 0 {
			if err := s.ProjectEmployeeRepo.DeleteAssignmentsByEmployeeID(txCtx, id); err != nil {
				logger := observability.GetLogger()
				logger.Error("Failed to delete employee project assignments", "employee_id", id, "error", err)
				return fmt.Errorf("%s: %w", constants.MsgFailedToDeleteEmployeeAssignmentsVN, err)
			}
		}

		// 4. Delete the employee (soft delete)
		if err := s.EmployeeRepo.Delete(txCtx, id); err != nil {
			logger := observability.GetLogger()
			logger.Error("Failed to delete employee", "employee_id", id, "error", err)
			return fmt.Errorf("%s: %w", constants.MsgFailedToDeleteEmployeeVN, err)
		}

		// 5. Delete related user if exists (soft delete)
		if deletedEmployee.UserID != nil {
			if err := s.UserRepo.Delete(txCtx, *deletedEmployee.UserID); err != nil {
				logger := observability.GetLogger()
				logger.Error("Failed to delete employee's user account", "employee_id", id, "user_id", *deletedEmployee.UserID, "error", err)
				return fmt.Errorf("failed to delete employee's user account: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Publish domain event (outside transaction)
	actorFullName := audit.GetActorFullName(ctx, s.UserRepo, deletedBy)
	event := domain.NewEmployeeDeletedEvent(ctx, deletedEmployee.ID, deletedEmployee.Fullname, deletedEmployee.CCCD, deletedBy, actorFullName)
	if err := s.events.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish EmployeeDeletedEvent", "employee_id", id, "error", err)
	}

	return nil
}

func (s *EmployeeService) ListEmployees(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	return s.EmployeeRepo.List(ctx, filters)
}

func (s *EmployeeService) ListEmployeesWithProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProject, error) {
	return s.EmployeeRepo.ListWithProjects(ctx, filters)
}

func (s *EmployeeService) ListEmployeesWithAllProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	// Apply a short-lived microcache for high-traffic employee list endpoints.
	if filters.Limit > 0 {
		cacheKey := s.generateEmployeeListCacheKey("all_projects", filters)

		var cached []*domain.EmployeeWithProjects
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}

		result, err := s.EmployeeRepo.ListWithAllProjects(ctx, filters)
		if err != nil {
			return nil, err
		}

		if len(result) > 0 {
			_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeeListCacheTTL)
		}

		return result, nil
	}

	return s.EmployeeRepo.ListWithAllProjects(ctx, filters)
}

func (s *EmployeeService) CountEmployees(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	return s.EmployeeRepo.Count(ctx, filters)
}

func (s *EmployeeService) GetEmployeeByCCCD(ctx context.Context, cccd string) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByCCCD(ctx, cccd)
}

// GetEmployeeStatistics delegates to domain service for business logic
func (s *EmployeeService) GetEmployeeStatistics(ctx context.Context) (*domain.EmployeeStatistics, error) {
	return s.EmployeeDomainService.GetEmployeeStatistics(ctx)
}

// GetEmployeesSummary delegates to domain service for business logic with caching
func (s *EmployeeService) GetEmployeesSummary(ctx context.Context) (*domain.EmployeesSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employees_summary")
	var summary domain.EmployeesSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.EmployeeDomainService.GetEmployeesSummary(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)

	return result, nil
}

// GetEmployeesSummaryForCreator gets summary for employees accessible to a specific user with caching
// Includes employees created by the user, shared with the user, or assigned to projects by the user
func (s *EmployeeService) GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*domain.EmployeesSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employees_summary_creator", strconv.FormatUint(uint64(createdBy), 10))
	var summary domain.EmployeesSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.EmployeeDomainService.GetEmployeesSummaryForCreator(ctx, createdBy)
	if err != nil {
		return nil, err
	}

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)

	return result, nil
}

// generateEmployeeListCacheKey builds a deterministic cache key for employee list queries.
func (s *EmployeeService) generateEmployeeListCacheKey(operation string, filters domain.EmployeeFilters) string {
	var b strings.Builder
	b.WriteString("employees:")
	b.WriteString(operation)

	if filters.CreatedBy != nil {
		b.WriteString(":created_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.CreatedBy), 10))
	}
	if filters.ProjectID != nil {
		b.WriteString(":project_id:")
		b.WriteString(strconv.FormatUint(uint64(*filters.ProjectID), 10))
	}
	if len(filters.ProjectIDs) > 0 {
		b.WriteString(":project_ids:")
		for i, id := range filters.ProjectIDs {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.FormatUint(uint64(id), 10))
		}
	}
	if filters.AccessibleBy != nil {
		b.WriteString(":accessible_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.AccessibleBy), 10))
	}
	if filters.Search != "" {
		b.WriteString(":search:")
		b.WriteString(strings.ToLower(strings.TrimSpace(filters.Search)))
	}
	if filters.Status != "" {
		b.WriteString(":status:")
		b.WriteString(filters.Status)
	}
	if filters.FromDate != nil {
		b.WriteString(":from:")
		b.WriteString(filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		b.WriteString(":to:")
		b.WriteString(filters.ToDate.Format("2006-01-02"))
	}
	if filters.Limit > 0 {
		b.WriteString(":limit:")
		b.WriteString(strconv.Itoa(filters.Limit))
	}
	if filters.Offset > 0 {
		b.WriteString(":offset:")
		b.WriteString(strconv.Itoa(filters.Offset))
	}
	if filters.SortBy != "" {
		b.WriteString(":sort_by:")
		b.WriteString(filters.SortBy)
	}
	if filters.SortOrder != "" {
		b.WriteString(":sort_order:")
		b.WriteString(filters.SortOrder)
	}

	return b.String()
}

// SearchEmployees delegates to domain service for normalized Vietnamese search
func (s *EmployeeService) SearchEmployees(ctx context.Context, query string, limit int) ([]*domain.EmployeeWithProject, error) {
	return s.EmployeeDomainService.SearchEmployees(ctx, query, limit)
}

// EmployeeSummary represents individual employee summary data
type EmployeeSummary struct {
	TotalPayrollPayments int
	TotalEarningsVND     int64
	LastPaymentDate      *time.Time
	AvgWeeklyEarningsVND int64
}

func (s *EmployeeService) GetEmployeeSummary(ctx context.Context, employeeID uint) (*EmployeeSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employee_summary", strconv.FormatUint(uint64(employeeID), 10))
	var summary EmployeeSummary
	if err := s.cache.Get(ctx, cacheKey, &summary); err == nil {
		return &summary, nil
	}

	// Fetch all paid timesheets for this employee, sorted by payment_date desc.
	// We use the most recent payment_date to determine the "last paid month", then
	// aggregate totals for that month — all from this single result set.
	filters := domain.TimesheetFilters{
		EmployeeID:    &employeeID,
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		Limit:         1000,
		SortBy:        "payment_date",
		SortOrder:     "desc",
	}

	timesheets, err := s.TimesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get paid timesheets: %w", err)
	}

	empty := &EmployeeSummary{}
	if len(timesheets) == 0 || timesheets[0].PaymentDate == nil {
		_ = s.cache.Set(ctx, cacheKey, empty, constants.EmployeeTimesheetSummaryCacheTTL)
		return empty, nil
	}

	// Determine the last paid month from the first (most recent) timesheet.
	lastPaid := timesheets[0].PaymentDate
	year, month, _ := lastPaid.Date()
	loc := lastPaid.Location()
	firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// Aggregate totals for that month from the already-fetched slice (no second DB call).
	var totalEarnings int64
	var lastPaymentDate *time.Time
	count := 0

	for _, ts := range timesheets {
		if ts.PaymentDate == nil {
			continue
		}
		if ts.PaymentDate.Before(firstDayOfMonth) || ts.PaymentDate.After(lastDayOfMonth) {
			continue
		}
		if ts.Amount < 0 {
			continue
		}
		totalEarnings += ts.Amount
		count++
		if lastPaymentDate == nil || ts.PaymentDate.After(*lastPaymentDate) {
			lastPaymentDate = ts.PaymentDate
		}
	}

	avgWeekly := int64(0)
	if totalEarnings > 0 {
		avgWeekly = (totalEarnings + 2) / 4
	}

	result := &EmployeeSummary{
		TotalPayrollPayments: count,
		TotalEarningsVND:     totalEarnings,
		LastPaymentDate:      lastPaymentDate,
		AvgWeeklyEarningsVND: avgWeekly,
	}
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)
	return result, nil
}

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

// createUserForEmployee creates a user account for an employee with auto-generated username and default password
func (s *EmployeeService) createUserForEmployee(ctx context.Context, employee *domain.Employee) (uint, error) {
	logger := observability.GetLogger()
	logger.Info("Creating user account for employee", "employee_name", employee.Fullname)

	// Generate username from employee's fullname
	baseUsername := utils.GenerateUsername(employee.Fullname)
	if baseUsername == "" {
		logger.Error("Failed to generate username", "employee_name", employee.Fullname)
		return 0, fmt.Errorf("failed to generate username for employee: %s", employee.Fullname)
	}

	logger.Info("Generated base username", "base_username", baseUsername)

	// Ensure username is unique
	username := s.ensureUniqueUsername(ctx, baseUsername)
	logger.Info("Ensured unique username", "final_username", username)

	// Prepare email for user account
	var userEmail string
	if employee.Email != nil && *employee.Email != "" {
		userEmail = *employee.Email
	}

	// Create user using UserService
	logger.Info("Calling CreateUser", "username", username, "email", userEmail)
	user, err := s.UserService.CreateUser(ctx, dto.CreateUserRequest{
		Username: username,
		Email:    userEmail,
		Password: DefaultEmployeePassword,
		Fullname: employee.Fullname,
		Role:     string(domain.RoleEmployee),
	})
	// Return error as-is to preserve the actual error message from the repository layer
	// This avoids hiding the root cause (e.g., duplicate username) behind generic error wrapping
	if err != nil {
		logger.Error("Failed to create user for employee",
			"employee_name", employee.Fullname,
			"username", username,
			"error_type", fmt.Sprintf("%T", err),
			"error_message", err.Error())
		return 0, err
	}

	logger.Info("User account created successfully for employee",
		"employee_name", employee.Fullname,
		"user_id", user.ID,
		"username", username)

	return user.ID, nil
}

// ensureUniqueUsername generates a unique username by appending numbers if needed.
// Uses a single DB query to fetch all existing variants with the same prefix,
// then picks the lowest available suffix in memory — O(1) DB round-trips.
func (s *EmployeeService) ensureUniqueUsername(ctx context.Context, baseUsername string) string {
	logger := observability.GetLogger()

	existing, err := s.UserRepo.GetUsernamesByPrefix(ctx, baseUsername)
	if err != nil || len(existing) == 0 {
		// No conflicts found — base username is available.
		return baseUsername
	}

	// Build a set of taken usernames for O(1) lookup.
	taken := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		taken[u] = struct{}{}
	}

	// Base username available?
	if _, conflict := taken[baseUsername]; !conflict {
		return baseUsername
	}

	// Find the lowest available numeric suffix.
	for counter := 2; counter <= 1000; counter++ {
		candidate := baseUsername + strconv.Itoa(counter)
		if _, conflict := taken[candidate]; !conflict {
			logger.Debug("Generated unique username", "base", baseUsername, "result", candidate)
			return candidate
		}
	}

	// Fallback: append a short random suffix (extremely unlikely to be needed).
	fallback := baseUsername + strconv.FormatInt(clock.Now().UnixNano()%10000, 10)
	logger.Warn("Username counter exceeded 1000, using timestamp fallback", "base", baseUsername, "fallback", fallback)
	return fallback
}
