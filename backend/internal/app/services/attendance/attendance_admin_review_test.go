package attendance

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// This file tests the admin dispute-resolution path (Approve/Reject) at the
// service level. The fakes come from attendance_auto_reject_test.go in the same
// package; reviewFakeAttendanceRepo embeds fakeAttendanceRepo and overrides
// MarkAdminReviewed so the conditional partial-update behaviour is exercised.

type reviewFakeAttendanceRepo struct {
	fakeAttendanceRepo
	reviewedAction  string
	reviewedNote    string
	reviewedAdminID uint
	reviewedEarn    *int64
	reviewed        bool
}

type rejectConflictAttendanceRepo struct {
	reviewFakeAttendanceRepo
}

func (f *rejectConflictAttendanceRepo) MarkAdminReviewed(
	_ context.Context, id uint, action domain.AttendanceReviewAction,
	_ string, _ uint, _ time.Time, _ *int64,
) (bool, error) {
	if action == domain.AttendanceReviewActionRejected {
		rec := f.findByID(id)
		approved := string(domain.AttendanceReviewActionApproved)
		rec.ReviewAction = &approved
	}
	return false, nil
}

func (f *reviewFakeAttendanceRepo) MarkAdminReviewed(
	_ context.Context, id uint, action domain.AttendanceReviewAction,
	note string, adminID uint, _ time.Time, earning *int64,
) (bool, error) {
	rec := f.findByID(id)
	if rec == nil {
		return false, nil
	}
	f.reviewedAction = string(action)
	f.reviewedNote = note
	f.reviewedAdminID = adminID
	f.reviewedEarn = earning
	f.reviewed = true

	// Mirror the real repo: apply the override in-memory so the returned record
	// reflects what a reload would yield.
	act := string(action)
	rec.ReviewAction = &act
	rec.ReviewNote = &note
	rec.EarningAmount = earning
	if action == domain.AttendanceReviewActionApproved {
		rec.SalaryRejectReason = nil
	} else {
		rec.SalaryRejectReason = &note
	}
	return true, nil
}

func newReviewService(att *domain.Attendance, payrate *domain.Payrate, assignment *domain.ProjectEmployee) (*AttendanceService, *reviewFakeAttendanceRepo) {
	repo := &reviewFakeAttendanceRepo{fakeAttendanceRepo: fakeAttendanceRepo{byID: att}}
	if payrate == nil {
		payrate = &domain.Payrate{
			Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
		}
	}
	if assignment == nil {
		assignment = &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}
	}
	return &AttendanceService{
		attendanceRepo:      repo,
		projectEmployeeRepo: &fakeProjectEmployeeRepo{assignment: assignment},
		projectRepo: &fakeProjectRepo{p: &domain.Project{
			ID:                   55,
			GeofenceRadiusMeters: 100,
			GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10.0, Lng: 106.0}},
		}},
		payrateRepo:        &fakePayrateRepo{pr: payrate},
		advancePaymentRepo: &fakeAdvancePaymentRepo{},
		transactionManager: &fakeTransactionManager{},
		clock:              clock.NewFake(time.Date(2026, 6, 22, 18, 0, 0, 0, clock.DefaultLocation)),
	}, repo
}

func TestApproveRecomputesEarningAndClearsRejectReason(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	zero := int64(0)
	priorReason := "Tự động huỷ do quá hạn tan ca"
	// An auto-rejected row: earning 0, reject reason set, no checkout.
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		EarningAmount:      &zero,
		SalaryRejectReason: &priorReason,
	}

	svc, repo := newReviewService(att, nil, nil)
	res, err := svc.Approve(context.Background(), 7, 42, "Khách xác nhận làm đủ ca")
	if err != nil {
		t.Fatalf("expected approve to succeed, got %v", err)
	}
	if !repo.reviewed {
		t.Fatal("expected MarkAdminReviewed to be called")
	}
	if repo.reviewedAction != string(domain.AttendanceReviewActionApproved) {
		t.Fatalf("expected action approved, got %q", repo.reviewedAction)
	}
	if repo.reviewedAdminID != 42 {
		t.Fatalf("expected admin id 42, got %d", repo.reviewedAdminID)
	}
	// 08:00-17:00 shift pays 300000 when checkout is the configured end 17:00.
	if repo.reviewedEarn == nil || *repo.reviewedEarn != 300000 {
		t.Fatalf("expected recomputed earning 300000, got %v", repo.reviewedEarn)
	}
	if res.EarningAmount == nil || *res.EarningAmount != 300000 {
		t.Fatalf("expected returned earning 300000, got %v", res.EarningAmount)
	}
	if res.SalaryRejectReason != nil {
		t.Fatalf("expected reject reason cleared, got %v", *res.SalaryRejectReason)
	}
	if res.ReviewAction == nil || *res.ReviewAction != "approved" {
		t.Fatalf("expected review_action=approved, got %v", res.ReviewAction)
	}
}

func TestApproveIdempotentWhenAlreadyApproved(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	approved := "approved"
	note := "đã duyệt"
	earn := int64(300000)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		ReviewAction: &approved, ReviewNote: &note, EarningAmount: &earn,
	}
	svc, repo := newReviewService(att, nil, nil)

	res, err := svc.Approve(context.Background(), 7, 42, "lần nữa")
	if err != nil {
		t.Fatalf("expected idempotent approve to succeed, got %v", err)
	}
	if repo.reviewed {
		t.Fatal("expected MarkAdminReviewed NOT to be called for an already-approved row")
	}
	if res.EarningAmount == nil || *res.EarningAmount != 300000 {
		t.Fatalf("expected earning unchanged at 300000, got %v", res.EarningAmount)
	}
}

func TestApproveRepairsApprovedRecordRejectedByDelayedWorker(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	approved := string(domain.AttendanceReviewActionApproved)
	delayedRejectReason := "Đã hết hạn tan ca"
	zero := int64(0)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		ReviewAction: &approved, SalaryRejectReason: &delayedRejectReason, EarningAmount: &zero,
	}

	svc, repo := newReviewService(att, nil, nil)
	res, err := svc.Approve(context.Background(), 7, 42, "Khôi phục quyết định duyệt")
	if err != nil {
		t.Fatalf("expected repair approval to succeed, got %v", err)
	}
	if !repo.reviewed {
		t.Fatal("expected inconsistent approved record to be repaired")
	}
	if res.SalaryRejectReason != nil {
		t.Fatalf("expected delayed reject reason cleared, got %q", *res.SalaryRejectReason)
	}
	if res.EarningAmount == nil || *res.EarningAmount != 300000 {
		t.Fatalf("expected earning restored to 300000, got %v", res.EarningAmount)
	}
}

func TestApproveFailsWhenShiftUnresolvable(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
	}
	// No payrate configured → resolveShiftForAttendance returns nil.
	svc, repo := newReviewService(att, &domain.Payrate{}, &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true})

	_, err := svc.Approve(context.Background(), 7, 42, "")
	if err == nil {
		t.Fatal("expected error when shift cannot be resolved")
	}
	if repo.reviewed {
		t.Fatal("expected MarkAdminReviewed NOT to be called when recompute fails")
	}
	if !strings.Contains(err.Error(), "ca làm việc") && !domain.IsValidationError(err) {
		t.Fatalf("expected a validation error about shift config, got %q", err.Error())
	}
}

func TestRejectZeroesEarningAndStampsReason(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	earn := int64(282400)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		EarningAmount: &earn,
	}
	svc, repo := newReviewService(att, nil, nil)

	res, err := svc.Reject(context.Background(), 7, 9, "NV rời dự án sớm")
	if err != nil {
		t.Fatalf("expected reject to succeed, got %v", err)
	}
	if !repo.reviewed {
		t.Fatal("expected MarkAdminReviewed to be called")
	}
	if repo.reviewedAction != string(domain.AttendanceReviewActionRejected) {
		t.Fatalf("expected action rejected, got %q", repo.reviewedAction)
	}
	if repo.reviewedEarn == nil || *repo.reviewedEarn != 0 {
		t.Fatalf("expected earning forced to 0, got %v", repo.reviewedEarn)
	}
	if repo.reviewedNote != "NV rời dự án sớm" {
		t.Fatalf("expected note stored, got %q", repo.reviewedNote)
	}
	if res.SalaryRejectReason == nil || *res.SalaryRejectReason != "NV rời dự án sớm" {
		t.Fatalf("expected salary_reject_reason set to note, got %v", res.SalaryRejectReason)
	}
	if res.ReviewAction == nil || *res.ReviewAction != "rejected" {
		t.Fatalf("expected review_action=rejected, got %v", res.ReviewAction)
	}
}

func TestRejectIdempotentWhenAlreadyRejected(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	rejected := "rejected"
	note := "đã từ chối"
	zero := int64(0)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		ReviewAction: &rejected, ReviewNote: &note, EarningAmount: &zero,
	}
	svc, repo := newReviewService(att, nil, nil)

	res, err := svc.Reject(context.Background(), 7, 9, "lần nữa")
	if err != nil {
		t.Fatalf("expected idempotent reject to succeed, got %v", err)
	}
	if repo.reviewed {
		t.Fatal("expected MarkAdminReviewed NOT to be called for an already-rejected row")
	}
	if res.EarningAmount == nil || *res.EarningAmount != 0 {
		t.Fatalf("expected earning unchanged at 0, got %v", res.EarningAmount)
	}
}

func TestRejectRefusesCompletedAdminApproval(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	approved := string(domain.AttendanceReviewActionApproved)
	earn := int64(300000)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckInGate: "Cổng chính",
		ReviewAction: &approved, EarningAmount: &earn,
	}
	svc, repo := newReviewService(att, nil, nil)

	_, err := svc.Reject(context.Background(), 7, 9, "đổi quyết định")
	if err == nil || !domain.IsValidationError(err) {
		t.Fatalf("expected validation error for terminal approval, got %v", err)
	}
	if repo.reviewed {
		t.Fatal("expected approved attendance and credited quota to remain unchanged")
	}
}

func TestRejectRefusesAttendanceWhoseQuotaWasAlreadyCredited(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	checkOut := time.Date(2026, 6, 22, 17, 0, 0, 0, loc)
	creditedAt := checkOut.Add(24 * time.Hour)
	earn := int64(300000)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, CheckOutTime: &checkOut,
		EarningAmount: &earn, QuotaCreditedAt: &creditedAt,
	}
	svc, repo := newReviewService(att, nil, nil)

	_, err := svc.Reject(context.Background(), 7, 9, "đổi quyết định")
	if err == nil || !domain.IsValidationError(err) {
		t.Fatalf("expected validation error for credited attendance, got %v", err)
	}
	if repo.reviewed {
		t.Fatal("expected credited attendance and employee quota to remain unchanged")
	}
}

func TestRejectReportsConflictWhenConcurrentApprovalWins(t *testing.T) {
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 6, 22, 8, 35, 0, 0, loc)
	earn := int64(300000)
	att := &domain.Attendance{
		ID: 7, ProjectID: 55, EmployeeID: 123,
		Date: checkIn, CheckInTime: checkIn, EarningAmount: &earn,
	}
	svc, baseRepo := newReviewService(att, nil, nil)
	conflictRepo := &rejectConflictAttendanceRepo{reviewFakeAttendanceRepo: *baseRepo}
	svc.attendanceRepo = conflictRepo

	_, err := svc.Reject(context.Background(), 7, 9, "đổi quyết định")
	if err == nil || !domain.IsValidationError(err) {
		t.Fatalf("expected validation conflict when approval wins, got %v", err)
	}
	if att.ReviewAction == nil || *att.ReviewAction != string(domain.AttendanceReviewActionApproved) {
		t.Fatalf("expected concurrent approval preserved, got %v", att.ReviewAction)
	}
}

func TestApproveReturnsNotFoundForMissingRecord(t *testing.T) {
	svc, _ := newReviewService(nil, nil, nil) // byID == nil
	_, err := svc.Approve(context.Background(), 999, 1, "")
	if err == nil || !domain.IsNotFoundError(err) {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}

func TestRejectReturnsNotFoundForMissingRecord(t *testing.T) {
	svc, _ := newReviewService(nil, nil, nil)
	_, err := svc.Reject(context.Background(), 999, 1, "note")
	if err == nil || !domain.IsNotFoundError(err) {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}
