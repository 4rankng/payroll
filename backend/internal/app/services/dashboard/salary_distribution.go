package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"math"
	"sort"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
)

// GetSalaryDistribution retrieves salary distribution data for dashboard charts,
// grouped by payment cycle (weekly, monthly, flexible).
func (s *Service) GetSalaryDistribution(ctx context.Context, req *dto.SalaryDistributionRequest) (*dto.SalaryDistributionResponse, *dto.SalaryDistributionMetaResponse, error) {
	fromDate, toDate, fromDateStr, toDateStr := resolveDateRange(req)

	s.logger.Info("Getting salary distribution data", "from", fromDateStr, "to", toDateStr)

	// Try to get from cache first
	cacheKey := s.CacheService.GenerateDashboardCacheKey("salary_distribution", fromDateStr, toDateStr)
	var cachedResponse struct {
		Data *dto.SalaryDistributionResponse     `json:"data"`
		Meta *dto.SalaryDistributionMetaResponse `json:"meta"`
	}
	if err := s.CacheService.Get(ctx, cacheKey, &cachedResponse); err == nil {
		s.logger.Info("Salary distribution retrieved from cache", "from", fromDateStr, "to", toDateStr)
		return cachedResponse.Data, cachedResponse.Meta, nil
	}

	// Get salary values from repository grouped by cycle
	cycleSalaries, err := s.BulkTransferFileRepo.GetSalaryDistribution(ctx, fromDate, toDate)
	if err != nil {
		s.logger.Error("Failed to get salary distribution", "error", err)
		return nil, nil, err
	}

	// Build grouped response
	response := &dto.SalaryDistributionResponse{}
	for _, cycle := range []string{"weekly", "monthly", "flexible"} {
		salaries, ok := cycleSalaries[cycle]
		if !ok || len(salaries) == 0 {
			continue
		}

		summary := calculateSalaryStatistics(salaries)
		cycleData := &dto.SalaryDistributionCycleData{
			Values:  salaries,
			Summary: summary,
		}

		switch cycle {
		case "weekly":
			response.Weekly = cycleData
		case "monthly":
			response.Monthly = cycleData
		case "flexible":
			response.Flexible = cycleData
		}

		s.logger.Info("Salary distribution cycle computed", "cycle", cycle, "count", summary.Count, "mean", summary.Mean)
	}

	meta := &dto.SalaryDistributionMetaResponse{
		Period: dto.SalaryDistributionPeriod{
			From: fromDateStr,
			To:   toDateStr,
		},
	}

	// Use a shorter TTL for the current month (data changes frequently) or when result is empty
	cacheTTL := constants.DashboardSalaryDistributionTTL
	now := clock.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	currentMonthEnd := currentMonthStart.AddDate(0, 1, -1)
	isCurrentMonth := fromDateStr == currentMonthStart.Format("2006-01-02") && toDateStr == currentMonthEnd.Format("2006-01-02")
	isEmpty := response.Weekly == nil && response.Monthly == nil && response.Flexible == nil
	if isCurrentMonth || isEmpty {
		cacheTTL = 5 * time.Minute
	}

	if err := s.CacheService.Set(ctx, cacheKey, map[string]interface{}{
		"data": response,
		"meta": meta,
	}, cacheTTL); err != nil {
		s.logger.Warn("Failed to cache salary distribution response", "error", err)
	}

	return response, meta, nil
}

// resolveDateRange derives fromDate/toDate from the request parameters.
// Priority: fromDate+toDate > month > default (current month).
func resolveDateRange(req *dto.SalaryDistributionRequest) (*time.Time, *time.Time, string, string) {
	now := clock.Now()

	// If fromDate and toDate are provided, use them directly
	if req.FromDate != nil && *req.FromDate != "" && req.ToDate != nil && *req.ToDate != "" {
		fd, err1 := time.Parse("2006-01-02", *req.FromDate)
		td, err2 := time.Parse("2006-01-02", *req.ToDate)
		if err1 == nil && err2 == nil {
			return &fd, &td, fd.Format("2006-01-02"), td.Format("2006-01-02")
		}
	}

	// If month is provided, compute first/last day
	if req.Month != nil && *req.Month != "" {
		parsed, err := time.Parse("2006-01", *req.Month)
		if err == nil {
			firstDay := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
			lastDay := firstDay.AddDate(0, 1, -1)
			return &firstDay, &lastDay, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02")
		}
	}

	// Default: current month
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	return &firstDay, &lastDay, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02")
}

// calculateSalaryStatistics calculates statistical measures for salary data
func calculateSalaryStatistics(salaries []int64) dto.SalaryDistributionSummary {
	n := len(salaries)
	if n == 0 {
		return dto.SalaryDistributionSummary{}
	}

	// Ensure salaries are sorted (they should be from repository, but ensure)
	sort.Slice(salaries, func(i, j int) bool {
		return salaries[i] < salaries[j]
	})

	// Count
	count := n

	// Min and Max
	min := salaries[0]
	max := salaries[n-1]

	// Mean (average)
	var sum int64
	for _, salary := range salaries {
		sum += salary
	}
	mean := sum / int64(count)

	// Median
	var median int64
	if count%2 == 0 {
		median = (salaries[count/2-1] + salaries[count/2]) / 2
	} else {
		median = salaries[count/2]
	}

	// Standard deviation
	var variance float64
	for _, salary := range salaries {
		diff := float64(salary) - float64(mean)
		variance += diff * diff
	}
	variance /= float64(count)
	stdDev := int64(math.Sqrt(variance))

	return dto.SalaryDistributionSummary{
		Count:  count,
		Mean:   mean,
		Median: median,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
	}
}
