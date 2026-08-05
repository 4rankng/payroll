package attendance

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

func newAdminCheckInService(repo *fakeAttendanceRepo, now time.Time) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{
			Position:       "Công nhân",
			CheckInEnabled: true,
		}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{ID: 55, IsFlexible: true}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}
}

func TestAdminCreateCheckInCreatesOnlyTheCheckIn(t *testing.T) {
	loc := clock.DefaultLocation
	now := time.Date(2026, 8, 5, 10, 15, 0, 0, loc)
	repo := &fakeAttendanceRepo{}
	svc := newAdminCheckInService(repo, now)

	attendance, err := svc.AdminCreateCheckIn(context.Background(), 123, 55, now, 0)
	if err != nil {
		t.Fatalf("AdminCreateCheckIn returned error: %v", err)
	}
	if attendance == nil {
		t.Fatal("expected an attendance record")
	}
	wantCheckIn := time.Date(2026, 8, 5, 8, 0, 0, 0, loc)
	if !attendance.CheckInTime.Equal(wantCheckIn) {
		t.Fatalf("check_in_time = %v, want configured shift start %v", attendance.CheckInTime, wantCheckIn)
	}
	if attendance.CheckInGate != "admin" {
		t.Fatalf("check_in_gate = %q, want admin", attendance.CheckInGate)
	}
	if attendance.CheckOutTime != nil || attendance.CheckOutGate != nil {
		t.Fatalf("admin check-in must not create checkout, got time=%v gate=%v", attendance.CheckOutTime, attendance.CheckOutGate)
	}
	if attendance.EarningAmount != nil || attendance.QuotaCreditedAt != nil {
		t.Fatalf("admin check-in must not create earning/quota, got earning=%v quota=%v", attendance.EarningAmount, attendance.QuotaCreditedAt)
	}
}

func TestAdminCreateCheckInRejectsNonFlexibleProject(t *testing.T) {
	loc := clock.DefaultLocation
	now := time.Date(2026, 8, 5, 10, 15, 0, 0, loc)
	repo := &fakeAttendanceRepo{}
	svc := newAdminCheckInService(repo, now)
	svc.projectRepo = &fakeProjectRepo{p: &domain.Project{ID: 55, IsFlexible: false}}

	_, err := svc.AdminCreateCheckIn(context.Background(), 123, 55, now, 0)
	if err == nil || !domain.IsValidationError(err) {
		t.Fatalf("AdminCreateCheckIn error = %v, want flexible-project validation error", err)
	}
}

func TestCheckInClosesLegacyApprovedOpenAttendanceBeforeCreatingNewShift(t *testing.T) {
	loc := clock.DefaultLocation
	yesterdayCheckIn := time.Date(2026, 8, 4, 8, 0, 0, 0, loc)
	approved := string(domain.AttendanceReviewActionApproved)
	legacy := &domain.Attendance{
		ID:           77,
		EmployeeID:   123,
		ProjectID:    55,
		Date:         yesterdayCheckIn,
		CheckInTime:  yesterdayCheckIn,
		CheckInGate:  "Cổng chính",
		ReviewAction: &approved,
	}
	repo := &fakeAttendanceRepo{approvedOpenBefore: legacy}
	now := time.Date(2026, 8, 5, 8, 10, 0, 0, loc)
	svc := newAdminCheckInService(repo, now)
	svc.projectRepo = &fakeProjectRepo{p: &domain.Project{
		ID:                   55,
		IsFlexible:           true,
		GeofenceRadiusMeters: 100,
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10, Lng: 106}},
	}}

	created, err := svc.CheckIn(context.Background(), 123, 55, domain.GeoReading{Lat: 10, Lng: 106})
	if err != nil {
		t.Fatalf("CheckIn returned error: %v", err)
	}
	if created == nil || created.Date.Format("2006-01-02") != "2026-08-05" {
		t.Fatalf("expected today's check-in, got %#v", created)
	}
	wantCheckout := time.Date(2026, 8, 4, 17, 0, 0, 0, loc)
	if legacy.CheckOutTime == nil || !legacy.CheckOutTime.Equal(wantCheckout) {
		t.Fatalf("legacy checkout = %v, want configured shift end %v", legacy.CheckOutTime, wantCheckout)
	}
	if legacy.CheckOutGate == nil || *legacy.CheckOutGate != "admin" {
		t.Fatalf("legacy checkout gate = %v, want admin", legacy.CheckOutGate)
	}
}
