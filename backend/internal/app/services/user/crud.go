package user

import (
	"context"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	auditservice "api-server/internal/app/services/audit"
	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/pkg/utils"
)

// CreateUser creates a new user with proper validation and security measures
func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
	s.logger.Info("Creating new user", "email", req.Email, "username", req.Username, "role", req.Role)

	// Validate password strength
	if err := s.passwordValidator.Validate(req.Password); err != nil {
		s.logger.Info("Password validation failed", "error", err.Error())
		return nil, domain.NewValidationError(err.Error())
	}

	// Check if user already exists (only check email if provided)
	if req.Email != "" {
		if _, err := s.UserRepo.GetByEmail(ctx, req.Email); err == nil {
			s.logger.Info("User creation failed: email already exists", "email", req.Email)
			return nil, domain.NewConflictError(constants.MsgUserWithEmailExistsVN)
		}
	}

	if _, err := s.UserRepo.GetByUsername(ctx, req.Username); err == nil {
		s.logger.Info("User creation failed: username already exists", "username", req.Username)
		return nil, domain.NewConflictError(constants.MsgUserWithUsernameExistsVN)
	}

	// Hash password securely
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password during user creation", "error", err)
		return nil, domain.NewInternalError(constants.MsgFailedToHashPasswordVN, err)
	}

	// Set default role if not provided
	role := domain.UserRole(req.Role)
	if role == "" {
		role = domain.RolePartner
		s.logger.Info("Using default role for new user", "role", role)
	}

	// Normalize fullname to Vietnamese title case
	normalizedFullname := utils.ToVietnameseTitleCase(req.Fullname)

	// Create user entity
	user := &domain.User{
		Username: req.Username,
		Password: hashedPassword,
		Fullname: normalizedFullname,
		Role:     role,
	}

	// Set email to nil if empty, otherwise set to pointer
	if req.Email != "" {
		user.Email = &req.Email
	}

	// Save to database
	if err := s.UserRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to save user to database", "error", err, "email", req.Email)
		return nil, err
	}

	// Publish domain event
	actorFullName := auditservice.GetActorFullName(ctx, s.UserRepo, user.ID)
	actorUserID := auditctx.GetUserIDOrZero(ctx)
	event := domain.NewUserCreatedEvent(ctx, user, actorUserID, actorFullName)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish UserCreatedEvent", "error", err, "user_id", user.ID)
	}

	response := dto.ToUserResponse(user)

	var logEmail string
	if user.Email != nil {
		logEmail = *user.Email
	}
	s.logger.Info("User created successfully",
		"user_id", user.ID,
		"email", logEmail,
		"username", user.Username,
		"role", user.Role)

	return &response, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id uint) (*domain.User, error) {
	s.logger.Info("Getting user by ID", "user_id", id)

	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Info("User not found", "user_id", id, "error", err)
		return nil, err
	}

	s.logger.Info("User retrieved successfully", "user_id", id, "username", user.Username)
	return user, nil
}

// GetUserResponse retrieves a user by ID and returns it as a response DTO
func (s *UserService) GetUserResponse(ctx context.Context, id uint) (*dto.UserResponse, error) {
	s.logger.Info("Getting user response by ID", "user_id", id)

	user, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	response := dto.ToUserResponse(user)
	return &response, nil
}

// UpdateUser updates an existing user's information
func (s *UserService) UpdateUser(ctx context.Context, id uint, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	var logEmail, logUsername string
	if req.Email != nil {
		logEmail = *req.Email
	}
	if req.Username != nil {
		logUsername = *req.Username
	}
	s.logger.Info("Updating user", "user_id", id, "email", logEmail, "username", logUsername)

	// Get existing user
	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Info("User not found for update", "user_id", id, "error", err)
		return nil, err
	}

	originalUser := *user
	originalEmail := user.Email
	originalUsername := user.Username

	// Update fields if provided
	// Handle email field:
	// - nil means field was not provided in JSON, don't change email
	// - pointer to empty string means set to NULL (remove email)
	// - pointer to non-empty string means update to that value
	if req.Email != nil {
		if *req.Email == "" {
			// Empty string means set email to NULL
			user.Email = nil
		} else {
			// Validate email format for non-empty values
			if err := s.ValidateEmailFormat(*req.Email); err != nil {
				s.logger.Info("Invalid email format", "email", *req.Email, "error", err.Error())
				return nil, domain.NewValidationError(constants.MsgInvalidEmailFormatVN)
			}

			if user.Email == nil || *req.Email != *user.Email {
				// Check if email is already taken by another user
				if existingUser, err := s.UserRepo.GetByEmail(ctx, *req.Email); err == nil && existingUser.ID != id {
					s.logger.Info("Email already taken by another user", "email", *req.Email, "existing_user_id", existingUser.ID)
					return nil, domain.NewConflictError(constants.MsgEmailAlreadyTakenVN)
				}
				user.Email = req.Email
			}
		}
	}

	if req.Username != nil && *req.Username != "" && *req.Username != user.Username {
		// Check if username is already taken by another user
		if existingUser, err := s.UserRepo.GetByUsername(ctx, *req.Username); err == nil && existingUser.ID != id {
			s.logger.Info("Username already taken by another user", "username", *req.Username, "existing_user_id", existingUser.ID)
			return nil, domain.NewConflictError(constants.MsgUsernameAlreadyTakenVN)
		}
		user.Username = *req.Username
	}

	if req.Fullname != nil && *req.Fullname != "" {
		user.Fullname = utils.ToVietnameseTitleCase(*req.Fullname)
	}

	if req.Role != nil && *req.Role != "" {
		user.Role = domain.UserRole(*req.Role)
	}

	// Save updated user
	if err := s.UserRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user", "error", err, "user_id", id)
		return nil, err
	}

	// Publish domain event
	actorFullName := auditservice.GetActorFullName(ctx, s.UserRepo, user.ID)
	actorUserID := auditctx.GetUserIDOrZero(ctx)
	event := domain.NewUserUpdatedEvent(ctx, user, actorUserID, actorFullName, &originalUser)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish UserUpdatedEvent", "error", err, "user_id", user.ID)
	}

	response := dto.ToUserResponse(user)

	var originalEmailLog, newEmailLog string
	if originalEmail != nil {
		originalEmailLog = *originalEmail
	}
	if user.Email != nil {
		newEmailLog = *user.Email
	}
	s.logger.Info("User updated successfully",
		"user_id", id,
		"original_email", originalEmailLog,
		"new_email", newEmailLog,
		"original_username", originalUsername,
		"new_username", user.Username,
		"role", user.Role)

	return &response, nil
}

// DeleteUser removes a user from the system
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	s.logger.Info("Deleting user", "user_id", id)

	// Check if user exists first
	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Info("User not found for deletion", "user_id", id, "error", err)
		return err
	}

	// Perform deletion
	if err := s.UserRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete user", "error", err, "user_id", id)
		return err
	}

	// Publish domain event
	actorFullName := auditservice.GetActorFullName(ctx, s.UserRepo, user.ID)
	actorUserID := auditctx.GetUserIDOrZero(ctx)
	event := domain.NewUserDeletedEvent(ctx, user.ID, user.Username, user.Fullname, actorUserID, actorFullName)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish UserDeletedEvent", "error", err, "user_id", user.ID)
	}

	var logEmail string
	if user.Email != nil {
		logEmail = *user.Email
	}
	s.logger.Info("User deleted successfully",
		"user_id", id,
		"email", logEmail,
		"username", user.Username)

	return nil
}

// RestoreUser restores a soft-deleted user
func (s *UserService) RestoreUser(ctx context.Context, id uint) error {
	s.logger.Info("Restoring user", "user_id", id)

	// Perform restoration
	if err := s.UserRepo.Restore(ctx, id); err != nil {
		s.logger.Error("Failed to restore user", "error", err, "user_id", id)
		return err
	}

	s.logger.Info("User restored successfully", "user_id", id)
	return nil
}

// ListUsers returns a paginated list of users with filtering support
func (s *UserService) ListUsers(ctx context.Context, req dto.ListUsersRequest) ([]dto.UserResponse, int64, error) {
	s.logger.Info("Listing users", "page", req.Page, "page_size", req.PageSize, "role", req.Role, "search", req.Search, "ids", req.IDs)

	// Parse role filter if provided
	var roleFilter *domain.UserRole
	if req.Role != "" {
		role := domain.UserRole(req.Role)
		roleFilter = &role
	}

	// Parse IDs filter if provided
	var ids []uint
	if req.IDs != "" {
		ids = s.parseIDs(req.IDs)
	}

	// When fetching by IDs, skip pagination — return all matching users directly
	pageSize := req.PageSize
	offset := (req.Page - 1) * req.PageSize
	if len(ids) > 0 {
		pageSize = len(ids)
		offset = 0
	}

	// Get users from repository with filters
	users, total, err := s.UserRepo.ListWithFilters(ctx, roleFilter, req.Search, ids, req.LastLoginToday, req.SortBy, req.SortOrder, pageSize, offset)
	if err != nil {
		s.logger.Error("Failed to list users", "error", err, "page_size", pageSize, "offset", offset)
		return nil, 0, err
	}

	userResponses := dto.ToUserResponseList(users)

	s.logger.Info("Users listed successfully",
		"returned_count", len(users),
		"total_count", total,
		"page", req.Page,
		"page_size", pageSize,
		"role_filter", req.Role,
		"search_term", req.Search,
		"ids_filter", req.IDs)

	return userResponses, total, nil
}

// ResetUserPassword resets a user's password (Admin only operation)
func (s *UserService) ResetUserPassword(ctx context.Context, userID uint, newPassword string) error {
	s.logger.Info("Admin resetting user password", "user_id", userID)

	// Validate new password
	if err := s.passwordValidator.Validate(newPassword); err != nil {
		s.logger.Info("New password validation failed", "user_id", userID, "error", err.Error())
		return domain.NewValidationError(err.Error())
	}

	// Get user
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Info("User not found for password reset", "user_id", userID, "error", err)
		return err
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		s.logger.Error("Failed to hash new password", "error", err, "user_id", userID)
		return domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, err)
	}

	// Update user password
	user.Password = hashedPassword
	err = s.UserRepo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user password", "error", err, "user_id", userID)
		return domain.NewInternalError(constants.MsgFailedToUpdateUserPasswordVN, err)
	}

	// Publish domain event
	actorFullName := auditservice.GetActorFullName(ctx, s.UserRepo, userID)
	actorUserID := auditctx.GetUserIDOrZero(ctx)
	event := domain.NewPasswordChangedEvent(ctx, userID, user.Username, "admin_reset", actorUserID, actorFullName)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish PasswordChangedEvent", "error", err, "user_id", userID)
	}

	var logEmail string
	if user.Email != nil {
		logEmail = *user.Email
	}
	s.logger.Info("User password reset successfully by admin",
		"user_id", userID,
		"username", user.Username,
		"email", logEmail)

	return nil
}

// ValidateEmailFormat validates email format using Go's net/mail package
func (s *UserService) ValidateEmailFormat(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}

// parseIDs parses a comma-separated string of IDs into a slice of uints
// Example: "1,2,3" -> []uint{1, 2, 3}
// Invalid IDs are silently skipped
func (s *UserService) parseIDs(idsStr string) []uint {
	if idsStr == "" {
		return nil
	}

	parts := strings.Split(idsStr, ",")
	ids := make([]uint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			s.logger.Warn("Invalid ID in IDs filter, skipping", "id", part, "error", err)
			continue
		}

		ids = append(ids, uint(id))
	}

	return ids
}

// getUnderlyingError extracts the underlying error from domain errors
func getUnderlyingError(err error) string {
	if err == nil {
		return ""
	}
	// domain.InternalError has an underlying error that we want to see
	type internalError interface {
		Unwrap() error
	}
	if unwrappable, ok := err.(internalError); ok {
		if unwrapped := unwrappable.Unwrap(); unwrapped != nil {
			return unwrapped.Error()
		}
	}
	return err.Error()
}

// ResetFirstTimeLoginPasswords resets passwords for all users who have never
// logged in (last_login IS NULL). Processes users in pages (default 500/page)
// so memory stays bounded regardless of how many never-logged-in users exist.
func (s *UserService) ResetFirstTimeLoginPasswords(ctx context.Context, defaultPassword string) (*dto.ResetFirstTimeLoginPasswordResponse, error) {
	s.logger.Info("Admin resetting passwords for first-time users (last_login IS NULL)")

	// Extended-timeout context: the HTTP context may expire, but the bulk job
	// needs enough time to page through every qualifying user.
	bulkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	const pageSize = 500
	totalFound, err := s.UserRepo.CountUsersWithNullLastLogin(bulkCtx)
	if err != nil {
		s.logger.Error("Failed to count users with null last_login", "error", err)
		return nil, err
	}
	s.logger.Info("Users with null last_login", "count", totalFound)

	affectedCount := 0
	failedCount := 0
	updatedUsernames := make([]string, 0, totalFound)

	// Iterate in pages. We re-query each page rather than holding a single
	// cursor, which is fine because the writes (Update password) mutate a column
	// that is NOT the last_login filter — the WHERE last_login IS NULL set is
	// stable across pages for the lifetime of this job.
	for offset := 0; ; offset += pageSize {
		if bulkCtx.Err() != nil {
			s.logger.Warn("Bulk operation context cancelled", "error", bulkCtx.Err(), "affected_count", affectedCount, "processed", affectedCount+failedCount)
			break
		}

		users, err := s.UserRepo.FindUsersWithNullLastLogin(bulkCtx, pageSize, offset)
		if err != nil {
			s.logger.Error("Failed to fetch page of users with null last_login", "error", err, "offset", offset)
			break
		}
		if len(users) == 0 {
			break // exhausted
		}

		for _, user := range users {
			if bulkCtx.Err() != nil {
				break
			}
			hashedPassword, err := s.hashPassword(defaultPassword)
			if err != nil {
				s.logger.Error("Failed to hash password for user", "error", err, "user_id", user.ID, "username", user.Username)
				failedCount++
				continue
			}
			user.Password = hashedPassword
			if err := s.UserRepo.Update(bulkCtx, user); err != nil {
				s.logger.Error("Failed to update user password", "error", err, "user_id", user.ID, "username", user.Username, "underlying_error", getUnderlyingError(err))
				failedCount++
				continue
			}
			affectedCount++
			updatedUsernames = append(updatedUsernames, user.Username)
		}

		if len(users) < pageSize {
			break // last page
		}
	}

	s.logger.Info("Completed resetting first-time login passwords", "affected_count", affectedCount, "failed_count", failedCount, "total_found", totalFound)

	return &dto.ResetFirstTimeLoginPasswordResponse{
		Total:     affectedCount,
		Usernames: updatedUsernames,
	}, nil
}
