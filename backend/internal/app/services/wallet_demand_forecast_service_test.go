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
	rows       []domain.CohortRow
	months     []string
	cycleState domain.AdvancePaymentCycleForecastState
}

func (r *walletDemandForecastRequestRepoStub) GetCohortByMonths(_ context.Context, forMonths []string) ([]domain.CohortRow, error) {
	r.months = append([]string(nil), forMonths...)
	return r.rows, nil
}

func (r *walletDemandForecastRequestRepoStub) GetTotalPayableAmount(context.Context) (int64, error) {
	var total int64
	for _, row := range r.rows {
		if row.Status == string(domain.AdvancePaymentStatusPending) ||
			row.Status == string(domain.AdvancePaymentStatusApproved) {
			total += row.TotalAmount
		}
	}
	return total, nil
}

func (r *walletDemandForecastRequestRepoStub) GetCycleForecastState(
	context.Context,
	string,
) (*domain.AdvancePaymentCycleForecastState, error) {
	state := r.cycleState
	return &state, nil
}

type walletDemandForecastWalletStub struct {
	walletdomain.WalletService
	balance *walletdomain.WalletBalance
}

func (w *walletDemandForecastWalletStub) GetBalance(context.Context) (*walletdomain.WalletBalance, error) {
	return w.balance, nil
}

func TestWalletDemandForecastLockedGapUsesUpcomingCycle(t *testing.T) {
	// Day 12 sits in the locked gap (request cutoff day 9 < day < period
	// start day 20): the forecast targets the upcoming, not-yet-open cycle.
	now := time.Date(2026, 7, 12, 9, 0, 0, 0, clock.DefaultLocation)
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
	if got.Prediction.RecommendedBalance <= 0 {
		t.Fatalf("RecommendedBalance = %d, want > 0 for the upcoming cycle", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != got.Prediction.RecommendedBalance {
		t.Fatalf(
			"Shortfall = %d, want RecommendedBalance %d for an empty wallet",
			got.Prediction.Shortfall,
			got.Prediction.RecommendedBalance,
		)
	}
	if got.Prediction.P50Reference <= 0 || got.Prediction.P90Reference <= 0 || got.Prediction.P99Reference <= 0 {
		t.Fatalf(
			"remaining-cycle forecast quantiles = p50:%d p90:%d p99:%d, want non-zero for the upcoming cycle",
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
			now:  time.Date(2026, 7, 12, 9, 0, 0, 0, clock.DefaultLocation),
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
		{
			desc: "sao_ke_day_locks_to_current_period",
			now:  time.Date(2026, 8, 9, 9, 0, 0, 0, clock.DefaultLocation),
			want: "2026-08",
		},
	}

	for _, c := range cases {
		if got := forecastForMonth(c.now); got != c.want {
			t.Errorf("%s: forecastForMonth = %q, want %q", c.desc, got, c.want)
		}
	}
}

func TestWalletDemandForecastBeforePeriodUsesUpcomingCycle(t *testing.T) {
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

// TestWalletDemandForecastEveOfPeriodStart covers the production
// "batch-at-payday" pattern: historical periods have no early-cycle volume and
// all demand lands around cycle day 10. The remaining-cycle recommendation must
// still include that later demand before the upcoming period opens.
func TestWalletDemandForecastEveOfPeriodStart(t *testing.T) {
	now := time.Date(2026, 7, 19, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			// Batch-at-payday: no early-cycle volume, all demand on day 10.
			{ForMonth: "2026-06", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
			{ForMonth: "2026-05", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 120_000_000},
			{ForMonth: "2026-04", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 90_000_000},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 53_554_680}}

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
	if got.Prediction.HorizonCycleDay != 2 {
		t.Fatalf("HorizonCycleDay = %d, want 2 (July 19 + 2 lead days = July 21 = cycle day 2)", got.Prediction.HorizonCycleDay)
	}
	// The recommendation must not collapse to 0 just because the early cycle
	// days are empty in the historical pattern.
	if got.Prediction.RecommendedBalance <= 0 {
		t.Fatalf("RecommendedBalance = %d, want > 0 for remaining-cycle demand", got.Prediction.RecommendedBalance)
	}
	// Sanity: the recommendation should be in the ballpark of the historical
	// grand totals (~100M), since cycleDist spans the full period.
	if got.Prediction.RecommendedBalance < 50_000_000 {
		t.Fatalf("RecommendedBalance = %d, want ≥ 50M (close to historical grand totals)", got.Prediction.RecommendedBalance)
	}
	// Wallet had 53.5M; with a ~100M recommendation the shortfall must fire.
	if got.Prediction.Shortfall <= 0 {
		t.Fatalf("Shortfall = %d, want > 0 (insufficient funds vs the upcoming cycle)", got.Prediction.Shortfall)
	}
}

func TestWalletDemandForecastRemainingCycleIgnoresLeadWindow(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	rows := []domain.CohortRow{
		{ForMonth: "2026-06", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
		{ForMonth: "2026-05", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 120_000_000},
		{ForMonth: "2026-04", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 90_000_000},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 0}}

	forecastWithLeadDays := func(leadDays int) *walletdomain.WalletDemandForecastResponse {
		t.Helper()
		repo := &walletDemandForecastRequestRepoStub{rows: rows}
		svc := NewWalletDemandForecastService(repo, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
			ServiceLevel:  0.95,
			NSim:          1000,
			HistoryMonths: 3,
			LeadDays:      leadDays,
		})
		got, err := svc.GetDemandForecast(context.Background())
		if err != nil {
			t.Fatalf("GetDemandForecast with LeadDays=%d returned error: %v", leadDays, err)
		}
		return got
	}

	oneDay := forecastWithLeadDays(1)
	tenDays := forecastWithLeadDays(10)
	if oneDay.Prediction.HorizonCycleDay == tenDays.Prediction.HorizonCycleDay {
		t.Fatalf("HorizonCycleDay should still reflect LeadDays metadata")
	}
	if oneDay.Prediction.RecommendedBalance != tenDays.Prediction.RecommendedBalance {
		t.Fatalf(
			"RecommendedBalance changed with LeadDays: one-day=%d ten-day=%d",
			oneDay.Prediction.RecommendedBalance,
			tenDays.Prediction.RecommendedBalance,
		)
	}
}

// TestWalletDemandForecastPeriodStartUsesRemainingCycle reproduces the wallet
// card showing 0 on July 20 even though the flexible-pay period has just opened
// and historical demand arrives later in the cycle.
func TestWalletDemandForecastPeriodStartUsesRemainingCycle(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-06", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
			{ForMonth: "2026-05", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 120_000_000},
			{ForMonth: "2026-04", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 90_000_000},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 52_358_330}}

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

	if got.CurrentCycleDay != 1 {
		t.Fatalf("CurrentCycleDay = %d, want 1 on period start", got.CurrentCycleDay)
	}
	if got.Prediction.RecommendedBalance < 50_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want >= 50M for remaining-cycle demand",
			got.Prediction.RecommendedBalance,
		)
	}
	if got.Prediction.Shortfall <= 0 {
		t.Fatalf("Shortfall = %d, want > 0 against remaining-cycle demand", got.Prediction.Shortfall)
	}
}

func TestWalletDemandForecastDoesNotDoubleCountCurrentPayableDemand(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 1, Status: string(domain.AdvancePaymentStatusPending), TotalAmount: 100_000_000},
			{ForMonth: "2026-06", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
			{ForMonth: "2026-05", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
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
	if got.Prediction.RecommendedBalance != 100_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want exactly 100M of known payable demand",
			got.Prediction.RecommendedBalance,
		)
	}
}

func TestWalletDemandForecastOnlyCountsPayableStatusesWithoutHistory(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 1, Status: string(domain.AdvancePaymentStatusPending), TotalAmount: 40_000_000},
			{ForMonth: "2026-07", CycleDay: 1, Status: string(domain.AdvancePaymentStatusApproved), TotalAmount: 30_000_000},
			{ForMonth: "2026-07", CycleDay: 1, Status: string(domain.AdvancePaymentStatusCancelled), TotalAmount: 20_000_000},
			{ForMonth: "2026-07", CycleDay: 1, Status: string(domain.AdvancePaymentStatusFailed), TotalAmount: 10_000_000},
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
	if got.Prediction.Method != "no-history" {
		t.Fatalf("Method = %q, want no-history", got.Prediction.Method)
	}
	if got.Prediction.RecommendedBalance != 70_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want 70M from PENDING + APPROVED only",
			got.Prediction.RecommendedBalance,
		)
	}
}

func TestWalletDemandForecastIncludesResidualDemandOnCutoffDay(t *testing.T) {
	// Aug 8 is the request cutoff day: the last cycle day of period 2026-07
	// (July has 31 days -> maxCycleDay 20), so same-day residual demand from
	// history must still be included on top of what today already requested.
	now := time.Date(2026, 8, 8, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 20, Status: string(domain.AdvancePaymentStatusPending), TotalAmount: 40_000_000},
			{ForMonth: "2026-05", CycleDay: 20, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
			{ForMonth: "2026-03", CycleDay: 20, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 0}}
	svc := NewWalletDemandForecastService(repo, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 5,
		LeadDays:      2,
	})

	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}
	if got.CurrentCycleDay != got.MaxCycleDay {
		t.Fatalf("CurrentCycleDay = %d, want cutoff day %d", got.CurrentCycleDay, got.MaxCycleDay)
	}
	if got.Prediction.RecommendedBalance != 100_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want 40M known + 60M residual cutoff demand",
			got.Prediction.RecommendedBalance,
		)
	}
}

func TestWalletDemandForecastIncludesPriorPeriodPayableCarryover(t *testing.T) {
	// Inside the locked gap (request cutoff day 9 < day < period start day 20)
	// the forecast targets the upcoming period while the prior period's
	// payable must still carry over into the recommendation.
	now := time.Date(2026, 7, 12, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-06", CycleDay: 19, Status: string(domain.AdvancePaymentStatusPending), TotalAmount: 40_000_000},
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
		t.Fatalf("CurrentForMonth = %q, want upcoming 2026-07 period", got.CurrentForMonth)
	}
	if got.Prediction.RecommendedBalance < 40_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want at least 40M prior-period payable carryover",
			got.Prediction.RecommendedBalance,
		)
	}
}

func TestWalletDemandForecastIncludesPayableOlderThanForecastHistory(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2025-12", CycleDay: 1, Status: string(domain.AdvancePaymentStatusApproved), TotalAmount: 40_000_000},
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
	if containsMonth(repo.months, "2025-12") {
		t.Fatalf("forecast history unexpectedly included old payable period: %v", repo.months)
	}
	if got.Prediction.RecommendedBalance != 40_000_000 {
		t.Fatalf(
			"RecommendedBalance = %d, want 40M all-time payable obligation",
			got.Prediction.RecommendedBalance,
		)
	}
}

func TestWalletDemandForecastConditionsActiveCycleOnObservedPaceAndUploadedCapacity(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 45_323_040},
			{ForMonth: "2026-06", CycleDay: 2, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 8_624_000},
			{ForMonth: "2026-06", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 82_242_060},
			{ForMonth: "2026-06", CycleDay: 17, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 15_323_280},
			{ForMonth: "2026-05", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 75_056_870},
			{ForMonth: "2026-05", CycleDay: 18, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 22_957_760},
			{ForMonth: "2026-04", CycleDay: 5, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 1_960_000},
			{ForMonth: "2026-04", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 8_820_000},
			{ForMonth: "2026-04", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 39_631_200},
			{ForMonth: "2026-04", CycleDay: 19, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 3_234_000},
		},
		cycleState: domain.AdvancePaymentCycleForecastState{
			MaxAdvanceAmount:  342_900_000,
			UsedRequestAmount: 45_920_000,
		},
	}
	walletSvc := &walletDemandForecastWalletStub{
		balance: &walletdomain.WalletBalance{Available: 53_516_340},
	}
	svc := NewWalletDemandForecastService(repo, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 4,
		LeadDays:      2,
	})

	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	maxRemainingCapacity := int64(342_900_000 - 45_920_000)
	if got.Prediction.RecommendedBalance <= 138_104_313 {
		t.Fatalf(
			"RecommendedBalance = %d, want current pace to raise it above the stale historical-only 138104313",
			got.Prediction.RecommendedBalance,
		)
	}
	if got.Prediction.RecommendedBalance > maxRemainingCapacity {
		t.Fatalf(
			"RecommendedBalance = %d, want <= remaining uploaded capacity %d",
			got.Prediction.RecommendedBalance,
			maxRemainingCapacity,
		)
	}
	if got.Prediction.P50Reference > maxRemainingCapacity ||
		got.Prediction.P90Reference > maxRemainingCapacity ||
		got.Prediction.P99Reference > maxRemainingCapacity {
		t.Fatalf(
			"reference ladder p50=%d p90=%d p99=%d exceeds remaining uploaded capacity %d",
			got.Prediction.P50Reference,
			got.Prediction.P90Reference,
			got.Prediction.P99Reference,
			maxRemainingCapacity,
		)
	}
}

func TestWalletDemandForecastDoesNotTreatNoRequestsAsZeroFutureDemand(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-06", CycleDay: 2, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 8_624_000},
			{ForMonth: "2026-06", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 82_242_060},
			{ForMonth: "2026-05", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 75_056_870},
			{ForMonth: "2026-05", CycleDay: 18, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 22_957_760},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{
		balance: &walletdomain.WalletBalance{Available: 0},
	}
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
	if got.Prediction.RecommendedBalance <= 0 {
		t.Fatalf(
			"RecommendedBalance = %d, want historical remaining demand before the first current request",
			got.Prediction.RecommendedBalance,
		)
	}
}
