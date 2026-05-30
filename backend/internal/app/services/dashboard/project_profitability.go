package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"sort"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// GetProjectProfitability returns profitability ranking for each project over the last 12 months,
// excluding projects with zero profit.
func (s *Service) GetProjectProfitability(ctx context.Context) (*dto.ProjectProfitabilityResponse, error) {
	since := clock.Now().AddDate(0, -12, 0)

	rows, err := s.TimesheetAnalyticsRepo.GetProfitByProjectSince(ctx, since)
	if err != nil {
		s.logger.Warn("GetProfitByProjectSince failed", "error", err)
		return &dto.ProjectProfitabilityResponse{Projects: []dto.ProjectProfitabilityItem{}, TotalCount: 0}, nil
	}

	// Fetch employee counts for all projects in one query
	filters := domain.ProjectFilters{Limit: 500}
	projects, _ := s.ProjectRepo.ListWithEmployeeCount(ctx, filters)
	empCount := make(map[uint]int, len(projects))
	statusMap := make(map[uint]string, len(projects))
	for _, p := range projects {
		empCount[p.ID] = p.EmployeeCount
		statusMap[p.ID] = string(p.ProjectStatus)
	}

	items := make([]dto.ProjectProfitabilityItem, 0, len(rows))
	for _, r := range rows {
		// Exclude projects with zero profit
		if r.TotalProfit == 0 {
			continue
		}
		margin := 0.0
		if r.TotalRevenue > 0 {
			margin = (r.TotalProfit / r.TotalRevenue) * 100
		}
		items = append(items, dto.ProjectProfitabilityItem{
			ProjectID:           r.ProjectID,
			ProjectName:         r.ProjectName,
			ClientName:          r.ClientName,
			EmployeeCount:       empCount[r.ProjectID],
			TotalReceivedVND:    r.TotalRevenue,
			TotalPayoutVND:      r.TotalPayout,
			NetProfitVND:        r.TotalProfit,
			ProfitMarginPercent: margin,
			Status:              statusMap[r.ProjectID],
		})
	}

	// Already sorted by total_profit DESC from SQL, assign ranks
	for i := range items {
		items[i].Rank = uint(i + 1)
	}

	return &dto.ProjectProfitabilityResponse{
		Projects:   items,
		TotalCount: len(items),
	}, nil
}

// GetProjectWeeklyProfit returns cumulative daily profit per project for the last N days.
// Profit per timesheet row = revenue_receivable - paid_amount.
// Every day in the window is always included (days with no data get 0), so the chart
// always extends to today even when disbursements were just entered.
func (s *Service) GetProjectWeeklyProfit(ctx context.Context, days int) (*dto.ProjectWeeklyProfitResponse, error) {
	if days <= 0 || days > 365 {
		days = 84 // default ~12 weeks
	}

	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowStart := today.AddDate(0, 0, -(days - 1))

	rows, err := s.TimesheetAnalyticsRepo.GetDailyProfitByProject(ctx, windowStart)
	if err != nil {
		s.logger.Warn("GetProjectWeeklyProfit query failed", "error", err)
		return &dto.ProjectWeeklyProfitResponse{Series: []dto.ProjectWeeklySeries{}, Days: []string{}}, nil
	}

	// Build the full day spine from windowStart to today (inclusive).
	daySlice := make([]string, 0, days)
	for d := windowStart; !d.After(today); d = d.AddDate(0, 0, 1) {
		daySlice = append(daySlice, d.Format("2006-01-02"))
	}

	type projectKey struct {
		id   uint
		name string
	}
	seriesMap := make(map[projectKey]map[string]float64)

	for _, r := range rows {
		key := projectKey{r.ProjectID, r.ProjectName}
		if seriesMap[key] == nil {
			seriesMap[key] = make(map[string]float64)
		}
		seriesMap[key][r.DayLabel] = r.TotalProfit
	}

	series := make([]dto.ProjectWeeklySeries, 0, len(seriesMap))
	for key, dayData := range seriesMap {
		// Build cumulative profits across the full day spine
		profits := make([]float64, len(daySlice))
		cumulative := 0.0
		total := 0.0
		for i, d := range daySlice {
			cumulative += dayData[d]
			profits[i] = cumulative
			total += dayData[d]
		}
		series = append(series, dto.ProjectWeeklySeries{
			ProjectID:   key.id,
			ProjectName: key.name,
			TotalProfit: total,
			Data:        profits,
		})
	}

	sort.Slice(series, func(i, j int) bool {
		return series[i].TotalProfit > series[j].TotalProfit
	})

	return &dto.ProjectWeeklyProfitResponse{
		Series: series,
		Days:   daySlice,
	}, nil
}
