package clock

import "time"

// Advance payment period constants.
const (
	// PeriodCycleStartDay is the day of month when the advance payment period
	// rolls over to the next month. Days 1–20 belong to the previous month's
	// period; from the 21st onward the current month's period begins.
	PeriodCycleStartDay = 21

	// RequestCutoffDay is the last day of the month (in the following calendar
	// month) through which employees may still request an advance for the
	// previous period.  The request window runs from PeriodCycleStartDay of
	// the current month through RequestCutoffDay of the next month (inclusive).
	RequestCutoffDay = 10
)

// CurrentAdvanceMonth returns the advance payment period month for "now".
// Period cycle: 21st of previous month to 20th of current month.
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

// EffectiveAdvanceMonth returns the month that CreateRequest would use right now,
// accounting for the three-phase window rule:
//   - Days 1–10  (tail of previous period): request for the previous advance month
//   - Days 11–16 (locked gap): locked — no requests allowed
//   - Days 21+   (new period open): request for the current advance month,
//     provided admin has already uploaded bang lương for that month
func EffectiveAdvanceMonth(c Clock) string {
	return EffectiveAdvanceMonthFromTime(c.Now())
}

// EffectiveAdvanceMonthFromTime returns the effective month for a given time.
func EffectiveAdvanceMonthFromTime(t time.Time) string {
	if IsBeforeCutoff(t) {
		return AdvanceMonthFromTime(t)
	}
	return NextAdvanceMonthFromTime(t)
}

// IsBeforeCutoff returns true if the given time is within the tail-end of the
// previous period's request window (days 1 through RequestCutoffDay inclusive).
func IsBeforeCutoff(t time.Time) bool {
	return t.Day() <= RequestCutoffDay
}

// IsAfterCutoff returns true if the given time is past the cutoff day.
func IsAfterCutoff(t time.Time) bool {
	return t.Day() > RequestCutoffDay
}

// IsInLockedGap returns true when the date falls in the inter-period gap
// (after the cutoff window but before the new period starts: days 11–16).
func IsInLockedGap(t time.Time) bool {
	return t.Day() > RequestCutoffDay && t.Day() < PeriodCycleStartDay
}

// IsInNewPeriod returns true when the date is on or after the new period
// start day (days 21+), meaning the current calendar month's advance period
// has begun.
func IsInNewPeriod(t time.Time) bool {
	return t.Day() >= PeriodCycleStartDay
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
