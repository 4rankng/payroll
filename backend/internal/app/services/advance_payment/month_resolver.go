package advance_payment

import (
	"time"

	"api-server/internal/pkg/clock"
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
