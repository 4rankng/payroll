package asynq

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/dto"
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
func (c *Client) EnqueueImportJob(jobID uint, forMonth string) error {
	payload, _ := json.Marshal(importJobPayload{JobID: jobID, ForMonth: forMonth})

	task := asynqlib.NewTask(TaskImportJob, payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.TaskID(fmt.Sprintf("import-job:%d:%s", jobID, forMonth)),
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
		"for_month", forMonth,
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

// PayrollReportEmailPayload is the async task payload for a payroll statement email.
type PayrollReportEmailPayload struct {
	Request        dto.SendPayrollReportEmailRequest `json:"request"`
	InitiatedBy    uint                              `json:"initiated_by"`
	IdempotencyKey string                            `json:"idempotency_key"`
}

// EnqueuePayrollReportEmail enqueues a payroll report email task and deduplicates
// repeated requests by an idempotency key. If the client supplies an
// Idempotency-Key header, retries of the same HTTP request share the same task;
// otherwise an identical payload from the same user falls back to a deterministic
// key.
func (c *Client) EnqueuePayrollReportEmail(req dto.SendPayrollReportEmailRequest, initiatedBy uint, requestKey string) (string, bool, error) {
	idempotencyKey := buildPayrollReportEmailIdempotencyKey(req, initiatedBy, requestKey)
	payload := PayrollReportEmailPayload{
		Request:        req,
		InitiatedBy:    initiatedBy,
		IdempotencyKey: idempotencyKey,
	}
	raw, _ := json.Marshal(payload)

	task := asynqlib.NewTask(TaskPayrollReportEmail, raw,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(0),
		asynqlib.Timeout(10*time.Minute),
		asynqlib.Retention(24*time.Hour),
		asynqlib.TaskID(idempotencyKey),
		asynqlib.Unique(24*time.Hour),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			logger.Info("Payroll report email task already queued",
				"task_id", idempotencyKey,
				"initiated_by", initiatedBy,
			)
			return idempotencyKey, true, nil
		}
		return "", false, fmt.Errorf("failed to enqueue payroll report email task: %w", err)
	}

	logger.Info("Enqueued payroll report email task",
		"task_id", info.ID,
		"initiated_by", initiatedBy,
		"queue", info.Queue,
	)
	return idempotencyKey, false, nil
}

func buildPayrollReportEmailIdempotencyKey(req dto.SendPayrollReportEmailRequest, initiatedBy uint, requestKey string) string {
	seed := strings.TrimSpace(requestKey)
	if seed == "" {
		seed = strings.Join([]string{
			req.ReportAtDate,
			canonicalEmailList(req.Recipients),
			canonicalEmailList(req.Cc),
			canonicalEmailList(req.Bcc),
		}, "|")
	}

	sum := sha256.Sum256([]byte(fmt.Sprintf("payroll-report-email|%d|%s", initiatedBy, seed)))
	return fmt.Sprintf("payroll-report-email:%x", sum[:16])
}

func canonicalEmailList(values []string) string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	sort.Strings(normalized)
	return strings.Join(normalized, ",")
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

// autoRejectCheckoutPayload is the asynq task payload for attendance:auto_reject.
type autoRejectCheckoutPayload struct {
	AttendanceID uint `json:"attendance_id"`
}

// EnqueueAutoRejectCheckout schedules a one-shot task to fire at `at` — the
// per-attendance checkout deadline K+4h (computed at check-in from the resolved
// shift end). The task auto-rejects the attendance if it still has no checkout
// when it fires. Deduplicated by TaskID per attendance, so repeated enqueues for
// the same attendance collapse to a single scheduled task.
func (c *Client) EnqueueAutoRejectCheckout(attendanceID uint, at time.Time) error {
	payload, _ := json.Marshal(autoRejectCheckoutPayload{AttendanceID: attendanceID})

	task := asynqlib.NewTask(TaskAutoRejectCheckout, payload,
		asynqlib.Queue(QueueDefault),
		asynqlib.MaxRetry(c.cfg.RetryMax),
		asynqlib.ProcessAt(at),
		asynqlib.TaskID(fmt.Sprintf("auto-reject-att:%d", attendanceID)),
	)

	info, err := c.client.Enqueue(task)
	if err != nil {
		if err == asynqlib.ErrDuplicateTask || err == asynqlib.ErrTaskIDConflict {
			return nil
		}
		return fmt.Errorf("failed to enqueue auto-reject checkout task: %w", err)
	}

	logger.Info("Enqueued attendance auto-reject task",
		"task_id", info.ID,
		"attendance_id", attendanceID,
		"fire_at", at.Format(time.RFC3339),
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
	JobID    uint   `json:"job_id"`
	ForMonth string `json:"for_month"`
}
