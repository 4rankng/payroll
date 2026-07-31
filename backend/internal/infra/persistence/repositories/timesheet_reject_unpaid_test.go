package repositories

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

func TestRejectUnpaidByProjectDateRangeScopesAndPreservesPaymentHistory(t *testing.T) {
	db := openRejectUnpaidTestDB(t)
	repo := NewTimesheetCommandRepository(db)

	approvedBy := uint(99)
	requestEditID := uint(77)
	paidRequestEditID := uint(78)
	paymentReference := "attempt-123"
	paymentDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	approvedAt := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)
	insideDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)

	timesheets := []*domain.Timesheet{
		newRejectUnpaidTimesheet(1, insideDate, domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending),
		newRejectUnpaidTimesheet(1, insideDate, domain.TimesheetStatusApproved, domain.PaymentStatusFailed),
		newRejectUnpaidTimesheet(1, insideDate, domain.TimesheetStatusApproved, domain.PaymentStatusCancelled),
		newRejectUnpaidTimesheet(1, insideDate, domain.TimesheetStatusApproved, domain.PaymentStatusPaid),
		newRejectUnpaidTimesheet(2, insideDate, domain.TimesheetStatusApproved, domain.PaymentStatusPending),
		newRejectUnpaidTimesheet(1, insideDate.AddDate(0, 0, 10), domain.TimesheetStatusApproved, domain.PaymentStatusPending),
		newRejectUnpaidTimesheet(1, insideDate, domain.TimesheetStatusRejected, domain.PaymentStatusPending),
	}

	for _, timesheet := range timesheets {
		timesheet.ApprovedBy = &approvedBy
		timesheet.ApprovedAt = &approvedAt
		timesheet.AllowedEdit = true
		timesheet.ForcePayroll = true
	}
	timesheets[0].RequestEditID = &requestEditID
	timesheets[3].RequestEditID = &paidRequestEditID
	timesheets[1].PaymentReference = &paymentReference
	timesheets[1].PaymentDate = &paymentDate
	timesheets[1].PaidAmount = 120000
	timesheets[6].RejectionReason = "Lý do trước đó"
	if err := db.Create(timesheets).Error; err != nil {
		t.Fatalf("seed timesheets: %v", err)
	}
	editRequests := []*domain.TimesheetEditRequest{
		{ID: requestEditID, TimesheetID: timesheets[0].ID, RequestedBy: 10, Status: domain.EditRequestStatusPending},
		{ID: paidRequestEditID, TimesheetID: timesheets[3].ID, RequestedBy: 10, Status: domain.EditRequestStatusPending},
		{ID: 79, TimesheetID: timesheets[0].ID, RequestedBy: 11, Status: domain.EditRequestStatusPending},
	}
	if err := db.Create(editRequests).Error; err != nil {
		t.Fatalf("seed edit requests: %v", err)
	}

	count, err := repo.RejectUnpaidByProjectDateRange(
		context.Background(),
		1,
		time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		"Sai kỳ công",
		55,
	)
	if err != nil {
		t.Fatalf("reject unpaid: %v", err)
	}
	if count != 3 {
		t.Fatalf("affected count = %d, want 3", count)
	}

	var got []*domain.Timesheet
	if err := db.Order("id ASC").Find(&got).Error; err != nil {
		t.Fatalf("reload timesheets: %v", err)
	}
	for index := 0; index < 3; index++ {
		assertRejectedTimesheetState(t, got[index], "Sai kỳ công")
	}
	if got[1].PaymentStatus != domain.PaymentStatusFailed || got[1].PaymentReference == nil || *got[1].PaymentReference != paymentReference || got[1].PaymentDate == nil || got[1].PaidAmount != 120000 {
		t.Fatalf("historical payment-attempt fields were not preserved: %+v", got[1])
	}
	if got[3].Status != domain.TimesheetStatusApproved || got[3].PaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("paid timesheet was changed: %+v", got[3])
	}
	if got[4].Status != domain.TimesheetStatusApproved {
		t.Fatalf("other-project timesheet was changed: %+v", got[4])
	}
	if got[5].Status != domain.TimesheetStatusApproved {
		t.Fatalf("out-of-range timesheet was changed: %+v", got[5])
	}
	if got[6].Status != domain.TimesheetStatusRejected || got[6].RejectionReason != "Lý do trước đó" {
		t.Fatalf("already-rejected timesheet was rewritten: %+v", got[6])
	}
	var gotRequests []domain.TimesheetEditRequest
	if err := db.Order("id ASC").Find(&gotRequests).Error; err != nil {
		t.Fatalf("reload edit requests: %v", err)
	}
	if gotRequests[0].Status != domain.EditRequestStatusRejected || gotRequests[0].RejectedBy == nil || *gotRequests[0].RejectedBy != 55 {
		t.Fatalf("eligible pending edit request was not rejected: %+v", gotRequests[0])
	}
	if gotRequests[1].Status != domain.EditRequestStatusPending {
		t.Fatalf("paid timesheet edit request was changed: %+v", gotRequests[1])
	}
	if gotRequests[2].Status != domain.EditRequestStatusPending {
		t.Fatalf("unlinked edit request was changed: %+v", gotRequests[2])
	}

	secondCount, err := repo.RejectUnpaidByProjectDateRange(
		context.Background(),
		1,
		time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		"Lý do mới",
		55,
	)
	if err != nil {
		t.Fatalf("repeat reject unpaid: %v", err)
	}
	if secondCount != 0 {
		t.Fatalf("repeat affected count = %d, want 0", secondCount)
	}
}

func TestRejectUnpaidByProjectDateRangeUsesTransactionContext(t *testing.T) {
	db := openRejectUnpaidTestDB(t)
	repo := NewTimesheetCommandRepository(db)
	workDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	timesheet := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusApproved, domain.PaymentStatusPending)
	if err := db.Create(timesheet).Error; err != nil {
		t.Fatalf("seed timesheet: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	txCtx := domain.WithTransactionContext(context.Background(), &domain.TransactionContext{TX: tx, IsTransactional: true})
	count, err := repo.RejectUnpaidByProjectDateRange(txCtx, 1, workDate, workDate, "rollback", 9)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("reject in transaction: %v", err)
	}
	if count != 1 {
		_ = tx.Rollback()
		t.Fatalf("affected count = %d, want 1", count)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var got domain.Timesheet
	if err := db.First(&got, timesheet.ID).Error; err != nil {
		t.Fatalf("reload timesheet: %v", err)
	}
	if got.Status != domain.TimesheetStatusApproved {
		t.Fatalf("transaction rollback did not restore status: %s", got.Status)
	}
}

func TestResetForApprovedEditRequestRejectsStaleLinkOrStatus(t *testing.T) {
	db := openRejectUnpaidTestDB(t)
	repo := NewTimesheetCommandRepository(db)
	workDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	requestID := uint(41)

	valid := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusApproved, domain.PaymentStatusPending)
	valid.RequestEditID = &requestID
	staleStatus := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusRejected, domain.PaymentStatusPending)
	staleStatus.RequestEditID = &requestID
	otherRequestID := uint(42)
	staleLink := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusApproved, domain.PaymentStatusPending)
	staleLink.RequestEditID = &otherRequestID
	if err := db.Create([]*domain.Timesheet{valid, staleStatus, staleLink}).Error; err != nil {
		t.Fatalf("seed timesheets: %v", err)
	}

	if err := repo.ResetForApprovedEditRequest(context.Background(), valid.ID, requestID); err != nil {
		t.Fatalf("reset valid edit request: %v", err)
	}
	if err := repo.ResetForApprovedEditRequest(context.Background(), staleStatus.ID, requestID); !domain.IsConflictError(err) {
		t.Fatalf("stale status error = %v, want conflict", err)
	}
	if err := repo.ResetForApprovedEditRequest(context.Background(), staleLink.ID, requestID); !domain.IsConflictError(err) {
		t.Fatalf("stale link error = %v, want conflict", err)
	}

	var got []domain.Timesheet
	if err := db.Order("id ASC").Find(&got).Error; err != nil {
		t.Fatalf("reload timesheets: %v", err)
	}
	if got[0].Status != domain.TimesheetStatusPendingApproval || !got[0].AllowedEdit || got[0].RequestEditID != nil {
		t.Fatalf("valid edit request did not reset timesheet: %+v", got[0])
	}
	if got[1].Status != domain.TimesheetStatusRejected {
		t.Fatalf("stale rejected timesheet was resurrected: %+v", got[1])
	}
	if got[2].RequestEditID == nil || *got[2].RequestEditID != otherRequestID {
		t.Fatalf("unrelated edit-request link was cleared: %+v", got[2])
	}
}

func TestCommandPaymentSettlementPreservesRejectedWorkflowState(t *testing.T) {
	db := openRejectUnpaidTestDB(t)
	repo := NewTimesheetCommandRepository(db)
	workDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	timesheet := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusRejected, domain.PaymentStatusPending)
	timesheet.RejectionReason = "Sai kỳ công"
	failedTimesheet := newRejectUnpaidTimesheet(1, workDate, domain.TimesheetStatusRejected, domain.PaymentStatusPending)
	failedTimesheet.RejectionReason = "Sai kỳ công khác"
	if err := db.Create([]*domain.Timesheet{timesheet, failedTimesheet}).Error; err != nil {
		t.Fatalf("seed rejected timesheets: %v", err)
	}

	if err := repo.BulkUpdatePaymentStatus(context.Background(), []domain.PaymentStatusUpdate{
		{TimesheetID: timesheet.ID, PaymentStatus: domain.PaymentStatusPaid},
		{TimesheetID: failedTimesheet.ID, PaymentStatus: domain.PaymentStatusFailed},
	}); err != nil {
		t.Fatalf("record settlement outcomes: %v", err)
	}

	var got domain.Timesheet
	if err := db.First(&got, timesheet.ID).Error; err != nil {
		t.Fatalf("reload timesheet: %v", err)
	}
	if got.Status != domain.TimesheetStatusRejected || got.PaymentStatus != domain.PaymentStatusPaid || got.RejectionReason != "Sai kỳ công" {
		t.Fatalf("settlement overwrote workflow state: %+v", got)
	}
	got = domain.Timesheet{}
	if err := db.First(&got, failedTimesheet.ID).Error; err != nil {
		t.Fatalf("reload failed timesheet: %v", err)
	}
	if got.Status != domain.TimesheetStatusRejected || got.PaymentStatus != domain.PaymentStatusFailed || got.RejectionReason != "Sai kỳ công khác" {
		t.Fatalf("failed outcome overwrote workflow state: %+v", got)
	}
}

func openRejectUnpaidTestDB(t *testing.T) *gorm.DB {
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timesheet_id INTEGER, status TEXT, requested_by INTEGER,
			approved_by INTEGER, rejected_by INTEGER, deleted_at DATETIME,
			created_at DATETIME, updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create edit requests: %v", err)
	}
	return db
}

func newRejectUnpaidTimesheet(projectID uint, date time.Time, status domain.TimesheetStatus, paymentStatus domain.PaymentStatus) *domain.Timesheet {
	return &domain.Timesheet{
		ProjectID:     projectID,
		EmployeeID:    projectID + 10,
		PayrateID:     1,
		Date:          date,
		PayType:       "worker.weekday.day",
		Status:        status,
		PaymentStatus: paymentStatus,
		CreatedBy:     1,
	}
}

func assertRejectedTimesheetState(t *testing.T, timesheet *domain.Timesheet, reason string) {
	t.Helper()
	if timesheet.Status != domain.TimesheetStatusRejected || timesheet.RejectionReason != reason {
		t.Fatalf("timesheet was not rejected: %+v", timesheet)
	}
	if timesheet.ApprovedBy != nil || timesheet.ApprovedAt != nil || timesheet.AllowedEdit || timesheet.RequestEditID != nil || timesheet.ForcePayroll {
		t.Fatalf("active approval/edit/payroll state was not cleared: %+v", timesheet)
	}
}
