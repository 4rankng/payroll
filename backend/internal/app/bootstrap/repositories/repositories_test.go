package repositories

import (
	"testing"

	"api-server/internal/infra/persistence"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitialize(t *testing.T) {
	gormDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db := &persistence.Database{DB: gormDB}

	repos := Initialize(db, nil)
	if repos == nil {
		t.Fatal("Initialize returned nil")
		return
	}

	if repos.User == nil {
		t.Error("User repository is nil")
	}

	if repos.Employee == nil {
		t.Error("Employee repository is nil")
	}

	if repos.Timesheet == nil {
		t.Error("Timesheet repository is nil")
	}

	if repos.Project == nil {
		t.Error("Project repository is nil")
	}

	if repos.ProjectEmployee == nil {
		t.Error("ProjectEmployee repository is nil")
	}

	if repos.Bank == nil {
		t.Error("Bank repository is nil")
	}

	if repos.Payrate == nil {
		t.Error("Payrate repository is nil")
	}

	if repos.Ledger == nil {
		t.Error("Ledger repository is nil")
	}

	if repos.Transaction == nil {
		t.Error("Transaction repository is nil")
	}

	if repos.Settlement == nil {
		t.Error("Settlement repository is nil")
	}

	if repos.Notification == nil {
		t.Error("Notification repository is nil")
	}

	if repos.Settings == nil {
		t.Error("Settings repository is nil")
	}

	if repos.Asset == nil {
		t.Error("Asset repository is nil")
	}

	if repos.BulkTransferFile == nil {
		t.Error("BulkTransferFile repository is nil")
	}

	if repos.Lender == nil {
		t.Error("Lender repository is nil")
	}

	if repos.Loan == nil {
		t.Error("Loan repository is nil")
	}

	if repos.LoanRepaymentSchedule == nil {
		t.Error("LoanRepaymentSchedule repository is nil")
	}

	if repos.APIMetric == nil {
		t.Error("APIMetric repository is nil")
	}

	if repos.User == nil {
		t.Error("User repository is nil")
	}

	if repos.BlacklistedToken == nil {
		t.Error("BlacklistedToken repository is nil")
	}

	if repos.AuditLog == nil {
		t.Error("AuditLog repository is nil")
	}

	if repos.ProjectUser == nil {
		t.Error("ProjectUser repository is nil")
	}

	if repos.EmployeeUser == nil {
		t.Error("EmployeeUser repository is nil")
	}

	if repos.TimesheetImportJob == nil {
		t.Error("TimesheetImportJob repository is nil")
	}

	if repos.FlexPaySalaryNotification == nil {
		t.Error("FlexPaySalaryNotification repository is nil")
	}
}
