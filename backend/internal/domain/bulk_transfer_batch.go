package domain

import (
	"context"
	"errors"
	"time"
)

// BulkTransferBatchStatus is the lifecycle state of a bulk transfer batch.
// pending → processing → completing → completed
//                              ↘ failed
//
// `completing` is a transient state set inside the lock transaction when all
// rows have reached terminal status; it triggers the book_batch_ledger asynq
// task. If the process dies between the lock commit and the ledger write,
// the completing-recovery cron re-attempts booking.
type BulkTransferBatchStatus string

const (
	BulkTransferBatchStatusPending    BulkTransferBatchStatus = "pending"
	BulkTransferBatchStatusProcessing BulkTransferBatchStatus = "processing"
	BulkTransferBatchStatusCompleting BulkTransferBatchStatus = "completing"
	BulkTransferBatchStatusCompleted  BulkTransferBatchStatus = "completed"
	BulkTransferBatchStatusFailed     BulkTransferBatchStatus = "failed"
)

// BulkTransferBatchEnqueueState is the batch-level outbox state.
// `pending` = batch row committed, per-row asynq tasks NOT yet enqueued
// (crash window). `enqueued` = all per-row tasks enqueued.
type BulkTransferBatchEnqueueState string

const (
	BulkTransferEnqueuePending  BulkTransferBatchEnqueueState = "pending"
	BulkTransferEnqueueEnqueued BulkTransferBatchEnqueueState = "enqueued"
)

// BulkTransferBatch represents one admin upload of a "Yêu cầu chuyển tiền"
// .xlsx file (exported from /admin/timesheet → "Chuyển OnePay"). Its `data`
// JSON column persists the parsed []BulkTransferRow so the outbox sweeper and
// the KQ Excel generator can re-read inputs without re-parsing the Excel.
//
// This entity is separate from BulkTransferFile (the existing 9Pay timesheet-
// driven flow) — they coexist and never share rows.
type BulkTransferBatch struct {
	ID              uint64                    `json:"id" gorm:"primaryKey;type:bigint unsigned;autoIncrement"`
	Filename        string                    `json:"filename" gorm:"column:filename;type:varchar(255);not null"`
	ContentHash     string                    `json:"content_hash" gorm:"column:content_hash;type:char(64);not null;uniqueIndex:uq_bulk_transfer_batches_content_hash"`
	Source          string                    `json:"source" gorm:"column:source;type:varchar(16);not null;default:'wallet_upload'"`
	Status          BulkTransferBatchStatus   `json:"status" gorm:"column:status;type:varchar(32);not null;default:'pending'"`
	EnqueueState    BulkTransferBatchEnqueueState `json:"enqueue_state" gorm:"column:enqueue_state;type:varchar(16);not null;default:'pending'"`
	TotalCount      int                       `json:"total_count" gorm:"column:total_count;not null;default:0"`
	SuccessCount    int                       `json:"success_count" gorm:"column:success_count;not null;default:0"`
	FailedCount     int                       `json:"failed_count" gorm:"column:failed_count;not null;default:0"`
	TransferAmount  int64                     `json:"transfer_amount" gorm:"column:transfer_amount;type:bigint;not null;default:0"`
	TotalFee        int64                     `json:"total_fee" gorm:"column:total_fee;type:bigint;not null;default:0"`
	LedgerTxnID     *uint64                   `json:"ledger_txn_id,omitempty" gorm:"column:ledger_txn_id;type:bigint unsigned"`
	FeeBookedAt     *time.Time                `json:"fee_booked_at,omitempty" gorm:"column:fee_booked_at;type:datetime(3)"`
	Data            string                    `json:"data" gorm:"column:data;type:json;not null"`
	AssetID         *uint64                   `json:"asset_id,omitempty" gorm:"column:asset_id;type:bigint unsigned"`
	CreatedBy       uint64                    `json:"created_by" gorm:"column:created_by;not null;type:bigint unsigned"`
	CreatedAt       time.Time                 `json:"created_at" gorm:"column:created_at;type:datetime(3);not null;autoCreateTime"`
	UpdatedAt       time.Time                 `json:"updated_at" gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime"`
	CompletedAt     *time.Time                `json:"completed_at,omitempty" gorm:"column:completed_at;type:datetime(3)"`
}

// TableName binds the entity to its physical table.
func (BulkTransferBatch) TableName() string { return "bulk_transfer_batches" }

// IsTerminal reports whether the batch has reached a final state.
func (b *BulkTransferBatch) IsTerminal() bool {
	return b.Status == BulkTransferBatchStatusCompleted ||
		b.Status == BulkTransferBatchStatusFailed
}

// BulkTransferBatchFilter carries the optional list filters.
type BulkTransferBatchFilter struct {
	Status  BulkTransferBatchStatus
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

// BulkTransferBatchRepository persists bulk_transfer_batches rows.
//
// UpdateWithLock is the atomic completion detector: it acquires a row lock,
// runs fn (which may bump SuccessCount/FailedCount and decide whether to
// transition to `completing`), saves the row, and returns whether the caller
// should enqueue the book_batch_ledger task. Returning a bool (not a sentinel
// error) lets GORM roll back cleanly on real errors without conflating the
// "no booking needed" signal with a failure.
type BulkTransferBatchRepository interface {
	Create(ctx context.Context, b *BulkTransferBatch) error
	GetByID(ctx context.Context, id uint64) (*BulkTransferBatch, error)
	GetByContentHash(ctx context.Context, hash string) (*BulkTransferBatch, error)
	Update(ctx context.Context, b *BulkTransferBatch) error
	UpdateWithLock(ctx context.Context, id uint64, fn func(*BulkTransferBatch) (shouldBook bool, err error)) (*BulkTransferBatch, bool, error)
	UpdateEnqueueState(ctx context.Context, id uint64, state BulkTransferBatchEnqueueState) error
	// DecrementTotalCount atomically subtracts n from total_count. Used by
	// the row worker when a row fails BEFORE a wallet_payment is created
	// (ErrDuplicatePaymentInProgress, ErrFeeResolution, pre-flight balance
	// rejection). Without this, those rows would never be counted as terminal
	// and the batch would never reach the success+failed==total threshold.
	// Guarded: refuses to push total_count below success_count+failed_count.
	DecrementTotalCount(ctx context.Context, id uint64, n int) error
	List(ctx context.Context, filter BulkTransferBatchFilter) ([]*BulkTransferBatch, int64, error)
	ListByEnqueueState(ctx context.Context, state BulkTransferBatchEnqueueState, olderThan time.Time, limit int) ([]*BulkTransferBatch, error)
	ListByStatusAndOlderThan(ctx context.Context, status BulkTransferBatchStatus, olderThan time.Time, limit int) ([]*BulkTransferBatch, error)
}

// BulkTransferBatch sentinel errors.
var (
	ErrBulkTransferBatchNotFound = errors.New("bulk_transfer_batch: not found")
)
