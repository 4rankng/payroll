// Package wallet_bulk_test hosts the synchronous dispatcher used by the
// integration test suite. _test.go suffix prevents production compilation
// (red-team v2 Security H6 — the codebase has zero build-tag precedent).
package wallet_bulk

import (
	"context"

	asynqlib "github.com/hibiken/asynq"
)

// SyncDispatcher is a test-only BulkTransferEnqueuer that synchronously
// invokes the worker / service handlers in-process. No Redis, no asynq boot.
// Used by Phase 6 integration tests to drive the full pipeline deterministically.
type SyncDispatcher struct {
	RowWorker      RowWorkerAdapter
	BookLedgerSvc  BookLedgerAdapter
	RowCalls       int
	BookLedgerCalls int
}

// RowWorkerAdapter is the narrow port for the row worker's ProcessJob.
// Implemented by *workers.WalletBulkTransferRowWorker; declared here to
// avoid importing the workers package (test-only).
type RowWorkerAdapter interface {
	ProcessRowTask(ctx context.Context, t *asynqlib.Task) error
}

// BookLedgerAdapter is the narrow port for ProcessBookBatchLedger.
type BookLedgerAdapter interface {
	ProcessBookBatchLedgerTask(ctx context.Context, t *asynqlib.Task) error
}

// EnqueueBulkTransferRow synchronously invokes the row worker.
func (d *SyncDispatcher) EnqueueBulkTransferRow(payload RowTaskPayload) error {
	d.RowCalls++
	if d.RowWorker == nil {
		return nil
	}
	task := asynqlib.NewTask(TaskBulkTransferRow, mustJSON(payload))
	return d.RowWorker.ProcessRowTask(context.Background(), task)
}

// EnqueueBookBatchLedger synchronously invokes the book-ledger handler.
func (d *SyncDispatcher) EnqueueBookBatchLedger(payload BookLedgerPayload) error {
	d.BookLedgerCalls++
	if d.BookLedgerSvc == nil {
		return nil
	}
	task := asynqlib.NewTask(TaskBookBatchLedger, mustJSON(payload))
	return d.BookLedgerSvc.ProcessBookBatchLedgerTask(context.Background(), task)
}
