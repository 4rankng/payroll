package domain

import (
	"testing"
	"time"
)

// TestGetStatusRejectedDerivation covers the auto-reject status rule added for
// the checkout-window auto-reject feature. Rejected is derived from
// CheckOutTime==nil && SalaryRejectReason!=nil and must take precedence over the
// 18h orphaned fallback, while a completed-but-unpaid record (has a checkout +
// a reject reason) must still read as completed.
func TestGetStatusRejectedDerivation(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, loc)
	reason := "Đã hết hạn tan ca"
	earning := int64(0)
	checkout := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)

	cases := []struct {
		name string
		att  Attendance
		want AttendanceStatus
	}{
		{
			name: "completed beats reject reason (completed-but-unpaid)",
			att: Attendance{
				CheckInTime:        now.Add(-20 * time.Hour),
				CheckOutTime:       &checkout,
				SalaryRejectReason: &reason,
				EarningAmount:      &earning,
			},
			want: AttendanceStatusCompleted,
		},
		{
			name: "rejected: no checkout + reason",
			att: Attendance{
				CheckInTime:        now.Add(-2 * time.Hour),
				SalaryRejectReason: &reason,
				EarningAmount:      &earning,
			},
			want: AttendanceStatusRejected,
		},
		{
			name: "rejected beats orphaned when check-in older than 18h",
			att: Attendance{
				CheckInTime:        now.Add(-20 * time.Hour), // would be orphaned, but…
				SalaryRejectReason: &reason,                  // …rejected wins
			},
			want: AttendanceStatusRejected,
		},
		{
			name: "orphaned: no checkout, no reason, older than 18h",
			att: Attendance{
				CheckInTime: now.Add(-20 * time.Hour),
			},
			want: AttendanceStatusOrphaned,
		},
		{
			name: "checked_in: no checkout, no reason, recent",
			att: Attendance{
				CheckInTime: now.Add(-2 * time.Hour),
			},
			want: AttendanceStatusCheckedIn,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.att.GetStatus(now); got != tc.want {
				t.Fatalf("GetStatus = %q, want %q", got, tc.want)
			}
		})
	}
}
