package handlers

import "testing"

// Regression: fromDateLocked used to lock the field whenever the *currently
// stored* fromDate sat outside the [floor, earliestLinked] window, instead of
// checking whether the window itself was empty. That stranded users on a
// read-only field even though /validate reported a correctable error with a
// concrete suggested_value (see "Chỉnh sửa cấu hình lương" bug report).
func TestStartDateWindowEmpty(t *testing.T) {
	cases := []struct {
		name           string
		floor          string
		earliestLinked string
		want           bool
	}{
		{
			name:           "floor before earliest linked — window open, not locked",
			floor:          "2026-08-01",
			earliestLinked: "2026-08-15",
			want:           false,
		},
		{
			name:           "floor equals earliest linked — single valid date, not locked",
			floor:          "2026-08-15",
			earliestLinked: "2026-08-15",
			want:           false,
		},
		{
			name:           "floor after earliest linked — genuinely no legal date, locked",
			floor:          "2026-08-16",
			earliestLinked: "2026-08-15",
			want:           true,
		},
		{
			name:           "stored fromDate later than earliest linked is irrelevant to lock state",
			floor:          "2026-07-01",
			earliestLinked: "2026-08-15",
			// Window [2026-07-01, 2026-08-15] is open — a stored fromDate of
			// 2026-08-29 (later than earliestLinked) is a correctable error,
			// not evidence the window itself is empty.
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := startDateWindowEmpty(tc.floor, tc.earliestLinked)
			if got != tc.want {
				t.Errorf("startDateWindowEmpty(%q, %q) = %v, want %v", tc.floor, tc.earliestLinked, got, tc.want)
			}
		})
	}
}
