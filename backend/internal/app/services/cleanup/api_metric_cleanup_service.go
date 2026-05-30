package cleanup

import (
	"context"
	"log/slog"

	"api-server/internal/domain"
)

// APIMetricCleanupService handles cleanup operations for API metrics
type APIMetricCleanupService struct {
	repo   domain.APIMetricRepository
	logger *slog.Logger
}

// NewAPIMetricCleanupService creates a new API metric cleanup service
func NewAPIMetricCleanupService(
	repo domain.APIMetricRepository,
	logger *slog.Logger,
) *APIMetricCleanupService {
	return &APIMetricCleanupService{
		repo:   repo,
		logger: logger,
	}
}

// DeleteOldMetrics deletes API metrics older than specified days
func (s *APIMetricCleanupService) DeleteOldMetrics(ctx context.Context, days int) (int64, error) {
	s.logger.Info("Starting API metrics cleanup", "days", days)

	count, err := s.repo.DeleteOldRecords(ctx, days)
	if err != nil {
		s.logger.Error("Failed to delete old API metrics", "error", err)
		return 0, err
	}

	s.logger.Info("Successfully deleted old API metrics", "count", count, "days", days)
	return count, nil
}
