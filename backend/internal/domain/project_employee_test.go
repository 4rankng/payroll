package domain

import (
	"testing"
	"time"

	"api-server/internal/pkg/clock"

	"github.com/stretchr/testify/assert"
)

// setFakeClock freezes the package-level clock used by the pending check-in
// rules and restores the real clock when the test finishes.
func setFakeClock(t *testing.T, now time.Time) {
	t.Helper()
	clock.SetGlobal(clock.NewFake(now))
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })
}

// newCheckinAssignment returns a currently-assigned row that tests can seed
// with check-in state; LastDate nil keeps IsCurrentlyAssigned true.
func newCheckinAssignment() *ProjectEmployee {
	return &ProjectEmployee{
		ID:              1,
		ProjectID:       5,
		EmployeeID:      99,
		PaymentSchedule: string(PaymentScheduleWeekly),
		StartDate:       time.Date(2026, 1, 1, 0, 0, 0, 0, clock.DefaultLocation),
	}
}

func TestProjectEmployee_ValidateProjectID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid project ID
	pe.ProjectID = 1
	err := pe.ValidateProjectID()
	assert.NoError(t, err)

	// Test invalid project ID
	pe.ProjectID = 0
	err = pe.ValidateProjectID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateEmployeeID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid employee ID
	pe.EmployeeID = 1
	err := pe.ValidateEmployeeID()
	assert.NoError(t, err)

	// Test invalid employee ID
	pe.EmployeeID = 0
	err = pe.ValidateEmployeeID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateDates(t *testing.T) {
	now := time.Now()
	var endDate *time.Time
	endTime := now.AddDate(0, 1, 1) // next month
	endDate = &endTime

	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, 1), // tomorrow
		LastDate:  endDate,              // next month
	}

	err := pe.ValidateDates()
	assert.NoError(t, err)

	// Test start date after end date
	startDate := now.AddDate(0, 1, 2) // after the end date
	pe.StartDate = startDate
	err = pe.ValidateDates()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidatePosition(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid position
	pe.Position = "Developer"
	err := pe.ValidatePosition()
	assert.NoError(t, err)

	// Test empty position
	pe.Position = ""
	err = pe.ValidatePosition()
	assert.Error(t, err)
}

func TestProjectEmployee_IsValid(t *testing.T) {
	now := time.Now()
	var endDate *time.Time
	endTime := now.AddDate(0, 1, 1) // next month
	endDate = &endTime

	pe := &ProjectEmployee{
		ProjectID:       1,
		EmployeeID:      1,
		Position:        "Developer",
		StartDate:       now.AddDate(0, 0, 1), // tomorrow
		LastDate:        endDate,              // next month
		PaymentSchedule: "weekly",             // Default payment schedule
	}

	// Test valid project employee
	err := pe.IsValid()
	assert.NoError(t, err)

	// Test invalid project employee
	pe.ProjectID = 0
	err = pe.IsValid()
	assert.Error(t, err)
}

func TestProjectEmployee_IsCurrentlyAssigned(t *testing.T) {
	now := time.Now()

	// Test assignment that is active now
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, -5), // 5 days ago
		LastDate:  nil,                   // Currently active
	}

	assert.True(t, pe.IsCurrentlyAssigned())

	// Test assignment that ended in the past
	endDate := now.AddDate(0, 0, -1) // yesterday
	pe.LastDate = &endDate
	assert.False(t, pe.IsCurrentlyAssigned())
}

func TestProjectEmployee_HasEnded(t *testing.T) {
	now := time.Now()

	// Test assignment that has ended
	endDate := now.AddDate(0, -1, 0) // 1 month ago
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, -2, 0), // 2 months ago
		LastDate:  &endDate,
	}

	assert.True(t, pe.HasEnded())

	// Test assignment that is still active
	pe.LastDate = nil
	assert.False(t, pe.HasEnded())
}

func TestProjectEmployee_GetAssignmentDuration(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // 1 month ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	duration := pe.GetAssignmentDuration()
	expectedDuration := int(endDate.Sub(startDate).Hours() / 24) // Convert duration to days
	assert.Equal(t, expectedDuration, duration)
}

func TestProjectEmployee_CanEndAssignment(t *testing.T) {
	now := time.Now()

	// Test assignment that can be ended (it's currently active)
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, -5), // 5 days ago
		LastDate:  nil,                   // Currently active
	}

	assert.True(t, pe.CanEndAssignment())

	// Test assignment that has already ended
	endDate := now.AddDate(0, 0, -1) // yesterday
	pe.LastDate = &endDate
	assert.False(t, pe.CanEndAssignment())
}

func TestProjectEmployee_EndAssignment(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  nil, // Currently active
	}

	// End the assignment
	newEndDate := now.AddDate(0, 0, -1) // yesterday
	err := pe.EndAssignment(newEndDate)
	assert.NoError(t, err)

	// Check if end date is set
	assert.True(t, pe.HasEnded())
}

func TestProjectEmployee_CanCreateTimesheet(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	// Should be able to create timesheet for today
	assert.True(t, pe.CanCreateTimesheet(now))

	// Should not be able to create timesheet before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.CanCreateTimesheet(beforeStartDate))

	// Should not be able to create timesheet after end date
	afterEndDate := now.AddDate(0, 2, 0)
	assert.False(t, pe.CanCreateTimesheet(afterEndDate))

	// Test when LastDate is nil (currently active)
	pe.LastDate = nil
	assert.True(t, pe.CanCreateTimesheet(now))
}

func TestProjectEmployee_IsAssignedOnDate(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	// Should be assigned on today
	assert.True(t, pe.IsAssignedOnDate(now))

	// Should not be assigned before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.IsAssignedOnDate(beforeStartDate))

	// Should not be assigned after end date
	afterEndDate := now.AddDate(0, 2, 0)
	assert.False(t, pe.IsAssignedOnDate(afterEndDate))

	// Test when LastDate is nil (currently active)
	pe.LastDate = nil
	assert.True(t, pe.IsAssignedOnDate(now))
}

// The chatbot enable follows the day-9 rule (owner ruling 2026-10-07): days
// 1-8 start in the current month, day 9 or later waits for the next month.
func TestProjectEmployee_ResolveCheckInEnableStartMonth_Day9Boundary(t *testing.T) {
	tests := []struct {
		name string
		day  int
		want CheckInStartMonth
	}{
		{name: "day 1 starts this month", day: 1, want: CheckInStartMonthThisMonth},
		{name: "day 8 still starts this month", day: 8, want: CheckInStartMonthThisMonth},
		{name: "day 9 counts as after and waits", day: 9, want: CheckInStartMonthNextMonth},
		{name: "day 15 waits", day: 15, want: CheckInStartMonthNextMonth},
		{name: "day 28 waits", day: 28, want: CheckInStartMonthNextMonth},
		{name: "last day of the month waits", day: 31, want: CheckInStartMonthNextMonth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 10, tt.day, 12, 0, 0, 0, clock.DefaultLocation)
			startMonth := ResolveCheckInEnableStartMonth(now)

			assert.Equal(t, tt.want, startMonth)

			// The choice always resolves to day 1 of the selected month —
			// the date the bot quotes as the effective date.
			effective := startMonth.EffectiveFrom(now)
			assert.Equal(t, 1, effective.Day())
			if tt.want == CheckInStartMonthThisMonth {
				assert.Equal(t, time.October, effective.Month())
			} else {
				assert.Equal(t, time.November, effective.Month())
			}
		})
	}
}

// RequestCheckInDisable refuses every state that may not queue a disable:
// an ended assignment, a row that is not enabled, and any pending change —
// in either direction.
func TestProjectEmployee_RequestCheckInDisable_Guards(t *testing.T) {
	effective := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)

	tests := []struct {
		name  string
		seed  func(t *testing.T) *ProjectEmployee
		still func(t *testing.T, pe *ProjectEmployee)
	}{
		{
			name: "ended assignment is refused",
			seed: func(t *testing.T) *ProjectEmployee {
				pe := newCheckinAssignment()
				pe.CheckInEnabled = true
				ended := time.Date(2026, 9, 30, 0, 0, 0, 0, clock.DefaultLocation)
				pe.LastDate = &ended
				return pe
			},
			still: func(t *testing.T, pe *ProjectEmployee) {
				assert.False(t, pe.HasPendingCheckInChange())
			},
		},
		{
			name: "not-enabled row is refused",
			seed: func(t *testing.T) *ProjectEmployee { return newCheckinAssignment() },
			still: func(t *testing.T, pe *ProjectEmployee) {
				assert.False(t, pe.HasPendingCheckInChange())
			},
		},
		{
			name: "pending enable blocks a disable",
			seed: func(t *testing.T) *ProjectEmployee {
				pe := newCheckinAssignment()
				assert.NoError(t, pe.RequestCheckInEnable(effective))
				return pe
			},
			still: func(t *testing.T, pe *ProjectEmployee) {
				assert.True(t, pe.HasPendingCheckInEnable())
			},
		},
		{
			name: "pending disable blocks a second disable",
			seed: func(t *testing.T) *ProjectEmployee {
				pe := newCheckinAssignment()
				pe.CheckInEnabled = true
				disabled := false
				pe.PendingCheckInEnabled = &disabled
				pe.CheckInEffectiveFrom = &effective
				return pe
			},
			still: func(t *testing.T, pe *ProjectEmployee) {
				assert.True(t, pe.HasPendingCheckInDisable())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := tt.seed(t)

			err := pe.RequestCheckInDisable(effective)

			assert.Error(t, err)
			tt.still(t, pe)
		})
	}
}

// ApplyPendingCheckInChange applies a pending enable or disable once the
// effective date (day 1 of a month) has arrived; until then the row stays
// untouched. A disable keeps CheckInStartDate so the roster keeps reporting
// when the service was last in use.
func TestProjectEmployee_ApplyPendingCheckInChange(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 15, 8, 0, 0, 0, clock.DefaultLocation))

	enabledOn := time.Date(2026, 8, 1, 0, 0, 0, 0, clock.DefaultLocation)

	tests := []struct {
		name        string
		seed        func(pe *ProjectEmployee) // leaves pe with CheckInEnabled per seed
		wantApplied bool
		wantKind    string
		check       func(t *testing.T, pe *ProjectEmployee)
	}{
		{
			name:        "no pending change is a no-op",
			seed:        func(pe *ProjectEmployee) { pe.CheckInEnabled = true },
			wantApplied: false,
			wantKind:    "",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.True(t, pe.CheckInEnabled)
			},
		},
		{
			name: "pending enable effective today activates",
			seed: func(pe *ProjectEmployee) {
				assert.NoError(t, pe.RequestCheckInEnable(time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)))
			},
			wantApplied: true,
			wantKind:    "enable",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.True(t, pe.CheckInEnabled)
				assert.False(t, pe.HasPendingCheckInChange())
				assert.NotNil(t, pe.CheckInStartDate)
				assert.True(t, pe.CheckInStartDate.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)))
			},
		},
		{
			name: "pending enable effective tomorrow stays queued",
			seed: func(pe *ProjectEmployee) {
				assert.NoError(t, pe.RequestCheckInEnable(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)))
			},
			wantApplied: false,
			wantKind:    "",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.False(t, pe.CheckInEnabled)
				assert.True(t, pe.HasPendingCheckInEnable())
			},
		},
		{
			name: "pending disable effective today turns the service off and keeps the start date",
			seed: func(pe *ProjectEmployee) {
				pe.CheckInEnabled = true
				pe.CheckInStartDate = &enabledOn
				disabled := false
				pe.PendingCheckInEnabled = &disabled
				pe.CheckInEffectiveFrom = &[]time.Time{time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)}[0]
			},
			wantApplied: true,
			wantKind:    "disable",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.False(t, pe.CheckInEnabled)
				assert.False(t, pe.HasPendingCheckInChange())
				// History: the disable must not erase when the service started.
				assert.NotNil(t, pe.CheckInStartDate)
				assert.True(t, pe.CheckInStartDate.Equal(enabledOn))
			},
		},
		{
			name: "pending disable effective tomorrow stays queued",
			seed: func(pe *ProjectEmployee) {
				pe.CheckInEnabled = true
				disabled := false
				pe.PendingCheckInEnabled = &disabled
				pe.CheckInEffectiveFrom = &[]time.Time{time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)}[0]
			},
			wantApplied: false,
			wantKind:    "",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.True(t, pe.CheckInEnabled)
				assert.True(t, pe.HasPendingCheckInDisable())
			},
		},
		{
			name: "pending disable overdue from a missed sweep run still applies",
			seed: func(pe *ProjectEmployee) {
				pe.CheckInEnabled = true
				disabled := false
				pe.PendingCheckInEnabled = &disabled
				pe.CheckInEffectiveFrom = &[]time.Time{time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)}[0]
			},
			wantApplied: true,
			wantKind:    "disable",
			check: func(t *testing.T, pe *ProjectEmployee) {
				assert.False(t, pe.CheckInEnabled)
				assert.False(t, pe.HasPendingCheckInChange())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := newCheckinAssignment()
			tt.seed(pe)

			applied, kind := pe.ApplyPendingCheckInChange()

			assert.Equal(t, tt.wantApplied, applied)
			assert.Equal(t, tt.wantKind, kind)
			tt.check(t, pe)
		})
	}
}

// CancelPendingCheckInChange clears both directions; cancelling a pending
// disable leaves the service running, and cancelling with nothing pending
// errors.
func TestProjectEmployee_CancelPendingCheckInChange(t *testing.T) {
	effective := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)

	// Pending enable.
	pe := newCheckinAssignment()
	assert.NoError(t, pe.RequestCheckInEnable(effective))
	assert.NoError(t, pe.CancelPendingCheckInChange())
	assert.False(t, pe.HasPendingCheckInChange())
	assert.False(t, pe.CheckInEnabled)

	// Pending disable: the row keeps running until someone asks again.
	pe = newCheckinAssignment()
	pe.CheckInEnabled = true
	disabled := false
	pe.PendingCheckInEnabled = &disabled
	pe.CheckInEffectiveFrom = &effective
	assert.NoError(t, pe.CancelPendingCheckInChange())
	assert.False(t, pe.HasPendingCheckInChange())
	assert.True(t, pe.CheckInEnabled, "cancelling a pending disable must not turn the service off")

	// Nothing pending.
	assert.Error(t, pe.CancelPendingCheckInChange())
}

// The pending columns carry the direction in the boolean's value: true is a
// pending enable, false is a pending disable. HasPendingCheckInEnable stays
// true-only so existing callers never mistake a queued disable for one.
func TestProjectEmployee_HasPendingCheckInSemantics(t *testing.T) {
	effective := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)

	// No pending change.
	pe := newCheckinAssignment()
	assert.False(t, pe.HasPendingCheckInChange())
	assert.False(t, pe.HasPendingCheckInEnable())
	assert.False(t, pe.HasPendingCheckInDisable())

	// Pending value without an effective date is not a pending change.
	pe = newCheckinAssignment()
	enabled := true
	pe.PendingCheckInEnabled = &enabled
	assert.False(t, pe.HasPendingCheckInChange())

	// Pending enable.
	pe = newCheckinAssignment()
	assert.NoError(t, pe.RequestCheckInEnable(effective))
	assert.True(t, pe.HasPendingCheckInChange())
	assert.True(t, pe.HasPendingCheckInEnable())
	assert.False(t, pe.HasPendingCheckInDisable())

	// Pending disable: a queued disable must NOT satisfy the enable-only check.
	pe = newCheckinAssignment()
	pe.CheckInEnabled = true
	disabled := false
	pe.PendingCheckInEnabled = &disabled
	pe.CheckInEffectiveFrom = &effective
	assert.True(t, pe.HasPendingCheckInChange())
	assert.False(t, pe.HasPendingCheckInEnable(), "a pending disable is not a pending enable")
	assert.True(t, pe.HasPendingCheckInDisable())
}
