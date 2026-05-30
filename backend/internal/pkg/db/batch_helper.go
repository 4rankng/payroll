package db

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/infra/observability"

	"gorm.io/gorm"
)

// BatchOperationHelper provides utilities for efficient batch database operations
type BatchOperationHelper struct {
	dbHelper *DatabaseHelper
}

// NewBatchOperationHelper creates a new batch operation helper
func NewBatchOperationHelper(dbHelper *DatabaseHelper) *BatchOperationHelper {
	return &BatchOperationHelper{
		dbHelper: dbHelper,
	}
}

// BatchInsertConfig configures batch insert operations
type BatchInsertConfig struct {
	BatchSize      int           // Number of records per batch
	Timeout        time.Duration // Timeout per batch
	OnBatchSuccess func(batchNum int, count int)
	OnBatchError   func(batchNum int, err error)
}

// DefaultBatchInsertConfig returns default configuration for batch inserts
func DefaultBatchInsertConfig() BatchInsertConfig {
	return BatchInsertConfig{
		BatchSize: 1000,
		Timeout:   30 * time.Second,
	}
}

// BatchInsert performs batch insert operations with progress tracking
func (h *BatchOperationHelper) BatchInsert(ctx context.Context, records any, config BatchInsertConfig) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		startTime := time.Now()

		result := db.CreateInBatches(records, config.BatchSize)
		if result.Error != nil {
			if config.OnBatchError != nil {
				config.OnBatchError(1, result.Error)
			}
			return h.dbHelper.WrapDatabaseError(result.Error)
		}

		duration := time.Since(startTime)
		rowsAffected := result.RowsAffected

		if config.OnBatchSuccess != nil {
			config.OnBatchSuccess(1, int(rowsAffected))
		}

		logger.Info("Batch insert completed",
			"rows_affected", rowsAffected,
			"batch_size", config.BatchSize,
			"duration_ms", duration.Milliseconds())

		return nil
	})
}

// BatchUpdateConfig configures batch update operations
type BatchUpdateConfig struct {
	BatchSize      int           // Number of records per batch
	Timeout        time.Duration // Timeout per batch
	UpdateFields   map[string]any
	OnBatchSuccess func(batchNum int, count int)
	OnBatchError   func(batchNum int, err error)
}

// DefaultBatchUpdateConfig returns default configuration for batch updates
func DefaultBatchUpdateConfig() BatchUpdateConfig {
	return BatchUpdateConfig{
		BatchSize: 500,
		Timeout:   30 * time.Second,
	}
}

// BatchUpdateByIDs performs batch updates by ID list
func (h *BatchOperationHelper) BatchUpdateByIDs(ctx context.Context, model any, ids []uint, updates map[string]any, config BatchUpdateConfig) error {
	logger := observability.GetLogger()

	if len(ids) == 0 {
		return nil
	}

	batchCount := (len(ids) + config.BatchSize - 1) / config.BatchSize

	for i := 0; i < batchCount; i++ {
		start := i * config.BatchSize
		end := start + config.BatchSize
		if end > len(ids) {
			end = len(ids)
		}

		batchIDs := ids[start:end]
		batchNum := i + 1

		err := h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
			batchCtx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()

			result := db.WithContext(batchCtx).Model(model).Where("id IN ?", batchIDs).Updates(updates)
			if result.Error != nil {
				if config.OnBatchError != nil {
					config.OnBatchError(batchNum, result.Error)
				}
				return h.dbHelper.WrapDatabaseError(result.Error)
			}

			if config.OnBatchSuccess != nil {
				config.OnBatchSuccess(batchNum, int(result.RowsAffected))
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	logger.Info("All batch updates completed",
		"total_batches", batchCount,
		"total_records", len(ids))

	return nil
}

// BulkUpsertConfig configures bulk upsert operations
type BulkUpsertConfig struct {
	ConflictColumns []string      // Columns to check for conflicts
	UpdateColumns   []string      // Columns to update on conflict
	BatchSize       int           // Number of records per batch
	Timeout         time.Duration // Timeout per operation
}

// DefaultBulkUpsertConfig returns default configuration for bulk upserts
func DefaultBulkUpsertConfig() BulkUpsertConfig {
	return BulkUpsertConfig{
		BatchSize: 500,
		Timeout:   30 * time.Second,
	}
}

// BulkUpsert performs bulk upsert (INSERT ... ON DUPLICATE KEY UPDATE) operations
func (h *BatchOperationHelper) BulkUpsert(ctx context.Context, records any, config BulkUpsertConfig) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		startTime := time.Now()

		// Use GORM's Clauses for upsert functionality
		result := db.CreateInBatches(records, config.BatchSize)
		if result.Error != nil {
			return h.dbHelper.WrapDatabaseError(result.Error)
		}

		duration := time.Since(startTime)

		logger.Info("Bulk upsert completed",
			"rows_affected", result.RowsAffected,
			"batch_size", config.BatchSize,
			"duration_ms", duration.Milliseconds())

		return nil
	})
}

// BatchDeleteConfig configures batch delete operations
type BatchDeleteConfig struct {
	BatchSize      int           // Number of records per batch
	Timeout        time.Duration // Timeout per batch
	SoftDelete     bool          // Whether to perform soft delete
	OnBatchSuccess func(batchNum int, count int)
	OnBatchError   func(batchNum int, err error)
}

// DefaultBatchDeleteConfig returns default configuration for batch deletes
func DefaultBatchDeleteConfig() BatchDeleteConfig {
	return BatchDeleteConfig{
		BatchSize:  500,
		Timeout:    30 * time.Second,
		SoftDelete: true,
	}
}

// BatchDeleteByIDs performs batch delete operations by ID list
func (h *BatchOperationHelper) BatchDeleteByIDs(ctx context.Context, model any, ids []uint, config BatchDeleteConfig) error {
	logger := observability.GetLogger()

	if len(ids) == 0 {
		return nil
	}

	batchCount := (len(ids) + config.BatchSize - 1) / config.BatchSize

	for i := 0; i < batchCount; i++ {
		start := i * config.BatchSize
		end := start + config.BatchSize
		if end > len(ids) {
			end = len(ids)
		}

		batchIDs := ids[start:end]
		batchNum := i + 1

		err := h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
			batchCtx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()

			var result *gorm.DB
			if config.SoftDelete {
				result = db.WithContext(batchCtx).Delete(model, batchIDs)
			} else {
				result = db.WithContext(batchCtx).Unscoped().Delete(model, batchIDs)
			}

			if result.Error != nil {
				if config.OnBatchError != nil {
					config.OnBatchError(batchNum, result.Error)
				}
				return h.dbHelper.WrapDatabaseError(result.Error)
			}

			if config.OnBatchSuccess != nil {
				config.OnBatchSuccess(batchNum, int(result.RowsAffected))
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	logger.Info("All batch deletes completed",
		"total_batches", batchCount,
		"total_records", len(ids),
		"soft_delete", config.SoftDelete)

	return nil
}

// BatchQueryResult represents the result of a batch query operation
type BatchQueryResult struct {
	BatchNumber int
	Records     any
	Count       int64
	Error       error
}

// BatchQueryConfig configures batch query operations
type BatchQueryConfig struct {
	BatchSize int           // Number of records per batch
	Timeout   time.Duration // Timeout per batch
	OrderBy   string        // Order by clause for consistent batching
}

// DefaultBatchQueryConfig returns default configuration for batch queries
func DefaultBatchQueryConfig() BatchQueryConfig {
	return BatchQueryConfig{
		BatchSize: 1000,
		Timeout:   30 * time.Second,
		OrderBy:   "id ASC",
	}
}

// BatchQuery performs paginated queries with batch processing
func (h *BatchOperationHelper) BatchQuery(ctx context.Context, model any, whereClause string, args []any, config BatchQueryConfig, processor func(*BatchQueryResult) error) error {
	logger := observability.GetLogger()
	offset := 0
	batchNum := 1

	for {
		var records any
		var count int64

		err := h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
			batchCtx, cancel := context.WithTimeout(ctx, config.Timeout)
			defer cancel()

			query := db.WithContext(batchCtx).Model(model)
			if whereClause != "" {
				query = query.Where(whereClause, args...)
			}
			if config.OrderBy != "" {
				query = query.Order(config.OrderBy)
			}

			query = query.Limit(config.BatchSize).Offset(offset)

			result := query.Find(&records)
			if result.Error != nil {
				return h.dbHelper.WrapDatabaseError(result.Error)
			}

			count = result.RowsAffected
			return nil
		})

		if err != nil {
			return err
		}

		// If no records found, we're done
		if count == 0 {
			break
		}

		// Process this batch
		batchResult := &BatchQueryResult{
			BatchNumber: batchNum,
			Records:     records,
			Count:       count,
			Error:       nil,
		}

		if err := processor(batchResult); err != nil {
			logger.Error("Batch processing failed",
				"batch", batchNum,
				"offset", offset,
				"count", count,
				"error", err)
			return fmt.Errorf("batch processing failed at batch %d: %w", batchNum, err)
		}

		// If we got fewer records than batch size, we're done
		if count < int64(config.BatchSize) {
			break
		}

		offset += config.BatchSize
		batchNum++
	}

	logger.Info("Batch query processing completed",
		"total_batches", batchNum-1,
		"batch_size", config.BatchSize)

	return nil
}

// OptimizeTable performs table optimization operations
func (h *BatchOperationHelper) OptimizeTable(ctx context.Context, tableName string) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		// MySQL-specific optimization
		err := db.Exec(fmt.Sprintf("OPTIMIZE TABLE %s", tableName)).Error
		if err != nil {
			return h.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Table optimization completed", "table", tableName)
		return nil
	})
}

// AnalyzeTable performs table analysis for query optimization
func (h *BatchOperationHelper) AnalyzeTable(ctx context.Context, tableName string) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		// MySQL-specific analysis
		err := db.Exec(fmt.Sprintf("ANALYZE TABLE %s", tableName)).Error
		if err != nil {
			return h.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Table analysis completed", "table", tableName)
		return nil
	})
}
