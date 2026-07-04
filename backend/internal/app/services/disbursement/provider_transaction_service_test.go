package disbursement_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"

	"github.com/google/uuid"
)

// fakeProviderTxRepo is a minimum in-memory WalletPaymentRepository
// that mimics the production unique-key behavior on request_id. The
// service-layer tests only exercise Create + GetByRequestID — the FSM
// transitions are covered by the repository tests against SQLite — so
// the rest of the interface is left as no-ops returning errors loud
// enough to fail any test that accidentally calls them.
type fakeProviderTxRepo struct {
	byRequestID map[string]*domaintx.WalletPayment
	createCalls int
	createErr   error
}

func newFakeRepo() *fakeProviderTxRepo {
	return &fakeProviderTxRepo{byRequestID: map[string]*domaintx.WalletPayment{}}
}

func (r *fakeProviderTxRepo) Create(_ context.Context, t *domaintx.WalletPayment) error {
	r.createCalls++
	if r.createErr != nil {
		return r.createErr
	}
	if _, exists := r.byRequestID[t.RequestID]; exists {
		// Mimic MySQL's "Duplicate entry ... for key" — the substring
		// the service layer matches on. Production driver returns the
		// real MySQL message; this test relies on the same substring.
		return errors.New("Error 1062: Duplicate entry '" + t.RequestID + "' for key 'idx_pt_request_id'")
	}
	t.ID = uint64(len(r.byRequestID) + 1)
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt
	// Store a copy so further mutations on the returned row don't bleed
	// into the "existing" row that idempotent retry returns.
	stored := *t
	r.byRequestID[t.RequestID] = &stored
	return nil
}

func (r *fakeProviderTxRepo) GetByRequestID(_ context.Context, requestID string) (*domaintx.WalletPayment, error) {
	row, ok := r.byRequestID[requestID]
	if !ok {
		return nil, domaintx.ErrNotFound
	}
	copy := *row
	return &copy, nil
}

// Unused interface methods — the Initiate path doesn't reach these.
// Returning ErrNotFound (or panicking on UpdateExpected) keeps the
// surface honest if a future test accidentally hits one.
func (r *fakeProviderTxRepo) GetByID(context.Context, uint64) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}
func (r *fakeProviderTxRepo) GetByInvoiceNo(context.Context, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}
func (r *fakeProviderTxRepo) GetByProviderInvoiceNo(context.Context, string, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}
func (r *fakeProviderTxRepo) ListByProviderAndCreatedRange(context.Context, string, time.Time, time.Time) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}
func (r *fakeProviderTxRepo) GetByTxnID(context.Context, string) (*domaintx.WalletPayment, error) {
	return nil, domaintx.ErrNotFound
}
func (r *fakeProviderTxRepo) ListRecent(_ context.Context, limit int) ([]*domaintx.WalletPayment, error) {
	return nil, nil
}
func (r *fakeProviderTxRepo) UpdateExpected(context.Context, uint64, int64, domaintx.UpdatePatch) error {
	panic("UpdateExpected: unexpected call in Initiate-only test")
}
func (r *fakeProviderTxRepo) StatsByErrorCode(context.Context, time.Time, time.Time) ([]domaintx.ErrorCodeStat, error) {
	return nil, nil
}
func (r *fakeProviderTxRepo) StatsByStatus(context.Context, time.Time, time.Time) ([]domaintx.StatusStat, error) {
	return nil, nil
}

func (r *fakeProviderTxRepo) MarkReconciled(_ context.Context, id uint64, status domaintx.State, reconciledAt time.Time) error {
	row, ok := r.byRequestID[""]
	_ = row
	_ = ok
	_ = id
	_ = status
	_ = reconciledAt
	return nil
}

func (r *fakeProviderTxRepo) ListByStatuses(_ context.Context, statuses []domaintx.State, from, to time.Time) ([]*domaintx.WalletPayment, error) {
	_ = statuses
	_ = from
	_ = to
	return nil, nil
}

func (r *fakeProviderTxRepo) ListByBatchID(_ context.Context, batchID string) ([]*domaintx.WalletPayment, error) {
	_ = batchID
	return nil, nil
}

func (r *fakeProviderTxRepo) ListStaleAuthorised(_ context.Context, provider string, cutoff time.Time, limit int) ([]*domaintx.WalletPayment, error) {
	_ = provider
	_ = cutoff
	_ = limit
	return nil, nil
}

func (r *fakeProviderTxRepo) HasPendingForRecipient(_ context.Context, accountNo, bank, provider string) (bool, error) {
	_ = accountNo
	_ = bank
	_ = provider
	return false, nil
}

func (r *fakeProviderTxRepo) HasNonTerminalByEntityID(_ context.Context, entityID uint64) (bool, error) {
	_ = entityID
	return false, nil
}

// fakeFee is a stub DisbursementFeeProvider that always returns the
// fixed VND amount it was constructed with. Lets tests assert the
// fee landed on the row without spinning up the settings stack.
type fakeFee int64

func (f fakeFee) GetDisbursementFeeVND(_ context.Context, _ string) (int64, bool, error) {
	return int64(f), false, nil
}

// errorFee is a stub DisbursementFeeProvider whose resolution always fails.
// Used to pin M12: a fee-resolution error must propagate out of Initiate
// wrapped with ErrFeeResolution instead of being swallowed into fee=0 (the
// legacy behavior that silently lost money on missing schedules).
type errorFee struct{}

func (errorFee) GetDisbursementFeeVND(_ context.Context, _ string) (int64, bool, error) {
	return 0, false, errors.New("no active disbursement fee schedule")
}

// TestInitiate_PropagatesFeeResolutionError pins M12: when the fee provider
// returns an error, Initiate surfaces it wrapped with ErrFeeResolution and
// creates no wallet_payment row. The worker's errors.Is(err, ErrFeeResolution)
// branch turns this into a terminal failure instead of stamping a guessed fee.
func TestInitiate_PropagatesFeeResolutionError(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, errorFee{}, nil, nil, slog.Default())

	in := disbursement.InitiateInput{
		RequestID:          "tt" + uuid.New().String()[:14],
		RequestedAmount:    150_000,
		RecipientName:      "NGUYEN VAN A",
		RecipientAccountNo: "9999000011",
		RecipientBank:      "VCB",
	}
	_, err := svc.Initiate(context.Background(), in)
	if err == nil {
		t.Fatal("Initiate: expected fee-resolution error, got nil")
	}
	if !errors.Is(err, disbursement.ErrFeeResolution) {
		t.Fatalf("Initiate error not wrapped with ErrFeeResolution: %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected no row created on fee-resolution failure, got %d Create call(s)", repo.createCalls)
	}
}

// TestInitiate_PersistsRequestIDFromCaller documents the contract
// production employee disbursements rely on: whatever request_id the
// caller passes in (e.g. the existing "tingting<...>" transaction
// code from advance_payment exports) lands on the row verbatim.
func TestInitiate_PersistsRequestIDFromCaller(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, fakeFee(200), nil, nil, slog.Default())

	in := disbursement.InitiateInput{
		RequestID:          "tt" + uuid.New().String()[:14],
		RequestedAmount:    150_000,
		RecipientName:      "NGUYEN VAN A",
		RecipientAccountNo: "9999000011",
		RecipientBank:      "VCB",
	}
	row, err := svc.Initiate(context.Background(), in)
	if err != nil {
		t.Fatalf("Initiate: %v", err)
	}
	if row.RequestID != in.RequestID {
		t.Fatalf("row.RequestID = %q, want %q (caller-provided code must be persisted verbatim)", row.RequestID, in.RequestID)
	}
	if row.Status != domaintx.StatePending {
		t.Fatalf("row.Status = %q, want verifying (Initiate inserts pre-account-check)", row.Status)
	}
	if row.Fee != 200 {
		t.Fatalf("row.Fee = %d, want 200 (stamped from fee provider at INSERT)", row.Fee)
	}
}

// TestRecordIPN_RejectsAmountMismatch pins M8: an inbound IPN whose stated
// amount differs from the persisted wallet_payment.RequestedAmount (the amount
// we asked the provider to disburse) is rejected with ErrIPNAmountMismatch
// before the FSM applies it. A matching amount is not rejected on amount
// grounds (the FSM may still reject for other reasons, but not this one).
func TestRecordIPN_RejectsAmountMismatch(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, fakeFee(200), nil, nil, slog.Default())

	row := &domaintx.WalletPayment{
		TxnID:              uuid.New(),
		RequestID:          "req-mismatch-" + uuid.NewString()[:8],
		RequestedAmount:    10_000,
		RecipientName:      "NGUYEN VAN A",
		RecipientAccountNo: "9999000011",
		RecipientBank:      "VCB",
		Status:             domaintx.StatePending,
	}
	if err := repo.Create(context.Background(), row); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	// Forged amount (9_999 vs requested 10_000) → rejected before the FSM.
	_, err := svc.RecordIPN(context.Background(), disbursement.IPNResult{
		Provider:  "9pay",
		RequestID: row.RequestID,
		Amount:    9_999,
		Status:    infrastructure.TransferStatusSuccess,
	})
	if !errors.Is(err, disbursement.ErrIPNAmountMismatch) {
		t.Fatalf("mismatched amount: err = %v, want ErrIPNAmountMismatch", err)
	}

	// Matching amount → not an amount-mismatch rejection.
	_, err = svc.RecordIPN(context.Background(), disbursement.IPNResult{
		Provider:  "9pay",
		RequestID: row.RequestID,
		Amount:    10_000,
		Status:    infrastructure.TransferStatusSuccess,
	})
	if errors.Is(err, disbursement.ErrIPNAmountMismatch) {
		t.Fatalf("matching amount: err = ErrIPNAmountMismatch, want none (or a non-amount error): %v", err)
	}
}

// TestInitiate_IsIdempotentOnDuplicateRequestID covers the manual
// re-fire scenario: an admin retries a disbursement against the same
// advance_payment_requests row, so the same transaction code arrives
// again. The unique index on request_id rejects the second INSERT;
// Initiate must catch that and return the EXISTING row at its current
// state rather than surfacing a duplicate-key error to the caller.
func TestInitiate_IsIdempotentOnDuplicateRequestID(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, fakeFee(200), nil, nil, slog.Default())

	requestID := "tt" + uuid.New().String()[:14]
	first, err := svc.Initiate(context.Background(), disbursement.InitiateInput{
		RequestID:          requestID,
		RequestedAmount:    150_000,
		RecipientName:      "NGUYEN VAN A",
		RecipientAccountNo: "9999000011",
		RecipientBank:      "VCB",
	})
	if err != nil {
		t.Fatalf("first Initiate: %v", err)
	}

	// Simulate retry — same request_id, same payload. Should not error.
	second, err := svc.Initiate(context.Background(), disbursement.InitiateInput{
		RequestID:          requestID,
		RequestedAmount:    150_000,
		RecipientName:      "NGUYEN VAN A",
		RecipientAccountNo: "9999000011",
		RecipientBank:      "VCB",
	})
	if err != nil {
		t.Fatalf("retry Initiate must be idempotent, got error: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("retry returned a different row: first.ID=%d, second.ID=%d", first.ID, second.ID)
	}
	if second.TxnID != first.TxnID {
		t.Fatalf("retry must return the existing TxnID, got %s, want %s", second.TxnID, first.TxnID)
	}
	if second.RequestID != requestID {
		t.Fatalf("retry RequestID = %q, want %q", second.RequestID, requestID)
	}
	// Both Create attempts hit the repo, but only one row exists.
	if repo.createCalls != 2 {
		t.Fatalf("repo.createCalls = %d, want 2 (original + duplicate-rejected retry)", repo.createCalls)
	}
	if got := len(repo.byRequestID); got != 1 {
		t.Fatalf("repo row count = %d, want 1", got)
	}
}

// TestInitiate_NonDuplicateErrorPropagates ensures the duplicate-key
// catch is narrow: any other Create error still propagates. Otherwise a
// production driver/connectivity failure would silently turn into a
// "row exists" lookup that returns ErrNotFound.
func TestInitiate_NonDuplicateErrorPropagates(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	repo.createErr = errors.New("connection refused")
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, fakeFee(200), nil, nil, slog.Default())

	_, err := svc.Initiate(context.Background(), disbursement.InitiateInput{
		RequestID:          "tingtingabcdef",
		RequestedAmount:    100,
		RecipientName:      "x",
		RecipientAccountNo: "y",
		RecipientBank:      "z",
	})
	if err == nil {
		t.Fatal("expected error to propagate, got nil")
	}
}

// fakeProviderTxRepoWithUpdate extends fakeProviderTxRepo to support the
// optimistic-locking update + GetByID path used by RecordIPN transitions.
type fakeProviderTxRepoWithUpdate struct {
	fakeProviderTxRepo
	byID   map[uint64]*domaintx.WalletPayment
	nextID uint64
}

func newFakeRepoWithUpdate() *fakeProviderTxRepoWithUpdate {
	return &fakeProviderTxRepoWithUpdate{
		fakeProviderTxRepo: *newFakeRepo(),
		byID:               map[uint64]*domaintx.WalletPayment{},
		nextID:             1,
	}
}

func (r *fakeProviderTxRepoWithUpdate) Create(_ context.Context, t *domaintx.WalletPayment) error {
	if _, exists := r.byRequestID[t.RequestID]; exists {
		return errors.New("Error 1062: Duplicate entry")
	}
	t.ID = r.nextID
	r.nextID++
	t.Version = 1
	stored := *t
	r.byRequestID[t.RequestID] = &stored
	r.byID[t.ID] = &stored
	return nil
}

func (r *fakeProviderTxRepoWithUpdate) GetByID(_ context.Context, id uint64) (*domaintx.WalletPayment, error) {
	row, ok := r.byID[id]
	if !ok {
		return nil, domaintx.ErrNotFound
	}
	copy := *row
	return &copy, nil
}

func (r *fakeProviderTxRepoWithUpdate) UpdateExpected(_ context.Context, id uint64, expectedVersion int64, patch domaintx.UpdatePatch) error {
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
	return nil
}

// TestRecordIPN_ReversedTransition verifies the completed→reversed FSM
// transition: when a provider sends a "reversed" IPN for a payment that
// was already completed, the row should move to the reversed state.
func TestRecordIPN_ReversedTransition(t *testing.T) {
	t.Parallel()

	repo := newFakeRepoWithUpdate()
	svc := disbursement.NewWalletPaymentService(repo, nil, nil, nil, nil, fakeFee(200), nil, nil, slog.Default())

	// Create a row in completed state (simulating a settled payment).
	txnID := uuid.New()
	row := &domaintx.WalletPayment{
		TxnID:     txnID,
		RequestID: "TF-reversal-test",
		Provider:  "1pay",
		Status:    domaintx.StateCompleted,
		Version:   1,
	}
	if err := repo.Create(context.Background(), row); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Fire the reversed IPN.
	result, err := svc.RecordIPN(context.Background(), disbursement.IPNResult{
		Provider:      "1pay",
		Status:        infrastructure.TransferStatusReversed,
		RequestID:     "TF-reversal-test",
		RawErrorCode:  "",
		FailureReason: "Bank reversal test",
	})
	if err != nil {
		t.Fatalf("RecordIPN reversed: %v", err)
	}

	if result.Status != domaintx.StateReversed {
		t.Fatalf("row.Status = %q, want %q", result.Status, domaintx.StateReversed)
	}
	if !result.IsTerminal() {
		t.Fatal("reversed should be terminal")
	}
}
