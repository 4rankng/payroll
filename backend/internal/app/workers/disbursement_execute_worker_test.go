package workers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	asynqlib "github.com/hibiken/asynq"

	disbursement "api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

// TestClassifyProviderFailure is the retry-policy contract for BOTH stages of
// the disbursement pipeline (account check + funds transfer). The distinction
// costs real money: every retry of a permanent data error is another provider
// call, and the provider bills its per-transfer fee for each one (error 86 was
// retried 13 times at 3,850 VND before anyone noticed).
func TestClassifyProviderFailure(t *testing.T) {
	cases := []struct {
		code string
		want providerFailureClass
	}{
		// Permanent — data errors no retry can fix.
		{"name_mismatch", providerFailurePermanent},        // bank-confirmed holder differs from stored name
		{"preflight_validation", providerFailurePermanent}, // rejected locally, endpoint never called
		{"amount_below_min", providerFailurePermanent},     // below the provider's per-transfer floor
		{"12", providerFailurePermanent},                   // Mã ngân hàng không hợp lệ
		{"14", providerFailurePermanent},                   // Thông tin thẻ không hợp lệ
		{"15", providerFailurePermanent},                   // Thông tin tài khoản không hợp lệ
		{"21", providerFailurePermanent},                   // Thông số không hợp lệ

		// Transient — may succeed on a later attempt.
		{"86", providerFailureTransient}, // Chức năng tạm thời đóng
		{"99", providerFailureTransient}, // Hệ thống ngân hàng gián đoạn

		// Unknown / empty — deliberately transient; the poller's retry budget,
		// not this table, is what bounds a code nobody has classified yet.
		{"", providerFailureTransient},
		{"00", providerFailureTransient},
		{"13", providerFailureTransient},
		{"E26", providerFailureTransient},
		{"NAME_MISMATCH", providerFailureTransient}, // case-sensitive: only the code we emit matches
	}

	for _, tc := range cases {
		if got := classifyProviderFailure(tc.code); got != tc.want {
			t.Errorf("classifyProviderFailure(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestIsRecipientAccountFailure pins which permanent codes implicate the
// employee's stored bank account data — only those may flip the employee's
// bank_account_status to invalid.
func TestIsRecipientAccountFailure(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"name_mismatch", true},
		{"12", true},
		{"14", true},
		{"15", true},
		{"21", false},                   // our payload is wrong, not the stored account
		{"preflight_validation", false}, // same
		{"amount_below_min", false},     // amount, not the account
		{"86", false},                   // provider availability
		{"99", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := isRecipientAccountFailure(tc.code); got != tc.want {
			t.Errorf("isRecipientAccountFailure(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestApplyPermanentProviderFailure_FailsOnceAndNotifiesOnce drives the
// terminal-failure policy through a repeat attempt: the first call must fail
// the request and notify the employee, the second (a duplicate task, an orphan
// re-enqueue, an asynq retry) must do neither — a second "advance failed" push
// for the same request is a support ticket.
func TestApplyPermanentProviderFailure_FailsOnceAndNotifiesOnce(t *testing.T) {
	repo := &stubAdvanceRequestRepo{
		req: &domain.AdvancePaymentRequest{
			ID:            256,
			EmployeeID:    9,
			RequestAmount: 5_000_000,
			Status:        domain.AdvancePaymentStatusApproved,
		},
	}
	notifier := &recordingEmployeeNotifier{}
	w := &DisbursementExecuteWorker{
		advanceReqRepo:   repo,
		employeeNotifier: notifier,
		logger:           testLogger(),
	}
	p := DisbursementExecutePayload{RequestID: "tt1236428580de4c", AdvanceRequestID: 256}

	w.applyPermanentProviderFailure(context.Background(), p, "14", "Thông tin thẻ không hợp lệ")
	w.applyPermanentProviderFailure(context.Background(), p, "14", "Thông tin thẻ không hợp lệ")

	if got := repo.statuses; len(got) != 1 || got[0] != domain.AdvancePaymentStatusFailed {
		t.Fatalf("UpdateStatus calls = %v, want exactly one FAILED", got)
	}
	if got := repo.refs; len(got) != 1 || got[0] != "14" {
		t.Fatalf("payment_reference reasons = %v, want exactly one \"14\"", got)
	}
	if len(notifier.failed) != 1 {
		t.Fatalf("NotifyAdvancePaymentFailed calls = %d, want 1", len(notifier.failed))
	}
	if n := notifier.failed[0]; n.employeeID != 9 || n.amount != 5_000_000 || n.detail != "Thông tin thẻ không hợp lệ" {
		t.Fatalf("notification = %+v, want employee 9 / 5,000,000 / provider detail", n)
	}
}

// TestApplyPermanentProviderFailure_FlagsEmployeeBankAccount covers the other
// half of the permanent-failure policy: a recipient-account code (14 here) must
// mark the employee's bank account invalid with the provider's Vietnamese
// reason, so the employee shows up in the missing-bank-details views and knows
// what to fix.
func TestApplyPermanentProviderFailure_FlagsEmployeeBankAccount(t *testing.T) {
	reqRepo := &stubAdvanceRequestRepo{
		req: &domain.AdvancePaymentRequest{
			ID:            157,
			EmployeeID:    42,
			RequestAmount: 3_000_000,
			Status:        domain.AdvancePaymentStatusApproved,
		},
	}
	empRepo := &stubEmployeeRepo{}
	svc := disbursement.NewWalletPaymentService(
		&stubWalletPaymentRepo{}, reqRepo, empRepo, nil, nil, nil, nil, nil, testLogger(),
	)

	notifier := &recordingEmployeeNotifier{}
	w := &DisbursementExecuteWorker{
		walletPaymentService: svc,
		advanceReqRepo:       reqRepo,
		employeeNotifier:     notifier,
		logger:               testLogger(),
	}

	w.applyPermanentProviderFailure(context.Background(),
		DisbursementExecutePayload{AdvanceRequestID: 157},
		"14", "Thông tin thẻ không hợp lệ")

	if empRepo.updatedID != 42 {
		t.Fatalf("employee UpdateColumns id = %d, want 42", empRepo.updatedID)
	}
	if got := empRepo.columns["bank_account_status"]; got != domain.BankAccountStatusInvalid {
		t.Errorf("bank_account_status = %v, want %q", got, domain.BankAccountStatusInvalid)
	}
	if got := empRepo.columns["bank_account_invalid_reason"]; got != "Thông tin thẻ không hợp lệ" {
		t.Errorf("bank_account_invalid_reason = %v, want the provider detail", got)
	}
	if _, ok := empRepo.columns["bank_account_validated_at"]; !ok {
		t.Error("bank_account_validated_at not written alongside the verdict")
	}
}

// TestApplyPermanentProviderFailure_NonAccountCodeKeepsBankStatus guards the
// boundary: a permanent code that blames our payload (preflight_validation)
// fails the request but must NOT brand the employee's stored bank account.
func TestApplyPermanentProviderFailure_NonAccountCodeKeepsBankStatus(t *testing.T) {
	reqRepo := &stubAdvanceRequestRepo{
		req: &domain.AdvancePaymentRequest{
			ID:            300,
			EmployeeID:    42,
			RequestAmount: 50_000,
			Status:        domain.AdvancePaymentStatusApproved,
		},
	}
	empRepo := &stubEmployeeRepo{}
	svc := disbursement.NewWalletPaymentService(
		&stubWalletPaymentRepo{}, reqRepo, empRepo, nil, nil, nil, nil, nil, testLogger(),
	)

	w := &DisbursementExecuteWorker{
		walletPaymentService: svc,
		advanceReqRepo:       reqRepo,
		employeeNotifier:     &recordingEmployeeNotifier{},
		logger:               testLogger(),
	}

	w.applyPermanentProviderFailure(context.Background(),
		DisbursementExecutePayload{AdvanceRequestID: 300},
		"preflight_validation", "onepay: Amount must be at least 100000 VND")

	if len(reqRepo.statuses) != 1 || reqRepo.statuses[0] != domain.AdvancePaymentStatusFailed {
		t.Fatalf("request statuses = %v, want one FAILED", reqRepo.statuses)
	}
	if empRepo.updatedID != 0 {
		t.Fatalf("employee bank columns written for a non-account code: %v", empRepo.columns)
	}
}

// TestProcessJob_AppliesStageAwareRetryPolicy drives a whole
// disbursement:execute task through both provider stages for the codes that made
// retrying expensive: a permanent one must fail the request (and flag the
// account when the code blames the recipient's data), a transient one must
// leave the request APPROVED so the poller's bounded orphan recovery retries it.
func TestProcessJob_AppliesStageAwareRetryPolicy(t *testing.T) {
	const (
		stageTransfer     = "transfer"
		stageAccountCheck = "account_check"
	)

	cases := []struct {
		name              string
		stage             string
		code              string
		message           string
		wantRequestFailed bool
		wantBankFlagged   bool
	}{
		{"transfer/14 permanent", stageTransfer, "14", "Thông tin thẻ không hợp lệ", true, true},
		{"transfer/86 transient", stageTransfer, "86", "Chức năng tạm thời đóng", false, false},
		{"account-check/name_mismatch permanent", stageAccountCheck, "name_mismatch", "Tên chủ tài khoản không khớp", true, true},
		{"account-check/86 transient", stageAccountCheck, "86", "Chức năng tạm thời đóng", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqRepo := &stubAdvanceRequestRepo{
				req: &domain.AdvancePaymentRequest{
					ID:            256,
					EmployeeID:    9,
					RequestAmount: 5_000_000,
					NetAmount:     4_996_150,
					Status:        domain.AdvancePaymentStatusApproved,
				},
			}
			wpRepo := newStubWalletPaymentRepo(0)
			empRepo := &stubEmployeeRepo{}
			svc := disbursement.NewWalletPaymentService(
				wpRepo, reqRepo, empRepo, nil, nil, nil, nil, nil, testLogger(),
			)

			base := &fakeDisbursementProvider{result: &infrastructure.TransferResult{
				RequestID:    "tt1",
				Status:       infrastructure.TransferStatusFailed,
				RawErrorCode: tc.code,
				RawMessage:   tc.message,
			}}
			var provider infrastructure.DisbursementProvider = base
			if tc.stage == stageAccountCheck {
				base.result = nil // the account check must stop the task before any transfer
				provider = &fakeVerifyingProvider{
					fakeDisbursementProvider: base,
					check: &infrastructure.AccountCheckResult{
						Valid:        false,
						RawErrorCode: tc.code,
						RawMessage:   tc.message,
					},
				}
			}

			notifier := &recordingEmployeeNotifier{}
			w := NewDisbursementExecuteWorker(
				svc, disbursement.NewRegistry(provider), nil, nil, reqRepo, notifier, testLogger(),
			)

			payload, err := json.Marshal(DisbursementExecutePayload{
				RequestID:          "tt1",
				AdvanceRequestID:   256,
				RequestedAmount:    4_996_150,
				RecipientName:      "NGUYEN VAN A",
				RecipientAccountNo: "1023020330000",
				RecipientBank:      "VCB",
			})
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			if err := w.ProcessJob(context.Background(), asynqlib.NewTask(TaskDisbursementExecute, payload)); err != nil {
				t.Fatalf("ProcessJob: %v", err)
			}

			row, err := wpRepo.GetByRequestID(context.Background(), "tt1")
			if err != nil {
				t.Fatalf("wallet_payment not recorded: %v", err)
			}
			if row.Status != domaintx.StateFailed {
				t.Fatalf("wallet_payment status = %q, want failed", row.Status)
			}

			requestFailed := len(reqRepo.statuses) > 0
			if requestFailed != tc.wantRequestFailed {
				t.Fatalf("advance request statuses = %v, want failed=%v", reqRepo.statuses, tc.wantRequestFailed)
			}
			if tc.wantRequestFailed && reqRepo.refs[0] != tc.code {
				t.Errorf("payment_reference = %q, want %q", reqRepo.refs[0], tc.code)
			}
			if got := len(notifier.failed); (got > 0) != tc.wantRequestFailed {
				t.Errorf("employee notifications = %d, want failed=%v", got, tc.wantRequestFailed)
			}
			if bankFlagged := empRepo.updatedID != 0; bankFlagged != tc.wantBankFlagged {
				t.Errorf("employee bank account flagged = %v, want %v (columns %v)",
					bankFlagged, tc.wantBankFlagged, empRepo.columns)
			}
			if tc.wantBankFlagged && empRepo.columns["bank_account_status"] != domain.BankAccountStatusInvalid {
				t.Errorf("bank_account_status = %v, want invalid", empRepo.columns["bank_account_status"])
			}
		})
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stubAdvanceRequestRepo implements the two methods the terminal-failure path
// touches. The embedded (nil) interface makes any other call a loud panic
// instead of a silent no-op.
type stubAdvanceRequestRepo struct {
	domain.AdvancePaymentRequestRepository

	req      *domain.AdvancePaymentRequest
	statuses []domain.AdvancePaymentRequestStatus
	refs     []string
}

func (r *stubAdvanceRequestRepo) GetByID(context.Context, uint64) (*domain.AdvancePaymentRequest, error) {
	return r.req, nil
}

func (r *stubAdvanceRequestRepo) UpdateStatus(_ context.Context, _ uint64, status domain.AdvancePaymentRequestStatus, paymentRef string, _ *time.Time) error {
	r.req.Status = status
	r.statuses = append(r.statuses, status)
	r.refs = append(r.refs, paymentRef)
	return nil
}

// stubEmployeeRepo records the column map written by MarkRecipientBankAccountInvalid.
type stubEmployeeRepo struct {
	domain.EmployeeRepository

	updatedID uint
	columns   map[string]any
}

func (r *stubEmployeeRepo) UpdateColumns(_ context.Context, id uint, columns map[string]any) error {
	r.updatedID = id
	r.columns = columns
	return nil
}

// stubWalletPaymentRepo is an in-memory WalletPaymentRepository: enough of the
// real contract (create, optimistic update, re-read) for the execute worker to
// drive a transfer through the FSM. The embedded interface keeps every method
// the tests don't implement a loud panic instead of a silent no-op.
type stubWalletPaymentRepo struct {
	domaintx.WalletPaymentRepository

	// failedAttempts is what CountFailedByEntityID reports, for any entity.
	failedAttempts int64

	byRequestID map[string]*domaintx.WalletPayment
	byID        map[uint64]*domaintx.WalletPayment
	nextID      uint64
}

func newStubWalletPaymentRepo(failedAttempts int64) *stubWalletPaymentRepo {
	return &stubWalletPaymentRepo{
		failedAttempts: failedAttempts,
		byRequestID:    map[string]*domaintx.WalletPayment{},
		byID:           map[uint64]*domaintx.WalletPayment{},
		nextID:         1,
	}
}

func (r *stubWalletPaymentRepo) CountFailedByEntityID(context.Context, uint64) (int64, error) {
	return r.failedAttempts, nil
}

func (r *stubWalletPaymentRepo) Create(_ context.Context, t *domaintx.WalletPayment) error {
	if _, exists := r.byRequestID[t.RequestID]; exists {
		return errors.New("Error 1062: Duplicate entry for idx_wp_request_id")
	}
	t.ID = r.nextID
	r.nextID++
	t.Version = 1
	stored := *t
	r.byRequestID[t.RequestID] = &stored
	r.byID[t.ID] = &stored
	return nil
}

func (r *stubWalletPaymentRepo) GetByRequestID(_ context.Context, requestID string) (*domaintx.WalletPayment, error) {
	row, ok := r.byRequestID[requestID]
	if !ok {
		return nil, domaintx.ErrNotFound
	}
	copied := *row
	return &copied, nil
}

func (r *stubWalletPaymentRepo) GetByID(_ context.Context, id uint64) (*domaintx.WalletPayment, error) {
	row, ok := r.byID[id]
	if !ok {
		return nil, domaintx.ErrNotFound
	}
	copied := *row
	return &copied, nil
}

func (r *stubWalletPaymentRepo) HasPendingForRecipient(context.Context, string, string, string) (bool, error) {
	return false, nil
}

func (r *stubWalletPaymentRepo) UpdateExpected(_ context.Context, id uint64, expectedVersion int64, patch domaintx.UpdatePatch) error {
	row, ok := r.byID[id]
	if !ok {
		return domaintx.ErrNotFound
	}
	if row.Version != expectedVersion {
		return domaintx.ErrConcurrentModification
	}
	row.Version++
	if patch.Status != nil {
		row.Status = *patch.Status
	}
	if patch.ErrorCode != nil {
		row.ErrorCode = patch.ErrorCode
	}
	if patch.ErrorMessage != nil {
		row.ErrorMessage = patch.ErrorMessage
	}
	if patch.Fee != nil {
		row.Fee = *patch.Fee
	}
	if patch.InvoiceNo != nil {
		row.InvoiceNo = patch.InvoiceNo
	}
	return nil
}

// fakeDisbursementProvider returns one canned transfer result. It deliberately
// does NOT implement AccountVerifier, so the execute worker skips the account
// check and the test exercises the funds-transfer stage.
type fakeDisbursementProvider struct {
	result        *infrastructure.TransferResult
	err           error
	transferCalls int
}

func (p *fakeDisbursementProvider) Name() string { return "9pay" }

func (p *fakeDisbursementProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	p.transferCalls++
	return p.result, p.err
}

func (p *fakeDisbursementProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, errors.New("webhook parsing not used in this test")
}

// fakeVerifyingProvider adds the optional AccountVerifier capability so the
// execute worker runs the account-check stage before any transfer.
type fakeVerifyingProvider struct {
	*fakeDisbursementProvider
	check *infrastructure.AccountCheckResult
}

func (p *fakeVerifyingProvider) CheckAccount(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	return p.check, nil
}

type recordingEmployeeNotifier struct {
	failed []struct {
		employeeID uint
		amount     uint64
		detail     string
	}
}

func (n *recordingEmployeeNotifier) NotifyTimesheetPaid(context.Context, []*domain.Timesheet) {}

func (n *recordingEmployeeNotifier) NotifyAdvancePaymentStatusChanged(context.Context, uint, domain.AdvancePaymentRequestStatus, uint64) {
}

func (n *recordingEmployeeNotifier) NotifyAdvancePaymentFailed(_ context.Context, employeeID uint, amount uint64, detail string) {
	n.failed = append(n.failed, struct {
		employeeID uint
		amount     uint64
		detail     string
	}{employeeID, amount, detail})
}
