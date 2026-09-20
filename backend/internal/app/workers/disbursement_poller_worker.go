package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
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

	// maxFailedDisbursementAttempts is the transient-retry budget for one
	// advance payment request. The poller's orphan recovery re-enqueues an
	// APPROVED request whenever its only wallet_payment rows are failed, with a
	// fresh request_id — and every attempt is a provider call that costs the
	// per-transfer fee (3,850 VND on OnePay). Error 86 ("Chức năng tạm thời
	// đóng") once burned 13 attempts before succeeding, so the budget caps the
	// damage at 5 paid retries; past that the request is failed out with reason
	// retry_limit_exceeded instead of being re-enqueued again.
	maxFailedDisbursementAttempts = 5

	// retryLimitExceededReason is written to advance_payment_requests
	// .payment_reference (the same slot as budget_exceeded / missing bank
	// details) so operators can tell a budget give-up apart from a data error.
	retryLimitExceededReason = "retry_limit_exceeded"

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

	// Step 1.5: Greedy balance optimization — sort by amount ascending so we pay
	// as many requests as possible with the available wallet balance.
	var remainingBalance int64
	if len(claimed) > 0 && w.walletSvc != nil {
		balance, balErr := w.walletSvc.GetBalance(ctx)
		if balErr != nil {
			w.logger.Warn("disbursement poller: failed to get balance, proceeding without check", "error", balErr)
			remainingBalance = -1 // sentinel: skip balance check below
		} else {
			remainingBalance = balance.Available
			if balance.Available <= 0 {
				w.logger.Warn("disbursement poller: wallet balance zero or negative — will reset all claimed",
					"available", balance.Available, "claimed", len(claimed))
			}
		}
	}

	// Sort claimed by NetAmount ascending (greedy: smallest first maximizes payment count)
	if len(claimed) > 1 {
		sort.Slice(claimed, func(i, j int) bool {
			return claimed[i].NetAmount < claimed[j].NetAmount
		})
	}

	// Step 2: Revalidate budget for each claimed request and enqueue if wallet
	// balance allows. Requests that fail budget check or exceed remaining balance
	// are reset to PENDING so they can be retried in the next cycle.
	enqueued := 0
	skippedIDs := make([]uint64, 0, len(claimed))
	var totalSkipped int64

	for _, req := range claimed {
		// Check wallet balance for this specific request (skip check if balance unknown)
		if remainingBalance >= 0 && int64(req.NetAmount) > remainingBalance {
			skippedIDs = append(skippedIDs, uint64(req.ID))
			totalSkipped += int64(req.NetAmount)
			continue
		}

		if err := w.validateBudget(ctx, req); err != nil {
			w.logger.Warn("disbursement poller: rejecting request — budget exceeded",
				"advance_request_id", req.ID, "employee_id", req.EmployeeID, "error", err)
			if err := w.advancePaymentReqRepo.UpdateStatus(ctx, uint64(req.ID), domain.AdvancePaymentStatusFailed, "budget_exceeded", nil); err != nil {
				w.logger.Error("disbursement poller: failed to mark request as budget-exceeded",
					"advance_request_id", req.ID, "error", err)
			}
			continue
		}

		enqueuedNow, err := w.enqueueRequest(ctx, req)
		if err != nil {
			w.logger.Error("disbursement poller: failed to enqueue request",
				"advance_request_id", req.ID, "error", err)
			skippedIDs = append(skippedIDs, uint64(req.ID))
			totalSkipped += int64(req.NetAmount)
		} else if enqueuedNow {
			enqueued++
			if remainingBalance >= 0 {
				remainingBalance -= int64(req.NetAmount)
			}
		}
	}

	// Reset skipped/unpaid requests back to PENDING for next cycle
	if len(skippedIDs) > 0 {
		if resetErr := w.advancePaymentReqRepo.ResetToPending(ctx, skippedIDs); resetErr != nil {
			w.logger.Error("disbursement poller: failed to reset skipped requests to pending", "error", resetErr)
		} else {
			w.logger.Info("disbursement poller: reset skipped requests to pending",
				"count", len(skippedIDs))
		}
	}

	// Notify admins if any requests were skipped due to insufficient balance
	if len(skippedIDs) > 0 && remainingBalance >= 0 && w.walletSvc != nil {
		w.notifyInsufficientBalanceSkipped(ctx, remainingBalance, totalSkipped, len(skippedIDs))
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
			enqueuedNow, err := w.enqueueRequest(ctx, req)
			if err != nil {
				w.logger.Error("disbursement poller: failed to enqueue orphan",
					"advance_request_id", req.ID, "error", err)
			} else if enqueuedNow {
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

// enqueueRequest validates the retry budget and bank details, then enqueues a
// disbursement:execute task.
//
// It reports whether a task was actually enqueued. Both terminal give-ups —
// missing bank details and an exhausted retry budget — mark the request FAILED
// here and return (false, nil); they are handled outcomes, not enqueue
// failures, so callers must not count them as skipped (that would try to reset
// the request to PENDING) or spend them against the wallet balance.
func (w *DisbursementPollerWorker) enqueueRequest(ctx context.Context, req *domain.AdvancePaymentRequest) (bool, error) {
	// Retry budget first: a request that has already burned its transient
	// retries must stop costing provider fees even when its bank details are
	// fine. This is the bound on the orphan-recovery loop, which re-enqueues
	// any APPROVED request whose attempts all failed.
	spent, err := w.retryBudgetSpent(ctx, req)
	if err != nil {
		return false, fmt.Errorf("retry budget check: %w", err)
	}
	if spent {
		w.logger.Warn("disbursement poller: retry budget exhausted — marking request FAILED",
			"advance_request_id", req.ID, "employee_id", req.EmployeeID,
			"max_attempts", maxFailedDisbursementAttempts)
		if err := w.advancePaymentReqRepo.UpdateStatus(ctx, uint64(req.ID), domain.AdvancePaymentStatusFailed, retryLimitExceededReason, nil); err != nil {
			return false, fmt.Errorf("mark retry-limit-exceeded request %d FAILED: %w", req.ID, err)
		}
		return false, nil
	}

	// Load employee with bank details
	employee, err := w.employeeRepo.GetByID(ctx, req.EmployeeID)
	if err != nil {
		return false, fmt.Errorf("load employee %d: %w", req.EmployeeID, err)
	}

	// Validate bank details — fail fast if missing
	if employee.BankAccountNumber == "" || employee.BankAccountName == "" || employee.BankID == nil || employee.Bank == nil {
		w.logger.Warn("disbursement poller: marking request as FAILED — missing bank details",
			"advance_request_id", req.ID, "employee_id", req.EmployeeID)
		reason := "missing bank details"
		if err := w.advancePaymentReqRepo.UpdateStatus(ctx, uint64(req.ID), domain.AdvancePaymentStatusFailed, reason, nil); err != nil {
			return false, err
		}
		return false, nil
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
		return false, fmt.Errorf("marshal payload: %w", err)
	}

	task := asynqlib.NewTask(TaskDisbursementExecute, data)
	if _, err := w.asynqClient.Enqueue(task, asynqlib.MaxRetry(5)); err != nil {
		return false, fmt.Errorf("enqueue disbursement:execute: %w", err)
	}

	w.logger.Info("disbursement poller: enqueued execute task",
		"advance_request_id", req.ID, "request_id", requestID, "employee_id", req.EmployeeID)

	return true, nil
}

// retryBudgetSpent reports whether one advance request has already spent its
// transient retry budget — maxFailedDisbursementAttempts wallet_payments rows
// recorded as failed for it. A query error is propagated rather than swallowed:
// without a verdict the caller cannot know whether enqueuing is safe, and
// "unknown" must not read as "budget available" — that is exactly the unbounded
// loop this budget exists to stop.
func (w *DisbursementPollerWorker) retryBudgetSpent(ctx context.Context, req *domain.AdvancePaymentRequest) (bool, error) {
	if w.walletPaymentService == nil || req == nil {
		return false, nil // not wired (dormant deployments / focused tests)
	}
	failed, err := w.walletPaymentService.CountFailedAttempts(ctx, uint64(req.ID))
	if err != nil {
		return false, fmt.Errorf("count failed attempts for request %d: %w", req.ID, err)
	}
	return !withinRetryBudget(failed), nil
}

// withinRetryBudget reports whether a request that already has failedAttempts
// terminal failures may be enqueued again.
func withinRetryBudget(failedAttempts int64) bool {
	return failedAttempts < maxFailedDisbursementAttempts
}

// validateBudget checks that the employee's total active requests (including
// this one) do not exceed the max advance limit for the current month.
// This is the second line of defense — catches races that leaked past CreateRequest.
func (w *DisbursementPollerWorker) validateBudget(ctx context.Context, req *domain.AdvancePaymentRequest) error {
	budgetMonth := requestBudgetMonth(req)

	maxAdv, err := w.advancePaymentRepo.SumMaxAdvByEmployeeMonth(ctx, uint64(req.EmployeeID), budgetMonth)
	if err != nil {
		return fmt.Errorf("validate budget: %w", err)
	}
	if maxAdv == 0 {
		return nil // no limit configured — allow
	}

	// Sum COMPLETED requests only (this request is APPROVED, others may also be APPROVED)
	completed, err := w.advancePaymentReqRepo.SumCompletedByEmployeeMonth(ctx, uint64(req.EmployeeID), budgetMonth)
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

func requestBudgetMonth(req *domain.AdvancePaymentRequest) string {
	if req != nil && req.AdvancePayment != nil && req.AdvancePayment.ForMonth != "" {
		return req.AdvancePayment.ForMonth
	}
	return advance_payment.GetCurrentMonth()
}

// notifyInsufficientBalanceSkipped sends an email and push notification to all
// admin users when requests were skipped due to insufficient wallet balance.
// It includes a circuit breaker that suppresses duplicate notifications within
// a configurable cooldown window (insufficientBalanceCooldown).
func (w *DisbursementPollerWorker) notifyInsufficientBalanceSkipped(ctx context.Context, remainingBalance, skippedAmount int64, skippedCount int) {
	w.insufficientBalanceMu.Lock()
	lastNotif := w.insufficientBalanceNotif
	w.insufficientBalanceMu.Unlock()

	if !lastNotif.IsZero() && time.Since(lastNotif) < insufficientBalanceCooldown {
		return
	}

	title := fmt.Sprintf("[TingTing] Cảnh báo: Số dư ví không đủ — %d yêu cầu bị bỏ qua", skippedCount)
	body := fmt.Sprintf(
		"Số dư ví không đủ thanh toán tất cả các yêu cầu ứng lương.\n\n"+
			"Số dư còn lại: %d VND\n"+
			"Tổng số tiền bị bỏ qua: %d VND\n"+
			"Số yêu cầu bị bỏ qua: %d\n\n"+
			"Vui lòng nạp thêm tiền vào ví để hệ thống tự động xử lý các yêu cầu còn lại.",
		remainingBalance, skippedAmount, skippedCount,
	)

	if w.notifications != nil {
		if err := w.notifications.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeCustom, title, body); err != nil {
			w.logger.Error("disbursement poller: failed to send push notification for skipped requests", "error", err)
		}
	}

	if w.userRepo == nil || w.emailSvc == nil {
		return
	}

	admins, err := w.userRepo.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		return
	}

	recipients := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin.Email != nil && *admin.Email != "" {
			recipients = append(recipients, *admin.Email)
		}
	}
	if len(recipients) == 0 {
		return
	}

	if _, emailErr := w.emailSvc.SendGenericEmail(ctx, &dto.SendEmailRequest{
		Recipients: recipients,
		Subject:    title,
		TextBody:   body,
	}); emailErr != nil {
		w.logger.Error("disbursement poller: failed to send email for skipped requests", "error", emailErr)
	}

	// Stamp the cooldown after successful send (even if email partially fails,
	// we still don't want to spam push notifications every 20s).
	w.insufficientBalanceMu.Lock()
	w.insufficientBalanceNotif = time.Now()
	w.insufficientBalanceMu.Unlock()
}
