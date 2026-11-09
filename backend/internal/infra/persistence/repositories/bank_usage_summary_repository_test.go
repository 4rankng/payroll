package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Hermetic fixture for the bank-usage summary slice. Tables mirror the columns
// the raw SQL touches — nothing else — so the test stays readable and runs on
// sqlite in-memory (AutoMigrate of the full entity graph is not an option:
// domain.User carries MySQL-only enums).
func newBankUsageSummaryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000&_time_format=sqlite"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	ddl := []string{
		`CREATE TABLE banks (id INTEGER PRIMARY KEY, branch_name TEXT)`,
		`CREATE TABLE employees (id INTEGER PRIMARY KEY, bank_id INTEGER, deleted_at DATETIME)`,
		`CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE project_employees (employee_id INTEGER, project_id INTEGER, deleted_at DATETIME, last_date DATETIME)`,
		`CREATE TABLE timesheets (id INTEGER PRIMARY KEY, employee_id INTEGER, project_id INTEGER, deleted_at DATETIME, payment_status TEXT, paid_amount REAL)`,
		`CREATE TABLE dashboard_bank_usage_summaries (
			project_id INTEGER NOT NULL DEFAULT 0,
			bank_name TEXT NOT NULL,
			employee_count INTEGER NOT NULL DEFAULT 0,
			total_paid REAL NOT NULL DEFAULT 0,
			transfer_count INTEGER NOT NULL DEFAULT 0,
			computed_at_millis INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (project_id, bank_name)
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("ddl %q: %v", stmt, err)
		}
	}
	return db
}

func mustExec(t *testing.T, db *gorm.DB, query string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(query, args...).Error; err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// TestScanBankUsageByEmployeeProjectBank pins the single-pass scan: paid and
// not-deleted only, one row per (employee, project, bank).
func TestScanBankUsageByEmployeeProjectBank(t *testing.T) {
	db := newBankUsageSummaryTestDB(t)
	ctx := context.Background()

	mustExec(t, db, `INSERT INTO banks (id, branch_name) VALUES (1, 'MB'), (2, 'VCB')`)
	mustExec(t, db, `INSERT INTO employees (id, bank_id, deleted_at) VALUES
		(1, 1, NULL), (2, 1, NULL), (3, 2, NULL), (4, 1, NULL),
		(5, 1, '2026-01-01 00:00:00')`)
	mustExec(t, db, `INSERT INTO timesheets (id, employee_id, project_id, deleted_at, payment_status, paid_amount) VALUES
		(1, 1, 10, NULL, 'paid', 100),
		(2, 1, 10, NULL, 'paid', 100),
		(3, 1, 10, '2026-01-01 00:00:00', 'paid', 500),
		(4, 2, 10, NULL, 'paid', 100),
		(5, 2, 20, NULL, 'paid', 50),
		(6, 3, 20, NULL, 'paid', 200),
		(7, 3, 20, '2026-01-02 00:00:00', 'paid', 999),
		(8, 4, 90, NULL, 'paid', 999),
		(9, 5, 10, NULL, 'paid', 999),
		(10, 1, 10, NULL, 'pending', 999)`)

	repo := NewTimesheetAnalyticsRepository(db)
	rows, err := repo.ScanBankUsageByEmployeeProjectBank(ctx)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	type key struct {
		employee, project uint64
		bank              string
	}
	got := map[key]EmployeeProjectBankUsageRow{}
	for _, row := range rows {
		got[key{row.EmployeeID, row.ProjectID, row.BankName}] = row
	}

	want := map[key]struct {
		paid float64
		n    int64
	}{
		{1, 10, "MB"}:  {200, 2},
		{2, 10, "MB"}:  {100, 1},
		{2, 20, "MB"}:  {50, 1},
		{3, 20, "VCB"}: {200, 1},
		{4, 90, "MB"}:  {999, 1},
	}
	if len(got) != len(want) {
		t.Fatalf("scan returned %d buckets, want %d: %+v", len(got), len(want), got)
	}
	for k, w := range want {
		row, ok := got[k]
		if !ok {
			t.Fatalf("missing bucket %+v", k)
		}
		if row.TotalPaid != w.paid || row.TransferCount != w.n {
			t.Errorf("bucket %+v: got paid=%v n=%d, want paid=%v n=%d", k, row.TotalPaid, row.TransferCount, w.paid, w.n)
		}
	}
}

// TestBankUsageSummaryRoundtrip pins the snapshot swap: replace wipes the old
// scope set, reads return the stored buckets and the staleness stamp.
func TestBankUsageSummaryRoundtrip(t *testing.T) {
	db := newBankUsageSummaryTestDB(t)
	ctx := context.Background()
	repo := NewTimesheetAnalyticsRepository(db)

	first := time.Now()
	rows := []BankUsageSummaryRow{
		{ProjectID: 0, BankName: "MB", EmployeeCount: 2, TotalPaid: 350, TransferCount: 4},
		{ProjectID: 0, BankName: "VCB", EmployeeCount: 1, TotalPaid: 200, TransferCount: 1},
		{ProjectID: 10, BankName: "MB", EmployeeCount: 2, TotalPaid: 300, TransferCount: 3},
	}
	if err := repo.ReplaceBankUsageSummary(ctx, rows, first); err != nil {
		t.Fatalf("replace: %v", err)
	}

	got, computed, err := repo.GetBankUsageSummary(ctx, nil)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(got), got)
	}
	if computed != first.UnixMilli() {
		t.Fatalf("computed millis %d, want %d", computed, first.UnixMilli())
	}

	scope := uint(10)
	got, _, err = repo.GetBankUsageSummary(ctx, &scope)
	if err != nil {
		t.Fatalf("get scope: %v", err)
	}
	if len(got) != 1 || got[0].BankName != "MB" || got[0].TotalPaid != 300 {
		t.Fatalf("scope 10 rows wrong: %+v", got)
	}

	// A second replace swaps the whole snapshot: scope 10 disappears.
	if err := repo.ReplaceBankUsageSummary(ctx, rows[:1], time.Now()); err != nil {
		t.Fatalf("replace 2: %v", err)
	}
	got, _, err = repo.GetBankUsageSummary(ctx, nil)
	if err != nil {
		t.Fatalf("get after swap: %v", err)
	}
	if len(got) != 1 || got[0].ProjectID != 0 {
		t.Fatalf("snapshot not swapped: %+v", got)
	}
}
