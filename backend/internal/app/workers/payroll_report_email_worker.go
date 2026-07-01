package workers

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/notification"
	auditctx "api-server/internal/pkg/context"
)

// PayrollReportEmailJob is the domain-level async job for sending a payroll
// statement email.
type PayrollReportEmailJob struct {
	Request        dto.SendPayrollReportEmailRequest
	InitiatedBy    uint
	IdempotencyKey string
}

// PayrollReportEmailWorker sends payroll statement emails outside the HTTP path.
type PayrollReportEmailWorker struct {
	emailService       *notification.EmailService
	idempotencyService *infrastructure.IdempotencyService
	logger             *slog.Logger
}

// NewPayrollReportEmailWorker creates a worker for async payroll report emails.
func NewPayrollReportEmailWorker(
	emailService *notification.EmailService,
	idempotencyService *infrastructure.IdempotencyService,
	logger *slog.Logger,
) *PayrollReportEmailWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &PayrollReportEmailWorker{
		emailService:       emailService,
		idempotencyService: idempotencyService,
		logger:             logger.With("component", "PayrollReportEmailWorker"),
	}
}

// ProcessJob sends the email once for an idempotency key. If the email provider
// returns an error, the job records failure and does not ask asynq to retry; the
// provider may already have accepted the message even when our HTTP client saw
// an ambiguous failure.
func (w *PayrollReportEmailWorker) ProcessJob(ctx context.Context, job PayrollReportEmailJob) error {
	if w.emailService == nil || w.idempotencyService == nil {
		return nil
	}
	if job.IdempotencyKey == "" {
		return fmt.Errorf("payroll report email idempotency key is required")
	}

	acquired, state, err := w.idempotencyService.TryAcquireLock(ctx, job.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("email:payroll_report acquire idempotency lock: %w", err)
	}
	if !acquired {
		w.logger.Info("payroll report email already handled",
			"idempotency_key", job.IdempotencyKey,
			"state", state,
		)
		return nil
	}

	if job.InitiatedBy > 0 {
		ctx = auditctx.WithUserID(ctx, job.InitiatedBy)
	}

	messageID, sendErr := w.emailService.SendPayrollReportEmail(ctx, &job.Request)
	if sendErr != nil {
		if err := w.idempotencyService.MarkFailed(ctx, job.IdempotencyKey, sendErr.Error()); err != nil {
			w.logger.Error("failed to mark payroll report email as failed",
				"idempotency_key", job.IdempotencyKey,
				"error", err,
			)
		}
		w.logger.Error("payroll report email failed",
			"idempotency_key", job.IdempotencyKey,
			"error", sendErr,
		)
		return nil
	}

	if err := w.idempotencyService.MarkCompleted(ctx, job.IdempotencyKey); err != nil {
		w.logger.Error("failed to mark payroll report email as completed",
			"idempotency_key", job.IdempotencyKey,
			"message_id", messageID,
			"error", err,
		)
		return nil
	}

	w.logger.Info("payroll report email sent",
		"idempotency_key", job.IdempotencyKey,
		"message_id", messageID,
	)
	return nil
}
