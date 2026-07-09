package clock

import (
	"testing"
	"time"
)

// TestRequestCutoffDayValue guards the business rule: the advance-payment
// request window closes after day 8 of the following month. Request creation,
// eligibility, and locked-gap detection all branch on this exact value, so a
// silent change would shift the three-phase window app-wide.
func TestRequestCutoffDayValue(t *testing.T) {
	if RequestCutoffDay != 8 {
		t.Fatalf("RequestCutoffDay = %d, want 8 (window = day 20 of M -> day 8 of M+1)", RequestCutoffDay)
	}
	if PeriodCycleStartDay != 20 {
		t.Fatalf("PeriodCycleStartDay = %d, want 20", PeriodCycleStartDay)
	}
}

// TestRequestWindowBoundaries exercises the three-phase window around the
// day-8 cutoff using Asia/Ho_Chi_Minh local dates (the timezone the business
// runs in). July 2026 anchors the phases relative to calendar July:
//
//	days 1–8  -> tail of the previous period (for_month = June)
//	days 9–19 -> locked gap
//	days 20+   → new period opens (for_month rolls to July/August)
func TestRequestWindowBoundaries(t *testing.T) {
	vn, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("load Asia/Ho_Chi_Minh: %v", err)
	}

	type expect struct {
		day            int
		beforeCutoff   bool
		inLockedGap    bool
		afterCutoff    bool
		effectiveMonth string
	}
	cases := []expect{
		{8, true, false, false, "2026-06"},  // final cutoff day, still previous period
		{9, false, true, true, "2026-07"},   // sao ke day, requests locked
		{10, false, true, true, "2026-07"},  // locked gap
		{11, false, true, true, "2026-07"},  // mid locked gap
		{19, false, true, true, "2026-07"},  // last locked-gap day
		{20, false, false, true, "2026-08"}, // new period opens
	}

	for _, tc := range cases {
		now := time.Date(2026, time.July, tc.day, 12, 0, 0, 0, vn)
		if got := IsBeforeCutoff(now); got != tc.beforeCutoff {
			t.Errorf("day %d: IsBeforeCutoff = %v, want %v", tc.day, got, tc.beforeCutoff)
		}
		if got := IsInLockedGap(now); got != tc.inLockedGap {
			t.Errorf("day %d: IsInLockedGap = %v, want %v", tc.day, got, tc.inLockedGap)
		}
		if got := IsAfterCutoff(now); got != tc.afterCutoff {
			t.Errorf("day %d: IsAfterCutoff = %v, want %v", tc.day, got, tc.afterCutoff)
		}
		if got := EffectiveAdvanceMonthFromTime(now); got != tc.effectiveMonth {
			t.Errorf("day %d: EffectiveAdvanceMonthFromTime = %q, want %q", tc.day, got, tc.effectiveMonth)
		}
	}
}
