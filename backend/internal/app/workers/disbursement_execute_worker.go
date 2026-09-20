package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
)

// DisbursementExecutePayload is the wire format for disbursement:execute tasks.
type DisbursementExecutePayload struct {
	RequestID          string `json:"request_id"`
	AdvanceRequestID   uint64 `json:"advance_request_id"`
	RequestedAmount    int64  `json:"requested_amount"`
	RecipientName      string `json:"recipient_name"`
	RecipientAccountNo string `json:"recipient_account_no"`
	RecipientBank      string `json:"recipient_bank"`
}

// DisbursementExecuteWorker processes individual disbursement tasks enqueued
// by the poller. It makes the HTTP calls to 9Pay (check-account + create-transfer)
// and drives the wallet_payment FSM through each transition.
type DisbursementExecuteWorker struct {
	walletPaymentService *disbursement.WalletPaymentService
	registry             *disbursement.Registry
	bankRepo             domain.BankRepository
	walletSvc            wallet.WalletService
	advanceReqRepo       domain.AdvancePaymentRequestRepository
	employeeNotifier     notification.EmployeeNotifier
	logger               *slog.Logger
}

func NewDisbursementExecuteWorker(
	walletPaymentService *disbursement.WalletPaymentService,
	registry *disbursement.Registry,
	bankRepo domain.BankRepository,
	walletSvc wallet.WalletService,
	advanceReqRepo domain.AdvancePaymentRequestRepository,
	employeeNotifier notification.EmployeeNotifier,
	logger *slog.Logger,
) *DisbursementExecuteWorker {
	return &DisbursementExecuteWorker{
		walletPaymentService: walletPaymentService,
		registry:             registry,
		bankRepo:             bankRepo,
		walletSvc:            walletSvc,
		advanceReqRepo:       advanceReqRepo,
		employeeNotifier:     employeeNotifier,
		logger:               logger,
	}
}

// resolveSwiftCode looks up the SwiftCode for a bank via the bank repository.
// Returns a non-nil error when the repo itself fails (DB timeout, connection
// error) so that the caller can propagate a retryable error to asynq.
// Returns ("", nil) when the bank exists but has no SwiftCode, or when
// bankRepo is nil (graceful degradation).
func (w *DisbursementExecuteWorker) resolveSwiftCode(ctx context.Context, bankCode string) (string, error) {
	if isSwiftCode(bankCode) {
		return bankCode, nil
	}
	if w.bankRepo == nil {
		return "", nil
	}
	bank, err := w.bankRepo.FindByBankCode(ctx, bankCode)
	if err != nil {
		return "", fmt.Errorf("resolve swift code for bank %q: %w", bankCode, err)
	}
	return bank.SwiftCode, nil
}

// ProcessJob handles a single disbursement:execute task.
func (w *DisbursementExecuteWorker) ProcessJob(ctx context.Context, t *asynqlib.Task) error {
	var p DisbursementExecutePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		w.logger.Error("disbursement execute: malformed payload", "error", err)
		return fmt.Errorf("unmarshal payload: %w", asynqlib.SkipRetry)
	}

	logger := w.logger.With(
		"advance_request_id", p.AdvanceRequestID,
		"request_id", p.RequestID,
	)

	// A retry that finds its own persisted pending row is resuming funds already
	// counted in wallet.Available. Do not deduct the same reservation a second
	// time or a lost pre-transfer task can never recover after the balance
	// changes. Verified rows are intentionally excluded: they are no longer
	// reserved in the wallet balance and must pass a fresh funds check before a
	// provider-not-found resend.
	resumingExisting := false
	if w.walletPaymentService != nil {
		existing, lookupErr := w.walletPaymentService.GetByRequestID(ctx, p.RequestID)
		switch {
		case lookupErr == nil && existing != nil && existing.Status == domaintx.StatePending:
			resumingExisting = true
		case lookupErr != nil && !errors.Is(lookupErr, domaintx.ErrNotFound):
			return fmt.Errorf("lookup existing wallet_payment: %w", lookupErr)
		}
	}

	// Step 0: Pre-flight balance check (safety net for races between poller and execute)
	if w.walletSvc != nil && !resumingExisting {
		balance, balErr := w.walletSvc.GetBalance(ctx)
		if balErr != nil {
			logger.Warn("disbursement execute: balance check failed, proceeding", "error", balErr)
		} else if balance.Available < p.RequestedAmount {
			logger.Info("disbursement execute: skipping — insufficient balance",
				"available", balance.Available, "requested", p.RequestedAmount)
			// Do NOT create wallet_payment row. Return nil so asynq doesn't retry.
			// The orphan recovery in the poller will re-enqueue after balance is topped up.
			return nil
		}
	}

	// Step 1: Initiate wallet_payment (idempotent — returns existing row if retry)
	row, err := w.walletPaymentService.Initiate(ctx, disbursement.InitiateInput{
		RequestID:          p.RequestID,
		RequestedAmount:    p.RequestedAmount,
		RecipientName:      p.RecipientName,
		RecipientAccountNo: p.RecipientAccountNo,
		RecipientBank:      p.RecipientBank,
		EntityID:           &p.AdvanceRequestID,
	})
	if err != nil {
		if errors.Is(err, disbursement.ErrDuplicatePaymentInProgress) {
			logger.Info("disbursement execute: skipping — duplicate payment already in progress for this recipient",
				"account_no", p.RecipientAccountNo, "bank", p.RecipientBank)
			return nil // terminal — don't retry; admin or another worker is handling it
		}
		if errors.Is(err, disbursement.ErrFeeResolution) {
			// Missing fee schedule in fail-closed mode — a config gap, not a
			// transient fault. Retrying won't help (the schedule/allowlist entry
			// must be added). Surface as a terminal failure so we don't stamp a
			// guessed fee and asynq doesn't pile on retries.
			logger.Error("disbursement execute: fee resolution failed — terminal (config gap), not retrying",
				"error", err, "advance_request_id", p.AdvanceRequestID, "request_id", p.RequestID)
			return fmt.Errorf("initiate wallet_payment: fee resolution: %w", asynqlib.SkipRetry)
		}
		return fmt.Errorf("initiate wallet_payment: %w", err)
	}

	// Step 2: Get active provider
	provider, err := w.registry.Active(ctx)
	if err != nil {
		return fmt.Errorf("no active provider: %w", err)
	}

	// A retry may find the row verified because the previous attempt completed
	// account validation but lost the transfer response or failed to persist it.
	// Query first; only an explicit provider not-found permits re-sending the
	// same idempotency key.
	switch row.Status {
	case domaintx.StatePending:
		// Normal first attempt; continue through account validation.
	case domaintx.StateVerified:
		recovered, notFound, recoverErr := recoverVerifiedTransfer(ctx, provider, w.walletPaymentService, row)
		if recoverErr != nil {
			return fmt.Errorf("recover verified transfer: %w", recoverErr)
		}
		if !notFound {
			logger.Info("disbursement execute: recovered existing provider transfer",
				"provider_ref", recovered.ProviderRef,
				"status", recovered.Status)
			return nil
		}
		logger.Info("disbursement execute: provider confirmed transfer not found; retrying same request_id")
	default:
		logger.Info("disbursement execute: skipping — already processed", "status", row.Status)
		return nil
	}

	// transferAccountName is the holder name sent to the provider on the
	// transfer. It defaults to our stored recipient name; if the account
	// check returns a bank-confirmed name, we prefer that (authoritative) so
	// we don't re-normalize and risk a holder-name mismatch at the provider —
	// OnePay compares the transfer's holder_name against the registered name
	// and rejects any divergence with response_code 15 "Invalid account info".
	transferAccountName := p.RecipientName
	if row.Status == domaintx.StateVerified {
		transferAccountName = row.RecipientName
	}

	// Step 3: Check account (if supported)
	if row.Status == domaintx.StatePending {
		verifier, ok := provider.(infrastructure.AccountVerifier)
		if ok {
			swiftCode, err := w.resolveSwiftCode(ctx, p.RecipientBank)
			if err != nil {
				return fmt.Errorf("check account: %w", err)
			}
			checkResult, err := verifier.CheckAccount(ctx, infrastructure.AccountCheckRequest{
				RequestID:   p.RequestID,
				BankCode:    p.RecipientBank,
				SwiftCode:   swiftCode,
				AccountNo:   p.RecipientAccountNo,
				AccountName: p.RecipientName,
				AccountType: infrastructure.AccountTypeBankAccount,
				Amount:      p.RequestedAmount,
			})
			if err != nil {
				return fmt.Errorf("check account: %w", err)
			}

			_, err = w.walletPaymentService.RecordAccountCheck(ctx, p.RequestID, disbursement.AccountCheckOutcome{
				Verified:     checkResult.Valid,
				AccountName:  checkResult.AccountName,
				RawErrorCode: checkResult.RawErrorCode,
				RawMessage:   checkResult.RawMessage,
				FeeWaived:    !checkResult.Valid,
			})
			if err != nil {
				return fmt.Errorf("record account check: %w", err)
			}

			if !checkResult.Valid {
				logger.Info("disbursement execute: account check failed",
					"error_code", checkResult.RawErrorCode)
				// Permanent account-data errors (e.g. the recipient name does
				// not match the bank-registered holder) will never succeed on
				// retry. Fail the request so orphan recovery stops re-enqueueing
				// it every cycle, and tell the employee what to fix. Transient
				// codes (86/99) keep the request APPROVED — the poller's retry
				// budget bounds how often we pay the provider to find that out.
				if classifyProviderFailure(checkResult.RawErrorCode) == providerFailurePermanent {
					w.applyPermanentProviderFailure(ctx, p, checkResult.RawErrorCode, checkResult.RawMessage)
				}
				return nil // terminal — don't retry
			}

			// Prefer the bank-confirmed holder name for the transfer.
			if checkResult.AccountName != "" {
				transferAccountName = checkResult.AccountName
			}
		}
	}

	// Step 4: Initiate transfer
	swiftCode, err := w.resolveSwiftCode(ctx, p.RecipientBank)
	if err != nil {
		return fmt.Errorf("initiate transfer: %w", err)
	}
	result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
		RequestID:   p.RequestID,
		Amount:      p.RequestedAmount,
		BankCode:    p.RecipientBank,
		SwiftCode:   swiftCode,
		AccountNo:   p.RecipientAccountNo,
		AccountName: transferAccountName,
		Description: p.RequestID,
		AccountType: infrastructure.AccountTypeBankAccount,
	})
	if err != nil {
		// Pre-flight validation rejection: the transfer endpoint was never
		// called, so no provider fee is charged and retrying won't help (the
		// request is malformed). Record the failure with the fee waived and
		// stop. Any other error is a transport failure → retry via asynq.
		if errors.Is(err, infrastructure.ErrPreflightValidation) {
			_, recordErr := w.walletPaymentService.RecordSyncResponse(ctx, p.RequestID, disbursement.SyncResult{
				Accepted:     false,
				RawErrorCode: "preflight_validation",
				RawMessage:   err.Error(),
				FeeWaived:    true,
			})
			if recordErr != nil {
				return fmt.Errorf("record pre-flight rejection: %w", recordErr)
			}
			logger.Info("disbursement execute: rejected at pre-flight (fee waived)", "error", err)
			// The payload is malformed (bad SWIFT, amount out of bounds, blank
			// holder name) — a retry re-sends the same bytes. Fail the request
			// instead of letting orphan recovery re-enqueue it forever.
			w.applyPermanentProviderFailure(ctx, p, "preflight_validation", err.Error())
			return nil // terminal — don't retry a malformed request
		}
		return fmt.Errorf("initiate transfer: %w", err)
	}

	// Step 5: Record sync response (transfer endpoint was called → fee applies)
	accepted := result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess
	_, err = w.walletPaymentService.RecordSyncResponse(ctx, p.RequestID, disbursement.SyncResult{
		Accepted:     accepted,
		InvoiceNo:    result.ProviderRef,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	})
	if err != nil {
		return fmt.Errorf("record sync response: %w", err)
	}

	if accepted {
		logger.Info("disbursement execute: transfer initiated",
			"provider_ref", result.ProviderRef, "status", string(result.Status))
		return nil
	}

	// The provider answered the transfer call synchronously with a rejection.
	// The fee is already charged (the endpoint was called), so this is where an
	// unclassified retry loop gets expensive: error 86 billed 13 × 3,850 VND
	// before the request was given up on. Data errors end the request here;
	// transient ones stay retryable, bounded by the poller's retry budget.
	logger.Info("disbursement execute: transfer rejected synchronously",
		"error_code", result.RawErrorCode, "status", string(result.Status))
	if classifyProviderFailure(result.RawErrorCode) == providerFailurePermanent {
		w.applyPermanentProviderFailure(ctx, p, result.RawErrorCode, result.RawMessage)
	}

	return nil
}

// providerFailureClass is the retry policy for one provider error code.
type providerFailureClass int

const (
	// providerFailureTransient means a later attempt may still succeed: the
	// provider was unavailable, throttling, or answered with a code we have
	// not classified. The request stays APPROVED so the poller's orphan
	// recovery re-enqueues it — bounded by maxFailedDisbursementAttempts, so a
	// permanently broken provider cannot bill us forever.
	providerFailureTransient providerFailureClass = iota
	// providerFailurePermanent means the request itself is wrong (bad account
	// data, malformed transfer). Re-sending the identical payload can never
	// succeed — it only burns provider fees (3,850 VND per transfer call) — so
	// the request is failed out of the retry loop immediately.
	providerFailurePermanent
)

// classifyProviderFailure is the single retry-policy classifier for BOTH stages
// of the advance disbursement pipeline — the account check (CheckAccount) and
// the funds transfer (InitiateTransfer) — because OnePay answers with the same
// response_code vocabulary at both, and our own stages add two synthetic codes.
//
// Wire codes (OnePay Payout API spec III.1 + our own):
//
//	code                  | meaning                                  | class
//	----------------------+------------------------------------------+-----------
//	name_mismatch         | bank holder ≠ stored holder (our synth)  | permanent
//	preflight_validation  | rejected locally, endpoint never called  | permanent
//	amount_below_min      | below the provider's per-transfer floor  | permanent
//	12                    | Mã ngân hàng không hợp lệ                | permanent
//	14                    | Thông tin thẻ không hợp lệ               | permanent
//	15                    | Thông tin tài khoản không hợp lệ         | permanent
//	21                    | Thông số không hợp lệ                    | permanent
//	86                    | Chức năng tạm thời đóng                  | transient
//	99                    | Hệ thống ngân hàng gián đoạn             | transient
//	"" / anything else    | unclassified — assume it may pass later  | transient
//
// Unknown codes are deliberately transient: the retry budget, not this table,
// is what stops a code nobody has classified yet from looping forever.
func classifyProviderFailure(errorCode string) providerFailureClass {
	switch errorCode {
	case "name_mismatch", // bank-confirmed holder differs from the stored name
		"preflight_validation", // our synthetic code for ErrPreflightValidation
		"amount_below_min",     // pre-flight amount floor
		"12",                   // invalid bank code
		"14",                   // invalid card info
		"15",                   // invalid account info
		"21":                   // invalid parameter
		return providerFailurePermanent
	default:
		return providerFailureTransient
	}
}

// isRecipientAccountFailure reports whether the code blames the recipient's
// stored bank account data, which is what makes the employee's
// bank_account_status flip to invalid. Only the account-data codes qualify:
// 12/14/15 are the provider's "this bank/card/account does not exist as given"
// answers and name_mismatch is the bank-confirmed holder-name divergence.
// preflight_validation and 21 describe our own payload, and 86/99 the provider's
// availability, so neither implicates the stored account.
func isRecipientAccountFailure(errorCode string) bool {
	switch errorCode {
	case "name_mismatch", "12", "14", "15":
		return true
	default:
		return false
	}
}

// employeeFacingDetail maps a permanent-failure detail to what the employee
// sees in the push notification. Provider messages (OnePay codes 12/14/15 and
// the name_mismatch synthesis) already arrive in Vietnamese; the synthetic
// preflight_validation code carries the local validator's English Go error
// text, which must not reach the employee — all user-facing text is
// Vietnamese. The raw English stays where it belongs: the wallet_payment
// audit row (RawMessage) and the logs.
func employeeFacingDetail(code, detail string) string {
	if code == "preflight_validation" {
		return "Thông tin yêu cầu chuyển tiền không hợp lệ. Vui lòng liên hệ quản trị viên để được hỗ trợ."
	}
	return detail
}

// applyPermanentProviderFailure is the terminal-failure policy for a permanent
// code, applied identically whatever the stage that produced it: the advance
// request is marked FAILED with the code as its reason, the employee is
// notified once (so they can fix their bank info), and — when the code blames
// the recipient account — the employee's bank account is flagged invalid for
// the missing-bank-details views.
func (w *DisbursementExecuteWorker) applyPermanentProviderFailure(ctx context.Context, p DisbursementExecutePayload, code, detail string) {
	w.failAdvanceRequest(ctx, p, code, detail)

	if !isRecipientAccountFailure(code) || w.walletPaymentService == nil {
		return
	}
	reason := detail
	if reason == "" {
		reason = code
	}
	if err := w.walletPaymentService.MarkRecipientBankAccountInvalid(ctx, p.AdvanceRequestID, reason); err != nil {
		w.logger.Warn("disbursement execute: failed to flag employee bank account invalid",
			"advance_request_id", p.AdvanceRequestID, "code", code, "error", err)
	}
}

// failAdvanceRequest marks an advance request FAILED (with the error code as
// the payment_reference reason, matching the poller's convention for
// budget_exceeded / missing bank details) and notifies the employee with the
// provider's message so they can fix their bank info and re-request.
//
// Notify-once: the employee notification is sent only by the caller that wins
// the APPROVED/PENDING → FAILED transition. A repeat attempt (concurrent task,
// orphan recovery racing a terminal failure, asynq retry) finds the request
// already resolved and returns before touching the notifier, so the employee
// never receives the same "advance failed" push twice.
//
// Best-effort: failures are logged, never propagated — the wallet_payment row
// is already recorded as failed, which is the audit trail that matters.
func (w *DisbursementExecuteWorker) failAdvanceRequest(ctx context.Context, p DisbursementExecutePayload, reason, detail string) {
	logger := w.logger.With("advance_request_id", p.AdvanceRequestID, "reason", reason)

	req, err := w.advanceReqRepo.GetByID(ctx, p.AdvanceRequestID)
	if err != nil {
		logger.Error("disbursement execute: failed to load request for terminal failure", "error", err)
		return
	}
	// Only fail requests still awaiting disbursement — never overwrite a status
	// an admin already resolved (e.g. cancelled) or that already completed.
	if req.Status != domain.AdvancePaymentStatusApproved && req.Status != domain.AdvancePaymentStatusPending {
		logger.Info("disbursement execute: skipping terminal failure — request already resolved", "status", string(req.Status))
		return
	}

	if err := w.advanceReqRepo.UpdateStatus(ctx, p.AdvanceRequestID, domain.AdvancePaymentStatusFailed, reason, nil); err != nil {
		logger.Error("disbursement execute: failed to mark request FAILED", "error", err)
		return
	}
	logger.Info("disbursement execute: request marked FAILED after permanent provider failure",
		"employee_id", req.EmployeeID, "amount", req.RequestAmount)

	if w.employeeNotifier != nil {
		w.employeeNotifier.NotifyAdvancePaymentFailed(ctx, req.EmployeeID, req.RequestAmount, detail)
	}
}
