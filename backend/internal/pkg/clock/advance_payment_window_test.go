package clock

import (
	"testing"
	"time"
)

// TestPeriodCycleStartDay guards the period rollover: the advance-payment
// period rolls on day 20. The request tail length is Admin-configurable
// (advance_request_stop_day, default 9), so it is no longer a constant here.
func TestPeriodCycleStartDay(t *testing.T) {
	if PeriodCycleStartDay != 20 {
		t.Fatalf("PeriodCycleStartDay = %d, want 20", PeriodCycleStartDay)
	}
}

// TestRequestWindowBoundaries exercises the three-phase window around the
// default day-9 stop day using Asia/Ho_Chi_Minh local dates (the timezone
// the business runs in). July 2026 anchors the phases relative to calendar
// July:
//
//	days 1–8  -> tail of the previous period (for_month = June)
//	days 9–19 -> locked gap
//	days 20+  → new period opens (for_month rolls to July/August)
func TestRequestWindowBoundaries(t *testing.T) {
	vn, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}

	type expect struct {
		day          int
		beforeCutoff bool
		inLockedGap  bool
	}
	cases := []expect{
		{8, true, false},   // final tail day, still previous period
		{9, false, true},   // stop day, requests locked
		{10, false, true},  // locked gap
		{15, false, true},  // mid locked gap
		{19, false, true},  // last locked-gap day
		{20, false, false}, // new period opens
	}

	for _, tc := range cases {
		now := time.Date(2026, time.July, tc.day, 12, 0, 0, 0, vn)
		if got := IsBeforeCutoff(now, 9); got != tc.beforeCutoff {
			t.Errorf("day %d: IsBeforeCutoff(now, 9) = %v, want %v", tc.day, got, tc.beforeCutoff)
		}
		if got := IsInLockedGap(now, 9); got != tc.inLockedGap {
			t.Errorf("day %d: IsInLockedGap(now, 9) = %v, want %v", tc.day, got, tc.inLockedGap)
		}
	}
}

// A configured stop day shifts the boundary: with stop day 12, day 10 stays
// requestable and day 12 is the first locked day while the gap still ends
// at day 20.
func TestRequestWindowBoundariesConfiguredStopDay(t *testing.T) {
	vn, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}

	cases := []struct {
		day          int
		beforeCutoff bool
		inLockedGap  bool
	}{
		{10, true, false},  // inside a stop day 12 tail
		{11, true, false},  // last tail day under stop day 12
		{12, false, true},  // stop day, requests locked
		{19, false, true},  // gap still ends before day 20
		{20, false, false}, // new period opens
	}

	for _, tc := range cases {
		now := time.Date(2026, time.July, tc.day, 12, 0, 0, 0, vn)
		if got := IsBeforeCutoff(now, 12); got != tc.beforeCutoff {
			t.Errorf("day %d: IsBeforeCutoff(now, 12) = %v, want %v", tc.day, got, tc.beforeCutoff)
		}
		if got := IsInLockedGap(now, 12); got != tc.inLockedGap {
			t.Errorf("day %d: IsInLockedGap(now, 12) = %v, want %v", tc.day, got, tc.inLockedGap)
		}
	}
}
