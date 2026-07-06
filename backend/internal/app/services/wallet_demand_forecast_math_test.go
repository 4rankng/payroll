package services

import (
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestMaxCycleDay(t *testing.T) {
	cases := map[string]int{
		"2026-01": (31 - 20 + 1) + 9, // January → 21
		"2026-02": (28 - 20 + 1) + 9, // Feb non-leap → 19
		"2024-02": (29 - 20 + 1) + 9, // Feb leap 2024 → 20
		"2026-04": (30 - 20 + 1) + 9, // April → 20
		"2026-06": (30 - 20 + 1) + 9, // June → 20
		"2026-07": (31 - 20 + 1) + 9, // July → 21
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
	// Period "2026-06" runs June 20 → July 9. June has 30 days → maxCycleDay 20.
	cases := []struct {
		desc string
		t    time.Time
		want int
	}{
		{"june20_start", time.Date(2026, 6, 20, 12, 0, 0, 0, loc), 1},
		{"june30", time.Date(2026, 6, 30, 23, 0, 0, 0, loc), 11},
		{"july1", time.Date(2026, 7, 1, 0, 0, 0, 0, loc), 12},
		{"july9_cutoff", time.Date(2026, 7, 9, 23, 0, 0, 0, loc), 20},
		{"july10_after_cutoff", time.Date(2026, 7, 10, 0, 0, 0, 0, loc), 0},
		{"june19_before_start", time.Date(2026, 6, 19, 23, 0, 0, 0, loc), 0},
	}
	for _, c := range cases {
		if got := cycleDayFor(c.t, "2026-06"); got != c.want {
			t.Errorf("%s: cycleDayFor = %d, want %d", c.desc, got, c.want)
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
	}
	for _, c := range cases {
		if got := confidenceLabel(c.dist); got != c.want {
			t.Errorf("confidenceLabel(basis=%d divergent=%v method=%s) = %q, want %q",
				c.dist.basisPeriods, c.dist.divergent, c.dist.method, got, c.want)
		}
	}
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
