package dashboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
)

// getWeeklyPayHistory returns weekly payment totals for the last 12 weeks.
// Uses SQL-level aggregation instead of fetching thousands of individual timesheets.
func (s *Service) getWeeklyPayHistory(ctx context.Context, now time.Time) ([]dto.WeeklyPayData, error) {
	weeksBack := 12
	startDate := now.AddDate(0, 0, -weeksBack*7)

	rows, err := s.TimesheetAnalyticsRepo.GetWeeklyPayAggregated(ctx, startDate, now)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weekly pay aggregation: %w", err)
	}

	agg := make(map[string]*dto.WeeklyPayData, len(rows))
	for _, row := range rows {
		avg := 0.0
		if row.PaidEmployees > 0 {
			avg = row.TotalPay / float64(row.PaidEmployees)
		}
		agg[row.WeekStart] = &dto.WeeklyPayData{
			Date:              row.WeekStart,
			TotalWeeklyPay:    row.TotalPay,
			PaidEmployees:     row.PaidEmployees,
			AvgPayPerEmployee: avg,
		}
	}

	result := make([]dto.WeeklyPayData, 0, weeksBack)
	for i := weeksBack - 1; i >= 0; i-- {
		weekKey := mondayOf(now.AddDate(0, 0, -i*7)).Format("2006-01-02")
		if w, ok := agg[weekKey]; ok {
			result = append(result, *w)
		}
	}

	s.logger.Info("Generated weekly pay data", "count", len(result))
	return result, nil
}

// getRevenueHistory returns cumulative revenue/expense/profit for the last 12 weeks.
// Optimized: fetches cumulative baseline before the window in one query,
// then only fetches ledger entries within the 12-week window for weekly deltas.
func (s *Service) getRevenueHistory(ctx context.Context, now time.Time) ([]dto.RevenueData, error) {
	weeksBack := 12
	windowStart := mondayOf(now.AddDate(0, 0, -weeksBack*7))

	var baseRevenue, baseExpense float64

	ledgerRepo, ok := s.LedgerRepo.(*persistence.LedgerEntryRepository)
	if ok {
		cum, err := ledgerRepo.GetCumulativeTotalsBeforeDate(ctx, windowStart)
		if err == nil {
			baseRevenue = cum.Revenue
			baseExpense = cum.Expense
		}
	}

	entries, err := s.getLedgerEntriesForProjection(ctx, windowStart, now)
	if err != nil {
		return nil, err
	}

	weeklyRev := make(map[string]float64)
	weeklyExp := make(map[string]float64)
	for _, le := range entries {
		k := mondayOf(le.Date).Format("2006-01-02")
		switch le.Account {
		case domain.AccountRevenue:
			weeklyRev[k] += float64(le.Credit - le.Debit)
		case domain.AccountExpense:
			weeklyExp[k] += float64(le.Debit - le.Credit)
		}
	}

	weeks := weeksBetween(windowStart, now)

	cumRev := make(map[string]float64, len(weeks))
	cumExp := make(map[string]float64, len(weeks))
	r, e := baseRevenue, baseExpense
	for _, w := range weeks {
		r += weeklyRev[w]
		e += weeklyExp[w]
		cumRev[w] = r
		cumExp[w] = e
	}

	result := make([]dto.RevenueData, 0, weeksBack)
	lastR, lastE := baseRevenue, baseExpense
	for i := weeksBack - 1; i >= 0; i-- {
		k := mondayOf(now.AddDate(0, 0, -i*7)).Format("2006-01-02")
		if v, ok := cumRev[k]; ok {
			lastR = v
		}
		if v, ok := cumExp[k]; ok {
			lastE = v
		}
		result = append(result, dto.RevenueData{
			Date:          k,
			TotalRevenue:  lastR,
			TotalExpenses: lastE,
			Profit:        lastR - lastE,
		})
	}

	s.logger.Info("Generated revenue data", "count", len(result))
	return result, nil
}

// getCapitalHistory returns cumulative capital (equity + loans) for the last 12 weeks.
// Uses the same cumulative-baseline optimization as getRevenueHistory.
func (s *Service) getCapitalHistory(ctx context.Context, now time.Time) ([]dto.CapitalData, error) {
	weeksBack := 12
	windowStart := mondayOf(now.AddDate(0, 0, -weeksBack*7))

	var baseCapital float64

	ledgerRepo, ok := s.LedgerRepo.(*persistence.LedgerEntryRepository)
	if ok {
		cum, err := ledgerRepo.GetCumulativeTotalsBeforeDate(ctx, windowStart)
		if err == nil {
			baseCapital = cum.Equity + cum.Loan
		}
	}

	entries, err := s.getLedgerEntriesForProjection(ctx, windowStart, now)
	if err != nil {
		return nil, err
	}

	weeklyCapital := make(map[string]float64)
	for _, le := range entries {
		if le.Account != domain.AccountEquity && le.Account != domain.AccountLoan {
			continue
		}
		k := mondayOf(le.Date).Format("2006-01-02")
		weeklyCapital[k] += float64(le.Credit - le.Debit)
	}

	weeks := weeksBetween(windowStart, now)
	cumCapital := make(map[string]float64, len(weeks))
	cum := baseCapital
	for _, w := range weeks {
		cum += weeklyCapital[w]
		cumCapital[w] = cum
	}

	result := make([]dto.CapitalData, 0, weeksBack)
	last := baseCapital
	for i := weeksBack - 1; i >= 0; i-- {
		k := mondayOf(now.AddDate(0, 0, -i*7)).Format("2006-01-02")
		if v, ok := cumCapital[k]; ok {
			last = v
		}
		result = append(result, dto.CapitalData{
			Date:         k,
			TotalCapital: last,
		})
	}

	s.logger.Info("Generated capital data", "count", len(result))
	return result, nil
}

// mondayOf returns the Monday of the week containing t.
func mondayOf(t time.Time) time.Time {
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// weeksBetween returns sorted week-key strings (Monday dates) from start to end.
func weeksBetween(start, end time.Time) []string {
	cur := mondayOf(start)
	days := end.Sub(cur).Hours() / 24
	weeks := make([]string, 0, int(days/7)+2)
	for !cur.After(end) {
		weeks = append(weeks, cur.Format("2006-01-02"))
		cur = cur.AddDate(0, 0, 7)
	}
	sort.Strings(weeks)
	return weeks
}

// getLedgerEntriesForProjection fetches lean ledger entries (no relations preloaded).
func (s *Service) getLedgerEntriesForProjection(ctx context.Context, startDate, endDate time.Time) ([]*domain.LedgerEntry, error) {
	repo, ok := s.LedgerRepo.(*persistence.LedgerEntryRepository)
	if !ok {
		return s.LedgerRepo.GetByDateRange(ctx, startDate, endDate)
	}
	entries, err := repo.GetAccountEntriesByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ledger entries: %w", err)
	}
	return entries, nil
}
