package repositories

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTimesheetCommandRepositoryUsesTransactionContextForReplacement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE timesheets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER, employee_id INTEGER, payrate_id INTEGER,
			date DATETIME, hours_worked REAL, paytype TEXT, payrate INTEGER,
			amount INTEGER, timesheet_status TEXT, payment_status TEXT,
			allowed_edit INTEGER, request_edit_id INTEGER,
			payment_reference TEXT, payment_date DATETIME, paid_amount INTEGER,
			paid_at DATETIME, revenue_receivable INTEGER, revenue_paid INTEGER,
			transaction_id INTEGER, deleted_at DATETIME, created_by INTEGER,
			approved_by INTEGER, approved_at DATETIME, rejection_reason TEXT,
			created_at DATETIME, updated_at DATETIME, force_payroll INTEGER
		)
	`).Error; err != nil {
		t.Fatalf("create timesheets: %v", err)
	}

	repo := NewTimesheetCommandRepository(db)
	original := &domain.Timesheet{
		ProjectID:     10,
		EmployeeID:    20,
		PayrateID:     30,
		Date:          time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		PayType:       "day",
		Status:        domain.TimesheetStatusPendingApproval,
		PaymentStatus: domain.PaymentStatusPending,
		CreatedBy:     1,
	}
	if err := db.Create(original).Error; err != nil {
		t.Fatalf("seed original: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	txCtx := domain.WithTransactionContext(context.Background(), &domain.TransactionContext{
		TX:              tx,
		IsTransactional: true,
	})

	if err := repo.HardDelete(txCtx, original.ID); err != nil {
		_ = tx.Rollback()
		t.Fatalf("hard delete in transaction: %v", err)
	}
	queryRepo := NewTimesheetQueryRepository(db)
	visible, err := queryRepo.GetByEmployeeDateCombos(txCtx, []domain.EmployeeDateCombo{{
		EmployeeID: original.EmployeeID,
		ProjectID:  original.ProjectID,
		Date:       original.Date,
	}})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("query deleted row in transaction: %v", err)
	}
	if len(visible) != 0 {
		_ = tx.Rollback()
		t.Fatalf("deleted row remained visible inside replacement transaction")
	}
	replacement := &domain.Timesheet{
		ProjectID:     10,
		EmployeeID:    20,
		PayrateID:     30,
		Date:          time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		PayType:       "day",
		Status:        domain.TimesheetStatusPendingApproval,
		PaymentStatus: domain.PaymentStatusPending,
		CreatedBy:     1,
	}
	if err := repo.Create(txCtx, replacement); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create replacement in transaction: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var originalCount int64
	if err := db.Model(&domain.Timesheet{}).Where("id = ?", original.ID).Count(&originalCount).Error; err != nil {
		t.Fatalf("count original: %v", err)
	}
	if originalCount != 1 {
		t.Fatalf("original count after rollback = %d, want 1", originalCount)
	}

	var replacementCount int64
	if err := db.Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ? AND date = ?", 10, 20, replacement.Date).
		Count(&replacementCount).Error; err != nil {
		t.Fatalf("count replacement: %v", err)
	}
	if replacementCount != 0 {
		t.Fatalf("replacement count after rollback = %d, want 0", replacementCount)
	}
}
