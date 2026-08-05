package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAttendanceStatusFiltersTreatAdminApprovalAsCompleted(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE attendances (
		id INTEGER PRIMARY KEY,
		employee_id INTEGER,
		check_out_time DATETIME,
		salary_reject_reason TEXT,
		review_action TEXT,
		check_in_time DATETIME,
		date DATE
	)`).Error; err != nil {
		t.Fatalf("create attendance table: %v", err)
	}
	now := clock.Now()
	if err := db.Exec(
		"INSERT INTO attendances (id, employee_id, review_action, check_in_time, date, check_out_time) VALUES (?, ?, ?, ?, ?, NULL), (?, ?, NULL, ?, ?, NULL), (?, ?, ?, ?, ?, ?)",
		1, 101, string(domain.AttendanceReviewActionApproved), now, now,
		2, 101, now, now,
		3, 101, string(domain.AttendanceReviewActionRejected), now, now, now,
	).Error; err != nil {
		t.Fatalf("insert attendances: %v", err)
	}

	repo := &attendanceRepository{db: db}
	completed := domain.AttendanceStatusCompleted
	checkedIn := domain.AttendanceStatusCheckedIn
	rejected := domain.AttendanceStatusRejected

	var completedCount int64
	if err := repo.buildFilterQuery(context.Background(), domain.AttendanceFilters{Status: &completed}).Count(&completedCount).Error; err != nil {
		t.Fatalf("count completed: %v", err)
	}
	if completedCount != 1 {
		t.Fatalf("completed count = %d, want 1 approved attendance", completedCount)
	}

	var checkedInCount int64
	if err := repo.buildFilterQuery(context.Background(), domain.AttendanceFilters{Status: &checkedIn}).Count(&checkedInCount).Error; err != nil {
		t.Fatalf("count checked in: %v", err)
	}
	if checkedInCount != 1 {
		t.Fatalf("checked-in count = %d, want only the unreviewed attendance", checkedInCount)
	}

	var rejectedCount int64
	if err := repo.buildFilterQuery(context.Background(), domain.AttendanceFilters{Status: &rejected}).Count(&rejectedCount).Error; err != nil {
		t.Fatalf("count rejected: %v", err)
	}
	if rejectedCount != 1 {
		t.Fatalf("rejected count = %d, want the admin-rejected checked-out attendance", rejectedCount)
	}
}

func TestMarkAutoRejectedDoesNotOverrideAdminApproval(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE attendances (
		id INTEGER PRIMARY KEY,
		check_out_time DATETIME,
		salary_reject_reason TEXT,
		review_action TEXT,
		earning_amount INTEGER,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create attendance table: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO attendances (id, review_action, earning_amount) VALUES (?, ?, ?)",
		1, string(domain.AttendanceReviewActionApproved), 300000,
	).Error; err != nil {
		t.Fatalf("insert approved attendance: %v", err)
	}

	repo := NewAttendanceRepository(db)
	updated, err := repo.MarkAutoRejected(context.Background(), 1, "Đã hết hạn tan ca")
	if err != nil {
		t.Fatalf("MarkAutoRejected: %v", err)
	}
	if updated {
		t.Fatal("expected delayed auto-reject to preserve admin approval")
	}

	var got struct {
		ReviewAction       *string
		SalaryRejectReason *string
		EarningAmount      int64
	}
	if err := db.Table("attendances").Select("review_action, salary_reject_reason, earning_amount").Where("id = ?", 1).Take(&got).Error; err != nil {
		t.Fatalf("reload attendance: %v", err)
	}
	if got.ReviewAction == nil || *got.ReviewAction != string(domain.AttendanceReviewActionApproved) {
		t.Fatalf("review_action = %v, want approved", got.ReviewAction)
	}
	if got.SalaryRejectReason != nil {
		t.Fatalf("salary_reject_reason = %q, want nil", *got.SalaryRejectReason)
	}
	if got.EarningAmount != 300000 {
		t.Fatalf("earning_amount = %d, want 300000", got.EarningAmount)
	}
}

func TestMarkAdminRejectedGuardsApprovedAndCreditedAttendance(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE attendances (
		id INTEGER PRIMARY KEY,
		review_action TEXT,
		review_note TEXT,
		reviewed_by INTEGER,
		reviewed_at DATETIME,
		earning_amount INTEGER,
		salary_reject_reason TEXT,
		quota_credited_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create attendance table: %v", err)
	}
	now := clock.Now()
	if err := db.Exec(
		"INSERT INTO attendances (id, review_action, earning_amount, quota_credited_at) VALUES (?, ?, ?, NULL), (?, NULL, ?, ?), (?, ?, 0, NULL), (?, NULL, ?, NULL)",
		1, string(domain.AttendanceReviewActionApproved), 300000,
		2, 300000, now,
		3, string(domain.AttendanceReviewActionRejected),
		4, 300000,
	).Error; err != nil {
		t.Fatalf("insert attendances: %v", err)
	}

	repo := NewAttendanceRepository(db)
	zero := int64(0)
	for _, id := range []uint{1, 2} {
		updated, err := repo.MarkAdminReviewed(context.Background(), id, domain.AttendanceReviewActionRejected, "đổi quyết định", 9, now, &zero, nil, nil)
		if err != nil {
			t.Fatalf("reject attendance %d: %v", id, err)
		}
		if updated {
			t.Fatalf("attendance %d was rejected despite terminal financial state", id)
		}
	}

	credited, err := repo.MarkQuotaCredited(context.Background(), 3, now)
	if err != nil {
		t.Fatalf("credit rejected attendance: %v", err)
	}
	if credited {
		t.Fatal("rejected attendance was credited from a stale earning snapshot")
	}
	credited, err = repo.MarkQuotaCredited(context.Background(), 4, now)
	if err != nil {
		t.Fatalf("credit payable attendance: %v", err)
	}
	if !credited {
		t.Fatal("expected payable attendance to remain creditable")
	}
}

func TestCompleteApprovedOpenPersistsCheckoutOnlyForApprovedLegacyRows(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE attendances (
		id INTEGER PRIMARY KEY,
		employee_id INTEGER,
		date DATE,
		check_in_time DATETIME,
		check_out_time DATETIME,
		check_out_gate TEXT,
		salary_reject_reason TEXT,
		review_action TEXT,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create attendance table: %v", err)
	}
	loc := clock.DefaultLocation
	checkIn := time.Date(2026, 8, 4, 8, 0, 0, 0, loc)
	if err := db.Exec(
		"INSERT INTO attendances (id, employee_id, date, check_in_time, review_action) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, NULL)",
		1, 123, checkIn, checkIn, string(domain.AttendanceReviewActionApproved),
		2, 123, checkIn, checkIn,
	).Error; err != nil {
		t.Fatalf("seed attendances: %v", err)
	}

	repo := NewAttendanceRepository(db)
	checkOut := time.Date(2026, 8, 4, 17, 0, 0, 0, loc)
	updated, err := repo.CompleteApprovedOpen(context.Background(), 1, checkOut, "admin")
	if err != nil {
		t.Fatalf("CompleteApprovedOpen: %v", err)
	}
	if !updated {
		t.Fatal("expected approved open row to be completed")
	}
	updated, err = repo.CompleteApprovedOpen(context.Background(), 2, checkOut, "admin")
	if err != nil {
		t.Fatalf("CompleteApprovedOpen non-approved row: %v", err)
	}
	if updated {
		t.Fatal("non-approved row must not be completed by legacy repair")
	}

	var got struct {
		CheckOutTime *time.Time
		CheckOutGate *string
	}
	if err := db.Table("attendances").Select("check_out_time, check_out_gate").Where("id = ?", 1).Take(&got).Error; err != nil {
		t.Fatalf("reload completed attendance: %v", err)
	}
	if got.CheckOutTime == nil || !got.CheckOutTime.Equal(checkOut) {
		t.Fatalf("check_out_time = %v, want %v", got.CheckOutTime, checkOut)
	}
	if got.CheckOutGate == nil || *got.CheckOutGate != "admin" {
		t.Fatalf("check_out_gate = %v, want admin", got.CheckOutGate)
	}
}
