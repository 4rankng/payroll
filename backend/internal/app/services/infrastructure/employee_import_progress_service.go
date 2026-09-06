package infrastructure

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/app/dto"
)

// EmployeeImportProgressService manages employee import job progress tracking using Redis
type EmployeeImportProgressService struct {
	cache *CacheService
}

// NewEmployeeImportProgressService creates a new employee import progress service
func NewEmployeeImportProgressService(cache *CacheService) *EmployeeImportProgressService {
	return &EmployeeImportProgressService{
		cache: cache,
	}
}

// GenerateImportKey generates a Redis key for employee import progress
func (s *EmployeeImportProgressService) GenerateImportKey(importID string) string {
	return fmt.Sprintf("import:employees:%s", importID)
}

// GenerateQueueKey generates the Redis key for the import job queue
func (s *EmployeeImportProgressService) GenerateQueueKey() string {
	return "import:employees:queue"
}

// StartImport initializes a new employee import progress tracking
func (s *EmployeeImportProgressService) StartImport(ctx context.Context, importID string, totalRows int, userID uint, filePath string) error {
	progress := dto.EmployeeImportProgress{
		ImportID:      importID,
		Status:        "pending",
		TotalRows:     totalRows,
		ProcessedRows: 0,
		CreatedCount:  0,
		UpdatedCount:  0,
		ErrorCount:    0,
		Errors:        []dto.RowError{},
		StartedAt:     clock.Now(),
		UserID:        userID,
		FilePath:      filePath,
	}

	return s.cache.Set(ctx, s.GenerateImportKey(importID), progress, 24*time.Hour)
}

// SetTotalRows updates the total rows count after parsing
func (s *EmployeeImportProgressService) SetTotalRows(ctx context.Context, importID string, totalRows int) error {
	// Get current progress
	var progress dto.EmployeeImportProgress
	if err := s.cache.Get(ctx, s.GenerateImportKey(importID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Update total rows
	progress.TotalRows = totalRows
	return s.cache.Set(ctx, s.GenerateImportKey(importID), progress, 24*time.Hour)
}

// UpdateProgress updates the import progress
func (s *EmployeeImportProgressService) UpdateProgress(ctx context.Context, importID string, processedRows, createdCount, updatedCount, errorCount int, errors []dto.RowError) error {
	// Get current progress
	var progress dto.EmployeeImportProgress
	if err := s.cache.Get(ctx, s.GenerateImportKey(importID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Update progress
	progress.ProcessedRows = processedRows
	progress.CreatedCount = createdCount
	progress.UpdatedCount = updatedCount
	progress.ErrorCount = errorCount
	progress.Errors = errors

	// Update status based on progress
	if processedRows > 0 && progress.Status == "pending" {
		progress.Status = "processing"
	}

	// Mark as completed when all rows are processed
	if progress.TotalRows > 0 && processedRows >= progress.TotalRows {
		progress.Status = "completed"
		now := clock.Now()
		progress.CompletedAt = &now
	}

	return s.cache.Set(ctx, s.GenerateImportKey(importID), progress, 24*time.Hour)
}

// GetProgress retrieves the current import progress
func (s *EmployeeImportProgressService) GetProgress(ctx context.Context, importID string) (*dto.EmployeeImportProgress, error) {
	var progress dto.EmployeeImportProgress
	err := s.cache.Get(ctx, s.GenerateImportKey(importID), &progress)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}
	return &progress, nil
}

// MarkAsFailed marks the import as failed with an error message
func (s *EmployeeImportProgressService) MarkAsFailed(ctx context.Context, importID string, errorMessage string) error {
	// Get current progress
	var progress dto.EmployeeImportProgress
	if err := s.cache.Get(ctx, s.GenerateImportKey(importID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	// Update progress
	progress.Status = "failed"
	errors := []dto.RowError{
		{
			RowNumber: 0,
			Message:   errorMessage,
		},
	}
	progress.Errors = errors
	now := clock.Now()
	progress.CompletedAt = &now

	return s.cache.Set(ctx, s.GenerateImportKey(importID), progress, 24*time.Hour)
}

// MarkAsProcessing marks the import as processing
func (s *EmployeeImportProgressService) MarkAsProcessing(ctx context.Context, importID string) error {
	// Get current progress
	var progress dto.EmployeeImportProgress
	if err := s.cache.Get(ctx, s.GenerateImportKey(importID), &progress); err != nil {
		return fmt.Errorf("failed to get current progress: %w", err)
	}

	progress.Status = "processing"
	return s.cache.Set(ctx, s.GenerateImportKey(importID), progress, 24*time.Hour)
}

// CleanupProgress removes progress data (optional)
func (s *EmployeeImportProgressService) CleanupProgress(ctx context.Context, importID string) error {
	return s.cache.Delete(ctx, s.GenerateImportKey(importID))
}

// GetImportStatus returns the public status (without internal fields like FilePath)
func (s *EmployeeImportProgressService) GetImportStatus(ctx context.Context, importID string) (*dto.EmployeeImportStatus, error) {
	progress, err := s.GetProgress(ctx, importID)
	if err != nil {
		return nil, err
	}

	percentage := 0
	if progress.TotalRows > 0 {
		percentage = int(float64(progress.ProcessedRows) / float64(progress.TotalRows) * 100)
	}
	if progress.Status == "completed" {
		percentage = 100
	}

	return &dto.EmployeeImportStatus{
		ImportID:      progress.ImportID,
		Status:        progress.Status,
		TotalRows:     progress.TotalRows,
		ProcessedRows: progress.ProcessedRows,
		CreatedCount:  progress.CreatedCount,
		UpdatedCount:  progress.UpdatedCount,
		ErrorCount:    progress.ErrorCount,
		Errors:        progress.Errors,
		Percentage:    percentage,
		StartedAt:     progress.StartedAt,
		CompletedAt:   progress.CompletedAt,
	}, nil
}
