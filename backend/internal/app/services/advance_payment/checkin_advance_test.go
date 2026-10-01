package advance_payment

import (
	"testing"
	"time"

	"api-server/internal/domain"
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
			if got := isCheckInRequestWindowOpen(c.now, c.forMonth); got != c.want {
				t.Errorf("isCheckInRequestWindowOpen(%s, %s) = %v, want %v",
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

// TestCheckInAdvanceConstants pins the window-open day so a future edit can't
// silently change the business rule without updating the test.
func TestCheckInAdvanceConstants(t *testing.T) {
	if SelfCheckInAdvanceWindowOpenDay != 10 {
		t.Errorf("SelfCheckInAdvanceWindowOpenDay = %d, want 10", SelfCheckInAdvanceWindowOpenDay)
	}
	if minCheckInAdvanceRequest != 10000 {
		t.Errorf("minCheckInAdvanceRequest = %d, want 10000", minCheckInAdvanceRequest)
	}
	if CheckInAdvanceDisclaimerVN == "" {
		t.Error("CheckInAdvanceDisclaimerVN must not be empty")
	}
}
