package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestProjectEmployeeRepository_UpdatePositionIfCurrent(t *testing.T) {
	dsn := fmt.Sprintf("file:project-employee-cas-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE project_employees (
		id INTEGER PRIMARY KEY,
		position TEXT NOT NULL,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec("INSERT INTO project_employees (id, position) VALUES (?, ?)", 1, "Lương 520").Error)

	repo := &ProjectEmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	ctx := context.Background()

	require.NoError(t, repo.UpdatePositionIfCurrent(ctx, 1, "Lương 520", "Lương 700"))

	var position string
	require.NoError(t, db.Raw("SELECT position FROM project_employees WHERE id = ?", 1).Scan(&position).Error)
	require.Equal(t, "Lương 700", position)

	err = repo.UpdatePositionIfCurrent(ctx, 1, "Lương 520", "Lương 900")
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrConflict)
	require.NoError(t, db.Raw("SELECT position FROM project_employees WHERE id = ?", 1).Scan(&position).Error)
	require.Equal(t, "Lương 700", position)
}

func TestProjectEmployeeRepository_GetActiveAssignmentsUsesTransactionContext(t *testing.T) {
	dsn := fmt.Sprintf("file:project-employee-active-assignments-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE project_employees (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			employee_name TEXT NOT NULL,
			employee_cccd TEXT NOT NULL,
			position TEXT NOT NULL,
			start_date DATETIME NOT NULL,
			last_date DATETIME,
			payment_schedule TEXT NOT NULL,
			check_in_enabled BOOLEAN NOT NULL DEFAULT 0,
			created_by INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE projects (id INTEGER PRIMARY KEY, deleted_at DATETIME);
		CREATE TABLE employees (id INTEGER PRIMARY KEY, deleted_at DATETIME);
		CREATE TABLE users (id INTEGER PRIMARY KEY, deleted_at DATETIME);
	`).Error)

	repo := &ProjectEmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer func() { _ = tx.Rollback().Error }()

	require.NoError(t, tx.Exec(`
		INSERT INTO project_employees (
			id, project_id, employee_id, employee_name, employee_cccd, position,
			start_date, payment_schedule, check_in_enabled, created_by
		) VALUES (1, 74, 1288, 'Nhân viên thử nghiệm', '031095000444', 'Lương 520', ?, 'weekly', 0, 1353)
	`, time.Now().AddDate(0, -1, 0)).Error)

	ctx := transactionContextForTest(tx)
	assignments, err := repo.GetActiveAssignments(ctx, 74)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, uint(1288), assignments[0].EmployeeID)
}

func TestWithAssignmentUpdateLockUsesTheSharedTransactionProtocol(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	unlocked := withAssignmentUpdateLock(context.Background(), db.Model(&domain.ProjectEmployee{}))
	_, hasLock := unlocked.Statement.Clauses["FOR"]
	require.False(t, hasLock)

	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer func() { _ = tx.Rollback().Error }()
	locked := withAssignmentUpdateLock(transactionContextForTest(tx), tx.Model(&domain.ProjectEmployee{}))
	lockClause, hasLock := locked.Statement.Clauses["FOR"]
	require.True(t, hasLock)
	require.Equal(t, clause.Locking{Strength: "UPDATE"}, lockClause.Expression)
}

func TestProjectEmployeeRepository_HasActiveFlexiblePaymentScheduleByEmployeeID(t *testing.T) {
	dsn := fmt.Sprintf("file:project-employee-flexible-payment-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE project_employees (
			id INTEGER PRIMARY KEY,
			employee_id INTEGER NOT NULL,
			start_date DATETIME NOT NULL,
			last_date DATETIME,
			payment_schedule TEXT NOT NULL,
			deleted_at DATETIME
		)
	`).Error)

	today := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO project_employees (id, employee_id, start_date, last_date, payment_schedule) VALUES
			(1, 201, ?, NULL, 'flexible'),
			(2, 202, ?, ?, 'flexible'),
			(3, 203, ?, NULL, 'weekly')
	`, today.AddDate(0, 0, -2), today.AddDate(0, 0, -7), today.AddDate(0, 0, -1), today.AddDate(0, 0, -2)).Error)

	repo := &ProjectEmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	hasFlexible, err := repo.HasActiveFlexiblePaymentScheduleByEmployeeID(context.Background(), 201)
	require.NoError(t, err)
	require.True(t, hasFlexible)

	hasFlexible, err = repo.HasActiveFlexiblePaymentScheduleByEmployeeID(context.Background(), 202)
	require.NoError(t, err)
	require.False(t, hasFlexible)

	hasFlexible, err = repo.HasActiveFlexiblePaymentScheduleByEmployeeID(context.Background(), 203)
	require.NoError(t, err)
	require.False(t, hasFlexible)
}

// TestHasAccessViaProject_ProjectCreator verifies that a user who CREATED a project
// can access an employee actively assigned to it — even when the creator has NO
// project_users row (creators are intentionally never stored in project_users; see
// ProjectPermissionService.CanUserAccessProject, which special-cases projects.created_by,
// and GetProjectUsers, which synthesizes a virtual owner entry).
//
// Regression for partner 403 "No access to this employee" on GET /employees/:id,
// where a partner (mrduc2 / user 976) created project 70 but could not open an
// employee assigned to it because HasAccessViaProject only joined project_users.
func TestHasAccessViaProject_ProjectCreator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}
	repo := &ProjectEmployeeRepository{BaseRepository: &BaseRepository{DB: db}}
	ctx := context.Background()

	const (
		creatorID  = uint(999950)
		memberID   = uint(999951)
		outsiderID = uint(999952)
		projectID  = uint(999950)
		empActive  = uint(999950) // active assignment (no last_date)
		empEnded   = uint(999951) // ended assignment (last_date in the past)
	)

	pastDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	cleanup := func() {
		db.Exec("DELETE FROM project_employees WHERE project_id = ?", projectID)
		db.Exec("DELETE FROM project_users WHERE project_id = ?", projectID)
		db.Exec("DELETE FROM projects WHERE id = ?", projectID)
		db.Exec("DELETE FROM employees WHERE id IN (?, ?)", empActive, empEnded)
		db.Exec("DELETE FROM users WHERE id IN (?, ?, ?)", creatorID, memberID, outsiderID)
	}
	cleanup() // remove any stale rows from a prior run
	defer cleanup()

	// Users: project creator, explicit project member, and an outsider.
	for _, u := range []struct {
		id   uint
		name string
	}{
		{creatorID, "test_pe_creator"},
		{memberID, "test_pe_member"},
		{outsiderID, "test_pe_outsider"},
	} {
		if err := db.Exec(
			"INSERT INTO users (id, username, password, fullname, role) VALUES (?, ?, 'x', ?, 'partner')",
			u.id, u.name, u.name,
		).Error; err != nil {
			t.Fatalf("setup users: %v", err)
		}
	}

	// Project created by creatorID — deliberately NO project_users row for the creator.
	if err := db.Exec(
		"INSERT INTO projects (id, name, total_payout_vnd, pending_payable_vnd, pending_receivable_vnd, total_received_vnd, project_status, created_by, off_days, is_flexible) VALUES (?, 'test proj', 0, 0, 0, 0, 'active', ?, 0, 0)",
		projectID, creatorID,
	).Error; err != nil {
		t.Fatalf("setup project: %v", err)
	}

	for _, e := range []uint{empActive, empEnded} {
		if err := db.Exec(
			"INSERT INTO employees (id, fullname, cccd, created_by) VALUES (?, 'test emp', '000000000000', ?)",
			e, creatorID,
		).Error; err != nil {
			t.Fatalf("setup employee %d: %v", e, err)
		}
	}

	// Active assignment (last_date NULL).
	if err := db.Exec(
		"INSERT INTO project_employees (project_id, employee_id, employee_name, employee_cccd, position, start_date, created_by, payment_schedule, check_in_enabled) VALUES (?, ?, 'n', 'c', 'p', ?, ?, 'weekly', 0)",
		projectID, empActive, pastDate, creatorID,
	).Error; err != nil {
		t.Fatalf("setup active assignment: %v", err)
	}
	// Ended assignment (last_date in the past) — active-window must still be enforced.
	if err := db.Exec(
		"INSERT INTO project_employees (project_id, employee_id, employee_name, employee_cccd, position, start_date, last_date, created_by, payment_schedule, check_in_enabled) VALUES (?, ?, 'n', 'c', 'p', ?, ?, ?, 'weekly', 0)",
		projectID, empEnded, pastDate, pastDate, creatorID,
	).Error; err != nil {
		t.Fatalf("setup ended assignment: %v", err)
	}

	// Explicit project member (memberID) gets a project_users row.
	if err := db.Exec(
		"INSERT INTO project_users (project_id, user_id, granted_by) VALUES (?, ?, ?)",
		projectID, memberID, creatorID,
	).Error; err != nil {
		t.Fatalf("setup project_users: %v", err)
	}

	// 1. Creator with no project_users row must access the actively-assigned employee.
	got, err := repo.HasAccessViaProject(ctx, empActive, creatorID)
	if err != nil {
		t.Fatalf("creator HasAccessViaProject: %v", err)
	}
	if !got {
		t.Fatal("project creator should access an employee actively assigned to their project")
	}

	// 2. Explicit project member must still access (no regression in the member path).
	got, err = repo.HasAccessViaProject(ctx, empActive, memberID)
	if err != nil {
		t.Fatalf("member HasAccessViaProject: %v", err)
	}
	if !got {
		t.Fatal("explicit project member should access an employee assigned to the project")
	}

	// 3. Outsider (not creator, not member) must NOT access.
	got, err = repo.HasAccessViaProject(ctx, empActive, outsiderID)
	if err != nil {
		t.Fatalf("outsider HasAccessViaProject: %v", err)
	}
	if got {
		t.Fatal("outsider should NOT access an employee they have no relation to")
	}

	// 4. Creator must NOT access via an ENDED assignment (active window still enforced).
	got, err = repo.HasAccessViaProject(ctx, empEnded, creatorID)
	if err != nil {
		t.Fatalf("ended HasAccessViaProject: %v", err)
	}
	if got {
		t.Fatal("creator should NOT access an employee whose only assignment has ended")
	}
}

func TestProjectEmployeeRepository_SearchVietnameseAndCountBeforePagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}
	repo := &ProjectEmployeeRepository{
		BaseRepository: &BaseRepository{DB: db},
		filterBuilder:  common.NewFilterBuilder(db),
	}
	ctx := context.Background()

	const (
		creatorID = uint(999970)
		projectID = uint(999970)
	)
	employeeIDs := []uint{999970, 999971, 999972}

	cleanup := func() {
		db.Exec("DELETE FROM project_employees WHERE project_id = ?", projectID)
		db.Exec("DELETE FROM projects WHERE id = ?", projectID)
		db.Exec("DELETE FROM employees WHERE id IN ?", employeeIDs)
		db.Exec("DELETE FROM users WHERE id = ?", creatorID)
	}
	cleanup()
	defer cleanup()

	if err := db.Exec(
		"INSERT INTO users (id, username, password, fullname, role) VALUES (?, 'test_pe_search', 'x', 'Search Owner', 'admin')",
		creatorID,
	).Error; err != nil {
		t.Fatalf("setup user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO projects (id, name, total_payout_vnd, pending_payable_vnd, pending_receivable_vnd, total_received_vnd, project_status, created_by, off_days, is_flexible) VALUES (?, 'Search Project', 0, 0, 0, 0, 'active', ?, 0, 1)",
		projectID, creatorID,
	).Error; err != nil {
		t.Fatalf("setup project: %v", err)
	}

	names := []string{"Việt Duy", "Trần Việt Duy", "Nguyễn Văn Khác"}
	for index, employeeID := range employeeIDs {
		cccd := "09999999997" + string(rune('0'+index))
		if err := db.Exec(
			"INSERT INTO employees (id, fullname, cccd, created_by) VALUES (?, ?, ?, ?)",
			employeeID, names[index], cccd, creatorID,
		).Error; err != nil {
			t.Fatalf("setup employee %d: %v", employeeID, err)
		}
		if err := db.Exec(
			"INSERT INTO project_employees (project_id, employee_id, employee_name, employee_cccd, position, start_date, created_by, payment_schedule, check_in_enabled) VALUES (?, ?, ?, ?, 'phổ thông', '2026-01-01', ?, 'weekly', 0)",
			projectID, employeeID, names[index], cccd, creatorID,
		).Error; err != nil {
			t.Fatalf("setup assignment %d: %v", employeeID, err)
		}
	}

	projectIDFilter := projectID
	filters := domain.ProjectEmployeeFilters{
		ProjectID: &projectIDFilter,
		Status:    "current",
		Search:    "viet duy",
		Limit:     1,
	}

	results, err := repo.List(ctx, filters)
	if err != nil {
		t.Fatalf("search list: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("paginated search returned %d rows, want 1", len(results))
	}

	total, err := repo.Count(ctx, filters)
	if err != nil {
		t.Fatalf("search count: %v", err)
	}
	if total != 2 {
		t.Fatalf("accent-insensitive search total = %d, want 2", total)
	}
}

func newCheckInConfigurationTestRepository(t *testing.T) (*ProjectEmployeeRepository, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:project-employee-checkin-configuration-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE project_employees (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			employee_name TEXT NOT NULL,
			employee_cccd TEXT NOT NULL,
			employee_code TEXT,
			start_date DATETIME NOT NULL,
			last_date DATETIME,
			check_in_enabled BOOLEAN NOT NULL DEFAULT 0,
			pending_check_in_enabled BOOLEAN,
			check_in_effective_from DATETIME,
			created_at DATETIME NOT NULL,
			deleted_at DATETIME
		);
		CREATE TABLE attendances (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			check_in_time DATETIME NOT NULL
		);
	`).Error)

	repo := &ProjectEmployeeRepository{
		BaseRepository: &BaseRepository{DB: db},
		filterBuilder:  common.NewFilterBuilder(db),
	}
	return repo, db
}

func seedCheckInConfigurationTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Exec(`
		INSERT INTO project_employees (
			id, project_id, employee_id, employee_name, employee_cccd,
			employee_code, start_date, last_date, check_in_enabled,
			pending_check_in_enabled, created_at
		) VALUES
			(1, 7, 101, 'Nguyễn Hoàng An', '001201000101', 'NV-101', '2026-01-01', NULL, 1, NULL, '2026-01-01 00:00:00'),
			(2, 7, 102, 'Trần Bình Minh', '001201000102', 'NV-102', '2026-01-01', NULL, 1, NULL, '2026-01-01 00:00:00'),
			(3, 7, 103, 'Lê Chi Lan', '001201000103', 'NV-103', '2026-01-01', NULL, 0, NULL, '2026-01-01 00:00:00'),
			(4, 7, 104, 'Phạm Duy Khang', '001201000104', 'NV-104', '2026-01-01', NULL, 0, 1, '2026-01-01 00:00:00'),
			(5, 7, 105, 'Vũ Gia Hân', '001201000105', 'NV-105', '2026-01-01', '2026-07-31', 1, NULL, '2026-01-01 00:00:00'),
			(6, 8, 106, 'Đỗ Hải Nam', '001201000106', 'NV-106', '2026-01-01', NULL, 1, NULL, '2026-01-01 00:00:00')
	`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO attendances (id, project_id, employee_id, check_in_time) VALUES
			(1, 7, 101, '2026-08-03 08:00:00'),
			(2, 7, 101, '2026-08-10 08:15:00'),
			(3, 7, 102, '2026-07-31 23:59:59'),
			(4, 7, 103, '2026-08-05 08:00:00'),
			(5, 8, 106, '2026-08-06 08:00:00')
	`).Error)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationClassifiesCurrentMonth(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)

	result, err := repo.GetCheckInConfiguration(context.Background(), domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusEnabled,
		Limit:      50,
	})
	require.NoError(t, err)

	require.Equal(t, domain.CheckInConfigurationSummary{
		Enabled:  2,
		Active:   1,
		Inactive: 1,
		Pending:  1,
	}, result.Summary)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, result.Employees, 2)
	require.Equal(t, uint(101), result.Employees[0].EmployeeID)
	require.Equal(t, int64(2), result.Employees[0].AttendanceCount)
	require.NotNil(t, result.Employees[0].LastCheckInAt)
	require.Equal(t, uint(102), result.Employees[1].EmployeeID)
	require.Zero(t, result.Employees[1].AttendanceCount)
	require.Nil(t, result.Employees[1].LastCheckInAt)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationUsesSelectedMonthForUsageData(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)
	require.NoError(t, db.Exec(`
		INSERT INTO attendances (id, project_id, employee_id, check_in_time)
		VALUES (6, 7, 101, '2026-07-15 08:30:00')
	`).Error)

	result, err := repo.GetCheckInConfiguration(context.Background(), domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusEnabled,
		Limit:      50,
	})
	require.NoError(t, err)

	require.Equal(t, domain.CheckInConfigurationSummary{
		Enabled:  2,
		Active:   2,
		Inactive: 0,
		Pending:  1,
	}, result.Summary)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, result.Employees, 2)

	employeesByID := make(map[uint]domain.CheckInConfigurationEmployee, len(result.Employees))
	for _, employee := range result.Employees {
		employeesByID[employee.EmployeeID] = employee
	}
	require.Equal(t, int64(1), employeesByID[101].AttendanceCount)
	require.NotNil(t, employeesByID[101].LastCheckInAt)
	require.Equal(t, int64(1), employeesByID[102].AttendanceCount)
	require.NotNil(t, employeesByID[102].LastCheckInAt)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationFiltersInactiveAcrossAllRows(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)

	result, err := repo.GetCheckInConfiguration(context.Background(), domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusInactive,
	})
	require.NoError(t, err)

	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Employees, 1)
	require.Equal(t, uint(102), result.Employees[0].EmployeeID)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationListsAllCurrentAssignments(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)

	result, err := repo.GetCheckInConfiguration(context.Background(), domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusAll,
		Limit:      50,
	})
	require.NoError(t, err)

	require.Equal(t, int64(4), result.Total)
	require.Len(t, result.Employees, 4)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationPaginatesInDatabase(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)

	query := domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusAll,
		Limit:      2,
	}

	firstPage, err := repo.GetCheckInConfiguration(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, int64(4), firstPage.Total)
	require.Len(t, firstPage.Employees, 2)

	query.Offset = 2
	secondPage, err := repo.GetCheckInConfiguration(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, int64(4), secondPage.Total)
	require.Len(t, secondPage.Employees, 2)

	seen := make(map[uint]struct{}, 4)
	for _, employee := range append(firstPage.Employees, secondPage.Employees...) {
		if _, duplicate := seen[employee.EmployeeID]; duplicate {
			t.Fatalf("employee %d appeared on more than one server page", employee.EmployeeID)
		}
		seen[employee.EmployeeID] = struct{}{}
	}
	require.Len(t, seen, 4)
}

func TestProjectEmployeeRepository_GetCheckInConfigurationUsesLatestCurrentAssignmentPerEmployee(t *testing.T) {
	repo, db := newCheckInConfigurationTestRepository(t)
	seedCheckInConfigurationTestData(t, db)
	require.NoError(t, db.Exec(`
		INSERT INTO project_employees (
			id, project_id, employee_id, employee_name, employee_cccd,
			employee_code, start_date, last_date, check_in_enabled,
			pending_check_in_enabled, created_at
		) VALUES (
			7, 7, 102, 'Trần Bình Minh', '001201000102', 'NV-102-new',
			'2026-02-01', NULL, 0, NULL, '2026-02-01 00:00:00'
		)
	`).Error)

	result, err := repo.GetCheckInConfiguration(context.Background(), domain.CheckInConfigurationQuery{
		ProjectID:  7,
		MonthStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		MonthEnd:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		AsOfDate:   time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		Status:     domain.CheckInConfigurationStatusAll,
	})
	require.NoError(t, err)
	require.Equal(t, int64(4), result.Total)
	require.Equal(t, int64(1), result.Summary.Enabled)
	require.Equal(t, int64(1), result.Summary.Active)
	require.Zero(t, result.Summary.Inactive)

	var employee102 []domain.CheckInConfigurationEmployee
	for _, employee := range result.Employees {
		if employee.EmployeeID == 102 {
			employee102 = append(employee102, employee)
		}
	}
	require.Len(t, employee102, 1)
	require.Equal(t, uint(7), employee102[0].AssignmentID)
	require.False(t, employee102[0].CheckInEnabled)
}
