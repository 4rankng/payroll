package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkTransferBatchRepository is the GORM implementation of
// domain.BulkTransferBatchRepository. Persists rows in bulk_transfer_batches.
type BulkTransferBatchRepository struct {
	*BaseRepository
}

// NewBulkTransferBatchRepository wires a repository onto the shared DB.
// Returns the domain interface so service-layer wiring depends on the port.
func NewBulkTransferBatchRepository(db *Database) domain.BulkTransferBatchRepository {
	return &BulkTransferBatchRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new bulk_transfer_batches row.
func (r *BulkTransferBatchRepository) Create(ctx context.Context, b *domain.BulkTransferBatch) error {
	if err := r.DB.WithContext(ctx).Create(b).Error; err != nil {
		return fmt.Errorf("bulk_transfer_batches: create: %w", err)
	}
	return nil
}

// GetByID returns the row with the given primary key, or ErrBulkTransferBatchNotFound.
func (r *BulkTransferBatchRepository) GetByID(ctx context.Context, id uint64) (*domain.BulkTransferBatch, error) {
	var b domain.BulkTransferBatch
	if err := r.DB.WithContext(ctx).First(&b, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBulkTransferBatchNotFound
		}
		return nil, fmt.Errorf("bulk_transfer_batches: get by id: %w", err)
	}
	return &b, nil
}

// GetByContentHash looks up by the SHA-256 content hash. Returns
// ErrBulkTransferBatchNotFound when no batch matches — the caller treats this
// as "no duplicate detected" and proceeds with INSERT.
func (r *BulkTransferBatchRepository) GetByContentHash(ctx context.Context, hash string) (*domain.BulkTransferBatch, error) {
	var b domain.BulkTransferBatch
	if err := r.DB.WithContext(ctx).First(&b, "content_hash = ?", hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrBulkTransferBatchNotFound
		}
		return nil, fmt.Errorf("bulk_transfer_batches: get by content_hash: %w", err)
	}
	return &b, nil
}

// Update saves all columns on b. Used by book_batch_ledger and recovery crons.
func (r *BulkTransferBatchRepository) Update(ctx context.Context, b *domain.BulkTransferBatch) error {
	if err := r.DB.WithContext(ctx).Save(b).Error; err != nil {
		return fmt.Errorf("bulk_transfer_batches: update: %w", err)
	}
	return nil
}

// UpdateWithLock acquires SELECT ... FOR UPDATE on the batch row, runs fn
// (which may mutate SuccessCount/FailedCount/Status and decide whether to
// transition to `completing`), saves, and returns whether the caller should
// enqueue the book_batch_ledger task.
//
// Returns (batch, shouldBook, err) — NOT a sentinel error for the
// "no booking needed" path, because GORM rolls back the transaction on any
// non-nil error from fn. The bool return lets the caller distinguish
// "completed earlier by another worker" from "this worker should book".
//
// The ledger write itself happens OUTSIDE this transaction (in the
// book_batch_ledger asynq task) so we never hold the row lock across an
// external call.
func (r *BulkTransferBatchRepository) UpdateWithLock(
	ctx context.Context,
	id uint64,
	fn func(*domain.BulkTransferBatch) (shouldBook bool, err error),
) (*domain.BulkTransferBatch, bool, error) {
	var (
		shouldBook bool
		result     *domain.BulkTransferBatch
	)
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var b domain.BulkTransferBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
			First(&b, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrBulkTransferBatchNotFound
			}
			return fmt.Errorf("lock batch: %w", err)
		}
		book, fnErr := fn(&b)
		if fnErr != nil {
			return fnErr // rolls back
		}
		if err := tx.Save(&b).Error; err != nil {
			return fmt.Errorf("save batch: %w", err)
		}
		shouldBook = book
		result = &b
		return nil // commits
	})
	if err != nil {
		return nil, false, err
	}
	return result, shouldBook, nil
}

// UpdateEnqueueState flips the batch-level outbox state. Used by the upload
// pipeline after all per-row tasks are enqueued, and by the stale-enqueue
// sweeper during recovery.
func (r *BulkTransferBatchRepository) UpdateEnqueueState(ctx context.Context, id uint64, state domain.BulkTransferBatchEnqueueState) error {
	if err := r.DB.WithContext(ctx).
		Model(&domain.BulkTransferBatch{}).
		Where("id = ?", id).
		Update("enqueue_state", string(state)).Error; err != nil {
		return fmt.Errorf("bulk_transfer_batches: update enqueue_state: %w", err)
	}
	return nil
}

// List returns batches matching the filter, ordered by created_at DESC.
// Also returns the total count (ignoring Limit/Offset) for pagination.
func (r *BulkTransferBatchRepository) List(ctx context.Context, filter domain.BulkTransferBatchFilter) ([]*domain.BulkTransferBatch, int64, error) {
	q := r.DB.WithContext(ctx).Model(&domain.BulkTransferBatch{})
	if filter.Status != "" {
		q = q.Where("status = ?", string(filter.Status))
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at < ?", *filter.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("bulk_transfer_batches: count: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var batches []*domain.BulkTransferBatch
	if err := q.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&batches).Error; err != nil {
		return nil, 0, fmt.Errorf("bulk_transfer_batches: list: %w", err)
	}
	return batches, total, nil
}

// ListByEnqueueState returns batches with the given outbox state created
// before olderThan. Used by the stale-enqueue sweeper to recover batches
// whose enqueue crashed mid-flight.
func (r *BulkTransferBatchRepository) ListByEnqueueState(ctx context.Context, state domain.BulkTransferBatchEnqueueState, olderThan time.Time, limit int) ([]*domain.BulkTransferBatch, error) {
	if limit <= 0 {
		limit = 50
	}
	var batches []*domain.BulkTransferBatch
	if err := r.DB.WithContext(ctx).
		Where("enqueue_state = ? AND created_at < ?", string(state), olderThan).
		Order("created_at ASC").
		Limit(limit).
		Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("bulk_transfer_batches: list by enqueue_state: %w", err)
	}
	return batches, nil
}

// ListByStatusAndOlderThan returns batches with the given status older than
// the cutoff. Used by the completing-batch recovery cron to re-attempt ledger
// booking for batches stuck in `completing` >10min.
func (r *BulkTransferBatchRepository) ListByStatusAndOlderThan(ctx context.Context, status domain.BulkTransferBatchStatus, olderThan time.Time, limit int) ([]*domain.BulkTransferBatch, error) {
	if limit <= 0 {
		limit = 50
	}
	var batches []*domain.BulkTransferBatch
	if err := r.DB.WithContext(ctx).
		Where("status = ? AND updated_at < ?", string(status), olderThan).
		Order("updated_at ASC").
		Limit(limit).
		Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("bulk_transfer_batches: list by status+older: %w", err)
	}
	return batches, nil
}
