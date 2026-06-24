package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDayBoundsInDateLocation is the regression guard for the
// "Tổng giờ làm việc ngày ... vượt quá 24 giờ" false rejection (employee 661,
// 2026-06-21: a 12.5h save was read as 24.5h and rejected).
//
// The bug: day-bound windows were built in time.UTC, which under a loc=Local DSN
// shifts the [start, end) window +7h on prod (UTC+7), so a day-D lookup returned
// day-D+1's rows. Verified on prod — UTC bounds returned 2026-06-22's 12.0h;
// Local bounds returned 2026-06-21's 12.5h.
//
// This test is pure (no DB) because the bug is driver/DSN-specific: SQLite strips
// zone info and compares wall-clocks, so it cannot reproduce the shift. Instead we
// assert the bound is constructed in time.Local at local midnight, which fails the
// moment the helper reverts to time.UTC.
func TestDayBoundsInDateLocation(t *testing.T) {
	// Pin a non-UTC local zone so Local != UTC and the regression is observable
	// regardless of the dev/CI machine's own timezone.
	saved := time.Local
	time.Local = time.FixedZone("ICT", 7*60*60) // UTC+7, mirrors prod
	t.Cleanup(func() { time.Local = saved })

	date := time.Date(2026, time.June, 21, 13, 37, 45, 0, time.Local) // arbitrary moment on day D
	start, end := dayBoundsInDateLocation(date)

	// Window opens at local midnight of day D, in time.Local (NOT time.UTC).
	assert.Equal(t, time.Local, start.Location(),
		"day bounds must be in time.Local; time.UTC shifts the window +offset under loc=Local DSN")
	expectStart := time.Date(2026, time.June, 21, 0, 0, 0, 0, time.Local)
	assert.True(t, start.Equal(expectStart), "start = %v, want local midnight %v", start, expectStart)
	assert.True(t, end.Equal(expectStart.Add(24*time.Hour)), "end must be start+24h")

	// UTC input must fall back to time.Local (same behaviour as the combos path).
	startFromUTC, _ := dayBoundsInDateLocation(time.Date(2026, time.June, 21, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, time.Local, startFromUTC.Location(), "UTC input should fall back to time.Local")
	assert.True(t, startFromUTC.Equal(expectStart))
}
