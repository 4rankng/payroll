package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// MCPAPIKeyRepository persists machine API keys for the MCP agent surface.
type MCPAPIKeyRepository struct {
	DB *Database
}

func NewMCPAPIKeyRepository(db *Database) *MCPAPIKeyRepository {
	return &MCPAPIKeyRepository{DB: db}
}

func (r *MCPAPIKeyRepository) Create(ctx context.Context, key *domain.MCPAPIKey) error {
	return r.DB.WithContext(ctx).Create(key).Error
}

// GetByHash resolves a key by its SHA-256 hex hash. A missing row is a
// not-found error so the authenticating service can map it to "invalid key".
func (r *MCPAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.MCPAPIKey, error) {
	var key domain.MCPAPIKey
	if err := r.DB.WithContext(ctx).Where("key_hash = ?", keyHash).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("mcp api key not found")
		}
		return nil, err
	}
	return &key, nil
}

// List returns every key, newest first (revoked keys included — the admin list
// is the audit record).
func (r *MCPAPIKeyRepository) List(ctx context.Context) ([]*domain.MCPAPIKey, error) {
	var keys []*domain.MCPAPIKey
	err := r.DB.WithContext(ctx).
		Model(&domain.MCPAPIKey{}).
		Order("id DESC").
		Find(&keys).Error
	return keys, err
}

// Revoke sets revoked_at on a live key. An unknown or already-revoked id
// affects zero rows and is reported as not-found.
func (r *MCPAPIKeyRepository) Revoke(ctx context.Context, id uint, revokedAt time.Time) error {
	res := r.DB.WithContext(ctx).
		Model(&domain.MCPAPIKey{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.NewNotFoundError("mcp api key not found")
	}
	return nil
}

// UpdateLastUsedAt stamps last_used_at on a used key (best-effort telemetry).
func (r *MCPAPIKeyRepository) UpdateLastUsedAt(ctx context.Context, id uint, at time.Time) error {
	return r.DB.WithContext(ctx).
		Model(&domain.MCPAPIKey{}).
		Where("id = ?", id).
		Update("last_used_at", at).Error
}
