package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newBankSyncTestDB builds a minimal in-memory schema covering the columns
// domain.Employee / domain.Bank persist, so the real EmployeeRepository.Update
// runs against the same table shape as production. Raw DDL keeps the test
// hermetic: domain.User carries a MySQL enum that SQLite cannot migrate, so
// AutoMigrate on the full entity graph is not an option here.
func newBankSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	const schema = `
CREATE TABLE banks (
	id INTEGER PRIMARY KEY,
	branch_name TEXT NOT NULL,
	bin TEXT,
	bank_code TEXT,
	swift_code TEXT
);
CREATE TABLE employees (
	id INTEGER PRIMARY KEY,
	fullname TEXT NOT NULL,
	email TEXT,
	cccd TEXT NOT NULL,
	address TEXT,
	mobile TEXT,
	bank_id INTEGER,
	bank_account_number TEXT,
	bank_account_name TEXT,
	bank_account_status TEXT NOT NULL DEFAULT 'valid',
	bank_account_invalid_reason TEXT,
	bank_account_validated_at DATETIME,
	date_of_birth DATETIME,
	user_id INTEGER,
	deleted_at DATETIME,
	created_by INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME,
	updated_at DATETIME
);
CREATE UNIQUE INDEX uniq_employees_cccd_live ON employees (cccd, deleted_at);
`
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("raw db handle: %v", err)
	}
	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	return db
}

// TestEmployeeUpdate_ChangingBankIDBeatsStaleAssociation reproduces the LGD
// FlexPay import bug: GetByCCCD preloads the employee's current Bank
// association; the import then resolves the workbook's bank name to a
// different bank and sets BankID. GORM's Save re-syncs the bank_id FK from
// the still-loaded association, so the update silently kept the employee's
// old bank (Excel said "MB BANK", the employee stayed on MSB, and OnePay
// rejected the disbursement with "Invalid account info").
//
// The repository contract after the fix: Update persists the BankID field
// the caller set, association or not.
func TestEmployeeUpdate_ChangingBankIDBeatsStaleAssociation(t *testing.T) {
	gormDB := newBankSyncTestDB(t)
	repo := NewEmployeeRepository(&Database{DB: gormDB})
	ctx := context.Background()

	mb := &domain.Bank{BranchName: "Quân đội (MB)", BankCode: "MB"}
	msb := &domain.Bank{BranchName: "Hàng hải (MSB)", BankCode: "MSB"}
	if err := gormDB.Create(mb).Error; err != nil {
		t.Fatalf("seed MB bank: %v", err)
	}
	if err := gormDB.Create(msb).Error; err != nil {
		t.Fatalf("seed MSB bank: %v", err)
	}

	employee := &domain.Employee{Fullname: "Nguyễn Văn Nguyên", CCCD: "034203013000", BankID: &msb.ID}
	if err := gormDB.Create(employee).Error; err != nil {
		t.Fatalf("seed employee: %v", err)
	}

	// Mirror GetByCCCD: load with the Bank association populated.
	loaded, err := repo.GetByCCCD(ctx, employee.CCCD)
	if err != nil {
		t.Fatalf("GetByCCCD: %v", err)
	}
	if loaded.Bank == nil || loaded.Bank.ID != msb.ID {
		t.Fatalf("precondition: employee should load with stale MSB association, got %+v", loaded.Bank)
	}

	// Mirror the import: workbook says MB BANK → resolve to the MB bank.
	wantBankID := mb.ID
	if wantBankID == msb.ID {
		t.Fatal("precondition: MB and MSB must be different banks")
	}
	// GORM writes the association's PK THROUGH the BankID pointer during
	// Save, so point it at a disposable copy and keep wantBankID untouched
	// as the expectation.
	persistedBankID := wantBankID
	loaded.BankID = &persistedBankID
	if err := repo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var after domain.Employee
	if err := gormDB.First(&after, employee.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.BankID == nil || *after.BankID != wantBankID {
		t.Fatalf("Update reverted the bank change: got bank_id=%v, want %d", after.BankID, wantBankID)
	}
}

// TestEmployeeUpdate_ExcelImportOverwritesEmployeeFields locks the full
// import contract: when a workbook row differs from the stored employee, the
// update must overwrite EVERY field the import manages — fullname, account
// number, account name, and the resolved bank — exactly as the FlexPay
// importer's GetByCCCD → mutate → Update sequence does.
func TestEmployeeUpdate_ExcelImportOverwritesEmployeeFields(t *testing.T) {
	gormDB := newBankSyncTestDB(t)
	repo := NewEmployeeRepository(&Database{DB: gormDB})
	ctx := context.Background()

	oldBank := &domain.Bank{BranchName: "Hàng hải (MSB)", BankCode: "MSB"}
	newBank := &domain.Bank{BranchName: "Quân đội (MB)", BankCode: "MB"}
	if err := gormDB.Create(oldBank).Error; err != nil {
		t.Fatalf("seed MSB bank: %v", err)
	}
	if err := gormDB.Create(newBank).Error; err != nil {
		t.Fatalf("seed MB bank: %v", err)
	}
	employee := &domain.Employee{
		Fullname:          "Old Name",
		CCCD:              "034203013000",
		BankID:            &oldBank.ID,
		BankAccountNumber: "0000000000",
		BankAccountName:   "OLD NAME",
		Mobile:            "0900000000",
	}
	if err := gormDB.Create(employee).Error; err != nil {
		t.Fatalf("seed employee: %v", err)
	}

	// Import row: load through GetByCCCD (preloads Bank), then overwrite the
	// managed fields with the workbook values — mirror of the importer.
	loaded, err := repo.GetByCCCD(ctx, employee.CCCD)
	if err != nil {
		t.Fatalf("GetByCCCD: %v", err)
	}
	loaded.Fullname = "Nguyễn Văn Nguyên"
	loaded.BankID = &newBank.ID
	loaded.BankAccountNumber = "0347921581"
	loaded.BankAccountName = "NGUYEN VAN NGUYEN"
	loaded.Mobile = "0347921581"
	if err := repo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var after domain.Employee
	if err := gormDB.First(&after, employee.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.BankID == nil || *after.BankID != newBank.ID {
		t.Fatalf("bank not updated from Excel: got bank_id=%v, want %d", after.BankID, newBank.ID)
	}
	if after.BankAccountNumber != "0347921581" {
		t.Fatalf("account number not updated from Excel: got %q", after.BankAccountNumber)
	}
	if after.BankAccountName != "NGUYEN VAN NGUYEN" {
		t.Fatalf("account name not updated from Excel: got %q", after.BankAccountName)
	}
	if after.Fullname != "Nguyễn Văn Nguyên" {
		t.Fatalf("fullname not updated from Excel: got %q", after.Fullname)
	}
	if after.Mobile != "0347921581" {
		t.Fatalf("mobile not updated from Excel: got %q", after.Mobile)
	}
}

// TestEmployeeUpdate_KeepsBankIDWhenAssociationUnchanged pins the flip side:
// an Update that does not touch banking fields must not clear or alter the
// stored bank_id just because an association happens to be (or not be) loaded.
func TestEmployeeUpdate_KeepsBankIDWhenAssociationUnchanged(t *testing.T) {
	gormDB := newBankSyncTestDB(t)
	repo := NewEmployeeRepository(&Database{DB: gormDB})
	ctx := context.Background()

	msb := &domain.Bank{BranchName: "Hàng hải (MSB)", BankCode: "MSB"}
	if err := gormDB.Create(msb).Error; err != nil {
		t.Fatalf("seed bank: %v", err)
	}
	employee := &domain.Employee{Fullname: "NVN", CCCD: "034203013000", BankID: &msb.ID}
	if err := gormDB.Create(employee).Error; err != nil {
		t.Fatalf("seed employee: %v", err)
	}

	loaded, err := repo.GetByCCCD(ctx, employee.CCCD)
	if err != nil {
		t.Fatalf("GetByCCCD: %v", err)
	}
	loaded.Fullname = "Nguyễn Văn Nguyên Mới"
	if err := repo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var after domain.Employee
	if err := gormDB.First(&after, employee.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.BankID == nil || *after.BankID != msb.ID {
		t.Fatalf("unrelated update changed bank_id: got %v, want %d", after.BankID, msb.ID)
	}
	if after.Fullname != "Nguyễn Văn Nguyên Mới" {
		t.Fatalf("fullname update lost: got %q", after.Fullname)
	}
}
