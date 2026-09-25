package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newFlexibleAssignmentTestDB builds the minimal project_employees / projects
// schema the gate queries, seeded via raw SQL so the test stays hermetic
// (domain.User's MySQL enum blocks sqlite AutoMigrate of the full graph).
func newFlexibleAssignmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	const schema = `
CREATE TABLE projects (
	id INTEGER PRIMARY KEY,
	deleted_at DATETIME
);
CREATE TABLE project_employees (
	id INTEGER PRIMARY KEY,
	employee_id INTEGER,
	project_id INTEGER,
	payment_schedule TEXT,
	deleted_at DATETIME
);
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

func seedAssignment(t *testing.T, db *gorm.DB, employeeID uint, schedule string) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("raw db handle: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO projects (id) VALUES ((SELECT COALESCE(MAX(id), 0) + 1 FROM projects))`,
	); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO project_employees (id, employee_id, project_id, payment_schedule) VALUES
			((SELECT COALESCE(MAX(id), 0) + 1 FROM project_employees), ?, (SELECT MAX(id) FROM projects), ?)`,
		employeeID, schedule,
	); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
}

// TestHasFlexibleAssignment mirrors the adv_partner edit gate to the
// advance-payments employee list scope. Production regression: FlexPay
// imports auto-create projects under the importing admin (created_by=1,
// no project_users row), so the old creator-or-member gate denied every
// partner save with NotFound ("Không thể lưu thông tin"). The gate must
// accept any live flexible assignment and reject weekly/soft-deleted ones.
func TestHasFlexibleAssignment(t *testing.T) {
	db := newFlexibleAssignmentTestDB(t)
	repo := NewProjectEmployeeRepository(&Database{DB: db})
	ctx := context.Background()

	seedAssignment(t, db, 101, string(domain.PaymentScheduleFlexible))
	seedAssignment(t, db, 102, "weekly")

	sqlDB, _ := db.DB()
	if _, err := sqlDB.Exec(
		`INSERT INTO project_employees (id, employee_id, project_id, payment_schedule, deleted_at)
		 VALUES (900, 103, (SELECT MAX(id) FROM projects), 'flexible', datetime('now'))`,
	); err != nil {
		t.Fatalf("seed soft-deleted assignment: %v", err)
	}

	cases := []struct {
		name       string
		employeeID uint
		want       bool
	}{
		{"flexible assignment grants access", 101, true},
		{"weekly-only employee denied", 102, false},
		{"soft-deleted flexible assignment denied", 103, false},
		{"unknown employee denied", 999, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.HasFlexibleAssignment(ctx, tc.employeeID)
			if err != nil {
				t.Fatalf("HasFlexibleAssignment: %v", err)
			}
			if got != tc.want {
				t.Fatalf("employee %d: HasFlexibleAssignment = %v, want %v", tc.employeeID, got, tc.want)
			}
		})
	}
}
