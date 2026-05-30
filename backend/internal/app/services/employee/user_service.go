package employee

import (
	"context"
	"fmt"
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
)

const DefaultEmployeePassword = "Vfic@1234"

type EmployeeUserService struct {
	EmployeeRepo domain.EmployeeRepository
	UserService  *user.UserService
}

func NewEmployeeUserService(
	employeeRepo domain.EmployeeRepository,
	userService *user.UserService,
) *EmployeeUserService {
	return &EmployeeUserService{
		EmployeeRepo: employeeRepo,
		UserService:  userService,
	}
}

// CreatedUserInfo represents information about a newly created user
type CreatedUserInfo struct {
	EmployeeID      uint   `json:"employee_id"`
	EmployeeName    string `json:"employee_name"`
	Username        string `json:"username"`
	DefaultPassword string `json:"default_password"`
}

// InitUserResult represents the result of user initialization
type InitUserResult struct {
	TotalEmployees  int               `json:"total_employees"`
	UsersCreated    int               `json:"users_created"`
	AlreadyHadUsers int               `json:"already_had_users"`
	CreatedUsers    []CreatedUserInfo `json:"created_users"`
	Errors          []string          `json:"errors,omitempty"`
}

// InitializeEmployeeUsers creates user accounts for all employees without user_id
func (s *EmployeeUserService) InitializeEmployeeUsers(ctx context.Context) (*InitUserResult, error) {
	// Get all employees without user_id - bypass accessibility filter
	filters := domain.EmployeeFilters{
		Limit:                   10000,
		Offset:                  0,
		SkipAccessibilityFilter: true,
	}

	employees, err := s.EmployeeRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}

	result := &InitUserResult{
		TotalEmployees:  len(employees),
		UsersCreated:    0,
		AlreadyHadUsers: 0,
		CreatedUsers:    []CreatedUserInfo{},
		Errors:          []string{},
	}

	log := observability.GetLogger()
	log.Info("InitializeEmployeeUsers", "total_fetched", len(employees))

	// Separate employees that need processing
	var needsUser []*domain.Employee
	for _, emp := range employees {
		if emp.UserID != nil {
			result.AlreadyHadUsers++
		} else {
			needsUser = append(needsUser, emp)
		}
	}
	log.Info("InitializeEmployeeUsers summary", "employees_without_users", len(needsUser))

	if len(needsUser) == 0 {
		return result, nil
	}

	// Collect all base usernames we'll need to look up
	baseUsernames := make([]string, 0, len(needsUser))
	for _, emp := range needsUser {
		base := utils.GenerateUsername(emp.Fullname)
		if base != "" {
			baseUsernames = append(baseUsernames, base)
		}
	}

	// Batch-fetch all users (including deleted) whose username starts with any base — 1 query per unique prefix
	// Build a set of prefixes to avoid duplicate queries
	prefixSeen := make(map[string]bool)
	// userByUsername holds all fetched users (active + deleted) keyed by username
	userByUsername := make(map[string]*domain.User)

	for _, prefix := range baseUsernames {
		if prefixSeen[prefix] {
			continue
		}
		prefixSeen[prefix] = true
		usernames, err := s.UserService.UserRepo.GetUsernamesByPrefix(ctx, prefix)
		if err != nil {
			continue // non-fatal; will fall back to individual checks
		}
		for _, uname := range usernames {
			if _, already := userByUsername[uname]; already {
				continue
			}
			// Fetch the full user record (including deleted)
			u, err := s.UserService.UserRepo.GetByUsernameIncludingDeleted(ctx, uname)
			if err == nil && u != nil {
				userByUsername[uname] = u
			}
		}
	}

	// Process each employee that needs a user
	for _, employee := range needsUser {
		baseUsername := utils.GenerateUsername(employee.Fullname)
		if baseUsername == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate username for employee ID %d", employee.ID))
			continue
		}

		log.Info("Checking if user exists", "baseUsername", baseUsername)

		// Check active user first (in-memory)
		if existing, ok := userByUsername[baseUsername]; ok && !existing.DeletedAt.Valid {
			log.Info("User check result", "username", baseUsername, "err", false, "existingUser", true)
			employee.UserID = &existing.ID
			if err := s.EmployeeRepo.Update(ctx, employee); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to link existing user to employee ID %d: %v", employee.ID, err))
				continue
			}
			result.CreatedUsers = append(result.CreatedUsers, CreatedUserInfo{
				EmployeeID:      employee.ID,
				EmployeeName:    employee.Fullname,
				Username:        existing.Username,
				DefaultPassword: "Already exists",
			})
			result.UsersCreated++
			continue
		}

		log.Info("User check result", "username", baseUsername, "err", false, "existingUser", false)

		// Check soft-deleted user (in-memory)
		if deleted, ok := userByUsername[baseUsername]; ok && deleted.DeletedAt.Valid {
			log.Info("Found deleted user, restoring", "username", deleted.Username, "deleted_at", deleted.DeletedAt)
			if err := s.UserService.RestoreUser(ctx, deleted.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore user for employee ID %d: %v", employee.ID, err))
				continue
			}
			employee.UserID = &deleted.ID
			if err := s.EmployeeRepo.Update(ctx, employee); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to link restored user to employee ID %d: %v", employee.ID, err))
				continue
			}
			result.CreatedUsers = append(result.CreatedUsers, CreatedUserInfo{
				EmployeeID:      employee.ID,
				EmployeeName:    employee.Fullname,
				Username:        deleted.Username,
				DefaultPassword: "Restored from deleted account",
			})
			result.UsersCreated++
			continue
		}

		// Resolve unique username in-memory using the pre-fetched map
		username := s.ensureUniqueUsernameFromMap(baseUsername, userByUsername)

		userID, err := s.CreateUserForEmployee(ctx, employee, username)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create user for employee ID %d: %v", employee.ID, err))
			continue
		}

		// Record the new username so subsequent employees don't collide
		userByUsername[username] = &domain.User{Username: username}

		employee.UserID = &userID
		if err := s.EmployeeRepo.Update(ctx, employee); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to link user to employee ID %d: %v", employee.ID, err))
			continue
		}

		result.CreatedUsers = append(result.CreatedUsers, CreatedUserInfo{
			EmployeeID:      employee.ID,
			EmployeeName:    employee.Fullname,
			Username:        username,
			DefaultPassword: DefaultEmployeePassword,
		})
		result.UsersCreated++
	}

	return result, nil
}

// ensureUniqueUsernameFromMap resolves a unique username using an in-memory map of taken names.
func (s *EmployeeUserService) ensureUniqueUsernameFromMap(baseUsername string, taken map[string]*domain.User) string {
	if _, exists := taken[baseUsername]; !exists {
		return baseUsername
	}
	counter := 1
	for {
		candidate := baseUsername + strconv.Itoa(counter)
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
		counter++
	}
}

// createUserForEmployee creates a user account for an employee
func (s *EmployeeUserService) createUserForEmployee(ctx context.Context, employee *domain.Employee, username string) (uint, error) {
	// Prepare email
	email := ""
	if employee.Email != nil {
		email = *employee.Email
	}

	// Create user using UserService which handles password hashing
	req := dto.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: DefaultEmployeePassword,
		Fullname: employee.Fullname,
		Role:     string(domain.RoleEmployee),
	}

	userResp, err := s.UserService.CreateUser(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return userResp.ID, nil
}

// EnsureUniqueUsername generates a unique username by appending numbers if needed.
func (s *EmployeeUserService) EnsureUniqueUsername(ctx context.Context, baseUsername string) string {
	username := baseUsername
	counter := 1

	// Keep trying until we find a unique username
	for {
		exists, err := s.UserService.UserRepo.ExistsByUsername(ctx, username)
		if err != nil || !exists {
			break
		}

		username = baseUsername + strconv.Itoa(counter)
		counter++
	}

	return username
}

// CreateUserForEmployee is an exported helper that creates a user for a given employee.
func (s *EmployeeUserService) CreateUserForEmployee(ctx context.Context, employee *domain.Employee, username string) (uint, error) {
	return s.createUserForEmployee(ctx, employee, username)
}
