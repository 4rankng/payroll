package ports

import (
	"context"

	"api-server/internal/domain"
)

// UserPort defines the interface for user operations
// This prevents direct dependency on user.Service
type UserPort interface {
	GetUserByID(ctx context.Context, id uint) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error
}
