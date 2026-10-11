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

func (f *fakeSyncPaymentRepo) sumWhere(statuses []string, unreconciledOnly bool) int64 {
	want := map[string]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	var total int64
	for _, r := range f.rows {
		if want[r.status] && (!unreconciledOnly || !r.reconciled) {
			total += r.amount
		}
	}
	return total
}

func (f *fakeSyncPaymentRepo) SumByStatuses(_ context.Context, statuses []string) (int64, error) {
	return f.sumWhere(statuses, false), nil
}

func (f *fakeSyncPaymentRepo) SumUnreconciledByStatuses(_ context.Context, statuses []string) (int64, error) {
	return f.sumWhere(statuses, true), nil
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

// SyncBalance is unguarded by design (owner decision 2026-10-11): the provider
// balance is the source of truth and the 5-minute cron re-converges, so
// nothing — in-flight payments or unreconciled failures, fresh or stale — may
// block the adjustment. The 2026-10 prod incident (126 stale unreconciled rows
// freezing the sync for months) is the reason these tests exist.

func TestSyncBalanceAdjustsDuringInFlightPayments(t *testing.T) {
	inFlight := fakePaymentRow{status: "authorised", amount: 5_000_000, createdAt: clock.Now()}
	svc, topups := newSyncTestService(100_000_000, 101_070_300, inFlight)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if !result.Adjusted {
		t.Fatalf("in-flight payments must not block the adjustment, got adjusted=%v", result.Adjusted)
	}
	// local = 100M topups − 5M in-flight = 95M; provider 101,070,300 → +6,070,300.
	if len(topups.created) != 1 {
		t.Fatalf("expected 1 sync topup, got %d", len(topups.created))
	}
	if got := topups.created[0].Amount; got != 6_070_300 {
		t.Fatalf("topup amount = %d, want 6070300", got)
	}
	if result.LocalBalance != 101_070_300 {
		t.Fatalf("refreshed local balance = %d, want 101070300", result.LocalBalance)
	}
}

func TestSyncBalanceAdjustsDespiteUnreconciledFailures(t *testing.T) {
	stale := fakePaymentRow{status: "failed", amount: 354_071_140, createdAt: clock.Now().AddDate(0, 0, -30)}
	recent := fakePaymentRow{status: "failed", amount: 2_499_000, createdAt: clock.Now().AddDate(0, 0, -1)}
	svc, topups := newSyncTestService(100_000_000, 101_070_300, stale, recent)

	result, err := svc.SyncBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncBalance: %v", err)
	}
	if !result.Adjusted {
		t.Fatal("unreconciled failures (stale or recent) must not block the adjustment")
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
