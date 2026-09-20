package workers

import (
	"context"
	"testing"
	"time"

	disbursement "api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

func TestRequestBudgetMonth_PrefersRequestAdvancePaymentMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	req := &domain.AdvancePaymentRequest{
		AdvancePayment: &domain.AdvancePayment{ForMonth: "2026-06"},
	}

	got := requestBudgetMonth(req)
	if got != "2026-06" {
		t.Fatalf("expected budget month 2026-06 from linked advance payment, got %s", got)
	}
}

func TestRequestBudgetMonth_FallsBackToCurrentAdvanceMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	got := requestBudgetMonth(&domain.AdvancePaymentRequest{})
	if got != "2026-06" {
		t.Fatalf("expected fallback budget month 2026-06 on 2026-06-20, got %s", got)
	}
}

// TestWithinRetryBudget pins the transient-retry boundary: the fifth recorded
// failure exhausts the budget, so the request is never enqueued a sixth time.
func TestWithinRetryBudget(t *testing.T) {
	cases := []struct {
		failed int64
		want   bool
	}{
		{0, true},
		{1, true},
		{4, true},                              // 5th attempt still allowed
		{maxFailedDisbursementAttempts, false}, // 5 failures → budget spent
		{13, false},                            // the error-86 loop that motivated this bound
	}
	for _, tc := range cases {
		if got := withinRetryBudget(tc.failed); got != tc.want {
			t.Errorf("withinRetryBudget(%d) = %v, want %v", tc.failed, got, tc.want)
		}
	}
}

// TestEnqueueRequest_RetryBudgetExhaustedFailsRequestWithoutEnqueue covers the
// orphan-recovery loop end to end for one request: with 5 failed wallet_payments
// rows already recorded, the poller must mark the request FAILED
// (retry_limit_exceeded) and return before touching the provider — the run
// below has no asynq client and no employee repository, so any attempt to
// enqueue or load bank details would panic.
func TestEnqueueRequest_RetryBudgetExhaustedFailsRequestWithoutEnqueue(t *testing.T) {
	reqRepo := &stubAdvanceRequestRepo{
		req: &domain.AdvancePaymentRequest{
			ID:            256,
			EmployeeID:    9,
			RequestAmount: 5_000_000,
			NetAmount:     4_996_150,
			Status:        domain.AdvancePaymentStatusApproved,
		},
	}
	svc := disbursement.NewWalletPaymentService(
		&stubWalletPaymentRepo{failedAttempts: 5}, reqRepo, nil, nil, nil, nil, nil, nil, testLogger(),
	)
	w := &DisbursementPollerWorker{
		advancePaymentReqRepo: reqRepo,
		walletPaymentService:  svc,
		logger:                testLogger(),
	}

	enqueued, err := w.enqueueRequest(context.Background(), reqRepo.req)
	if err != nil {
		t.Fatalf("enqueueRequest returned error: %v", err)
	}
	if enqueued {
		t.Fatal("request with 5 recorded failures was enqueued again")
	}
	if len(reqRepo.statuses) != 1 || reqRepo.statuses[0] != domain.AdvancePaymentStatusFailed {
		t.Fatalf("request statuses = %v, want one FAILED", reqRepo.statuses)
	}
	if reqRepo.refs[0] != retryLimitExceededReason {
		t.Fatalf("payment_reference = %q, want %q", reqRepo.refs[0], retryLimitExceededReason)
	}
}

// TestRetryBudgetSpent_UnderBudgetKeepsRetrying is the other side of the
// cap: a request that still has budget left must stay eligible for re-enqueue.
func TestRetryBudgetSpent_UnderBudgetKeepsRetrying(t *testing.T) {
	svc := disbursement.NewWalletPaymentService(
		&stubWalletPaymentRepo{failedAttempts: maxFailedDisbursementAttempts - 1},
		&stubAdvanceRequestRepo{}, nil, nil, nil, nil, nil, nil, testLogger(),
	)
	w := &DisbursementPollerWorker{walletPaymentService: svc, logger: testLogger()}

	spent, err := w.retryBudgetSpent(context.Background(), &domain.AdvancePaymentRequest{ID: 256})
	if err != nil {
		t.Fatalf("retryBudgetSpent: %v", err)
	}
	if spent {
		t.Fatal("request with 4 recorded failures must stay within its retry budget")
	}
}
