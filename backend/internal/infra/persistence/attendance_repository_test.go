package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
