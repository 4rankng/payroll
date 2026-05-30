package utils

import (
	"time"
)

// PayrollPeriod represents a Vietnamese payroll period
type PayrollPeriod struct {
	StartDate    time.Time
	EndDate      time.Time
	PayDate      time.Time
	PeriodNumber int // 1-4 representing the 4 periods in a month
}

// GetPayrollPeriodForDate returns the payroll period for a given date
func GetPayrollPeriodForDate(date time.Time) PayrollPeriod {
	day := date.Day()
	year := date.Year()
	month := date.Month()

	switch {
	case day >= 1 && day <= 7:
		// Period 1: Days 1-7 (paid on 7th)
		return PayrollPeriod{
			StartDate:    time.Date(year, month, 1, 0, 0, 0, 0, date.Location()),
			EndDate:      time.Date(year, month, 7, 23, 59, 59, 999999999, date.Location()),
			PayDate:      time.Date(year, month, 7, 0, 0, 0, 0, date.Location()),
			PeriodNumber: 1,
		}
	case day >= 8 && day <= 14:
		// Period 2: Days 8-14 (paid on 14th)
		return PayrollPeriod{
			StartDate:    time.Date(year, month, 8, 0, 0, 0, 0, date.Location()),
			EndDate:      time.Date(year, month, 14, 23, 59, 59, 999999999, date.Location()),
			PayDate:      time.Date(year, month, 14, 0, 0, 0, 0, date.Location()),
			PeriodNumber: 2,
		}
	case day >= 15 && day <= 21:
		// Period 3: Days 15-21 (paid on 21st)
		return PayrollPeriod{
			StartDate:    time.Date(year, month, 15, 0, 0, 0, 0, date.Location()),
			EndDate:      time.Date(year, month, 21, 23, 59, 59, 999999999, date.Location()),
			PayDate:      time.Date(year, month, 21, 0, 0, 0, 0, date.Location()),
			PeriodNumber: 3,
		}
	default:
		// Period 4: Days 22-28 or end of month (paid on 28th or last day if month has <28 days)
		lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, date.Location()).Day()
		payDay := min(28, lastDayOfMonth)

		return PayrollPeriod{
			StartDate:    time.Date(year, month, 22, 0, 0, 0, 0, date.Location()),
			EndDate:      time.Date(year, month, payDay, 23, 59, 59, 999999999, date.Location()),
			PayDate:      time.Date(year, month, payDay, 0, 0, 0, 0, date.Location()),
			PeriodNumber: 4,
		}
	}
}

// GetPayrollPeriodsInRange generates all payroll periods within the given date range
func GetPayrollPeriodsInRange(startDate, endDate time.Time) []PayrollPeriod {
	var periods []PayrollPeriod

	// Start from the first day of the period containing startDate
	currentDate := startDate
	period := GetPayrollPeriodForDate(currentDate)

	// Adjust to start from the beginning of the period
	currentDate = period.StartDate

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		period = GetPayrollPeriodForDate(currentDate)

		// Add period if it overlaps with the date range
		if period.EndDate.After(startDate) && period.StartDate.Before(endDate) {
			periods = append(periods, period)
		}

		// Move to the next period
		currentDate = period.PayDate.AddDate(0, 0, 1)

		// Prevent infinite loop
		if len(periods) > 100 {
			break
		}
	}

	return periods
}

// GetPayrollPeriodEndDate returns the end date for the period containing the given date
func GetPayrollPeriodEndDate(date time.Time) time.Time {
	period := GetPayrollPeriodForDate(date)
	return period.PayDate
}

// IsPayrollPeriodEnd checks if the given date is a payroll period end date
func IsPayrollPeriodEnd(date time.Time) bool {
	day := date.Day()
	return day == 7 || day == 14 || day == 21 || day == 28
}

// GetPreviousPayrollPeriodEnd returns the most recent payroll period end date before the given date
func GetPreviousPayrollPeriodEnd(date time.Time) time.Time {
	// Start from yesterday and go back to find the last period end
	currentDate := date.AddDate(0, 0, -1)

	for i := 0; i < 31; i++ { // Search max 31 days back
		if IsPayrollPeriodEnd(currentDate) {
			return currentDate
		}
		currentDate = currentDate.AddDate(0, 0, -1)
	}

	// If no period end found in the last month, return the 28th of previous month
	return time.Date(date.Year(), date.Month()-1, 28, 0, 0, 0, 0, date.Location())
}
