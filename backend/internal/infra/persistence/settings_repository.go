package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type SettingsRepository struct {
	*BaseRepository
}

func NewSettingsRepository(db *Database) domain.SettingsRepository {
	return &SettingsRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *SettingsRepository) Create(ctx context.Context, settings *domain.Settings) error {
	return r.DB.WithContext(ctx).Create(settings).Error
}

func (r *SettingsRepository) GetByID(ctx context.Context, id uint) (*domain.Settings, error) {
	var settings domain.Settings
	err := r.DB.WithContext(ctx).
		First(&settings, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return &settings, nil
}

func (r *SettingsRepository) GetByKey(ctx context.Context, key string) (*domain.Settings, error) {
	var settings domain.Settings
	err := r.DB.WithContext(ctx).
		Where("`key` = ?", key).
		First(&settings).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return &settings, nil
}

func (r *SettingsRepository) Update(ctx context.Context, settings *domain.Settings) error {
	return r.DB.WithContext(ctx).Save(settings).Error
}

func (r *SettingsRepository) Delete(ctx context.Context, id uint) error {
	// Use GORM soft delete
	return r.DB.WithContext(ctx).Delete(&domain.Settings{}, id).Error
}

func (r *SettingsRepository) List(ctx context.Context, filters domain.SettingsFilters) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	query := r.DB.WithContext(ctx)

	// Apply filters
	query = r.applyFilters(query, filters)

	// Apply sorting
	sortBy := "updated_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&settings).Error
	return settings, err
}

func (r *SettingsRepository) Count(ctx context.Context, filters domain.SettingsFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.Settings{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *SettingsRepository) GetActiveSettings(ctx context.Context) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	err := r.DB.WithContext(ctx).
		Order("`key` ASC").
		Find(&settings).Error

	return settings, err
}

// applyFilters applies the common filters to the query
func (r *SettingsRepository) applyFilters(query *gorm.DB, filters domain.SettingsFilters) *gorm.DB {
	// Filter by value type
	if len(filters.ValueType) > 0 {
		query = query.Where("value_type IN ?", filters.ValueType)
	}

	// Search functionality
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		query = query.Where("LOWER(`key`) LIKE LOWER(?)", searchTerm)
	}

	return query
}
