package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"math"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence/repositories"
	"api-server/internal/pkg/timeutil"

	"golang.org/x/sync/errgroup"
)

func isActiveByCutoff(lastDate *string) bool {
	if lastDate == nil {
		return false
	}
	cutoff := clock.Now().AddDate(0, 0, -14)
	if d, err := time.Parse("2006-01-02", *lastDate); err == nil {
		return !d.Before(cutoff)
	}
	return false
}

func derefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func toDetailItem(r *repositories.PartnerEmployeeDetailRow, active bool) dto.PartnerEmployeeDetailItem {
	return dto.PartnerEmployeeDetailItem{
		EmployeeID:        r.EmployeeID,
		EmployeeName:      r.EmployeeName,
		Mobile:            r.Mobile,
		CCCD:              r.CCCD,
		ProjectName:       r.ProjectName,
		LastTimesheetDate: derefStr(r.LastTimesheetDate),
		TotalPaidVND:      int64(r.TotalPaid),
		LastPaidVND:       int64(r.LastPaidVND),
		LastPaidDate:      derefStr(r.LastPaidDate),
		IsActive:          active,
	}
}

// GetTopPaidEmployees returns the top N employees by total paid amount.
// If month is provided (YYYY-MM), filters by that month; otherwise returns all-time.
func (s *Service) GetTopPaidEmployees(ctx context.Context, req *dto.TopPaidEmployeesRequest) (*dto.TopPaidEmployeesResponse, error) {
	s.logger.Info("Getting top paid employees", "month", req.Month, "limit", req.Limit)

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var startDate, endDate *time.Time
	period := "all_time"

	if req.Month != "" {
		t, err := time.Parse("2006-01", req.Month)
		if err == nil {
			start := timeutil.StartOfMonth(t)
			end := timeutil.EndOfMonth(t).Add(time.Nanosecond)
			startDate = &start
			endDate = &end
			period = req.Month
		}
	}

	rows, err := s.TimesheetAnalyticsRepo.GetTopPaidEmployees(ctx, startDate, endDate, limit)
	if err != nil {
		s.logger.Error("Failed to get top paid employees", "error", err)
		return nil, err
	}

	cutoff := clock.Now().AddDate(0, 0, -14)
	items := make([]dto.TopPaidEmployeeItem, 0, len(rows))
	for i, row := range rows {
		isActive := false
		if row.LastTimesheetDate != nil {
			if lastDate, err := time.Parse("2006-01-02", *row.LastTimesheetDate); err == nil {
				isActive = lastDate.After(cutoff) || lastDate.Equal(cutoff)
			}
		}
		items = append(items, dto.TopPaidEmployeeItem{
			Rank:              i + 1,
			EmployeeID:        row.EmployeeID,
			EmployeeName:      row.EmployeeName,
			TotalPaidVND:      int64(row.TotalPaid),
			IsActive:          isActive,
			LastTimesheetDate: derefStr(row.LastTimesheetDate),
		})
	}

	return &dto.TopPaidEmployeesResponse{
		Employees: items,
		Period:    period,
	}, nil
}

// GetPartnerDashboard returns the partner dashboard overview for the authenticated partner.
func (s *Service) GetPartnerDashboard(ctx context.Context, partnerID uint, req *dto.PartnerDashboardRequest) (*dto.PartnerDashboardResponse, error) {
	s.logger.Info("Getting partner dashboard", "partner_id", partnerID, "month", req.Month)

	var startDate, endDate *time.Time
	period := "all_time"

	if req.Month != "" {
		t, err := time.Parse("2006-01", req.Month)
		if err == nil {
			start := timeutil.StartOfMonth(t)
			end := timeutil.EndOfMonth(t).Add(time.Nanosecond)
			startDate = &start
			endDate = &end
			period = req.Month
		}
	}

	// Run all 4 independent data fetches concurrently
	var (
		stats        *repositories.PartnerEmployeeStatsRow
		topRows      []repositories.PartnerTopPaidEmployeeRow
		weeklyStats  []repositories.PartnerWeeklyPaidStats
		monthlyStats []repositories.PartnerMonthlyPaidStats
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		stats, err = s.TimesheetAnalyticsRepo.GetPartnerEmployeeStats(gctx, partnerID, startDate, endDate)
		if err != nil {
			s.logger.Error("Failed to get partner employee stats", "error", err)
		}
		return err
	})

	g.Go(func() error {
		var err error
		topRows, err = s.TimesheetAnalyticsRepo.GetPartnerTopPaidEmployees(gctx, partnerID, startDate, endDate, 10)
		if err != nil {
			s.logger.Error("Failed to get partner top paid employees", "error", err)
		}
		return err
	})

	g.Go(func() error {
		var err error
		weeklyStats, err = s.TimesheetAnalyticsRepo.GetPartnerWeeklyPaidStats(gctx, partnerID, 4)
		if err != nil {
			s.logger.Warn("Failed to get partner weekly stats", "error", err)
			return nil // non-critical, don't fail the whole request
		}
		return err
	})

	g.Go(func() error {
		var err error
		monthlyStats, err = s.TimesheetAnalyticsRepo.GetPartnerMonthlyPaidStats(gctx, partnerID, 3)
		if err != nil {
			s.logger.Warn("Failed to get partner monthly stats", "error", err)
			return nil // non-critical, don't fail the whole request
		}
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	cutoff := clock.Now().AddDate(0, 0, -14)
	topItems := make([]dto.TopPaidEmployeeItem, 0, len(topRows))
	for i, row := range topRows {
		isActive := false
		if row.LastTimesheetDate != nil {
			if lastDate, err := time.Parse("2006-01-02", *row.LastTimesheetDate); err == nil {
				isActive = lastDate.After(cutoff) || lastDate.Equal(cutoff)
			}
		}
		topItems = append(topItems, dto.TopPaidEmployeeItem{
			Rank:              i + 1,
			EmployeeID:        row.EmployeeID,
			EmployeeName:      row.EmployeeName,
			TotalPaidVND:      int64(row.TotalPaid),
			IsActive:          isActive,
			LastTimesheetDate: derefStr(row.LastTimesheetDate),
		})
	}

	wowPaidEmployees := dto.PartnerDashboardWoWChange{}
	wowPaidAmount := dto.PartnerDashboardWoWChange{}
	if len(weeklyStats) >= 2 {
		cur := weeklyStats[len(weeklyStats)-1]
		prev := weeklyStats[len(weeklyStats)-2]
		wowPaidEmployees.CurrentWeek = int64(cur.PaidEmployees)
		wowPaidEmployees.PreviousWeek = int64(prev.PaidEmployees)
		wowPaidEmployees.ChangeAmount = int64(cur.PaidEmployees) - int64(prev.PaidEmployees)
		if prev.PaidEmployees > 0 {
			wowPaidEmployees.ChangePct = math.Round((float64(cur.PaidEmployees-prev.PaidEmployees)/float64(prev.PaidEmployees))*10000) / 100
		}
		wowPaidAmount.CurrentWeek = int64(cur.TotalPaid)
		wowPaidAmount.PreviousWeek = int64(prev.TotalPaid)
		wowPaidAmount.ChangeAmount = int64(cur.TotalPaid) - int64(prev.TotalPaid)
		if prev.TotalPaid > 0 {
			wowPaidAmount.ChangePct = math.Round(((cur.TotalPaid-prev.TotalPaid)/prev.TotalPaid)*10000) / 100
		}
	} else if len(weeklyStats) == 1 {
		cur := weeklyStats[0]
		wowPaidEmployees.CurrentWeek = int64(cur.PaidEmployees)
		wowPaidAmount.CurrentWeek = int64(cur.TotalPaid)
	}

	momPaidEmployees := dto.PartnerDashboardMoMChange{}
	momPaidAmount := dto.PartnerDashboardMoMChange{}
	if len(monthlyStats) >= 2 {
		cur := monthlyStats[len(monthlyStats)-1]
		prev := monthlyStats[len(monthlyStats)-2]
		momPaidEmployees.CurrentMonth = int64(cur.PaidEmployees)
		momPaidEmployees.PreviousMonth = int64(prev.PaidEmployees)
		momPaidEmployees.ChangeAmount = int64(cur.PaidEmployees) - int64(prev.PaidEmployees)
		if prev.PaidEmployees > 0 {
			momPaidEmployees.ChangePct = math.Round((float64(cur.PaidEmployees-prev.PaidEmployees)/float64(prev.PaidEmployees))*10000) / 100
		}
		momPaidAmount.CurrentMonth = int64(cur.TotalPaid)
		momPaidAmount.PreviousMonth = int64(prev.TotalPaid)
		momPaidAmount.ChangeAmount = int64(cur.TotalPaid) - int64(prev.TotalPaid)
		if prev.TotalPaid > 0 {
			momPaidAmount.ChangePct = math.Round(((cur.TotalPaid-prev.TotalPaid)/prev.TotalPaid)*10000) / 100
		}
	} else if len(monthlyStats) == 1 {
		cur := monthlyStats[0]
		momPaidEmployees.CurrentMonth = int64(cur.PaidEmployees)
		momPaidAmount.CurrentMonth = int64(cur.TotalPaid)
	}

	return &dto.PartnerDashboardResponse{
		ActiveEmployees:  stats.ActiveEmployees,
		DroppedEmployees: stats.DroppedEmployees,
		PaidEmployees:    stats.PaidEmployees,
		TotalPaidVND:     int64(stats.TotalPaidAmount),
		WoWPaidEmployees: wowPaidEmployees,
		WoWPaidAmount:    wowPaidAmount,
		MoMPaidEmployees: momPaidEmployees,
		MoMPaidAmount:    momPaidAmount,
		TopPaidEmployees: topItems,
		Period:           period,
	}, nil
}

// GetPartnerEmployeeList returns the employee list for a given category (active/dropped/paid).
func (s *Service) GetPartnerEmployeeList(ctx context.Context, partnerID uint, req *dto.PartnerEmployeeListRequest) (*dto.PartnerEmployeeListResponse, error) {
	s.logger.Info("Getting partner employee list", "partner_id", partnerID, "type", req.Type)

	switch req.Type {
	case "active":
		rows, err := s.TimesheetAnalyticsRepo.GetPartnerActiveEmployees(ctx, partnerID)
		if err != nil {
			return nil, err
		}
		items := make([]dto.PartnerEmployeeDetailItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, toDetailItem(&r, isActiveByCutoff(r.LastTimesheetDate)))
		}
		return &dto.PartnerEmployeeListResponse{Employees: items, Type: "active", Total: len(items)}, nil

	case "dropped":
		rows, err := s.TimesheetAnalyticsRepo.GetPartnerDroppedEmployees(ctx, partnerID)
		if err != nil {
			return nil, err
		}
		items := make([]dto.PartnerEmployeeDetailItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, toDetailItem(&r, false))
		}
		return &dto.PartnerEmployeeListResponse{Employees: items, Type: "dropped", Total: len(items)}, nil

	case "paid":
		var startDate, endDate *time.Time
		if req.Month != "" {
			if t, err := time.Parse("2006-01", req.Month); err == nil {
				start := timeutil.StartOfMonth(t)
				end := timeutil.EndOfMonth(t).Add(time.Nanosecond)
				startDate = &start
				endDate = &end
			}
		}
		rows, err := s.TimesheetAnalyticsRepo.GetPartnerPaidEmployees(ctx, partnerID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		items := make([]dto.PartnerEmployeeDetailItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, toDetailItem(&r, isActiveByCutoff(r.LastTimesheetDate)))
		}
		return &dto.PartnerEmployeeListResponse{Employees: items, Type: "paid", Total: len(items)}, nil

	default:
		return nil, domain.NewValidationError("type phải là active, dropped hoặc paid")
	}
}
