package wallet_bulk

import (
	"encoding/json"
	"fmt"
	"time"

	asynqlib "github.com/hibiken/asynq"
)

// Asynq task type strings. Live here (next to the worker) rather than in
// internal/infra/asynq/handlers.go so the wallet_bulk package is self-contained
// and can be tested without importing the full infra layer.
const (
	// TaskBulkTransferRow drives ONE row through the full 5-step OnePay
	// transfer flow. One task per parsed row, enqueued by Upload.
	TaskBulkTransferRow = "wallet:bulk_transfer_row"
	// TaskBookBatchLedger books payroll receivable/cash-out accounting,
	// successful-timesheet links, and the aggregate OnePay fee.
	TaskBookBatchLedger = "wallet:book_batch_ledger"
	// TaskStaleEnqueueSweeper is the periodic cron (@every 1m) that
	// re-enqueues per-row tasks for batches whose outbox state is still
	// 'pending' (worker crashed between batch INSERT and enqueue).
	TaskStaleEnqueueSweeper = "wallet:bulk_stale_enqueue_sweeper"
	// TaskCompletingRecovery is the periodic cron (@every 5m) that
	// re-attempts ledger booking for batches stuck in 'completing' >10min.
	TaskCompletingRecovery = "wallet:bulk_completing_recovery"
)

// RowTaskPayload is the wire format for TaskBulkTransferRow. Carries the
// FULL BulkTransferRow so the worker doesn't need to re-read batch.data.
type RowTaskPayload struct {
	BatchID uint64          `json:"batch_id"`
	Row     BulkTransferRow `json:"row"`
}

// BookLedgerPayload is the wire format for TaskBookBatchLedger.
type BookLedgerPayload struct {
	BatchID uint64 `json:"batch_id"`
}

// BulkTransferEnqueuer is the port the service uses to dispatch asynq tasks.
// Implemented by the real asynq.Client wrapper AND the test SyncDispatcher.
type BulkTransferEnqueuer interface {
	EnqueueBulkTransferRow(payload RowTaskPayload) error
	EnqueueBookBatchLedger(payload BookLedgerPayload) error
}

// AuditEventEmitter is the narrow port for emitting audit log entries.
// Implemented by the asynq client wrapper (EnqueueAuditLogWrite); nil-safe
// in tests. Required for C3 fix — every upload emits an audit row.
type AuditEventEmitter interface {
	EnqueueAuditLogWrite(p AuditLogPayload) error
}

// AuditLogPayload mirrors asynq.AuditLogWritePayload but lives in the
// wallet_bulk package to keep the service decoupled from infra.
type AuditLogPayload struct {
	UserID       uint      `json:"user_id"`
	Action       string    `json:"action"`
	EntityType   string    `json:"entity_type"`
	EntityID     *uint     `json:"entity_id,omitempty"`
	Message      string    `json:"message"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// RowTaskHandler is the port the asynq mux calls into. Implemented by the
// worker; defined here so the mux doesn't depend on the concrete struct.
type RowTaskHandler interface {
	ProcessJob(ctx interface{}, t *asynqlib.Task) error
}

// mustJSON marshals p and panics on failure. Payload types are tiny structs
// with no custom marshaling — a failure here is a programmer error.
func mustJSON(p any) []byte {
	b, err := json.Marshal(p)
	if err != nil {
		panic(fmt.Sprintf("wallet_bulk: marshal payload: %v", err))
	}
	return b
}

// MaxRowRetry caps how many times the stale-enqueue sweeper will re-enqueue
// a single row. After this, the row is left for manual inspection.
const MaxRowRetry = 3

// Default task scheduling hints. The real asynq client uses these; the test
// SyncDispatcher ignores them.
var (
	DefaultRowTaskTimeout    = 2 * time.Minute
	DefaultLedgerTaskTimeout = 30 * time.Second
)
