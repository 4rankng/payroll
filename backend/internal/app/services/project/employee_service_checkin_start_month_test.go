package project

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// The start-month rules: the admin picks the month the service starts on and
// the day is always the 1st.
//   - "next month" → pending, activates on day 1 of the next month
//   - "this month" → the 1st has already passed, so the service activates now
//     and the start date is recorded as the 1st of the current month
//   - a pending enable can be moved to the other month, which activates
//     immediately when the target month has already begun

func TestToggleCheckInEnableThisMonthActivatesImmediately(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthThisMonth, 1); err != nil {
		t.Fatalf("toggle enable: %v", err)
	}

	a := repo.assignment
	if !a.CheckInEnabled {
		t.Error("a this-month enable must be live immediately, not pending")
	}
	if a.HasPendingCheckInEnable() {
		t.Error("an activated row must not stay pending")
	}
	want := time.Date(2026, 8, 1, 0, 0, 0, 0, clock.DefaultLocation)
	if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(want) {
		t.Errorf("CheckInStartDate = %v, want %v (day 1 of the chosen month)", a.CheckInStartDate, want)
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved = %d rows, want 1", len(repo.saved))
	}
}

func TestToggleCheckInEnableNextMonthStillDefersAndRecordsNothingYet(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("toggle enable: %v", err)
	}

	a := repo.assignment
	if a.CheckInEnabled {
		t.Error("a next-month enable must stay pending")
	}
	if a.CheckInStartDate != nil {
		t.Errorf("CheckInStartDate = %v, want nil until the pending enable activates", a.CheckInStartDate)
	}
}

func TestToggleCheckInMovesPendingEnableToThisMonth(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("queue for next month: %v", err)
	}
	if !repo.assignment.HasPendingCheckInEnable() {
		t.Fatal("first enable must leave the row pending")
	}

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthThisMonth, 1); err != nil {
		t.Fatalf("move to this month: %v", err)
	}

	a := repo.assignment
	if !a.CheckInEnabled || a.HasPendingCheckInEnable() {
		t.Errorf("moving a pending enable into the current month must activate it, got enabled=%v pending=%v", a.CheckInEnabled, a.HasPendingCheckInEnable())
	}
	want := time.Date(2026, 8, 1, 0, 0, 0, 0, clock.DefaultLocation)
	if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(want) {
		t.Errorf("CheckInStartDate = %v, want %v", a.CheckInStartDate, want)
	}
}

func TestToggleCheckInRescheduleToSameMonthIsNotWritten(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("first enable: %v", err)
	}
	savesAfterFirst := len(repo.saved)

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("repeat enable: %v", err)
	}
	if len(repo.saved) != savesAfterFirst {
		t.Error("re-picking the same month must not write again")
	}
}

func TestBulkToggleCheckInThisMonthAppliesToWholeSelection(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	first := newPendingAssignment()
	second := newPendingAssignment()
	second.ID = 2
	second.EmployeeID = 100
	repo := &inactiveCheckInAssignmentRepo{
		assignments: map[uint]*domain.ProjectEmployee{99: first, 100: second},
	}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99, 100}, true, domain.CheckInStartMonthThisMonth, 1); err != nil {
		t.Fatalf("bulk enable: %v", err)
	}

	for _, assignment := range []*domain.ProjectEmployee{first, second} {
		if !assignment.CheckInEnabled {
			t.Errorf("employee %d must be live after a this-month bulk enable", assignment.EmployeeID)
		}
		want := time.Date(2026, 8, 1, 0, 0, 0, 0, clock.DefaultLocation)
		if assignment.CheckInStartDate == nil || !assignment.CheckInStartDate.Equal(want) {
			t.Errorf("employee %d CheckInStartDate = %v, want %v", assignment.EmployeeID, assignment.CheckInStartDate, want)
		}
	}
}

// The start month must not already be funded by the admin workbook: enabling
// self check-in there would give the employee the workbook amount AND their own
// check-in earnings for the same period.
func TestToggleCheckInEnableRefusesMonthWithWorkbookQuota(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	advanceRepo := &checkinPendingAdvanceRepo{rows: []*domain.AdvancePayment{{
		ProjectID: 5, EmployeeID: 99, ForMonth: "2026-08",
		MaxAdvAmount: 2_000_000, LastAppliedAssetID: uintPtr(55),
	}}}
	svc := newCheckinPendingService(repo, advanceRepo)

	err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthThisMonth, 1)
	if err == nil {
		t.Fatal("enabling for a month that already has a workbook quota must be refused")
	}
	if !strings.Contains(err.Error(), "bảng lương") {
		t.Errorf("error should name the workbook as the reason, got %v", err)
	}
	if repo.assignment.CheckInEnabled || repo.assignment.HasPendingCheckInEnable() {
		t.Error("a refused enable must leave the assignment untouched")
	}
	if len(repo.saved) != 0 {
		t.Errorf("a refused enable must not write, got %d writes", len(repo.saved))
	}

	// The next month is free of workbook quota, so the same enable is allowed.
	advanceRepo.rows = nil
	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("next-month enable: %v", err)
	}
	if !repo.assignment.HasPendingCheckInEnable() {
		t.Error("next-month enable must be queued")
	}
}

func uintPtr(v uint) *uint { return &v }
