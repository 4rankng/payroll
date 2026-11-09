package clock

import "time"

// Advance payment period constants.
const (
	// PeriodCycleStartDay is the day of month when the advance payment period
	// rolls over to the next month. Days 1–19 belong to the previous month's
	// period; from the 20th onward the current month's period begins.
	PeriodCycleStartDay = 20
)

// CurrentAdvanceMonth returns the advance payment period month for "now".
// Period cycle: 20th of previous month to 19th of current month.
func CurrentAdvanceMonth(c Clock) string {
	return AdvanceMonthFromTime(c.Now())
}

// AdvanceMonthFromTime determines the advance period month from a given time.
func AdvanceMonthFromTime(t time.Time) string {
	if t.Day() >= PeriodCycleStartDay {
		return t.Format("2006-01")
	}
	return t.AddDate(0, -1, 0).Format("2006-01")
}

// NextAdvanceMonth returns the next salary period month relative to now.
func NextAdvanceMonth(c Clock) string {
	return NextAdvanceMonthFromTime(c.Now())
}

// NextAdvanceMonthFromTime returns the next period month for a given time.
func NextAdvanceMonthFromTime(t time.Time) string {
	if t.Day() >= PeriodCycleStartDay {
		return t.AddDate(0, 1, 0).Format("2006-01")
	}
	return t.Format("2006-01")
}

// IsBeforeCutoff returns true when t is still inside the previous period's
// request tail: day of month strictly below the configured request stop day
// (the day new requests are refused). The stop day comes from the
// advance_request_stop_day setting (default 9), so the tail is days
// 1..stopDay-1.
func IsBeforeCutoff(t time.Time, requestStopDay int) bool {
	return t.Day() < requestStopDay
}

// IsInLockedGap returns true when t has passed the request stop day but the
// new period has not started yet (days stopDay..PeriodCycleStartDay-1).
func IsInLockedGap(t time.Time, requestStopDay int) bool {
	return t.Day() >= requestStopDay && t.Day() < PeriodCycleStartDay
}

// FormatMonthDisplay converts "2006-01" to "MM/YYYY" for display.
func FormatMonthDisplay(yyyyMM string) string {
	t, err := time.Parse("2006-01", yyyyMM)
	if err != nil {
		return yyyyMM
	}
	return t.Format("01/2006")
}

// ParseMonth parses a month string in YYYY-MM format.
func ParseMonth(monthStr string) (time.Time, error) {
	return time.Parse("2006-01", monthStr)
}

// DefaultDateRange returns the default date range for pending requests
// (yesterday + today) using the injected clock.
func DefaultDateRange(c Clock) (fromDate, toDate time.Time) {
	now := c.Now()
	fromDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	toDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	return
}
