package workers

import (
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

func TestRequestBudgetMonth_PrefersRequestAdvancePaymentMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	req := &domain.AdvancePaymentRequest{
		AdvancePayment: &domain.AdvancePayment{ForMonth: "2026-06"},
	}

	got := requestBudgetMonth(req)
	if got != "2026-06" {
		t.Fatalf("expected budget month 2026-06 from linked advance payment, got %s", got)
	}
}

func TestRequestBudgetMonth_FallsBackToCurrentAdvanceMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	got := requestBudgetMonth(&domain.AdvancePaymentRequest{})
	if got != "2026-06" {
		t.Fatalf("expected fallback budget month 2026-06 on 2026-06-20, got %s", got)
	}
}
