package domain

import (
	"context"
)

// NewUserCreatedEvent creates a UserCreatedEvent
func NewUserCreatedEvent(ctx context.Context, user *User, actorUserID uint, actorFullName string) UserCreatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeUser,
		actorFullName,
		user.Username,
	)

	return UserCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "UserCreated", user.ID, actorUserID, AuditActionCreate, EntityTypeUser, auditMessage),
		Username:  user.Username,
		Fullname:  user.Fullname,
		Email:     user.Email,
		Role:      user.Role,
	}
}

// NewUserUpdatedEvent creates a UserUpdatedEvent
func NewUserUpdatedEvent(ctx context.Context, user *User, actorUserID uint, actorFullName string, original *User) UserUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeUser,
		actorFullName,
		user.Username,
	)

	return UserUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "UserUpdated", user.ID, actorUserID, AuditActionUpdate, EntityTypeUser, auditMessage),
		Username:      user.Username,
		Fullname:      user.Fullname,
		Email:         user.Email,
		Role:          user.Role,
		ChangedFields: CompareUsers(original, user),
	}
}

// NewUserDeletedEvent creates a UserDeletedEvent
func NewUserDeletedEvent(ctx context.Context, userID uint, username, fullname string, actorUserID uint, actorFullName string) UserDeletedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeUser,
		actorFullName,
		username,
	)

	return UserDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "UserDeleted", userID, actorUserID, AuditActionDelete, EntityTypeUser, auditMessage),
		Username:  username,
		Fullname:  fullname,
	}
}

// NewPasswordChangedEvent creates a PasswordChangedEvent
func NewPasswordChangedEvent(ctx context.Context, userID uint, targetName, method string, actorUserID uint, actorFullName string) PasswordChangedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionChangePassword,
		EntityTypeUser,
		actorFullName,
		targetName,
		method,
	)

	return PasswordChangedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "PasswordChanged", userID, actorUserID, AuditActionChangePassword, EntityTypeUser, auditMessage),
		Method:    method,
	}
}

// NewUserLoginEvent creates a UserLoginEvent
func NewUserLoginEvent(ctx context.Context, userID uint, username, userFullName, ipAddress, userAgent, reason, attemptedIdentifier string) UserLoginEvent {
	success := reason == ""

	// Use full name for audit message, fallback to username if full name is empty
	actorName := userFullName
	if actorName == "" {
		actorName = username
	}

	// Build proper audit message using template system
	var loginMessage string
	if reason != "" {
		loginMessage = "đăng nhập thất bại"
	} else {
		// Use template matrix for successful login
		loginMessage = BuildAuditMessageFromMatrix(AuditActionLogin, EntityTypeUser, actorName)
	}

	base := newBaseEventWithAudit(ctx, "UserLogin", userID, AuditActionLogin, EntityTypeUser, loginMessage)
	// Login events are published from a background goroutine with context.Background(),
	// so getUserIDFromContext returns 0. Override ActorUserID with the actual user ID.
	// Requirement 8.3: ActorUserID must be set to the authenticating user's ID even before JWT is issued.
	base.ActorUserID = userID

	// Requirement 4.6: Propagate IPAddress and UserAgent from context.
	// If context does not carry them (e.g. context.Background()), fall back to the
	// explicit parameters so the event always carries the request's IP and UA.
	if base.IPAddress == "" && ipAddress != "" {
		base.IPAddress = ipAddress
	}
	if base.UserAgent == "" && userAgent != "" {
		base.UserAgent = userAgent
	}

	return UserLoginEvent{
		BaseEvent:           base,
		Username:            username,
		IPAddress:           ipAddress,
		UserAgent:           userAgent,
		Success:             success,
		Reason:              reason,
		AttemptedIdentifier: attemptedIdentifier,
	}
}

// NewUserLogoutEvent creates a UserLogoutEvent
func NewUserLogoutEvent(ctx context.Context, userID uint, username, userFullName string) UserLogoutEvent {
	// Use full name for audit message, fallback to username if full name is empty
	actorName := userFullName
	if actorName == "" {
		actorName = username
	}

	// Use template matrix for logout message
	logoutMessage := BuildAuditMessageFromMatrix(AuditActionLogout, EntityTypeUser, actorName)

	base := newBaseEventWithAudit(ctx, "UserLogout", userID, AuditActionLogout, EntityTypeUser, logoutMessage)
	// Ensure ActorUserID is set to the actual user ID in case context lacks it
	if base.ActorUserID == 0 {
		base.ActorUserID = userID
	}

	return UserLogoutEvent{
		BaseEvent: base,
		Username:  username,
	}
}
