package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// GetCheckInHealth returns the anomaly/throughput counts across the three
// pipeline stages (check-in/out, quota, advance requests) for the admin
// health dashboard. month is "YYYY-MM" (defaults to current month). Results
// are cached for constants.CheckInHealthCacheTTL.
//
// The expensive queries (failed-attempts + attendance health) fan out
// concurrently with a sync.WaitGroup (template: summary.go); the cheaper
// quota/request counts run sequentially after. The first hard error short-circuits.
func (s *Service) GetCheckInHealth(ctx context.Context, month string) (*dto.CheckInHealthResponse, error) {
	s.logger.Info("Getting check-in health", "month", month)

	now := clock.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1) // exclusive upper bound

	// Resolve the month window for quota metrics.
	var monthTime time.Time
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, fmt.Errorf("invalid month format: %w", err)
		}
		monthTime = t
	} else {
		monthTime = now
	}
	monthKey := monthTime.Format("2006-01")
	forMonth := monthKey

	// Try cache first. Include today's date so that "today" tiles are not stale
	// when the month-keyed cache survives past midnight.
	cacheKey := s.CacheService.GenerateDashboardCacheKey("check_in_health", monthKey+":"+todayStart.Format("2006-01-02"))
	var cached dto.CheckInHealthResponse
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		s.logger.Info("Check-in health retrieved from cache", "month", monthKey)
		return &cached, nil
	}

	// Fan out the expensive queries concurrently.
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		qErr error

		failedByCategory []domain.FailedAttemptCategoryCount
		attStats         *domain.AttendanceHealthStats
		failedByType     []domain.FailedAttemptTypeCount
	)

	setErr := func(err error) {
		mu.Lock()
		if qErr == nil {
			qErr = err
		}
		mu.Unlock()
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		cats, err := s.AttendanceFailedAttemptRepo.GetCategoryCounts(ctx, todayStart, todayEnd)
		if err != nil {
			setErr(fmt.Errorf("failed-attempts categories: %w", err))
			return
		}
		mu.Lock()
		failedByCategory = cats
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		stats, err := s.AttendanceRepo.GetHealthStats(ctx, todayStart, todayEnd)
		if err != nil {
			setErr(fmt.Errorf("attendance health stats: %w", err))
			return
		}
		mu.Lock()
		attStats = stats
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		types, err := s.AttendanceFailedAttemptRepo.GetCountByAttemptType(ctx, todayStart, todayEnd)
		if err != nil {
			setErr(fmt.Errorf("failed-attempts by type: %w", err))
			return
		}
		mu.Lock()
		failedByType = types
		mu.Unlock()
	}()
	wg.Wait()

	if qErr != nil {
		return nil, qErr
	}

	// Quota anomaly counts for the month (cheap, run sequentially).
	driftCount, err := s.AdvancePaymentRepo.CountQuotaAnomalies(ctx, forMonth, "drift")
	if err != nil {
		return nil, fmt.Errorf("quota drift count: %w", err)
	}
	missingCount, err := s.AdvancePaymentRepo.CountQuotaAnomalies(ctx, forMonth, "missing")
	if err != nil {
		return nil, fmt.Errorf("quota missing count: %w", err)
	}
	staleCount, err := s.AdvancePaymentRepo.CountQuotaAnomalies(ctx, forMonth, "stale")
	if err != nil {
		return nil, fmt.Errorf("quota stale count: %w", err)
	}

	// Quota earned this month (throughput tile B4).
	quotaSalaryU, quotaMaxAdvU, err := s.AdvancePaymentRepo.SumSalaryAndMaxAdvForMonth(ctx, forMonth)
	if err != nil {
		return nil, fmt.Errorf("quota month sum: %w", err)
	}

	// Advance request throughput + stuck-pending.
	stuckPending, err := s.AdvancePaymentRequestRepo.CountStuckPending(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("stuck pending: %w", err)
	}
	statusCounts, err := s.AdvancePaymentRequestRepo.CountByStatusSince(ctx, todayStart)
	if err != nil {
		return nil, fmt.Errorf("requests by status: %w", err)
	}

	resp := &dto.CheckInHealthResponse{
		FailedAttemptsByCategory:  toDTOCategoryCounts(failedByCategory),
		FailedCheckInToday:        countByType(failedByType, "check_in"),
		FailedCheckOutToday:       countByType(failedByType, "check_out"),
		OpenCheckedIn:             safeStats(attStats).OpenCheckedIn,
		Orphaned:                  safeStats(attStats).Orphaned,
		AutoRejectedToday:         safeStats(attStats).AutoRejected,
		CompletedZeroEarningToday: safeStats(attStats).CompletedZeroEarning,
		SuccessfulCheckoutsToday:  safeStats(attStats).SuccessfulCheckouts,
		QuotaInvariantDrift:       driftCount,
		MissingQuotaRows:          missingCount,
		StaleQuotaAfterDisable:    staleCount,
		QuotaSalaryThisMonth:      int64(quotaSalaryU),
		QuotaMaxAdvThisMonth:      int64(quotaMaxAdvU),
		RequestsStuckPending:      int(stuckPending),
		RequestsFailedToday:       int(statusCounts[domain.AdvancePaymentStatusFailed]),
		RequestsCompletedToday:    int(statusCounts[domain.AdvancePaymentStatusCompleted]),
		RequestsTotalToday:        totalStatusCounts(statusCounts),
	}

	if cacheErr := s.CacheService.Set(ctx, cacheKey, resp, constants.CheckInHealthCacheTTL); cacheErr != nil {
		s.logger.Warn("Failed to cache check-in health", "error", cacheErr)
	}

	s.logger.Info("Check-in health retrieved",
		"open_checked_in", resp.OpenCheckedIn,
		"orphaned", resp.Orphaned,
		"quota_drift", resp.QuotaInvariantDrift,
		"stuck_pending", resp.RequestsStuckPending)

	return resp, nil
}

// GetQuotaAnomalies returns the drill-down rows for a given anomaly type. It is
// the detail view behind each quota tile in the health strip.
func (s *Service) GetQuotaAnomalies(ctx context.Context, forMonth, anomalyType string) ([]dto.QuotaAnomalyRow, error) {
	s.logger.Info("Getting quota anomalies", "for_month", forMonth, "type", anomalyType)

	if forMonth == "" {
		forMonth = clock.Now().Format("2006-01")
	}

	rows, err := s.AdvancePaymentRepo.GetQuotaAnomalies(ctx, forMonth, anomalyType)
	if err != nil {
		return nil, err
	}

	out := make([]dto.QuotaAnomalyRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.QuotaAnomalyRow{
			EmployeeID:   r.EmployeeID,
			ProjectID:    r.ProjectID,
			ForMonth:     r.ForMonth,
			Salary:       int64(r.Salary),
			MaxAdvAmount: int64(r.MaxAdvAmount),
			ExpectedMax:  expectedMaxAdv(int64(r.Salary)),
			Reason:       r.Reason,
		})
	}
	return out, nil
}

// expectedMaxAdv computes floor(salary * 70 / 100) — the invariant value for the
// self-check-in flow. Centralized here so the DTO agrees with the repo's drift
// predicate (advance_payment_repository.go:quotaAnomalySQL).
func expectedMaxAdv(salary int64) int64 {
	return (salary * int64(domain.SelfCheckInAdvanceablePercent)) / 100
}

// toDTOCategoryCounts maps the domain count slice to the DTO slice, returning a
// non-nil empty slice when there are no categories so the JSON field is [] not null.
func toDTOCategoryCounts(in []domain.FailedAttemptCategoryCount) []dto.FailedAttemptCategoryCount {
	out := make([]dto.FailedAttemptCategoryCount, len(in))
	for i, c := range in {
		out[i] = dto.FailedAttemptCategoryCount{Category: c.Category, Count: c.Count}
	}
	return out
}

// safeStats returns stats or a zero-value struct when the goroutine did not populate it
// (e.g. an error path). Prevents nil-pointer access during response assembly.
func safeStats(s *domain.AttendanceHealthStats) *domain.AttendanceHealthStats {
	if s != nil {
		return s
	}
	return &domain.AttendanceHealthStats{}
}

// countByType returns the count for a given attempt_type from the grouped results,
// returning 0 when the type is not present (safe for zero-value initialization).
func countByType(rows []domain.FailedAttemptTypeCount, attemptType string) int {
	for _, r := range rows {
		if r.AttemptType == attemptType {
			return r.Count
		}
	}
	return 0
}

// totalStatusCounts sums all per-status counts into a single today total.
func totalStatusCounts(m map[domain.AdvancePaymentRequestStatus]int64) int {
	var t int
	for _, v := range m {
		t += int(v)
	}
	return t
}
