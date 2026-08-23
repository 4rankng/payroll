package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/retry"

	"gorm.io/gorm"
)

// DatabaseHelper provides utilities for reliable database operations
type DatabaseHelper struct {
	db             *gorm.DB
	retryConfig    retry.Config
	batchHelper    *BatchOperationHelper
	temporalHelper *TemporalHelper
}

// NewDatabaseHelper creates a new database helper
func NewDatabaseHelper(db *gorm.DB) *DatabaseHelper {
	helper := &DatabaseHelper{
		db:          db,
		retryConfig: retry.DefaultConfig(),
	}
	helper.batchHelper = NewBatchOperationHelper(helper)
	helper.temporalHelper = NewTemporalHelper(helper)
	return helper
}

// WithRetryConfig allows customization of retry configuration
func (h *DatabaseHelper) WithRetryConfig(cfg retry.Config) *DatabaseHelper {
	h.retryConfig = cfg
	return h
}

// ExecuteWithRetry executes a database operation with retry logic
func (h *DatabaseHelper) ExecuteWithRetry(ctx context.Context, operation func(*gorm.DB) error) error {
	return retry.WithExponentialBackoff(ctx, h.retryConfig, func() error {
		// Check if context is cancelled before operation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Validate connection health before operation
		if err := h.validateConnection(ctx); err != nil {
			return fmt.Errorf("connection validation failed: %w", err)
		}

		return operation(h.db.WithContext(ctx))
	})
}

// ExecuteInTransactionWithRetry executes operations within a transaction with retry
func (h *DatabaseHelper) ExecuteInTransactionWithRetry(ctx context.Context, operations func(*gorm.DB) error) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		// Check if we're already in a transaction to avoid nested transactions
		if h.isInTransaction(db) {
			// Already in transaction, execute operations directly
			return operations(db)
		}

		// Start new transaction with timeout
		return db.Transaction(func(tx *gorm.DB) error {
			// Set transaction timeout
			txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			tx = tx.WithContext(txCtx)
			return operations(tx)
		})
	})
}

// BatchUpdate performs batch updates with retry logic
func (h *DatabaseHelper) BatchUpdate(ctx context.Context, model any, updates map[string]any, whereClause string, args ...any) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		result := db.Model(model).Where(whereClause, args...).Updates(updates)
		if result.Error != nil {
			return h.WrapDatabaseError(result.Error)
		}
		return nil
	})
}

// SafeCreate creates a record with retry and better error handling
func (h *DatabaseHelper) SafeCreate(ctx context.Context, value any) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		if err := db.Create(value).Error; err != nil {
			return h.WrapDatabaseError(err)
		}
		return nil
	})
}

// SafeUpdate updates a record with retry and better error handling
func (h *DatabaseHelper) SafeUpdate(ctx context.Context, value any) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		if err := db.Save(value).Error; err != nil {
			return h.WrapDatabaseError(err)
		}
		return nil
	})
}

// SafeFind performs a find operation with retry
func (h *DatabaseHelper) SafeFind(ctx context.Context, dest any, conditions ...any) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		if err := db.Find(dest, conditions...).Error; err != nil {
			return h.WrapDatabaseError(err)
		}
		return nil
	})
}

// SafeFirst performs a first operation with retry
func (h *DatabaseHelper) SafeFirst(ctx context.Context, dest any, conditions ...any) error {
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		err := db.First(dest, conditions...).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err // Don't wrap ErrRecordNotFound
			}
			return h.WrapDatabaseError(err)
		}
		return nil
	})
}

// CountWithRetry performs count operation with retry
func (h *DatabaseHelper) CountWithRetry(ctx context.Context, model any, whereClause string, args ...any) (int64, error) {
	var count int64
	err := h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		return db.Model(model).Where(whereClause, args...).Count(&count).Error
	})
	return count, err
}

// RawQueryWithRetry executes raw SQL with retry
func (h *DatabaseHelper) RawQueryWithRetry(ctx context.Context, sql string, values ...any) (*gorm.DB, error) {
	var result *gorm.DB
	err := h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		result = db.Raw(sql, values...)
		return result.Error
	})
	return result, err
}

// validateConnection checks if the database connection is healthy
func (h *DatabaseHelper) validateConnection(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Use a shorter timeout for connection validation
	validateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(validateCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// isInTransaction checks if we're already in a transaction
func (h *DatabaseHelper) isInTransaction(db *gorm.DB) bool {
	// GORM doesn't expose transaction state directly
	// We use a heuristic approach by checking the statement context
	stmt := db.Statement
	if stmt == nil {
		return false
	}

	// If we have a statement with a non-nil DB, we're likely in a transaction context
	// This is not foolproof but works for most cases
	return stmt.DB != nil
}

// WrapDatabaseError wraps database errors with domain-specific errors
func (h *DatabaseHelper) WrapDatabaseError(err error) error {
	if err == nil {
		return nil
	}

	logger := observability.GetLogger()
	errStr := strings.ToLower(err.Error())

	// Handle connection errors
	if h.isConnectionError(err) {
		logger.Warn("Database connection error detected", "error", err)
		return domain.NewInternalError("Database connection error. Please try again later.", err)
	}

	// Handle constraint violations
	if strings.Contains(errStr, "constraint") || strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "1062") {
		if strings.Contains(errStr, "chk_payrates_date_order") {
			return domain.NewValidationError(constants.MsgCannotProcessPayrateInvalidDateVN)
		}
		if strings.Contains(errStr, "uq_payrates_live_project_from_date") {
			return domain.NewValidationError(constants.MsgPayrateDuplicateEffectiveDateVN)
		}
		if strings.Contains(errStr, "uq_payrates_one_open_per_project") {
			return domain.NewValidationError(constants.MsgPayrateMultipleOpenConfigurationsVN)
		}

		// Handle MySQL duplicate key errors for employees
		if strings.Contains(errStr, "idx_employees_cccd") || strings.Contains(errStr, "cccd") {
			return domain.NewConflictError(constants.MsgEmployeeWithCCCDExistsVN)
		}
		if strings.Contains(errStr, "idx_employees_email") || (strings.Contains(errStr, "email") && strings.Contains(errStr, "employees")) {
			return domain.NewConflictError(constants.MsgEmployeeWithEmailExistsVN)
		}

		// Generic constraint violation fallback
		return domain.NewValidationError(constants.MsgDatabaseConstraintViolationVN + ": " + err.Error())
	}

	// Handle timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		logger.Warn("Database operation timeout", "error", err)
		return domain.NewInternalError("Database operation timed out. Please try again.", err)
	}

	// Return original error for other cases
	return err
}

// isConnectionError checks if an error is a database connection error
func (h *DatabaseHelper) isConnectionError(err error) bool {
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
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// GetConnectionStats returns database connection statistics
func (h *DatabaseHelper) GetConnectionStats() map[string]any {
	stats := make(map[string]any)
	sqlDB, err := h.db.DB()
	if err != nil {
		stats["error"] = err.Error()
		return stats
	}

	dbStats := sqlDB.Stats()
	stats["max_open_connections"] = dbStats.MaxOpenConnections
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use"] = dbStats.InUse
	stats["idle"] = dbStats.Idle
	stats["wait_count"] = dbStats.WaitCount
	stats["wait_duration"] = dbStats.WaitDuration.String()

	if dbStats.MaxOpenConnections > 0 {
		utilization := float64(dbStats.OpenConnections) / float64(dbStats.MaxOpenConnections) * 100
		stats["utilization_percent"] = utilization
		stats["high_utilization"] = utilization > 80.0
	}

	return stats
}

// HealthCheck performs a comprehensive database health check
func (h *DatabaseHelper) HealthCheck(ctx context.Context) error {
	// Check basic connectivity
	if err := h.validateConnection(ctx); err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}

	// Check connection pool health
	stats := h.GetConnectionStats()
	if highUtil, ok := stats["high_utilization"].(bool); ok && highUtil {
		observability.GetLogger().Warn("Database connection pool utilization is high", "stats", stats)
	}

	// Perform a simple query to verify database responsiveness
	return h.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		var result int
		return db.Raw("SELECT 1").Scan(&result).Error
	})
}

// GetBatchHelper returns the batch operation helper
func (h *DatabaseHelper) GetBatchHelper() *BatchOperationHelper {
	return h.batchHelper
}

// GetTemporalHelper returns the temporal operation helper
func (h *DatabaseHelper) GetTemporalHelper() *TemporalHelper {
	return h.temporalHelper
}
