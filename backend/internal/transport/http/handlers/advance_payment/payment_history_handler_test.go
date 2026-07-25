package advance_payment

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	advancepaymentservice "api-server/internal/app/services/advance_payment"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

type historyEmployeeRepo struct {
	domain.EmployeeRepository
	employee *domain.Employee
}

func (f *historyEmployeeRepo) GetByUserID(context.Context, uint) (*domain.Employee, error) {
	return f.employee, nil
}

type historyHandlerRequestRepo struct {
	domain.AdvancePaymentRequestRepository
	requests []*domain.AdvancePaymentRequest
}

func (f *historyHandlerRequestRepo) GetByEmployee(
	context.Context,
	uint64,
	int,
	int,
	*time.Time,
	*time.Time,
	*string,
) ([]*domain.AdvancePaymentRequest, int64, error) {
	return f.requests, int64(len(f.requests)), nil
}

func TestGetMyAdvancePaymentHistoryIncludesForMonth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := advancepaymentservice.NewService(&advancepaymentservice.Config{
		EmployeeRepo: &historyEmployeeRepo{employee: &domain.Employee{ID: 740}},
		AdvancePaymentRequestRepo: &historyHandlerRequestRepo{
			requests: []*domain.AdvancePaymentRequest{
				{
					ID:            101,
					RequestAmount: 1_260_000,
					Status:        domain.AdvancePaymentStatusCompleted,
					AdvancePayment: &domain.AdvancePayment{
						ForMonth: "2026-07",
					},
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := &AdvancePaymentHandler{service: service}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/me/advance-payment/history?page=1&pageSize=100", nil)
	ctx.Set("user_id", uint(99))

	handler.GetMyAdvancePaymentHistory(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data []struct {
			ForMonth string `json:"forMonth"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ForMonth != "2026-07" {
		t.Fatalf("history response must expose forMonth=2026-07, got %s", recorder.Body.String())
	}
}
