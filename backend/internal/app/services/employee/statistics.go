package employee

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
)

// GetEmployeeStatistics delegates to domain service for business logic
func (s *EmployeeService) GetEmployeeStatistics(ctx context.Context) (*domain.EmployeeStatistics, error) {
	return s.EmployeeDomainService.GetEmployeeStatistics(ctx)
}

// GetEmployeesSummary delegates to domain service for business logic with caching
func (s *EmployeeService) GetEmployeesSummary(ctx context.Context) (*domain.EmployeesSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employees_summary")
	var summary domain.EmployeesSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.EmployeeDomainService.GetEmployeesSummary(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)

	return result, nil
}

// GetEmployeesSummaryForCreator gets summary for employees accessible to a specific user with caching
// Includes employees created by the user, shared with the user, or assigned to projects by the user
func (s *EmployeeService) GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*domain.EmployeesSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employees_summary_creator", strconv.FormatUint(uint64(createdBy), 10))
	var summary domain.EmployeesSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.EmployeeDomainService.GetEmployeesSummaryForCreator(ctx, createdBy)
	if err != nil {
		return nil, err
	}

	// Cache the result with appropriate TTL
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)

	return result, nil
}

// EmployeeSummary represents individual employee summary data
type EmployeeSummary struct {
	TotalPayrollPayments int
	TotalEarningsVND     int64
	LastPaymentDate      *time.Time
	AvgWeeklyEarningsVND int64
}

func (s *EmployeeService) GetEmployeeSummary(ctx context.Context, employeeID uint) (*EmployeeSummary, error) {
	// Try to get from cache
	cacheKey := s.cache.GenerateDashboardCacheKey("employee_summary", strconv.FormatUint(uint64(employeeID), 10))
	var summary EmployeeSummary
	if err := s.cache.Get(ctx, cacheKey, &summary); err == nil {
		return &summary, nil
	}

	// Fetch all paid timesheets for this employee, sorted by payment_date desc.
	// We use the most recent payment_date to determine the "last paid month", then
	// aggregate totals for that month — all from this single result set.
	filters := domain.TimesheetFilters{
		EmployeeID:    &employeeID,
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		Limit:         1000,
		SortBy:        "payment_date",
		SortOrder:     "desc",
	}

	timesheets, err := s.TimesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get paid timesheets: %w", err)
	}

	empty := &EmployeeSummary{}
	if len(timesheets) == 0 || timesheets[0].PaymentDate == nil {
		_ = s.cache.Set(ctx, cacheKey, empty, constants.EmployeeTimesheetSummaryCacheTTL)
		return empty, nil
	}

	// Determine the last paid month from the first (most recent) timesheet.
	lastPaid := timesheets[0].PaymentDate
	year, month, _ := lastPaid.Date()
	loc := lastPaid.Location()
	firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	lastDayOfMonth := timeutil.EndOfDay(firstDayOfMonth.AddDate(0, 1, -1))

	// Aggregate totals for that month from the already-fetched slice (no second DB call).
	var totalEarnings int64
	var lastPaymentDate *time.Time
	count := 0

	for _, ts := range timesheets {
		if ts.PaymentDate == nil {
			continue
		}
		if ts.PaymentDate.Before(firstDayOfMonth) || ts.PaymentDate.After(lastDayOfMonth) {
			continue
		}
		if ts.Amount < 0 {
			continue
		}
		totalEarnings += ts.Amount
		count++
		if lastPaymentDate == nil || ts.PaymentDate.After(*lastPaymentDate) {
			lastPaymentDate = ts.PaymentDate
		}
	}

	avgWeekly := int64(0)
	if totalEarnings > 0 {
		avgWeekly = (totalEarnings + 2) / 4
	}

	result := &EmployeeSummary{
		TotalPayrollPayments: count,
		TotalEarningsVND:     totalEarnings,
		LastPaymentDate:      lastPaymentDate,
		AvgWeeklyEarningsVND: avgWeekly,
	}
	_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeePayrollSummaryCacheTTL)
	return result, nil
}
