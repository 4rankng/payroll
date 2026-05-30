package user

import (
	"api-server/internal/constants"
	"context"

	auditservice "api-server/internal/app/services/audit"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

// VerifyPassword verifies user credentials and returns the user if valid
func (s *UserService) VerifyPassword(ctx context.Context, email, password string) (*domain.User, error) {
	s.logger.Info("Verifying user password", "email", email)

	// Get user by email
	user, err := s.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Info("User not found during password verification", "email", email)
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Check if user account is active
	if !user.IsActive() {
		s.logger.Info("Login attempt on inactive account", "email", email, "user_id", user.ID)
		return nil, domain.NewUnauthorizedError(constants.MsgAccountDisabledVN)
	}

	// Verify password
	if !s.VerifyPasswordHash(password, user.Password) {
		s.logger.Info("Invalid password attempt", "email", email, "user_id", user.ID)
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	s.logger.Info("Password verification successful", "email", email, "user_id", user.ID)
	return user, nil
}

// ChangePassword changes a user's password (requires current password verification)
func (s *UserService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword, ipAddress, userAgent string) error {
	s.logger.Info("Processing password change", "user_id", userID, "ip", ipAddress)

	// Validate new password
	if err := s.passwordValidator.Validate(newPassword); err != nil {
		s.logger.Info("New password validation failed during change", "user_id", userID, "error", err.Error())
		return domain.NewValidationError(err.Error())
	}

	// Get user
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Info("User not found during password change", "user_id", userID, "error", err)
		return domain.NewNotFoundError(constants.MsgUserNotFoundVN2)
	}

	// Verify current password
	if !s.VerifyPasswordHash(currentPassword, user.Password) {
		s.logger.Info("Incorrect current password during password change", "user_id", userID)
		return domain.NewUnauthorizedError(constants.MsgCurrentPasswordIncorrectVN)
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		s.logger.Error("Failed to hash new password during change", "error", err, "user_id", userID)
		return domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, err)
	}

	// Update user password
	user.Password = hashedPassword
	err = s.UserRepo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user password during change", "error", err, "user_id", userID)
		return domain.NewInternalError(constants.MsgFailedToUpdateUserPasswordVN, err)
	}

	// Publish domain event
	actorFullName := auditservice.GetActorFullName(ctx, s.UserRepo, userID)
	event := domain.NewPasswordChangedEvent(ctx, userID, user.Username, "self_service", auditctx.GetUserIDOrZero(ctx), actorFullName)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish PasswordChangedEvent", "error", err, "user_id", userID)
	}

	s.logger.Info("Password changed successfully",
		"user_id", userID,
		"email", user.Email,
		"username", user.Username)

	return nil
}
