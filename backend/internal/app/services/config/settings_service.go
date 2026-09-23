package config

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/secret"
)

// Cache TTL constants for settings
const (
	SettingsListCacheTTL   = 1 * time.Hour // Settings list cache
	SettingsDetailCacheTTL = 2 * time.Hour // Individual settings details
)

// settingsCacheVersion namespaces cached settings payloads. It was bumped to v2
// when "zalo.credentials" started being sealed at rest: v1 payloads may hold the
// decrypted credential, and a versioned key makes them unreachable immediately
// instead of leaving them in Redis until the TTL expires. Cache invalidation
// deletes "settings:<operation>:*", so the version stays purgeable.
const settingsCacheVersion = "v2"

// settingsConfigCacheInvalidator clears the in-process read cache held by
// SettingsConfigService. Declared as an interface so SettingsService can
// invalidate it without depending on the concrete type (and so tests can
// substitute a fake).
type settingsConfigCacheInvalidator interface {
	InvalidateCache()
}

type SettingsService struct {
	logger                  *slog.Logger
	SettingsRepo            domain.SettingsRepository
	QuotaRepo               domain.AdvancePaymentRepository
	AttendanceRepo          domain.AttendanceRepository
	TxManager               domain.TransactionManager
	CacheService            *infrastructure.CacheService
	EventBus                domain.EventBus
	quotaCreditTaskEnqueuer interface {
		EnqueueCreditQuota(attendanceID uint, at time.Time) error
	}
	// configCacheInvalidator is bound at bootstrap to the SettingsConfigService
	// that reads business settings (payment percentages, beneficiary bank
	// details, ...). It is cleared after every committed mutation so a
	// completed Admin update is authoritative for the next read instead of
	// waiting out SettingsConfigService's short TTL — the same guarantee
	// money-moving values already get by bypassing that cache entirely.
	configCacheInvalidator settingsConfigCacheInvalidator
}

// SetConfigCacheInvalidator binds the SettingsConfigService read cache so
// settings mutations invalidate it. Optional: an unbound service simply skips
// the in-process invalidation and relies on the cache TTL.
func (s *SettingsService) SetConfigCacheInvalidator(invalidator settingsConfigCacheInvalidator) {
	s.configCacheInvalidator = invalidator
}

// invalidateConfigCache clears the in-process read cache after a committed
// mutation.
func (s *SettingsService) invalidateConfigCache() {
	if s.configCacheInvalidator != nil {
		s.configCacheInvalidator.InvalidateCache()
	}
}

func NewSettingsService(
	settingsRepo domain.SettingsRepository,
	quotaRepo domain.AdvancePaymentRepository,
	attendanceRepo domain.AttendanceRepository,
	txManager domain.TransactionManager,
	cacheService *infrastructure.CacheService,
	eventBus domain.EventBus,
	quotaCreditTaskEnqueuer interface {
		EnqueueCreditQuota(attendanceID uint, at time.Time) error
	},
) *SettingsService {
	return &SettingsService{
		logger:                  observability.GetLogger(),
		SettingsRepo:            settingsRepo,
		QuotaRepo:               quotaRepo,
		AttendanceRepo:          attendanceRepo,
		TxManager:               txManager,
		CacheService:            cacheService,
		EventBus:                eventBus,
		quotaCreditTaskEnqueuer: quotaCreditTaskEnqueuer,
	}
}

func (s *SettingsService) CreateSetting(ctx context.Context, settings *domain.Settings, createdBy uint) (*domain.Settings, error) {
	// Validate settings
	if err := settings.IsValid(); err != nil {
		return nil, err
	}
	if err := validateBusinessSetting(settings); err != nil {
		return nil, err
	}

	// Set default value type if not provided
	if settings.ValueType == "" {
		settings.ValueType = domain.ValueTypeString
	}

	if err := s.SettingsRepo.Create(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to create setting: %w", err)
	}

	// Publish event
	if err := s.EventBus.Publish(ctx, domain.NewSettingsCreatedEvent(ctx, settings.Key, settings.ID)); err != nil {
		s.logger.Warn("Failed to publish settings created event", "settingsID", settings.ID, "key", settings.Key, "error", err)
	}

	// Invalidate settings cache
	s.invalidateSettingsCache(ctx)
	s.invalidateConfigCache()

	return settings, nil
}

func (s *SettingsService) GetSetting(ctx context.Context, id uint) (*domain.Settings, error) {
	return s.SettingsRepo.GetByID(ctx, id)
}

func (s *SettingsService) GetSettingByKey(ctx context.Context, key string) (*domain.Settings, error) {
	// Protected values are never cached: the cache payload would hold the
	// decrypted credential, which is the very thing the repository seals at rest.
	if s.CacheService == nil || secret.IsProtectedKey(key) {
		return s.SettingsRepo.GetByKey(ctx, key)
	}

	// Generate cache key for setting detail
	cacheKey := s.CacheService.GenerateSettingsCacheKey("detail", settingsCacheVersion, fmt.Sprintf("key:%s", strings.ToLower(key)))

	// Try to get from cache first
	var cachedSetting *domain.Settings
	if err := s.CacheService.Get(ctx, cacheKey, &cachedSetting); err == nil && cachedSetting != nil {
		return cachedSetting, nil
	}

	// Cache miss - get from database
	setting, err := s.SettingsRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if cacheErr := s.CacheService.Set(ctx, cacheKey, setting, SettingsDetailCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache setting detail", "error", cacheErr)
	}

	return setting, nil
}

// GetSettingByKeyAuthoritative reads directly from the repository. Financial
// controls use this path so a successful Admin update cannot be masked by a
// stale cache entry or a cache-aside invalidation race.
func (s *SettingsService) GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error) {
	return s.SettingsRepo.GetByKey(ctx, key)
}

// GetSettingByKeyAuthoritativeForUpdate reads directly from the repository and
// acquires a row lock when called inside a transaction.
func (s *SettingsService) GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return s.SettingsRepo.GetByKeyForUpdate(ctx, key)
}

func (s *SettingsService) UpdateSetting(ctx context.Context, settingID uint, updateData map[string]any, updatedBy uint) (*domain.Settings, error) {
	var updatedSetting *domain.Settings
	var rescheduledQuotaCredits []domain.QuotaCreditSchedule
	err := s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existingSetting, err := s.SettingsRepo.GetByIDForUpdate(txCtx, settingID)
		if err != nil {
			return err
		}

		if key, ok := updateData["key"].(string); ok {
			existingSetting.Key = key
		}
		switch value := updateData["value"].(type) {
		case string:
			existingSetting.Value = &value
		case nil:
			// Null means "clear" for an ordinary setting. For a protected key it
			// arrives on every round-trip — the API never returns the stored
			// secret — so a client PUTting back what it just read would wipe the
			// credential. Null therefore means "unchanged" there: a secret is
			// replaced by supplying a new value, and removed by deleting the row.
			if !secret.IsProtectedKey(existingSetting.Key) {
				existingSetting.Value = nil
			}
		}
		if valueTypeStr, ok := updateData["value_type"].(string); ok {
			valueType := domain.SettingsValueType(valueTypeStr)
			existingSetting.ValueType = valueType
		}

		if err := existingSetting.IsValid(); err != nil {
			return err
		}
		if err := validateBusinessSetting(existingSetting); err != nil {
			return err
		}
		if err := s.SettingsRepo.Update(txCtx, existingSetting); err != nil {
			return fmt.Errorf("failed to update setting: %w", err)
		}
		if existingSetting.Key == SettingKeySelfCheckInAdvancePercent {
			percent, err := parseSelfCheckInAdvancePercent(existingSetting)
			if err != nil {
				return err
			}
			if err := s.QuotaRepo.RecomputeActiveCheckInMaxAdvance(txCtx, percent); err != nil {
				return fmt.Errorf("failed to recompute active self check-in quota: %w", err)
			}
		}
		if existingSetting.Key == SettingKeySelfCheckInAdvanceHold && s.AttendanceRepo != nil {
			holdHours, err := parseSelfCheckInAdvanceHoldHours(existingSetting)
			if err != nil {
				return err
			}
			schedules, err := s.AttendanceRepo.RecalculatePendingQuotaCreditSchedules(txCtx, time.Duration(holdHours)*time.Hour)
			if err != nil {
				return fmt.Errorf("failed to recalculate pending self check-in quota deadlines: %w", err)
			}
			rescheduledQuotaCredits = schedules
		}

		updatedSetting = existingSetting
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, schedule := range rescheduledQuotaCredits {
		if s.quotaCreditTaskEnqueuer == nil {
			break
		}
		if err := s.quotaCreditTaskEnqueuer.EnqueueCreditQuota(schedule.AttendanceID, schedule.EligibleAt); err != nil {
			s.logger.Warn("failed to enqueue recalculated quota credit task",
				"attendance_id", schedule.AttendanceID,
				"eligible_at", schedule.EligibleAt,
				"error", err)
		}
	}

	if err := s.EventBus.Publish(ctx, domain.NewSettingsUpdatedEvent(ctx, updatedSetting.Key, updatedSetting.ID)); err != nil {
		s.logger.Warn("Failed to publish settings updated event", "settingsID", updatedSetting.ID, "key", updatedSetting.Key, "error", err)
	}
	s.invalidateSettingsCache(ctx)
	s.invalidateConfigCache()
	return updatedSetting, nil
}

func validateBusinessSetting(setting *domain.Settings) error {
	if setting == nil {
		return nil
	}
	switch setting.Key {
	case SettingKeyBulkTransferWorkbookLimit:
		_, err := parseBulkTransferWorkbookLimit(setting)
		return err
	case SettingKeySelfCheckInAdvancePercent:
		_, err := parseSelfCheckInAdvancePercent(setting)
		return err
	case SettingKeySelfCheckInAdvanceHold:
		_, err := parseSelfCheckInAdvanceHoldHours(setting)
		return err
	default:
		return nil
	}
}

func (s *SettingsService) DeleteSetting(ctx context.Context, id uint, deletedBy uint) error {
	// Get setting details before deletion
	deletedSetting, err := s.SettingsRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get setting for deletion: %w", err)
	}

	if err := s.SettingsRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	// Publish event
	if err := s.EventBus.Publish(ctx, domain.NewSettingsDeletedEvent(ctx, deletedSetting.Key, id)); err != nil {
		s.logger.Warn("Failed to publish settings deleted event", "settingsID", id, "key", deletedSetting.Key, "error", err)
	}

	// Invalidate settings cache
	s.invalidateSettingsCache(ctx)
	s.invalidateConfigCache()

	return nil
}

func (s *SettingsService) ListSettings(ctx context.Context, filters domain.SettingsFilters) ([]*domain.Settings, int64, error) {
	if s.CacheService == nil {
		settings, err := s.SettingsRepo.List(ctx, filters)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list settings: %w", err)
		}
		count, err := s.SettingsRepo.Count(ctx, filters)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count settings: %w", err)
		}
		return settings, count, nil
	}

	// Generate cache key based on filters
	cacheKey := s.CacheService.GenerateSettingsCacheKey("list",
		settingsCacheVersion,
		fmt.Sprintf("limit:%d", filters.Limit),
		fmt.Sprintf("offset:%d", filters.Offset),
		fmt.Sprintf("search:%s", filters.Search),
		fmt.Sprintf("sortBy:%s", filters.SortBy),
		fmt.Sprintf("sortOrder:%s", filters.SortOrder))

	// Try to get from cache first
	type cachedResult struct {
		Settings []*domain.Settings `json:"settings"`
		Count    int64              `json:"count"`
	}

	var cached cachedResult
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		return cached.Settings, cached.Count, nil
	}

	// Cache miss - get from database
	settings, err := s.SettingsRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list settings: %w", err)
	}

	count, err := s.SettingsRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count settings: %w", err)
	}

	// Store in cache. A page containing a protected key is never cached — the
	// payload would carry the decrypted credential — which also filters out
	// searches such as "zalo" that intentionally ask for that row.
	if !containsProtectedKey(settings) {
		result := cachedResult{Settings: settings, Count: count}
		if cacheErr := s.CacheService.Set(ctx, cacheKey, result, SettingsListCacheTTL); cacheErr != nil {
			s.logger.Warn("Failed to cache settings list", "error", cacheErr)
		}
	}

	return settings, count, nil
}

// containsProtectedKey reports whether any row holds a sealed value.
func containsProtectedKey(settings []*domain.Settings) bool {
	for _, setting := range settings {
		if setting != nil && secret.IsProtectedKey(setting.Key) {
			return true
		}
	}
	return false
}

// invalidateSettingsCache clears all settings-related cache entries
func (s *SettingsService) invalidateSettingsCache(ctx context.Context) {
	if s.CacheService == nil {
		return
	}

	// Clear all settings list cache entries
	if err := s.CacheService.DeletePattern(ctx, "settings:list:*"); err != nil {
		s.logger.Warn("Failed to clear settings list cache", "error", err)
	}

	// Clear active settings cache
	if err := s.CacheService.DeletePattern(ctx, "settings:active:*"); err != nil {
		s.logger.Warn("Failed to clear active settings cache", "error", err)
	}

	// Clear individual settings detail cache entries
	if err := s.CacheService.DeletePattern(ctx, "settings:detail:*"); err != nil {
		s.logger.Warn("Failed to clear settings detail cache", "error", err)
	}
}
