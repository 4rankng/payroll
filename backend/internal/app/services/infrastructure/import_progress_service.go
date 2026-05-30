package infrastructure

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ImportProgress represents the progress of an import operation
type ImportProgress struct {
	AssetID       uint      `json:"asset_id"`
	ForMonth      string    `json:"for_month"`
	TotalRows     int       `json:"total_rows"`
	ProcessedRows int       `json:"processed_rows"`
	Status        string    `json:"status"`
	Error         *string   `json:"error,omitempty"`
	Result        *string   `json:"result,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ImportProgressService manages import progress tracking using Redis
type ImportProgressService struct {
	cache *CacheService
}

// NewImportProgressService creates a new import progress service
func NewImportProgressService(cache *CacheService) *ImportProgressService {
	return &ImportProgressService{
		cache: cache,
	}
}

// GenerateProgressKey generates a Redis key for import progress
func (s *ImportProgressService) GenerateProgressKey(assetID uint) string {
	return fmt.Sprintf("import:progress:%d", assetID)
}

// StartImport initializes a new import progress tracking
func (s *ImportProgressService) StartImport(ctx context.Context, assetID uint, forMonth string) error {
	progress := ImportProgress{
		AssetID:   assetID,
		ForMonth:  forMonth,
		Status:    "pending",
		StartedAt: clock.Now(),
		UpdatedAt: clock.Now(),
	}

	return s.cache.Set(ctx, s.GenerateProgressKey(assetID), progress, 24*time.Hour)
}

// UpdateProgress updates the import progress
func (s *ImportProgressService) UpdateProgress(ctx context.Context, assetID uint, totalRows, processedRows int) error {
	// Get current progress
	var progress ImportProgress
	if err := s.cache.Get(ctx, s.GenerateProgressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Update progress
	progress.TotalRows = totalRows
	progress.ProcessedRows = processedRows
	progress.UpdatedAt = clock.Now()

	// Update status based on progress
	if totalRows > 0 {
		if processedRows >= totalRows {
			progress.Status = "completed"
		} else if processedRows > 0 {
			progress.Status = "processing"
		}
	}

	return s.cache.Set(ctx, s.GenerateProgressKey(assetID), progress, 24*time.Hour)
}

// GetProgress retrieves the current import progress
func (s *ImportProgressService) GetProgress(ctx context.Context, assetID uint) (*ImportProgress, error) {
	var progress ImportProgress
	err := s.cache.Get(ctx, s.GenerateProgressKey(assetID), &progress)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}
	return &progress, nil
}

// MarkAsFailed marks the import as failed with an error message
func (s *ImportProgressService) MarkAsFailed(ctx context.Context, assetID uint, errorMessage string) error {
	// Get current progress
	var progress ImportProgress
	if err := s.cache.Get(ctx, s.GenerateProgressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Update progress
	progress.Status = "failed"
	progress.Error = &errorMessage
	progress.UpdatedAt = clock.Now()

	return s.cache.Set(ctx, s.GenerateProgressKey(assetID), progress, 24*time.Hour)
}

// CompleteImport marks the import as completed with result
func (s *ImportProgressService) CompleteImport(ctx context.Context, assetID uint, result interface{}) error {
	// Get current progress
	var progress ImportProgress
	if err := s.cache.Get(ctx, s.GenerateProgressKey(assetID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Convert result to JSON
	resultJSON, _ := json.Marshal(result)
	resultStr := string(resultJSON)

	// Update progress
	progress.Status = "completed"
	progress.Result = &resultStr
	progress.UpdatedAt = clock.Now()

	return s.cache.Set(ctx, s.GenerateProgressKey(assetID), progress, 24*time.Hour)
}

// GetPercentage calculates the completion percentage
func (s *ImportProgressService) GetPercentage(progress *ImportProgress) int {
	if progress.TotalRows == 0 {
		return 0
	}
	percentage := float64(progress.ProcessedRows) / float64(progress.TotalRows) * 100
	return int(percentage)
}

// CleanupProgress removes progress data after completion (optional)
func (s *ImportProgressService) CleanupProgress(ctx context.Context, assetID uint) error {
	return s.cache.Delete(ctx, s.GenerateProgressKey(assetID))
}
