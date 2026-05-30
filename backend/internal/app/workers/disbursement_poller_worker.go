package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
)

const (
	// TaskDisbursementPoller is the periodic task type for the disbursement poller.
	TaskDisbursementPoller = "disbursement:poller"
	// TaskDisbursementExecute is the per-request task type for individual disbursements.
	TaskDisbursementExecute = "disbursement:execute"

	pollerBatchSize = 50
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
	logger                *slog.Logger
}

func NewDisbursementPollerWorker(
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	employeeRepo domain.EmployeeRepository,
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	asynqClient *asynqlib.Client,
	logger *slog.Logger,
) *DisbursementPollerWorker {
	return &DisbursementPollerWorker{
		advancePaymentReqRepo: advancePaymentReqRepo,
		advancePaymentRepo:    advancePaymentRepo,
		employeeRepo:          employeeRepo,
		walletPaymentService:  walletPaymentService,
		registry:              registry,
		asynqClient:           asynqClient,
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
