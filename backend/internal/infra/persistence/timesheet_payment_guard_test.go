package persistence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPaymentFinalizationAndRejectionRemainMutuallyExclusive(t *testing.T) {
	t.Run("rejection wins workflow while later settlement records paid", func(t *testing.T) {
		repo, db := newPaymentGuardRepository(t)
		timesheet := seedPaymentGuardTimesheet(t, db)
		sibling := seedPaymentGuardTimesheet(t, db)
		sibling.ProjectID = 2
		if err := db.Model(sibling).Update("project_id", sibling.ProjectID).Error; err != nil {
			t.Fatalf("move sibling to other project: %v", err)
		}
		workDate := timesheet.Date

		rejector := repo.(domain.TimesheetUnpaidRejector)
		count, err := rejector.RejectUnpaidByProjectDateRange(context.Background(), timesheet.ProjectID, workDate, workDate, "Sai kỳ công", 9)
		if err != nil || count != 1 {
			t.Fatalf("reject count=%d err=%v", count, err)
		}
		if err := repo.BulkUpdatePaymentStatus(context.Background(), []domain.PaymentStatusUpdate{
			{TimesheetID: timesheet.ID, PaymentStatus: domain.PaymentStatusPaid},
			{TimesheetID: sibling.ID, PaymentStatus: domain.PaymentStatusPaid},
		}); err != nil {
			t.Fatalf("authoritative settlement after rejection: %v", err)
		}

		got := reloadPaymentGuardTimesheet(t, db, timesheet.ID)
		if got.Status != domain.TimesheetStatusRejected || got.PaymentStatus != domain.PaymentStatusPaid || got.RejectionReason != "Sai kỳ công" {
			t.Fatalf("invalid rejected/payment state: %+v", got)
		}
		gotSibling := reloadPaymentGuardTimesheet(t, db, sibling.ID)
		if gotSibling.Status != domain.TimesheetStatusApproved || gotSibling.PaymentStatus != domain.PaymentStatusPaid {
			t.Fatalf("eligible sibling was not finalized: %+v", gotSibling)
		}
	})

	t.Run("payment wins then rejection is a zero match", func(t *testing.T) {
		repo, db := newPaymentGuardRepository(t)
		timesheet := seedPaymentGuardTimesheet(t, db)
		workDate := timesheet.Date

		if err := repo.BulkUpdatePaymentStatus(context.Background(), []domain.PaymentStatusUpdate{{
			TimesheetID:   timesheet.ID,
			PaymentStatus: domain.PaymentStatusPaid,
		}}); err != nil {
			t.Fatalf("finalize payment: %v", err)
		}
		rejector := repo.(domain.TimesheetUnpaidRejector)
		count, err := rejector.RejectUnpaidByProjectDateRange(context.Background(), timesheet.ProjectID, workDate, workDate, "Sai kỳ công", 9)
		if err != nil || count != 0 {
			t.Fatalf("reject after payment count=%d err=%v", count, err)
		}

		got := reloadPaymentGuardTimesheet(t, db, timesheet.ID)
		if got.Status != domain.TimesheetStatusApproved || got.PaymentStatus != domain.PaymentStatusPaid {
			t.Fatalf("paid timesheet was rejected: %+v", got)
		}
	})
}

func TestProtectedHistoryPreventsOperationalHardDelete(t *testing.T) {
	repo, db := newPaymentGuardRepository(t)
	approvedPending := seedPaymentGuardTimesheet(t, db)
	approvedPending.EmployeeID = 901
	if err := db.Model(approvedPending).Update("employee_id", approvedPending.EmployeeID).Error; err != nil {
		t.Fatalf("move approved timesheet to employee: %v", err)
	}

	operational := seedPaymentGuardTimesheet(t, db)
	operational.EmployeeID = 902
	operational.Status = domain.TimesheetStatusPendingApproval
	if err := db.Model(operational).Updates(map[string]any{
		"employee_id":      operational.EmployeeID,
		"timesheet_status": operational.Status,
	}).Error; err != nil {
		t.Fatalf("prepare operational timesheet: %v", err)
	}

	protected, err := repo.HasProtectedTimesheetsByEmployeeID(context.Background(), 901)
	if err != nil {
		t.Fatalf("check approved payroll protection: %v", err)
	}
	if !protected {
		t.Fatal("approved payroll must be retained while an asynchronous bank result can still arrive")
	}

	protected, err = repo.HasProtectedTimesheetsByEmployeeID(context.Background(), 902)
	if err != nil {
		t.Fatalf("check operational history: %v", err)
	}
	if protected {
		t.Fatal("pending, unpaid operational timesheet should not prevent permanent deletion")
	}

	if err := repo.HardDeleteOperationalByEmployeeID(context.Background(), 901); err != nil {
		t.Fatalf("hard delete approved employee operational rows: %v", err)
	}
	var approvedRemaining int64
	if err := db.Model(&domain.Timesheet{}).Where("employee_id = ?", 901).Count(&approvedRemaining).Error; err != nil {
		t.Fatalf("count approved employee timesheets: %v", err)
	}
	if approvedRemaining != 1 {
		t.Fatalf("approved timesheet was deleted, remaining=%d", approvedRemaining)
	}

	if err := repo.HardDeleteOperationalByEmployeeID(context.Background(), 902); err != nil {
		t.Fatalf("hard delete operational employee rows: %v", err)
	}
	var operationalRemaining int64
	if err := db.Model(&domain.Timesheet{}).Where("employee_id = ?", 902).Count(&operationalRemaining).Error; err != nil {
		t.Fatalf("count operational employee timesheets: %v", err)
	}
	if operationalRemaining != 0 {
		t.Fatalf("operational timesheet was retained, remaining=%d", operationalRemaining)
	}
}

func newPaymentGuardRepository(t *testing.T) (domain.TimesheetRepository, *gorm.DB) {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName)), &gorm.Config{
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
	if err := db.Exec(`
		CREATE TABLE timesheet_edit_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT, timesheet_id INTEGER,
			status TEXT, requested_by INTEGER, approved_by INTEGER,
			rejected_by INTEGER, deleted_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create edit requests: %v", err)
	}

	return NewTimesheetRepository(&Database{DB: db}, nil), db
}

func seedPaymentGuardTimesheet(t *testing.T, db *gorm.DB) *domain.Timesheet {
	t.Helper()
	timesheet := &domain.Timesheet{
		ProjectID:     1,
		EmployeeID:    2,
		PayrateID:     3,
		Date:          time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		PayType:       "worker.weekday.day",
		Status:        domain.TimesheetStatusApproved,
		PaymentStatus: domain.PaymentStatusPending,
		CreatedBy:     1,
	}
	if err := db.Create(timesheet).Error; err != nil {
		t.Fatalf("seed timesheet: %v", err)
	}
	return timesheet
}

func reloadPaymentGuardTimesheet(t *testing.T, db *gorm.DB, id uint) domain.Timesheet {
	t.Helper()
	var timesheet domain.Timesheet
	if err := db.First(&timesheet, id).Error; err != nil {
		t.Fatalf("reload timesheet: %v", err)
	}
	return timesheet
}
