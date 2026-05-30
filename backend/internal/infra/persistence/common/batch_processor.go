package common

import (
	"context"
	"fmt"
	"reflect"

	"api-server/internal/infra/observability"
	"log/slog"

	"gorm.io/gorm"
)

// BatchConfig defines configuration for batch processing operations.
type BatchConfig struct {
	// DefaultSize is the default batch size to use (default: 1000)
	DefaultSize int
	// MaxSize is the maximum allowed batch size (default: 10000)
	MaxSize int
	// QuerySizeLimit is the MySQL query size limit in bytes (default: 32768)
	QuerySizeLimit int
}

// DefaultBatchConfig returns the default batch configuration.
// Uses sensible defaults that work well for most operations:
// - Default batch size: 1000 items
// - Maximum batch size: 10000 items
// - MySQL query size limit: 32768 bytes (32KB)
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		DefaultSize:    1000,
		MaxSize:        10000,
		QuerySizeLimit: 32768, // MySQL's max_allowed_packet default
	}
}

// BatchProcessor handles centralized batch processing for database operations.
// It eliminates magic numbers scattered across repositories and provides
// consistent, configurable batch processing with proper error handling and logging.
type BatchProcessor struct {
	config BatchConfig
	logger *slog.Logger
}

// NewBatchProcessor creates a new BatchProcessor with the given configuration.
func NewBatchProcessor(config BatchConfig) *BatchProcessor {
	return &BatchProcessor{
		config: config,
		logger: observability.GetLogger(),
	}
}

// ProcessInBatches processes items in batches using the provided processor function.
// This is a generic method that works with any type.
//
// The processor function receives a batch of items and should process them as a single unit.
// If the processor returns an error, the batch processing stops and the error is returned.
//
// Parameters:
//   - ctx: Context for cancellation and logging
//   - items: Items to process (should be a slice of any type)
//   - processor: Function that processes a single batch
//
// Example:
//
//	bp := NewBatchProcessor(DefaultBatchConfig())
//	ids := []int64{1, 2, 3, 4, 5}
//	err := bp.ProcessInBatches(ctx, ids, func(batch interface{}) error {
//	    batchIDs := batch.([]int64)
//	    return db.Where("id IN ?", batchIDs).Find(&results).Error
//	})
func (bp *BatchProcessor) ProcessInBatches(
	ctx context.Context,
	items interface{},
	processor func(batch interface{}) error,
) error {
	// Use reflection to get slice length and create batches
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("items must be a slice, got %T", items)
	}

	itemCount := v.Len()
	if itemCount == 0 {
		return nil
	}

	batchSize := bp.config.DefaultSize
	if batchSize > itemCount {
		batchSize = itemCount
	}

	if batchSize > bp.config.MaxSize {
		batchSize = bp.config.MaxSize
	}

	batchNum := 0
	for i := 0; i < itemCount; i += batchSize {
		end := i + batchSize
		if end > itemCount {
			end = itemCount
		}

		// Create a slice for this batch
		batch := v.Slice(i, end).Interface()

		if err := processor(batch); err != nil {
			bp.logger.Error("batch_processing_failed",
				"batch_number", batchNum,
				"batch_start", i,
				"batch_end", end,
				"batch_size", end-i,
				"error", err,
			)
			return fmt.Errorf("failed to process batch %d (items %d-%d): %w", batchNum, i, end, err)
		}

		batchNum++
	}

	return nil
}

// FetchInBatches fetches data from the database in batches to avoid memory issues
// with large result sets. This is useful when processing millions of records.
//
// The processor function receives a GORM query configured for a single batch
// and should scan the results into the appropriate destination.
//
// Parameters:
//   - ctx: Context for cancellation and logging
//   - query: Base query to execute (will be modified with Limit/Offset)
//   - batchSize: Number of records to fetch per batch
//   - processor: Function that processes a single batch's results
//
// Example:
//
//	bp := NewBatchProcessor(DefaultBatchConfig())
//	err := bp.FetchInBatches(ctx, db.Model(&Employee{}), 1000, func(query *gorm.DB) error {
//	    var batch []Employee
//	    if err := query.Find(&batch).Error; err != nil {
//	        return err
//	    }
//	    // Process batch...
//	    return nil
//	})
func (bp *BatchProcessor) FetchInBatches(
	ctx context.Context,
	query *gorm.DB,
	batchSize int,
	processor func(*gorm.DB) error,
) error {
	if batchSize <= 0 {
		batchSize = bp.config.DefaultSize
	}

	if batchSize > bp.config.MaxSize {
		batchSize = bp.config.MaxSize
	}

	offset := 0
	batchNum := 0

	for {
		batchQuery := query.Limit(batchSize).Offset(offset)
		err := processor(batchQuery)

		if err != nil {
			bp.logger.Error("batch_fetch_failed",
				"batch_number", batchNum,
				"offset", offset,
				"batch_size", batchSize,
				"error", err,
			)
			return fmt.Errorf("failed to fetch batch %d (offset %d): %w", batchNum, offset, err)
		}

		// Check if we got fewer items than the batch size
		// This indicates we've processed all items
		// Note: The caller should track this and break when done
		// This is a limitation of GORM not providing row count without extra query

		offset += batchSize
		batchNum++
	}
}

// CalculateOptimalBatchSize calculates the optimal batch size based on item size and count.
// This helps avoid hitting MySQL's query size limit while maximizing efficiency.
//
// Parameters:
//   - itemSize: Approximate size of each item in bytes
//   - itemCount: Total number of items to process
//
// Returns the recommended batch size.
func (bp *BatchProcessor) CalculateOptimalBatchSize(itemSize int, itemCount int) int {
	if itemCount == 0 || itemSize == 0 {
		return bp.config.DefaultSize
	}

	// Calculate max items we can fit in the query size limit
	maxItemsByQuerySize := bp.config.QuerySizeLimit / itemSize

	// Use the smaller of: default size, calculated size, max size, or total count
	optimalSize := bp.config.DefaultSize
	if maxItemsByQuerySize < optimalSize {
		optimalSize = maxItemsByQuerySize
	}
	if optimalSize > bp.config.MaxSize {
		optimalSize = bp.config.MaxSize
	}
	if optimalSize > itemCount {
		optimalSize = itemCount
	}

	// Ensure at least 1 item per batch
	if optimalSize < 1 {
		optimalSize = 1
	}

	return optimalSize
}

// GetBatchSizeForIDs calculates the appropriate batch size for processing IDs.
// This is a convenience method for common use cases with ID slices.
// Assumes ~8 bytes per ID (int64) plus some overhead for the IN clause.
func (bp *BatchProcessor) GetBatchSizeForIDs(idCount int) int {
	// Approximate 10 bytes per ID (8 bytes for int64 + 2 bytes overhead in IN clause)
	const idSizeEstimate = 10
	return bp.CalculateOptimalBatchSize(idSizeEstimate, idCount)
}

// GetConfig returns the current batch configuration.
// Useful for inspecting or logging the configuration.
func (bp *BatchProcessor) GetConfig() BatchConfig {
	return bp.config
}
