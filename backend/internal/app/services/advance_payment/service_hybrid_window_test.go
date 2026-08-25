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

// Hybrid employees (self-checkin enabled + prior-month FlexPay quota) may use
// the regular endpoint ONLY for the previous-period tail (days 1-9). The
// current month must go through the self-checkin flow whose window opens on
// day 10 — the regular endpoint must never serve it to them.
func TestCreateRequestHybridWindow(t *testing.T) {
	cases := []struct {
		name     string
		now      time.Time
		forMonth string
		wantErr  bool
	}{
		{"day 5 prev month (tail open)", time.Date(2026, 8, 5, 9, 0, 0, 0, clock.DefaultLocation), "2026-07", false},
		{"day 9 prev month (last tail day)", time.Date(2026, 8, 9, 23, 0, 0, 0, clock.DefaultLocation), "2026-07", false},
		{"day 10 prev month (tail closed)", time.Date(2026, 8, 10, 0, 0, 30, 0, clock.DefaultLocation), "2026-07", true},
		{"day 31 prev month (closes the hole)", time.Date(2026, 8, 31, 9, 0, 0, 0, clock.DefaultLocation), "2026-07", true},
		{"day 5 current month (tail is prev-only)", time.Date(2026, 8, 5, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", true},
		{"day 10 current month (checkin window bypass)", time.Date(2026, 8, 10, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", true},
		{"day 15 current month (checkin window bypass)", time.Date(2026, 8, 15, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", true},
		{"day 25 current month (checkin window bypass)", time.Date(2026, 8, 25, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClock := clock.NewFake(tc.now)
			clock.SetGlobal(fakeClock)
			t.Cleanup(func() { clock.SetGlobal(clock.New()) })

			svc := NewService(&Config{
				AdvancePaymentRepo: &hybridCreateAdvanceRepo{
					byMonth: map[string][]*domain.AdvancePayment{
						"2026-07": {{ID: 1, ProjectID: 5, MaxAdvAmount: 3_000_000}},
						"2026-08": {{ID: 2, ProjectID: 5, MaxAdvAmount: 5_000_000}},
					},
				},
				AdvancePaymentRequestRepo: &hybridCreateRequestRepo{},
				ProjectEmployeeRepo: &hybridCreateProjectEmployeeRepo{
					assignments: []*domain.ProjectEmployee{
						{
							EmployeeID:      99,
							ProjectID:       5,
							PaymentSchedule: string(domain.PaymentScheduleFlexible),
							CheckInEnabled:  true,
						},
					},
				},
				FeeResolver: &hybridStubFeeResolver{},
			}, slog.New(slog.NewTextHandler(io.Discard, nil)))

			_, err := svc.CreateRequest(context.Background(), 99, 500_000, tc.forMonth)
			if tc.wantErr && err == nil {
				t.Fatalf("expected rejection for %s", tc.forMonth)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected acceptance, got: %v", err)
			}
		})
	}
}

// Non-hybrid flexible employees keep the standard three-phase rule.
func TestCreateRequestNonHybridWindowUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		now      time.Time
		forMonth string
		wantErr  bool
	}{
		{"day 5 prev month", time.Date(2026, 8, 5, 9, 0, 0, 0, clock.DefaultLocation), "2026-07", false},
		{"day 10 prev month", time.Date(2026, 8, 10, 9, 0, 0, 0, clock.DefaultLocation), "2026-07", true},
		{"day 10 current month with quota", time.Date(2026, 8, 10, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", false},
		{"day 25 current month", time.Date(2026, 8, 25, 9, 0, 0, 0, clock.DefaultLocation), "2026-08", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClock := clock.NewFake(tc.now)
			clock.SetGlobal(fakeClock)
			t.Cleanup(func() { clock.SetGlobal(clock.New()) })

			svc := NewService(&Config{
				AdvancePaymentRepo: &hybridCreateAdvanceRepo{
					byMonth: map[string][]*domain.AdvancePayment{
						"2026-07": {{ID: 1, ProjectID: 5, MaxAdvAmount: 3_000_000}},
						"2026-08": {{ID: 2, ProjectID: 5, MaxAdvAmount: 5_000_000}},
					},
				},
				AdvancePaymentRequestRepo: &hybridCreateRequestRepo{},
				ProjectEmployeeRepo: &hybridCreateProjectEmployeeRepo{
					assignments: []*domain.ProjectEmployee{
						{
							EmployeeID:      99,
							ProjectID:       5,
							PaymentSchedule: string(domain.PaymentScheduleFlexible),
						},
					},
				},
				FeeResolver: &hybridStubFeeResolver{},
			}, slog.New(slog.NewTextHandler(io.Discard, nil)))

			_, err := svc.CreateRequest(context.Background(), 99, 500_000, tc.forMonth)
			if tc.wantErr && err == nil {
				t.Fatalf("expected rejection for %s", tc.forMonth)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected acceptance, got: %v", err)
			}
		})
	}
}

type hybridCreateAdvanceRepo struct {
	domain.AdvancePaymentRepository
	byMonth map[string][]*domain.AdvancePayment
}

func (f *hybridCreateAdvanceRepo) GetByEmployeeAndMonth(_ context.Context, _ uint64, forMonth string) ([]*domain.AdvancePayment, error) {
	if rows, ok := f.byMonth[forMonth]; ok {
		return rows, nil
	}
	return nil, nil
}

type hybridCreateRequestRepo struct {
	domain.AdvancePaymentRequestRepository
	created []*domain.AdvancePaymentRequest
}

func (f *hybridCreateRequestRepo) CreateWithBudgetCheck(_ context.Context, req *domain.AdvancePaymentRequest, _ uint64, _ string) error {
	f.created = append(f.created, req)
	return nil
}

type hybridCreateProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
	assignments []*domain.ProjectEmployee
}

func (f *hybridCreateProjectEmployeeRepo) GetByEmployee(_ context.Context, _ uint) ([]*domain.ProjectEmployee, error) {
	return f.assignments, nil
}

type hybridStubFeeResolver struct{}

func (s *hybridStubFeeResolver) ResolveFee(_ context.Context, _ uint64, _ time.Time) uint64 {
	return 10_000
}
