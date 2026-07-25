package advance_payment

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

type historyRequestRepo struct {
	domain.AdvancePaymentRequestRepository
	requests []*domain.AdvancePaymentRequest
}

func (f *historyRequestRepo) GetByEmployee(
	_ context.Context,
	_ uint64,
	_, _ int,
	_, _ *time.Time,
	_ *string,
) ([]*domain.AdvancePaymentRequest, int64, error) {
	return f.requests, int64(len(f.requests)), nil
}

func TestGetRequestHistoryIncludesPayrollMonth(t *testing.T) {
	service := NewService(&Config{
		AdvancePaymentRequestRepo: &historyRequestRepo{
			requests: []*domain.AdvancePaymentRequest{
				{
					ID:            10,
					RequestAmount: 1_260_000,
					Status:        domain.AdvancePaymentStatusCompleted,
					AdvancePayment: &domain.AdvancePayment{
						ForMonth: "2026-07",
					},
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	items, _, err := service.GetRequestHistory(context.Background(), 99, 100, 0, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetRequestHistory returned error: %v", err)
	}

	payload, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	if !strings.Contains(string(payload), `"for_month":"2026-07"`) {
		t.Fatalf("history payload must include payroll month, got %s", payload)
	}
}
