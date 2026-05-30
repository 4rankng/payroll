package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"api-server/internal/infra/observability"
)

type Config struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

func DefaultConfig() Config {
	return Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

func isDatabaseConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	connectionErrors := []string{
		"driver: bad connection",
		"connection refused",
		"connection reset",
		"connection timeout",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"connection lost",
		"server has gone away",
		"connection closed",
		"invalid connection",
		"connection aborted",
		"connection interrupted",
		"network error",
		"database is locked",
		"deadlock detected",
		"lock wait timeout",
		"timeout expired",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// isTransientError checks if an error is transient and should be retried
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Connection errors are always transient
	if isDatabaseConnectionError(err) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	transientErrors := []string{
		"too many connections",
		"max connections reached",
		"temporary failure",
		"service unavailable",
		"resource temporarily unavailable",
	}

	for _, transErr := range transientErrors {
		if strings.Contains(errStr, transErr) {
			return true
		}
	}

	return false
}

func WithExponentialBackoff(ctx context.Context, cfg Config, operation func() error) error {
	logger := observability.GetLogger()

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt-1)))
			delay = min(delay, cfg.MaxDelay)

			logger.Info("Retrying operation after delay",
				"attempt", attempt,
				"max_attempts", cfg.MaxRetries+1,
				"delay_ms", delay.Milliseconds())

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := operation()
		if err == nil {
			if attempt > 0 {
				logger.Info("Operation succeeded after retries", "successful_attempt", attempt+1)
			}
			return nil
		}

		lastErr = err

		if !isTransientError(err) {
			logger.Info("Non-retryable error encountered", "error", err, "error_type", "permanent")
			return err
		}

		if attempt < cfg.MaxRetries {
			logger.Warn("Database connection error, will retry",
				"attempt", attempt+1,
				"error", err)
		}
	}

	logger.Error("All retry attempts exhausted",
		"max_attempts", cfg.MaxRetries+1,
		"final_error", lastErr)

	return errors.New("database connection failed after multiple retries: " + lastErr.Error())
}

// DatabaseOperationConfig provides configuration for database operations
type DatabaseOperationConfig struct {
	MaxRetries       int
	InitialDelay     time.Duration
	MaxDelay         time.Duration
	BackoffFactor    float64
	OperationTimeout time.Duration
}

// DefaultDatabaseConfig returns default configuration for database operations
func DefaultDatabaseConfig() DatabaseOperationConfig {
	return DatabaseOperationConfig{
		MaxRetries:       5, // More retries for database operations
		InitialDelay:     50 * time.Millisecond,
		MaxDelay:         10 * time.Second,
		BackoffFactor:    1.5, // Gentler backoff for database operations
		OperationTimeout: 30 * time.Second,
	}
}

// WithDatabaseRetry executes database operations with specialized retry logic
func WithDatabaseRetry(ctx context.Context, cfg DatabaseOperationConfig, operation func() error) error {
	logger := observability.GetLogger()

	// Create operation context with timeout
	opCtx, cancel := context.WithTimeout(ctx, cfg.OperationTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check if context is cancelled
		if opCtx.Err() != nil {
			logger.Warn("Database operation cancelled due to context",
				"attempt", attempt,
				"context_error", opCtx.Err())
			return opCtx.Err()
		}

		if attempt > 0 {
			delay := time.Duration(float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt-1)))
			delay = min(delay, cfg.MaxDelay)

			logger.Info("Retrying database operation after delay",
				"attempt", attempt,
				"max_attempts", cfg.MaxRetries+1,
				"delay_ms", delay.Milliseconds(),
				"operation_timeout", cfg.OperationTimeout.String())

			select {
			case <-opCtx.Done():
				return opCtx.Err()
			case <-time.After(delay):
			}
		}

		startTime := time.Now()
		err := operation()
		duration := time.Since(startTime)

		if err == nil {
			if attempt > 0 {
				logger.Info("Database operation succeeded after retries",
					"successful_attempt", attempt+1,
					"duration_ms", duration.Milliseconds())
			}
			return nil
		}

		lastErr = err

		// Log operation performance
		logger.Info("Database operation attempt completed",
			"attempt", attempt+1,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error())

		// Check if error is retryable
		if !isTransientError(err) {
			logger.Info("Non-retryable database error encountered",
				"error", err,
				"error_type", "permanent",
				"duration_ms", duration.Milliseconds())
			return err
		}

		if attempt < cfg.MaxRetries {
			logger.Warn("Transient database error, will retry",
				"attempt", attempt+1,
				"max_attempts", cfg.MaxRetries+1,
				"error", err,
				"duration_ms", duration.Milliseconds())
		}
	}

	logger.Error("Database operation failed after all retry attempts",
		"max_attempts", cfg.MaxRetries+1,
		"final_error", lastErr,
		"total_timeout", cfg.OperationTimeout.String())

	return fmt.Errorf("database operation failed after %d retries: %w", cfg.MaxRetries+1, lastErr)
}
