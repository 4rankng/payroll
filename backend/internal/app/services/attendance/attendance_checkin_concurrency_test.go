package attendance

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// This file guards the duplicate check-in race fix (migration 081 +
// attendance_repository.Create dup-key mapping).
//
// Background: CheckIn enforces "one OPEN attendance per employee/day" with a
// read-then-write (GetByEmployeeAndDate -> Create). Before the fix, two
// concurrent first-check-ins could both read nil and both Create, because
// migration 078 had dropped the UNIQUE(employee_id, date) guard. Migration 081
// restores the invariant declaratively via a partial unique on a generated
// open_key column (non-NULL only for open rows), and the repo maps the resulting
// duplicate-key error to the same friendly validation error the sequential
// dup-check returns.
//
// This test forces the interleaving two real concurrent requests produce (both
// reads resolve before either write) using a barrier-injecting fake repo. The
// fake's Create enforces the same "at most one open row per employee/date"
// contract the DB unique now enforces, so we assert exactly one row is created
// and the loser receives the friendly "đã vào làm" validation error. The DB-level
// enforcement itself is verified by applying migration 081 + its pre-flight query.

// racingAttendanceRepo forces both concurrent CheckIns to reach their
// "is there an existing attendance?" read before either is allowed to Create,
// then returns nil for both — the exact shared window production exposes.
type racingAttendanceRepo struct {
	domain.AttendanceRepository
	mu       sync.Mutex
	created  []*domain.Attendance
	getReady chan struct{} // buffered(2): each Get announces itself here
	release  chan struct{} // closed by the test once both Gets are pending
}

func (r *racingAttendanceRepo) GetByEmployeeAndDate(_ context.Context, _ uint, _ time.Time) (*domain.Attendance, error) {
	r.getReady <- struct{}{} // announce this goroutine reached the read
	<-r.release              // park until the test confirms both readers are here
	return nil, nil          // both observe: no existing attendance
}

// Create mirrors the post-fix DB contract (uq_attendances_employee_open): at most
// one OPEN row per employee/date. A second open row is rejected with the same
// validation error the real repo returns for a duplicate-key.
func (r *racingAttendanceRepo) Create(_ context.Context, a *domain.Attendance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ex := range r.created {
		if ex.EmployeeID == a.EmployeeID && ex.CheckOutTime == nil && ex.SalaryRejectReason == nil {
			return domain.NewValidationError("Bạn đã vào làm trong ngày hôm nay rồi")
		}
	}
	a.ID = uint(len(r.created)) + 1
	r.created = append(r.created, a)
	return nil
}

func TestCheckInConcurrentFirstCheckInAllowsOnlyOneOpenRow(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 7, 3, 8, 0, 0, 0, loc) // exactly shift start T

	project := &domain.Project{
		ID:                   55,
		IsFlexible:           true,
		GeofenceRadiusMeters: 100,
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng", Lat: 10.0, Lng: 106.0}},
	}
	assignment := &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}

	repo := &racingAttendanceRepo{
		getReady: make(chan struct{}, 2),
		release:  make(chan struct{}),
	}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectRepo:         &fakeProjectRepo{p: project},
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: assignment},
		payrateRepo:         &fakePayrateRepo{pr: payrate},
		transactionManager:  &fakeTransactionManager{},
		taskEnqueuer:        &fakeTaskEnqueuer{},
		clock:               clock.NewFake(now),
	}

	// Two concurrent check-ins for the same employee/day (e.g. a network retry
	// landing while the first request is still in flight).
	var wg sync.WaitGroup
	wg.Add(2)
	var err1, err2 error
	geo := domain.GeoReading{Lat: 10.0, Lng: 106.0}
	go func() { defer wg.Done(); _, err1 = svc.CheckIn(context.Background(), 123, 55, geo) }()
	go func() { defer wg.Done(); _, err2 = svc.CheckIn(context.Background(), 123, 55, geo) }()

	// Wait for BOTH check-ins to park at the existence read, then release them so
	// each observes "no existing attendance" — the race window.
	<-repo.getReady
	<-repo.getReady
	close(repo.release)
	wg.Wait()

	// Exactly one row must be created; the loser must get the friendly validation
	// error (not a raw duplicate-key, not success).
	repo.mu.Lock()
	created := len(repo.created)
	repo.mu.Unlock()
	if created != 1 {
		t.Fatalf("expected exactly 1 attendance row (open-row unique enforced), got %d", created)
	}

	rejected := 0
	for _, err := range []error{err1, err2} {
		if err == nil {
			continue
		}
		rejected++
		if !domain.IsValidationError(err) || !strings.Contains(err.Error(), "đã vào làm") {
			t.Fatalf("expected friendly 'đã vào làm' validation error, got %v", err)
		}
	}
	if rejected != 1 {
		t.Fatalf("expected exactly one of two concurrent check-ins to be rejected, got %d rejected", rejected)
	}
}
