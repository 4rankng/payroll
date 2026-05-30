package bulktransfer

import (
	"api-server/internal/constants"
	"context"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// PeriodCalculator handles date and period calculations for bulk transfers
type PeriodCalculator struct {
	projectRepo ProjectRepository
}

// NewPeriodCalculator creates a new PeriodCalculator instance
func NewPeriodCalculator(projectRepo ProjectRepository) *PeriodCalculator {
	return &PeriodCalculator{
		projectRepo: projectRepo,
	}
}

type projectPeriod struct {
	start time.Time
	end   time.Time
}

// ResolveWeeklyRange parses and validates weekly date range from request
func (pc *PeriodCalculator) ResolveWeeklyRange(req *dto.ExportBulkTransferRequest) (time.Time, time.Time, error) {
	if strings.TrimSpace(req.FromDate) == "" || strings.TrimSpace(req.ToDate) == "" {
		return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgFromDateRequiredForWeeklyVN)
	}

	loc, _ := time.LoadLocation("Local")
	fromDate, err := time.ParseInLocation("2006-01-02", req.FromDate, loc)
	if err != nil {
		return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgFromDateInvalidFormatVN2)
	}

	toDate, err := time.ParseInLocation("2006-01-02", req.ToDate, loc)
	if err != nil {
		return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgToDateInvalidFormatVN2)
	}

	if fromDate.After(toDate) {
		return time.Time{}, time.Time{}, domain.NewValidationError(constants.MsgFromDateAfterToDateVN2)
	}

	return fromDate, toDate, nil
}

// ResolveMonthlyRange parses and validates monthly period from request
func (pc *PeriodCalculator) ResolveMonthlyRange(req *dto.ExportBulkTransferRequest) (time.Time, error) {
	monthStr := strings.TrimSpace(req.ForMonth)
	if monthStr == "" {
		return time.Time{}, domain.NewValidationError(constants.MsgForMonthRequiredVN)
	}

	monthValue, err := time.Parse("2006-01", monthStr)
	if err != nil {
		return time.Time{}, domain.NewValidationError(constants.MsgForMonthInvalidFormatVN)
	}

	monthStart := time.Date(monthValue.Year(), monthValue.Month(), 1, 0, 0, 0, 0, time.UTC)
	return monthStart, nil
}

// GetProjectPeriod retrieves project-specific salary period with caching
func (pc *PeriodCalculator) GetProjectPeriod(ctx context.Context, projectID uint, monthStart time.Time, cache map[uint]projectPeriod) (projectPeriod, error) {
	if period, ok := cache[projectID]; ok {
		return period, nil
	}

	project, err := pc.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return projectPeriod{}, err
	}

	start := computeMonthlyStart(monthStart, project.SalaryPeriodFrom)
	end := computeMonthlyEnd(monthStart, project.SalaryPeriodTo)
	if end.Before(start) {
		end = endOfMonth(monthStart)
	}

	period := projectPeriod{start: start, end: end}
	cache[projectID] = period
	return period, nil
}

// computeMonthlyStart calculates the start date for a monthly period
func computeMonthlyStart(monthStart time.Time, day int) time.Time {
	if day == 0 {
		return monthStart
	}
	prevMonth := monthStart.AddDate(0, -1, 0)
	maxDay := daysInMonth(prevMonth)
	clamped := clampDay(day, maxDay)
	return time.Date(prevMonth.Year(), prevMonth.Month(), clamped, 0, 0, 0, 0, time.UTC)
}

// computeMonthlyEnd calculates the end date for a monthly period
func computeMonthlyEnd(monthStart time.Time, day int) time.Time {
	if day == 0 {
		return endOfMonth(monthStart)
	}
	maxDay := daysInMonth(monthStart)
	clamped := clampDay(day, maxDay)
	return time.Date(monthStart.Year(), monthStart.Month(), clamped, 0, 0, 0, 0, time.UTC)
}

// endOfMonth returns the last day of the month
func endOfMonth(monthStart time.Time) time.Time {
	return monthStart.AddDate(0, 1, -1)
}

// daysInMonth returns the number of days in a given month
func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

// clampDay ensures day value is within valid range [1, max]
func clampDay(day int, max int) int {
	if day < 1 {
		return 1
	}
	if day > max {
		return max
	}
	return day
}

// ParseDateRange parses and validates date range from string request parameters
func (pc *PeriodCalculator) ParseDateRange(fromDateStr, toDateStr string) (*time.Time, *time.Time, error) {
	loc, _ := time.LoadLocation("Local")
	var fromDate, toDate *time.Time

	if fromDateStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", fromDateStr, loc)
		if err != nil {
			return nil, nil, domain.NewValidationError(constants.MsgFromDateInvalidVN)
		}
		fromDate = &parsed
	}

	if toDateStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", toDateStr, loc)
		if err != nil {
			return nil, nil, domain.NewValidationError(constants.MsgToDateInvalidVN)
		}
		toDate = &parsed
	}

	if fromDate != nil && toDate != nil && fromDate.After(*toDate) {
		return nil, nil, domain.NewValidationError(constants.MsgFromDateAfterToDateVN3)
	}

	return fromDate, toDate, nil
}
