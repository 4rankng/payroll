package advance_payment

import (
	"time"

	"api-server/internal/pkg/clock"
)

// Period cycle constants (same values as clock package, kept for compat).
const (
	PeriodCycleStartDay = clock.PeriodCycleStartDay
	RequestCutoffDay    = clock.RequestCutoffDay
)

// GetCurrentMonth determines the current month for advance payment purposes.
// Period cycle: 20th of previous month to 19th of current month.
func GetCurrentMonth() string {
	return clock.AdvanceMonthFromTime(clock.Now())
}

// GetCurrentMonthFromTime determines the current month from a given time.
func GetCurrentMonthFromTime(t time.Time) string {
	return clock.AdvanceMonthFromTime(t)
}

// GetNextPeriodMonth returns the next salary period's for_month relative to now.
func GetNextPeriodMonth() string {
	return clock.NextAdvanceMonthFromTime(clock.Now())
}

// GetNextPeriodMonthFromTime returns the next period month for a given time.
func GetNextPeriodMonthFromTime(t time.Time) string {
	return clock.NextAdvanceMonthFromTime(t)
}

// IsBeforeCutoff returns true if the time is within the tail-end of the
// previous period's request window (days 1 through RequestCutoffDay inclusive).
func IsBeforeCutoff(t time.Time) bool {
	return clock.IsBeforeCutoff(t)
}

// IsAfterCutoff returns true if the time is past the cutoff day.
func IsAfterCutoff(t time.Time) bool {
	return clock.IsAfterCutoff(t)
}

// IsInLockedGap returns true for the inter-period gap (days 11–20): after the
// previous period's window has closed but before the new period has started.
func IsInLockedGap(t time.Time) bool {
	return clock.IsInLockedGap(t)
}

// IsInNewPeriod returns true when the date is on or after PeriodCycleStartDay
// (day 20), meaning the current calendar month's advance period has begun.
func IsInNewPeriod(t time.Time) bool {
	return clock.IsInNewPeriod(t)
}

// FormatMonthDisplay converts "2006-01" to "MM/YYYY" for display.
func FormatMonthDisplay(yyyyMM string) string {
	return clock.FormatMonthDisplay(yyyyMM)
}

// ParseMonth parses a month string in YYYY-MM format.
func ParseMonth(monthStr string) (time.Time, error) {
	return clock.ParseMonth(monthStr)
}

// GetDefaultDateRange returns the default date range for pending requests
// (yesterday + today).
func GetDefaultDateRange() (fromDate, toDate time.Time) {
	return clock.DefaultDateRange(clock.Global())
}
