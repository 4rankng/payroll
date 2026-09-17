package persistence

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Async imports create employees and assignments in one transaction. Coverage
// backdating and bulk validation must see those rows before the import commits.
func TestImportAssignmentReadsSeeUncommittedRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/import.db?_journal_mode=WAL"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
	CREATE TABLE project_employees (
		id INTEGER PRIMARY KEY, project_id INTEGER, employee_id INTEGER,
		employee_name TEXT, employee_cccd TEXT, employee_code TEXT, position TEXT,
		start_date DATETIME, last_date DATETIME, payment_schedule TEXT,
		pending_payment_schedule TEXT, schedule_effective_from DATETIME,
		check_in_enabled BOOLEAN, advance_request_enabled BOOLEAN,
		pending_check_in_enabled BOOLEAN, check_in_effective_from DATETIME,
		created_by INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME);
	CREATE TABLE projects (id INTEGER PRIMARY KEY, deleted_at DATETIME);
	CREATE TABLE employees (id INTEGER PRIMARY KEY, fullname TEXT, deleted_at DATETIME);
	CREATE TABLE users (id INTEGER PRIMARY KEY, deleted_at DATETIME);
	`).Error)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer func() { _ = tx.Rollback().Error }()
	require.NoError(t, tx.Exec("INSERT INTO projects (id) VALUES (7)").Error)
	require.NoError(t, tx.Exec("INSERT INTO employees (id, fullname) VALUES (8, 'Synthetic import employee')").Error)
	require.NoError(t, tx.Exec("INSERT INTO project_employees (id, project_id, employee_id, start_date) VALUES (9, 7, 8, ?)", time.Now()).Error)
	ctx := transactionContextForTest(tx)
	repo := &ProjectEmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	t.Run("assignment by ID", func(t *testing.T) {
		assignment, err := repo.GetByID(ctx, 9)
		require.NoError(t, err)
		require.Equal(t, uint(8), assignment.EmployeeID)
	})
	t.Run("bulk assignments", func(t *testing.T) {
		assignments, err := repo.GetActiveAssignmentsByProjectsAndEmployees(ctx, []uint{7}, []uint{8})
		require.NoError(t, err)
		require.Len(t, assignments, 1)
	})
	t.Run("bulk employees", func(t *testing.T) {
		employees, err := NewEmployeeRepository(&Database{DB: db}).GetByIDs(ctx, []int64{8})
		require.NoError(t, err)
		require.Len(t, employees, 1)
		require.Equal(t, "Synthetic import employee", employees[0].Fullname)
	})
	require.NoError(t, tx.Rollback().Error)
	var count int64
	require.NoError(t, db.Table("project_employees").Count(&count).Error)
	require.Zero(t, count, "import rollback must leave no assignment")
	require.NoError(t, db.Table("employees").Count(&count).Error)
	require.Zero(t, count, "import rollback must leave no employee")
}
