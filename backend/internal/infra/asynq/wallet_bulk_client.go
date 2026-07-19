package asynq

import (
	"encoding/json"
	"fmt"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/wallet_bulk"
)

// EnqueueBulkTransferRow enqueues a wallet:bulk_transfer_row task. One task
// per parsed row. Deduplicated by (batch_id, vfic) via TaskID so asynq
// retries and sweeper re-enqueues collapse to a single execution.
func (c *Client) EnqueueBulkTransferRow(payload wallet_bulk.RowTaskPayload) error {
	raw, _ := json.Marshal(payload)
	taskID := fmt.Sprintf("wb:row:%d:%s", payload.BatchID, payload.Row.VFICCode)
	task := asynqlib.NewTask(wallet_bulk.TaskBulkTransferRow, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(wallet_bulk.DefaultRowTaskTimeout),
		asynqlib.TaskID(taskID),
	)
	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			return nil // already enqueued — idempotent
		}
		return fmt.Errorf("enqueue %s: %w", wallet_bulk.TaskBulkTransferRow, err)
	}
	_ = info
	return nil
}

// EnqueueBookBatchLedger enqueues a wallet:book_batch_ledger task. Runs the
// aggregate Expense ledger booking for a batch. Idempotent via the
// ledger_txn_id != nil guard inside the handler.
func (c *Client) EnqueueBookBatchLedger(payload wallet_bulk.BookLedgerPayload) error {
	raw, _ := json.Marshal(payload)
	taskID := fmt.Sprintf("wb:ledger:%d", payload.BatchID)
	task := asynqlib.NewTask(wallet_bulk.TaskBookBatchLedger, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(wallet_bulk.DefaultLedgerTaskTimeout),
		asynqlib.TaskID(taskID),
	)
	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			return nil
		}
		return fmt.Errorf("enqueue %s: %w", wallet_bulk.TaskBookBatchLedger, err)
	}
	_ = info
	return nil
}
