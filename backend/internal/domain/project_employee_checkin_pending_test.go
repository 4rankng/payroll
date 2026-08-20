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
