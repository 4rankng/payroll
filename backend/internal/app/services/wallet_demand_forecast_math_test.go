package services

import (
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestMaxCycleDay(t *testing.T) {
	cases := map[string]int{
		"2026-01": (31 - 20 + 1) + 8, // January -> 20
		"2026-02": (28 - 20 + 1) + 8, // Feb non-leap -> 17
		"2024-02": (29 - 20 + 1) + 8, // Feb leap 2024 -> 18
		"2026-04": (30 - 20 + 1) + 8, // April -> 19
		"2026-06": (30 - 20 + 1) + 8, // June -> 19
		"2026-07": (31 - 20 + 1) + 8, // July -> 20
	}
	for fm, want := range cases {
		if got := maxCycleDay(fm); got != want {
			t.Errorf("maxCycleDay(%s) = %d, want %d", fm, got, want)
		}
	}
}

func TestCycleDayFor(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}
	// Period "2026-06" runs June 20 -> July 8 (request cutoff day 8).
	// June has 30 days -> maxCycleDay 19.
	cases := []struct {
		desc string
		t    time.Time
		want int
	}{
		{"june20_start", time.Date(2026, 6, 20, 12, 0, 0, 0, loc), 1},
		{"june30", time.Date(2026, 6, 30, 23, 0, 0, 0, loc), 11},
		{"july1", time.Date(2026, 7, 1, 0, 0, 0, 0, loc), 12},
		{"july8_cutoff", time.Date(2026, 7, 8, 23, 0, 0, 0, loc), 19},
		{"july9_sao_ke_day", time.Date(2026, 7, 9, 23, 0, 0, 0, loc), 0},
		{"july10", time.Date(2026, 7, 10, 0, 0, 0, 0, loc), 0},
		{"june19_before_start", time.Date(2026, 6, 19, 23, 0, 0, 0, loc), 0},
	}
	for _, c := range cases {
		if got := cycleDayFor(c.t, "2026-06"); got != c.want {
			t.Errorf("%s: cycleDayFor = %d, want %d", c.desc, got, c.want)
		}
	}
}

func TestCycleDayForHandlesYearBoundary(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}

	cases := []struct {
		desc string
		t    time.Time
		want int
	}{
		{"dec20_start", time.Date(2026, 12, 20, 12, 0, 0, 0, loc), 1},
		{"jan1_tail", time.Date(2027, 1, 1, 12, 0, 0, 0, loc), 13},
		{"jan8_cutoff", time.Date(2027, 1, 8, 12, 0, 0, 0, loc), 20},
		{"jan9_sao_ke_day", time.Date(2027, 1, 9, 12, 0, 0, 0, loc), 0},
		{"jan10_closed", time.Date(2027, 1, 10, 12, 0, 0, 0, loc), 0},
	}
	for _, c := range cases {
		if got := cycleDayFor(c.t, "2026-12"); got != c.want {
			t.Errorf("%s: cycleDayFor = %d, want %d", c.desc, got, c.want)
		}
	}
}

func TestForecastHorizonCycleDayLeadWindow(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}

	cases := []struct {
		desc     string
		now      time.Time
		forMonth string
		leadDays int
		want     int
	}{
		{
			desc:     "closed_gap_before_lead_window",
			now:      time.Date(2026, 7, 9, 12, 0, 0, 0, loc),
			forMonth: "2026-07",
			leadDays: 2,
			want:     0,
		},
		{
			desc:     "lead_window_reaches_next_period_start",
			now:      time.Date(2026, 7, 18, 12, 0, 0, 0, loc),
			forMonth: "2026-07",
			leadDays: 2,
			want:     1,
		},
		{
			desc:     "lead_window_crosses_cutoff",
			now:      time.Date(2026, 8, 7, 12, 0, 0, 0, loc),
			forMonth: "2026-07",
			leadDays: 2,
			want:     maxCycleDay("2026-07"),
		},
	}
	for _, c := range cases {
		if got := forecastHorizonCycleDay(c.now, c.forMonth, c.leadDays); got != c.want {
			t.Errorf("%s: forecastHorizonCycleDay = %d, want %d", c.desc, got, c.want)
		}
	}
}

func TestPivotCohortCumulative(t *testing.T) {
	rows := []domain.CohortRow{
		{ForMonth: "2026-06", CycleDay: 1, Status: "PENDING", TotalAmount: 100},
		{ForMonth: "2026-06", CycleDay: 1, Status: "COMPLETED", TotalAmount: 50},
		{ForMonth: "2026-06", CycleDay: 3, Status: "COMPLETED", TotalAmount: 30},
		{ForMonth: "2026-06", CycleDay: 99, Status: "PENDING", TotalAmount: 9999}, // out-of-window, dropped
		{ForMonth: "2026-05", CycleDay: 1, Status: "PENDING", TotalAmount: 7},     // other period, dropped
	}
	s := pivotCohort(rows, "2026-06", true)
	if s.grandTotal != 180 {
		t.Errorf("grandTotal = %d, want 180", s.grandTotal)
	}
	if s.completedTotal != 80 {
		t.Errorf("completedTotal = %d, want 80", s.completedTotal)
	}
	if s.cumulativeAt(1) != 150 {
		t.Errorf("cumulativeAt(1) = %d, want 150", s.cumulativeAt(1))
	}
	if s.cumulativeAt(2) != 150 {
		t.Errorf("cumulativeAt(2) = %d, want 150 (no row on day 2)", s.cumulativeAt(2))
	}
	if s.cumulativeAt(3) != 180 {
		t.Errorf("cumulativeAt(3) = %d, want 180", s.cumulativeAt(3))
	}
	if s.cumulativeAt(99) != 180 {
		t.Errorf("cumulativeAt(99) = %d, want 180 (clamped to maxCycleDay)", s.cumulativeAt(99))
	}
}

func TestForecastProjectedTotal(t *testing.T) {
	// Two historical periods, both 50% complete by cycle day 10.
	hist := []cohortSeries{
		{maxCycleDay: 20, grandTotal: 200, cumulative: map[int]int64{10: 100}},
		{maxCycleDay: 20, grandTotal: 400, cumulative: map[int]int64{10: 200}},
	}

	// (1) cohort-median: 2 valid ratios, todayCycleDay=10 → high, basis 2, projected 300.
	projected, method, basis, conf := forecastProjectedTotal(150, hist, 10)
	if method != "cohort-median" || basis != 2 || conf != "high" || projected != 300 {
		t.Errorf("cohort-median: projected=%d method=%s basis=%d conf=%s, want 300/cohort-median/2/high",
			projected, method, basis, conf)
	}

	// (2) avg-final with 2 finals, early period (todayCycleDay<5) → medium, basis 2, projected 300.
	projected, method, basis, conf = forecastProjectedTotal(150, hist, 3)
	if method != "avg-final" || basis != 2 || conf != "medium" || projected != 300 {
		t.Errorf("avg-final/2: projected=%d method=%s basis=%d conf=%s, want 300/avg-final/2/medium",
			projected, method, basis, conf)
	}

	// (3) single basis → avg-final, low, projected = the one final (200).
	single := []cohortSeries{{maxCycleDay: 20, grandTotal: 200, cumulative: map[int]int64{10: 100}}}
	projected, method, basis, conf = forecastProjectedTotal(150, single, 10)
	if method != "avg-final" || basis != 1 || conf != "low" || projected != 200 {
		t.Errorf("single: projected=%d method=%s basis=%d conf=%s, want 200/avg-final/1/low",
			projected, method, basis, conf)
	}

	// (4) no history → echo actualSoFar.
	projected, method, basis, conf = forecastProjectedTotal(150, nil, 10)
	if method != "no-history" || basis != 0 || conf != "low" || projected != 150 {
		t.Errorf("no-history: projected=%d method=%s basis=%d conf=%s, want 150/no-history/0/low",
			projected, method, basis, conf)
	}
}

func TestCompletionRate(t *testing.T) {
	if got := completionRate(nil); got != 0 {
		t.Errorf("nil completionRate = %f, want 0", got)
	}
	allCompleted := []cohortSeries{{completedTotal: 100, grandTotal: 100}}
	if got := completionRate(allCompleted); got != 1 {
		t.Errorf("all-completed = %f, want 1", got)
	}
	mixed := []cohortSeries{
		{completedTotal: 60, grandTotal: 100},
		{completedTotal: 40, grandTotal: 200}, // Σcompleted 100 / Σall 300 = 1/3
	}
	if got := completionRate(mixed); got != 1.0/3.0 {
		t.Errorf("mixed = %f, want %f", got, 1.0/3.0)
	}
}

// --- Newsvendor / MC engine tests ----------------------------------------

// mkHist builds a historical cohortSeries whose remaining net demand at `day`
// equals `remaining` (grandTotal − cumulativeAt(day)). maxCycleDay is 20.
func mkHist(remaining, grandTotal int64, day int) cohortSeries {
	return cohortSeries{
		maxCycleDay: 20,
		grandTotal:  grandTotal,
		cumulative:  map[int]int64{day: grandTotal - remaining},
	}
}

func almostEq(a, b, tol float64) bool {
	if a-b > tol || b-a > tol {
		return false
	}
	return true
}

func TestQuantileOfSorted(t *testing.T) {
	// R-7 interpolation on [1..11]: median (p=0.5) → index 5 → value 6.
	eleven := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	if v, c := quantileOfSorted(eleven, 0.5); !almostEq(v, 6, 1e-9) || c {
		t.Errorf("p50 of [1..11] = %v (clamped=%v), want 6 / false", v, c)
	}
	if v, c := quantileOfSorted(eleven, 0.0); v != 1 || c {
		t.Errorf("p0 = %v, want 1", v)
	}
	if v, c := quantileOfSorted(eleven, 1.0); v != 11 || c {
		t.Errorf("p100 = %v, want 11", v)
	}
	// Small-n honesty: 2 points, p>0.5 → max, clamped.
	two := []float64{10, 20}
	if v, c := quantileOfSorted(two, 0.95); v != 20 || !c {
		t.Errorf("p95 of [10,20] = %v (clamped=%v), want 20 / true", v, c)
	}
	// Small-n but p<=0.5 still interpolates.
	if v, c := quantileOfSorted(two, 0.5); !almostEq(v, 15, 1e-9) || c {
		t.Errorf("p50 of [10,20] = %v (clamped=%v), want 15 / false", v, c)
	}
	if v, c := quantileOfSorted(nil, 0.95); v != 0 || !c {
		t.Errorf("empty → (0, true), got (%v, %v)", v, c)
	}
	one := []float64{42}
	if v, c := quantileOfSorted(one, 0.95); v != 42 || c {
		t.Errorf("single → (42, false), got (%v, %v)", v, c)
	}
}

func TestServiceLevelConfig_EffectiveQuantile(t *testing.T) {
	cases := []struct {
		sl   serviceLevelConfig
		want float64
	}{
		{serviceLevelConfig{Quantile: 0.95}, 0.95},
		{serviceLevelConfig{CostUnder: 95, CostOver: 5}, 0.95},
		{serviceLevelConfig{}, 0.95}, // default
		{serviceLevelConfig{Quantile: 0.5}, 0.5},
		{serviceLevelConfig{Quantile: 1.5}, 0.999},                          // clamped
		{serviceLevelConfig{Quantile: 0.9, CostUnder: 1, CostOver: 1}, 0.9}, // explicit wins
	}
	for _, c := range cases {
		if got := c.sl.effectiveQuantile(); !almostEq(got, c.want, 1e-9) {
			t.Errorf("effectiveQuantile(%+v) = %v, want %v", c.sl, got, c.want)
		}
	}
}

func TestFitGammaMoM(t *testing.T) {
	// [90,100,110]: mean 100, population var 200/3 ≈ 66.67 → shape 150, scale 0.6667.
	shape, scale := fitGammaMoM([]float64{90, 100, 110})
	if !almostEq(shape, 150, 0.1) || !almostEq(scale, 2.0/3.0, 1e-4) {
		t.Errorf("fitGammaMoM([90,100,110]) = (shape=%v, scale=%v), want (~150, ~0.667)", shape, scale)
	}
	// Degenerate inputs return (0,0).
	if s, sc := fitGammaMoM([]float64{100, 100, 100}); s != 0 || sc != 0 {
		t.Errorf("zero-variance → (0,0), got (%v, %v)", s, sc)
	}
	if s, sc := fitGammaMoM(nil); s != 0 || sc != 0 {
		t.Errorf("empty → (0,0), got (%v, %v)", s, sc)
	}
}

func TestConfidenceLabel(t *testing.T) {
	cases := []struct {
		dist demandDistribution
		want string
	}{
		{demandDistribution{method: "no-history"}, "low"},
		{demandDistribution{method: "gamma-fit", basisPeriods: 1}, "low"},
		{demandDistribution{method: "gamma-fit", basisPeriods: 2}, "medium"},
		{demandDistribution{method: "gamma-fit", basisPeriods: 2, divergent: true}, "low"},
		{demandDistribution{method: "monte-carlo", basisPeriods: 3}, "high"},
		{demandDistribution{method: "monte-carlo", basisPeriods: 3, divergent: true}, "low"},
		{demandDistribution{method: "monte-carlo", basisPeriods: 6}, "high"},
		// Strong directional growth (5.7×) → gamma treats growth as variance → medium.
		{demandDistribution{method: "monte-carlo", basisPeriods: 6, trendRatio: 5.7}, "medium"},
		// Just at the cutoff → medium.
		{demandDistribution{method: "monte-carlo", basisPeriods: 3, trendRatio: 3.0}, "medium"},
		// Below the cutoff → high (stationary basis, gamma fit trustworthy).
		{demandDistribution{method: "monte-carlo", basisPeriods: 4, trendRatio: 2.5}, "high"},
		// Growth + divergence → low (worst case).
		{demandDistribution{method: "monte-carlo", basisPeriods: 6, trendRatio: 5.7, divergent: true}, "low"},
	}
	for _, c := range cases {
		if got := confidenceLabel(c.dist); got != c.want {
			t.Errorf("confidenceLabel(basis=%d divergent=%v trendRatio=%.1f method=%s) = %q, want %q",
				c.dist.basisPeriods, c.dist.divergent, c.dist.trendRatio, c.dist.method, got, c.want)
		}
	}
}

// TestTrendRatio_RealKy2Data reproduces the production Ky-2 cohort (Jan–Jun 2026)
// and confirms the trendRatio reflects the 5.7× growth that the gamma fit would
// otherwise treat as random variance.
func TestTrendRatio_RealKy2Data(t *testing.T) {
	// Historical Ky-2 grand totals (VND) from the production database, where
	// cumulativeAt(4) = 0 for every cycle (batch-at-payday approval pattern), so
	// remaining = grandTotal for each.
	historical := []cohortSeries{
		{forMonth: "2026-01", maxCycleDay: 10, grandTotal: 80_350_500, cumulative: cumSteps(80_350_500, 10)},
		{forMonth: "2026-02", maxCycleDay: 10, grandTotal: 130_036_050, cumulative: cumSteps(130_036_050, 10)},
		{forMonth: "2026-03", maxCycleDay: 10, grandTotal: 132_502_900, cumulative: cumSteps(132_502_900, 10)},
		{forMonth: "2026-04", maxCycleDay: 10, grandTotal: 158_614_931, cumulative: cumSteps(158_614_931, 10)},
		{forMonth: "2026-05", maxCycleDay: 10, grandTotal: 240_188_932, cumulative: cumSteps(240_188_932, 10)},
		{forMonth: "2026-06", maxCycleDay: 10, grandTotal: 457_082_437, cumulative: cumSteps(457_082_437, 10)},
	}
	// fromCycleDay=4 (today 11/07), throughCycleDay=10 (pay day 17/07).
	dist := forecastDemandDistributionBetween(historical, 4, 10, 5000, 42, 1.0)

	if dist.trendRatio < 5.0 {
		t.Errorf("trendRatio = %.2f, want ≥ 5.0 (457M/80M = 5.69× growth)", dist.trendRatio)
	}
	if got := confidenceLabel(dist); got == "high" {
		t.Errorf("confidence = %q, want NOT high (strong trend should downgrade)", got)
	}
	if dist.basisPeriods != 6 {
		t.Errorf("basisPeriods = %d, want 6", dist.basisPeriods)
	}
}

// cumSteps builds a cumulative map where ALL volume lands on the last cycle-day
// (batch-at-payday pattern): cumulative[1..maxDay-1] = 0, cumulative[maxDay] = total.
func cumSteps(total int64, maxDay int) map[int]int64 {
	c := make(map[int]int64, maxDay)
	for d := 1; d < maxDay; d++ {
		c[d] = 0
	}
	c[maxDay] = total
	return c
}

func TestNewsvendorRecommendation(t *testing.T) {
	// samples [0..99]: p95 ≈ 94.05, p99 ≈ 98.01.
	samples := make([]float64, 100)
	for i := range samples {
		samples[i] = float64(i)
	}
	dist := demandDistribution{samples: samples, p99: 98.01}

	// p* = 0.95, no safety stock → recommended ≈ 94.
	rec, cov := newsvendorRecommendation(dist, serviceLevelConfig{Quantile: 0.95})
	if cov != 0.95 || rec < 93 || rec > 95 {
		t.Errorf("p95 no-safety: recommended=%d coverage=%v, want ~94 / 0.95", rec, cov)
	}

	// Safety stock 0.5 of (p99 − p*) ≈ 0.5×(98−94) = +2 → ~96.
	recSS, _ := newsvendorRecommendation(dist, serviceLevelConfig{Quantile: 0.95, UncertaintyFactor: 0.5})
	if recSS < rec || recSS-rec > 4 {
		t.Errorf("safety stock: recommended=%d, want in (%d, %d]", recSS, rec, rec+4)
	}

	// Divergent: empirical max floors the recommendation above the quantile.
	div := demandDistribution{samples: samples, p99: 98.01, empiricalMax: 200, divergent: true}
	recDiv, _ := newsvendorRecommendation(div, serviceLevelConfig{Quantile: 0.95})
	if recDiv != 200 {
		t.Errorf("divergent floor: recommended=%d, want 200 (empiricalMax)", recDiv)
	}

	// No-history → 0 recommendation (orchestrator applies its own floor).
	recNH, _ := newsvendorRecommendation(demandDistribution{method: "no-history"}, serviceLevelConfig{Quantile: 0.95})
	if recNH != 0 {
		t.Errorf("no-history recommended=%d, want 0", recNH)
	}
}

func TestForecastDemandDistribution_NoHistory(t *testing.T) {
	// No historical periods at all.
	dist := forecastDemandDistribution(nil, 10, 5000, 1, 1)
	if dist.method != "no-history" || len(dist.samples) != 1 || dist.samples[0] != 0 {
		t.Errorf("nil history: method=%s samples=%v, want no-history/[0]", dist.method, dist.samples)
	}
	// Historical periods all have zero grand total → still no usable history.
	zero := []cohortSeries{{maxCycleDay: 20, grandTotal: 0}}
	dist = forecastDemandDistribution(zero, 10, 5000, 1, 1)
	if dist.method != "no-history" {
		t.Errorf("all-zero history: method=%s, want no-history", dist.method)
	}
}

func TestForecastDemandDistribution_MethodAndBasis(t *testing.T) {
	day := 10
	// 1 usable period → gamma-fit, basis 1.
	dist := forecastDemandDistribution([]cohortSeries{mkHist(100, 200, day)}, day, 2000, 7, 1)
	if dist.method != "gamma-fit" || dist.basisPeriods != 1 {
		t.Errorf("n=1: method=%s basis=%d, want gamma-fit/1", dist.method, dist.basisPeriods)
	}
	// 2 usable periods → gamma-fit, basis 2.
	dist = forecastDemandDistribution([]cohortSeries{mkHist(100, 200, day), mkHist(120, 200, day)}, day, 2000, 7, 1)
	if dist.method != "gamma-fit" || dist.basisPeriods != 2 {
		t.Errorf("n=2: method=%s basis=%d, want gamma-fit/2", dist.method, dist.basisPeriods)
	}
	// 3 usable periods → monte-carlo, basis 3.
	dist = forecastDemandDistribution(
		[]cohortSeries{mkHist(90, 200, day), mkHist(100, 200, day), mkHist(110, 200, day)},
		day, 2000, 7, 1,
	)
	if dist.method != "monte-carlo" || dist.basisPeriods != 3 {
		t.Errorf("n=3: method=%s basis=%d, want monte-carlo/3", dist.method, dist.basisPeriods)
	}
}

func TestForecastDemandDistribution_Ordering(t *testing.T) {
	dist := forecastDemandDistribution(
		[]cohortSeries{mkHist(80, 200, 10), mkHist(100, 200, 10), mkHist(140, 200, 10)},
		10, 3000, 42, 1,
	)
	if !(dist.p50 <= dist.p90 && dist.p90 <= dist.p95 && dist.p95 <= dist.p99) {
		t.Errorf("quantile ordering violated: p50=%v p90=%v p95=%v p99=%v",
			dist.p50, dist.p90, dist.p95, dist.p99)
	}
	if dist.p50 < 0 {
		t.Errorf("p50 negative: %v", dist.p50)
	}
}

func TestForecastDemandDistribution_PaidFracScaling(t *testing.T) {
	hist := []cohortSeries{mkHist(90, 200, 10), mkHist(100, 200, 10), mkHist(110, 200, 10)}
	full := forecastDemandDistribution(hist, 10, 3000, 42, 1)
	half := forecastDemandDistribution(hist, 10, 3000, 42, 0.5)
	if !almostEq(half.p50, full.p50/2, full.p50*0.05) {
		t.Errorf("paidFrac=0.5 should halve p50: full=%v half=%v", full.p50, half.p50)
	}
}

func TestForecastDemandDistribution_Determinism(t *testing.T) {
	hist := []cohortSeries{mkHist(90, 200, 10), mkHist(100, 200, 10), mkHist(110, 200, 10)}
	a := forecastDemandDistribution(hist, 10, 3000, 12345, 1)
	b := forecastDemandDistribution(hist, 10, 3000, 12345, 1)
	if a.p50 != b.p50 || a.p95 != b.p95 || a.p99 != b.p99 {
		t.Errorf("seeded non-deterministic: a=(%v,%v,%v) b=(%v,%v,%v)",
			a.p50, a.p95, a.p99, b.p50, b.p95, b.p99)
	}
}

func TestForecastDemandDistribution_Degenerate(t *testing.T) {
	// Identical remaining across 3 periods → zero variance → point mass at mean.
	hist := []cohortSeries{mkHist(100, 200, 10), mkHist(100, 200, 10), mkHist(100, 200, 10)}
	dist := forecastDemandDistribution(hist, 10, 2000, 9, 1)
	if !(almostEq(dist.p50, 100, 1e-9) && almostEq(dist.p95, 100, 1e-9) && almostEq(dist.p99, 100, 1e-9)) {
		t.Errorf("degenerate point mass: p50=%v p95=%v p99=%v, want all 100",
			dist.p50, dist.p95, dist.p99)
	}
}

// --- Growth EWMA tests ------------------------------------------------------

func TestGrowthEWMA_KnownRates(t *testing.T) {
	// 100 → 200 → 400: growth rates 2.0, 2.0. EWMA(0.5) = 0.5*2.0 + 0.5*2.0 = 2.0.
	got := growthEWMA([]int64{100, 200, 400}, 0.5)
	if !almostEq(got, 2.0, 1e-9) {
		t.Errorf("growthEWMA([100,200,400], 0.5) = %v, want 2.0", got)
	}
	// 100 → 150 → 300: rates 1.5, 2.0. EWMA(0.5) = 0.5*2.0 + 0.5*1.5 = 1.75.
	got = growthEWMA([]int64{100, 150, 300}, 0.5)
	if !almostEq(got, 1.75, 1e-9) {
		t.Errorf("growthEWMA([100,150,300], 0.5) = %v, want 1.75", got)
	}
}

func TestGrowthEWMA_EdgeCases(t *testing.T) {
	// n < 2 → 1.0 (no growth).
	if got := growthEWMA([]int64{100}, 0.5); !almostEq(got, 1.0, 1e-9) {
		t.Errorf("growthEWMA([100]) = %v, want 1.0", got)
	}
	if got := growthEWMA([]int64{}, 0.5); !almostEq(got, 1.0, 1e-9) {
		t.Errorf("growthEWMA([]) = %v, want 1.0", got)
	}
	// All-zero basis → 1.0 (no division by zero).
	if got := growthEWMA([]int64{0, 0, 0}, 0.5); !almostEq(got, 1.0, 1e-9) {
		t.Errorf("growthEWMA([0,0,0]) = %v, want 1.0", got)
	}
	// alpha=0 → uses default (0.5).
	got := growthEWMA([]int64{100, 200}, 0)
	if !almostEq(got, 2.0, 1e-9) {
		t.Errorf("growthEWMA([100,200], 0→default) = %v, want 2.0", got)
	}
}

func TestGrowthEWMA_LastThreeWindow(t *testing.T) {
	// 6 points: growth rates 2, 2, 2, 2, 2. Last 3 = [2, 2, 2]. EWMA = 2.0.
	got := growthEWMA([]int64{10, 20, 40, 80, 160, 320}, 0.5)
	if !almostEq(got, 2.0, 1e-9) {
		t.Errorf("growthEWMA(6 doubling points) = %v, want 2.0", got)
	}
}

// TestGrowthAdjustment_RealKy2Data verifies the growth-adjustment branch fires
// on the real production Ky-2 cohort and lifts the P50 above the flat-gamma
// baseline of ~170M toward the trend trajectory.
func TestGrowthAdjustment_RealKy2Data(t *testing.T) {
	// Real Ky-2 grand totals (VND) from the production DB, Jan–Jun 2026.
	// Batch-at-payday pattern: cumulativeAt(4) = 0, so rem = grandTotal.
	historical := []cohortSeries{
		{forMonth: "2026-01", maxCycleDay: 10, grandTotal: 80_350_500, cumulative: cumSteps(80_350_500, 10)},
		{forMonth: "2026-02", maxCycleDay: 10, grandTotal: 130_036_050, cumulative: cumSteps(130_036_050, 10)},
		{forMonth: "2026-03", maxCycleDay: 10, grandTotal: 132_502_900, cumulative: cumSteps(132_502_900, 10)},
		{forMonth: "2026-04", maxCycleDay: 10, grandTotal: 158_614_931, cumulative: cumSteps(158_614_931, 10)},
		{forMonth: "2026-05", maxCycleDay: 10, grandTotal: 240_188_932, cumulative: cumSteps(240_188_932, 10)},
		{forMonth: "2026-06", maxCycleDay: 10, grandTotal: 457_082_437, cumulative: cumSteps(457_082_437, 10)},
	}
	// fromCycleDay=4 (today 11/07), throughCycleDay=10 (pay day 17/07).
	dist := forecastDemandDistributionBetween(historical, 4, 10, 5000, 42, 1.0)

	if dist.method != "growth-adjusted" {
		t.Errorf("method = %q, want growth-adjusted (trendRatio=%.1f >= %v)",
			dist.method, dist.trendRatio, trendRatioCutoff)
	}
	if dist.growthFactor <= 1.0 {
		t.Errorf("growthFactor = %.3f, want > 1.0 (cohort is growing)", dist.growthFactor)
	}
	// Flat-gamma P50 ≈ 170M. Growth-adjusted should be meaningfully higher.
	if dist.p50 < 250_000_000 {
		t.Errorf("growth-adjusted p50 = %.0f, want >= 250M (flat gamma was ~170M)", dist.p50)
	}
	t.Logf("growth-adjusted: factor=%.3f p50=%.0f p95=%.0f (flat gamma was p50~170M p95~443M)",
		dist.growthFactor, dist.p50, dist.p95)
}

// TestGrowthAdjustment_StationaryUnchanged verifies the growth branch does NOT
// fire on stationary data — output is identical to the pre-change path.
func TestGrowthAdjustment_StationaryUnchanged(t *testing.T) {
	// 3 cycles all totaling 1000 — trendRatio = 1.0 (< cutoff).
	hist := []cohortSeries{
		{forMonth: "2026-04", maxCycleDay: 10, grandTotal: 1000, cumulative: cumSteps(1000, 10)},
		{forMonth: "2026-05", maxCycleDay: 10, grandTotal: 1000, cumulative: cumSteps(1000, 10)},
		{forMonth: "2026-06", maxCycleDay: 10, grandTotal: 1000, cumulative: cumSteps(1000, 10)},
	}
	dist := forecastDemandDistributionBetween(hist, 3, 10, 5000, 7, 1.0)
	if dist.method == "growth-adjusted" {
		t.Errorf("method = %q on stationary data, want NOT growth-adjusted", dist.method)
	}
	if !almostEq(dist.growthFactor, 1.0, 1e-9) {
		t.Errorf("growthFactor = %v on stationary data, want 1.0", dist.growthFactor)
	}
}

// TestChronologicalGrandTotals verifies extraction + sort order.
func TestChronologicalGrandTotals(t *testing.T) {
	// Deliberately unsorted input (map-iteration order simulation).
	hist := []cohortSeries{
		{forMonth: "2026-06", grandTotal: 457},
		{forMonth: "2026-01", grandTotal: 80},
		{forMonth: "2026-03", grandTotal: 132},
	}
	got := chronologicalGrandTotals(hist)
	want := []int64{80, 132, 457}
	if len(got) != len(want) {
		t.Fatalf("got %d totals, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chronologicalGrandTotals[%d] = %d, want %d (chronological)", i, got[i], want[i])
		}
	}
}

// The pace scale cap keeps an outlier cycle's ratio from stretching the
// historical tail into an over-compounded safety margin, while calm-cycle
// down-scaling stays exact.
func TestConditionDistributionOnCurrentPaceScaleCap(t *testing.T) {
	dist := demandDistribution{
		samples: []float64{10_000_000, 20_000_000, 30_000_000, 40_000_000},
		p50:     25_000_000,
		p90:     37_000_000,
		p95:     39_000_000,
		p99:     40_000_000,
		method:  "monte-carlo",
	}

	// targetP50 = 8x the raw p50 — the cap must hold the upscale at 2x.
	got := conditionDistributionOnCurrentPace(dist, 200_000_000, -1, 2.0)
	if got.p50 != 50_000_000 {
		t.Fatalf("p50 = %f, want 50000000 (scale capped at 2x)", got.p50)
	}
	if got.p95 > 39_000_000*2.0*1.001 {
		t.Fatalf("p95 = %f, want ≤ 2x raw p95 (78000000)", got.p95)
	}

	// Down-scaling (calm cycle) is never capped. Fresh dist: the function
	// mutates the samples slice in place, so reuse would mix states.
	fresh := demandDistribution{
		samples: []float64{10_000_000, 20_000_000, 30_000_000, 40_000_000},
		p50:     25_000_000,
		method:  "monte-carlo",
	}
	gotDown := conditionDistributionOnCurrentPace(fresh, 12_500_000, -1, 2.0)
	if gotDown.p50 != 12_500_000 {
		t.Fatalf("p50 = %f, want 12500000 (exact down-scale)", gotDown.p50)
	}
}
