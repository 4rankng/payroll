package advance_payment

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// TestIsCheckInRequestWindowOpen covers the self-check-in advance request
// window. A check-in period must have exactly ONE request path, so the flow
// mirrors the regular endpoint's calendar: the previous month during the
// days 1-8 tail, the current month from day 10, and nothing at all on day 9.
// Financial correctness depends on this.
func TestIsCheckInRequestWindowOpen(t *testing.T) {
	loc := time.FixedZone("ICT", 7*3600) // Asia/Ho_Chi_Minh = UTC+7, no tz DB needed
	cases := []struct {
		name     string
		now      time.Time
		forMonth string
		want     bool
	}{
		{"day 1 requesting april tail - open", time.Date(2026, 5, 1, 0, 0, 0, 0, loc), "2026-04", true},
		{"day 8 requesting april tail - open", time.Date(2026, 5, 8, 23, 59, 0, 0, loc), "2026-04", true},
		{"day 5 same month - locked", time.Date(2026, 5, 5, 12, 0, 0, 0, loc), "2026-05", false},
		{"day 9 gap day - locked", time.Date(2026, 5, 9, 23, 59, 0, 0, loc), "2026-04", false},
		{"day 9 same month - locked", time.Date(2026, 5, 9, 23, 59, 0, 0, loc), "2026-05", false},
		{"day 10 same month - open", time.Date(2026, 5, 10, 0, 1, 0, 0, loc), "2026-05", true},
		{"day 15 same month - open", time.Date(2026, 5, 15, 9, 0, 0, 0, loc), "2026-05", true},
		{"day 31 same month - open", time.Date(2026, 5, 31, 23, 59, 0, 0, loc), "2026-05", true},
		{"jun 15 requesting jun - open", time.Date(2026, 6, 15, 12, 0, 0, 0, loc), "2026-06", true},
		{"jun 5 requesting jun - before day 10", time.Date(2026, 6, 5, 12, 0, 0, 0, loc), "2026-06", false},
		{"may 15 requesting jun - future month", time.Date(2026, 5, 15, 12, 0, 0, 0, loc), "2026-06", false},
		{"jun 15 requesting may - period already rolled", time.Date(2026, 6, 15, 12, 0, 0, 0, loc), "2026-05", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isCheckInRequestWindowOpen(c.now, c.forMonth, SelfCheckInAdvanceWindowOpenDay); got != c.want {
				t.Errorf("isCheckInRequestWindowOpen(%s, %s) = %v, want %v",
					c.now.Format("2006-01-02"), c.forMonth, got, c.want)
			}
		})
	}
}

// With a configured window-open day of 15, the default day-10 expectations no
// longer apply: day 14 stays locked, day 15 opens.
func TestIsCheckInRequestWindowOpenConfiguredDay(t *testing.T) {
	loc := time.FixedZone("ICT", 7*3600)
	cases := []struct {
		name     string
		now      time.Time
		forMonth string
		want     bool
	}{
		{"day 14 same month - locked", time.Date(2026, 5, 14, 23, 59, 0, 0, loc), "2026-05", false},
		{"day 15 same month - open", time.Date(2026, 5, 15, 0, 1, 0, 0, loc), "2026-05", true},
		{"day 10 same month - locked before configured day", time.Date(2026, 5, 10, 12, 0, 0, 0, loc), "2026-05", false},
		{"day 28 same month - open", time.Date(2026, 5, 28, 12, 0, 0, 0, loc), "2026-05", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isCheckInRequestWindowOpen(c.now, c.forMonth, 15); got != c.want {
				t.Errorf("isCheckInRequestWindowOpen(%s, %s, 15) = %v, want %v",
					c.now.Format("2006-01-02"), c.forMonth, got, c.want)
			}
		})
	}
}

// TestDefaultSelfCheckInAdvancePercentage verifies the fallback percentage and
// integer advanceable formula (salary * percent / 100). Floor semantics, no
// float drift.
func TestDefaultSelfCheckInAdvancePercentage(t *testing.T) {
	if domain.DefaultSelfCheckInAdvancePercentage != 70 {
		t.Fatalf("DefaultSelfCheckInAdvancePercentage = %d, want 70", domain.DefaultSelfCheckInAdvancePercentage)
	}
	cases := []struct {
		salary uint64
		want   uint64
	}{
		{0, 0},
		{100, 70},
		{105, 73},      // floor(73.5) = 73
		{142, 99},      // floor(99.4) = 99
		{99999, 69999}, // floor(69999.3) = 69999
		{1000000, 700000},
		{1234567, 864196}, // floor(864196.9) = 864196
	}
	for _, c := range cases {
		got := (c.salary * domain.DefaultSelfCheckInAdvancePercentage) / 100
		if got != c.want {
			t.Errorf("advanceable(salary=%d) = %d, want %d", c.salary, got, c.want)
		}
	}
}

// TestCheckInAdvanceConstants pins the fallback window-open day so a future
// edit can't silently change the business rule without updating the test.
func TestCheckInAdvanceConstants(t *testing.T) {
	if domain.DefaultSelfCheckInAdvanceWindowOpenDay != 10 {
		t.Errorf("DefaultSelfCheckInAdvanceWindowOpenDay = %d, want 10", domain.DefaultSelfCheckInAdvanceWindowOpenDay)
	}
	if SelfCheckInAdvanceWindowOpenDay != int(domain.DefaultSelfCheckInAdvanceWindowOpenDay) {
		t.Errorf("SelfCheckInAdvanceWindowOpenDay = %d, want int(domain.DefaultSelfCheckInAdvanceWindowOpenDay)", SelfCheckInAdvanceWindowOpenDay)
	}
	if minCheckInAdvanceRequest != 10000 {
		t.Errorf("minCheckInAdvanceRequest = %d, want 10000", minCheckInAdvanceRequest)
	}
	if CheckInAdvanceDisclaimerVN == "" {
		t.Error("CheckInAdvanceDisclaimerVN must not be empty")
	}
}

// windowDayAdvanceRepo serves the Sum* aggregates GetCheckInAdvanceInfo needs.
type windowDayAdvanceRepo struct {
	domain.AdvancePaymentRepository
}

func (f *windowDayAdvanceRepo) SumSalaryAndMaxAdvByEmployeeMonth(_ context.Context, _ uint64, _ string) (uint64, uint64, error) {
	return 5_000_000, 3_500_000, nil
}

func (f *windowDayAdvanceRepo) SumPendingEarningsByEmployeeMonth(_ context.Context, _ uint64, _ string) (uint64, error) {
	return 0, nil
}

// windowDayRequestRepo serves the completed/pending sums GetCheckInAdvanceInfo needs.
type windowDayRequestRepo struct {
	domain.AdvancePaymentRequestRepository
}

func (f *windowDayRequestRepo) SumCompletedByEmployeeMonth(_ context.Context, _ uint64, _ string) (uint64, error) {
	return 0, nil
}

func (f *windowDayRequestRepo) SumPendingByEmployeeMonth(_ context.Context, _ uint64, _ string) (uint64, error) {
	return 0, nil
}

// TestGetCheckInAdvanceInfoConfiguredWindowDay proves the employee-facing info
// follows the Admin-configured window-open day: a service wired with a getter
// returning 15 reports WindowOpenDay 15 (day 12 is before the window → closed,
// reason names "ngày 15"); a service with the getter nil falls back to 10.
func TestGetCheckInAdvanceInfoConfiguredWindowDay(t *testing.T) {
	loc := time.FixedZone("ICT", 7*3600)
	// Day 12: past the days 1-8 tail, before a day-15 window.
	clock.SetGlobal(clock.NewFake(time.Date(2026, 5, 12, 9, 0, 0, 0, loc)))
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })

	newSvc := func(getter func(ctx context.Context) uint64) *Service {
		return NewService(&Config{
			AdvancePaymentRepo:        &windowDayAdvanceRepo{},
			AdvancePaymentRequestRepo: &windowDayRequestRepo{},
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
			GetSelfCheckInAdvanceWindowOpenDay: getter,
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	t.Run("configured day 15 governs the served info", func(t *testing.T) {
		info, err := newSvc(func(_ context.Context) uint64 { return 15 }).GetCheckInAdvanceInfo(context.Background(), 99)
		if err != nil {
			t.Fatalf("GetCheckInAdvanceInfo: %v", err)
		}
		if info.WindowOpenDay != 15 {
			t.Errorf("WindowOpenDay = %d, want 15", info.WindowOpenDay)
		}
		if info.ForMonth != "2026-05" {
			t.Errorf("ForMonth = %s, want 2026-05", info.ForMonth)
		}
		if info.CanRequest {
			t.Error("day 12 must be before a day-15 window")
		}
		if !strings.Contains(info.CanRequestReason, "ngày 15") {
			t.Errorf("CanRequestReason = %q, want it to name ngày 15", info.CanRequestReason)
		}
	})

	t.Run("nil getter falls back to default day 10", func(t *testing.T) {
		// Day 9 (the gap day) sits before the default day-10 window, so the
		// closed-window reason must name the fallback day.
		clock.SetGlobal(clock.NewFake(time.Date(2026, 5, 9, 12, 0, 0, 0, loc)))
		info, err := newSvc(nil).GetCheckInAdvanceInfo(context.Background(), 99)
		if err != nil {
			t.Fatalf("GetCheckInAdvanceInfo: %v", err)
		}
		if info.WindowOpenDay != 10 {
			t.Errorf("WindowOpenDay = %d, want 10", info.WindowOpenDay)
		}
		if !strings.Contains(info.CanRequestReason, "ngày 10") {
			t.Errorf("CanRequestReason = %q, want it to name ngày 10", info.CanRequestReason)
		}
	})
}
