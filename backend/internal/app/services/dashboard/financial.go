package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/dashboard/calculator"
	"api-server/internal/constants"
	"api-server/internal/domain"
)

// GetFinancialOverview retrieves financial data for dashboard charts
func (s *Service) GetFinancialOverview(ctx context.Context, req *dto.FinancialOverviewRequest) (*dto.FinancialOverviewResponse, error) {
	s.logger.Info("Getting financial overview", "period", req.Period, "months", req.Months, "year", req.Year)

	// Set defaults
	if req.Period == "" {
		req.Period = "month"
	}
	if req.Months <= 0 || req.Months > 24 {
		req.Months = 12
	}
	if req.Year <= 0 {
		req.Year = clock.Now().Year()
	}

	// Try to get from cache first
	cacheKey := s.CacheService.GenerateDashboardCacheKey("financial-overview", req.Period, fmt.Sprint(req.Months), fmt.Sprint(req.Year))
	var cachedResponse dto.FinancialOverviewResponse
	err := s.CacheService.Get(ctx, cacheKey, &cachedResponse)
	if err == nil {
		s.logger.Info("Financial overview retrieved from cache")
		return &cachedResponse, nil
	}

	// Calculate current and previous month
	now := clock.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	previousMonth := currentMonth.AddDate(0, -1, 0)

	response := &dto.FinancialOverviewResponse{
		Period: req.Period,
		Year:   req.Year,
	}

	// Calculate date range for batch query (includes current, previous, and chart data months)
	oldestMonth := currentMonth.AddDate(0, -(req.Months - 1), 0)
	startDate := time.Date(oldestMonth.Year(), oldestMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := currentMonth.AddDate(0, 1, 0).Add(-time.Second)

	// Fetch all monthly data in a single batch query
	monthlyData, err := s.LedgerRepo.GetMonthlyFinancialsBatch(ctx, startDate, endDate)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToGetMonthlyFinancialsVN, err)
	}

	// Extract current month data
	currentKey := currentMonth.Format("2006-01")
	current, currentExists := monthlyData[currentKey]
	var currentRevenue, currentExpenses int64
	if currentExists {
		currentRevenue = current.Revenue
		currentExpenses = current.Expenses
	}

	response.CurrentMonth = dto.CurrentMonthFinancial{
		Month:                  currentKey,
		TotalRevenueVND:        currentRevenue,
		TotalExpensesVND:       currentExpenses,
		NetProfitVND:           currentRevenue - currentExpenses,
		ProfitMarginPercentage: calculator.CalculateProfitMargin(currentRevenue, currentExpenses),
	}

	// Extract previous month data
	prevKey := previousMonth.Format("2006-01")
	previous, prevExists := monthlyData[prevKey]
	var prevRevenue, prevExpenses int64
	if prevExists {
		prevRevenue = previous.Revenue
		prevExpenses = previous.Expenses
	}

	response.PreviousMonth = dto.CurrentMonthFinancial{
		Month:                  prevKey,
		TotalRevenueVND:        prevRevenue,
		TotalExpensesVND:       prevExpenses,
		NetProfitVND:           prevRevenue - prevExpenses,
		ProfitMarginPercentage: calculator.CalculateProfitMargin(prevRevenue, prevExpenses),
	}

	// Calculate growth metrics
	response.Growth = dto.GrowthMetrics{
		RevenueChangePercentage: calculator.CalculatePercentageChangeFloat64(float64(prevRevenue), float64(currentRevenue)),
		ExpenseChangePercentage: calculator.CalculatePercentageChangeFloat64(float64(prevExpenses), float64(currentExpenses)),
		ProfitChangePercentage:  calculator.CalculatePercentageChangeFloat64(float64(prevRevenue-prevExpenses), float64(currentRevenue-currentExpenses)),
	}

	// Build chart data from batch results
	chartData := make([]dto.MonthlyFinancialData, 0, req.Months)
	for i := req.Months - 1; i >= 0; i-- {
		month := currentMonth.AddDate(0, -i, 0)
		monthKey := month.Format("2006-01")

		var revenue, expenses int64
		if data, exists := monthlyData[monthKey]; exists {
			revenue = data.Revenue
			expenses = data.Expenses
		}

		chartData = append(chartData, dto.MonthlyFinancialData{
			Month:       monthKey,
			RevenueVND:  revenue,
			ExpensesVND: expenses,
		})
	}

	response.ChartData = chartData

	// Store in cache for future requests
	if cacheErr := s.CacheService.Set(ctx, cacheKey, response, constants.DashboardFinancialCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache financial overview", "error", cacheErr)
	}

	s.logger.Info("Financial overview retrieved successfully",
		"current_revenue", currentRevenue,
		"current_expenses", currentExpenses,
		"chart_points", len(chartData))

	return response, nil
}

// GetFinancialChartData retrieves financial chart data based on period
func (s *Service) GetFinancialChartData(ctx context.Context, req *dto.FinancialChartRequest) (*dto.FinancialChartResponse, error) {
	s.logger.Info("Getting financial chart data", "period", req.Period)

	if req.Period == "" {
		req.Period = "month"
	}

	now := clock.Now()
	var startDate time.Time
	switch req.Period {
	case "day":
		startDate = now.AddDate(0, 0, -180)
	case "week":
		startDate = now.AddDate(0, 0, -364)
	case "month":
		startDate = now.AddDate(-2, 0, 0)
	case "quarter":
		startDate = now.AddDate(-3, 0, 0)
	case "year":
		startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return nil, domain.NewValidationError(constants.MsgInvalidPeriodFormatVN)
	}

	dataPoints, err := s.LedgerRepo.GetFinancialChartData(ctx, req.Period, startDate, now)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToGetFinancialChartDataVN, err)
	}

	// Resolve each data point's date to a time.Time for the rolling-window calculation
	type pointDate struct {
		idx  int
		date time.Time
	}
	resolved := make([]pointDate, 0, len(dataPoints))
	for i, dp := range dataPoints {
		t, err := parseChartDate(dp.Date)
		if err == nil {
			resolved = append(resolved, pointDate{i, t})
		}
	}

	if len(resolved) > 0 {
		// Determine overall date range needed for a single timesheet query
		earliest := resolved[0].date.AddDate(0, 0, -29)
		latest := resolved[len(resolved)-1].date

		var paidTimesheets []*domain.Timesheet
		if err := s.TimesheetRepo.GetPaidTimesheetsInDateRange(ctx, earliest, latest, &paidTimesheets); err != nil {
			s.logger.Warn("Failed to query paid timesheets for active employee counts", "error", err)
		} else {
			// Build date → employee set map
			dateEmployees := make(map[string]map[uint]struct{})
			for _, ts := range paidTimesheets {
				var dk string
				if ts.PaidAt != nil {
					dk = ts.PaidAt.Format("2006-01-02")
				} else if ts.PaymentDate != nil {
					dk = ts.PaymentDate.Format("2006-01-02")
				} else {
					continue
				}
				if dateEmployees[dk] == nil {
					dateEmployees[dk] = make(map[uint]struct{})
				}
				dateEmployees[dk][ts.EmployeeID] = struct{}{}
			}

			for _, pd := range resolved {
				end := time.Date(pd.date.Year(), pd.date.Month(), pd.date.Day(), 23, 59, 59, 0, time.UTC)
				start := end.AddDate(0, 0, -29)
				unique := make(map[uint]struct{})
				for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
					for empID := range dateEmployees[d.Format("2006-01-02")] {
						unique[empID] = struct{}{}
					}
				}
				dataPoints[pd.idx].ActiveEmployees = len(unique)
			}
		}
	}

	response := make(dto.FinancialChartResponse, len(dataPoints))
	for i, p := range dataPoints {
		response[i] = dto.FinancialDataPoint{
			Date:            p.Date,
			CashVND:         p.CashVND,
			ReceivableVND:   p.ReceivableVND,
			PayableVND:      p.PayableVND,
			RevenueVND:      p.RevenueVND,
			ExpensesVND:     p.ExpensesVND,
			ProfitVND:       p.ProfitVND,
			ActiveEmployees: p.ActiveEmployees,
		}
	}

	s.logger.Info("Financial chart data retrieved", "period", req.Period, "data_points", len(response))
	return &response, nil
}

// parseChartDate parses YYYY-MM-DD or YYYY-MM date strings into time.Time.
func parseChartDate(s string) (time.Time, error) {
	if len(s) == 10 {
		return time.Parse("2006-01-02", s)
	}
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, err
	}
	// Use end-of-month for monthly data points
	return t.AddDate(0, 1, 0).Add(-time.Nanosecond), nil
}
