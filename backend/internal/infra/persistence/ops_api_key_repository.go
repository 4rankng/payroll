package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// OpsAPIKeyRepository persists machine API keys for the payroll-ops-mcp
// surface (pk_ops_*).
type OpsAPIKeyRepository struct {
	DB *Database
}

func NewOpsAPIKeyRepository(db *Database) *OpsAPIKeyRepository {
	return &OpsAPIKeyRepository{DB: db}
}

func (r *OpsAPIKeyRepository) Create(ctx context.Context, key *domain.OpsAPIKey) error {
	return r.DB.WithContext(ctx).Create(key).Error
}

// GetByHash resolves a key by its SHA-256 hex hash. A missing row is a
// not-found error so the authenticating service can map it to "invalid key".
func (r *OpsAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.OpsAPIKey, error) {
	var key domain.OpsAPIKey
	if err := r.DB.WithContext(ctx).Where("key_hash = ?", keyHash).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("ops api key not found")
		}
		return nil, err
	}
	return &key, nil
}

func (r *OpsAPIKeyRepository) GetByID(ctx context.Context, id uint) (*domain.OpsAPIKey, error) {
	var key domain.OpsAPIKey
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("ops api key not found")
		}
		return nil, err
	}
	return &key, nil
}

// List returns every key, newest first (revoked keys included — the admin
// list is the audit record).
func (r *OpsAPIKeyRepository) List(ctx context.Context) ([]*domain.OpsAPIKey, error) {
	var keys []*domain.OpsAPIKey
	err := r.DB.WithContext(ctx).
		Model(&domain.OpsAPIKey{}).
		Order("id DESC").
		Find(&keys).Error
	return keys, err
}

// Revoke sets revoked_at on a live key. An unknown or already-revoked id
// affects zero rows and is reported as not-found.
func (r *OpsAPIKeyRepository) Revoke(ctx context.Context, id uint, revokedAt time.Time) error {
	res := r.DB.WithContext(ctx).
		Model(&domain.OpsAPIKey{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.NewNotFoundError("ops api key not found")
	}
	return nil
}

func (r *OpsAPIKeyRepository) UpdateLastUsedAt(ctx context.Context, id uint, at time.Time) error {
	return r.DB.WithContext(ctx).
		Model(&domain.OpsAPIKey{}).
		Where("id = ?", id).
		Update("last_used_at", at).Error
}

// UpdateScope persists an admin scope decision. read_write carries a
// write-window bound; read_only clears it.
func (r *OpsAPIKeyRepository) UpdateScope(ctx context.Context, id uint, scope domain.OpsAPIKeyScope, writeWindowExpiresAt *time.Time) error {
	res := r.DB.WithContext(ctx).
		Model(&domain.OpsAPIKey{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{
			"scope":                   scope,
			"write_window_expires_at": writeWindowExpiresAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.NewNotFoundError("ops api key not found")
	}
	return nil
}

// RevertExpiredWriteWindows converges every lapsed read_write key back to
// read_only. Called best-effort from the authentication path so the stored
// state matches the evaluated truth without needing a cron.
func (r *OpsAPIKeyRepository) RevertExpiredWriteWindows(ctx context.Context, now time.Time) (int64, error) {
	res := r.DB.WithContext(ctx).
		Model(&domain.OpsAPIKey{}).
		Where("scope = ? AND write_window_expires_at IS NOT NULL AND write_window_expires_at < ? AND revoked_at IS NULL", domain.OpsAPIKeyScopeReadWrite, now).
		Updates(map[string]any{
			"scope":                   domain.OpsAPIKeyScopeReadOnly,
			"write_window_expires_at": nil,
		})
	return res.RowsAffected, res.Error
}
