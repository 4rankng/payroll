package clock

import (
	"testing"
	"time"
)

func TestKyFromWorkDay(t *testing.T) {
	cases := []struct {
		day  int
		want int
	}{
		{1, 1}, {7, 1},
		{8, 2}, {14, 2},
		{15, 3}, {21, 3},
		{22, 4}, {28, 4}, {29, 4}, {31, 4},
	}
	for _, c := range cases {
		if got := KyFromWorkDay(c.day); got != c.want {
			t.Errorf("KyFromWorkDay(%d) = %d, want %d", c.day, got, c.want)
		}
	}
}

func TestMaxCycleDay(t *testing.T) {
	cases := []struct {
		ky        int
		year      int
		month     time.Month
		want      int
	}{
		{1, 2026, time.July, 10},
		{2, 2026, time.July, 10},
		{3, 2026, time.July, 10},
		{4, 2026, time.July, 11},  // 31-day July: (31-22+1)+1 = 11
		{4, 2026, time.June, 10},  // 30-day June: (30-22+1)+1 = 10
		{4, 2026, time.February, 8}, // 28-day Feb: (28-22+1)+1 = 8
	}
	for _, c := range cases {
		if got := MaxCycleDay(c.ky, c.year, c.month); got != c.want {
			t.Errorf("MaxCycleDay(%d, %d, %s) = %d, want %d", c.ky, c.year, c.month, got, c.want)
		}
	}
}

func TestPayDate(t *testing.T) {
	july := time.Date(2026, time.July, 10, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(1, 2026, time.July); !got.Equal(july) {
		t.Errorf("PayDate(1, Jul) = %v, want %v", got, july)
	}
	// Ky 4 crosses the month boundary: work July → pay day 1 of August.
	aug1 := time.Date(2026, time.August, 1, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(4, 2026, time.July); !got.Equal(aug1) {
		t.Errorf("PayDate(4, Jul) = %v, want %v (Aug 1)", got, aug1)
	}
	// Ky 4 year-boundary: work December → pay day 1 of January next year.
	jan1 := time.Date(2027, time.January, 1, 0, 0, 0, 0, DefaultLocation)
	if got := PayDate(4, 2026, time.December); !got.Equal(jan1) {
		t.Errorf("PayDate(4, Dec) = %v, want %v (Jan 1 2027)", got, jan1)
	}
}

func TestCycleDayForApproval(t *testing.T) {
	// Ky 2 (work-start day 8) approval on day 11 → cycle-day 4.
	d11 := time.Date(2026, time.July, 11, 9, 30, 0, 0, DefaultLocation)
	if got := CycleDayForApproval(2, 2026, time.July, d11); got != 4 {
		t.Errorf("Ky2 approval Jul 11 = cycle-day %d, want 4", got)
	}
	// Approval before the work window clamps to cycle-day 1.
	d5 := time.Date(2026, time.July, 5, 0, 0, 0, 0, DefaultLocation)
	if got := CycleDayForApproval(2, 2026, time.July, d5); got != 1 {
		t.Errorf("Ky2 approval Jul 5 (pre-window) = cycle-day %d, want 1", got)
	}
	// Approval after the pay date clamps to MaxCycleDay.
	d30 := time.Date(2026, time.July, 30, 0, 0, 0, 0, DefaultLocation)
	if got := CycleDayForApproval(2, 2026, time.July, d30); got != 10 {
		t.Errorf("Ky2 approval Jul 30 (post-pay) = cycle-day %d, want 10 (clamp)", got)
	}
	// Ky 4 month-boundary: work-start Jul 22, approval Aug 1 → cycle-day 11.
	aug1 := time.Date(2026, time.August, 1, 12, 0, 0, 0, DefaultLocation)
	if got := CycleDayForApproval(4, 2026, time.July, aug1); got != 11 {
		t.Errorf("Ky4 approval Aug 1 = cycle-day %d, want 11 (boundary)", got)
	}
	// Ky 4 approval still within July (Jul 29) → cycle-day 8.
	jul29 := time.Date(2026, time.July, 29, 0, 0, 0, 0, DefaultLocation)
	if got := CycleDayForApproval(4, 2026, time.July, jul29); got != 8 {
		t.Errorf("Ky4 approval Jul 29 = cycle-day %d, want 8", got)
	}
}

func TestCycleDayForApprovalTimezoneSafe(t *testing.T) {
	// A timestamp at 23:00 UTC on Jul 11 is 06:00 +07:00 on Jul 12 — the same
	// instant must map to cycle-day 5 regardless of which timezone the caller
	// passes, because approval is interpreted in DefaultLocation.
	utc := time.Date(2026, time.July, 11, 23, 0, 0, 0, time.UTC)
	localSame := utc.In(DefaultLocation)
	gotUTC := CycleDayForApproval(2, 2026, time.July, utc)
	gotLocal := CycleDayForApproval(2, 2026, time.July, localSame)
	if gotUTC != gotLocal {
		t.Errorf("same instant mapped differently across timezones: utc=%d local=%d", gotUTC, gotLocal)
	}
	if gotUTC != 5 {
		t.Errorf("Jul 11 23:00 UTC = Jul 12 local → cycle-day %d, want 5", gotUTC)
	}
}

func TestNextTimesheetPayCycle(t *testing.T) {
	cases := []struct {
		desc         string
		now          time.Time
		wantKy       int
		wantPayDay   int
		wantCycleDay int
		wantMaxDay   int
	}{
		{"Jul 3 (mid Ky1)", dateAt(2026, time.July, 3), 1, 10, 3, 10},
		{"Jul 8 (post-work, pre-pay Ky1)", dateAt(2026, time.July, 8), 1, 10, 8, 10},
		{"Jul 9 (eve of Ky1 pay)", dateAt(2026, time.July, 9), 1, 10, 9, 10},
		// On a cycle's pay day the card must roll FORWARD: that cycle's transfer
		// is happening and its prep window (pay date − lead) has closed.
		{"Jul 10 (Ky1 pay day → advance to Ky2)", dateAt(2026, time.July, 10), 2, 17, 3, 10},
		{"Jul 11 (Ky2)", dateAt(2026, time.July, 11), 2, 17, 4, 10},
		{"Jul 16 (eve of Ky2 pay)", dateAt(2026, time.July, 16), 2, 17, 9, 10},
		{"Jul 17 (Ky2 pay day → advance to Ky3)", dateAt(2026, time.July, 17), 3, 24, 3, 10},
		{"Jul 20 (Ky3)", dateAt(2026, time.July, 20), 3, 24, 6, 10},
		{"Jul 24 (Ky3 pay day → advance to Ky4)", dateAt(2026, time.July, 24), 4, 1, 3, 11},
		{"Jul 25 (Ky4)", dateAt(2026, time.July, 25), 4, 1, 4, 11},
		{"Jul 31 (Ky4, last day)", dateAt(2026, time.July, 31), 4, 1, 10, 11},
	}
	for _, c := range cases {
		pc := NextTimesheetPayCycle(c.now)
		if pc.Ky != c.wantKy || pc.NextPayDate.Day() != c.wantPayDay || pc.CycleDayToday != c.wantCycleDay || pc.MaxCycleDay != c.wantMaxDay {
			t.Errorf("%s: got Ky=%d payDay=%d cycleDay=%d maxDay=%d, want Ky=%d payDay=%d cycleDay=%d maxDay=%d",
				c.desc, pc.Ky, pc.NextPayDate.Day(), pc.CycleDayToday, pc.MaxCycleDay,
				c.wantKy, c.wantPayDay, c.wantCycleDay, c.wantMaxDay)
		}
		// Ky4 pay date must land in the next calendar month.
		if pc.Ky == 4 && !pc.NextPayDate.After(c.now) {
			t.Errorf("%s: Ky4 pay date %v not after now %v", c.desc, pc.NextPayDate, c.now)
		}
	}
}

func TestNextTimesheetPayCycleKy4MonthBoundary(t *testing.T) {
	// The primary correctness risk: Ky 4 work window Jul 22–28, pay Aug 1.
	pc := NextTimesheetPayCycle(dateAt(2026, time.July, 29))
	if pc.Ky != 4 {
		t.Fatalf("Jul 29 → Ky %d, want 4", pc.Ky)
	}
	wantPay := time.Date(2026, time.August, 1, 0, 0, 0, 0, DefaultLocation)
	if !pc.NextPayDate.Equal(wantPay) {
		t.Errorf("Ky4 Jul pay date = %v, want %v", pc.NextPayDate, wantPay)
	}
	if pc.MaxCycleDay != 11 {
		t.Errorf("Ky4 July MaxCycleDay = %d, want 11", pc.MaxCycleDay)
	}
	// cycle-day 11 = the Aug 1 pay date.
	if got := CycleDayForApproval(4, 2026, time.July, wantPay); got != 11 {
		t.Errorf("Aug 1 maps to cycle-day %d, want 11", got)
	}
}

func TestPrepareByDate(t *testing.T) {
	pc := NextTimesheetPayCycle(dateAt(2026, time.July, 3)) // Ky1, pay Jul 10
	want := time.Date(2026, time.July, 8, 0, 0, 0, 0, DefaultLocation)
	if got := pc.PrepareByDate(2); !got.Equal(want) {
		t.Errorf("PrepareByDate(2) = %v, want %v", got, want)
	}
	if got := pc.PrepareByDate(-1); !got.Equal(pc.NextPayDate) {
		t.Errorf("PrepareByDate(-1) should clamp to 0 = pay date %v, got %v", pc.NextPayDate, got)
	}
}

// dateAt builds a midnight DefaultLocation time for the given calendar date.
func dateAt(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, DefaultLocation)
}

// TestNextPayCycleAfter_WrapAround exercises the helper the settlement
// simulation uses to project "current + next N" cycles. Kỳ 4 must wrap to
// Kỳ 1 of the next month; Kỳ 1→2→3 stay in the same month.
func TestNextPayCycleAfter_WrapAround(t *testing.T) {
	july1 := NextTimesheetPayCycle(dateAt(2026, time.July, 3)) // Kỳ 1, July
	if july1.Ky != 1 {
		t.Fatalf("seed: expected Kỳ 1, got %d", july1.Ky)
	}

	k2 := NextPayCycleAfter(july1)
	if k2.Ky != 2 || k2.WorkMonth.Month() != time.July {
		t.Errorf("Kỳ 1 → Kỳ %d @ %v, want Kỳ 2 @ July", k2.Ky, k2.WorkMonth)
	}
	k3 := NextPayCycleAfter(k2)
	if k3.Ky != 3 || k3.WorkMonth.Month() != time.July {
		t.Errorf("Kỳ 2 → Kỳ %d @ %v, want Kỳ 3 @ July", k3.Ky, k3.WorkMonth)
	}
	k4 := NextPayCycleAfter(k3)
	if k4.Ky != 4 || k4.WorkMonth.Month() != time.July {
		t.Errorf("Kỳ 3 → Kỳ %d @ %v, want Kỳ 4 @ July", k4.Ky, k4.WorkMonth)
	}
	// Wrap: Kỳ 4 of July → Kỳ 1 of August.
	nextK1 := NextPayCycleAfter(k4)
	if nextK1.Ky != 1 || nextK1.WorkMonth.Month() != time.August {
		t.Errorf("Kỳ 4 wrap → Kỳ %d @ %v, want Kỳ 1 @ August", nextK1.Ky, nextK1.WorkMonth)
	}
	// And the August Kỳ 1 pays on Aug 10.
	if !nextK1.NextPayDate.Equal(dateAt(2026, time.August, 10)) {
		t.Errorf("Aug Kỳ 1 pay date = %v, want 2026-08-10", nextK1.NextPayDate)
	}
}

// TestCycleWindow covers each Kỳ's work-day window, including the Kỳ 4
// "rest-of-month" case (July has 31 days → days 22–31).
func TestCycleWindow(t *testing.T) {
	cases := []struct {
		ky       int
		fromDay  int
		toDay    int
	}{
		{1, 1, 7},
		{2, 8, 14},
		{3, 15, 21},
		{4, 22, 31}, // July has 31 days
	}
	month := dateAt(2026, time.July, 1)
	for _, c := range cases {
		pc := TimesheetPayCycle{Ky: c.ky, WorkMonth: month}
		from, to := pc.CycleWindow()
		if from.Day() != c.fromDay {
			t.Errorf("Kỳ %d from day = %d, want %d", c.ky, from.Day(), c.fromDay)
		}
		if to.Day() != c.toDay {
			t.Errorf("Kỳ %d to day = %d, want %d", c.ky, to.Day(), c.toDay)
		}
	}
}
