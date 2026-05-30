package asynq

import (
	"encoding/json"
	"fmt"
	"time"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/config"
)

// Client wraps asynq.Client for enqueueing tasks
type Client struct {
	client *asynqlib.Client
	cfg    config.AsynqConfig
}

// AsynqClient returns the underlying asynq client for direct task enqueueing.
func (c *Client) AsynqClient() *asynqlib.Client { return c.client }

// NewClient creates a new Asynq client
func NewClient(cfg config.AsynqConfig) (*Client, error) {
	client := asynqlib.NewClient(asynqlib.RedisClientOpt{
		Addr: cfg.RedisAddr,
		DB:   cfg.RedisDB,
	})

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// EnqueueEmployeeImport enqueues an employee import task
func (c *Client) EnqueueEmployeeImport(importID string) error {
	payload, _ := json.Marshal(employeeImportPayload{ImportID: importID})

	task := asynqlib.NewTask(TaskEmployeeImport, payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Unique(24*time.Hour),
		asynqlib.TaskID(fmt.Sprintf("employee-import:%s", importID)),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask {
			return nil
		}
		return fmt.Errorf("failed to enqueue employee import task: %w", err)
	}

	logger.Info("Enqueued employee import task",
		"task_id", info.ID,
		"import_id", importID,
		"queue", info.Queue,
	)
	return nil
}

// EnqueueImportJob enqueues an import job task (advance payment)
func (c *Client) EnqueueImportJob(jobID uint) error {
	payload, _ := json.Marshal(importJobPayload{JobID: jobID})

	task := asynqlib.NewTask(TaskImportJob, payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.TaskID(fmt.Sprintf("import-job:%d", jobID)),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask {
			return nil
		}
		return fmt.Errorf("failed to enqueue import job task: %w", err)
	}

	logger.Info("Enqueued import job task",
		"task_id", info.ID,
		"job_id", jobID,
		"queue", info.Queue,
	)
	return nil
}

// IPNProcessPayload is the verified, parsed disbursement IPN event the
// webhook handler hands off to the asynq worker. The signature has
// already been validated synchronously in the HTTP path; the worker
// only does the FSM transition and DB write.
//
// Provider is the registry name (e.g. "9pay") so the worker can pick
// the right route through the registry — adding a new provider does not
// require a new asynq task type.
type IPNProcessPayload struct {
	Provider      string `json:"provider"`
	InvoiceNo     string `json:"invoice_no"`
	RequestID     string `json:"request_id"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	RawErrorCode  string `json:"raw_error_code"`
	FailureReason string `json:"failure_reason"`
	IPNRecordID   uint64 `json:"ipn_record_id"`
}

// EnqueueIPNProcess enqueues a verified disbursement IPN event for async
// processing. Idempotency: the task is deduplicated by provider +
// invoice_no + status within a 1-hour window — so a provider's habit of
// redelivering the same IPN multiple times collapses to a single FSM
// transition. The FSM itself is also idempotent against duplicate
// triggers, so the dedupe is belt-and-braces.
//
// Returns nil on duplicate enqueue (deduped) so the caller's webhook
// handler can still respond 200 OK to the provider.
func (c *Client) EnqueueIPNProcess(p IPNProcessPayload) error {
	payload, _ := json.Marshal(p)

	key := p.InvoiceNo
	if key == "" {
		key = p.RequestID
	}

	task := asynqlib.NewTask(TaskIPNProcess, payload,
		asynqlib.Queue(QueueCritical),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(30*time.Second),
		asynqlib.Retention(24*time.Hour),
		asynqlib.TaskID(fmt.Sprintf("ipn:%s:%s:%s", p.Provider, key, p.Status)),
		asynqlib.Unique(time.Hour),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			// Provider redelivered the same IPN — already queued or processed.
			return nil
		}
		return fmt.Errorf("failed to enqueue %s IPN task: %w", p.Provider, err)
	}

	logger.Info("Enqueued disbursement IPN task",
		"task_id", info.ID,
		"provider", p.Provider,
		"invoice_no", p.InvoiceNo,
		"request_id", p.RequestID,
		"status", p.Status,
		"queue", info.Queue,
	)
	return nil
}

// EnqueueBulkTransferTransaction enqueues a task that creates the revenue transaction
// and ledger entries for a parsed bulk transfer result. Deduplicated by EventID within
// 24 hours so a duplicate upload or asynq retry collapses to a single execution.
func (c *Client) EnqueueBulkTransferTransaction(payload bulktransfer.BulkTransferTaskPayload) error {
	return c.enqueueBulkTransferTask(TaskBulkTransferTransaction, "bt:tx", payload, 10*time.Minute)
}

// EnqueueBulkTransferPayment enqueues a task that updates timesheet payment statuses
// for a parsed bulk transfer result. Deduplicated by EventID within 24 hours.
func (c *Client) EnqueueBulkTransferPayment(payload bulktransfer.BulkTransferTaskPayload) error {
	return c.enqueueBulkTransferTask(TaskBulkTransferPayment, "bt:pay", payload, 5*time.Minute)
}

func (c *Client) enqueueBulkTransferTask(taskType, prefix string, payload bulktransfer.BulkTransferTaskPayload, timeout time.Duration) error {
	raw, _ := json.Marshal(payload)

	task := asynqlib.NewTask(taskType, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(timeout),
		asynqlib.TaskID(fmt.Sprintf("%s:%s", prefix, payload.EventID)),
		asynqlib.Unique(24*time.Hour),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			return nil
		}
		return fmt.Errorf("failed to enqueue %s task: %w", taskType, err)
	}

	logger.Info("Enqueued bulk transfer task",
		"task_type", taskType,
		"task_id", info.ID,
		"event_id", payload.EventID,
		"queue", info.Queue,
	)
	return nil
}

// AuditEnqueuer is the minimal interface the audit event handler needs from the asynq client.
// Using an interface here keeps the handler testable without a real Redis connection.
type AuditEnqueuer interface {
	EnqueueAuditLogWrite(p AuditLogWritePayload) error
}

// AuditLogWritePayload is the asynq task payload for audit:log:write.
// The event handler builds it from a domain event; the worker deserialises and persists the row.
// CreatedAt is captured at enqueue time so retries preserve the original action timestamp.
type AuditLogWritePayload struct {
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

// EnqueueAuditLogWrite enqueues a single audit log write task.
// No deduplication: every audit event is a distinct row.
func (c *Client) EnqueueAuditLogWrite(p AuditLogWritePayload) error {
	raw, _ := json.Marshal(p)

	task := asynqlib.NewTask(TaskAuditLogWrite, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.Timeout(10*time.Second),
		asynqlib.Retention(24*time.Hour),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue audit log write task: %w", err)
	}

	logger.Info("Enqueued audit log write task",
		"task_id", info.ID,
		"user_id", p.UserID,
		"action", p.Action,
		"entity_type", p.EntityType,
		"queue", info.Queue,
	)
	return nil
}

// Close closes the asynq client connection
func (c *Client) Close() error {
	return c.client.Close()
}

type employeeImportPayload struct {
	ImportID string `json:"import_id"`
}

type importJobPayload struct {
	JobID uint `json:"job_id"`
}
