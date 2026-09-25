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

// prodLGD2026Cohort is the real advance-payment cohort from production on
// 2026-09-25 (cycle day 6 of 2026-09): July spent ~197M, August ~904M — a
// single 4.6x spike from the LGD project onboarding — and the current cycle
// had paid ~209M by day 6.
func prodLGD2026Cohort() []domain.CohortRow {
	return []domain.CohortRow{
		{ForMonth: "2026-07", CycleDay: 6, Status: "COMPLETED", TotalAmount: 68890780},
		{ForMonth: "2026-07", CycleDay: 7, Status: "COMPLETED", TotalAmount: 26239325},
		{ForMonth: "2026-07", CycleDay: 8, Status: "COMPLETED", TotalAmount: 2940000},
		{ForMonth: "2026-07", CycleDay: 10, Status: "COMPLETED", TotalAmount: 3136000},
		{ForMonth: "2026-07", CycleDay: 11, Status: "COMPLETED", TotalAmount: 2386300},
		{ForMonth: "2026-07", CycleDay: 12, Status: "COMPLETED", TotalAmount: 8953700},
		{ForMonth: "2026-07", CycleDay: 17, Status: "COMPLETED", TotalAmount: 14074620},
		{ForMonth: "2026-07", CycleDay: 18, Status: "COMPLETED", TotalAmount: 43965740},
		{ForMonth: "2026-07", CycleDay: 19, Status: "COMPLETED", TotalAmount: 13581700},
		{ForMonth: "2026-07", CycleDay: 20, Status: "COMPLETED", TotalAmount: 10948840},
		{ForMonth: "2026-07", CycleDay: 21, Status: "COMPLETED", TotalAmount: 2704800},
		{ForMonth: "2026-08", CycleDay: -8, Status: "COMPLETED", TotalAmount: 690900},
		{ForMonth: "2026-08", CycleDay: -5, Status: "COMPLETED", TotalAmount: 430000},
		{ForMonth: "2026-08", CycleDay: 0, Status: "COMPLETED", TotalAmount: 602700},
		{ForMonth: "2026-08", CycleDay: 1, Status: "COMPLETED", TotalAmount: 345000},
		{ForMonth: "2026-08", CycleDay: 3, Status: "COMPLETED", TotalAmount: 340000},
		{ForMonth: "2026-08", CycleDay: 4, Status: "COMPLETED", TotalAmount: 3521600},
		{ForMonth: "2026-08", CycleDay: 7, Status: "CANCELLED", TotalAmount: 2940000},
		{ForMonth: "2026-08", CycleDay: 7, Status: "COMPLETED", TotalAmount: 460195120},
		{ForMonth: "2026-08", CycleDay: 8, Status: "COMPLETED", TotalAmount: 35387980},
		{ForMonth: "2026-08", CycleDay: 9, Status: "COMPLETED", TotalAmount: 25812920},
		{ForMonth: "2026-08", CycleDay: 10, Status: "COMPLETED", TotalAmount: 29393050},
		{ForMonth: "2026-08", CycleDay: 11, Status: "COMPLETED", TotalAmount: 8393200},
		{ForMonth: "2026-08", CycleDay: 12, Status: "COMPLETED", TotalAmount: 3861200},
		{ForMonth: "2026-08", CycleDay: 13, Status: "COMPLETED", TotalAmount: 21430600},
		{ForMonth: "2026-08", CycleDay: 14, Status: "COMPLETED", TotalAmount: 1813000},
		{ForMonth: "2026-08", CycleDay: 15, Status: "COMPLETED", TotalAmount: 2097200},
		{ForMonth: "2026-08", CycleDay: 16, Status: "COMPLETED", TotalAmount: 16606520},
		{ForMonth: "2026-08", CycleDay: 17, Status: "COMPLETED", TotalAmount: 131096910},
		{ForMonth: "2026-08", CycleDay: 18, Status: "COMPLETED", TotalAmount: 105004200},
		{ForMonth: "2026-08", CycleDay: 19, Status: "COMPLETED", TotalAmount: 34280540},
		{ForMonth: "2026-08", CycleDay: 20, Status: "COMPLETED", TotalAmount: 17228040},
		{ForMonth: "2026-08", CycleDay: 21, Status: "COMPLETED", TotalAmount: 5256800},
		{ForMonth: "2026-09", CycleDay: -8, Status: "COMPLETED", TotalAmount: 1166200},
		{ForMonth: "2026-09", CycleDay: -6, Status: "COMPLETED", TotalAmount: 646800},
		{ForMonth: "2026-09", CycleDay: 4, Status: "COMPLETED", TotalAmount: 980000},
		{ForMonth: "2026-09", CycleDay: 6, Status: "COMPLETED", TotalAmount: 199860920},
		{ForMonth: "2026-09", CycleDay: 6, Status: "FAILED", TotalAmount: 8053920},
	}
}

func lgd2026Historical() []cohortSeries {
	rows := prodLGD2026Cohort()
	byMonth := make(map[string]cohortSeries)
	var historical []cohortSeries
	for i, fm := range []string{"2026-09", "2026-08", "2026-07"} {
		ps := pivotCohort(rows, fm, i == 0)
		byMonth[fm] = ps
		if !ps.isCurrent {
			historical = append(historical, ps)
		}
	}
	return historical
}

// TestForecastProjectedTotal_NearZeroPaceRatioIsDegenerate reproduces the
// production wallet-forecast blow-up: on 2026-09-25 the cohort-median pace
// projection inverted August's 0.47%-by-day-6 ratio into a 44B projection,
// the median of [~600M, ~44B] came out at 22.6B, and pace conditioning then
// pinned the whole recommendation to the wallet ceiling — the "Cần nạp thêm
// 1.5 tỷ" card. A period with less than paceScaleFloor of its final volume
// observed carries no pace signal and must not participate.
func TestForecastProjectedTotal_NearZeroPaceRatioIsDegenerate(t *testing.T) {
	historical := lgd2026Historical()

	// July had 35% of its final volume by day 6; August only 0.47%.
	predicted, method, _, _ := forecastProjectedTotal(208894840, historical, 6)

	if method != "avg-final" {
		t.Fatalf("expected degenerate cohort-median to fall back to avg-final, got method=%s predicted=%d", method, predicted)
	}
	july := historical[1].grandTotal
	august := historical[0].grandTotal
	low, high := july, august
	if low > high {
		low, high = high, low
	}
	if predicted < low || predicted > high {
		t.Fatalf("avg-final projection %d escaped the historical range [%d, %d]", predicted, low, high)
	}
}

// TestGrowthRatesConsistent_SingleRateIsNotATrend pins the second production
// defect: the July→August single 4.6x jump (one project onboarding) was
// treated as a consistent trend because a one-element window passed the
// dispersion check vacuously, doubling the whole demand distribution.
func TestGrowthRatesConsistent_SingleRateIsNotATrend(t *testing.T) {
	if growthRatesConsistent([]float64{4.57}) {
		t.Fatal("a single growth rate must not count as consistent directional growth")
	}
	if !growthRatesConsistent([]float64{1.2, 1.3, 1.25}) {
		t.Fatal("a genuine shared-direction window should stay consistent")
	}
}

// TestForecastDistribution_SingleSpikeBasisKeepsRawLevel pins the end-to-end
// effect on the distribution: with a two-period basis whose growth is one
// spike, the returned distribution must keep the raw historical level
// (growthFactor 1) instead of extrapolating the spike.
func TestForecastDistribution_SingleSpikeBasisKeepsRawLevel(t *testing.T) {
	historical := lgd2026Historical()

	dist := forecastRemainingCycleDistribution(historical, 6, maxCycleDay("2026-09"), historical[0].dailyAmount[6], 500, forecastSeed("2026-09", maxCycleDay("2026-09")), 1.0)

	if dist.growthFactor != 1.0 {
		t.Fatalf("single-spike basis must not be growth-adjusted, got factor %.2f", dist.growthFactor)
	}
}

// TestWalletDemandForecast_QuotaBasedRuleOnProdCohort replays the full
// 2026-09-25 production input through the service and pins the deterministic
// quota-based rule: the bảng công for 2026-09 IS uploaded (advance_payments
// rows exist), so the recommendation is exactly quota − completed
// disbursements, with the shortfall measured against the wallet balance.
func TestWalletDemandForecast_QuotaBasedRuleOnProdCohort(t *testing.T) {
	const quotaCeiling = uint64(2036704600)
	const walletAvailable = int64(174141975)

	// Completed 2026-09 rows in the fixture: 1.166.620 + 646.800 + 980.000 +
	// 199.860.920 = 202.653.920 (the FAILED 8.053.920 row is excluded).
	const completedDisbursed = int64(202653920)

	now := time.Date(2026, 9, 25, 13, 45, 0, 0, clock.DefaultLocation)
	repo := &walletDemandForecastRequestRepoStub{
		rows: prodLGD2026Cohort(),
		cycleState: domain.AdvancePaymentCycleForecastState{
			MaxAdvanceAmount:  quotaCeiling,
			UsedRequestAmount: 205385000,
		},
	}
	walletSvc := &walletDemandForecastWalletStub{balance: &walletdomain.WalletBalance{Available: walletAvailable}}

	svc := NewWalletDemandForecastService(repo, &walletDemandForecastAdvPayRepoStub{rows: repo.rows}, walletSvc, clock.NewFake(now), config.WalletForecastConfig{
		ServiceLevel:  0.95,
		NSim:          5000,
		HistoryMonths: 3,
		LeadDays:      2,
	})

	got, err := svc.GetDemandForecast(context.Background())
	if err != nil {
		t.Fatalf("GetDemandForecast: %v", err)
	}

	pred := got.Prediction
	if pred.Method != "quota-based" {
		t.Fatalf("Method = %q, want quota-based (bảng công uploaded for 2026-09)", pred.Method)
	}
	wantRecommended := int64(quotaCeiling) - completedDisbursed
	if pred.RecommendedBalance != wantRecommended {
		t.Fatalf("RecommendedBalance = %d, want quota − disbursed = %d", pred.RecommendedBalance, wantRecommended)
	}
	wantShortfall := wantRecommended - walletAvailable
	if pred.Shortfall != wantShortfall {
		t.Fatalf("Shortfall = %d, want %d", pred.Shortfall, wantShortfall)
	}
}
