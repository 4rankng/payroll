package advance_payment

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

func TestIsRequestWindowLocked(t *testing.T) {
	lockedGap := time.Date(2026, 6, 19, 9, 0, 0, 0, clock.DefaultLocation)
	newPeriodStart := time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation)
	openDay := time.Date(2026, 6, 25, 9, 0, 0, 0, clock.DefaultLocation)

	if !isRequestWindowLocked(lockedGap, false) {
		t.Fatalf("expected day 19 without quota to stay locked")
	}
	if isRequestWindowLocked(lockedGap, true) {
		t.Fatalf("expected day 19 with uploaded quota to unlock requests")
	}
	if !isRequestWindowLocked(newPeriodStart, false) {
		t.Fatalf("expected day 20 without quota to stay locked")
	}
	if !isRequestWindowLocked(openDay, false) {
		t.Fatalf("expected day 25 without quota to stay locked")
	}
	if isRequestWindowLocked(openDay, true) {
		t.Fatalf("expected day 25 with uploaded quota to stay open")
	}
}

func TestIsRequestMonthAllowed(t *testing.T) {
	day5 := time.Date(2026, 6, 5, 9, 0, 0, 0, clock.DefaultLocation)
	day20 := time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation)
	day25 := time.Date(2026, 6, 25, 9, 0, 0, 0, clock.DefaultLocation)

	if !isRequestMonthAllowed(day5, "2026-05") {
		t.Fatalf("expected day 5 to allow previous calendar month only")
	}
	if isRequestMonthAllowed(day5, "2026-06") {
		t.Fatalf("did not expect day 5 to allow current calendar month (tail = previous month only)")
	}
	if isRequestMonthAllowed(day5, "2026-04") {
		t.Fatalf("did not expect day 5 to allow unrelated month")
	}

	if !isRequestMonthAllowed(day20, "2026-06") {
		t.Fatalf("expected day 20 to allow current month when quota exists")
	}
	if isRequestMonthAllowed(day20, "2026-05") {
		t.Fatalf("did not expect day 20 to allow previous month")
	}

	if !isRequestMonthAllowed(day25, "2026-06") {
		t.Fatalf("expected day 25 to allow current month")
	}
	if isRequestMonthAllowed(day25, "2026-05") {
		t.Fatalf("did not expect day 25 to allow previous month")
	}
}

func TestGetEmployeeAdvanceInfoFallsBackToPreviousMonthWhileCurrentUploadMissing(t *testing.T) {
	fakeClock := clock.NewFake(time.Date(2026, 7, 20, 9, 0, 0, 0, clock.DefaultLocation))
	clock.SetGlobal(fakeClock)
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })

	advanceRepo := &employeeInfoAdvanceRepo{
		maxByMonth: map[string]uint64{
			"2026-06": 3_000_000,
		},
		hasDataByMonth: map[string]bool{
			"2026-07": true,
		},
	}
	requestRepo := &employeeInfoAdvanceRequestRepo{
		completedByMonth: map[string]uint64{
			"2026-06": 500_000,
		},
		pendingByMonth: map[string]uint64{
			"2026-06": 200_000,
		},
	}

	svc := NewService(&Config{
		AdvancePaymentRepo:        advanceRepo,
		AdvancePaymentRequestRepo: requestRepo,
		ProjectEmployeeRepo: &employeeInfoProjectEmployeeRepo{
			assignments: []*domain.ProjectEmployee{
				{
					EmployeeID:            99,
					PaymentSchedule:       string(domain.PaymentScheduleFlexible),
					AdvanceRequestEnabled: true,
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	info, err := svc.GetEmployeeAdvanceInfo(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetEmployeeAdvanceInfo returned error: %v", err)
	}

	if got, want := info.TotalMaxAdvance, uint64(3_000_000); got != want {
		t.Fatalf("TotalMaxAdvance = %d, want %d", got, want)
	}
	if got, want := info.CompletedAmount, uint64(500_000); got != want {
		t.Fatalf("CompletedAmount = %d, want %d", got, want)
	}
	if got, want := info.PendingAmount, uint64(200_000); got != want {
		t.Fatalf("PendingAmount = %d, want %d", got, want)
	}
	if got, want := info.RemainingAmount, uint64(2_300_000); got != want {
		t.Fatalf("RemainingAmount = %d, want %d", got, want)
	}
	if len(info.Quotas) != 1 || info.Quotas[0].ForMonth != "2026-06" {
		t.Fatalf("expected one quota for 2026-06, got %#v", info.Quotas)
	}
	if info.CanRequest {
		t.Fatalf("expected request to stay blocked until 2026-07 payroll is uploaded")
	}
	if info.CanRequestReason == "" {
		t.Fatalf("expected waiting-for-upload reason")
	}
}

type employeeInfoAdvanceRepo struct {
	domain.AdvancePaymentRepository
	maxByMonth     map[string]uint64
	hasDataByMonth map[string]bool
}

func (f *employeeInfoAdvanceRepo) SumMaxAdvByEmployeeMonth(_ context.Context, _ uint64, forMonth string) (uint64, error) {
	return f.maxByMonth[forMonth], nil
}

func (f *employeeInfoAdvanceRepo) HasDataForMonth(_ context.Context, forMonth string) (bool, error) {
	return f.hasDataByMonth[forMonth], nil
}

type employeeInfoAdvanceRequestRepo struct {
	domain.AdvancePaymentRequestRepository
	completedByMonth map[string]uint64
	pendingByMonth   map[string]uint64
}

func (f *employeeInfoAdvanceRequestRepo) SumCompletedByEmployeeMonth(_ context.Context, _ uint64, forMonth string) (uint64, error) {
	return f.completedByMonth[forMonth], nil
}

func (f *employeeInfoAdvanceRequestRepo) SumPendingByEmployeeMonth(_ context.Context, _ uint64, forMonth string) (uint64, error) {
	return f.pendingByMonth[forMonth], nil
}

type employeeInfoProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
	assignments []*domain.ProjectEmployee
}

func (f *employeeInfoProjectEmployeeRepo) GetByEmployee(_ context.Context, _ uint) ([]*domain.ProjectEmployee, error) {
	return f.assignments, nil
}
