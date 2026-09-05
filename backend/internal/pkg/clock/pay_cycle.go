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
// (day 20 → day 8, modelled in advance_payment.go) — the two must never share
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

// NextTimesheetPayCycle resolves the next pay cycle the admin should be
// preparing cash for, relative to t. The target is the smallest Ky pay date
// STRICTLY AFTER today: once today reaches a cycle's pay date, that cycle's
// transfer is happening and its preparation window (pay date − lead days) has
// already closed, so the card rolls forward to the next cycle.
//
// Concretely (HCM calendar): days 1–9 → Ky 1 (pay 10); days 10–16 → Ky 2
// (pay 17); days 17–23 → Ky 3 (pay 24); days 24–end → Ky 4 (pay day 1 of the
// next month). Keeping CycleDayToday strictly below MaxCycleDay is also what
// leaves the projection window open — landing on the pay day itself would
// collapse the accrual forecast to zero. All times are interpreted in
// DefaultLocation (Asia/Ho_Chi_Minh).
func NextTimesheetPayCycle(t time.Time) TimesheetPayCycle {
	now := t.In(DefaultLocation)
	year, month := now.Year(), now.Month()
	today := time.Date(year, month, now.Day(), 0, 0, 0, 0, DefaultLocation)
	workMonth := time.Date(year, month, 1, 0, 0, 0, 0, DefaultLocation)

	for ky := 1; ky <= TimesheetKyCount; ky++ {
		pay := PayDate(ky, year, month)
		if !pay.After(today) {
			continue
		}
		return TimesheetPayCycle{
			Ky:            ky,
			WorkMonth:     workMonth,
			NextPayDate:   pay,
			CycleDayToday: CycleDayForApproval(ky, year, month, today),
			MaxCycleDay:   MaxCycleDay(ky, year, month),
		}
	}
	// Defensive fallback: Ky 4 pays on day 1 of next month, which is always
	// strictly after any day in this month, so the loop returns by Ky 4 in
	// practice. Reaching here would only mean t is itself past day 1 of next
	// month, in which case Ky 4 of this month is the correct nearest cycle.
	ky := TimesheetKyCount
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

// NextPayCycleAfter returns the pay cycle strictly after the given one,
// walking the monthly Kỳ 1→2→3→4 sequence and wrapping Kỳ 4 to Kỳ 1 of the
// next month. Used by the settlement simulation to project "current + next N"
// cycles without re-resolving from wall-clock time.
//
// The returned cycle's WorkMonth is the first-of-month containing the next
// cycle's work window. Kỳ 4's work window is in the current month but pays on
// day 1 of the next month, so NextPayCycleAfter(Kỳ 4 of month M) = Kỳ 1 of
// month M+1 (not M).
func NextPayCycleAfter(current TimesheetPayCycle) TimesheetPayCycle {
	nextKy := current.Ky + 1
	workMonth := current.WorkMonth
	if nextKy > TimesheetKyCount {
		// Wrap: Kỳ 4 → Kỳ 1 of next month.
		nextKy = 1
		workMonth = workMonth.AddDate(0, 1, 0)
	}
	year, month := workMonth.Year(), workMonth.Month()
	return TimesheetPayCycle{
		Ky:            nextKy,
		WorkMonth:     workMonth,
		NextPayDate:   PayDate(nextKy, year, month),
		MaxCycleDay:   MaxCycleDay(nextKy, year, month),
		CycleDayToday: 0, // not meaningful for a projected future cycle
	}
}

// CycleWindow returns the [from, to] work-day window (inclusive, HCM timezone,
// date-only) for a cycle. Kỳ 1 → days 1–7, Kỳ 2 → 8–14, Kỳ 3 → 15–21,
// Kỳ 4 → 22 through last day of the work month. Used by the settlement
// simulation to scope each projected cycle's ExportPlanner.Plan() call.
func (c TimesheetPayCycle) CycleWindow() (from, to time.Time) {
	year, month := c.WorkMonth.Year(), c.WorkMonth.Month()
	startDay := WorkStartDay(c.Ky)
	from = time.Date(year, month, startDay, 0, 0, 0, 0, DefaultLocation)
	if c.Ky == TimesheetKyCount {
		// Kỳ 4 runs from day 22 to the last day of the work month.
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, DefaultLocation).Day()
		to = time.Date(year, month, lastDay, 0, 0, 0, 0, DefaultLocation)
	} else {
		to = from.AddDate(0, 0, 6) // 7-day window
	}
	return from, to
}
