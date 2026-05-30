package audit

import (
	"context"

	"api-server/internal/domain"
)

// MessageBuilder builds audit messages with user names
type MessageBuilder struct {
	userRepo domain.UserRepository
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder(userRepo domain.UserRepository) *MessageBuilder {
	return &MessageBuilder{
		userRepo: userRepo,
	}
}

// BuildMessageWithUser builds an audit message using the template matrix with user's full name as actor
func (b *MessageBuilder) BuildMessageWithUser(ctx context.Context, userID uint, action domain.AuditAction, entityType domain.EntityType, entityName string) string {
	// Get user information (actor name)
	actorName := b.getUserName(ctx, userID)
	if actorName == "" {
		actorName = "Người dùng" // Fallback
	}

	// Use the template matrix which includes actor as first parameter
	return domain.BuildAuditMessageFromMatrix(action, entityType, actorName, entityName)
}

// BuildBulkMessageWithUser builds a bulk audit message using the template matrix with user's full name as actor
func (b *MessageBuilder) BuildBulkMessageWithUser(ctx context.Context, userID uint, action domain.AuditAction, entityType domain.EntityType, count int) string {
	// Get user information (actor name)
	actorName := b.getUserName(ctx, userID)
	if actorName == "" {
		actorName = "Người dùng" // Fallback
	}

	// Use the template matrix which includes actor as first parameter
	// For bulk operations, we need to provide count and potentially project name
	return domain.BuildAuditMessageFromMatrix(action, entityType, actorName, count, "dự án")
}

// getUserName fetches and formats user name from repository
func (b *MessageBuilder) getUserName(ctx context.Context, userID uint) string {
	if userID == 0 || b.userRepo == nil {
		return ""
	}

	user, err := b.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}

	// Prefer full name, fall back to username
	if user.Fullname != "" {
		return user.Fullname
	}
	return user.Username
}
