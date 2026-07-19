package disbursement

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"

	"github.com/google/uuid"
)

// maxTransitionRetries caps how many times we re-read the row and re-fire
// the FSM trigger when the optimistic version check loses to a concurrent
// writer. Three is enough in practice — every retry attempt corresponds
// to a real concurrent commit landing on the row, and races deeper than
// that are vanishingly rare.
const maxTransitionRetries = 3

// DisbursementFeeProvider is the read-side surface the service uses to
// stamp the per-transfer disbursement fee onto new rows. Implemented by
// *FeeScheduleService.GetDisbursementFeeVND, which resolves the active
// scheduled fee at clock.Now() — declared as an interface here to keep
// Initiate testable with a stub fee source.
type DisbursementFeeProvider interface {
	GetDisbursementFeeVND(ctx context.Context, provider string) (fee int64, waived bool, err error)
}

// WalletPaymentService is the use-case layer over the
// provider_transactions table. It is the only thing in the codebase
// that calls the FSM directly: handlers and the 9pay client all go
// through Initiate / RecordAccountCheck / RecordSyncResponse / RecordIPN.
//
// Notification side effects live here (not in the FSM, not in the
// repository) because notifications are best-effort: a failed push
// shouldn't roll back the state transition. We fire-and-forget through
// the FSM's OnEntry hook; if it errors we log and move on.
type WalletPaymentService struct {
	repo               domaintx.WalletPaymentRepository
	advancePaymentReqs domain.AdvancePaymentRequestRepository
	employees          domain.EmployeeRepository
	bankRepo           domain.BankRepository
	notifications      *notification.NotificationService
	feeProvider        DisbursementFeeProvider
	errorTranslator    infrastructure.ErrorTranslator
	registry           *Registry
	logger             *slog.Logger
}

// NewWalletPaymentService wires a service against the live
// repository and notification stack. Any of advancePaymentReqs,
// employees, notifications, feeProvider, errorTranslator may be nil —
// when feeProvider is nil, Initiate stamps fee=0 and logs a warning;
// when errorTranslator is nil, the raw provider message passes through.
// The other nil dependencies cause employee notifications to be skipped
// silently.
func NewWalletPaymentService(
	repo domaintx.WalletPaymentRepository,
	advancePaymentReqs domain.AdvancePaymentRequestRepository,
	employees domain.EmployeeRepository,
	bankRepo domain.BankRepository,
	notifications *notification.NotificationService,
	feeProvider DisbursementFeeProvider,
	errorTranslator infrastructure.ErrorTranslator,
	registry *Registry,
	logger *slog.Logger,
) *WalletPaymentService {
	if logger == nil {
		logger = observability.GetLogger()
	}
	return &WalletPaymentService{
		repo:               repo,
		advancePaymentReqs: advancePaymentReqs,
		employees:          employees,
		bankRepo:           bankRepo,
		notifications:      notifications,
		feeProvider:        feeProvider,
		errorTranslator:    errorTranslator,
		registry:           registry,
		logger:             logger,
	}
}

// InitiateInput is the minimum the service needs to insert a fresh
// row in pending state, just before the synchronous provider call.
// EntityID is optional and links back to advance_payment_requests.id;
// debug-rig calls leave it nil.
type InitiateInput struct {
	RequestID          string
	RequestedAmount    int64
	RecipientName      string
	RecipientAccountNo string
	RecipientBank      string
	Description        string
	EntityID           *uint64
	CreatedBy          *uint64
	BatchID            *string
}

// Initiate inserts a new provider_transactions row in verifying state.
// The row's fee is stamped from the active disbursement_fee_schedules
// entry at INSERT time so historical rows preserve the rate in effect
// when the transfer happened, even if an admin later schedules a new
// rate to take effect on a future date.
// Returns the persisted row so the caller can keep the txn_id and the
// initial version number.
//
// Idempotency: when the unique index on request_id rejects the INSERT
// (caller is retrying with the same advance_payment transaction code),
// the existing row is returned in its current state instead of erroring.
// This lets a manual re-fire after a transient failure resume from the
// last known status — the caller can inspect row.Status to decide
// whether to call the provider again or simply observe the in-flight
// state.
func (s *WalletPaymentService) Initiate(ctx context.Context, in InitiateInput) (*domaintx.WalletPayment, error) {
	// Resolve provider name (needed for fee stamp, duplicate guard, and row storage).
	var fee int64
	var providerName string
	if s.registry != nil {
		if p, err := s.registry.Active(ctx); err == nil {
			providerName = p.Name()
		}
	}
	if s.feeProvider != nil {
		resolved, waived, ferr := s.feeProvider.GetDisbursementFeeVND(ctx, providerName)
		if ferr != nil {
			return nil, fmt.Errorf("%w: provider %q: %w", ErrFeeResolution, providerName, ferr)
		}
		fee = resolved
		if waived {
			s.logger.Info("provider_transactions: fee waived (zero-fee route)",
				"request_id", in.RequestID, "provider", providerName)
		}
	} else {
		s.logger.Warn("provider_transactions: fee provider not wired; stamping fee=0",
			"request_id", in.RequestID)
	}

	// Resolve bank code → SWIFT code early.
	// Callers pass bank code (e.g. "MB"); we persist the SWIFT code
	// (e.g. "MBVNVNVN") so recipient_bank is always a SWIFT code.
	// Resolution must happen BEFORE the duplicate guard so the guard
	// queries with the same SWIFT code stored in the DB.
	recipientBank := in.RecipientBank
	if s.bankRepo != nil && recipientBank != "" {
		if bank, err := s.bankRepo.FindByBankCode(ctx, recipientBank); err == nil && bank != nil && bank.SwiftCode != "" {
			recipientBank = bank.SwiftCode
		}
	}

	// Duplicate guard: prevent simultaneous payments to the same recipient
	// for the same provider. Both the auto-poller and manual admin
	// disbursement can target the same employee — this check blocks the
	// second path before creating a wallet_payment.
	//
	// NOTE: This is an application-level guard, not a DB constraint. A
	// narrow TOCTOU window exists between this SELECT and the subsequent
	// INSERT. A future migration should add a unique index on
	// (recipient_account_no, recipient_bank, provider) filtered to
	// non-terminal statuses for watertight protection.
	hasPending, dupErr := s.repo.HasPendingForRecipient(ctx, in.RecipientAccountNo, recipientBank, providerName)
	if dupErr != nil {
		return nil, fmt.Errorf("provider_transactions: duplicate check: %w", dupErr)
	}
	if hasPending {
		return nil, ErrDuplicatePaymentInProgress
	}

	var description *string
	if in.Description != "" {
		description = &in.Description
	}
	invoiceNo := in.RequestID
	row := &domaintx.WalletPayment{
		TxnID:              uuid.New(),
		RequestID:          in.RequestID,
		InvoiceNo:          &invoiceNo,
		Provider:           providerName,
		RequestedAmount:    in.RequestedAmount,
		Fee:                fee,
		RecipientName:      in.RecipientName,
		RecipientAccountNo: in.RecipientAccountNo,
		RecipientBank:      recipientBank,
		Description:        description,
		Status:             domaintx.StatePending,
		EntityID:           in.EntityID,
		CreatedBy:          in.CreatedBy,
		BatchID:            in.BatchID,
	}
	err := s.repo.Create(ctx, row)
	if err == nil {
		s.logger.Info("provider_transactions: initiated",
			"id", row.ID, "txn_id", row.TxnID.String(), "request_id", row.RequestID,
			"amount", row.RequestedAmount, "fee", row.Fee, "entity_id", row.EntityID)
		return row, nil
	}
	if !isDuplicateRequestIDError(err) {
		return nil, fmt.Errorf("provider_transactions: initiate: %w", err)
	}
	existing, lookupErr := s.repo.GetByRequestID(ctx, in.RequestID)
	if lookupErr != nil {
		return nil, fmt.Errorf("provider_transactions: initiate: duplicate request_id %q but lookup failed: %w", in.RequestID, lookupErr)
	}
	s.logger.Info("provider_transactions: idempotent initiate; returning existing row",
		"id", existing.ID, "txn_id", existing.TxnID.String(), "request_id", existing.RequestID,
		"current_status", existing.Status, "entity_id", existing.EntityID)
	return existing, nil
}

// GetByTxnID returns the current state of a provider transaction by its
// public-facing UUID. Returns ErrNotFound when no row matches.
func (s *WalletPaymentService) GetByTxnID(ctx context.Context, txnID string) (*domaintx.WalletPayment, error) {
	return s.repo.GetByTxnID(ctx, txnID)
}

// GetByRequestID returns the current state of a provider transaction by its
// provider-issued request_id. Returns ErrNotFound when no row matches.
func (s *WalletPaymentService) GetByRequestID(ctx context.Context, requestID string) (*domaintx.WalletPayment, error) {
	return s.repo.GetByRequestID(ctx, requestID)
}

// ListRecent returns the most recent provider_transactions ordered by
// created_at DESC. limit caps the result count; default 20.
func (s *WalletPaymentService) ListRecent(ctx context.Context, limit int) ([]*domaintx.WalletPayment, error) {
	return s.repo.ListRecent(ctx, limit)
}

// isDuplicateRequestIDError reports whether err is a unique-index
// violation from inserting a duplicate request_id. Matches MySQL,
// SQLite and Postgres error text so the same path works in production
// and the test suite.
func isDuplicateRequestIDError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "Duplicate entry") &&
		!strings.Contains(msg, "UNIQUE constraint failed") &&
		!strings.Contains(msg, "duplicate key value violates unique constraint") {
		return false
	}
	return strings.Contains(msg, "idx_wp_request_id") ||
		strings.Contains(msg, "request_id") ||
		strings.Contains(msg, "UNIQUE constraint failed: wallet_payments.request_id")
}

// AccountCheckOutcome is what RecordAccountCheck needs to know about
// the synchronous /disbursement/check-account reply. Verified=true
// fires TriggerAccountVerified (verifying → pending); false fires
// TriggerAccountRejected (verifying → failed). The raw error fields
// are persisted regardless so failed-row inspectors can see why 9pay
// refused the account.
type AccountCheckOutcome struct {
	Verified     bool
	RawErrorCode string
	RawMessage   string
}

// RecordAccountCheck fires the account-verification FSM trigger after
// the synchronous /disbursement/check-account call. Must be called
// before any attempt to invoke the create-disbursement endpoint — when
// Verified=false, the row terminates as failed and the create call
// never happens.
//
// Concurrency: under maxTransitionRetries with optimistic version checks.
func (s *WalletPaymentService) RecordAccountCheck(ctx context.Context, requestID string, outcome AccountCheckOutcome) (*domaintx.WalletPayment, error) {
	trigger := domaintx.TriggerVerify
	if !outcome.Verified {
		trigger = domaintx.TriggerReject
	}
	return s.transition(ctx, lookupByRequestID(s.repo, requestID), trigger, s.accountCheckPatch(outcome))
}

// accountCheckPatch persists the raw provider response fields so an
// operator inspecting a failed row sees the exact error code/message
// that 9pay returned. Mirrors syncPatch in shape — only error_code
// and error_message are written; the verified branch overwrites them
// with success values, which is fine (the row's status carries the
// actual outcome).
func (s *WalletPaymentService) accountCheckPatch(o AccountCheckOutcome) domaintx.UpdatePatch {
	patch := domaintx.UpdatePatch{}
	code := o.RawErrorCode
	patch.ErrorCode = &code
	msg := s.translateMessage(o.RawErrorCode, o.RawMessage)
	patch.ErrorMessage = &msg
	return patch
}

// SyncResult is what RecordSyncResponse needs to know about the
// provider's synchronous reply: was it accepted (queued for IPN) or
// hard-rejected, and what raw error code/message did it carry.
type SyncResult struct {
	Accepted     bool   // true → TriggerProviderAccepted; false → TriggerProviderRejected
	InvoiceNo    string // optional; captured into row.invoice_no when non-empty
	RawErrorCode string
	RawMessage   string
	// FeeWaived is true when the transfer endpoint was NEVER called (a
	// pre-flight validation rejection), so the provider's per-transfer fee
	// was NOT charged. syncPatch zeroes the stamped fee in that case so
	// SUM(fee) reflects only fees actually charged. Defaults false — the
	// common path (endpoint was called) keeps the stamped fee untouched.
	FeeWaived bool
}

// RecordSyncResponse fires the appropriate FSM trigger for the
// synchronous provider reply and applies the corresponding patch.
// The row should be in verified state (after RecordAccountCheck).
//
// Concurrency: under maxTransitionRetries with optimistic version checks.
func (s *WalletPaymentService) RecordSyncResponse(ctx context.Context, requestID string, result SyncResult) (*domaintx.WalletPayment, error) {
	trigger := domaintx.TriggerAuthorise
	if !result.Accepted {
		trigger = domaintx.TriggerReject
	}
	return s.transition(ctx, lookupByRequestID(s.repo, requestID), trigger, s.syncPatch(result))
}

// IPNResult is the normalized IPN payload the service needs.
// Status drives the trigger; the rest are patch fields.
// Provider is set from the IPNJob to disambiguate lookups.
type IPNResult struct {
	Provider      string
	Status        infrastructure.TransferStatus
	InvoiceNo     string
	RequestID     string // fallback lookup when invoice_no is missing
	Amount        int64  // provider-reported amount, retained for debug logs (verified == requested)
	RawErrorCode  string
	RawMessage    string
	FailureReason string
	Source        string // resolution source: ipn, status_inquiry, manual
}

// RecordIPN fires the IPN-side FSM trigger for an inbound webhook event.
// Looks the row up by invoice_no first (the 9pay primary key the IPN
// carries — wire-named payment_no), falling back to request_id when
// invoice_no is empty.
//
// The provider's translateStatus already collapses 9pay's many states
// into our 4 normalized values, so the mapping here is straightforward:
//
//	success  → ipn_completed
//	failed   → ipn_failed
//	reversed → ipn_reversed
//	(anything else, including pending/unknown, is a no-op
//	values that don't drive a transition shouldn't be applied)
func (s *WalletPaymentService) RecordIPN(ctx context.Context, result IPNResult) (*domaintx.WalletPayment, error) {
	var trigger domaintx.Trigger
	switch result.Status {
	case infrastructure.TransferStatusSuccess:
		trigger = domaintx.TriggerIPNCompleted
	case infrastructure.TransferStatusFailed:
		trigger = domaintx.TriggerIPNFailed
	case infrastructure.TransferStatusReversed:
		trigger = domaintx.TriggerIPNReversed
	default:
		s.logger.Info("provider_transactions: ignoring non-terminal IPN status",
			"status", result.Status, "invoice_no", result.InvoiceNo, "request_id", result.RequestID)
		return nil, nil
	}

	lookup := lookupByProviderInvoiceNo(s.repo, result.Provider, result.InvoiceNo)
	if result.InvoiceNo == "" {
		lookup = lookupByRequestID(s.repo, result.RequestID)
	}

	// Amount cross-check: the IPN-claimed amount must match what we persisted at
	// Initiate (RequestedAmount). A mismatch signals a forged/tampered callback
	// and must never reach the FSM. Skipped when the IPN omits an amount (some
	// provider payloads do for non-terminal statuses) — we only hard-reject on a
	// present-and-differing amount, never break on omission.
	if result.Amount > 0 {
		existing, lerr := lookup(ctx)
		switch {
		case lerr == nil && existing != nil:
			if existing.RequestedAmount != result.Amount {
				s.logger.Error("provider_transactions: IPN amount mismatch — rejecting",
					"request_id", result.RequestID,
					"invoice_no", result.InvoiceNo,
					"provider", result.Provider,
					"expected", existing.RequestedAmount,
					"reported", result.Amount)
				return nil, fmt.Errorf("%w: expected %d, reported %d (request_id=%s)",
					ErrIPNAmountMismatch, existing.RequestedAmount, result.Amount, result.RequestID)
			}
		case errors.Is(lerr, domaintx.ErrNotFound):
			// Unknown invoice — let transition return the canonical ErrNotFound below.
		case lerr != nil:
			return nil, lerr
		}
	}

	return s.transition(ctx, lookup, trigger, s.ipnPatch(result, trigger))
}

// transition is the canonical optimistic-locking retry loop.
//
//  1. Look up the row.
//  2. Build a fresh FSM with the row's current state plus the hooks
//     that close over this row's data (so OnEntry has the amount,
//     employee link, etc. it needs).
//  3. Fire the trigger; bail on FSM rejection.
//  4. UPDATE with version=expected. On version miss, retry from (1).
//
// Returns the row in its post-update state on success.
func (s *WalletPaymentService) transition(ctx context.Context, lookup func(context.Context) (*domaintx.WalletPayment, error), trigger domaintx.Trigger, patch domaintx.UpdatePatch) (*domaintx.WalletPayment, error) {
	var lastErr error
	for attempt := 0; attempt < maxTransitionRetries; attempt++ {
		row, err := lookup(ctx)
		if err != nil {
			return nil, err
		}

		// Bail early if the FSM would reject this trigger from the
		// current state — saves us issuing a doomed UPDATE.
		if !domaintx.CanFire(row.Status, trigger) {
			return nil, domaintx.TerminalStateError{State: row.Status, Trigger: trigger}
		}

		hooks := s.buildHooks(row)
		sm := domaintx.Build(row.Status, hooks)
		if err := sm.FireCtx(ctx, trigger); err != nil {
			return nil, fmt.Errorf("provider_transactions: fire trigger %q: %w", trigger, err)
		}

		// The FSM has computed the destination state and run OnEntry.
		// Capture the new state for the patch — but only if the patch
		// caller didn't already set Status (debug paths may want to
		// override this for testing; production paths leave it nil).
		newStateRaw, err := sm.State(ctx)
		if err != nil {
			return nil, fmt.Errorf("provider_transactions: state: %w", err)
		}
		newState, ok := newStateRaw.(domaintx.State)
		if !ok {
			return nil, fmt.Errorf("provider_transactions: unexpected state type %T", newStateRaw)
		}
		if patch.Status == nil {
			patch.Status = &newState
		}

		err = s.repo.UpdateExpected(ctx, row.ID, row.Version, patch)
		if errors.Is(err, domaintx.ErrConcurrentModification) {
			s.logger.Info("provider_transactions: version miss; retrying",
				"id", row.ID, "version", row.Version, "trigger", trigger, "attempt", attempt)
			lastErr = err
			base := time.Duration(50*(1<<attempt)) * time.Millisecond
			time.Sleep(base + time.Duration(rand.Int64N(int64(base)/2)))
			continue
		}
		if err != nil {
			return nil, err
		}

		// Re-fetch so the caller gets the canonical post-update view.
		// This costs one extra round-trip but means the returned row's
		// fields (notably settled_at) match what's in the DB.
		row, err = s.repo.GetByID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		s.logger.Info("provider_transactions: transition applied",
			"id", row.ID, "trigger", trigger, "to_status", row.Status,
			"version", row.Version, "settled_at", row.SettledAt)
		return row, nil
	}
	return nil, fmt.Errorf("provider_transactions: exceeded %d retries: %w", maxTransitionRetries, lastErr)
}

// buildHooks closes over the row so the FSM's OnEntry callback has
// everything it needs (amount, error message, employee link) without
// reaching back into the repository at notification time.
//
// Per spec: notify on completed and failed; do NOT notify on reversed.
func (s *WalletPaymentService) buildHooks(row *domaintx.WalletPayment) domaintx.Hooks {
	return domaintx.Hooks{
		OnEnterCompleted: func(ctx context.Context, _, _ domaintx.State, trigger domaintx.Trigger) error {
			// Update advance payment request to COMPLETED on IPN success
			if row.EntityID != nil && s.advancePaymentReqs != nil {
				now := clock.Now()
				if err := s.advancePaymentReqs.UpdateStatus(ctx, uint64(*row.EntityID), domain.AdvancePaymentStatusCompleted, row.TxnID.String(), &now); err != nil {
					s.logger.Warn("wallet_payments: failed to update advance request to COMPLETED",
						"entity_id", *row.EntityID, "error", err)
				}
			}
			s.notifyEmployee(ctx, row, completedMessage(row))
			s.notifyInitiator(ctx, row, initiatorCompletedMessage(row))
			return nil
		},
		OnEnterFailed: func(ctx context.Context, _, _ domaintx.State, trigger domaintx.Trigger) error {
			// Do NOT update advance request on IPN failure — wait for reconciliation.
			// Do NOT notify employee of failure (premature until recon confirms).
			// Only notify admin/initiator so ops can monitor.
			if trigger == domaintx.TriggerIPNFailed {
				s.logger.Warn("wallet_payments: payment failed via IPN; advance request unchanged until reconciliation",
					"id", row.ID, "txn_id", row.TxnID.String(), "entity_id", row.EntityID)
				s.notifyInitiator(ctx, row, initiatorFailedMessage(row))
				return nil
			}
			// TriggerReject (account check failure): no money moved, notify initiator only
			s.notifyInitiator(ctx, row, initiatorFailedMessage(row))
			return nil
		},
		OnEnterReversed: func(ctx context.Context, _, _ domaintx.State, _ domaintx.Trigger) error {
			s.logger.Warn("wallet_payments: payment reversed via IPN; advance request unchanged until reconciliation",
				"id", row.ID, "txn_id", row.TxnID.String(), "entity_id", row.EntityID)
			s.notifyInitiator(ctx, row, initiatorFailedMessage(row))
			return nil
		},
	}
}

// notifyEmployee resolves the recipient user_id and fires a push notification.
// Best-effort: any failure is logged, never returned.
//
// Resolution path depends on what links the row to an employee:
//   - Advance-payment / FlexPay rows (EntityID != nil): advance_payment_requests
//     → employees → user_id.
//   - Wallet bulk-transfer rows (EntityID == nil, per Phase 3 of the wallet
//     bulk transfer pipeline): recipient_account_no →
//     employees.bank_account_number → user_id. (H6 fix — Validation Decision V6.)
func (s *WalletPaymentService) notifyEmployee(ctx context.Context, row *domaintx.WalletPayment, msg notificationMessage) {
	if s.notifications == nil {
		return // not wired (tests / dormant deployments)
	}

	var userID *uint

	if row.EntityID != nil {
		// Advance-payment path.
		if s.advancePaymentReqs == nil || s.employees == nil {
			return
		}
		req, err := s.advancePaymentReqs.GetByID(ctx, uint64(*row.EntityID))
		if err != nil {
			s.logger.Warn("provider_transactions: notification skipped — advance payment request not found",
				"entity_id", *row.EntityID, "error", err)
			return
		}
		emp, err := s.employees.GetByID(ctx, uint(req.EmployeeID))
		if err != nil || emp == nil || emp.UserID == nil {
			s.logger.Warn("provider_transactions: notification skipped — employee or user not found",
				"employee_id", req.EmployeeID, "error", err)
			return
		}
		userID = emp.UserID
	} else {
		// Bulk-transfer path — resolve via bank account number (H6 fix).
		// Skipped silently when recipient_account_no is empty or no employee
		// matches; bulk rows for non-employee recipients (partner invoices,
		// test fixtures) have nobody to notify.
		if s.employees == nil || row.RecipientAccountNo == "" {
			return
		}
		emp, err := s.employees.GetByBankAccountNumber(ctx, row.RecipientAccountNo)
		if err != nil || emp == nil || emp.UserID == nil {
			s.logger.Info("provider_transactions: bulk row notification skipped — no employee matches account_no",
				"txn_id", row.TxnID.String(), "request_id", row.RequestID, "account_no", row.RecipientAccountNo)
			return
		}
		userID = emp.UserID
	}

	if userID == nil {
		return
	}
	if err := s.notifications.CreateNotification(ctx, *userID, domain.NotificationTypeAdvancePaymentStatusChanged, msg.Title, msg.Body); err != nil {
		s.logger.Warn("provider_transactions: notification failed",
			"user_id", *userID, "txn_id", row.TxnID.String(), "error", err)
	}
}

// notifyInitiator sends a push notification to the user who initiated
// the transfer (row.CreatedBy). Skipped when CreatedBy is nil (debug rig,
// legacy rows) or when it matches the employee user ID already notified
// by notifyEmployee (dedup). Best-effort: failures are logged, never returned.
func (s *WalletPaymentService) notifyInitiator(ctx context.Context, row *domaintx.WalletPayment, msg notificationMessage) {
	if row.CreatedBy == nil {
		return
	}
	if s.notifications == nil {
		return
	}
	if row.EntityID != nil && s.advancePaymentReqs != nil && s.employees != nil {
		if req, err := s.advancePaymentReqs.GetByID(ctx, uint64(*row.EntityID)); err == nil {
			if emp, err := s.employees.GetByID(ctx, uint(req.EmployeeID)); err == nil && emp.UserID != nil {
				if *emp.UserID == uint(*row.CreatedBy) {
					return
				}
			}
		}
	}
	userID := uint(*row.CreatedBy)
	if err := s.notifications.CreateNotification(ctx, userID, domain.NotificationTypeAdvancePaymentStatusChanged, msg.Title, msg.Body); err != nil {
		s.logger.Warn("provider_transactions: initiator notification failed",
			"user_id", userID, "txn_id", row.TxnID.String(), "error", err)
	}
}

// notificationMessage is a tiny pair so completedMessage and
// failedMessage can both return Title + Body without a dedicated struct
// per call.
type notificationMessage struct {
	Title string
	Body  string
}

func completedMessage(row *domaintx.WalletPayment) notificationMessage {
	return notificationMessage{
		Title: "Tạm ứng đã được chuyển",
		Body:  fmt.Sprintf("Tạm ứng %s đã được chuyển vào tài khoản của bạn.", utils.FormatVND(row.RequestedAmount)),
	}
}

// failureReason extracts a meaningful failure reason from the row.
// Returns "" when error_code is a known provider success code (9Pay: "000"/"0",
// OnePay: "00") — those codes appear on the initial acceptance response and
// are not a real failure reason. A non-empty error_code that is not a success
// code, or an empty error_code with a non-empty message, is surfaced as-is.
func failureReason(row *domaintx.WalletPayment) string {
	if row.ErrorMessage == nil || *row.ErrorMessage == "" {
		return ""
	}
	code := ""
	if row.ErrorCode != nil {
		code = strings.TrimSpace(*row.ErrorCode)
	}
	// Only suppress the message when we have a known success code.
	// An empty code with a non-empty message may carry real diagnostic info.
	switch code {
	case "0", "00", "000":
		return ""
	}
	return *row.ErrorMessage
}

func failedMessage(row *domaintx.WalletPayment) notificationMessage {
	reason := failureReason(row)
	body := fmt.Sprintf("Tạm ứng %s thất bại.", utils.FormatVND(row.RequestedAmount))
	if reason != "" {
		body = fmt.Sprintf("Tạm ứng %s thất bại. Lý do: %s.", utils.FormatVND(row.RequestedAmount), reason)
	}
	return notificationMessage{
		Title: "Chuyển tạm ứng thất bại",
		Body:  body,
	}
}

func initiatorCompletedMessage(row *domaintx.WalletPayment) notificationMessage {
	return notificationMessage{
		Title: "Chuyển tiền thành công",
		Body:  fmt.Sprintf("Chuyển tiền %s đến %s (%s) đã thành công.", utils.FormatVND(row.RequestedAmount), row.RecipientName, row.RecipientAccountNo),
	}
}

func initiatorFailedMessage(row *domaintx.WalletPayment) notificationMessage {
	reason := failureReason(row)
	body := fmt.Sprintf("Chuyển tiền %s đến %s (%s) thất bại.", utils.FormatVND(row.RequestedAmount), row.RecipientName, row.RecipientAccountNo)
	if reason != "" {
		body = fmt.Sprintf("%s Lý do: %s.", body, reason)
	}
	return notificationMessage{
		Title: "Chuyển tiền thất bại",
		Body:  body,
	}
}

// lookupByRequestID returns a closure the transition loop can call to
// re-fetch the row each retry. Same pattern as lookupByInvoiceNo.
func lookupByRequestID(repo domaintx.WalletPaymentRepository, requestID string) func(context.Context) (*domaintx.WalletPayment, error) {
	return func(ctx context.Context) (*domaintx.WalletPayment, error) {
		return repo.GetByRequestID(ctx, requestID)
	}
}

// lookupByProviderInvoiceNo returns a closure that resolves the row by the
// (provider, invoice_no) composite key. Used by RecordIPN.
func lookupByProviderInvoiceNo(repo domaintx.WalletPaymentRepository, provider, invoiceNo string) func(context.Context) (*domaintx.WalletPayment, error) {
	return func(ctx context.Context) (*domaintx.WalletPayment, error) {
		return repo.GetByProviderInvoiceNo(ctx, provider, invoiceNo)
	}
}

// basePatch sets the InvoiceNo, ErrorCode, and ErrorMessage fields shared by
// both sync and async patch paths. error_code/error_message are written
// unconditionally: on Accepted they typically carry "000"/"Thành công",
// which is informational data that should still appear in the row.
func (s *WalletPaymentService) basePatch(invoiceNo, rawErrorCode, errorMessage string) domaintx.UpdatePatch {
	patch := domaintx.UpdatePatch{}
	if invoiceNo != "" {
		v := invoiceNo
		patch.InvoiceNo = &v
	}
	code := rawErrorCode
	patch.ErrorCode = &code
	patch.ErrorMessage = &errorMessage
	return patch
}

// syncPatch builds an UpdatePatch from a synchronous provider reply.
func (s *WalletPaymentService) syncPatch(r SyncResult) domaintx.UpdatePatch {
	patch := s.basePatch(r.InvoiceNo, r.RawErrorCode, s.translateMessage(r.RawErrorCode, r.RawMessage))
	// FeeWaived (pre-flight rejection) → transfer endpoint never called →
	// zero the stamped fee. See SyncResult.FeeWaived for the full rationale.
	if r.FeeWaived {
		zero := int64(0)
		patch.Fee = &zero
	}
	return patch
}

// ipnPatch builds an UpdatePatch from a normalized IPN result. The
// charged-amount column was dropped in migration 049 (9pay's IPN
// amount always equals our requested amount, verified empirically).
// The disbursement-provider fee is stamped at INSERT time from the
// active disbursement_fee_schedules entry and never overwritten —
// 9pay's IPN payload does not carry a fee field.
func (s *WalletPaymentService) ipnPatch(r IPNResult, _ domaintx.Trigger) domaintx.UpdatePatch {
	failure := r.FailureReason
	if failure == "" {
		failure = s.translateMessage(r.RawErrorCode, r.RawMessage)
	}
	patch := s.basePatch(r.InvoiceNo, r.RawErrorCode, failure)
	if domaintx.ValidResolutionSource(r.Source) {
		source := r.Source
		patch.ResolutionSource = &source
	}
	return patch
}

// translateMessage prefers the active provider's Vietnamese translation
// when one is wired and a known code is in play; falls back to whatever
// raw provider message we got. The raw English message is kept for
// diagnostics in logs but not surfaced to employees.
//
// Provider translators return a generic "Lỗi không xác định (mã: X)"
// fallback for codes outside their map (e.g. preflight validation codes
// like "amount_below_min"). In those cases the raw message is already
// Vietnamese and preferable to the generic fallback.
func (s *WalletPaymentService) translateMessage(code, raw string) string {
	if s.errorTranslator != nil {
		if vi := s.errorTranslator.TranslateError(code); vi != "" && vi != code {
			if strings.HasPrefix(vi, "Lỗi không xác định") {
				return raw
			}
			return vi
		}
	}
	return raw
}

// ReconcilePayment applies the reconciliation result to a wallet_payment.
// This is the daily midnight cron path — it bypasses the FSM and directly
// updates status + reconciled_at, then updates the linked advance payment
// request accordingly.
//
// Rules:
//   - failed + recon success → override to completed, advance request → COMPLETED
//   - failed + recon failed → confirm failed, advance request → FAILED
//   - completed + recon success → set reconciled_at (confirm)
//   - completed + recon failed → log CRITICAL alert
//   - authorised + recon success → set completed + reconciled_at
//   - authorised + recon failed → set failed + reconciled_at
//   - verified + recon success → set completed + reconciled_at
//   - verified + recon failed → set failed + reconciled_at
func (s *WalletPaymentService) ReconcilePayment(ctx context.Context, provider, invoiceNo string, reconSuccess bool) (*domaintx.WalletPayment, error) {
	row, err := s.repo.GetByProviderInvoiceNo(ctx, provider, invoiceNo)
	if err != nil {
		return nil, fmt.Errorf("reconcile: lookup %q: %w", invoiceNo, err)
	}

	now := clock.Now()

	switch row.Status {
	case domaintx.StateFailed:
		if reconSuccess {
			s.logger.Warn("wallet_payments: reconciliation override — failed → completed",
				"id", row.ID, "invoice_no", invoiceNo, "entity_id", row.EntityID)
			if err := s.repo.MarkReconciled(ctx, row.ID, domaintx.StateCompleted, now); err != nil {
				return nil, fmt.Errorf("reconcile: mark reconciled completed: %w", err)
			}
			s.updateAdvanceRequest(ctx, row, domain.AdvancePaymentStatusCompleted, &now)
			row, _ = s.repo.GetByID(ctx, row.ID)
			s.notifyEmployee(ctx, row, completedMessage(row))
			s.notifyInitiator(ctx, row, initiatorCompletedMessage(row))
		} else {
			s.logger.Info("wallet_payments: reconciliation confirmed failed",
				"id", row.ID, "invoice_no", invoiceNo)
			if err := s.repo.MarkReconciled(ctx, row.ID, domaintx.StateFailed, now); err != nil {
				return nil, fmt.Errorf("reconcile: mark reconciled failed: %w", err)
			}
			s.updateAdvanceRequest(ctx, row, domain.AdvancePaymentStatusFailed, nil)
			row, _ = s.repo.GetByID(ctx, row.ID)
			s.notifyEmployee(ctx, row, failedMessage(row))
			s.notifyInitiator(ctx, row, initiatorFailedMessage(row))
		}

	case domaintx.StateCompleted:
		if !reconSuccess {
			s.logger.Error("wallet_payments: CRITICAL — completed payment but recon says failed",
				"id", row.ID, "invoice_no", invoiceNo, "amount", row.RequestedAmount)
		}
		if err := s.repo.MarkReconciled(ctx, row.ID, domaintx.StateCompleted, now); err != nil {
			return nil, fmt.Errorf("reconcile: mark reconciled: %w", err)
		}
		row, _ = s.repo.GetByID(ctx, row.ID)

	case domaintx.StateAuthorised, domaintx.StateVerified:
		// IPN never arrived (authorised) or sync response never arrived (verified) —
		// use recon as the terminal outcome.
		if reconSuccess {
			if err := s.repo.MarkReconciled(ctx, row.ID, domaintx.StateCompleted, now); err != nil {
				return nil, fmt.Errorf("reconcile: mark %s→completed: %w", row.Status, err)
			}
			s.updateAdvanceRequest(ctx, row, domain.AdvancePaymentStatusCompleted, &now)
			row, _ = s.repo.GetByID(ctx, row.ID)
			s.notifyEmployee(ctx, row, completedMessage(row))
			s.notifyInitiator(ctx, row, initiatorCompletedMessage(row))
		} else {
			if err := s.repo.MarkReconciled(ctx, row.ID, domaintx.StateFailed, now); err != nil {
				return nil, fmt.Errorf("reconcile: mark %s→failed: %w", row.Status, err)
			}
			s.updateAdvanceRequest(ctx, row, domain.AdvancePaymentStatusFailed, nil)
			row, _ = s.repo.GetByID(ctx, row.ID)
			s.notifyEmployee(ctx, row, failedMessage(row))
			s.notifyInitiator(ctx, row, initiatorFailedMessage(row))
		}

	default:
		s.logger.Info("wallet_payments: reconciliation skipped — unexpected status",
			"id", row.ID, "status", row.Status, "invoice_no", invoiceNo)
	}

	return row, nil
}

// updateAdvanceRequest is a best-effort helper that updates the linked
// advance_payment_request status. Failures are logged, never returned.
func (s *WalletPaymentService) updateAdvanceRequest(ctx context.Context, row *domaintx.WalletPayment, status domain.AdvancePaymentRequestStatus, paidAt *time.Time) {
	if row.EntityID == nil || s.advancePaymentReqs == nil {
		return
	}
	paymentRef := row.GetInvoiceNo()
	if err := s.advancePaymentReqs.UpdateStatus(ctx, uint64(*row.EntityID), status, paymentRef, paidAt); err != nil {
		s.logger.Warn("wallet_payments: failed to update advance request",
			"entity_id", *row.EntityID, "status", status, "error", err)
	}
}
