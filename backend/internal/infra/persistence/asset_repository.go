package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/timeutil"

	"gorm.io/gorm"
)

type AssetRepository struct {
	*BaseRepository
	eventBus domain.EventBus
}

func NewAssetRepository(db *Database, eventBus domain.EventBus) domain.AssetRepository {
	return &AssetRepository{
		BaseRepository: NewBaseRepository(db),
		eventBus:       eventBus,
	}
}

func (r *AssetRepository) Create(ctx context.Context, asset *domain.Asset) (*domain.Asset, error) {
	if err := r.DB.WithContext(ctx).Create(asset).Error; err != nil {
		return nil, domain.NewInternalError("failed to create asset", err)
	}

	// Emit audit event for asset creation (only for user-initiated actions)
	// System-created assets (uploaded_by = 0) are not audited
	if r.eventBus != nil && asset.UploadedBy > 0 {
		event := domain.NewAssetCreatedEvent(ctx, asset)
		// Publish in a non-blocking way
		go func() {
			if err := r.eventBus.Publish(context.Background(), event); err != nil {
				observability.GetLogger().Error("failed to publish AssetCreatedEvent", "error", err)
			}
		}()
	}

	return asset, nil
}

func (r *AssetRepository) GetByID(ctx context.Context, id uint) (*domain.Asset, error) {
	var asset domain.Asset
	if err := r.DB.WithContext(ctx).
		Preload("Uploader").
		First(&asset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("asset not found")
		}
		return nil, domain.NewInternalError("failed to get asset by ID", err)
	}

	return &asset, nil
}

func (r *AssetRepository) GetByChecksum(ctx context.Context, checksum string, uploadType string) (*domain.Asset, error) {
	var asset domain.Asset
	if err := r.DB.WithContext(ctx).
		Where("checksum = ? AND upload_type = ?", checksum, uploadType).
		First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("asset with checksum not found")
		}
		return nil, domain.NewInternalError("failed to get asset by checksum", err)
	}

	return &asset, nil
}

func (r *AssetRepository) CountByFilePath(ctx context.Context, filePath string) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).
		Model(&domain.Asset{}).
		Where("file_path = ?", filePath).
		Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count assets by file path", err)
	}

	return count, nil
}

func (r *AssetRepository) List(ctx context.Context, filters domain.AssetFilters) ([]*domain.Asset, error) {
	var assets []*domain.Asset

	query := r.DB.WithContext(ctx).Model(&domain.Asset{})
	query = applyAssetFilters(query, filters)

	// Default sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "desc"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(common.SanitizeSortColumn(sortBy, "created_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC"))

	// Default pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}

	if filters.Offset < 0 {
		filters.Offset = 0
	}

	query = query.Limit(filters.Limit).Offset(filters.Offset)

	if err := query.Preload("Uploader").Find(&assets).Error; err != nil {
		return nil, domain.NewInternalError("failed to list assets", err)
	}

	return assets, nil
}

func (r *AssetRepository) Count(ctx context.Context, filters domain.AssetFilters) (int64, error) {
	var count int64

	query := r.DB.WithContext(ctx).Model(&domain.Asset{})
	query = applyAssetFilters(query, filters)

	if err := query.Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count assets", err)
	}

	return count, nil
}

func (r *AssetRepository) Update(ctx context.Context, asset *domain.Asset) error {
	if err := r.DB.WithContext(ctx).Save(asset).Error; err != nil {
		return domain.NewInternalError("failed to update asset", err)
	}

	return nil
}

func (r *AssetRepository) UpdateMetadata(ctx context.Context, id uint, metadata string) error {
	if err := r.DB.WithContext(ctx).
		Model(&domain.Asset{}).
		Where("id = ?", id).
		Update("metadata", metadata).Error; err != nil {
		return domain.NewInternalError("failed to update asset metadata", err)
	}
	return nil
}

func (r *AssetRepository) FindOrphaned(ctx context.Context, olderThan time.Time) ([]*domain.Asset, error) {
	var assets []*domain.Asset

	// Find assets that are not referenced by any entity and are older than specified time
	// This is a basic implementation - you may need to customize based on your specific use case
	query := r.DB.WithContext(ctx).
		Where("reference_type IS NULL OR reference_id IS NULL").
		Where("created_at < ?", olderThan)

	if err := query.Find(&assets).Error; err != nil {
		return nil, domain.NewInternalError("failed to find orphaned assets", err)
	}

	return assets, nil
}

// applyAssetFilters applies common filter conditions to a GORM query.
// Returns the filtered query for safe chaining (GORM Where() may return a new instance).
func applyAssetFilters(query *gorm.DB, filters domain.AssetFilters) *gorm.DB {
	if filters.UploadType != nil {
		query = query.Where("upload_type = ?", *filters.UploadType)
	}

	if len(filters.UploadTypes) > 0 {
		query = query.Where("upload_type IN ?", filters.UploadTypes)
	}

	if filters.UploadedBy != nil {
		query = query.Where("uploaded_by = ?", *filters.UploadedBy)
	}

	if filters.FromDate != nil {
		query = query.Where("created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		endOfDay := timeutil.EndOfDay(*filters.ToDate)
		query = query.Where("created_at <= ?", endOfDay)
	}

	for key, value := range filters.MetadataQuery {
		query = query.Where("JSON_UNQUOTE(JSON_EXTRACT(metadata, ?)) = ?", fmt.Sprintf("$.%s", key), value)
	}

	return query
}
