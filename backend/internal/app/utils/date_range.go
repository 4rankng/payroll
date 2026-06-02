package utils

import "time"

// CalculateDateRange determines the start and end dates of a salary period
// based on a reference date (atDate) and cycle configuration.
func CalculateDateRange(atDate time.Time, salaryPeriodFrom int, salaryPeriodTo int) (time.Time, time.Time) {
	// 1. Determine Anchor Month using day-of-month heuristic.
	// Early month (day ≤ 15): anchor to previous month (previous cycle just ended).
	// Late month (day > 15): anchor to current month (current cycle is running/complete).
	// This applies uniformly to all period types including weekly (salaryPeriodTo == 0).
	var anchorDate time.Time
	if atDate.Day() <= 15 {
		anchorDate = atDate.AddDate(0, -1, 0)
	} else {
		anchorDate = atDate
	}

	// 2. Calculate fromDate
	var fromDate time.Time
	if salaryPeriodFrom > 0 {
		// Custom cycle (e.g. start on the 20th).
		// The start date is in the month PRIOR to the Anchor Month.
		// Example: Anchor Nov, Start Day 20 -> Oct 20
		fromDate = time.Date(anchorDate.Year(), anchorDate.Month()-1, salaryPeriodFrom, 0, 0, 0, 0, atDate.Location())
	} else {
		// Standard cycle (Start of month).
		// The start date is the 1st of the Anchor Month.
		// Example: Anchor Nov -> Nov 1
		fromDate = time.Date(anchorDate.Year(), anchorDate.Month(), 1, 0, 0, 0, 0, atDate.Location())
	}

	// 3. Calculate toDate
	var toDate time.Time
	if salaryPeriodTo > 0 {
		// Custom cycle (e.g. end on the 21st).
		// The end date is in the Anchor Month.
		toDate = time.Date(anchorDate.Year(), anchorDate.Month(), salaryPeriodTo, 0, 0, 0, 0, atDate.Location())
	} else {
		// Standard cycle (End of month).
		// The end date is the last day of the Anchor Month.
		// Trick: Set date to the 1st of the NEXT month, then subtract 1 day.
		nextMonth := time.Date(anchorDate.Year(), anchorDate.Month()+1, 1, 0, 0, 0, 0, atDate.Location())
		toDate = nextMonth.AddDate(0, 0, -1)
	}

	return fromDate, toDate
}
