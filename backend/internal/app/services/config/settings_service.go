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
)

// Cache TTL constants for settings
const (
	SettingsListCacheTTL   = 1 * time.Hour // Settings list cache
	SettingsDetailCacheTTL = 2 * time.Hour // Individual settings details
)

type SettingsService struct {
	logger       *slog.Logger
	SettingsRepo domain.SettingsRepository
	QuotaRepo    domain.AdvancePaymentRepository
	TxManager    domain.TransactionManager
	CacheService *infrastructure.CacheService
	EventBus     domain.EventBus
}

func NewSettingsService(
	settingsRepo domain.SettingsRepository,
	quotaRepo domain.AdvancePaymentRepository,
	txManager domain.TransactionManager,
	cacheService *infrastructure.CacheService,
	eventBus domain.EventBus,
) *SettingsService {
	return &SettingsService{
		logger:       observability.GetLogger(),
		SettingsRepo: settingsRepo,
		QuotaRepo:    quotaRepo,
		TxManager:    txManager,
		CacheService: cacheService,
		EventBus:     eventBus,
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

	return settings, nil
}

func (s *SettingsService) GetSetting(ctx context.Context, id uint) (*domain.Settings, error) {
	return s.SettingsRepo.GetByID(ctx, id)
}

func (s *SettingsService) GetSettingByKey(ctx context.Context, key string) (*domain.Settings, error) {
	if s.CacheService == nil {
		return s.SettingsRepo.GetByKey(ctx, key)
	}

	// Generate cache key for setting detail
	cacheKey := s.CacheService.GenerateSettingsCacheKey("detail", fmt.Sprintf("key:%s", strings.ToLower(key)))

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
	err := s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		existingSetting, err := s.SettingsRepo.GetByIDForUpdate(txCtx, settingID)
		if err != nil {
			return err
		}

		if key, ok := updateData["key"].(string); ok {
			existingSetting.Key = key
		}
		if value, ok := updateData["value"].(string); ok {
			existingSetting.Value = &value
		} else if updateData["value"] == nil {
			existingSetting.Value = nil
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

		updatedSetting = existingSetting
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.EventBus.Publish(ctx, domain.NewSettingsUpdatedEvent(ctx, updatedSetting.Key, updatedSetting.ID)); err != nil {
		s.logger.Warn("Failed to publish settings updated event", "settingsID", updatedSetting.ID, "key", updatedSetting.Key, "error", err)
	}
	s.invalidateSettingsCache(ctx)
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

	// Store in cache
	result := cachedResult{Settings: settings, Count: count}
	if cacheErr := s.CacheService.Set(ctx, cacheKey, result, SettingsListCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache settings list", "error", cacheErr)
	}

	return settings, count, nil
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
