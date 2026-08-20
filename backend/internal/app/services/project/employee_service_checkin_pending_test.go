package project

import (
	"context"
	"testing"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// The toggle's deferred-activation rules (2026-08-20 plan):
//   - enable → pending, activates day 1 of NEXT month (strict)
//   - enable on the 1st → still next month
//   - disable while pending → cancels pending, NO quota zeroing
//   - disable while active → instant + ZeroOutQuota (regression)
//   - duplicate enable (active or pending) → no-op

type checkinPendingAssignmentRepo struct {
	domain.ProjectEmployeeRepository
	assignment *domain.ProjectEmployee
	saved      []*domain.ProjectEmployee
}

func (f *checkinPendingAssignmentRepo) GetActiveAssignmentByProjectAndEmployee(_ context.Context, _, _ uint) (*domain.ProjectEmployee, error) {
	return f.assignment, nil
}

func (f *checkinPendingAssignmentRepo) GetEmployeesWithPendingCheckInEnable(_ context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error) {
	if f.assignment == nil || !f.assignment.HasPendingCheckInEnable() {
		return nil, nil
	}
	if f.assignment.CheckInEffectiveFrom != nil && f.assignment.CheckInEffectiveFrom.After(effectiveDate) {
		return nil, nil
	}
	return []*domain.ProjectEmployee{f.assignment}, nil
}

func (f *checkinPendingAssignmentRepo) GetByID(_ context.Context, _ uint) (*domain.ProjectEmployee, error) {
	return f.assignment, nil
}

func (f *checkinPendingAssignmentRepo) Update(_ context.Context, a *domain.ProjectEmployee) error {
	f.saved = append(f.saved, a)
	return nil
}

type checkinPendingAdvanceRepo struct {
	domain.AdvancePaymentRepository
	zeroedMonths []string
}

func (f *checkinPendingAdvanceRepo) ZeroOutQuota(_ context.Context, _, _ uint, month string) error {
	f.zeroedMonths = append(f.zeroedMonths, month)
	return nil
}

func (f *checkinPendingAdvanceRepo) BatchZeroOutQuota(_ context.Context, _ uint, _ []uint, month string) error {
	f.zeroedMonths = append(f.zeroedMonths, month)
	return nil
}

type checkinPendingPayrateRepo struct {
	domain.PayrateRepository
	hasRates bool
}

func (f *checkinPendingPayrateRepo) GetActiveByProject(_ context.Context, _ uint, _ time.Time) ([]*domain.Payrate, error) {
	if f.hasRates {
		return []*domain.Payrate{{ID: 1}}, nil
	}
	return nil, nil
}

func newCheckinPendingService(repo domain.ProjectEmployeeRepository, advanceRepo domain.AdvancePaymentRepository) *ProjectEmployeeService {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return NewProjectEmployeeService(
		repo,
		nil, // employeeRepo
		nil, // employeeUserRepo
		nil, // projectRepo
		nil, // timesheetRepo
		nil, // auditLogRepo
		infrastructure.NewTransactionManager(db),
		advanceRepo,
		&checkinPendingPayrateRepo{hasRates: true},
		nil, // notification
		nil, // eventBus
		nil, // payCycleEventPublisher
		nil, // timesheetRecalculator
		nil, // cache
	)
}

func newPendingAssignment() *domain.ProjectEmployee {
	return &domain.ProjectEmployee{
		ID:              1,
		ProjectID:       5,
		EmployeeID:      99,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
		StartDate:       time.Date(2026, 1, 1, 0, 0, 0, 0, clock.DefaultLocation),
	}
}

func setFakeClock(t *testing.T, now time.Time) {
	t.Helper()
	clock.SetGlobal(clock.NewFake(now))
	t.Cleanup(func() { clock.SetGlobal(clock.New()) })
}

func TestToggleCheckInEnableWritesPendingWithNextMonthEffective(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 29, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("toggle enable: %v", err)
	}

	a := repo.assignment
	if a.CheckInEnabled {
		t.Error("CheckInEnabled must stay false while pending")
	}
	if !a.HasPendingCheckInEnable() {
		t.Fatal("expected pending enable")
	}
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)
	if a.CheckInEffectiveFrom == nil || !a.CheckInEffectiveFrom.Equal(want) {
		t.Errorf("effective = %v, want %v (day 1 of next month)", a.CheckInEffectiveFrom, want)
	}
}

func TestToggleCheckInEnableOnDay1StillDefersToNextMonth(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 1, 0, 30, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("toggle enable: %v", err)
	}

	want := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)
	got := repo.assignment.CheckInEffectiveFrom
	if got == nil || !got.Equal(want) {
		t.Errorf("effective = %v, want %v (strict next-month even on the 1st)", got, want)
	}
}

func TestToggleCheckInDisableWhilePendingCancelsWithoutZeroing(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	advanceRepo := &checkinPendingAdvanceRepo{}
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, advanceRepo)

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, false, 1); err != nil {
		t.Fatalf("disable: %v", err)
	}

	a := repo.assignment
	if a.CheckInEnabled || a.HasPendingCheckInEnable() {
		t.Errorf("pending must be cleared and stay inactive, got enabled=%v pending=%v", a.CheckInEnabled, a.HasPendingCheckInEnable())
	}
	if len(advanceRepo.zeroedMonths) != 0 {
		t.Errorf("ZeroOutQuota must NOT fire for a never-active enable, got %v", advanceRepo.zeroedMonths)
	}
}

func TestToggleCheckInDisableActiveStillZeroesQuota(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	advanceRepo := &checkinPendingAdvanceRepo{}
	a := newPendingAssignment()
	a.CheckInEnabled = true
	repo := &checkinPendingAssignmentRepo{assignment: a}
	svc := newCheckinPendingService(repo, advanceRepo)

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, false, 1); err != nil {
		t.Fatalf("disable: %v", err)
	}

	if a.CheckInEnabled {
		t.Error("active disable must flip CheckInEnabled off immediately")
	}
	if len(advanceRepo.zeroedMonths) != 1 || advanceRepo.zeroedMonths[0] != "2026-08" {
		t.Errorf("ZeroOutQuota(current month) must fire exactly once, got %v", advanceRepo.zeroedMonths)
	}
}

func TestToggleCheckInDuplicateEnableIsNoop(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("first enable: %v", err)
	}
	savesAfterFirst := len(repo.saved)
	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("second enable: %v", err)
	}
	if len(repo.saved) != savesAfterFirst {
		t.Error("duplicate enable (already pending) must not write again")
	}

	// Active row: enable is also a no-op.
	active := newPendingAssignment()
	active.CheckInEnabled = true
	repo2 := &checkinPendingAssignmentRepo{assignment: active}
	svc2 := newCheckinPendingService(repo2, &checkinPendingAdvanceRepo{})
	if err := svc2.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("enable on active: %v", err)
	}
	if len(repo2.saved) != 0 {
		t.Error("enable on already-active row must not write")
	}
}

func TestBulkToggleCheckInPendingPaths(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	advanceRepo := &checkinPendingAdvanceRepo{}
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, advanceRepo)

	// Bulk enable → pending
	if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99}, true, 1); err != nil {
		t.Fatalf("bulk enable: %v", err)
	}
	if !repo.assignment.HasPendingCheckInEnable() {
		t.Fatal("bulk enable must write pending")
	}

	// Bulk disable while pending → cancel, no zeroing
	if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99}, false, 1); err != nil {
		t.Fatalf("bulk disable: %v", err)
	}
	if repo.assignment.HasPendingCheckInEnable() || repo.assignment.CheckInEnabled {
		t.Error("bulk disable must cancel pending without activating")
	}
	if len(advanceRepo.zeroedMonths) != 0 {
		t.Errorf("bulk disable of pending must not zero quota, got %v", advanceRepo.zeroedMonths)
	}

	// Bulk enable then bulk disable after activation → zeroing fires
	repo.assignment.CheckInEnabled = true
	if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99}, false, 1); err != nil {
		t.Fatalf("bulk disable active: %v", err)
	}
	if len(advanceRepo.zeroedMonths) != 1 {
		t.Errorf("bulk disable of active row must zero quota, got %v", advanceRepo.zeroedMonths)
	}
}

func TestCancelPendingCheckInEnableService(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if err := svc.CancelPendingCheckInEnable(context.Background(), 1, 1); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if repo.assignment.HasPendingCheckInEnable() {
		t.Error("cancel must clear pending")
	}

	// Second cancel → validation error
	if err := svc.CancelPendingCheckInEnable(context.Background(), 1, 1); err == nil {
		t.Error("cancel with nothing pending must fail")
	}
}

func TestApplyPendingCheckInEnablesAppliesDueRows(t *testing.T) {
	// Effective far in the past → due today regardless of the real clock.
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := repo.assignment.RequestCheckInEnable(time.Date(2020, 1, 1, 0, 0, 0, 0, clock.DefaultLocation)); err != nil {
		t.Fatalf("request: %v", err)
	}

	if err := svc.ApplyPendingCheckInEnables(context.Background()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !repo.assignment.CheckInEnabled {
		t.Error("due pending must activate")
	}
	if repo.assignment.HasPendingCheckInEnable() {
		t.Error("activation must clear pending fields")
	}
}

func TestToggleCheckInEnableRequiresPayrate(t *testing.T) {
	setFakeClock(t, time.Date(2026, 8, 20, 10, 0, 0, 0, clock.DefaultLocation))
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	svc := NewProjectEmployeeService(
		&checkinPendingAssignmentRepo{assignment: newPendingAssignment()},
		nil, nil, nil, nil, nil,
		infrastructure.NewTransactionManager(db),
		&checkinPendingAdvanceRepo{},
		&checkinPendingPayrateRepo{hasRates: false},
		nil, nil, nil, nil, nil,
	)

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, 1); err == nil {
		t.Fatal("enable without project payrate must be rejected")
	}
}
