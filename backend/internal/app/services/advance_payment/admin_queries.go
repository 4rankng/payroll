package advance_payment

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

func (s *Service) ListRequests(ctx context.Context, filters domain.AdvancePaymentRequestFilters) ([]*domain.AdvancePaymentRequest, int64, error) {
	return s.config.AdvancePaymentRequestRepo.List(ctx, filters)
}

func (s *Service) GetPendingGroupedForExport(ctx context.Context, forMonth string) ([]*domain.EmployeePendingRequests, error) {
	return s.config.AdvancePaymentRequestRepo.GetPendingGroupedByEmployee(ctx, forMonth)
}

func (s *Service) GetSummary(ctx context.Context, fromDate, toDate time.Time, forMonth string) (*dto.AdvancePaymentSummaryResponse, error) {
	stats, err := s.config.AdvancePaymentRequestRepo.GetStatsSummary(ctx, fromDate, toDate, forMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stats summary")
	}

	allTimeProviderFee, err := s.config.AdvancePaymentRequestRepo.GetTotalProviderFee(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total provider fee")
	}

	allTimeFeeEarned, err := s.config.AdvancePaymentRequestRepo.GetTotalFeeEarned(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total fee earned")
	}

	resp := &dto.AdvancePaymentSummaryResponse{
		TotalRequests:           stats.TotalRequests,
		TotalPending:            stats.TotalPending,
		TotalApproved:           stats.TotalApproved,
		TotalCancelled:          stats.TotalCancelled,
		TotalFailed:             stats.TotalFailed,
		TotalPaid:               stats.TotalPaid,
		TotalAmount:             stats.TotalAmount,
		TotalPaidAmount:         stats.TotalPaidAmount,
		TotalPendingAmount:      stats.TotalPendingAmount,
		TotalFailedAmount:       stats.TotalFailedAmount,
		TotalCancelledAmount:    stats.TotalCancelledAmount,
		TotalFee:                stats.TotalFee,
		TotalFeeEarned:          stats.TotalFeeEarned,
		TotalFeeEarnedAllTime:   allTimeFeeEarned,
		TotalNet:                stats.TotalAmount - stats.TotalFee,
		TotalProviderFee:        stats.TotalProviderFee,
		TotalProviderFeeAllTime: allTimeProviderFee,
		AvgProcessingTimeSecs:   stats.AvgProcessingTimeSecs,
		CompletedUnder30s:       stats.CompletedUnder30s,
	}

	// Derived metrics — computed server-side, no client-side calculation needed
	if stats.TotalPaidAmount > 0 {
		resp.FeePercentage = float64(stats.TotalFeeEarned) / float64(stats.TotalPaidAmount) * 100
	}
	if stats.TotalPaid > 0 {
		resp.AvgFeePerRequest = stats.TotalFeeEarned / uint64(stats.TotalPaid)
	}
	if stats.TotalRequests > 0 {
		resp.AvgFeePerEmployee = stats.TotalFeeEarned / uint64(stats.TotalRequests)
	}
	effectiveTotal := stats.TotalRequests - stats.TotalCancelled
	if effectiveTotal > 0 {
		resp.SuccessRate = float64(stats.TotalPaid) / float64(effectiveTotal) * 100
	}
	if stats.TotalAmount > 0 {
		resp.DisbursementPercentage = float64(stats.TotalPaidAmount) / float64(stats.TotalAmount) * 100
	}

	if !fromDate.IsZero() {
		fromDateStr := fromDate.Format("2006-01-02")
		resp.FromDate = &fromDateStr
	}
	if !toDate.IsZero() {
		toDateStr := toDate.Format("2006-01-02")
		resp.ToDate = &toDateStr
	}

	return resp, nil
}

func (s *Service) GetEmployeeAdvanceStats(ctx context.Context, filters domain.EmployeeAdvanceStatsFilters) ([]*domain.EmployeeAdvanceStats, string, int64, error) {
	if filters.ForMonth == nil {
		latestMonth, err := s.config.AdvancePaymentRepo.GetLatestForMonth(ctx)
		if err != nil {
			return nil, "", 0, errors.Wrap(err, "failed to get latest for_month")
		}
		filters.ForMonth = &latestMonth
	}

	if filters.Limit <= 0 {
		filters.Limit = 20
	}

	if filters.SortBy == "" {
		filters.SortBy = "e.fullname"
	}

	if filters.SortOrder == "" {
		filters.SortOrder = "ASC"
	}

	employees, total, err := s.config.AdvancePaymentRepo.GetEmployeeAdvanceStats(ctx, filters)
	if err != nil {
		return nil, "", 0, errors.Wrap(err, "failed to get employee advance stats")
	}

	s.logger.Info("retrieved employee advance stats",
		"for_month", *filters.ForMonth,
		"total", total,
		"search", filters.Search,
	)

	return employees, *filters.ForMonth, total, nil
}

func (s *Service) GetAvailableMonths(ctx context.Context) ([]*domain.AvailableMonth, error) {
	months, err := s.config.AdvancePaymentRepo.GetAvailableMonths(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available months")
	}

	s.logger.Info("retrieved available months", "count", len(months))
	return months, nil
}
