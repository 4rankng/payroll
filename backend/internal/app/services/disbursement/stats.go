package disbursement

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
)

// StatsService aggregates provider_transactions rows for the admin
// dashboard. Two group_by modes — error_code (default) and status —
// each with its own response shape (see DTOs below).
//
// The service is read-only over the same repository the use-case
// service writes through. Range bounds are MySQL DATETIME(3) friendly:
// half-open [from, to) so day-rollover edge cases stay deterministic.
type StatsService struct {
	repo            domaintx.WalletPaymentRepository
	errorTranslator infrastructure.ErrorTranslator
}

// NewStatsService wires the stats service over the same repository.
// errorTranslator may be nil — when nil, ViMessage stays empty.
func NewStatsService(repo domaintx.WalletPaymentRepository, errorTranslator infrastructure.ErrorTranslator) *StatsService {
	return &StatsService{repo: repo, errorTranslator: errorTranslator}
}

// ErrorCodeGroup is one row of the error_code stats response. ViMessage
// is the Vietnamese translation produced by the configured provider's
// ErrorTranslator; for the "no error code" bucket (NULL) it stays empty.
type ErrorCodeGroup struct {
	ErrorCode      string     `json:"error_code"`
	Count          int64      `json:"count"`
	ViMessage      string     `json:"vi_message"`
	LastOccurredAt *time.Time `json:"last_occurred_at"`
}

// StatusGroup is one row of the status stats response. TotalFee is a
// plain int64 (always present; fee is NOT NULL DEFAULT 0 since
// migration 049). The legacy total_charged_amount field has been
// dropped — 9pay's IPN amount equals our requested amount, so it
// carried no information beyond requested_amount.
type StatusGroup struct {
	Status               domaintx.State `json:"status"`
	Count                int64          `json:"count"`
	TotalRequestedAmount int64          `json:"total_requested_amount"`
	TotalFee int64 `json:"total_fee"`
}

// ByErrorCode returns a flat list of ErrorCodeGroup sorted by count DESC.
// from / to are date-only strings (YYYY-MM-DD); the service expands them
// into [from 00:00, to+1day 00:00) to capture full days inclusively.
func (s *StatsService) ByErrorCode(ctx context.Context, from, to time.Time) ([]ErrorCodeGroup, error) {
	rows, err := s.repo.StatsByErrorCode(ctx, from, to.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}
	out := make([]ErrorCodeGroup, 0, len(rows))
	for _, r := range rows {
		out = append(out, ErrorCodeGroup{
			ErrorCode:      r.ErrorCode,
			Count:          r.Count,
			ViMessage:      s.translate(r.ErrorCode),
			LastOccurredAt: r.LastOccurredAt,
		})
	}
	return out, nil
}

// translate returns the user-facing Vietnamese message for a raw error
// code via the configured provider's ErrorTranslator, or "" when the
// translator is unwired or the code is empty.
func (s *StatsService) translate(code string) string {
	if code == "" || s.errorTranslator == nil {
		return ""
	}
	return s.errorTranslator.TranslateError(code)
}

// ByStatus returns one row per status with totals. Rows are sorted by
// status alphabetically (completed/failed/pending/reversed).
func (s *StatsService) ByStatus(ctx context.Context, from, to time.Time) ([]StatusGroup, error) {
	rows, err := s.repo.StatsByStatus(ctx, from, to.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}
	out := make([]StatusGroup, 0, len(rows))
	for _, r := range rows {
		out = append(out, StatusGroup{
			Status:               r.Status,
			Count:                r.Count,
			TotalRequestedAmount: r.TotalRequestedAmount,
			TotalFee:             r.TotalFee,
		})
	}
	return out, nil
}

// ParseDateRange parses YYYY-MM-DD strings into time.Time. Returns
// inclusive [from, to] dates — caller is responsible for the
// half-open expansion at the SQL level (handled inside ByErrorCode /
// ByStatus). Empty strings default to "today" for to and "30 days
// ago" for from, matching the admin dashboard's default view.
func ParseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	const layout = "2006-01-02"
	now := clock.NowUTC()
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	from := to.AddDate(0, 0, -30)
	if fromStr != "" {
		t, err := time.Parse(layout, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date %q: %w", fromStr, err)
		}
		from = t
	}
	if toStr != "" {
		t, err := time.Parse(layout, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date %q: %w", toStr, err)
		}
		to = t
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("to (%s) is before from (%s)", to.Format(layout), from.Format(layout))
	}
	return from, to, nil
}
