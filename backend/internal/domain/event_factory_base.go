package domain

import (
	"context"

	"api-server/internal/pkg/clock"
	auditctx "api-server/internal/pkg/context"
)

// newBaseEvent creates a base event with context information, action, and entity type.
// IP address and user agent are extracted from context automatically.
func newBaseEvent(ctx context.Context, eventName string, entityID uint,
	action AuditAction, entityType EntityType) BaseEvent {
	userID := getUserIDFromContext(ctx)

	return BaseEvent{
		EventName:    eventName,
		Timestamp:    clock.Now(),
		EntityID:     entityID,
		ActorUserID:  userID,
		AuditMessage: "",
		Action:       action,
		EntityType:   entityType,
		IPAddress:    auditctx.GetIPAddress(ctx),
		UserAgent:    auditctx.GetUserAgent(ctx),
	}
}

// newBaseEventWithAudit creates a base event with context information, action, entity type, and audit message.
func newBaseEventWithAudit(ctx context.Context, eventName string, entityID uint,
	action AuditAction, entityType EntityType, auditMessage string) BaseEvent {
	userID := getUserIDFromContext(ctx)

	return BaseEvent{
		EventName:    eventName,
		Timestamp:    clock.Now(),
		EntityID:     entityID,
		ActorUserID:  userID,
		AuditMessage: auditMessage,
		Action:       action,
		EntityType:   entityType,
		IPAddress:    auditctx.GetIPAddress(ctx),
		UserAgent:    auditctx.GetUserAgent(ctx),
	}
}

// newBaseEventWithActor creates a base event with an explicit actor user ID, action, and entity type.
// Use this instead of newBaseEvent/newBaseEventWithAudit when the actor ID is known
// but may not be present in context (e.g. background goroutines, bulk imports).
func newBaseEventWithActor(ctx context.Context, eventName string, entityID uint,
	actorUserID uint, action AuditAction, entityType EntityType, auditMessage string) BaseEvent {
	// Prefer context user ID if available, fall back to explicit actorUserID
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		userID = actorUserID
	}

	return BaseEvent{
		EventName:    eventName,
		Timestamp:    clock.Now(),
		EntityID:     entityID,
		ActorUserID:  userID,
		AuditMessage: auditMessage,
		Action:       action,
		EntityType:   entityType,
		IPAddress:    auditctx.GetIPAddress(ctx),
		UserAgent:    auditctx.GetUserAgent(ctx),
	}
}

// getUserIDFromContext extracts the user ID from the context
func getUserIDFromContext(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}

	if userIDPtr := auditctx.GetUserID(ctx); userIDPtr != nil {
		return *userIDPtr
	}

	// Try different context keys
	if userID, ok := ctx.Value("user_id").(uint); ok {
		return userID
	}

	if userID, ok := ctx.Value("user_id").(int); ok && userID > 0 {
		return uint(userID)
	}

	return 0
}
