package services

import (
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

// TestPostBatchDailyTotal pins the "post-batch daily total" math used by
// ValidateBulkDailyHoursOptimized. The bug this guards against: a request
// that overwrites an existing row (same paytype) used to be summed on top of
// the existing row, producing 2.0 + 10.5 + 12.5 = 25.0h instead of the
// correct 2.0 + 12.5 = 14.5h. Employee 661, 2026-06-21.
//
// Pure function tests — no DB, no mocks, no SQLite-vs-MySQL drift.

func makeExisting(paytype string, hours float64, date time.Time) *domain.Timesheet {
	return &domain.Timesheet{
		ProjectID:   65,
		EmployeeID:  661,
		Date:        date,
		HoursWorked: hours,
		PayType:     paytype,
	}
}

func TestPostBatchDailyTotal(t *testing.T) {
	day := time.Date(2026, 6, 21, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name         string
		existing     []*domain.Timesheet
		batchHours   []float64
		deletionHrs  float64
		upsertHrs    float64
		wantTotal    float64
		wantOver24h  bool
		description  string
	}{
		{
			name:        "upsert replaces existing row (the user's bug)",
			existing:    []*domain.Timesheet{makeExisting("com", 2.0, day), makeExisting("ot200", 10.5, day)},
			batchHours:  []float64{12.5},
			upsertHrs:   10.5, // old ot200 hours — replaced by 12.5 in the request
			wantTotal:   14.5, // 2.0 + 12.5 (ot200 10.5 excluded by upsert subtraction)
			wantOver24h: false,
			description: "Employee 661 day 21: updating ot200 10.5→12.5 must read as 14.5h, not 25.0h",
		},
		{
			name:        "pure addition — no overwrite",
			existing:    []*domain.Timesheet{makeExisting("com", 8.0, day)},
			batchHours:  []float64{8.0},
			upsertHrs:   0,
			wantTotal:   16.0,
			wantOver24h: false,
			description: "Fresh entry on top of existing rows; no upsert subtraction",
		},
		{
			name:        "deletion sentinel zeros existing",
			existing:    []*domain.Timesheet{makeExisting("com", 8.0, day), makeExisting("ot200", 4.0, day)},
			batchHours:  []float64{8.0},
			deletionHrs: -1, // sentinel: delete all existing
			upsertHrs:   0,
			wantTotal:   8.0,
			wantOver24h: false,
			description: "hoursWorked=0 request zeroes the existing total",
		},
		{
			name:        "deletion + upsert on different rows",
			existing:    []*domain.Timesheet{makeExisting("com", 8.0, day), makeExisting("ot200", 10.5, day)},
			batchHours:  []float64{12.5},
			deletionHrs: -1,
			upsertHrs:   10.5, // ot200 10.5 in the existing list, but sentinel zeros first
			wantTotal:   12.5,
			wantOver24h: false,
			description: "Sentinel zeroes before upsert subtraction (so upsertHrs subtracts from 0, clamped)",
		},
		{
			name:        "upsert alone without sentinel — subtraction only",
			existing:    []*domain.Timesheet{makeExisting("com", 8.0, day), makeExisting("ot200", 10.5, day)},
			batchHours:  []float64{12.5},
			deletionHrs: 0,
			upsertHrs:   10.5,
			wantTotal:   18.5 + 12.5 - 10.5,
			wantOver24h: false,
		},
		{
			name:        "over 24h without upsert — still rejected",
			existing:    []*domain.Timesheet{makeExisting("com", 24.0, day)},
			batchHours:  []float64{1.0},
			upsertHrs:   0,
			wantTotal:   25.0,
			wantOver24h: true,
			description: "Sanity: positive case must still reject",
		},
		{
			name:        "over 24h caused by upsert that REPLACES a smaller value",
			existing:    []*domain.Timesheet{makeExisting("com", 8.0, day), makeExisting("ot200", 4.0, day)},
			batchHours:  []float64{20.0},
			upsertHrs:   4.0, // replacing ot200 4.0 → 20.0; total = 8.0 + 20.0 = 28.0
			wantTotal:   28.0,
			wantOver24h: true,
			description: "Upsert that itself pushes over 24h must still reject",
		},
		{
			name:        "upsertHours clamps existingTotal at 0 (defensive)",
			existing:    []*domain.Timesheet{makeExisting("ot200", 5.0, day)},
			batchHours:  []float64{8.0},
			upsertHrs:   999.0, // corrupt caller data — must not produce negative total
			wantTotal:   8.0,
			wantOver24h: false,
			description: "If upsertHrs > existingTotal, clamp at 0 (don't go negative)",
		},
		{
			name:        "empty inputs",
			existing:    nil,
			batchHours:  nil,
			upsertHrs:   0,
			wantTotal:   0,
			wantOver24h: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := postBatchDailyTotal(tc.existing, tc.batchHours, tc.deletionHrs, tc.upsertHrs)
			assert.InDelta(t, tc.wantTotal, got, 1e-9, "%s — got %.4f", tc.description, got)
			assert.Equal(t, tc.wantOver24h, got > 24, "%s — over-24h flag", tc.description)
		})
	}
}

// TestPostBatchDailyTotal_Employee661Regression is the named end-to-end
// reproduction of the user's exact scenario. Kept as a separate test so
// failure messages point directly at the incident.
func TestPostBatchDailyTotal_Employee661Regression(t *testing.T) {
	day := time.Date(2026, 6, 21, 0, 0, 0, 0, time.Local)

	// Prod DB on 2026-06-21 (verified via read-only query):
	//   com 2.0 + ot200 10.5 = 12.5h
	existing := []*domain.Timesheet{
		makeExisting("com.weekday.com", 2.0, day),
		makeExisting("com.weekday.ot200", 10.5, day),
	}

	// UI submission (POST /api/v1/timesheets/preview):
	//   { projectId:65, employeeId:661, date:"2026-06-21", hoursWorked:12.5, hourType:"OT200" }
	// The request's paytype matches the existing ot200 row → upsert, not addition.
	batchHours := []float64{12.5}
	upsertHours := 10.5 // the OLD ot200 hours being replaced

	got := postBatchDailyTotal(existing, batchHours, 0, upsertHours)
	assert.InDelta(t, 14.5, got, 1e-9,
		"emp 661 / 2026-06-21 / OT200 12.5h upsert: must read as 14.5h (matches UI), not 25.0h (the prod false reject)")
	assert.False(t, got > 24, "14.5h must NOT trigger the >24h rejection")
}
