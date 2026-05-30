package audit

import (
	"context"

	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

// Service provides centralized audit context management
type Service struct {
	userRepo domain.UserRepository
}

// NewService creates a new audit service
func NewService(userRepo domain.UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

// GetActorFullName gets the actor's full name from context, with database fallback
func (s *Service) GetActorFullName(ctx context.Context, userID uint) string {
	// First try to get from context (from JWT)
	if fullName := auditctx.GetFullName(ctx); fullName != "" {
		return fullName
	}

	// Fallback to database lookup
	if s.userRepo != nil {
		if user, err := s.userRepo.GetByID(ctx, userID); err == nil {
			return user.Fullname
		}
	}

	// Final fallback
	return "Unknown"
}

// EnrichAuditContext adds user full name to context from JWT claims or database
func (s *Service) EnrichAuditContext(ctx context.Context, userID uint) context.Context {
	if fullName := auditctx.GetFullName(ctx); fullName == "" {
		// Only perform database lookup if not already in context
		if s.userRepo != nil {
			if user, err := s.userRepo.GetByID(ctx, userID); err == nil {
				ctx = auditctx.WithFullName(ctx, user.Fullname)
			}
		}
	}
	return ctx
}
