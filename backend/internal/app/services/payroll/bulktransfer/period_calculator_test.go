package bulktransfer

import (
	"testing"
	"time"
)

// The from-day lives in the month BEFORE monthStart and rolls forward when that
// month lacks it (Feb 29 in a non-leap year -> Mar 1).
func TestComputeMonthlyStartRollsMissingDayForward(t *testing.T) {
	march := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	if got := computeMonthlyStart(march, 29); got.Format("2006-01-02") != "2026-03-01" {
		t.Fatalf("start(29) = %s, want 2026-03-01", got.Format("2006-01-02"))
	}

	// Leap year: the 29th exists and must be used as-is.
	feb2028Cycle := time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC)
	if got := computeMonthlyStart(feb2028Cycle, 29); got.Format("2006-01-02") != "2028-02-29" {
		t.Fatalf("leap start(29) = %s, want 2028-02-29", got.Format("2006-01-02"))
	}

	// A day further past the month keeps rolling (Feb 30 -> Mar 2).
	if got := computeMonthlyStart(march, 30); got.Format("2006-01-02") != "2026-03-02" {
		t.Fatalf("start(30) = %s, want 2026-03-02", got.Format("2006-01-02"))
	}

	// Ordinary days are untouched.
	if got := computeMonthlyStart(march, 26); got.Format("2006-01-02") != "2026-02-26" {
		t.Fatalf("start(26) = %s, want 2026-02-26", got.Format("2006-01-02"))
	}
}

// An end day past the month's length clamps to month end so the window never
// swallows days belonging to the next cycle.
func TestComputeMonthlyEndClampsMissingDay(t *testing.T) {
	april := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	if got := computeMonthlyEnd(april, 31); got.Format("2006-01-02") != "2026-04-30" {
		t.Fatalf("end(31) = %s, want 2026-04-30", got.Format("2006-01-02"))
	}
	if got := computeMonthlyEnd(april, 28); got.Format("2006-01-02") != "2026-04-28" {
		t.Fatalf("end(28) = %s, want 2026-04-28", got.Format("2006-01-02"))
	}
	// 0 sentinel = last day of the month.
	if got := computeMonthlyEnd(april, 0); got.Format("2006-01-02") != "2026-04-30" {
		t.Fatalf("end(0) = %s, want 2026-04-30", got.Format("2006-01-02"))
	}
}

// The reported case: cycle "29 previous month -> 28 this month" must tile the
// calendar around February without gaps or overlaps.
func TestMonthlyWindowTilesAroundFebruary(t *testing.T) {
	feb := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	march := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	prevStart, prevEnd := computeMonthlyStart(feb, 29), computeMonthlyEnd(feb, 28)
	if prevStart.Format("2006-01-02") != "2026-01-29" || prevEnd.Format("2006-01-02") != "2026-02-28" {
		t.Fatalf("feb window = %s..%s, want 2026-01-29..2026-02-28",
			prevStart.Format("2006-01-02"), prevEnd.Format("2006-01-02"))
	}

	start, end := computeMonthlyStart(march, 29), computeMonthlyEnd(march, 28)
	if start.Format("2006-01-02") != "2026-03-01" || end.Format("2006-01-02") != "2026-03-28" {
		t.Fatalf("march window = %s..%s, want 2026-03-01..2026-03-28",
			start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	if !start.After(prevEnd) {
		t.Fatalf("windows overlap: feb ends %s, march starts %s",
			prevEnd.Format("2006-01-02"), start.Format("2006-01-02"))
	}
}
