package persistence

import (
	"context"
	"errors"
	"time"

	"api-server/internal/domain"
	dbhelper "api-server/internal/pkg/db"
	"api-server/internal/pkg/retry"

	"gorm.io/gorm"
)

// BaseRepository provides common database operations with retry logic for all repositories
type BaseRepository struct {
	DB       *gorm.DB // Exposed for complex queries in derived repositories
	dbHelper *dbhelper.DatabaseHelper
}

// NewBaseRepository creates a new base repository with database helper
func NewBaseRepository(db *Database) *BaseRepository {
	return &BaseRepository{
		DB:       db.DB,
		dbHelper: dbhelper.NewDatabaseHelper(db.DB),
	}
}

// WithRetryConfig allows customization of retry configuration
func (r *BaseRepository) WithRetryConfig(cfg retry.DatabaseOperationConfig) *BaseRepository {
	r.dbHelper = r.dbHelper.WithRetryConfig(retry.Config{
		MaxRetries:    cfg.MaxRetries,
		InitialDelay:  cfg.InitialDelay,
		MaxDelay:      cfg.MaxDelay,
		BackoffFactor: cfg.BackoffFactor,
	})
	return r
}

// SafeCreate creates a record with retry logic
func (r *BaseRepository) SafeCreate(ctx context.Context, value any) error {
	return r.dbHelper.SafeCreate(ctx, value)
}

// SafeUpdate updates a record with retry logic
func (r *BaseRepository) SafeUpdate(ctx context.Context, value any) error {
	return r.dbHelper.SafeUpdate(ctx, value)
}

// SafeFind finds records with retry logic
func (r *BaseRepository) SafeFind(ctx context.Context, dest any, conditions ...any) error {
	return r.dbHelper.SafeFind(ctx, dest, conditions...)
}

// SafeFirst finds first record with retry logic
func (r *BaseRepository) SafeFirst(ctx context.Context, dest any, conditions ...any) error {
	return r.dbHelper.SafeFirst(ctx, dest, conditions...)
}

// SafeCount counts records with retry logic
func (r *BaseRepository) SafeCount(ctx context.Context, model any, whereClause string, args ...any) (int64, error) {
	return r.dbHelper.CountWithRetry(ctx, model, whereClause, args...)
}

// SafeDelete soft deletes a record with retry logic
func (r *BaseRepository) SafeDelete(ctx context.Context, value any, conditions ...any) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		result := db.Delete(value, conditions...)
		if result.Error != nil {
			return r.dbHelper.WrapDatabaseError(result.Error)
		}
		return nil
	})
}

// SafeGetByID gets a record by ID with retry logic and proper error handling
func (r *BaseRepository) SafeGetByID(ctx context.Context, dest any, id uint, preloads ...string) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		query := db
		for _, preload := range preloads {
			query = query.Preload(preload)
		}

		err := query.First(dest, id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewNotFoundError("record not found")
			}
			return r.dbHelper.WrapDatabaseError(err)
		}
		return nil
	})
}

// SafeList lists records with filters, pagination, and retry logic
func (r *BaseRepository) SafeList(ctx context.Context, dest any, query *gorm.DB, limit, offset int, sortBy, sortOrder string) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		// Apply sorting
		if sortBy == "" {
			sortBy = "created_at"
		}
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		query = query.Order(sortBy + " " + sortOrder)

		// Apply pagination
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		err := query.Find(dest).Error
		if err != nil {
			return r.dbHelper.WrapDatabaseError(err)
		}
		return nil
	})
}

// ExecuteInTransaction executes operations within a transaction with retry logic
func (r *BaseRepository) ExecuteInTransaction(ctx context.Context, operations func(*gorm.DB) error) error {
	return r.dbHelper.ExecuteInTransactionWithRetry(ctx, operations)
}

// BatchUpdate performs batch updates with retry logic
func (r *BaseRepository) BatchUpdate(ctx context.Context, model any, updates map[string]any, whereClause string, args ...any) error {
	return r.dbHelper.BatchUpdate(ctx, model, updates, whereClause, args...)
}

// BatchCreate creates multiple records in a single operation with retry logic
func (r *BaseRepository) BatchCreate(ctx context.Context, values any, batchSize int) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		if batchSize > 0 {
			err := db.CreateInBatches(values, batchSize).Error
			if err != nil {
				return r.dbHelper.WrapDatabaseError(err)
			}
		} else {
			err := db.Create(values).Error
			if err != nil {
				return r.dbHelper.WrapDatabaseError(err)
			}
		}
		return nil
	})
}

// SafeRawQuery executes raw SQL queries with retry logic
func (r *BaseRepository) SafeRawQuery(ctx context.Context, dest any, sql string, values ...any) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		err := db.Raw(sql, values...).Scan(dest).Error
		if err != nil {
			return r.dbHelper.WrapDatabaseError(err)
		}
		return nil
	})
}

// SafeExec executes SQL commands with retry logic
func (r *BaseRepository) SafeExec(ctx context.Context, sql string, values ...any) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		err := db.Exec(sql, values...).Error
		if err != nil {
			return r.dbHelper.WrapDatabaseError(err)
		}
		return nil
	})
}

// HealthCheck performs a health check on the database connection
func (r *BaseRepository) HealthCheck(ctx context.Context) error {
	return r.dbHelper.HealthCheck(ctx)
}

// GetConnectionStats returns database connection statistics
func (r *BaseRepository) GetConnectionStats() map[string]any {
	return r.dbHelper.GetConnectionStats()
}

// WithTimeout creates a context with timeout for database operations
func (r *BaseRepository) WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// StandardPreloads returns common preload relationships
func (r *BaseRepository) StandardPreloads() []string {
	return []string{
		"LastUpdatedUser",
		"LastApprovedUser",
	}
}

// AuditPreloads returns preload relationships for auditable entities
func (r *BaseRepository) AuditPreloads() []string {
	return []string{
		"LastUpdatedUser",
		"LastApprovedUser",
		"CreatedByUser",
	}
}

// LogOperation logs database operations for debugging and monitoring
func (r *BaseRepository) LogOperation(ctx context.Context, operation string, entityType string, entityID *uint, duration time.Duration, err error) {
	// Implementation would depend on your logging setup
	// This is a placeholder for operation logging
	_ = ctx
	_ = operation
	_ = entityType
	_ = entityID
	_ = duration
	_ = err
}
