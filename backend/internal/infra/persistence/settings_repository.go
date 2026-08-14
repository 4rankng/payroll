package persistence

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsRepository struct {
	*BaseRepository
}

func NewSettingsRepository(db *Database) domain.SettingsRepository {
	return &SettingsRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *SettingsRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func (r *SettingsRepository) Create(ctx context.Context, settings *domain.Settings) error {
	return r.getDB(ctx).Create(settings).Error
}

func (r *SettingsRepository) GetByID(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.getByID(ctx, id, false)
}

func (r *SettingsRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.getByID(ctx, id, true)
}

func (r *SettingsRepository) getByID(ctx context.Context, id uint, forUpdate bool) (*domain.Settings, error) {
	var settings domain.Settings
	query := r.getDB(ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&settings, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return &settings, nil
}

func (r *SettingsRepository) GetByKey(ctx context.Context, key string) (*domain.Settings, error) {
	return r.getByKey(ctx, key, false)
}

func (r *SettingsRepository) GetByKeyForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return r.getByKey(ctx, key, true)
}

func (r *SettingsRepository) getByKey(ctx context.Context, key string, forUpdate bool) (*domain.Settings, error) {
	var settings domain.Settings
	query := r.getDB(ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Where("`key` = ?", key).First(&settings).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return &settings, nil
}

func (r *SettingsRepository) Update(ctx context.Context, settings *domain.Settings) error {
	return r.getDB(ctx).Save(settings).Error
}

// CompareAndSwapValue updates a setting only when its JSON value still equals
// the value read by the caller. Zalo credentials use this to prevent a stale
// status/error writer from restoring a refresh token that another request has
// already consumed and replaced.
func (r *SettingsRepository) CompareAndSwapValue(
	ctx context.Context,
	key string,
	currentValue string,
	nextValue string,
	valueType domain.SettingsValueType,
) (bool, error) {
	db := r.getDB(ctx)
	query := db.Model(&domain.Settings{}).Where("`key` = ?", key)
	if currentValue == "" {
		query = query.Where("(`value` = ? OR `value` IS NULL)", currentValue)
	} else if db.Dialector.Name() == "mysql" {
		// Tokens are case-sensitive. MySQL TEXT equality otherwise inherits the
		// database collation, which is commonly case-insensitive and can let a
		// stale value differing only by case pass the CAS predicate.
		query = query.Where("BINARY `value` = BINARY ?", currentValue)
	} else {
		query = query.Where("`value` = ?", currentValue)
	}
	result := query.
		Updates(map[string]any{
			"value":      nextValue,
			"value_type": valueType,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *SettingsRepository) Delete(ctx context.Context, id uint) error {
	// Use GORM soft delete
	return r.getDB(ctx).Delete(&domain.Settings{}, id).Error
}

func (r *SettingsRepository) List(ctx context.Context, filters domain.SettingsFilters) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	query := r.getDB(ctx)

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

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "updated_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

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
	query := r.getDB(ctx).Model(&domain.Settings{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *SettingsRepository) GetActiveSettings(ctx context.Context) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	err := r.getDB(ctx).
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
