package clock

import "time"

// Timesheet pay-cycle model.
//
// The timesheet payroll runs on a global, fixed 4-cycle monthly schedule — the
// same for every employee and project. Each Ky (cycle) has a 7-day work window
// followed by a pay date roughly three days later, during which approvals keep
// trickling in. Cash-readiness forecasting samples cumulative approved pay
// across this axis to project the next bulk transfer.
//
//	Ky 1: work days 1–7   → pay day 10
//	Ky 2: work days 8–14  → pay day 17
//	Ky 3: work days 15–21 → pay day 24
//	Ky 4: work days 22–28 → pay day 1 of the NEXT month
//
// cycle-day 1 is the Ky's work-start calendar day; the pay date lands on
// MaxCycleDay. This is a DISTINCT cycle from the advance-payment period
// (day 20 → day 9, modelled in advance_payment.go) — the two must never share
// constants.

const (
	// TimesheetKyCount is the number of pay cycles in a month.
	TimesheetKyCount = 4
)

// kyWorkStartDay[ky] is the calendar day each Ky's work window opens (index 1..4).
var kyWorkStartDay = [TimesheetKyCount + 1]int{0, 1, 8, 15, 22}

// kyPayDayInMonth[ky] is the pay day expressed in the work month's calendar.
// Ky 4 pays on day 1 of the next month, encoded as 0 and handled by PayDate.
var kyPayDayInMonth = [TimesheetKyCount + 1]int{0, 10, 17, 24, 0}

// KyFromWorkDay returns the Ky (1..4) whose work window contains the given
// day-of-month. Days 22–31 (and any out-of-range day) map to Ky 4.
func KyFromWorkDay(day int) int {
	switch {
	case day >= 1 && day <= 7:
		return 1
	case day >= 8 && day <= 14:
		return 2
	case day >= 15 && day <= 21:
		return 3
	default:
		return 4
	}
}

// WorkStartDay returns the calendar day the given Ky's work window opens.
// Returns 0 for an out-of-range Ky.
func WorkStartDay(ky int) int {
	if ky < 1 || ky > TimesheetKyCount {
		return 0
	}
	return kyWorkStartDay[ky]
}

// daysInMonth returns the calendar day count for (year, month).
func daysInMonth(year int, month time.Month) int {
	// Day 0 of the next month = last day of this month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// MaxCycleDay returns the last cycle-day index for the given Ky in (year, month):
// the count of days from the work-start through the pay date.
//
//	Ky 1–3 → 10 (e.g. work day 1 through pay day 10).
//	Ky 4   → (daysInMonth − 22 + 1) + 1: work day 22 through pay day 1 next month.
//	         July (31 days) → 11; February (28 days) → 8.
//
// Returns 0 for an out-of-range Ky.
func MaxCycleDay(ky int, year int, month time.Month) int {
	if ky < 1 || ky > TimesheetKyCount {
		return 0
	}
	if ky == 4 {
		return (daysInMonth(year, month) - WorkStartDay(4) + 1) + 1
	}
	return kyPayDayInMonth[ky] - WorkStartDay(ky) + 1
}

// PayDate returns the pay date (date-only, DefaultLocation) for the Ky whose
// work window falls in (year, month). Ky 1–3 pay within the same month; Ky 4
// pays on day 1 of the following month. Returns the zero time for an invalid Ky.
func PayDate(ky int, year int, month time.Month) time.Time {
	if ky < 1 || ky > TimesheetKyCount {
		return time.Time{}
	}
	if ky == 4 {
		return time.Date(year, month, 1, 0, 0, 0, 0, DefaultLocation).AddDate(0, 1, 0)
	}
	return time.Date(year, month, kyPayDayInMonth[ky], 0, 0, 0, 0, DefaultLocation)
}

// CycleDayForApproval maps an approval calendar date onto the cycle-day axis of
// the Ky whose work window opens at (year, month, WorkStartDay(ky)).
//
// cycle-day 1 = the work-start day; the pay date = MaxCycleDay. Approvals before
// the window clamp to 1; approvals after the pay date clamp to MaxCycleDay, so a
// late approval can never corrupt the cumulative curve. Returns 0 for an invalid
// Ky.
func CycleDayForApproval(ky int, year int, month time.Month, approvalDate time.Time) int {
	maxDay := MaxCycleDay(ky, year, month)
	if maxDay <= 0 {
		return 0
	}
	workStart := time.Date(year, month, WorkStartDay(ky), 0, 0, 0, 0, DefaultLocation)
	approvalDay := time.Date(approvalDate.In(DefaultLocation).Year(),
		approvalDate.In(DefaultLocation).Month(),
		approvalDate.In(DefaultLocation).Day(),
		0, 0, 0, 0, DefaultLocation)
	// Count calendar days via UTC-normalized date-only times so the result is
	// exact even if DefaultLocation ever observes DST (both operands become
	// midnight UTC, 24h apart per day).
	d := calendarDaysBetween(workStart, approvalDay) + 1
	if d < 1 {
		return 1
	}
	if d > maxDay {
		return maxDay
	}
	return d
}

// calendarDaysBetween returns the signed count of calendar days from `from` to
// `to` (to − from), DST-safe via UTC date-only normalization.
func calendarDaysBetween(from, to time.Time) int {
	const day = 24 * time.Hour
	fromUTC := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toUTC.Sub(fromUTC) / day)
}

// TimesheetPayCycle describes the next upcoming pay cycle relative to a time.
type TimesheetPayCycle struct {
	Ky            int       // 1..4
	WorkMonth     time.Time // date-only (1st of the work month), DefaultLocation
	NextPayDate   time.Time // date-only, DefaultLocation
	CycleDayToday int       // current position on the cycle-day axis (>= 1)
	MaxCycleDay   int       // pay date's cycle-day
}

// NextTimesheetPayCycle resolves the next upcoming pay cycle from t. The next
// pay date is the smallest Ky pay date >= today; on a pay day itself that Ky is
// current (the transfer is happening today). All times are interpreted in
// DefaultLocation (Asia/Ho_Chi_Minh).
func NextTimesheetPayCycle(t time.Time) TimesheetPayCycle {
	now := t.In(DefaultLocation)
	year, month, day := now.Year(), now.Month(), now.Day()

	// The next pay date is determined by today's day-of-month. The thresholds
	// mirror the 4-cycle table (pay days 10 / 17 / 24 / day-1-next-month).
	var ky int
	switch {
	case day <= 10:
		ky = 1
	case day <= 17:
		ky = 2
	case day <= 24:
		ky = 3
	default:
		ky = 4
	}

	today := time.Date(year, month, day, 0, 0, 0, 0, DefaultLocation)
	workMonth := time.Date(year, month, 1, 0, 0, 0, 0, DefaultLocation)
	return TimesheetPayCycle{
		Ky:            ky,
		WorkMonth:     workMonth,
		NextPayDate:   PayDate(ky, year, month),
		CycleDayToday: CycleDayForApproval(ky, year, month, today),
		MaxCycleDay:   MaxCycleDay(ky, year, month),
	}
}

// PrepareByDate returns the date by which cash should be ready:
// next pay date − leadDays. A negative leadDays is treated as 0.
func (c TimesheetPayCycle) PrepareByDate(leadDays int) time.Time {
	if leadDays < 0 {
		leadDays = 0
	}
	return c.NextPayDate.AddDate(0, 0, -leadDays)
}
