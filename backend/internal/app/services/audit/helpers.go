package audit

import (
	"context"

	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

// GetActorFullName is a convenience function that gets actor full name
// This can be used by services that don't want to inject the audit service
func GetActorFullName(ctx context.Context, userRepo domain.UserRepository, userID uint) string {
	// First try to get from context (from JWT)
	if fullName := auditctx.GetFullName(ctx); fullName != "" {
		return fullName
	}

	// Fallback to database lookup
	if user, err := userRepo.GetByID(ctx, userID); err == nil {
		return user.Fullname
	}

	// Final fallback
	return "Unknown"
}

// WithAuditContext enriches context with audit information from JWT
// This should be called in authentication middleware
func WithAuditContext(ctx context.Context, userID uint, fullName string) context.Context {
	ctx = auditctx.WithUserID(ctx, userID)
	if fullName != "" {
		ctx = auditctx.WithFullName(ctx, fullName)
	}
	return ctx
}
