package advance_payment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// Quota provenance must not gate an employee whose self check-in service is
// OFF: /me/check-in-advance refuses them ("chưa được cấp quyền"), so the
// regular endpoint is the only request path they have. Whatever funded
// max_adv_amount, remaining budget is the gate. The check-in-only lock stays
// for employees with the service on — the check-in flow owns those periods —
// and the mixed-source refusal stays for everyone (period_source_test.go).
func TestCreateRequestServesCheckInFundedQuotaWhenServiceOff(t *testing.T) {
	setQuotaSourceClock(t, time.Date(2026, 11, 5, 9, 0, 0, 0, clock.DefaultLocation))

	requestRepo := &hybridCreateRequestRepo{}
	svc := NewService(&Config{
		AdvancePaymentRepo: &hybridCreateAdvanceRepo{
			byMonth: map[string][]*domain.AdvancePayment{
				// October quota banked from check-in/out earnings: Salary > 0
				// while the employee was on self check-in (since disabled).
				"2026-10": {{ID: 1, ProjectID: 5, MaxAdvAmount: 9_721_000, Salary: 13_887_142}},
			},
		},
		AdvancePaymentRequestRepo: requestRepo,
		ProjectEmployeeRepo: &hybridCreateProjectEmployeeRepo{
			assignments: []*domain.ProjectEmployee{
				{
					EmployeeID:            99,
					ProjectID:             5,
					PaymentSchedule:       string(domain.PaymentScheduleFlexible),
					AdvanceRequestEnabled: true, // CheckInEnabled stays false
				},
			},
		},
		FeeResolver: &hybridStubFeeResolver{},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := svc.CreateRequest(context.Background(), 99, 5_185_000, "2026-10"); err != nil {
		t.Fatalf("expected check-in-funded quota to back a regular request, got: %v", err)
	}
	if len(requestRepo.created) != 1 {
		t.Fatalf("expected exactly one request created, got %d", len(requestRepo.created))
	}
}

// Employees with the service still on keep the routing: the self-check-in
// flow owns check-in periods, the regular endpoint must keep refusing them.
func TestCreateRequestStillGatesCheckInPeriodForCheckInEmployee(t *testing.T) {
	setQuotaSourceClock(t, time.Date(2026, 11, 5, 9, 0, 0, 0, clock.DefaultLocation))

	svc := NewService(&Config{
		AdvancePaymentRepo: &hybridCreateAdvanceRepo{
			byMonth: map[string][]*domain.AdvancePayment{
				"2026-10": {{ID: 1, ProjectID: 5, MaxAdvAmount: 9_721_000, Salary: 13_887_142}},
			},
		},
		AdvancePaymentRequestRepo: &hybridCreateRequestRepo{},
		ProjectEmployeeRepo: &hybridCreateProjectEmployeeRepo{
			assignments: []*domain.ProjectEmployee{
				{
					EmployeeID:            99,
					ProjectID:             5,
					PaymentSchedule:       string(domain.PaymentScheduleFlexible),
					CheckInEnabled:        true,
					AdvanceRequestEnabled: true,
				},
			},
		},
		FeeResolver: &hybridStubFeeResolver{},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := svc.CreateRequest(context.Background(), 99, 500_000, "2026-10")
	if err == nil {
		t.Fatal("expected the check-in-only rejection for an employee with the service on")
	}
	want := fmt.Sprintf(constants.MsgAdvancePeriodCheckInOnlyVN, FormatMonthDisplay("2026-10"))
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestGetEmployeeAdvanceInfoServesCheckInFundedQuotaWhenServiceOff(t *testing.T) {
	setQuotaSourceClock(t, time.Date(2026, 11, 5, 9, 0, 0, 0, clock.DefaultLocation))

	svc := NewService(&Config{
		AdvancePaymentRepo: &quotaSourceAdvanceRepo{
			months:     []string{"2026-10"},
			maxByMonth: map[string]uint64{"2026-10": 9_721_000},
			rowsByMonth: map[string][]*domain.AdvancePayment{
				"2026-10": {{ForMonth: "2026-10", MaxAdvAmount: 9_721_000, Salary: 13_887_142}},
			},
		},
		AdvancePaymentRequestRepo: &employeeInfoAdvanceRequestRepo{
			completedByMonth: map[string]uint64{"2026-10": 4_535_000},
		},
		ProjectEmployeeRepo: &employeeInfoProjectEmployeeRepo{
			assignments: []*domain.ProjectEmployee{
				{
					EmployeeID:            99,
					PaymentSchedule:       string(domain.PaymentScheduleFlexible),
					AdvanceRequestEnabled: true, // CheckInEnabled stays false
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	info, err := svc.GetEmployeeAdvanceInfo(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetEmployeeAdvanceInfo returned error: %v", err)
	}
	if !info.CanRequest {
		t.Fatalf("expected CanRequest=true against check-in-funded quota, got title=%q reason=%q", info.CanRequestTitle, info.CanRequestReason)
	}
	if got, want := info.RemainingAmount, uint64(5_186_000); got != want {
		t.Fatalf("RemainingAmount = %d, want %d", got, want)
	}
}

func TestGetEmployeeAdvanceInfoStillLocksCheckInPeriodForCheckInEmployee(t *testing.T) {
	setQuotaSourceClock(t, time.Date(2026, 11, 5, 9, 0, 0, 0, clock.DefaultLocation))

	svc := NewService(&Config{
		AdvancePaymentRepo: &quotaSourceAdvanceRepo{
			months:     []string{"2026-10"},
			maxByMonth: map[string]uint64{"2026-10": 9_721_000},
			rowsByMonth: map[string][]*domain.AdvancePayment{
				"2026-10": {{ForMonth: "2026-10", MaxAdvAmount: 9_721_000, Salary: 13_887_142}},
			},
		},
		AdvancePaymentRequestRepo: &employeeInfoAdvanceRequestRepo{
			completedByMonth: map[string]uint64{"2026-10": 4_535_000},
		},
		ProjectEmployeeRepo: &employeeInfoProjectEmployeeRepo{
			assignments: []*domain.ProjectEmployee{
				{
					EmployeeID:            99,
					PaymentSchedule:       string(domain.PaymentScheduleFlexible),
					CheckInEnabled:        true,
					AdvanceRequestEnabled: true,
				},
			},
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	info, err := svc.GetEmployeeAdvanceInfo(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetEmployeeAdvanceInfo returned error: %v", err)
	}
	if info.CanRequest {
		t.Fatal("expected CanRequest=false for an employee with the service on")
	}
	if info.CanRequestTitle != constants.MsgAdvanceCheckInOnlyTitleVN {
		t.Fatalf("CanRequestTitle = %q, want %q", info.CanRequestTitle, constants.MsgAdvanceCheckInOnlyTitleVN)
	}
	wantReason := fmt.Sprintf(constants.MsgAdvancePeriodCheckInOnlyVN, FormatMonthDisplay("2026-10"))
	if info.CanRequestReason != wantReason {
		t.Fatalf("CanRequestReason = %q, want %q", info.CanRequestReason, wantReason)
	}
}

func setQuotaSourceClock(t *testing.T, now time.Time) {
	t.Helper()
	clock.SetGlobal(clock.NewFake(now))
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })
}

// quotaSourceAdvanceRepo backs the info path with rows whose provenance is
// check-in earnings (Salary > 0, no workbook stamp).
type quotaSourceAdvanceRepo struct {
	domain.AdvancePaymentRepository
	months      []string
	maxByMonth  map[string]uint64
	rowsByMonth map[string][]*domain.AdvancePayment
}

func (f *quotaSourceAdvanceRepo) SumMaxAdvByEmployeeMonth(_ context.Context, _ uint64, forMonth string) (uint64, error) {
	return f.maxByMonth[forMonth], nil
}

func (f *quotaSourceAdvanceRepo) GetByEmployeeAndMonth(_ context.Context, _ uint64, forMonth string) ([]*domain.AdvancePayment, error) {
	return f.rowsByMonth[forMonth], nil
}

func (f *quotaSourceAdvanceRepo) GetMonthsByEmployee(_ context.Context, _ uint64) ([]string, error) {
	return f.months, nil
}
