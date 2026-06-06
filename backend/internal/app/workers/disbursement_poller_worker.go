package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
)

const (
	// TaskDisbursementPoller is the periodic task type for the disbursement poller.
	TaskDisbursementPoller = "disbursement:poller"
	// TaskDisbursementExecute is the per-request task type for individual disbursements.
	TaskDisbursementExecute = "disbursement:execute"

	pollerBatchSize = 50

	// insufficientBalanceCooldown is the minimum interval between consecutive
	// "insufficient wallet balance" notifications. Prevents notification spam
	// when the poller reclaims the same pending requests every ~20s cycle.
	insufficientBalanceCooldown = 24 * time.Hour
)

// DisbursementPollerWorker is a periodic task that claims PENDING advance
// payment requests and enqueues individual disbursement:execute tasks for
// each one. It does NOT make HTTP calls — that is the execute worker's job.
//
// It also runs orphan recovery: APPROVED requests with no wallet_payment
// that were approved more than 5 minutes ago.
type DisbursementPollerWorker struct {
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository
	advancePaymentRepo    domain.AdvancePaymentRepository
	employeeRepo          domain.EmployeeRepository
	walletPaymentService  *disbursement.WalletPaymentService
	registry              *disbursement.Registry
	asynqClient           *asynqlib.Client
	walletSvc             wallet.WalletService
	userRepo              domain.UserRepository
	emailSvc              *notification.EmailService
	notifications         *notification.NotificationService
	logger                *slog.Logger

	// Circuit breaker: prevents notification spam when wallet stays insufficient
	// across multiple poller cycles (~20s each).
	insufficientBalanceMu    sync.Mutex
	insufficientBalanceNotif time.Time // last time an insufficient-balance notification was sent
}

func NewDisbursementPollerWorker(
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	employeeRepo domain.EmployeeRepository,
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	asynqClient *asynqlib.Client,
	walletSvc wallet.WalletService,
	userRepo domain.UserRepository,
	emailSvc *notification.EmailService,
	notifications *notification.NotificationService,
	logger *slog.Logger,
) *DisbursementPollerWorker {
	return &DisbursementPollerWorker{
		advancePaymentReqRepo: advancePaymentReqRepo,
		advancePaymentRepo:    advancePaymentRepo,
		employeeRepo:          employeeRepo,
		walletPaymentService:  walletPaymentService,
		registry:              registry,
		asynqClient:           asynqClient,
		walletSvc:             walletSvc,
		userRepo:              userRepo,
		emailSvc:              emailSvc,
		notifications:         notifications,
		logger:                logger,
	}
}

// ProcessJob is called by the asynq periodic scheduler every ~20 seconds.
func (w *DisbursementPollerWorker) ProcessJob(ctx context.Context) error {
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return nil // no provider configured — no-op
	}
	w.logger.Debug("disbursement poller: using provider", "name", provider.Name())

	// Step 1: Claim new PENDING requests
	claimed, err := w.advancePaymentReqRepo.ClaimPendingForDisbursement(ctx, pollerBatchSize)
	if err != nil {
		return fmt.Errorf("disbursement poller: claim: %w", err)
	}

	// Step 1.5: All-or-nothing balance gate
	if len(claimed) > 0 && w.walletSvc != nil {
		balance, balErr := w.walletSvc.GetBalance(ctx)
		if balErr != nil {
			w.logger.Warn("disbursement poller: failed to get balance, proceeding without check", "error", balErr)
		} else {
			var totalAmount int64
			for _, req := range claimed {
				totalAmount += int64(req.NetAmount)
			}
			if balance.Available < totalAmount {
				w.logger.Warn("disbursement poller: skipping batch — wallet balance insufficient",
					"available", balance.Available, "needed", totalAmount, "claimed", len(claimed))

				// Release all claimed requests back to PENDING
				ids := make([]uint64, len(claimed))
				for i, req := range claimed {
					ids[i] = uint64(req.ID)
				}
				if resetErr := w.advancePaymentReqRepo.ResetToPending(ctx, ids); resetErr != nil {
					w.logger.Error("disbursement poller: failed to reset claimed requests to pending", "error", resetErr)
				}

				// Notify admins via email
				w.notifyInsufficientBalance(ctx, balance.Available, totalAmount, len(claimed))

				claimed = nil // skip budget validation loop below
			}
		}
	}

	// Step 2: Revalidate budget for each claimed request before enqueue
	// This catches race conditions where multiple requests passed CreateRequest
	// validation but together exceed the limit.
	enqueued := 0
	for _, req := range claimed {
		if err := w.validateBudget(ctx, req); err != nil {
			w.logger.Warn("disbursement poller: rejecting request — budget exceeded",
				"advance_request_id", req.ID, "employee_id", req.EmployeeID, "error", err)
			if err := w.advancePaymentReqRepo.UpdateStatus(ctx, uint64(req.ID), domain.AdvancePaymentStatusFailed, "budget_exceeded", nil); err != nil {
				w.logger.Error("disbursement poller: failed to mark request as budget-exceeded",
					"advance_request_id", req.ID, "error", err)
			}
			continue
		}

		if err := w.enqueueRequest(ctx, req); err != nil {
			w.logger.Error("disbursement poller: failed to enqueue request",
				"advance_request_id", req.ID, "error", err)
		} else {
			enqueued++
		}
	}

	// Step 2: Orphan recovery
	activeProvider, _ := w.registry.ActiveProviderName(ctx)
	orphans, err := w.advancePaymentReqRepo.GetOrphanedApproved(ctx, pollerBatchSize, activeProvider)
	if err != nil {
		w.logger.Warn("disbursement poller: orphan recovery query failed", "error", err)
	} else if len(orphans) > 0 {
		w.logger.Warn("disbursement poller: found orphaned requests",
			"count", len(orphans))
		for _, req := range orphans {
			if err := w.enqueueRequest(ctx, req); err != nil {
				w.logger.Error("disbursement poller: failed to enqueue orphan",
					"advance_request_id", req.ID, "error", err)
			} else {
				enqueued++
			}
		}
	}

	if enqueued > 0 {
		w.logger.Info("disbursement poller: batch complete",
			"claimed", len(claimed), "orphans", len(orphans), "enqueued", enqueued)
	}

	return nil
}

// enqueueRequest validates bank details and enqueues a disbursement:execute task.
// If bank details are missing, the request is marked FAILED immediately.
func (w *DisbursementPollerWorker) enqueueRequest(ctx context.Context, req *domain.AdvancePaymentRequest) error {
	// Load employee with bank details
	employee, err := w.employeeRepo.GetByID(ctx, req.EmployeeID)
	if err != nil {
		return fmt.Errorf("load employee %d: %w", req.EmployeeID, err)
	}

	// Validate bank details — fail fast if missing
	if employee.BankAccountNumber == "" || employee.BankAccountName == "" || employee.BankID == nil || employee.Bank == nil {
		w.logger.Warn("disbursement poller: marking request as FAILED — missing bank details",
			"advance_request_id", req.ID, "employee_id", req.EmployeeID)
		reason := "missing bank details"
		return w.advancePaymentReqRepo.UpdateStatus(ctx, uint64(req.ID), domain.AdvancePaymentStatusFailed, reason, nil)
	}

	// Generate idempotent request ID using tt prefix
	requestID := advance_payment.GenerateTransactionCode(true, true)

	payload := DisbursementExecutePayload{
		RequestID:          requestID,
		AdvanceRequestID:   uint64(req.ID),
		RequestedAmount:    int64(req.NetAmount),
		RecipientName:      employee.BankAccountName,
		RecipientAccountNo: employee.BankAccountNumber,
		RecipientBank:      employee.Bank.BankCode,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	task := asynqlib.NewTask(TaskDisbursementExecute, data)
	if _, err := w.asynqClient.Enqueue(task, asynqlib.MaxRetry(5)); err != nil {
		return fmt.Errorf("enqueue disbursement:execute: %w", err)
	}

	w.logger.Info("disbursement poller: enqueued execute task",
		"advance_request_id", req.ID, "request_id", requestID, "employee_id", req.EmployeeID)

	return nil
}

// validateBudget checks that the employee's total active requests (including
// this one) do not exceed the max advance limit for the current month.
// This is the second line of defense — catches races that leaked past CreateRequest.
func (w *DisbursementPollerWorker) validateBudget(ctx context.Context, req *domain.AdvancePaymentRequest) error {
	currentMonth := advance_payment.GetCurrentMonth()

	maxAdv, err := w.advancePaymentRepo.SumMaxAdvByEmployeeMonth(ctx, uint64(req.EmployeeID), currentMonth)
	if err != nil {
		return fmt.Errorf("validate budget: %w", err)
	}
	if maxAdv == 0 {
		return nil // no limit configured — allow
	}

	// Sum COMPLETED requests only (this request is APPROVED, others may also be APPROVED)
	completed, err := w.advancePaymentReqRepo.SumCompletedByEmployeeMonth(ctx, uint64(req.EmployeeID), currentMonth)
	if err != nil {
		return fmt.Errorf("validate budget: sum completed: %w", err)
	}

	// If this single request + completed exceeds the limit, reject
	if completed+req.RequestAmount > maxAdv {
		return fmt.Errorf("budget exceeded: max=%d, completed=%d, this_request=%d",
			maxAdv, completed, req.RequestAmount)
	}

	return nil
}

// notifyInsufficientBalance sends an email and push notification to all admin
// users when the disbursement poller skips a batch due to insufficient wallet balance.
// It includes a circuit breaker that suppresses duplicate notifications within
// a configurable cooldown window (insufficientBalanceCooldown).
func (w *DisbursementPollerWorker) notifyInsufficientBalance(ctx context.Context, available, needed int64, count int) {
	// Circuit breaker: skip if we already notified within the cooldown window.
	w.insufficientBalanceMu.Lock()
	lastNotif := w.insufficientBalanceNotif
	w.insufficientBalanceMu.Unlock()

	if !lastNotif.IsZero() && time.Since(lastNotif) < insufficientBalanceCooldown {
		w.logger.Info("disbursement poller: suppressing duplicate insufficient-balance notification",
			"last_notif_ago", time.Since(lastNotif).Round(time.Second),
			"cooldown", insufficientBalanceCooldown)
		return
	}

	title := fmt.Sprintf("[TingTing] Cảnh báo: Số dư ví không đủ — %d yêu cầu đang chờ", count)
	body := fmt.Sprintf(
		"Số dư ví không đủ để xử lý các yêu cầu ứng lương.\n\n"+
			"Số dư khả dụng: %d VND\n"+
			"Tổng cần thanh toán: %d VND\n"+
			"Số yêu cầu bị tạm hoãn: %d\n\n"+
			"Vui lòng nạp thêm tiền vào ví để hệ thống tự động xử lý.",
		available, needed, count,
	)

	// Push notification to all admins
	if w.notifications != nil {
		if err := w.notifications.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeCustom, title, body); err != nil {
			w.logger.Error("disbursement poller: failed to send insufficient balance push notification", "error", err)
		}
	}

	// Email to all admins
	if w.userRepo == nil || w.emailSvc == nil {
		w.logger.Warn("disbursement poller: cannot send insufficient balance email — missing userRepo or emailSvc")
		return
	}

	admins, err := w.userRepo.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		w.logger.Error("disbursement poller: failed to list admin users for email", "error", err)
		return
	}

	recipients := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin.Email != nil && *admin.Email != "" {
			recipients = append(recipients, *admin.Email)
		}
	}
	if len(recipients) == 0 {
		w.logger.Warn("disbursement poller: no admin emails found, skipping email")
		return
	}

	if _, emailErr := w.emailSvc.SendGenericEmail(ctx, &dto.SendEmailRequest{
		Recipients: recipients,
		Subject:    title,
		TextBody:   body,
	}); emailErr != nil {
		w.logger.Error("disbursement poller: failed to send insufficient balance email",
			"error", emailErr, "recipients", recipients)
	}

	// Stamp the cooldown after successful send (even if email partially fails,
	// we still don't want to spam push notifications every 20s).
	w.insufficientBalanceMu.Lock()
	w.insufficientBalanceNotif = time.Now()
	w.insufficientBalanceMu.Unlock()
}
