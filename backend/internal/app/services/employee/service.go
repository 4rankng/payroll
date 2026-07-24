package employee

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
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

	// bankAccountValidator performs live OnePay account verification.
	// May be nil when no disbursement provider is configured.
	bankAccountValidator *BankAccountValidator
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
		bankAccountValidator:  cfg.BankAccountValidator,
	}
}

// BankAccountValidator exposes the validator so co-located services
// (e.g. ImportService) that already depend on EmployeeService can perform
// account validation without a separate constructor parameter. Returns
// nil when validation is disabled.
func (s *EmployeeService) BankAccountValidator() *BankAccountValidator {
	return s.bankAccountValidator
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

	// 1b. Live OnePay account verification. Non-blocking on invalid
	// (decision: allow + flag) — the outcome is persisted on the entity
	// so the warning list can surface the reason to admins/partners.
	applyBankAccountValidation(ctx, s.bankAccountValidator, employee)

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

		// 1b. Live OnePay account verification. Only re-validate when the
		// bank fields actually changed, to avoid a redundant OnePay call
		// on every unrelated profile edit.
		if bankFieldsChanged(originalEmployee, employee) {
			applyBankAccountValidation(txCtx, s.bankAccountValidator, employee)
		} else if originalEmployee != nil {
			// Preserve the previous validation outcome on the entity so
			// the repository Update doesn't clobber it with the zero value.
			employee.BankAccountStatus = originalEmployee.BankAccountStatus
			employee.BankAccountInvalidReason = originalEmployee.BankAccountInvalidReason
			employee.BankAccountValidatedAt = originalEmployee.BankAccountValidatedAt
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
