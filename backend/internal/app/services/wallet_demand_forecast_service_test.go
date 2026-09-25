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

// GetCompletedDisbursedTotal derives the "already disbursed" figure from the
// stub rows, mirroring the production query's shape (SUM over COMPLETED
// requests for one for_month).
func (r *walletDemandForecastRequestRepoStub) GetCompletedDisbursedTotal(_ context.Context, forMonth string) (int64, error) {
	var total int64
	for _, row := range r.rows {
		if row.ForMonth == forMonth && row.Status == string(domain.AdvancePaymentStatusCompleted) {
			total += row.TotalAmount
		}
	}
	return total, nil
}

// walletDemandForecastAdvPayRepoStub derives the bảng-cong-uploaded switch
// from the cohort rows: a for_month with any row counts as uploaded.
type walletDemandForecastAdvPayRepoStub struct {
	domain.AdvancePaymentRepository
	rows []domain.CohortRow
}

func (r *walletDemandForecastAdvPayRepoStub) HasForMonth(_ context.Context, forMonth string) (bool, error) {
	for _, row := range r.rows {
		if row.ForMonth == forMonth {
			return true, nil
		}
	}
	return false, nil
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

	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
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
	// Bảng công not uploaded for 2026-07 → fall back to the previous kỳ's
	// total disbursement: 90M + 45M = 135M for 2026-06.
	if got.Prediction.RecommendedBalance != 135_000_000 {
		t.Fatalf("RecommendedBalance = %d, want 135000000 (previous kỳ disbursement)", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != got.Prediction.RecommendedBalance {
		t.Fatalf(
			"Shortfall = %d, want RecommendedBalance %d for an empty wallet",
			got.Prediction.Shortfall,
			got.Prediction.RecommendedBalance,
		)
	}
	if got.Prediction.Method != "prev-cycle" {
		t.Fatalf("Method = %q, want prev-cycle when the bảng công is not uploaded", got.Prediction.Method)
	}
	if len(repo.months) == 0 || repo.months[0] != "2026-07" {
		t.Fatalf("queried months = %v, want current effective period first", repo.months)
	}
}

func TestForecastForMonthTable(t *testing.T) {
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

	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
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

// TestWalletDemandForecastUploadedUsesQuotaMinusDisbursed pins rule 2: when
// the bảng công has been uploaded for the kỳ, the recommendation is the
// uploaded quota minus what has already been disbursed (completed payouts),
// regardless of how the remaining requests later play out.
func TestWalletDemandForecastUploadedUsesQuotaMinusDisbursed(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 30_000_000},
			{ForMonth: "2026-07", CycleDay: 6, Status: string(domain.AdvancePaymentStatusPending), TotalAmount: 12_000_000},
			{ForMonth: "2026-06", CycleDay: 2, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 8_624_000},
			{ForMonth: "2026-05", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 75_056_870},
		},
		cycleState: domain.AdvancePaymentCycleForecastState{
			MaxAdvanceAmount:  200_000_000,
			UsedRequestAmount: 42_000_000,
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 50_000_000}}

	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 3,
		LeadDays:      2,
	})
	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	// quota 200M − completed 30M = 170M; pending sits inside the quota and is
	// deliberately NOT subtracted again.
	if got.Prediction.RecommendedBalance != 170_000_000 {
		t.Fatalf("RecommendedBalance = %d, want 170000000 (quota − disbursed)", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != 120_000_000 {
		t.Fatalf("Shortfall = %d, want 120000000", got.Prediction.Shortfall)
	}
	if got.Prediction.Method != "quota-based" {
		t.Fatalf("Method = %q, want quota-based when the bảng công is uploaded", got.Prediction.Method)
	}
}

// TestWalletDemandForecastQuotaBelowDisbursedClampsToZero pins the clamp: a
// fully-disbursed (or over-disbursed) quota means nothing more can go out —
// the wallet needs no top-up and the surplus is reported.
func TestWalletDemandForecastQuotaBelowDisbursedClampsToZero(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-07", CycleDay: 6, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 220_000_000},
		},
		cycleState: domain.AdvancePaymentCycleForecastState{
			MaxAdvanceAmount:  200_000_000,
			UsedRequestAmount: 220_000_000,
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 80_000_000}}

	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          1000,
		HistoryMonths: 3,
		LeadDays:      2,
	})
	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	if got.Prediction.RecommendedBalance != 0 {
		t.Fatalf("RecommendedBalance = %d, want 0 (quota consumed)", got.Prediction.RecommendedBalance)
	}
	if got.Prediction.Shortfall != 0 {
		t.Fatalf("Shortfall = %d, want 0", got.Prediction.Shortfall)
	}
	if got.Prediction.Surplus != 80_000_000 {
		t.Fatalf("Surplus = %d, want 80000000", got.Prediction.Surplus)
	}
}

func TestWalletDemandForecastRemainingCycleIgnoresLeadWindow(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation)
	rows := []domain.CohortRow{
		{ForMonth: "2026-06", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 100_000_000},
		{ForMonth: "2026-05", CycleDay: 10, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 120_000_000},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: 0}}

	forecastWithLeadDays := func(leadDays int) *walletdomain.WalletDemandForecastResponse {
		t.Helper()
		repo := &walletDemandForecastRequestRepoStub{rows: rows}
		svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
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

// TestWalletDemandForecastPrevCycleIsDeterministic replays the 2026-09-17
// production situation: the locked gap (day 17) with 2026-09's bảng công not
// yet uploaded. The recommendation is exactly the previous kỳ's completed
// disbursement — no statistical tails.
func TestWalletDemandForecastPrevCycleIsDeterministic(t *testing.T) {
	now := time.Date(2026, 9, 17, 20, 0, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: []domain.CohortRow{
			{ForMonth: "2026-08", CycleDay: 7, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 439_531_960},
			{ForMonth: "2026-08", CycleDay: 18, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 205_502_990},
			{ForMonth: "2026-08", CycleDay: 19, Status: string(domain.AdvancePaymentStatusCompleted), TotalAmount: 236_721_850},
		},
	}
	walletSvc := &walletDemandForecastWalletStub{
		balance: &walletdomain.WalletBalance{Available: 48_767_269},
	}
	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          5000,
		HistoryMonths: 5,
		LeadDays:      2,
	})

	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast returned error: %v", err)
	}

	const prevDisbursed = int64(439_531_960 + 205_502_990 + 236_721_850) // 881.756.800
	if got.Prediction.Method != "prev-cycle" {
		t.Fatalf("Method = %q, want prev-cycle", got.Prediction.Method)
	}
	if got.Prediction.RecommendedBalance != prevDisbursed {
		t.Fatalf("RecommendedBalance = %d, want %d (previous kỳ disbursement)", got.Prediction.RecommendedBalance, prevDisbursed)
	}
	if got.Prediction.Shortfall != prevDisbursed-48_767_269 {
		t.Fatalf("Shortfall = %d, want %d", got.Prediction.Shortfall, prevDisbursed-48_767_269)
	}
}
