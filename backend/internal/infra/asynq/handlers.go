package asynq

import (
	"context"
	"encoding/json"
	"fmt"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/app/workers"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

var logger = observability.GetLogger().With("component", "asynq")

// Task type constants
const (
	TaskEmployeeImport = "employee:import"
	TaskImportJob      = "import:job"
	// TaskIPNProcess is the provider-agnostic task type for verified
	// disbursement IPN events. The wire payload carries the provider
	// name so a single worker handles every disbursement provider.
	TaskIPNProcess = "ipn:process"
	// TaskDisbursementPoller is the periodic task type for the advance
	// payment disbursement poller. Defined here as a re-export of the
	// constant in the workers package to avoid import cycles.
	TaskDisbursementPoller = workers.TaskDisbursementPoller
	// TaskDisbursementExecute is the per-request task type for individual
	// advance payment disbursements.
	TaskDisbursementExecute = workers.TaskDisbursementExecute
	// TaskNinePayBulkExecute is the task type for 9Pay bulk transfer batch execution.
	TaskNinePayBulkExecute = "bulk_transfer:ninepay_execute"
	// TaskBulkTransferTransaction is the asynq task type for creating the
	// revenue transaction and ledger entries after a bulk transfer result upload.
	TaskBulkTransferTransaction = "bulk_transfer:transaction"
	// TaskBulkTransferPayment is the asynq task type for updating timesheet
	// payment statuses after a bulk transfer result upload.
	TaskBulkTransferPayment = "bulk_transfer:payment"
	// TaskAuditLogWrite is the asynq task type for persisting audit log entries.
	// Replaces the in-process event-bus goroutine to add durable retries and bounded concurrency.
	TaskAuditLogWrite = "audit:log:write"
	// TaskPayrollReportEmail is the task type for async payroll statement emails.
	TaskPayrollReportEmail = "email:payroll_report"
	// TaskAutoRejectCheckout is the one-shot task scheduled at an attendance's
	// checkout deadline (K+3h) when the employee checks in. It auto-rejects the
	// attendance if the window closes with no checkout.
	TaskAutoRejectCheckout = "attendance:auto_reject"
	// TaskAutoRejectSweep is the periodic safety-net task that finalizes
	// attendances whose scheduled K+3h task was lost (Redis/process outage at
	// check-in). Runs on a fixed schedule via the asynq scheduler.
	TaskAutoRejectSweep = "attendance:auto_reject_sweep"
	// TaskWalletSettlement is the asynq task type for the EOD wallet settlement cron.
	// Re-exported from workers package.
	TaskWalletSettlement = workers.TaskWalletSettlement
	// TaskStatusInquiry is the periodic task type for polling stuck authorised
	// payments via the provider's inquiry endpoint. Re-exported from workers package.
	TaskStatusInquiry = workers.TaskStatusInquiry
)

// Queue name constants
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// payloadToBulkTransferEvent reconstructs a BulkTransferResultParsedEvent
// from an asynq task payload. The BaseEvent carries only the fields that
// workers actually read (EventName and ActorUserID); audit/IP fields are
// not populated because this is a deferred retry, not an HTTP request.
func payloadToBulkTransferEvent(p bulktransfer.BulkTransferTaskPayload) domain.BulkTransferResultParsedEvent {
	return domain.BulkTransferResultParsedEvent{
		BaseEvent: domain.BaseEvent{
			EventName:   "BulkTransferResultParsed",
			ActorUserID: p.ActorUserID,
		},
		BulkFileID:      p.BulkFileID,
		AssetID:         p.AssetID,
		Filename:        p.Filename,
		TotalTransfers:  p.TotalTransfers,
		CompletedCount:  p.CompletedCount,
		FailedCount:     p.FailedCount,
		TotalAmount:     p.TotalAmount,
		ParsedDataJSON:  p.ParsedDataJSON,
		UpdatedDataJSON: p.UpdatedDataJSON,
		ProcessedBy:     p.ProcessedBy,
		EventID:         p.EventID,
	}
}

// Handlers holds all task handler dependencies
type Handlers struct {
	employeeImportWorker          *workers.EmployeeImportWorker
	importJobWorker               *workers.ImportJobWorker
	ipnProcessWorker              *workers.IPNProcessWorker
	disbursementPollerWorker      *workers.DisbursementPollerWorker
	disbursementExecuteWorker     *workers.DisbursementExecuteWorker
	ninePayBulkExecuteWorker      *workers.NinePayBulkExecuteWorker
	bulkTransferTransactionWorker *workers.BulkTransferTransactionWorker
	bulkTransferPaymentWorker     *workers.BulkTransferPaymentWorker
	auditLogWriteWorker           *workers.AuditLogWriteWorker
	payrollReportEmailWorker      *workers.PayrollReportEmailWorker
	walletSettlementWorker        *workers.WalletSettlementWorker
	statusInquiryPollerWorker     *workers.StatusInquiryPollerWorker
	autoRejectCheckoutWorker      *workers.AutoRejectCheckoutWorker
	autoRejectSweepWorker         *workers.AutoRejectSweepWorker
}

// NewHandlers creates a new Handlers instance
func NewHandlers(
	employeeImportWorker *workers.EmployeeImportWorker,
	importJobWorker *workers.ImportJobWorker,
	ipnProcessWorker *workers.IPNProcessWorker,
	disbursementPollerWorker *workers.DisbursementPollerWorker,
	disbursementExecuteWorker *workers.DisbursementExecuteWorker,
	ninePayBulkExecuteWorker *workers.NinePayBulkExecuteWorker,
	bulkTransferTransactionWorker *workers.BulkTransferTransactionWorker,
	bulkTransferPaymentWorker *workers.BulkTransferPaymentWorker,
	auditLogWriteWorker *workers.AuditLogWriteWorker,
	payrollReportEmailWorker *workers.PayrollReportEmailWorker,
	walletSettlementWorker *workers.WalletSettlementWorker,
	statusInquiryPollerWorker *workers.StatusInquiryPollerWorker,
	autoRejectCheckoutWorker *workers.AutoRejectCheckoutWorker,
	autoRejectSweepWorker *workers.AutoRejectSweepWorker,
) *Handlers {
	return &Handlers{
		employeeImportWorker:          employeeImportWorker,
		importJobWorker:               importJobWorker,
		ipnProcessWorker:              ipnProcessWorker,
		disbursementPollerWorker:      disbursementPollerWorker,
		disbursementExecuteWorker:     disbursementExecuteWorker,
		ninePayBulkExecuteWorker:      ninePayBulkExecuteWorker,
		bulkTransferTransactionWorker: bulkTransferTransactionWorker,
		bulkTransferPaymentWorker:     bulkTransferPaymentWorker,
		auditLogWriteWorker:           auditLogWriteWorker,
		payrollReportEmailWorker:      payrollReportEmailWorker,
		walletSettlementWorker:        walletSettlementWorker,
		statusInquiryPollerWorker:     statusInquiryPollerWorker,
		autoRejectCheckoutWorker:      autoRejectCheckoutWorker,
		autoRejectSweepWorker:         autoRejectSweepWorker,
	}
}

// HandlePayrollReportEmail processes an async payroll report email task.
// Malformed payloads are dropped. The worker itself is idempotent, so duplicate
// tasks or replayed HTTP requests do not send duplicate emails.
func (h *Handlers) HandlePayrollReportEmail(ctx context.Context, t *asynqlib.Task) error {
	var p PayrollReportEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("email:payroll_report unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}
	if h.payrollReportEmailWorker == nil {
		return nil
	}
	return h.payrollReportEmailWorker.ProcessJob(ctx, workers.PayrollReportEmailJob{
		Request:        p.Request,
		InitiatedBy:    p.InitiatedBy,
		IdempotencyKey: p.IdempotencyKey,
	})
}

// HandleEmployeeImport processes employee import tasks
func (h *Handlers) HandleEmployeeImport(ctx context.Context, t *asynqlib.Task) error {
	var p employeeImportPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal employee import payload: %w", err)
	}

	logger.Info("Processing employee import task", "import_id", p.ImportID)

	if err := h.employeeImportWorker.ProcessJob(ctx, p.ImportID); err != nil {
		return fmt.Errorf("employee import failed: %w", err)
	}

	return nil
}

// HandleImportJob processes import job tasks (advance payment)
func (h *Handlers) HandleImportJob(ctx context.Context, t *asynqlib.Task) error {
	var p importJobPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal import job payload: %w", err)
	}

	logger.Info("Processing import job task", "job_id", p.JobID, "for_month", p.ForMonth)

	if err := h.importJobWorker.ProcessJob(ctx, p.JobID, p.ForMonth); err != nil {
		return fmt.Errorf("import job failed: %w", err)
	}

	return nil
}

// HandleIPNProcess processes a verified disbursement IPN event off the
// HTTP path. Verification has already happened in the webhook handler;
// this worker just drives the FSM and persists the row update. The
// payload carries the provider name so a single handler covers every
// disbursement provider.
func (h *Handlers) HandleIPNProcess(ctx context.Context, t *asynqlib.Task) error {
	var p IPNProcessPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// Malformed payload — never going to succeed; tell asynq to drop it.
		logger.Error("ipn:process: unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}

	job := workers.IPNJob{
		Provider:      p.Provider,
		InvoiceNo:     p.InvoiceNo,
		RequestID:     p.RequestID,
		Status:        p.Status,
		Amount:        p.Amount,
		RawErrorCode:  p.RawErrorCode,
		FailureReason: p.FailureReason,
		IPNRecordID:   p.IPNRecordID,
	}
	if err := h.ipnProcessWorker.ProcessJob(ctx, job); err != nil {
		return fmt.Errorf("ipn:process: %w", err)
	}

	return nil
}

// HandleDisbursementPoller processes the periodic disbursement poller task.
func (h *Handlers) HandleDisbursementPoller(ctx context.Context, _ *asynqlib.Task) error {
	if h.disbursementPollerWorker == nil {
		return nil
	}
	return h.disbursementPollerWorker.ProcessJob(ctx)
}

// HandleDisbursementExecute processes an individual disbursement execute task.
func (h *Handlers) HandleDisbursementExecute(ctx context.Context, t *asynqlib.Task) error {
	if h.disbursementExecuteWorker == nil {
		return nil
	}
	return h.disbursementExecuteWorker.ProcessJob(ctx, t)
}

// HandleNinePayBulkExecute processes a 9Pay bulk transfer batch execution task.
func (h *Handlers) HandleNinePayBulkExecute(ctx context.Context, t *asynqlib.Task) error {
	if h.ninePayBulkExecuteWorker == nil {
		return nil
	}
	return h.ninePayBulkExecuteWorker.ProcessJob(ctx, t)
}

// HandleBulkTransferTransaction processes a bulk transfer transaction creation task.
// Reconstructs the BulkTransferResultParsedEvent and delegates to BulkTransferTransactionWorker.
// Asynq provides automatic retries; the worker is idempotent via Redis-backed idempotency locks.
func (h *Handlers) HandleBulkTransferTransaction(ctx context.Context, t *asynqlib.Task) error {
	var p bulktransfer.BulkTransferTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("bulk_transfer:transaction unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}

	event := payloadToBulkTransferEvent(p)
	if err := h.bulkTransferTransactionWorker.Handle(ctx, event); err != nil {
		return fmt.Errorf("bulk_transfer:transaction: %w", err)
	}

	return nil
}

// HandleBulkTransferPayment processes a bulk transfer payment status update task.
// Reconstructs the BulkTransferResultParsedEvent and delegates to BulkTransferPaymentWorker.
// Asynq provides automatic retries; the worker is idempotent via per-transaction-code Redis locks.
func (h *Handlers) HandleBulkTransferPayment(ctx context.Context, t *asynqlib.Task) error {
	var p bulktransfer.BulkTransferTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("bulk_transfer:payment unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}

	event := payloadToBulkTransferEvent(p)
	if err := h.bulkTransferPaymentWorker.Handle(ctx, event); err != nil {
		return fmt.Errorf("bulk_transfer:payment: %w", err)
	}

	return nil
}

// HandleAuditLogWrite processes an audit log write task.
// Malformed payloads are dropped immediately (SkipRetry); DB errors are retried by asynq.
func (h *Handlers) HandleAuditLogWrite(ctx context.Context, t *asynqlib.Task) error {
	var p AuditLogWritePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("audit:log:write unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}

	job := workers.AuditLogWriteJob{
		UserID:       p.UserID,
		Action:       p.Action,
		EntityType:   p.EntityType,
		EntityID:     p.EntityID,
		Message:      p.Message,
		IPAddress:    p.IPAddress,
		UserAgent:    p.UserAgent,
		MetadataJSON: p.MetadataJSON,
		CreatedAt:    p.CreatedAt,
	}
	if err := h.auditLogWriteWorker.ProcessJob(ctx, job); err != nil {
		return fmt.Errorf("audit:log:write: %w", err)
	}

	return nil
}

// HandleWalletSettlement processes the periodic EOD wallet settlement task.
func (h *Handlers) HandleWalletSettlement(ctx context.Context, _ *asynqlib.Task) error {
	if h.walletSettlementWorker == nil {
		return nil
	}
	return h.walletSettlementWorker.ProcessJob(ctx)
}

// HandleStatusInquiry processes the periodic status inquiry poller task.
func (h *Handlers) HandleStatusInquiry(ctx context.Context, _ *asynqlib.Task) error {
	if h.statusInquiryPollerWorker == nil {
		return nil
	}
	return h.statusInquiryPollerWorker.ProcessJob(ctx)
}

// HandleAutoRejectCheckout processes the one-shot attendance:auto_reject task
// scheduled at an attendance's checkout deadline (K+3h). Malformed payloads are
// dropped (SkipRetry); DB errors are retried by asynq. The worker is idempotent,
// so duplicate or retried tasks are safe.
func (h *Handlers) HandleAutoRejectCheckout(ctx context.Context, t *asynqlib.Task) error {
	var p autoRejectCheckoutPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logger.Error("attendance:auto_reject unmarshal payload", "error", err)
		return asynqlib.SkipRetry
	}
	if err := h.autoRejectCheckoutWorker.ProcessJob(ctx, p.AttendanceID); err != nil {
		return fmt.Errorf("attendance:auto_reject: %w", err)
	}
	return nil
}

// HandleAutoRejectSweep processes the periodic attendance:auto_reject_sweep
// task — the fallback that finalizes attendances whose scheduled K+3h task was
// lost. The worker is idempotent, so retries and overlapping runs are safe.
func (h *Handlers) HandleAutoRejectSweep(ctx context.Context, _ *asynqlib.Task) error {
	if h.autoRejectSweepWorker == nil {
		return nil
	}
	if err := h.autoRejectSweepWorker.ProcessJob(ctx); err != nil {
		return fmt.Errorf("attendance:auto_reject_sweep: %w", err)
	}
	return nil
}
