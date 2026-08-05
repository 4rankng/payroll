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

// This file tests the checkout-window auto-reject feature at the service level:
//   - AutoRejectIfExpired idempotency (nil / completed / already-rejected / open)
//   - CheckOut rejects an already-auto-rejected attendance
//   - CheckIn enqueues the auto-reject task at the checkout deadline K+4h after commit
//
// The fakes below stand in for the transaction manager, repositories, clock, and
// the asynq task enqueuer so these run without a database or Redis.

// --- fakes ---

// fakeTransactionManager runs fn immediately and fires after-commit callbacks
// only after fn returns nil, mirroring a real commit.
type fakeTransactionManager struct{}

func (f *fakeTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	txCtx := domain.WithTransactionContext(ctx, &domain.TransactionContext{IsTransactional: true})
	if err := fn(txCtx); err != nil {
		return err
	}
	if tx, ok := domain.GetTransactionFromContext(txCtx); ok {
		tx.RunAfterCommitCallbacks()
	}
	return nil
}

func (f *fakeTransactionManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	txCtx := domain.WithTransactionContext(ctx, &domain.TransactionContext{IsTransactional: true})
	res, err := fn(txCtx)
	if err != nil {
		return res, err
	}
	if tx, ok := domain.GetTransactionFromContext(txCtx); ok {
		tx.RunAfterCommitCallbacks()
	}
	return res, nil
}

// fakeAttendanceRepo embeds the interface (nil) so only the methods overridden
// here are usable; unexpected calls panic — which surfaces as a clear test fail.
type fakeAttendanceRepo struct {
	domain.AttendanceRepository
	byID                   *domain.Attendance
	byDate                 *domain.Attendance
	byDateByDay            map[string]*domain.Attendance
	approvedOpenBefore     *domain.Attendance
	created                *domain.Attendance
	updated                *domain.Attendance
	orphanCandidates       []*domain.Attendance
	markedAutoRejected     bool
	markedIDs              []uint
	quotaCreditedIDs       []uint
	overdueQuotaCandidates []uint
	nextID                 uint
}

func (f *fakeAttendanceRepo) Create(_ context.Context, a *domain.Attendance) error {
	if f.nextID == 0 {
		f.nextID = 1
	}
	a.ID = f.nextID
	f.nextID++
	f.created = a
	f.byID = a
	return nil
}
func (f *fakeAttendanceRepo) GetByID(_ context.Context, _ uint) (*domain.Attendance, error) {
	return f.byID, nil
}
func (f *fakeAttendanceRepo) GetByEmployeeAndDate(_ context.Context, _ uint, date time.Time) (*domain.Attendance, error) {
	if f.byDateByDay != nil {
		return f.byDateByDay[date.Format("2006-01-02")], nil
	}
	return f.byDate, nil
}
func (f *fakeAttendanceRepo) GetApprovedOpenBefore(_ context.Context, _ uint, _ time.Time) (*domain.Attendance, error) {
	return f.approvedOpenBefore, nil
}
func (f *fakeAttendanceRepo) CompleteApprovedOpen(_ context.Context, id uint, checkOutTime time.Time, checkOutGate string) (bool, error) {
	if f.approvedOpenBefore == nil || f.approvedOpenBefore.ID != id || f.approvedOpenBefore.CheckOutTime != nil {
		return false, nil
	}
	f.approvedOpenBefore.CheckOutTime = &checkOutTime
	f.approvedOpenBefore.CheckOutGate = &checkOutGate
	return true, nil
}
func (f *fakeAttendanceRepo) Update(_ context.Context, a *domain.Attendance) error {
	f.updated = a
	return nil
}

// findByID resolves a record by id across the fake's stores (byID for the
// single-record AutoRejectIfExpired tests, orphanCandidates for the sweep test).
func (f *fakeAttendanceRepo) findByID(id uint) *domain.Attendance {
	if f.byID != nil && f.byID.ID == id {
		return f.byID
	}
	if f.byDate != nil && f.byDate.ID == id {
		return f.byDate
	}
	for _, a := range f.byDateByDay {
		if a != nil && a.ID == id {
			return a
		}
	}
	for _, a := range f.orphanCandidates {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// GetOrphanCandidates returns the fake's configured candidates, ignoring the
// time bounds (the sweep test asserts on which IDs get rejected, not on time).
func (f *fakeAttendanceRepo) GetOrphanCandidates(_ context.Context, _, _ time.Time) ([]*domain.Attendance, error) {
	return f.orphanCandidates, nil
}

// MarkAutoRejected mirrors the SQL conditional: it acts only on an open (no
// checkout), unrejected, unreviewed record matching id, mirroring the real
// repo's SQL race guard.
func (f *fakeAttendanceRepo) MarkAutoRejected(_ context.Context, id uint, reason string) (bool, error) {
	rec := f.findByID(id)
	if rec == nil || rec.CheckOutTime != nil || rec.SalaryRejectReason != nil || rec.IsReviewed() {
		return false, nil
	}
	zero := int64(0)
	rec.EarningAmount = &zero
	rec.SalaryRejectReason = &reason
	f.markedAutoRejected = true
	f.markedIDs = append(f.markedIDs, id)
	return true, nil
}

// MarkQuotaCredited mirrors the real repo's conditional: it stamps
// quota_credited_at only when nil and still payable, returning true the first time and false on
// repeats (idempotent). Tracks credited IDs + the new running salary so the
// deferred-credit and Approve-credit paths are testable in-memory. The advance
// payment rows live on the paired fakeAdvancePaymentRepo (salaryByMonth).
func (f *fakeAttendanceRepo) MarkQuotaCredited(_ context.Context, id uint, at time.Time) (bool, error) {
	rec := f.findByID(id)
	if rec == nil || rec.QuotaCreditedAt != nil || rec.EarningAmount == nil || *rec.EarningAmount <= 0 || rec.SalaryRejectReason != nil || rec.IsRejectedByAdmin() {
		return false, nil
	}
	rec.QuotaCreditedAt = &at
	f.quotaCreditedIDs = append(f.quotaCreditedIDs, id)
	return true, nil
}

// GetOverdueQuotaCreditCandidates returns the records configured as overdue
// candidates for the sweep test. The fake ignores the time bounds — sweep tests
// assert on which IDs get credited, not on the cutoff.
func (f *fakeAttendanceRepo) GetOverdueQuotaCreditCandidates(_ context.Context, _ time.Time, _ int) ([]uint, error) {
	return f.overdueQuotaCandidates, nil
}

// fakeAdvancePaymentRepo is the in-memory advance-payment repo for the credit
// path. It tracks salary/max_adv_amount per (employee,project,month) so
// CreditAttendanceQuota can be exercised end-to-end in service-level tests.
// Only the methods the credit path calls are overridden; others panic via the
// nil embedded interface (clear test failure on unexpected use).
type fakeAdvancePaymentRepo struct {
	domain.AdvancePaymentRepository
	rows   []*domain.AdvancePayment
	nextID uint
}

func (f *fakeAdvancePaymentRepo) GetByEmployeeAndMonth(_ context.Context, employeeID uint64, forMonth string) ([]*domain.AdvancePayment, error) {
	var out []*domain.AdvancePayment
	for _, ap := range f.rows {
		if uint64(ap.EmployeeID) == employeeID && ap.ForMonth == forMonth {
			out = append(out, ap)
		}
	}
	return out, nil
}

func (f *fakeAdvancePaymentRepo) Create(_ context.Context, ap *domain.AdvancePayment) error {
	if f.nextID == 0 {
		f.nextID = 1
	}
	ap.ID = f.nextID
	f.nextID++
	f.rows = append(f.rows, ap)
	return nil
}

func (f *fakeAdvancePaymentRepo) AccumulateSalary(_ context.Context, id uint64, earning int64, advancePercentage uint64) error {
	for _, ap := range f.rows {
		if uint64(ap.ID) == id {
			ap.Salary = uint64(int64(ap.Salary) + earning)
			ap.MaxAdvAmount = (ap.Salary * advancePercentage) / 100
			return nil
		}
	}
	return nil
}

func (f *fakeAdvancePaymentRepo) RecomputeActiveCheckInMaxAdvance(_ context.Context, advancePercentage uint64) error {
	for _, ap := range f.rows {
		ap.MaxAdvAmount = (ap.Salary * advancePercentage) / 100
	}
	return nil
}

type fakeSelfCheckInSettingsConfig struct {
	percent uint64
}

func (f fakeSelfCheckInSettingsConfig) GetSelfCheckInAdvancePercentageForUpdate(context.Context) (uint64, error) {
	if f.percent == 0 {
		return domain.DefaultSelfCheckInAdvancePercentage, nil
	}
	return f.percent, nil
}

// salaryFor returns the total credited salary for an employee/month (read path).
func (f *fakeAdvancePaymentRepo) salaryFor(employeeID uint64, forMonth string) uint64 {
	var total uint64
	for _, ap := range f.rows {
		if uint64(ap.EmployeeID) == employeeID && ap.ForMonth == forMonth {
			total += ap.Salary
		}
	}
	return total
}

type fakeProjectRepo struct {
	domain.ProjectRepository
	p *domain.Project
}

func (f *fakeProjectRepo) GetByID(_ context.Context, _ uint) (*domain.Project, error) {
	return f.p, nil
}

type fakeProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
	assignment *domain.ProjectEmployee
}

func (f *fakeProjectEmployeeRepo) GetActiveAssignmentByProjectAndEmployee(_ context.Context, _, _ uint) (*domain.ProjectEmployee, error) {
	return f.assignment, nil
}

type fakePayrateRepo struct {
	domain.PayrateRepository
	pr *domain.Payrate
}

func (f *fakePayrateRepo) GetActiveByProjectAndDate(_ context.Context, _ uint, _ time.Time) (*domain.Payrate, error) {
	return f.pr, nil
}

type fakeEnqCall struct {
	id uint
	at time.Time
}

// fakeTaskEnqueuer records EnqueueAutoRejectCheckout calls and signals `done` so
// tests can synchronize on the after-commit goroutine.
type fakeTaskEnqueuer struct {
	mu          sync.Mutex
	calls       []fakeEnqCall
	creditCalls []fakeEnqCall
	done        chan struct{}
}

func (f *fakeTaskEnqueuer) EnqueueAutoRejectCheckout(id uint, at time.Time) error {
	f.mu.Lock()
	f.calls = append(f.calls, fakeEnqCall{id: id, at: at})
	f.mu.Unlock()
	if f.done != nil {
		f.done <- struct{}{}
	}
	return nil
}

// EnqueueCreditQuota records the deferred quota-credit enqueue (24h hold) so
// CheckOut tests can assert the task is scheduled at checkOutTime + hold.
func (f *fakeTaskEnqueuer) EnqueueCreditQuota(id uint, at time.Time) error {
	f.mu.Lock()
	f.creditCalls = append(f.creditCalls, fakeEnqCall{id: id, at: at})
	f.mu.Unlock()
	if f.done != nil {
		f.done <- struct{}{}
	}
	return nil
}

func (f *fakeTaskEnqueuer) snapshot() []fakeEnqCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]fakeEnqCall, len(f.calls))
	copy(cp, f.calls)
	return cp
}

// creditSnapshot returns the recorded EnqueueCreditQuota calls.
func (f *fakeTaskEnqueuer) creditSnapshot() []fakeEnqCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]fakeEnqCall, len(f.creditCalls))
	copy(cp, f.creditCalls)
	return cp
}

// --- tests ---

func TestAutoRejectIfExpired(t *testing.T) {
	existingReason := "previously rejected"
	completed := time.Date(2026, 6, 22, 17, 0, 0, 0, clock.DefaultLocation)
	approved := string(domain.AttendanceReviewActionApproved)

	cases := []struct {
		name          string
		existing      *domain.Attendance
		wantNoUpdate  bool
		wantEarning   int64
		wantReasonHas string // substring expected in the reject reason; "" => any
	}{
		{name: "nil attendance is no-op", existing: nil, wantNoUpdate: true},
		{name: "completed attendance is no-op", existing: &domain.Attendance{ID: 1, CheckOutTime: &completed}, wantNoUpdate: true},
		{name: "already rejected is no-op", existing: &domain.Attendance{ID: 1, SalaryRejectReason: &existingReason}, wantNoUpdate: true},
		{name: "admin-approved attendance is no-op", existing: &domain.Attendance{ID: 1, ReviewAction: &approved}, wantNoUpdate: true},
		{name: "open attendance is rejected with earning 0", existing: &domain.Attendance{ID: 1}, wantNoUpdate: false, wantEarning: 0, wantReasonHas: "hết hạn"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeAttendanceRepo{byID: tc.existing}
			svc := &AttendanceService{
				attendanceRepo:      repo,
				payrateRepo:         &fakePayrateRepo{},
				projectEmployeeRepo: &fakeProjectEmployeeRepo{},
			}

			if err := svc.AutoRejectIfExpired(context.Background(), 1); err != nil {
				t.Fatalf("AutoRejectIfExpired returned error: %v", err)
			}

			if tc.wantNoUpdate {
				if repo.markedAutoRejected {
					t.Fatalf("expected NO update, but attendance was marked auto-rejected")
				}
				return
			}
			if !repo.markedAutoRejected {
				t.Fatal("expected attendance to be marked auto-rejected, but it was not")
			}
			if repo.byID.EarningAmount == nil || *repo.byID.EarningAmount != tc.wantEarning {
				t.Fatalf("expected earning %d, got %v", tc.wantEarning, repo.byID.EarningAmount)
			}
			if repo.byID.SalaryRejectReason == nil || !strings.Contains(*repo.byID.SalaryRejectReason, tc.wantReasonHas) {
				t.Fatalf("expected reject reason containing %q, got %v", tc.wantReasonHas, repo.byID.SalaryRejectReason)
			}
		})
	}
}

func TestCheckOutRejectsAutoRejectedAttendance(t *testing.T) {
	reason := "Đã hết hạn tan ca"
	repo := &fakeAttendanceRepo{byDate: &domain.Attendance{ID: 7, SalaryRejectReason: &reason}}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		clock:              clock.NewFake(time.Date(2026, 6, 22, 12, 0, 0, 0, clock.DefaultLocation)),
		transactionManager: &fakeTransactionManager{},
	}

	_, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: 10.0, Lng: 106.0}, false)
	if err == nil {
		t.Fatal("expected CheckOut to reject an auto-rejected attendance")
	}
	if !strings.Contains(err.Error(), "tự động từ chối") {
		t.Fatalf("expected auto-reject validation message, got: %v", err)
	}
	if repo.updated != nil {
		t.Fatalf("expected no persistence on a rejected checkout, but attendance was updated")
	}
}

func TestCheckOutRejectsAdminApprovedAttendance(t *testing.T) {
	approved := string(domain.AttendanceReviewActionApproved)
	earning := int64(300000)
	att := &domain.Attendance{
		ID:              8,
		EmployeeID:      123,
		ReviewAction:    &approved,
		EarningAmount:   &earning,
		CheckInTime:     time.Date(2026, 6, 22, 8, 0, 0, 0, clock.DefaultLocation),
		QuotaCreditedAt: timePointer(time.Date(2026, 6, 22, 17, 0, 0, 0, clock.DefaultLocation)),
	}
	repo := &fakeAttendanceRepo{byDate: att}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		clock:              clock.NewFake(time.Date(2026, 6, 22, 18, 0, 0, 0, clock.DefaultLocation)),
		transactionManager: &fakeTransactionManager{},
	}

	_, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: 10.0, Lng: 106.0}, false)
	if err == nil || !strings.Contains(err.Error(), "đã tan ca") {
		t.Fatalf("expected approved shift to reject checkout as completed, got %v", err)
	}
	if repo.updated != nil {
		t.Fatal("expected approved earning not to be overwritten by checkout")
	}
	if att.EarningAmount == nil || *att.EarningAmount != earning {
		t.Fatalf("approved earning = %v, want %d", att.EarningAmount, earning)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func TestCancelCurrentAttendanceMarksOpenShiftRejected(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 7, 7, 8, 30, 0, 0, loc)
	today := time.Date(2026, 7, 7, 0, 0, 0, 0, loc)
	att := &domain.Attendance{
		ID:          19,
		EmployeeID:  123,
		ProjectID:   55,
		Date:        today,
		CheckInTime: checkIn,
		CheckInGate: "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDate: att}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		clock:              clock.NewFake(time.Date(2026, 7, 7, 9, 0, 0, 0, loc)),
		transactionManager: &fakeTransactionManager{},
	}

	got, err := svc.CancelCurrentAttendance(context.Background(), 123)
	if err != nil {
		t.Fatalf("CancelCurrentAttendance returned error: %v", err)
	}
	if got == nil || got.ID != att.ID {
		t.Fatalf("expected canceled attendance %d, got %#v", att.ID, got)
	}
	if !repo.markedAutoRejected {
		t.Fatal("expected repository to mark attendance rejected")
	}
	if got.EarningAmount == nil || *got.EarningAmount != 0 {
		t.Fatalf("expected earning 0, got %v", got.EarningAmount)
	}
	if got.SalaryRejectReason == nil || !strings.Contains(*got.SalaryRejectReason, "vào nhầm ca") {
		t.Fatalf("expected wrong-shift cancel reason, got %v", got.SalaryRejectReason)
	}
	if got.GetStatus(svc.clock.Now()) != domain.AttendanceStatusRejected {
		t.Fatalf("expected rejected status, got %s", got.GetStatus(svc.clock.Now()))
	}
}

func TestCancelCurrentAttendanceRejectsCompletedShift(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 7, 7, 8, 30, 0, 0, loc)
	checkOut := time.Date(2026, 7, 7, 17, 0, 0, 0, loc)
	today := time.Date(2026, 7, 7, 0, 0, 0, 0, loc)
	repo := &fakeAttendanceRepo{byDate: &domain.Attendance{
		ID:           20,
		EmployeeID:   123,
		ProjectID:    55,
		Date:         today,
		CheckInTime:  checkIn,
		CheckOutTime: &checkOut,
		CheckInGate:  "Cổng chính",
	}}
	svc := &AttendanceService{
		attendanceRepo:     repo,
		clock:              clock.NewFake(time.Date(2026, 7, 7, 18, 0, 0, 0, loc)),
		transactionManager: &fakeTransactionManager{},
	}

	_, err := svc.CancelCurrentAttendance(context.Background(), 123)
	if err == nil {
		t.Fatal("expected completed shift cancel to fail")
	}
	if repo.markedAutoRejected {
		t.Fatal("expected completed shift not to be marked rejected")
	}
}

func TestCheckOutAllowsConfirmedNoSalaryOutsideWindow(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 35, 0, 0, loc)
	now := time.Date(2026, 6, 21, 9, 0, 0, 0, loc)
	att := &domain.Attendance{
		ID:          8,
		ProjectID:   55,
		EmployeeID:  123,
		Date:        checkIn,
		CheckInTime: checkIn,
		CheckInGate: "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDate: att}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10.0, Lng: 106.0}},
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}

	_, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: 10.0, Lng: 106.0}, false)
	if err == nil {
		t.Fatal("expected unconfirmed checkout before K-1h to be rejected")
	}
	if !strings.Contains(err.Error(), "Chỉ có thể tan ca từ 16:00 đến 21:00") {
		t.Fatalf("expected checkout window reason, got %q", err.Error())
	}
	if repo.updated != nil {
		t.Fatal("expected rejected checkout not to update attendance")
	}

	updated, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: 10.0, Lng: 106.0}, true)
	if err != nil {
		t.Fatalf("expected confirmed no-salary checkout to succeed, got %v", err)
	}
	if updated.CheckOutTime == nil || !updated.CheckOutTime.Equal(now.In(updated.CheckOutTime.Location())) {
		t.Fatalf("expected checkout time to be set to %v, got %v", now, updated.CheckOutTime)
	}
	if updated.EarningAmount == nil || *updated.EarningAmount != 0 {
		t.Fatalf("expected zero earning, got %v", updated.EarningAmount)
	}
	if updated.SalaryRejectReason == nil || !strings.Contains(*updated.SalaryRejectReason, "không ghi nhận tiền lương") {
		t.Fatalf("expected no-salary reject reason, got %v", updated.SalaryRejectReason)
	}
	if repo.updated != updated {
		t.Fatal("expected attendance update to be persisted")
	}
}

func TestCheckOutRejectsOutsideGeofence(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 35, 0, 0, loc)
	now := time.Date(2026, 6, 21, 17, 30, 0, 0, loc)
	att := &domain.Attendance{
		ID:          9,
		ProjectID:   55,
		EmployeeID:  123,
		Date:        checkIn,
		CheckInTime: checkIn,
		CheckInGate: "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDate: att}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}

	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, -800)
	_, err := svc.CheckOut(context.Background(), 123, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 10}, false)
	if err == nil {
		t.Fatal("expected outside-geofence checkout to be rejected")
	}
	if !strings.Contains(err.Error(), "ngoài khu vực chấm công") {
		t.Fatalf("expected geofence rejection, got %q", err.Error())
	}
	if repo.updated != nil {
		t.Fatal("expected rejected checkout not to update attendance")
	}
}

func TestCheckInAllowsAfterConfirmedNoSalaryCheckoutSameDay(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 6, 21, 20, 5, 0, 0, loc)
	zero := int64(0)
	closedAt := time.Date(2026, 6, 21, 9, 0, 0, 0, loc)
	reason := "Bạn mới vào làm lúc 08:35. Chỉ có thể tan ca từ 16:00 đến 21:00. " + confirmedNoSalaryCheckoutReason
	repo := &fakeAttendanceRepo{byDate: &domain.Attendance{
		ID:                 8,
		ProjectID:          55,
		EmployeeID:         123,
		Date:               time.Date(2026, 6, 21, 0, 0, 0, 0, loc),
		CheckInTime:        time.Date(2026, 6, 21, 8, 35, 0, 0, loc),
		CheckOutTime:       &closedAt,
		EarningAmount:      &zero,
		SalaryRejectReason: &reason,
	}}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			IsFlexible:           true,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10.0, Lng: 106.0}},
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":350000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}

	att, err := svc.CheckIn(context.Background(), 123, 55, domain.GeoReading{Lat: 10.0, Lng: 106.0})
	if err != nil {
		t.Fatalf("expected next same-day check-in to be allowed after confirmed no-salary checkout, got %v", err)
	}
	if att == nil || repo.created != att {
		t.Fatal("expected a new attendance record to be created")
	}
	if !att.CheckInTime.Equal(now.In(att.CheckInTime.Location())) {
		t.Fatalf("expected new check-in time %v, got %v", now, att.CheckInTime)
	}
}

func TestCheckInAllowsAfterAutoRejectedAttendanceSameDay(t *testing.T) {
	loc := clock.DefaultLocation
	now := time.Date(2026, 6, 21, 20, 5, 0, 0, loc)
	zero := int64(0)
	reason := "Đã hết hạn tan ca (Vào làm: 08:35; Tan ca: 17:00 (hạn chót 21:00))"
	repo := &fakeAttendanceRepo{byDate: &domain.Attendance{
		ID:                 8,
		ProjectID:          55,
		EmployeeID:         123,
		Date:               time.Date(2026, 6, 21, 0, 0, 0, 0, loc),
		CheckInTime:        time.Date(2026, 6, 21, 8, 35, 0, 0, loc),
		EarningAmount:      &zero,
		SalaryRejectReason: &reason,
	}}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			IsFlexible:           true,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10.0, Lng: 106.0}},
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":350000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(now),
	}

	att, err := svc.CheckIn(context.Background(), 123, 55, domain.GeoReading{Lat: 10.0, Lng: 106.0})
	if err != nil {
		t.Fatalf("expected next same-day check-in to be allowed after auto-rejected attendance, got %v", err)
	}
	if att == nil || repo.created != att {
		t.Fatal("expected a new attendance record to be created")
	}
	if !att.CheckInTime.Equal(now.In(att.CheckInTime.Location())) {
		t.Fatalf("expected new check-in time %v, got %v", now, att.CheckInTime)
	}
}

func TestGetTodayAttendanceReturnsOpenYesterdayNightShift(t *testing.T) {
	loc := clock.DefaultLocation
	today := time.Date(2026, 6, 22, 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)
	openYesterday := &domain.Attendance{
		ID:          9,
		ProjectID:   55,
		EmployeeID:  123,
		Date:        yesterday,
		CheckInTime: time.Date(2026, 6, 21, 20, 5, 0, 0, loc),
		CheckInGate: "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDateByDay: map[string]*domain.Attendance{
		yesterday.Format("2006-01-02"): openYesterday,
	}}
	svc := &AttendanceService{
		attendanceRepo: repo,
		clock:          clock.NewFake(time.Date(2026, 6, 22, 6, 30, 0, 0, loc)),
	}

	got, err := svc.GetTodayAttendance(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetTodayAttendance returned error: %v", err)
	}
	if got != openYesterday {
		t.Fatalf("expected open previous-day attendance, got %+v", got)
	}
}

func TestGetTodayAttendanceIgnoresAutoRejectedToday(t *testing.T) {
	loc := clock.DefaultLocation
	today := time.Date(2026, 6, 22, 0, 0, 0, 0, loc)
	zero := int64(0)
	reason := "Đã hết hạn tan ca (Vào làm: 08:35; Tan ca: 17:00 (hạn chót 21:00))"
	autoRejectedToday := &domain.Attendance{
		ID:                 10,
		ProjectID:          55,
		EmployeeID:         123,
		Date:               today,
		CheckInTime:        time.Date(2026, 6, 22, 8, 35, 0, 0, loc),
		EarningAmount:      &zero,
		SalaryRejectReason: &reason,
		CheckInGate:        "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDateByDay: map[string]*domain.Attendance{
		today.Format("2006-01-02"): autoRejectedToday,
	}}
	svc := &AttendanceService{
		attendanceRepo: repo,
		clock:          clock.NewFake(time.Date(2026, 6, 22, 20, 5, 0, 0, loc)),
	}

	got, err := svc.GetTodayAttendance(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetTodayAttendance returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected auto-rejected current-day attendance to be ignored, got %+v", got)
	}
}

func TestGetTodayAttendanceIgnoresCompletedYesterday(t *testing.T) {
	loc := clock.DefaultLocation
	today := time.Date(2026, 6, 22, 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)
	checkOut := time.Date(2026, 6, 22, 4, 30, 0, 0, loc)
	completedYesterday := &domain.Attendance{
		ID:           10,
		ProjectID:    55,
		EmployeeID:   123,
		Date:         yesterday,
		CheckInTime:  time.Date(2026, 6, 21, 20, 5, 0, 0, loc),
		CheckOutTime: &checkOut,
		CheckInGate:  "Cổng chính",
	}
	repo := &fakeAttendanceRepo{byDateByDay: map[string]*domain.Attendance{
		yesterday.Format("2006-01-02"): completedYesterday,
	}}
	svc := &AttendanceService{
		attendanceRepo: repo,
		clock:          clock.NewFake(time.Date(2026, 6, 22, 6, 30, 0, 0, loc)),
	}

	got, err := svc.GetTodayAttendance(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetTodayAttendance returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected completed previous-day attendance to be ignored, got %+v", got)
	}
}

func TestGetTodayAttendanceRepairsApprovedOpenYesterday(t *testing.T) {
	loc := clock.DefaultLocation
	today := time.Date(2026, 8, 5, 0, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)
	approved := string(domain.AttendanceReviewActionApproved)
	legacy := &domain.Attendance{
		ID:           77,
		EmployeeID:   123,
		ProjectID:    55,
		Date:         yesterday,
		CheckInTime:  time.Date(2026, 8, 4, 8, 0, 0, 0, loc),
		ReviewAction: &approved,
	}
	repo := &fakeAttendanceRepo{
		byDateByDay:        map[string]*domain.Attendance{yesterday.Format("2006-01-02"): legacy},
		approvedOpenBefore: legacy,
	}
	svc := &AttendanceService{
		attendanceRepo: repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: &domain.ProjectEmployee{
			Position: "Công nhân",
		}},
		payrateRepo: &fakePayrateRepo{pr: &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 8, 5, 8, 10, 0, 0, loc)),
	}

	got, err := svc.GetTodayAttendance(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetTodayAttendance returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected repaired approved record to be absent from today's read model, got %+v", got)
	}
	wantCheckout := time.Date(2026, 8, 4, 17, 0, 0, 0, loc)
	if legacy.CheckOutTime == nil || !legacy.CheckOutTime.Equal(wantCheckout) {
		t.Fatalf("legacy checkout = %v, want %v", legacy.CheckOutTime, wantCheckout)
	}
	if legacy.CheckOutGate == nil || *legacy.CheckOutGate != "admin" {
		t.Fatalf("legacy checkout gate = %v, want admin", legacy.CheckOutGate)
	}
}

func TestCheckInEnqueuesAutoRejectAtDeadline(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 0, 0, 0, loc) // 08:00, inside (07:00, 09:00) for an 08:00 shift

	payrate := &domain.Payrate{Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`)}
	project := &domain.Project{
		ID:                   5,
		IsFlexible:           true,
		GeofenceRadiusMeters: 500,
		GeofenceGates:        []domain.GeofenceGate{{Name: "gate", Lat: 10.0, Lng: 106.0}},
	}
	assignment := &domain.ProjectEmployee{ProjectID: 5, Position: "Công nhân", CheckInEnabled: true}

	enqueuer := &fakeTaskEnqueuer{done: make(chan struct{}, 1)}
	repo := &fakeAttendanceRepo{byDate: nil} // no existing check-in today

	svc := &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: assignment},
		projectRepo:         &fakeProjectRepo{p: project},
		payrateRepo:         &fakePayrateRepo{pr: payrate},
		taskEnqueuer:        enqueuer,
		clock:               clock.NewFake(checkIn),
		transactionManager:  &fakeTransactionManager{},
	}

	if _, err := svc.CheckIn(context.Background(), 999 /*employee*/, 5 /*project*/, domain.GeoReading{Lat: 10.0, Lng: 106.0}); err != nil {
		t.Fatalf("CheckIn returned error: %v", err)
	}

	// The enqueue fires from a goroutine started by RunAfterCommitCallbacks.
	select {
	case <-enqueuer.done:
	case <-time.After(2 * time.Second):
		t.Fatal("auto-reject enqueue did not fire after commit")
	}

	calls := enqueuer.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 enqueue call, got %d", len(calls))
	}
	wantDeadline := time.Date(2026, 6, 22, 21, 0, 0, 0, loc) // shift end 17:00 + checkOutUpperGrace (4h)
	if !calls[0].at.Equal(wantDeadline) {
		t.Fatalf("expected deadline %v, got %v", wantDeadline, calls[0].at)
	}
	if repo.created == nil || calls[0].id != repo.created.ID {
		t.Fatalf("expected enqueue for the created attendance, got id=%d", calls[0].id)
	}
}

func TestAutoRejectSweep(t *testing.T) {
	open1 := &domain.Attendance{ID: 101}
	open2 := &domain.Attendance{ID: 102}
	existingReason := "previously rejected"
	alreadyRejected := &domain.Attendance{ID: 103, SalaryRejectReason: &existingReason}
	completedAt := time.Date(2026, 6, 22, 17, 0, 0, 0, clock.DefaultLocation)
	completed := &domain.Attendance{ID: 104, CheckOutTime: &completedAt}
	approvedAction := string(domain.AttendanceReviewActionApproved)
	approvedEarning := int64(300000)
	adminApproved := &domain.Attendance{ID: 105, ReviewAction: &approvedAction, EarningAmount: &approvedEarning}

	repo := &fakeAttendanceRepo{orphanCandidates: []*domain.Attendance{open1, alreadyRejected, open2, completed, adminApproved}}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		payrateRepo:         &fakePayrateRepo{},
		projectEmployeeRepo: &fakeProjectEmployeeRepo{},
		clock:               clock.NewFake(time.Date(2026, 6, 23, 12, 0, 0, 0, clock.DefaultLocation)),
	}

	n, err := svc.AutoRejectSweep(context.Background())
	if err != nil {
		t.Fatalf("AutoRejectSweep returned error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 records finalized, got %d (markedIDs=%v)", n, repo.markedIDs)
	}
	// Open records finalized with earning 0 and a reject reason containing "hết hạn".
	for _, att := range []*domain.Attendance{open1, open2} {
		if att.EarningAmount == nil || *att.EarningAmount != 0 {
			t.Errorf("id %d: expected earning 0, got %v", att.ID, att.EarningAmount)
		}
		if att.SalaryRejectReason == nil || !strings.Contains(*att.SalaryRejectReason, "hết hạn") {
			t.Errorf("id %d: expected reject reason, got %v", att.ID, att.SalaryRejectReason)
		}
	}
	// Already-rejected record untouched (earning stays nil, reason unchanged).
	if alreadyRejected.EarningAmount != nil {
		t.Errorf("already-rejected id %d: expected earning nil, got %v", alreadyRejected.ID, alreadyRejected.EarningAmount)
	}
	if alreadyRejected.SalaryRejectReason == nil || *alreadyRejected.SalaryRejectReason != existingReason {
		t.Errorf("already-rejected id %d: reason should be unchanged", alreadyRejected.ID)
	}
	// Completed record untouched (no reject reason written over a valid checkout).
	if completed.SalaryRejectReason != nil {
		t.Errorf("completed id %d: should not get a reject reason", completed.ID)
	}
	if adminApproved.SalaryRejectReason != nil || adminApproved.EarningAmount == nil || *adminApproved.EarningAmount != approvedEarning {
		t.Errorf("admin-approved id %d: decision and earning should be preserved", adminApproved.ID)
	}
}

// TestFormatAutoRejectReason covers the dynamic Vietnamese message built when
// the configured shift end (K) and the grace-window upper bound (K+4h) are
// both available. It anchors the employee-visible contract of the new format.
func TestFormatAutoRejectReason(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 0, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 17, 0, 0, 0, loc)

	got := formatAutoRejectReason(checkIn, shiftEnd)
	want := "Đã hết hạn tan ca (Vào làm: 08:00; Tan ca: 17:00 (hạn chót 21:00))"
	if got != want {
		t.Fatalf("formatAutoRejectReason:\n got: %q\nwant: %q", got, want)
	}
}

// TestAutoRejectIfExpiredFormatsDynamicReason exercises the path where the
// configured shift is resolvable at reject time, so the reason must include
// the actual check-in / shift-end / deadline (not just the generic fallback).
func TestAutoRejectIfExpiredFormatsDynamicReason(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 0, 0, 0, loc)
	open := &domain.Attendance{
		ID:          1,
		ProjectID:   5,
		EmployeeID:  999,
		CheckInTime: checkIn,
		Date:        checkIn,
	}
	payrate := &domain.Payrate{Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`)}
	assignment := &domain.ProjectEmployee{ProjectID: 5, EmployeeID: 999, Position: "Công nhân"}

	repo := &fakeAttendanceRepo{byID: open}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		payrateRepo:         &fakePayrateRepo{pr: payrate},
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: assignment},
	}

	if err := svc.AutoRejectIfExpired(context.Background(), 1); err != nil {
		t.Fatalf("AutoRejectIfExpired returned error: %v", err)
	}
	if open.SalaryRejectReason == nil {
		t.Fatal("expected SalaryRejectReason to be set")
	}
	want := "Đã hết hạn tan ca (Vào làm: 08:00; Tan ca: 17:00 (hạn chót 21:00))"
	if *open.SalaryRejectReason != want {
		t.Fatalf("reject reason:\n got: %q\nwant: %q", *open.SalaryRejectReason, want)
	}
}

// TestAutoRejectSweepFormatsDynamicReason exercises the same dynamic-format
// path in the fallback sweep — each candidate must use its own check-in time
// and the configured shift end, not a shared constant.
func TestAutoRejectSweepFormatsDynamicReason(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 0, 0, 0, loc)
	open := &domain.Attendance{
		ID:          201,
		ProjectID:   5,
		EmployeeID:  999,
		CheckInTime: checkIn,
		Date:        checkIn,
	}
	payrate := &domain.Payrate{Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`)}
	assignment := &domain.ProjectEmployee{ProjectID: 5, EmployeeID: 999, Position: "Công nhân"}

	repo := &fakeAttendanceRepo{orphanCandidates: []*domain.Attendance{open}}
	svc := &AttendanceService{
		attendanceRepo:      repo,
		payrateRepo:         &fakePayrateRepo{pr: payrate},
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: assignment},
		clock:               clock.NewFake(time.Date(2026, 6, 23, 12, 0, 0, 0, loc)),
	}

	if _, err := svc.AutoRejectSweep(context.Background()); err != nil {
		t.Fatalf("AutoRejectSweep returned error: %v", err)
	}
	if open.SalaryRejectReason == nil {
		t.Fatal("expected SalaryRejectReason to be set")
	}
	want := "Đã hết hạn tan ca (Vào làm: 08:00; Tan ca: 17:00 (hạn chót 21:00))"
	if *open.SalaryRejectReason != want {
		t.Fatalf("reject reason:\n got: %q\nwant: %q", *open.SalaryRejectReason, want)
	}
}
