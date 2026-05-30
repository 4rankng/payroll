package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"strings"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

var allowedAuditSortFields = map[string]bool{
	"created_at": true, "action": true, "entity_type": true,
	"user_id": true, "id": true, "ip_address": true,
}

var allowedBlacklistSortFields = map[string]bool{
	"blacklisted_at": true, "user_id": true, "expires_at": true, "id": true,
}

var allowedSortOrders = map[string]bool{"ASC": true, "DESC": true}

// BlacklistedTokenRepository
type BlacklistedTokenRepository struct {
	*BaseRepository
}

func NewBlacklistedTokenRepository(db *Database) domain.BlacklistedTokenRepository {
	return &BlacklistedTokenRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *BlacklistedTokenRepository) Create(ctx context.Context, token *domain.BlacklistedToken) error {
	return r.DB.WithContext(ctx).Create(token).Error
}

func (r *BlacklistedTokenRepository) GetByJTI(ctx context.Context, jti string) (*domain.BlacklistedToken, error) {
	var token domain.BlacklistedToken
	err := r.DB.WithContext(ctx).
		Select("blacklisted_tokens.*, users.fullname as user_fullname").
		Joins("LEFT JOIN users ON blacklisted_tokens.user_id = users.id AND users.deleted_at IS NULL").
		Where("blacklisted_tokens.token_jti = ?", jti).
		First(&token).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("blacklisted token not found")
		}
		return nil, err
	}

	return &token, nil
}

func (r *BlacklistedTokenRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	var exists bool
	err := r.DB.WithContext(ctx).
		Model(&domain.BlacklistedToken{}).
		Select("COUNT(*) > 0").
		Where("token_jti = ? AND expires_at > ?", jti, clock.Now()).
		Scan(&exists).Error

	return exists, err
}

func (r *BlacklistedTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.DB.WithContext(ctx).
		Where("expires_at <= ?", clock.Now()).
		Delete(&domain.BlacklistedToken{}).Error
}

func (r *BlacklistedTokenRepository) List(ctx context.Context, filters domain.BlacklistFilters) ([]*domain.BlacklistedToken, error) {
	var tokens []*domain.BlacklistedToken
	query := r.DB.WithContext(ctx).
		Select("blacklisted_tokens.*, users.fullname as user_fullname").
		Joins("LEFT JOIN users ON blacklisted_tokens.user_id = users.id AND users.deleted_at IS NULL")

	query = r.applyBlacklistFilters(query, filters)

	sortBy := "blacklisted_tokens.blacklisted_at"
	if filters.SortBy != "" && allowedBlacklistSortFields[filters.SortBy] {
		sortBy = "blacklisted_tokens." + filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		if upper := strings.ToUpper(filters.SortOrder); allowedSortOrders[upper] {
			sortOrder = upper
		}
	}

	query = query.Order(sortBy + " " + sortOrder)

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&tokens).Error
	return tokens, err
}

func (r *BlacklistedTokenRepository) Count(ctx context.Context, filters domain.BlacklistFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.BlacklistedToken{})
	query = r.applyBlacklistFilters(query, filters)
	err := query.Count(&count).Error
	return count, err
}

func (r *BlacklistedTokenRepository) GetByUser(ctx context.Context, userID uint) ([]*domain.BlacklistedToken, error) {
	var tokens []*domain.BlacklistedToken
	err := r.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("blacklisted_at DESC").
		Find(&tokens).Error

	return tokens, err
}

func (r *BlacklistedTokenRepository) BlacklistToken(ctx context.Context, jti string, userID uint, expiresAt time.Time, reason domain.BlacklistReason) error {
	token := &domain.BlacklistedToken{
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Reason:    reason,
	}

	return r.DB.WithContext(ctx).Create(token).Error
}

func (r *BlacklistedTokenRepository) applyBlacklistFilters(query *gorm.DB, filters domain.BlacklistFilters) *gorm.DB {
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}

	if len(filters.Reason) > 0 {
		query = query.Where("reason IN ?", filters.Reason)
	}

	if filters.FromDate != nil {
		query = query.Where("blacklisted_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("blacklisted_at <= ?", *filters.ToDate)
	}

	if filters.Expired != nil {
		if *filters.Expired {
			query = query.Where("expires_at <= ?", clock.Now())
		} else {
			query = query.Where("expires_at > ?", clock.Now())
		}
	}

	return query
}

// AuditLogRepository
type AuditLogRepository struct {
	*BaseRepository
}

func NewAuditLogRepository(db *Database) domain.AuditLogRepository {
	return &AuditLogRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.DB.WithContext(ctx).Create(log).Error
}

func (r *AuditLogRepository) GetByID(ctx context.Context, id uint) (*domain.AuditLog, error) {
	var log domain.AuditLog
	err := r.DB.WithContext(ctx).
		Select("audit_logs.*, users.fullname as user_fullname, users.username as user_username, users.role as user_role").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id AND users.deleted_at IS NULL").
		Where("audit_logs.id = ?", id).
		First(&log).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("audit log not found")
		}
		return nil, err
	}

	return &log, nil
}

func (r *AuditLogRepository) List(ctx context.Context, filters domain.AuditFilters) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.DB.WithContext(ctx).
		Select("audit_logs.*, users.fullname as user_fullname, users.username as user_username, users.role as user_role").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id AND users.deleted_at IS NULL")

	query = r.applyAuditFilters(query, filters)

	sortBy := "audit_logs.created_at"
	if filters.SortBy != "" && allowedAuditSortFields[filters.SortBy] {
		sortBy = "audit_logs." + filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		if upper := strings.ToUpper(filters.SortOrder); allowedSortOrders[upper] {
			sortOrder = upper
		}
	}

	query = query.Order(sortBy + " " + sortOrder)

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

func (r *AuditLogRepository) Count(ctx context.Context, filters domain.AuditFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.AuditLog{})
	query = r.applyAuditFilters(query, filters)
	err := query.Count(&count).Error
	return count, err
}

func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) error {
	return r.DB.WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&domain.AuditLog{}).Error
}

func (r *AuditLogRepository) applyAuditFilters(query *gorm.DB, filters domain.AuditFilters) *gorm.DB {
	if filters.UserID != nil {
		query = query.Where("audit_logs.user_id = ?", *filters.UserID)
	}
	if len(filters.Action) > 0 {
		query = query.Where("audit_logs.action IN ?", filters.Action)
	}
	if len(filters.EntityType) > 0 {
		query = query.Where("audit_logs.entity_type IN ?", filters.EntityType)
	}
	if filters.EntityID != nil {
		query = query.Where("audit_logs.entity_id = ?", *filters.EntityID)
	}
	if filters.IPAddress != nil {
		query = query.Where("audit_logs.ip_address = ?", *filters.IPAddress)
	}
	if filters.FromDate != nil {
		query = query.Where("audit_logs.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("audit_logs.created_at <= ?", *filters.ToDate)
	}
	return query
}
