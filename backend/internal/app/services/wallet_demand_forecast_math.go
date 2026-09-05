package services

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// This file holds the PURE, I/O-free math for the wallet demand-forecast:
// cycle-day derivation, cohort pivoting, the projected-total forecast, and the
// completion rate. Keeping it free of repository/wallet/clock-now dependencies
// makes the forecast logic trivially unit-testable.
//
// Period model: the advance-payment period for forMonth M runs from day 20 of M
// through day 8 (= clock.RequestCutoffDay) of M+1. Cycle day 1 = day 20 of M.

// cohortSeries is the per-period pivot of raw cohort rows: daily and cumulative
// net employee-requested amounts (request_amount - fee), plus completed/grand
// totals. Demand comes from advance_payment_requests only. Uploaded
// quota/max_adv_amount may cap an in-progress projection, but must never be
// treated as demand by itself.
type cohortSeries struct {
	forMonth       string
	isCurrent      bool
	maxCycleDay    int
	dailyAmount    map[int]int64
	cumulative     map[int]int64
	completedTotal int64
	payableTotal   int64
	grandTotal     int64
}

// cumulativeAt returns the cumulative-all request amount through day, clamped to
// [1, maxCycleDay]. Days with no rows contribute 0 to the running sum.
func (s cohortSeries) cumulativeAt(day int) int64 {
	if day < 1 {
		return 0
	}
	if day > s.maxCycleDay {
		day = s.maxCycleDay
	}
	return s.cumulative[day]
}

// daysInMonth returns the calendar day count for (year, month).
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// maxCycleDay returns the last cycle-day index for the period that begins on
// day 20 of forMonth: (daysInMonth(forMonth) - 20 + 1) + RequestCutoffDay.
// 30-day June → 20; 31-day July → 21; 28-day Feb → 18; 29-day Feb → 19.
func maxCycleDay(forMonth string) int {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return 0
	}
	return (daysInMonth(m.Year(), m.Month()) - clock.PeriodCycleStartDay + 1) + clock.RequestCutoffDay
}

// forecastHorizonCycleDay returns the cycle day covered by the configured lead
// window. It is retained as response metadata; the remaining-cycle wallet target
// is not limited by this horizon.
func forecastHorizonCycleDay(now time.Time, forMonth string, leadDays int) int {
	if leadDays < 0 {
		leadDays = 0
	}
	start, end, ok := periodWindow(forMonth)
	if !ok {
		return 0
	}
	today := dateOnly(now.In(clock.DefaultLocation))
	leadEnd := today.AddDate(0, 0, leadDays)
	if leadEnd.Before(start) || today.After(end) {
		return 0
	}
	if leadEnd.After(end) {
		return maxCycleDay(forMonth)
	}
	return cycleDayFor(leadEnd, forMonth)
}

func periodWindow(forMonth string) (time.Time, time.Time, bool) {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	start := time.Date(m.Year(), m.Month(), clock.PeriodCycleStartDay, 0, 0, 0, 0, clock.DefaultLocation)
	return start, start.AddDate(0, 0, maxCycleDay(forMonth)-1), true
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, clock.DefaultLocation)
}

// cycleDayFor returns the 1-indexed cycle day of t within the period that begins
// on day 20 of forMonth. Day 20 of forMonth -> 1; day 8 of the next month ->
// maxCycleDay. Returns 0 when t falls outside the [day-20, day-8] window.
func cycleDayFor(t time.Time, forMonth string) int {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return 0
	}
	year, month := m.Year(), m.Month()

	// Same calendar month as forMonth: days 20..end.
	if t.Year() == year && t.Month() == month && t.Day() >= clock.PeriodCycleStartDay {
		return t.Day() - clock.PeriodCycleStartDay + 1
	}
	// Next calendar month: days 1..RequestCutoffDay.
	nextMonth := time.Date(year, month, 1, 0, 0, 0, 0, clock.DefaultLocation).AddDate(0, 1, 0)
	if t.Year() == nextMonth.Year() && t.Month() == nextMonth.Month() && t.Day() >= 1 && t.Day() <= clock.RequestCutoffDay {
		return (daysInMonth(year, month) - clock.PeriodCycleStartDay + 1) + t.Day()
	}
	return 0
}

// pivotCohort builds the per-period series from raw cohort rows, dropping rows
// outside [1, maxCycleDay] (locked-gap stragglers / out-of-window noise) and
// rows tagged with a different forMonth.
func pivotCohort(rows []domain.CohortRow, forMonth string, isCurrent bool) cohortSeries {
	maxDay := maxCycleDay(forMonth)
	s := cohortSeries{
		forMonth:    forMonth,
		isCurrent:   isCurrent,
		maxCycleDay: maxDay,
		dailyAmount: make(map[int]int64),
	}
	for _, r := range rows {
		if r.ForMonth != forMonth {
			continue
		}
		if r.CycleDay < 1 || r.CycleDay > maxDay {
			continue
		}
		s.dailyAmount[r.CycleDay] += r.TotalAmount
		s.grandTotal += r.TotalAmount
		if r.Status == string(domain.AdvancePaymentStatusCompleted) {
			s.completedTotal += r.TotalAmount
		}
		if r.Status == string(domain.AdvancePaymentStatusPending) ||
			r.Status == string(domain.AdvancePaymentStatusApproved) {
			s.payableTotal += r.TotalAmount
		}
	}
	s.cumulative = make(map[int]int64, maxDay)
	var run int64
	for d := 1; d <= maxDay; d++ {
		run += s.dailyAmount[d]
		s.cumulative[d] = run
	}
	return s
}

// forecastProjectedTotal estimates the total request volume the current period
// will reach, based on how far along historical periods were at todayCycleDay.
//
//   - cohort-median: ≥2 historical periods with a valid (0,1] ratio at
//     todayCycleDay (and todayCycleDay ≥ 5) → median(actualSoFar / ratio_h).
//   - avg-final: otherwise, if any historical period has a final total → mean.
//   - no-history: no usable historical data → actualSoFar.
//
// Returns (projected, method, basisPeriods, confidence).
func forecastProjectedTotal(actualSoFar int64, historical []cohortSeries, todayCycleDay int) (projected int64, method string, basis int, confidence string) {
	var scales []float64
	for _, h := range historical {
		final := h.grandTotal
		if final <= 0 {
			continue
		}
		cumThrough := h.cumulativeAt(todayCycleDay)
		if cumThrough <= 0 {
			continue
		}
		r := float64(cumThrough) / float64(final)
		if r > 0 && r <= 1 {
			scales = append(scales, r)
		}
	}

	if todayCycleDay >= 5 && len(scales) >= 2 {
		projections := make([]float64, len(scales))
		for i, sc := range scales {
			projections[i] = float64(actualSoFar) / sc
		}
		sort.Float64s(projections)
		var median float64
		n := len(projections)
		if n%2 == 1 {
			median = projections[n/2]
		} else {
			median = (projections[n/2-1] + projections[n/2]) / 2
		}
		return int64(math.Round(median)), "cohort-median", len(scales), "high"
	}

	var sumFinal int64
	var nFinal int
	for _, h := range historical {
		if h.grandTotal > 0 {
			sumFinal += h.grandTotal
			nFinal++
		}
	}
	if nFinal > 0 {
		confidence := "medium"
		if nFinal < 2 {
			confidence = "low" // single basis period
		}
		return sumFinal / int64(nFinal), "avg-final", nFinal, confidence
	}
	return actualSoFar, "no-history", 0, "low"
}

// completionRate is Σ COMPLETED amount / Σ all amount over historical periods,
// clamped to [0,1]. Returns 0 when there is no historical volume.
func completionRate(historical []cohortSeries) float64 {
	var completed, all int64
	for _, h := range historical {
		completed += h.completedTotal
		all += h.grandTotal
	}
	if all <= 0 {
		return 0
	}
	r := float64(completed) / float64(all)
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// --- Newsvendor / tail-risk engine ---------------------------------------
//
// The goal is "no service disruption", i.e. the wallet must cover the rest of
// the cycle. A median (p50) target gives a 50% chance of falling short — the
// wrong target for a service-level objective. Instead we build a predictive
// DISTRIBUTION of remaining-cycle net cash-out and read the p*-quantile
// (newsvendor), p* = Cu/(Cu+Co). With Cu ≫ Co, p* is high (0.95 by default).

// trendRatioCutoff flags when the basis shows directional growth rather than
// random variance. A max/min ratio ≥ 3 across ≥3 periods means the latest
// cycles are 3× the earliest — the gamma fit spreads this growth into the
// distribution's variance, making its quantiles unreliable. Below the cutoff,
// the basis is treated as stationary and the gamma fit is trusted at face value.
const trendRatioCutoff = 3.0

// defaultPaceScaleCap bounds the pace-conditioning upscale factor when no
// explicit cap is configured (see conditionDistributionOnCurrentPace).
const defaultPaceScaleCap = 2.0

// demandDistribution is the predictive distribution of the current period's
// REMAINING net cash-out (cash still to leave the wallet from tomorrow through
// cycle end), conditional on demand observed through todayCycleDay. All fields
// are VND cash (already scaled by the historical completion fraction). It is
// deterministic given the input cohorts + rngSeed.
type demandDistribution struct {
	samples      []float64 // sorted ascending; remaining cash-out draws (≥0)
	p50          float64
	p90          float64
	p95          float64
	p99          float64
	gammaShape   float64 // MoM fit params (0 when degenerate)
	gammaScale   float64
	empiricalMax float64 // max historical remaining cash-out (observed tail)
	method       string  // "monte-carlo" | "gamma-fit" | "no-history" | "growth-adjusted"
	basisPeriods int     // # usable historical periods
	nSim         int
	divergent    bool    // gamma p95 under-represents the observed tail
	trendRatio   float64 // max/min of the basis (≥1); high values signal directional growth
	growthFactor float64 // EWMA growth multiplier applied (1.0 when not trending)
}

// serviceLevelConfig is the newsvendor cost / quantile knob. Precedence:
// Quantile > 0 wins; else CostUnder/(CostUnder+CostOver); else 0.95.
type serviceLevelConfig struct {
	Quantile          float64
	CostUnder         float64
	CostOver          float64
	UncertaintyFactor float64 // adds safety stock from the p99−p* slack
}

// effectiveQuantile resolves the newsvendor service level p*.
func (sl serviceLevelConfig) effectiveQuantile() float64 {
	if sl.Quantile > 0 {
		return clampF(sl.Quantile, 0.5, 0.999)
	}
	if sl.CostUnder > 0 || sl.CostOver > 0 {
		return clampF(sl.CostUnder/(sl.CostUnder+sl.CostOver), 0.5, 0.999)
	}
	return 0.95
}

// quantileOfSorted returns the p-th percentile (R-7 linear interpolation) of an
// ascending-sorted slice. HONEST SMALL-N: when len < 3 and p > 0.5 it returns
// the max element with clamped=true — never fabricate a tail by interpolating
// between one or two points. len==0 → (0, true); len==1 → (x, false).
func quantileOfSorted(sorted []float64, p float64) (value float64, clamped bool) {
	n := len(sorted)
	if n == 0 {
		return 0, true
	}
	if n == 1 {
		return sorted[0], false
	}
	if n < 3 && p > 0.5 {
		return sorted[n-1], true
	}
	if p <= 0 {
		return sorted[0], false
	}
	if p >= 1 {
		return sorted[n-1], false
	}
	h := float64(n-1) * p
	lo := int(math.Floor(h))
	if lo >= n-1 {
		return sorted[n-1], false
	}
	frac := h - float64(lo)
	return sorted[lo] + frac*(sorted[lo+1]-sorted[lo]), false
}

// fitGammaMoM fits a gamma distribution by method of moments. Returns (0,0) on
// a degenerate input (empty, non-positive mean, or zero variance); the caller
// then falls back to a point mass at the mean.
func fitGammaMoM(xs []float64) (shape, scale float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	mean := meanF(xs)
	if mean <= 0 {
		return 0, 0
	}
	var ssd float64
	for _, x := range xs {
		d := x - mean
		ssd += d * d
	}
	variance := ssd / float64(len(xs)) // population variance
	if variance <= 0 {
		return 0, 0
	}
	return mean * mean / variance, variance / mean
}

// gammaSample draws one sample from Gamma(shape, scale) via Marsaglia-Tsang
// (shape≥1) with the standard boost for shape<1. r must be a seeded source.
func gammaSample(r *rand.Rand, shape, scale float64) float64 {
	if shape <= 0 || scale <= 0 {
		return 0
	}
	if shape < 1 {
		g := gammaSample(r, shape+1, 1)
		u := r.Float64()
		for u <= 0 {
			u = r.Float64()
		}
		return g * math.Pow(u, 1/shape) * scale
	}
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)
	for {
		x := boxMuller(r)
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := r.Float64()
		if u < 1-0.0331*x*x*x*x {
			return d * v * scale
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v * scale
		}
	}
}

// boxMuller draws a standard normal from two uniform draws in (0,1).
func boxMuller(r *rand.Rand) float64 {
	u1 := nextUnit(r)
	u2 := nextUnit(r)
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// nextUnit returns a uniform float in (0,1), excluding the open endpoints so
// log/cos in boxMuller stay finite.
func nextUnit(r *rand.Rand) float64 {
	u := r.Float64()
	for u <= 1e-300 || u >= 1 {
		u = r.Float64()
	}
	return u
}

// forecastDemandDistribution projects the distribution of REMAINING net cash-out
// for the current period. It fits a gamma (method of moments) to each historical
// period's remaining net demand at todayCycleDay, then draws nSim samples scaled
// by paidFrac (the historical completion fraction) to express the result as cash
// that will actually leave the wallet. I/O-free; pin rngSeed for deterministic
// output.
func forecastDemandDistribution(
	historical []cohortSeries,
	todayCycleDay int,
	nSim int,
	rngSeed int64,
	paidFrac float64,
) demandDistribution {
	return forecastDemandDistributionBetween(historical, todayCycleDay, 1<<30, nSim, rngSeed, paidFrac)
}

// forecastDemandDistributionBetween projects the distribution of net cash-out
// between two cycle-day positions. fromCycleDay is the observed boundary and
// throughCycleDay is the forecast horizon. When the horizon is not ahead of the
// observed boundary, the future-demand distribution is zero.
func forecastDemandDistributionBetween(
	historical []cohortSeries,
	fromCycleDay int,
	throughCycleDay int,
	nSim int,
	rngSeed int64,
	paidFrac float64,
) demandDistribution {
	pf := clampF(paidFrac, 0, 1)

	// Net demand in the forecast window for each usable historical period.
	var rem []float64
	for _, h := range historical {
		if h.grandTotal <= 0 {
			continue
		}
		var r float64
		if throughCycleDay > fromCycleDay {
			endAmount := h.cumulativeAt(throughCycleDay)
			if throughCycleDay > h.maxCycleDay {
				endAmount = h.grandTotal
			}
			r = float64(endAmount - h.cumulativeAt(fromCycleDay))
		}
		if r < 0 {
			r = 0
		}
		rem = append(rem, r)
	}
	return forecastDemandDistributionFromRemaining(historical, rem, nSim, rngSeed, pf)
}

// forecastRemainingCycleDistribution keeps the unobserved part of the current
// cycle day in the forecast without counting demand already seen today twice.
// Future cycle days are always included in full.
func forecastRemainingCycleDistribution(
	historical []cohortSeries,
	todayCycleDay int,
	throughCycleDay int,
	observedToday int64,
	nSim int,
	rngSeed int64,
	paidFrac float64,
) demandDistribution {
	pf := clampF(paidFrac, 0, 1)
	var rem []float64
	for _, h := range historical {
		if h.grandTotal <= 0 {
			continue
		}

		futureAmount := int64(0)
		if throughCycleDay > todayCycleDay {
			endAmount := h.cumulativeAt(throughCycleDay)
			if throughCycleDay > h.maxCycleDay {
				endAmount = h.grandTotal
			}
			futureAmount = endAmount - h.cumulativeAt(todayCycleDay)
		}

		sameDayResidual := int64(0)
		if todayCycleDay >= 1 && todayCycleDay <= throughCycleDay {
			sameDayResidual = max(int64(0), h.dailyAmount[todayCycleDay]-observedToday)
		}
		rem = append(rem, float64(max(int64(0), futureAmount)+sameDayResidual))
	}
	return forecastDemandDistributionFromRemaining(historical, rem, nSim, rngSeed, pf)
}

// conditionDistributionOnCurrentPace re-centres the historical remaining-cycle
// distribution on the current cycle's observed pace. maxFuture is the remaining
// uploaded request capacity; a negative value means no authoritative ceiling is
// available. The historical shape and tail remain intact while the forecast
// level follows the current cycle.
//
// scaleCap bounds how far the current pace may stretch the historical
// distribution upward. The tail spread is historical evidence; multiplying the
// worst observed tail by a large pace ratio compounds two safety margins into
// an over-conservative recommendation (an outlier cycle easily produces a 5-8x
// ratio). Down-scaling (calm cycle) is never capped.
func conditionDistributionOnCurrentPace(
	dist demandDistribution,
	targetP50 int64,
	maxFuture int64,
	scaleCap float64,
) demandDistribution {
	if scaleCap <= 0 {
		scaleCap = defaultPaceScaleCap
	}
	if dist.method == "no-history" || len(dist.samples) == 0 {
		return dist
	}
	if targetP50 < 0 {
		targetP50 = 0
	}
	if maxFuture >= 0 && targetP50 > maxFuture {
		targetP50 = maxFuture
	}

	scale := 0.0
	shift := 0.0
	if dist.p50 > 0 {
		scale = float64(targetP50) / dist.p50
		if scale > scaleCap {
			scale = scaleCap
		}
	} else if targetP50 > 0 {
		scale = 1
		shift = float64(targetP50)
	}
	capValue := math.Inf(1)
	if maxFuture >= 0 {
		capValue = float64(maxFuture)
	}

	for i, sample := range dist.samples {
		adjusted := sample*scale + shift
		if adjusted > capValue {
			adjusted = capValue
		}
		dist.samples[i] = max(0, adjusted)
	}
	sort.Float64s(dist.samples)

	dist.p50, _ = quantileOfSorted(dist.samples, 0.50)
	dist.p90, _ = quantileOfSorted(dist.samples, 0.90)
	dist.p95, _ = quantileOfSorted(dist.samples, 0.95)
	dist.p99, _ = quantileOfSorted(dist.samples, 0.99)
	dist.empiricalMax = min(dist.empiricalMax*scale+shift, capValue)
	dist.divergent = dist.empiricalMax > 0 && dist.p95 < dist.empiricalMax*0.75
	return dist
}

func forecastDemandDistributionFromRemaining(
	historical []cohortSeries,
	rem []float64,
	nSim int,
	rngSeed int64,
	paidFrac float64,
) demandDistribution {
	pf := clampF(paidFrac, 0, 1)
	if len(rem) == 0 {
		return demandDistribution{samples: []float64{0}, method: "no-history"}
	}

	empiricalMaxDemand := 0.0
	empiricalMinDemand := math.MaxFloat64
	for _, r := range rem {
		if r > empiricalMaxDemand {
			empiricalMaxDemand = r
		}
		if r < empiricalMinDemand {
			empiricalMinDemand = r
		}
	}
	trendRatio := 1.0
	if empiricalMinDemand > 0 {
		trendRatio = empiricalMaxDemand / empiricalMinDemand
	}

	shape, scale := fitGammaMoM(rem)
	n := nSim
	if n < 1000 {
		n = 1000
	}
	rng := rand.New(rand.NewPCG(uint64(rngSeed), 0))
	samples := make([]float64, n)
	if shape <= 0 || scale <= 0 {
		// Degenerate (zero variance): point mass at the mean × paidFrac.
		v := meanF(rem) * pf
		for i := range samples {
			samples[i] = v
		}
	} else {
		for i := range samples {
			s := gammaSample(rng, shape, scale) * pf
			if s < 0 {
				s = 0
			}
			samples[i] = s
		}
	}
	sort.Float64s(samples)

	p50, _ := quantileOfSorted(samples, 0.50)
	p90, _ := quantileOfSorted(samples, 0.90)
	p95, _ := quantileOfSorted(samples, 0.95)
	p99, _ := quantileOfSorted(samples, 0.99)

	empiricalMaxCash := empiricalMaxDemand * pf
	// Divergence: the smooth gamma p95 must not fall far below the worst
	// observed period; if it does, the tail is under-represented.
	divergent := empiricalMaxCash > 0 && p95 < empiricalMaxCash*0.75

	method := "monte-carlo"
	if len(rem) < 3 {
		method = "gamma-fit"
	}

	// Growth adjustment: when the basis shows directional growth (trendRatio
	// ≥ cutoff), the gamma fit treats the growth as random variance — its
	// quantiles are centered on the historical mean, not the trend's
	// trajectory. Multiply the entire distribution by the EWMA growth factor
	// (derived from grand totals) to re-center on the trend level. The factor
	// is uniform, so the distribution SHAPE (skew, tail ratio) is preserved;
	// only the LEVEL shifts. empiricalMaxCash is scaled too, keeping the
	// divergent comparison self-consistent.
	growthFactor := 1.0
	if trendRatio >= trendRatioCutoff {
		growthFactor = growthEWMA(chronologicalGrandTotals(historical), growthEWMAlphaDefault)
		if growthFactor > 0 && growthFactor != 1.0 {
			for i := range samples {
				samples[i] *= growthFactor
			}
			p50 *= growthFactor
			p90 *= growthFactor
			p95 *= growthFactor
			p99 *= growthFactor
			empiricalMaxCash *= growthFactor
			method = "growth-adjusted"
		}
	}

	return demandDistribution{
		samples:      samples,
		p50:          p50,
		p90:          p90,
		p95:          p95,
		p99:          p99,
		gammaShape:   shape,
		gammaScale:   scale,
		empiricalMax: empiricalMaxCash,
		method:       method,
		basisPeriods: len(rem),
		nSim:         n,
		divergent:    divergent,
		trendRatio:   trendRatio,
		growthFactor: growthFactor,
	}
}

// growthEWMAlphaDefault is the EWMA smoothing factor for the growth-rate
// adjustment. 0.5 gives equal weight to the latest and historical growth —
// responsive to recent momentum without overreacting to a single cycle. Env-
// tunable via CashForecastConfig.GrowthEWMAlpha.
const growthEWMAlphaDefault = 0.5

// growthEWMA computes the exponentially-weighted moving average of the last 3
// month-over-month growth rates from grandTotals (which MUST be chronological).
// Returns a multiplier (e.g., 1.5 = 50% expected growth). Seeded to 1.0 (no
// growth) when fewer than 2 data points exist or when all prior values are 0.
//
// The EWMA seed is the first growth rate in the window; subsequent rates are
// smoothed with alpha. The "last 3" window prevents a single stale early-cycle
// growth rate from dominating on a long basis.
func growthEWMA(grandTotals []int64, alpha float64) float64 {
	if alpha <= 0 {
		alpha = growthEWMAlphaDefault
	}
	if len(grandTotals) < 2 {
		return 1.0
	}
	var rates []float64
	for i := 1; i < len(grandTotals); i++ {
		if grandTotals[i-1] > 0 {
			rates = append(rates, float64(grandTotals[i])/float64(grandTotals[i-1]))
		}
	}
	if len(rates) == 0 {
		return 1.0
	}
	// Take the last 3 growth rates (or fewer when the basis is thin).
	start := len(rates) - min(3, len(rates))
	window := rates[start:]
	ewma := window[0]
	for _, r := range window[1:] {
		ewma = alpha*r + (1-alpha)*ewma
	}
	return ewma
}

// chronologicalGrandTotals extracts the grandTotal from each cohortSeries,
// sorted chronologically by forMonth. The cohortSeries slice from
// buildTimesheetCohort is already sorted (since the 2026-07-11 fix), but this
// helper guarantees order regardless of the caller — the growth EWMA is
// index-sensitive and must not receive a randomized slice.
func chronologicalGrandTotals(historical []cohortSeries) []int64 {
	type entry struct {
		forMonth   string
		grandTotal int64
	}
	entries := make([]entry, 0, len(historical))
	for _, h := range historical {
		if h.grandTotal > 0 {
			entries = append(entries, entry{forMonth: h.forMonth, grandTotal: h.grandTotal})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].forMonth < entries[j].forMonth
	})
	out := make([]int64, len(entries))
	for i, e := range entries {
		out[i] = e.grandTotal
	}
	return out
}

// newsvendorRecommendation reads the distribution at the service-level quantile
// p* and adds optional safety stock proportional to the p99−p* slack. When the
// gamma fit diverges from the observed tail, the empirical max floors the
// recommendation so a smoothed-away outlier can never understate the buffer.
func newsvendorRecommendation(dist demandDistribution, sl serviceLevelConfig) (recommended int64, coverage float64) {
	q := sl.effectiveQuantile()
	if dist.method == "no-history" {
		return 0, q
	}
	val, _ := quantileOfSorted(dist.samples, q)
	rec := val + sl.UncertaintyFactor*math.Max(0, dist.p99-val)
	if dist.divergent {
		rec = max(rec, dist.empiricalMax)
	}
	return int64(math.Round(rec)), q
}

// confidenceLabel derives the advisory confidence from the basis size and
// whether the gamma fit diverged from the observed tail. When the basis shows
// strong directional growth (trend, not variance), confidence is downgraded —
// the gamma model treats the growth as random spread, so its quantiles under-
// represent the trend.
func confidenceLabel(dist demandDistribution) string {
	switch {
	case dist.method == "no-history", dist.basisPeriods <= 1:
		return "low"
	case dist.basisPeriods == 2:
		if dist.divergent {
			return "low"
		}
		return "medium"
	default:
		if dist.divergent {
			return "low"
		}
		// Strong monotonic trend → gamma treats growth as variance, so its
		// quantiles are unreliable regardless of basis count.
		if dist.trendRatio >= trendRatioCutoff {
			return "medium"
		}
		return "high"
	}
}

// forecastSeed derives a deterministic seed from the period and cycle day so the
// same request returns stable numbers within a day (no UI flicker on refetch).
func forecastSeed(forMonth string, todayCycleDay int) int64 {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return int64(todayCycleDay)
	}
	return int64(m.Year())*10000 + int64(m.Month())*100 + int64(todayCycleDay)
}

// meanF returns the arithmetic mean of xs (0 if empty).
func meanF(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// clampF clamps v to [lo, hi].
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
