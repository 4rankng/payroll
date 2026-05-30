package infrastructure

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/constants"

	"github.com/redis/go-redis/v9"
)

// IdempotencyService manages idempotency using Redis keys
type IdempotencyService struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewIdempotencyService creates a new IdempotencyService
func NewIdempotencyService(redis *redis.Client, logger *slog.Logger) *IdempotencyService {
	return &IdempotencyService{
		redis:  redis,
		logger: logger,
	}
}

// IdempotencyState represents the state of an idempotent operation
type IdempotencyState string

const (
	StateNotStarted IdempotencyState = "not_started"
	StateProcessing IdempotencyState = "processing"
	StateCompleted  IdempotencyState = "completed"
	StateFailed     IdempotencyState = "failed"
)

// IdempotencyResult contains the result of an idempotency check
type IdempotencyResult struct {
	State   IdempotencyState
	Message string
}

// TryAcquireLock attempts to acquire a processing lock for the given key
// Returns true if lock was acquired, false if already being processed or completed
func (s *IdempotencyService) TryAcquireLock(ctx context.Context, idempotencyKey string) (bool, IdempotencyState, error) {
	startedKey := fmt.Sprintf("idem:started:%s", idempotencyKey)
	resultKey := fmt.Sprintf("idem:result:%s", idempotencyKey)

	// Check if already completed
	exists, err := s.redis.Exists(ctx, resultKey).Result()
	if err != nil {
		return false, StateNotStarted, fmt.Errorf("failed to check completion: %w", err)
	}
	if exists > 0 {
		s.logger.Info("Idempotent operation already completed", "key", idempotencyKey)
		return false, StateCompleted, nil
	}

	// Try to acquire processing lock using SET NX (set if not exists)
	acquired, err := s.redis.SetNX(ctx, startedKey, clock.Now().Unix(), constants.IdempotencyLockTTL).Result()
	if err != nil {
		return false, StateNotStarted, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		s.logger.Info("Idempotent operation already in progress", "key", idempotencyKey)
		return false, StateProcessing, nil
	}

	s.logger.Info("Acquired processing lock", "key", idempotencyKey)
	return true, StateNotStarted, nil
}

// MarkCompleted marks an idempotent operation as completed
func (s *IdempotencyService) MarkCompleted(ctx context.Context, idempotencyKey string) error {
	startedKey := fmt.Sprintf("idem:started:%s", idempotencyKey)
	resultKey := fmt.Sprintf("idem:result:%s", idempotencyKey)

	// Set result key to "completed" with TTL
	err := s.redis.Set(ctx, resultKey, "completed", constants.IdempotencyKeyTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to mark as completed: %w", err)
	}

	// Delete the processing lock
	err = s.redis.Del(ctx, startedKey).Err()
	if err != nil {
		s.logger.Warn("Failed to delete processing lock", "key", idempotencyKey, "error", err)
		// Non-critical error, continue
	}

	s.logger.Info("Marked operation as completed", "key", idempotencyKey)
	return nil
}

// MarkFailed marks an idempotent operation as failed
func (s *IdempotencyService) MarkFailed(ctx context.Context, idempotencyKey string, errorMsg string) error {
	startedKey := fmt.Sprintf("idem:started:%s", idempotencyKey)
	resultKey := fmt.Sprintf("idem:result:%s", idempotencyKey)

	// Set result key to "failed:{error}" with shorter TTL for retry
	failureValue := fmt.Sprintf("failed:%s", errorMsg)
	err := s.redis.Set(ctx, resultKey, failureValue, constants.IdempotencyResultTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to mark as failed: %w", err)
	}

	// Delete the processing lock
	err = s.redis.Del(ctx, startedKey).Err()
	if err != nil {
		s.logger.Warn("Failed to delete processing lock", "key", idempotencyKey, "error", err)
	}

	s.logger.Warn("Marked operation as failed", "key", idempotencyKey, "error", errorMsg)
	return nil
}

// ReleaseLock releases the processing lock (use when operation is aborted/retried)
func (s *IdempotencyService) ReleaseLock(ctx context.Context, idempotencyKey string) error {
	startedKey := fmt.Sprintf("idem:started:%s", idempotencyKey)

	err := s.redis.Del(ctx, startedKey).Err()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	s.logger.Info("Released processing lock", "key", idempotencyKey)
	return nil
}

// GetState retrieves the current state of an idempotent operation
func (s *IdempotencyService) GetState(ctx context.Context, idempotencyKey string) (IdempotencyState, error) {
	startedKey := fmt.Sprintf("idem:started:%s", idempotencyKey)
	resultKey := fmt.Sprintf("idem:result:%s", idempotencyKey)

	// Check result key first
	resultExists, err := s.redis.Exists(ctx, resultKey).Result()
	if err != nil {
		return StateNotStarted, fmt.Errorf("failed to check result key: %w", err)
	}
	if resultExists > 0 {
		resultVal, _ := s.redis.Get(ctx, resultKey).Result()
		if resultVal == "completed" {
			return StateCompleted, nil
		}
		return StateFailed, nil
	}

	// Check started key
	startedExists, err := s.redis.Exists(ctx, startedKey).Result()
	if err != nil {
		return StateNotStarted, fmt.Errorf("failed to check started key: %w", err)
	}
	if startedExists > 0 {
		return StateProcessing, nil
	}

	return StateNotStarted, nil
}

// IsCompleted checks if an idempotent operation has been completed
func (s *IdempotencyService) IsCompleted(ctx context.Context, idempotencyKey string) (bool, error) {
	state, err := s.GetState(ctx, idempotencyKey)
	if err != nil {
		return false, err
	}
	return state == StateCompleted, nil
}
