package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"sync"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
)

// GetDashboardSummary retrieves dashboard summary statistics
// monthParam is an optional string in YYYY-MM format (e.g., "2025-04"). If nil, uses current month.
func (s *Service) GetDashboardSummary(ctx context.Context, monthParam *string) (*dto.DashboardSummaryResponse, error) {
	s.logger.Info("Getting dashboard summary", "month_param", monthParam)

	// Current time for all-time totals
	now := clock.Now()

	// Parse month parameter or use current month
	var targetMonth time.Time
	var monthKey string
	if monthParam != nil && *monthParam != "" {
		// Parse YYYY-MM format
		parsedMonth, err := time.Parse("2006-01", *monthParam)
		if err != nil {
			s.logger.Warn("Invalid month format", "month", *monthParam, "error", err)
			return nil, domain.NewValidationError(constants.MsgInvalidFromDateFormatVN)
		}
		targetMonth = parsedMonth
		monthKey = *monthParam
	} else {
		targetMonth = now
		monthKey = targetMonth.Format("2006-01")
	}

	// Try to get from cache first
	cacheKey := s.CacheService.GenerateDashboardCacheKey("summary", monthKey)
	var cachedResponse dto.DashboardSummaryResponse
	if err := s.CacheService.Get(ctx, cacheKey, &cachedResponse); err == nil {
		s.logger.Info("Dashboard summary retrieved from cache", "month", monthKey)
		return &cachedResponse, nil
	}

	loc := targetMonth.Location()
	startOfMonth := time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, loc)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	// Use WaitGroup to parallelize independent queries
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Variables to store results
	var totalEmployees, employeesHiredThisMonth int64
	var workingEmployees, weeklyEmployeesCount, monthlyEmployeesCount int
	var totalPaidSalary, pendingSalaryThisMonth, paidSalaryThisMonth int64
	var avgWeeklySalary, avgMonthlySalary int64
	var queryErrors []error

	// Helper to track errors
	addError := func(err error) {
		if err != nil {
			mu.Lock()
			queryErrors = append(queryErrors, err)
			mu.Unlock()
		}
	}

	// Group 1: Employee counts (3 queries in parallel)
	wg.Add(3)
	go func() {
		defer wg.Done()
		// Count only employees currently assigned to projects
		count, err := s.ProjectEmployeeRepo.CountWorkingEmployees(ctx)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToCountTotalEmployeesVN, err))
			return
		}
		totalEmployees = int64(count)
	}()

	go func() {
		defer wg.Done()
		// Use weekly period data to match historical endpoint's weekly_pay.paid_employees
		endDate := clock.Now()
		startDate := endDate.AddDate(-1, 0, 0) // Last 365 days to get latest period

		results, err := s.TimesheetQueryRepo.GetWeeklyPayByPeriod(ctx, startDate, endDate)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToCountWorkingEmployeesVN, err))
			return
		}

		// Extract the most recent period's paid_employees count (last entry in results)
		if len(results) > 0 {
			lastResult := results[len(results)-1]
			if count, ok := lastResult["paid_employees"]; ok {
				switch v := count.(type) {
				case int64:
					workingEmployees = int(v)
				case float64:
					workingEmployees = int(v)
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		summary, err := s.EmployeeRepo.GetEmployeesSummary(ctx)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToGetEmployeesSummaryForHiresVN, err))
			return
		}
		employeesHiredThisMonth = summary.EmployeesHiredThisMonth
	}()

	// Group 2: Salary queries (3 queries in parallel)
	wg.Add(3)
	go func() {
		defer wg.Done()
		salary, err := s.TimesheetDashboard.GetTotalPaidSalary(ctx)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToGetTotalPaidSalaryVN, err))
			return
		}
		totalPaidSalary = salary
	}()

	go func() {
		defer wg.Done()

		salary, err := s.TimesheetDashboard.GetPendingSalaryForMonth(ctx, startOfMonth, endOfMonth)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToGetPendingSalaryVN, err))
			return
		}
		s.logger.Info("Getting pending salary for current month",
			"month_start", startOfMonth.Format("2006-01-02"),
			"month_end", endOfMonth.Format("2006-01-02"))
		pendingSalaryThisMonth = salary
	}()

	go func() {
		defer wg.Done()

		salary, err := s.TimesheetDashboard.GetPaidSalaryForMonth(ctx, startOfMonth, endOfMonth)
		if err != nil {
			addError(domain.NewInternalError(constants.MsgFailedToGetPaidSalaryVN, err))
			return
		}
		s.logger.Info("Getting paid salary for current month",
			"month_start", startOfMonth.Format("2006-01-02"),
			"month_end", endOfMonth.Format("2006-01-02"))
		paidSalaryThisMonth = salary
	}()

	// Group 3: All-time and monthly profit from timesheets (revenue_receivable - paid_amount where profit > 0)
	// plus advance payment fees
	wg.Add(4)
	var allTimeTimesheetRevenue, allTimeTimesheetProfit int64
	go func() {
		defer wg.Done()
		summary, err := s.TimesheetAnalyticsRepo.GetAllTimeProfitSummaryFromTimesheets(ctx)
		if err != nil {
			s.logger.Warn("Failed to get all-time timesheet profit", "error", err)
			return
		}
		allTimeTimesheetRevenue = summary.TotalRevenue
		allTimeTimesheetProfit = summary.TotalProfit
	}()

	// Monthly profit from timesheets
	var monthlyTimesheetProfit int64
	var monthlyTimesheetRevenue int64
	go func() {
		defer wg.Done()
		summary, err := s.TimesheetAnalyticsRepo.GetProfitSummaryFromTimesheets(ctx, startOfMonth, startOfMonth.AddDate(0, 1, 0))
		if err != nil {
			s.logger.Warn("Failed to get monthly timesheet profit", "error", err)
			return
		}
		monthlyTimesheetRevenue = summary.TotalRevenue
		monthlyTimesheetProfit = summary.TotalProfit
	}()

	// Monthly advance payment fees (completed)
	var monthlyAdvanceFeeEarned int64
	go func() {
		defer wg.Done()
		stats, err := s.AdvancePaymentRequestRepo.GetStatsSummary(ctx, startOfMonth, startOfMonth.AddDate(0, 1, 0), "")
		if err != nil {
			s.logger.Warn("Failed to get monthly advance payment fee stats", "error", err)
			return
		}
		monthlyAdvanceFeeEarned = int64(stats.TotalFeeEarned)
	}()

	// All-time advance payment fees (completed)
	var allTimeAdvanceFeeEarned int64
	go func() {
		defer wg.Done()
		stats, err := s.AdvancePaymentRequestRepo.GetStatsSummary(ctx, time.Time{}, time.Time{}, "")
		if err != nil {
			s.logger.Warn("Failed to get all-time advance payment fee stats", "error", err)
			return
		}
		allTimeAdvanceFeeEarned = int64(stats.TotalFeeEarned)
	}()

	// Group 4: Payment schedule counts (2 queries in parallel)
	wg.Add(2)
	go func() {
		defer wg.Done()
		count, err := s.ProjectEmployeeRepo.CountActiveEmployeesByPaymentSchedule(ctx, domain.PaymentScheduleWeekly)
		if err != nil {
			s.logger.Warn("Failed to count weekly employees", "error", err)
			return
		}
		weeklyEmployeesCount = count
	}()

	go func() {
		defer wg.Done()
		count, err := s.ProjectEmployeeRepo.CountActiveEmployeesByPaymentSchedule(ctx, domain.PaymentScheduleMonthly)
		if err != nil {
			s.logger.Warn("Failed to count monthly employees", "error", err)
			return
		}
		monthlyEmployeesCount = count
	}()

	// Wait for all parallel queries to complete
	wg.Wait()

	// Check for critical errors
	if len(queryErrors) > 0 {
		return nil, queryErrors[0]
	}

	// Calculate totals: timesheet profit + advance payment fees
	totalProfit := allTimeTimesheetProfit + allTimeAdvanceFeeEarned
	totalProfitThisMonth := monthlyTimesheetProfit + monthlyAdvanceFeeEarned

	// Calculate average salaries from actual transfer data (direct call — no goroutine needed)
	weeklyAvg, monthlyAvg, err := s.SalaryCalculationSvc.CalculateAverageSalariesFromTransfers(ctx)
	if err != nil {
		s.logger.Warn("Failed to calculate average salaries from transfer data", "error", err)
	} else {
		avgWeeklySalary = weeklyAvg
		avgMonthlySalary = monthlyAvg
	}

	response := &dto.DashboardSummaryResponse{
		TotalEmployees:              int(totalEmployees),
		TotalWorkingEmployees:       workingEmployees,
		EmployeesHiredThisMonth:     int(employeesHiredThisMonth),
		TotalPaidSalary:             totalPaidSalary,
		PendingSalaryThisMonth:      pendingSalaryThisMonth,
		PaidSalaryThisMonth:         paidSalaryThisMonth,
		TotalRevenue:                allTimeTimesheetRevenue,
		TotalRevenueThisMonth:       monthlyTimesheetRevenue,
		TotalProfit:                 totalProfit,
		TotalProfitThisMonth:        totalProfitThisMonth,
		TotalWeeklySalaryEmployees:  weeklyEmployeesCount,
		TotalMonthlySalaryEmployees: monthlyEmployeesCount,
		AvgWeeklySalary:             avgWeeklySalary,
		AvgMonthlySalary:            avgMonthlySalary,
	}

	// Store in cache for future requests
	if cacheErr := s.CacheService.Set(ctx, cacheKey, response, constants.DashboardSummaryCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache dashboard summary", "error", cacheErr)
	}

	s.logger.Info("Dashboard summary retrieved successfully",
		"total_employees", response.TotalEmployees,
		"employees_paid_latest_weekly_period", response.TotalWorkingEmployees,
		"hired_this_month", response.EmployeesHiredThisMonth)

	return response, nil
}
