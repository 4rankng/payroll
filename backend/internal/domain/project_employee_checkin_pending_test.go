package domain

import (
	"testing"
	"time"
)

func vnTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 10, 0, 0, 0, time.FixedZone("VN", 7*3600))
}

func startOfVNDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.FixedZone("VN", 7*3600))
}

func activeAssignment() *ProjectEmployee {
	return &ProjectEmployee{
		PaymentSchedule: string(PaymentScheduleFlexible),
		StartDate:       vnTime(2026, 1, 1),
	}
}

func TestRequestCheckInEnableStoresPending(t *testing.T) {
	pe := activeAssignment()
	effective := startOfVNDay(2026, 9, 1)

	if err := pe.RequestCheckInEnable(effective); err != nil {
		t.Fatalf("RequestCheckInEnable: %v", err)
	}

	if pe.CheckInEnabled {
		t.Error("CheckInEnabled must stay false while pending")
	}
	if !pe.HasPendingCheckInEnable() {
		t.Error("HasPendingCheckInEnable should be true")
	}
	if pe.PendingCheckInEnabled == nil || !*pe.PendingCheckInEnabled {
		t.Error("PendingCheckInEnabled should be true pointer")
	}
	if pe.CheckInEffectiveFrom == nil || !pe.CheckInEffectiveFrom.Equal(effective) {
		t.Errorf("CheckInEffectiveFrom = %v, want %v", pe.CheckInEffectiveFrom, effective)
	}
}

func TestRequestCheckInEnableRejectsActiveAndEnded(t *testing.T) {
	active := activeAssignment()
	active.CheckInEnabled = true
	if err := active.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err == nil {
		t.Error("already-enabled assignment should be rejected")
	}

	ended := activeAssignment()
	end := vnTime(2026, 7, 31)
	ended.LastDate = &end
	if err := ended.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err == nil {
		t.Error("ended assignment should be rejected")
	}
}

func TestRequestCheckInEnableRejectsDuplicatePending(t *testing.T) {
	pe := activeAssignment()
	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 10, 1)); err == nil {
		t.Error("second pending request should be rejected")
	}
}

func TestApplyPendingCheckInBeforeAndOnEffectiveDate(t *testing.T) {
	pe := activeAssignment()
	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err != nil {
		t.Fatalf("request: %v", err)
	}

	// clock.Now() is the real Asia/Ho_Chi_Minh clock; the effective date is
	// chosen far in the past so "due" is deterministic regardless of run date.
	past := activeAssignment()
	if err := past.RequestCheckInEnable(startOfVNDay(2020, 1, 1)); err != nil {
		t.Fatalf("past request: %v", err)
	}
	if !past.ApplyPendingCheckIn() {
		t.Error("past-dated pending should apply")
	}
	if !past.CheckInEnabled || past.HasPendingCheckInEnable() {
		t.Error("apply must set CheckInEnabled and clear pending fields")
	}

	// Far-future effective date never applies today.
	pe.CheckInEffectiveFrom = nil
	future := activeAssignment()
	if err := future.RequestCheckInEnable(startOfVNDay(2100, 1, 1)); err != nil {
		t.Fatalf("future request: %v", err)
	}
	if future.ApplyPendingCheckIn() {
		t.Error("future-dated pending must not apply")
	}
	if future.CheckInEnabled {
		t.Error("future pending must leave CheckInEnabled false")
	}
}

func TestApplyPendingCheckInNoopWithoutPending(t *testing.T) {
	pe := activeAssignment()
	if pe.ApplyPendingCheckIn() {
		t.Error("no pending → no change")
	}
}

func TestCancelPendingCheckInEnable(t *testing.T) {
	pe := activeAssignment()
	if err := pe.CancelPendingCheckInEnable(); err == nil {
		t.Error("cancel without pending should error")
	}

	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err != nil {
		t.Fatalf("request: %v", err)
	}
	if err := pe.CancelPendingCheckInEnable(); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if pe.HasPendingCheckInEnable() || pe.CheckInEnabled {
		t.Error("cancel must clear pending without enabling")
	}
}

func TestParseCheckInStartMonth(t *testing.T) {
	tests := []struct {
		value   string
		want    CheckInStartMonth
		wantErr bool
	}{
		{value: "this_month", want: CheckInStartMonthThisMonth},
		{value: "next_month", want: CheckInStartMonthNextMonth},
		{value: " next_month ", want: CheckInStartMonthNextMonth},
		{value: "", want: CheckInStartMonthNextMonth},
		{value: "2026-10", wantErr: true},
		{value: "next", wantErr: true},
	}

	for _, tt := range tests {
		got, err := ParseCheckInStartMonth(tt.value)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseCheckInStartMonth(%q) = %q, want a validation error", tt.value, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCheckInStartMonth(%q): %v", tt.value, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseCheckInStartMonth(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestCheckInStartMonthEffectiveFromIsAlwaysDayOne(t *testing.T) {
	// Mid-month: this month is already in the past, next month is ahead.
	now := vnTime(2026, 10, 15)
	thisMonth := CheckInStartMonthThisMonth.EffectiveFrom(now)
	if want := startOfVNDay(2026, 10, 1); !thisMonth.Equal(want) {
		t.Errorf("this_month effective = %v, want %v", thisMonth, want)
	}
	nextMonth := CheckInStartMonthNextMonth.EffectiveFrom(now)
	if want := startOfVNDay(2026, 11, 1); !nextMonth.Equal(want) {
		t.Errorf("next_month effective = %v, want %v", nextMonth, want)
	}

	// Month rollover in December must not roll into January of the wrong year.
	december := CheckInStartMonthNextMonth.EffectiveFrom(vnTime(2026, 12, 31))
	if want := startOfVNDay(2027, 1, 1); !december.Equal(want) {
		t.Errorf("December next_month effective = %v, want %v", december, want)
	}
}

func TestActivateCheckInRecordsStartDateAndClearsPending(t *testing.T) {
	pe := activeAssignment()
	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err != nil {
		t.Fatalf("request: %v", err)
	}

	pe.ActivateCheckIn(startOfVNDay(2026, 9, 1))

	if !pe.CheckInEnabled {
		t.Error("ActivateCheckIn must turn the service on")
	}
	if pe.HasPendingCheckInEnable() {
		t.Error("ActivateCheckIn must clear the pending request")
	}
	if pe.CheckInStartDate == nil || !pe.CheckInStartDate.Equal(startOfVNDay(2026, 9, 1)) {
		t.Errorf("CheckInStartDate = %v, want the activation day (2026-09-01)", pe.CheckInStartDate)
	}
}

func TestReschedulePendingCheckInEnableMovesPendingToChosenMonth(t *testing.T) {
	pe := activeAssignment()
	if err := pe.RequestCheckInEnable(startOfVNDay(2026, 9, 1)); err != nil {
		t.Fatalf("request: %v", err)
	}

	changed, err := pe.ReschedulePendingCheckInEnable(startOfVNDay(2026, 8, 1))
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !changed {
		t.Error("moving to a different month must report a change")
	}
	if !pe.HasPendingCheckInEnable() {
		t.Fatal("reschedule must keep the enable pending")
	}
	if !pe.CheckInEffectiveFrom.Equal(startOfVNDay(2026, 8, 1)) {
		t.Errorf("effective = %v, want 2026-08-01", pe.CheckInEffectiveFrom)
	}

	// Same month again is a no-op so the caller skips the write.
	changed, err = pe.ReschedulePendingCheckInEnable(startOfVNDay(2026, 8, 1))
	if err != nil {
		t.Fatalf("reschedule same month: %v", err)
	}
	if changed {
		t.Error("rescheduling to the same month must report no change")
	}

	// A rescheduled month that has already begun applies right away.
	if !pe.ApplyPendingCheckIn() {
		t.Fatal("a past effective date must apply")
	}
	if !pe.CheckInEnabled || pe.HasPendingCheckInEnable() {
		t.Errorf("expected immediate activation, got enabled=%v pending=%v", pe.CheckInEnabled, pe.HasPendingCheckInEnable())
	}
	if pe.CheckInStartDate == nil || !pe.CheckInStartDate.Equal(startOfVNDay(2026, 8, 1)) {
		t.Errorf("CheckInStartDate = %v, want 2026-08-01", pe.CheckInStartDate)
	}
}

func TestReschedulePendingCheckInEnableRejectsNonPendingRow(t *testing.T) {
	pe := activeAssignment()
	if _, err := pe.ReschedulePendingCheckInEnable(startOfVNDay(2026, 8, 1)); err == nil {
		t.Error("rescheduling a row with nothing pending must fail")
	}
}
