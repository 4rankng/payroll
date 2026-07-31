package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
)

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
