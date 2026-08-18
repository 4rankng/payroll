package query_builders

import (
	"strings"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildMissingBankDetailsQueryIncludesEveryRequiredBankField(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)

	builder := NewEmployeeProjectQueryBuilder(db)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		builder.db = tx
		return builder.BuildMissingBankDetailsQuery(domain.EmployeeFilters{}).
			Find(&[]domain.Employee{})
	})

	normalizedSQL := strings.ToLower(sql)
	assert.Contains(t, normalizedSQL, "employees.bank_id is null")
	assert.Contains(t, normalizedSQL, "coalesce(trim(employees.bank_account_number), '') = ''")
	assert.Contains(t, normalizedSQL, "coalesce(trim(employees.bank_account_name), '') = ''")
	assert.Contains(t, normalizedSQL, "employees.bank_account_status = 'invalid'")
	assert.NotContains(t, normalizedSQL, "employees.bank_account_status = 'unverified'")
	// Pin the branch structure: pending-work EXISTS clauses must stay inside
	// the missing-fields branch, NOT wrap the invalid branch — otherwise
	// invalid rows would vanish again once their work is approved/paid.
	invalidIdx := strings.Index(normalizedSQL, "employees.bank_account_status = 'invalid'")
	pendingIdx := strings.Index(normalizedSQL, "t.timesheet_status = 'pending_approval'")
	assert.GreaterOrEqual(t, invalidIdx, 0)
	assert.GreaterOrEqual(t, pendingIdx, 0)
	assert.Less(t, invalidIdx, pendingIdx, "invalid branch must precede the pending-work EXISTS clauses")
	assert.Contains(t, normalizedSQL, "apr.status = 'pending'")
}

// TestMissingBankDetailsPredicateHandlesNullAndEmptyValues keeps guarding the
// missing-fields predicate in isolation: only incomplete bank info matches.
func TestMissingBankDetailsPredicateHandlesNullAndEmptyValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			bank_id INTEGER NULL,
			bank_account_number TEXT NULL,
			bank_account_name TEXT NULL,
			bank_account_status TEXT NULL
		)
	`).Error)

	rows := []struct {
		id            int
		bankID        any
		accountNumber any
		accountName   any
		status        any
	}{
		{1, 10, nil, "NGUYEN VAN A", "valid"},
		{2, 10, "123", nil, "valid"},
		{3, 10, "   ", "NGUYEN VAN A", "valid"},
		{4, 10, "123", "NGUYEN VAN A", "invalid"},
		{5, nil, "123", "NGUYEN VAN A", "valid"},
		{6, 10, "123", "NGUYEN VAN A", "unverified"},
		{7, 10, "123", "NGUYEN VAN A", "valid"},
	}
	for _, row := range rows {
		require.NoError(t, db.Exec(
			`INSERT INTO employees
				(id, bank_id, bank_account_number, bank_account_name, bank_account_status)
			 VALUES (?, ?, ?, ?, ?)`,
			row.id,
			row.bankID,
			row.accountNumber,
			row.accountName,
			row.status,
		).Error)
	}

	var ids []int
	require.NoError(t, db.Table("employees").
		Where(missingBankFieldsPredicate).
		Order("id").
		Pluck("id", &ids).Error)

	// Only incomplete bank info matches; status alone no longer does.
	assert.Equal(t, []int{1, 2, 3, 5}, ids)
}

// missingBankSchemas creates the tables the full base query touches. SQLite
// stands in for MySQL here — CURDATE() is MySQL-only, so the date condition
// is expressed with date('now') in the test's subquery shim below.
func missingBankSchemas(t *testing.T, db *gorm.DB) {
	require.NoError(t, db.Exec(`
		CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			created_at DATETIME,
			deleted_at DATETIME,
			bank_id INTEGER NULL,
			bank_account_number TEXT NULL,
			bank_account_name TEXT NULL,
			bank_account_status TEXT NULL
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE project_employees (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			start_date DATE NOT NULL,
			last_date DATE NULL,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE timesheets (
			id INTEGER PRIMARY KEY,
			employee_id INTEGER NOT NULL,
			timesheet_status TEXT NOT NULL
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE advance_payment_requests (
			id INTEGER PRIMARY KEY,
			employee_id INTEGER NOT NULL,
			status TEXT NOT NULL
		)
	`).Error)
}

// TestMissingBankDetailsBaseQueryBranchSemantics covers the full two-branch
// WHERE clause of buildMissingBankDetailsBaseQuery on real sqlite rows:
//
//	branch A: bank_account_status = 'invalid' AND active project → always listed
//	branch B: missing fields AND active project AND pending work → listed
//
// The builder's MySQL-specific CURDATE() is incompatible with sqlite, so the
// test substitutes the equivalent date('now') subquery and asserts against
// the substituted SQL rather than calling the builder verbatim.
func TestMissingBankDetailsBaseQueryBranchSemantics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	missingBankSchemas(t, db)

	// The full base query as built by the builder, with CURDATE() rewritten
	// for sqlite. Mirrors buildMissingBankDetailsBaseQuery exactly except the
	// date function.
	activeProjectSubquery := db.Model(&domain.ProjectEmployee{}).
		Select("DISTINCT project_employees.employee_id").
		Joins("INNER JOIN projects p ON project_employees.project_id = p.id").
		Where("project_employees.last_date IS NULL AND project_employees.start_date <= date('now')")
	fullPredicate := `(
		(
			employees.bank_account_status = 'invalid'
			AND employees.id IN (?)
		)
		OR
		(
			` + missingBankFieldsPredicate + `
			AND employees.id IN (?)
			AND (
				EXISTS (SELECT 1 FROM timesheets t WHERE t.employee_id = employees.id AND t.timesheet_status = 'pending_approval')
				OR
				EXISTS (SELECT 1 FROM advance_payment_requests apr WHERE apr.employee_id = employees.id AND apr.status = 'PENDING')
			)
		)
	)`

	type seed struct {
		id               int
		bankID           any
		accountNumber    any
		accountName      any
		status           string
		activeAssignment bool
		endedAssignment  bool // last_date in the past
		pendingTimesheet bool
		pendingAPR       bool
	}

	seeds := []seed{
		// 1: invalid + complete + no pending + active → listed (NEW behavior)
		{id: 1, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "invalid", activeAssignment: true},
		// 2: invalid + complete + no pending + ended assignment → hidden
		{id: 2, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "invalid", endedAssignment: true},
		// 3: invalid + has pending + active → listed
		{id: 3, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "invalid", activeAssignment: true, pendingTimesheet: true},
		// 4: missing + no pending + active → hidden (unchanged)
		{id: 4, bankID: nil, accountNumber: nil, accountName: nil, status: "valid", activeAssignment: true},
		// 5: missing + pending timesheet + active → listed (unchanged)
		{id: 5, bankID: nil, accountNumber: nil, accountName: nil, status: "valid", activeAssignment: true, pendingTimesheet: true},
		// 6: unverified + complete + no pending → hidden (unchanged)
		{id: 6, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "unverified", activeAssignment: true},
		// 7: unverified + complete + pending → still hidden (unchanged)
		{id: 7, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "unverified", activeAssignment: true, pendingTimesheet: true},
		// 8: valid + complete + no pending → hidden (unchanged)
		{id: 8, bankID: 10, accountNumber: "123", accountName: "NGUYEN VAN A", status: "valid", activeAssignment: true},
		// 9: invalid + missing fields (manual DB edit) + no pending + active → listed (branch A wins)
		{id: 9, bankID: nil, accountNumber: nil, accountName: nil, status: "invalid", activeAssignment: true},
	}

	require.NoError(t, db.Exec(`INSERT INTO projects (id) VALUES (100)`).Error)
	for _, s := range seeds {
		require.NoError(t, db.Exec(
			`INSERT INTO employees (id, bank_id, bank_account_number, bank_account_name, bank_account_status)
			 VALUES (?, ?, ?, ?, ?)`,
			s.id, s.bankID, s.accountNumber, s.accountName, s.status,
		).Error)
		if s.activeAssignment {
			require.NoError(t, db.Exec(
				`INSERT INTO project_employees (project_id, employee_id, start_date, last_date)
				 VALUES (100, ?, date('now', '-30 day'), NULL)`,
				s.id,
			).Error)
		} else if s.endedAssignment {
			require.NoError(t, db.Exec(
				`INSERT INTO project_employees (project_id, employee_id, start_date, last_date)
				 VALUES (100, ?, date('now', '-60 day'), date('now', '-10 day'))`,
				s.id,
			).Error)
		}
		if s.pendingTimesheet {
			require.NoError(t, db.Exec(
				`INSERT INTO timesheets (employee_id, timesheet_status) VALUES (?, 'pending_approval')`,
				s.id,
			).Error)
		}
		if s.pendingAPR {
			require.NoError(t, db.Exec(
				`INSERT INTO advance_payment_requests (employee_id, status) VALUES (?, 'PENDING')`,
				s.id,
			).Error)
		}
	}

	var ids []int
	require.NoError(t, db.Table("employees").
		Where(fullPredicate, activeProjectSubquery, activeProjectSubquery).
		Order("id").
		Pluck("id", &ids).Error)

	// 1 (invalid, no pending — NEW), 3 (invalid+pending), 5 (missing+pending),
	// 9 (invalid wins over missing fields). 2/4/6/7/8 stay hidden.
	assert.Equal(t, []int{1, 3, 5, 9}, ids)
}
