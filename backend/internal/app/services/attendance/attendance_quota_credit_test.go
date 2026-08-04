package attendance

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// This file tests the 24h quota-credit hold:
//   - CreditAttendanceQuota idempotency + zero-earning/missing no-ops
//   - forMonth is derived from the check-out DATE (not credit-time now), so a
//     31-Jul 23:55 check-out credits to July even when the timer fires in August
//   - CheckOut no longer banks salary inline; it defers a credit task at +24h
//   - Approve credits immediately (no hold) and stamps quota_credited_at
//   - CreditOverduePendingQuota sweep banks each overdue candidate once
//
// Fakes come from attendance_auto_reject_test.go in the same package.

type staleCreditAfterRejectRepo struct {
	fakeAttendanceRepo
}

func (f *staleCreditAfterRejectRepo) MarkQuotaCredited(_ context.Context, id uint, _ time.Time) (bool, error) {
	rec := f.findByID(id)
	rejected := string(domain.AttendanceReviewActionRejected)
	reason := "admin rejected while credit was waiting"
	zero := int64(0)
	rec.ReviewAction = &rejected
	rec.SalaryRejectReason = &reason
	rec.EarningAmount = &zero
	return false, nil
}

func TestCreditAttendanceQuotaIdempotent(t *testing.T) {
	loc := clock.DefaultLocation
	co := time.Date(2026, 6, 22, 17, 0, 0, 0, loc)
	earn := int64(300000)
	att := &domain.Attendance{
		ID:            7,
		ProjectID:     55,
		EmployeeID:    123,
		Date:          time.Date(2026, 6, 22, 0, 0, 0, 0, loc),
		CheckInTime:   time.Date(2026, 6, 22, 8, 35, 0, 0, loc),
		CheckOutTime:  &co,
		EarningAmount: &earn,
	}
	repo := &fakeAttendanceRepo{byID: att}
	advRepo := &fakeAdvancePaymentRepo{}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		advancePaymentRepo: advRepo,
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 6, 23, 17, 0, 0, 0, loc)),
	}

	banked, err := svc.CreditAttendanceQuota(context.Background(), 7)
	if err != nil {
		t.Fatalf("first credit returned error: %v", err)
	}
	if !banked {
		t.Fatal("expected first credit to bank the earning")
	}
	if got := advRepo.salaryFor(123, "2026-06"); got != 300000 {
		t.Fatalf("expected 300000 credited to 2026-06, got %d", got)
	}
	if att.QuotaCreditedAt == nil {
		t.Fatal("expected quota_credited_at to be stamped after credit")
	}

	// Second credit must be a safe no-op — no double-banking.
	banked2, err := svc.CreditAttendanceQuota(context.Background(), 7)
	if err != nil {
		t.Fatalf("second credit returned error: %v", err)
	}
	if banked2 {
		t.Fatal("expected second credit to be a no-op (already credited)")
	}
	if got := advRepo.salaryFor(123, "2026-06"); got != 300000 {
		t.Fatalf("expected salary still 300000 after repeated credit, got %d", got)
	}
}

func TestCreditAttendanceQuotaUsesConfiguredPercent(t *testing.T) {
	loc := clock.DefaultLocation
	co := time.Date(2026, 6, 22, 17, 0, 0, 0, loc)
	earn := int64(300000)
	att := &domain.Attendance{
		ID:            17,
		ProjectID:     55,
		EmployeeID:    321,
		Date:          time.Date(2026, 6, 22, 0, 0, 0, 0, loc),
		CheckInTime:   time.Date(2026, 6, 22, 8, 35, 0, 0, loc),
		CheckOutTime:  &co,
		EarningAmount: &earn,
	}
	repo := &fakeAttendanceRepo{byID: att}
	advRepo := &fakeAdvancePaymentRepo{}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		advancePaymentRepo: advRepo,
		settingsConfig:     fakeSelfCheckInSettingsConfig{percent: 85},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 6, 23, 17, 0, 0, 0, loc)),
	}

	banked, err := svc.CreditAttendanceQuota(context.Background(), 17)
	if err != nil {
		t.Fatalf("credit returned error: %v", err)
	}
	if !banked {
		t.Fatal("expected first credit to bank the earning")
	}
	if len(advRepo.rows) != 1 {
		t.Fatalf("expected 1 advance payment row, got %d", len(advRepo.rows))
	}
	if advRepo.rows[0].MaxAdvAmount != 255000 {
		t.Fatalf("expected max advance 255000 at 85%%, got %d", advRepo.rows[0].MaxAdvAmount)
	}
}

func TestCreditAttendanceQuotaMonthBoundary(t *testing.T) {
	loc := clock.DefaultLocation
	// Checkout at 31-Jul 23:55; the 24h timer fires in August. The earning must
	// land in July's quota (the check-out month), not August's.
	co := time.Date(2026, 7, 31, 23, 55, 0, 0, loc)
	earn := int64(300000)
	att := &domain.Attendance{
		ID:            7,
		ProjectID:     55,
		EmployeeID:    123,
		Date:          time.Date(2026, 7, 31, 0, 0, 0, 0, loc),
		CheckInTime:   time.Date(2026, 7, 31, 20, 0, 0, 0, loc),
		CheckOutTime:  &co,
		EarningAmount: &earn,
	}
	repo := &fakeAttendanceRepo{byID: att}
	advRepo := &fakeAdvancePaymentRepo{}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		advancePaymentRepo: advRepo,
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 8, 1, 23, 55, 0, 0, loc)),
	}

	if _, err := svc.CreditAttendanceQuota(context.Background(), 7); err != nil {
		t.Fatalf("credit returned error: %v", err)
	}
	if got := advRepo.salaryFor(123, "2026-07"); got != 300000 {
		t.Fatalf("expected 300000 credited to check-out month 2026-07, got %d", got)
	}
	if got := advRepo.salaryFor(123, "2026-08"); got != 0 {
		t.Fatalf("expected nothing credited to credit-time month 2026-08, got %d", got)
	}
}

func TestCreditAttendanceQuotaNoOpsForZeroEarningAndMissing(t *testing.T) {
	loc := clock.DefaultLocation
	now := time.Date(2026, 6, 23, 9, 0, 0, 0, loc)

	// Missing record → no-op, no error.
	missingSvc := &AttendanceService{
		attendanceRepo:     &fakeAttendanceRepo{}, // byID nil
		advancePaymentRepo: &fakeAdvancePaymentRepo{},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}
	if banked, err := missingSvc.CreditAttendanceQuota(context.Background(), 99); err != nil || banked {
		t.Fatalf("expected no-op for missing record, got banked=%v err=%v", banked, err)
	}

	// Zero earning → no-op, no error, no stamp.
	zero := int64(0)
	att := &domain.Attendance{
		ID:            5,
		ProjectID:     55,
		EmployeeID:    123,
		EarningAmount: &zero,
	}
	zeroSvc := &AttendanceService{
		attendanceRepo:     &fakeAttendanceRepo{byID: att},
		advancePaymentRepo: &fakeAdvancePaymentRepo{},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}
	if banked, err := zeroSvc.CreditAttendanceQuota(context.Background(), 5); err != nil || banked {
		t.Fatalf("expected no-op for zero earning, got banked=%v err=%v", banked, err)
	}
	if att.QuotaCreditedAt != nil {
		t.Fatal("expected quota_credited_at left nil when there is nothing to credit")
	}
}

func TestCreditAttendanceQuotaDoesNotBankWhenConcurrentRejectWins(t *testing.T) {
	loc := clock.DefaultLocation
	earn := int64(300000)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date:          time.Date(2026, 6, 22, 0, 0, 0, 0, loc),
		CheckInTime:   time.Date(2026, 6, 22, 8, 0, 0, 0, loc),
		EarningAmount: &earn,
	}
	repo := &staleCreditAfterRejectRepo{fakeAttendanceRepo: fakeAttendanceRepo{byID: att}}
	advRepo := &fakeAdvancePaymentRepo{}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		advancePaymentRepo: advRepo,
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 6, 23, 18, 0, 0, 0, loc)),
	}

	banked, err := svc.CreditAttendanceQuota(context.Background(), 7)
	if err != nil {
		t.Fatalf("credit after concurrent reject: %v", err)
	}
	if banked {
		t.Fatal("expected stale credit to lose after rejection")
	}
	if got := advRepo.salaryFor(123, "2026-06"); got != 0 {
		t.Fatalf("employee quota increased by %d after rejection", got)
	}
	if att.QuotaCreditedAt != nil {
		t.Fatal("rejected attendance must not be stamped as credited")
	}
}

func TestCheckOutDefersQuotaCredit(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 35, 0, 0, loc)
	now := time.Date(2026, 6, 21, 17, 0, 0, 0, loc) // 17:00 — within [16:00, 21:00]
	att := &domain.Attendance{
		ID:          8,
		ProjectID:   55,
		EmployeeID:  123,
		Date:        checkIn,
		CheckInTime: checkIn,
		CheckInGate: "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDate: att}
	advRepo := &fakeAdvancePaymentRepo{}
	enq := &fakeTaskEnqueuer{done: make(chan struct{}, 1)}
	svc := &AttendanceService{
		attendanceRepo: repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{
			Position: "Công nhân", CheckInEnabled: true,
		}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10.0, Lng: 106.0}},
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}},
		advancePaymentRepo: advRepo,
		transactionManager: &fakeTransactionManager{},
		taskEnqueuer:       enq,
		clock:              clock.NewFake(now),
	}

	res, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: 10.0, Lng: 106.0}, false)
	if err != nil {
		t.Fatalf("CheckOut returned error: %v", err)
	}
	// The after-commit callback fires the enqueue in a goroutine, so wait on
	// the done signal before asserting on the recorded calls.
	<-enq.done
	if res.EarningAmount == nil || *res.EarningAmount != 300000 {
		t.Fatalf("expected earning 300000 recorded on attendance, got %v", res.EarningAmount)
	}

	// The earning must NOT be banked into the quota pool at check-out — it sits
	// in the 24h hold (quota_credited_at stays nil, salary still 0).
	if got := advRepo.salaryFor(123, "2026-06"); got != 0 {
		t.Fatalf("expected no immediate salary credit (24h hold), got %d", got)
	}
	if att.QuotaCreditedAt != nil {
		t.Fatal("expected quota_credited_at nil during the 24h hold")
	}

	// A deferred credit task is enqueued at checkOutTime + QuotaCreditHoldDuration.
	calls := enq.creditSnapshot()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 credit-quota enqueue, got %d", len(calls))
	}
	if calls[0].id != 8 {
		t.Fatalf("expected enqueue for attendance 8, got %d", calls[0].id)
	}
	wantAt := now.Add(domain.QuotaCreditHoldDuration)
	if !calls[0].at.Equal(wantAt) {
		t.Fatalf("expected credit task fire-at %v, got %v", wantAt, calls[0].at)
	}
}

func TestApproveCreditsQuotaImmediately(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	zero := int64(0)
	priorReason := "Tự động huỷ do quá hạn tan ca"
	att := &domain.Attendance{
		ID:                 7,
		ProjectID:          55,
		EmployeeID:         123,
		Date:               checkIn,
		CheckInTime:        checkIn,
		CheckInGate:        "Cổng chính",
		EarningAmount:      &zero,
		SalaryRejectReason: &priorReason,
	}
	svc, _ := newReviewService(att, nil, nil)

	if _, err := svc.Approve(context.Background(), 7, 42, "Khách xác nhận làm đủ ca"); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}

	adv := svc.advancePaymentRepo.(*fakeAdvancePaymentRepo)
	// Admin approval credits immediately — no 24h hold.
	if got := adv.salaryFor(123, "2026-06"); got != 300000 {
		t.Fatalf("expected approved earning 300000 credited immediately, got %d", got)
	}
	if att.QuotaCreditedAt == nil {
		t.Fatal("expected quota_credited_at stamped on immediate admin-approve credit")
	}
}

func TestCreditOverduePendingQuotaSweep(t *testing.T) {
	loc := clock.DefaultLocation
	earn := int64(250000)
	co := time.Date(2026, 6, 20, 17, 0, 0, 0, loc)
	att := &domain.Attendance{
		ID:            10,
		ProjectID:     55,
		EmployeeID:    200,
		Date:          time.Date(2026, 6, 20, 0, 0, 0, 0, loc),
		CheckInTime:   time.Date(2026, 6, 20, 8, 0, 0, 0, loc),
		CheckOutTime:  &co,
		EarningAmount: &earn,
	}
	repo := &fakeAttendanceRepo{byID: att, overdueQuotaCandidates: []uint{10}}
	advRepo := &fakeAdvancePaymentRepo{}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		advancePaymentRepo: advRepo,
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 6, 22, 9, 0, 0, 0, loc)),
	}

	credited, err := svc.CreditOverduePendingQuota(context.Background())
	if err != nil {
		t.Fatalf("sweep returned error: %v", err)
	}
	if credited != 1 {
		t.Fatalf("expected 1 record banked by sweep, got %d", credited)
	}
	if got := advRepo.salaryFor(200, "2026-06"); got != 250000 {
		t.Fatalf("expected sweep to bank 250000, got %d", got)
	}

	// Second sweep pass is a no-op (already credited) — idempotent.
	credited2, err := svc.CreditOverduePendingQuota(context.Background())
	if err != nil {
		t.Fatalf("second sweep returned error: %v", err)
	}
	if credited2 != 0 {
		t.Fatalf("expected 0 on re-sweep (already credited), got %d", credited2)
	}
	if got := advRepo.salaryFor(200, "2026-06"); got != 250000 {
		t.Fatalf("expected salary still 250000 after re-sweep, got %d", got)
	}
}
