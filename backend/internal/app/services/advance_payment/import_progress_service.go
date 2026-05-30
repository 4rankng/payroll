package advance_payment

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	infra "api-server/internal/app/services/infrastructure"
)

// ImportProgressService tracks import progress using Redis.
type ImportProgressService struct {
	cache *infra.CacheService
}

// NewImportProgressService creates a new import progress service backed by Redis
func NewImportProgressService(cache *infra.CacheService) *ImportProgressService {
	return &ImportProgressService{cache: cache}
}

func (s *ImportProgressService) progressKey(assetID uint) string {
	return fmt.Sprintf("import:progress:%d", assetID)
}

// StartImport initializes import progress tracking for an asset
func (s *ImportProgressService) StartImport(ctx context.Context, assetID uint, forMonth string) error {
	progress := &infra.ImportProgress{
		AssetID:   assetID,
		ForMonth:  forMonth,
		Status:    "pending",
		StartedAt: clock.Now(),
		UpdatedAt: clock.Now(),
	}
	return s.cache.Set(ctx, s.progressKey(assetID), progress, 24*time.Hour)
}

// UpdateProgress updates the import progress
func (s *ImportProgressService) UpdateProgress(ctx context.Context, assetID uint, totalRows, processedRows int) error {
	var progress infra.ImportProgress
	if err := s.cache.Get(ctx, s.progressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	progress.TotalRows = totalRows
	progress.ProcessedRows = processedRows
	progress.UpdatedAt = clock.Now()

	if totalRows > 0 {
		if processedRows >= totalRows {
			progress.Status = "completed"
		} else if processedRows > 0 {
			progress.Status = "processing"
		}
	}

	return s.cache.Set(ctx, s.progressKey(assetID), progress, 24*time.Hour)
}

// MarkAsFailed marks the import as failed with an error message
func (s *ImportProgressService) MarkAsFailed(ctx context.Context, assetID uint, errorMessage string) error {
	var progress infra.ImportProgress
	if err := s.cache.Get(ctx, s.progressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	progress.Status = "failed"
	progress.Error = &errorMessage
	progress.UpdatedAt = clock.Now()

	return s.cache.Set(ctx, s.progressKey(assetID), progress, 24*time.Hour)
}

// GetProgress retrieves the current import progress from Redis
func (s *ImportProgressService) GetProgress(ctx context.Context, assetID uint) (*infra.ImportProgress, error) {
	var progress infra.ImportProgress
	if err := s.cache.Get(ctx, s.progressKey(assetID), &progress); err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}
	return &progress, nil
}

// CompleteImport marks the import as completed with result
func (s *ImportProgressService) CompleteImport(ctx context.Context, assetID uint, result interface{}) error {
	var progress infra.ImportProgress
	if err := s.cache.Get(ctx, s.progressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	progress.Status = "completed"
	progress.UpdatedAt = clock.Now()

	return s.cache.Set(ctx, s.progressKey(assetID), progress, 24*time.Hour)
}
