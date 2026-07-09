package services

import (
	"context"
	"testing"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	walletdomain "api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

type walletDemandForecastRequestRepoStub struct {
	domain.AdvancePaymentRequestRepository
	rows   []domain.CohortRow
	months []string
}

func (r *walletDemandForecastRequestRepoStub) GetCohortByMonths(_ context.Context, forMonths []string) ([]domain.CohortRow, error) {
	r.months = append([]string(nil), forMonths...)
	return r.rows, nil
}

type walletDemandForecastWalletStub struct {
	walletdomain.WalletService
	balance *walletdomain.WalletBalance
}

func (w *walletDemandForecastWalletStub) GetBalance(context.Context) (*walletdomain.WalletBalance, error) {
	return w.balance, nil
}

func TestWalletDemandForecastLockedGapHasNoTopUpNeed(t *testing.T) {
	now := time.Date(2026, 7, 9, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-06", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 90_000_000},
			{ForMonth: "2026-06", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 45_000_000},
			{ForMonth: "2026-05", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 90_000_000},
			{ForMonth: "2026-05", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 45_000_000},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 0}}

	svc := NewWalletDemandForecastService(repo, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 3,
		LeadDays:      2,
	})
	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	if got.CurrentForMonth != "2026-07" {
		t.Fatalf("CurrentForMonth = %q, want 2026-07", got.CurrentForMonth)
	}
	if got.CurrentCycleDay != 0 {
		t.Fatalf("CurrentCycleDay = %d, want 0 outside request window", got.CurrentCycleDay)
	}
	if got.Prediction.HorizonCycleDay != 0 {
		t.Fatalf("HorizonCycleDay = %d, want 0 before the lead window reaches day 20", got.Prediction.HorizonCycleDay)
	}
	if got.Prediction.RecommendedBalance != 0 {
		t.Fatalf("RecommendedBalance = %d, want 0", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != 0 {
		t.Fatalf("Shortfall = %d, want 0", got.Prediction.Shortfall)
	}
	if got.Prediction.P50Reference <= 0 || got.Prediction.P90Reference <= 0 || got.Prediction.P99Reference <= 0 {
		t.Fatalf(
			"remaining-cycle forecast quantiles = p50:%d p90:%d p99:%d, want non-zero despite zero near-term top-up need",
			got.Prediction.P50Reference,
			got.Prediction.P90Reference,
			got.Prediction.P99Reference,
		)
	}
	if len(repo.months) == 0 || repo.months[0] != "2026-07" {
		t.Fatalf("queried months = %v, want current effective period first", repo.months)
	}
}

func TestWalletDemandForecastForMonthSelection(t *testing.T) {
	cases := []struct {
		desc string
		now  time.Time
		want string
	}{
		{
			desc: "tail_window_uses_open_previous_period",
			now:  time.Date(2026, 7, 8, 9, 0, 0, 0, clock.DefaultLocation),
			want: "2026-06",
		},
		{
			desc: "locked_gap_looks_at_upcoming_period",
			now:  time.Date(2026, 7, 9, 9, 0, 0, 0, clock.DefaultLocation),
			want: "2026-07",
		},
		{
			desc: "period_start_uses_newly_opened_period",
			now:  time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation),
			want: "2026-07",
		},
		{
			desc: "cutoff_uses_open_previous_period",
			now:  time.Date(2026, 8, 8, 9, 0, 0, 0, clock.DefaultLocation),
			want: "2026-07",
		},
	}

	for _, c := range cases {
		if got := forecastForMonth(c.now); got != c.want {
			t.Errorf("%s: forecastForMonth = %q, want %q", c.desc, got, c.want)
		}
	}
}

func TestWalletDemandForecastLeadWindowReachesNextPeriod(t *testing.T) {
	now := time.Date(2026, 7, 18, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-06", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
			{ForMonth: "2026-05", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 60_000_000}}

	svc := NewWalletDemandForecastService(repo, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 3,
		LeadDays:      2,
	})
	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	if got.CurrentForMonth != "2026-07" {
		t.Fatalf("CurrentForMonth = %q, want 2026-07", got.CurrentForMonth)
	}
	if got.CurrentCycleDay != 0 {
		t.Fatalf("CurrentCycleDay = %d, want 0 before period start", got.CurrentCycleDay)
	}
	if got.Prediction.HorizonCycleDay != 1 {
		t.Fatalf("HorizonCycleDay = %d, want 1 when the two-day lead reaches day 20", got.Prediction.HorizonCycleDay)
	}
	if got.Prediction.RecommendedBalance != 100_000_000 {
		t.Fatalf("RecommendedBalance = %d, want 100000000", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != 40_000_000 {
		t.Fatalf("Shortfall = %d, want 40000000", got.Prediction.Shortfall)
	}
}
