package services

import (
	"math"
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
// through day 9 (= clock.RequestCutoffDay) of M+1. Cycle day 1 = day 20 of M.

// cohortSeries is the per-period pivot of raw cohort rows: daily and cumulative
// request amounts (all statuses), plus completed/grand totals.
type cohortSeries struct {
	forMonth       string
	isCurrent      bool
	maxCycleDay    int
	dailyAmount    map[int]int64
	cumulative     map[int]int64
	completedTotal int64
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
// 30-day June → 20; 31-day July → 21; 28-day Feb → 19; 29-day Feb → 20.
func maxCycleDay(forMonth string) int {
	m, err := clock.ParseMonth(forMonth)
	if err != nil {
		return 0
	}
	return (daysInMonth(m.Year(), m.Month()) - clock.PeriodCycleStartDay + 1) + clock.RequestCutoffDay
}

// cycleDayFor returns the 1-indexed cycle day of t within the period that begins
// on day 20 of forMonth. Day 20 of forMonth → 1; day 9 of the next month →
// maxCycleDay. Returns 0 when t falls outside the [day-20, day-9] window.
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
	if t.Year() == year && t.Month() == month+1 && t.Day() >= 1 && t.Day() <= clock.RequestCutoffDay {
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
