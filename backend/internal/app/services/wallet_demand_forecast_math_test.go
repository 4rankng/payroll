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
