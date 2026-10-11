package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/domain/wallet"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// ── Fakes ──────────────────────────────────────────────────────────────────

type fakeSyncTopupRepo struct {
	wallet.WalletTopupRepository
	sum     int64
	created []*wallet.WalletTopup
}

func (f *fakeSyncTopupRepo) Sum(context.Context) (int64, error) { return f.sum, nil }

func (f *fakeSyncTopupRepo) GetByBankRef(context.Context, string) (*wallet.WalletTopup, error) {
	return nil, errors.New("not found")
}

func (f *fakeSyncTopupRepo) Create(_ context.Context, t *wallet.WalletTopup) error {
	f.created = append(f.created, t)
	f.sum += t.Amount // mirrors the real repo: an inserted topup raises Sum
	return nil
}

type fakePaymentRow struct {
	status     string
	amount     int64
	reconciled bool
	createdAt  time.Time
}

type fakeSyncPaymentRepo struct {
	wallet.WalletPaymentRepository
	rows []fakePaymentRow
}

func (f *fakeSyncPaymentRepo) sumWhere(statuses []string, unreconciledOnly bool, since *time.Time) int64 {
	want := map[string]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	var total int64
	for _, r := range f.rows {
		if !want[r.status] {
			continue
		}
		if unreconciledOnly && r.reconciled {
			continue
		}
		if since != nil && r.createdAt.Before(*since) {
			continue
		}
		total += r.amount
	}
	return total
}

func (f *fakeSyncPaymentRepo) SumByStatuses(_ context.Context, statuses []string) (int64, error) {
	return f.sumWhere(statuses, false, nil), nil
}

func (f *fakeSyncPaymentRepo) SumUnreconciledByStatuses(_ context.Context, statuses []string) (int64, error) {
	return f.sumWhere(statuses, true, nil), nil
}

func (f *fakeSyncPaymentRepo) SumUnreconciledByStatusesSince(_ context.Context, statuses []string, since time.Time) (int64, error) {
	return f.sumWhere(statuses, true, &since), nil
}

type fakeSyncRegistry struct {
	balance int64
}

func (f *fakeSyncRegistry) ActiveProviderName(context.Context) (string, error) {
	return "onepay", nil
}

func (f *fakeSyncRegistry) GetProviderBalance(context.Context) (int64, error) {
	return f.balance, nil
}

func newSyncTestService(topupSum, providerBalance int64, rows ...fakePaymentRow) (*walletService, *fakeSyncTopupRepo) {
	topups := &fakeSyncTopupRepo{sum: topupSum}
	return &walletService{
		topupRepo:   topups,
		paymentRepo: &fakeSyncPaymentRepo{rows: rows},
		registry:    &fakeSyncRegistry{balance: providerBalance},
		logger:      observability.GetLogger(),
	}, topups
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestSyncBalanceAdjustsDespiteStaleUnreconciledFailures(t *testing.T) {
	// Prod regression (2026-10): 126 unreconciled failed rows dating back to
	// June froze the 5-minute sync forever — the provider showed 1,070,300
	// more than local and only the manual dialog could reconcile. Failures
	// older than the guard window must not block the auto-adjustment.
	stale := fakePaymentRow{status: "failed", amount: 354_071_140, createdAt: clock.Now().AddDate(0, 0, -30)}
	svc, topups := newSyncTestService(100_000_000, 101_070_300, stale)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if !result.Adjusted {
		t.Fatalf("expected auto-adjust with only stale failures, got adjusted=%v provider=%d local=%d",
			result.Adjusted, result.ProviderBalance, result.LocalBalance)
	}
	if len(topups.created) != 1 {
		t.Fatalf("expected 1 sync topup, got %d", len(topups.created))
	}
	if got := topups.created[0].Amount; got != 1_070_300 {
		t.Fatalf("topup amount = %d, want 1070300", got)
	}
	if result.LocalBalance != 101_070_300 {
		t.Fatalf("refreshed local balance = %d, want 101070300", result.LocalBalance)
	}
}

func TestSyncBalanceSkipsWhenRecentFailureUnreconciled(t *testing.T) {
	stale := fakePaymentRow{status: "failed", amount: 354_071_140, createdAt: clock.Now().AddDate(0, 0, -30)}
	recent := fakePaymentRow{status: "failed", amount: 2_499_000, createdAt: clock.Now().AddDate(0, 0, -1)}
	svc, topups := newSyncTestService(100_000_000, 101_070_300, stale, recent)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if result.Adjusted {
		t.Fatal("a recent unreconciled failure must block the adjustment")
	}
	if len(topups.created) != 0 {
		t.Fatalf("no topup expected while a recent failure is unreconciled, got %d", len(topups.created))
	}
}

func TestSyncBalanceSkipsWhenPaymentsInFlight(t *testing.T) {
	inFlight := fakePaymentRow{status: "authorised", amount: 5_000_000, createdAt: clock.Now()}
	svc, topups := newSyncTestService(100_000_000, 101_070_300, inFlight)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if result.Adjusted {
		t.Fatal("in-flight payments must block the adjustment")
	}
	if len(topups.created) != 0 {
		t.Fatalf("no topup expected while payments are in flight, got %d", len(topups.created))
	}
}

func TestSyncBalanceNoAdjustmentWhenBalancesMatch(t *testing.T) {
	svc, topups := newSyncTestService(100_000_000, 100_000_000)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if result.Adjusted {
		t.Fatal("matching balances must not create a topup")
	}
	if len(topups.created) != 0 {
		t.Fatalf("no topup expected, got %d", len(topups.created))
	}
	if result.LocalBalance != 100_000_000 {
		t.Fatalf("local balance = %d, want 100000000", result.LocalBalance)
	}
}
