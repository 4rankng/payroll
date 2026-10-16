package utils

import (
	"testing"
	"time"
)

// The from-day is a day of the PREVIOUS month; a day missing there (Feb 29 in a
// non-leap year) rolls forward to the next existing day.
func TestCalculateDateRangeRollsMissingStartDayForward(t *testing.T) {
	// March 2026 cycle: from=29 -> Feb 29 2026 does not exist -> Mar 1 2026.
	atDate := time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)
	from, to := CalculateDateRange(atDate, 29, 28)

	if got := from.Format("2006-01-02"); got != "2026-03-01" {
		t.Fatalf("from = %s, want 2026-03-01", got)
	}
	if got := to.Format("2006-01-02"); got != "2026-03-28" {
		t.Fatalf("to = %s, want 2026-03-28", got)
	}
}

// In a leap year the 29th exists, so the start day is used as-is.
func TestCalculateDateRangeKeepsExistingStartDay(t *testing.T) {
	atDate := time.Date(2028, time.March, 20, 0, 0, 0, 0, time.UTC)
	from, _ := CalculateDateRange(atDate, 29, 28)
	if got := from.Format("2006-01-02"); got != "2028-02-29" {
		t.Fatalf("from = %s, want 2028-02-29", got)
	}
}

// A missing end day clamps to the month's last day instead of bleeding into the
// next cycle's days ("cuối tháng" semantics of the 0 sentinel).
func TestCalculateDateRangeClampsMissingEndDay(t *testing.T) {
	atDate := time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC)
	_, to := CalculateDateRange(atDate, 1, 31)
	if got := to.Format("2006-01-02"); got != "2026-04-30" {
		t.Fatalf("to = %s, want 2026-04-30", got)
	}
}

// 0 sentinels: from=0 starts at the 1st, to=0 ends at month end.
func TestCalculateDateRangeSentinels(t *testing.T) {
	atDate := time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC)
	from, to := CalculateDateRange(atDate, 0, 0)
	if got := from.Format("2006-01-02"); got != "2026-10-01" {
		t.Fatalf("from = %s, want 2026-10-01", got)
	}
	if got := to.Format("2006-01-02"); got != "2026-10-31" {
		t.Fatalf("to = %s, want 2026-10-31", got)
	}
}
