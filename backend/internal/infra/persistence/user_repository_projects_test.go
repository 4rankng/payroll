package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUserListProjects covers the Du an dimension of the user list: the batch
// current-project-names lookup and the project filter both respect the active
// assignment window (started, not yet ended) and soft-deleted rows.
func TestUserListProjects(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			role TEXT NOT NULL,
			deleted_at DATETIME,
			last_login DATETIME
		)`,
		`CREATE TABLE employees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE TABLE project_employees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL,
			project_id INTEGER NOT NULL,
			payment_schedule TEXT,
			start_date DATETIME,
			last_date DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			deleted_at DATETIME
		)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}

	now := time.Now()
	seed := []string{
		`INSERT INTO users (id, username, role) VALUES (1, 'u1', 'employee'), (2, 'u2', 'employee')`,
		`INSERT INTO employees (id, user_id) VALUES (1, 1), (2, 2)`,
		`INSERT INTO projects (id, name) VALUES (1, 'LGD'), (2, 'Samsung SDS'), (3, 'Da xoa')`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	assignments := []struct {
		id         uint
		employeeID uint
		projectID  uint
		startDate  time.Time
		lastDate   *time.Time
		deletedAt  *time.Time
	}{
		{1, 1, 1, now.Add(-30 * 24 * time.Hour), nil, nil},
		{2, 1, 2, now.Add(-30 * 24 * time.Hour), nil, nil},
		{3, 2, 2, now.Add(-30 * 24 * time.Hour), new(time.Now().Add(-24 * time.Hour)), nil}, // ended
		{4, 2, 3, now.Add(-30 * 24 * time.Hour), nil, nil},                                  // project deleted
	}
	for _, a := range assignments {
		if err := db.Exec(
			`INSERT INTO project_employees (id, employee_id, project_id, payment_schedule, start_date, last_date, deleted_at)
			 VALUES (?, ?, ?, 'weekly', ?, ?, ?)`,
			a.id, a.employeeID, a.projectID, a.startDate, a.lastDate, a.deletedAt,
		).Error; err != nil {
			t.Fatalf("insert assignment %d: %v", a.id, err)
		}
	}
	if err := db.Exec(`UPDATE projects SET deleted_at = ? WHERE id = 3`, now).Error; err != nil {
		t.Fatalf("soft-delete project: %v", err)
	}

	repo := NewUserRepository(&Database{DB: db})
	ctx := context.Background()

	names, err := repo.GetCurrentProjectNamesByUserIDs(ctx, []uint{1, 2})
	assert.NoError(t, err)
	assert.Equal(t, map[uint][]string{1: {"LGD", "Samsung SDS"}}, names,
		"active assignments only; ended/deleted excluded; absent users omitted")

	all, total, err := repo.ListWithFilters(ctx, nil, "", nil, false, nil, "id", "asc", 50, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, all, 2)

	p1 := uint(1)
	byProject, total, err := repo.ListWithFilters(ctx, nil, "", nil, false, &p1, "id", "asc", 50, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, byProject, 1)
	assert.Equal(t, uint(1), byProject[0].ID)

	p2 := uint(2)
	byProject, total, err = repo.ListWithFilters(ctx, nil, "", nil, false, &p2, "id", "asc", 50, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total, "user 2's Samsung assignment has ended")
	assert.Equal(t, uint(1), byProject[0].ID)

	p3 := uint(3)
	byProject, total, err = repo.ListWithFilters(ctx, nil, "", nil, false, &p3, "id", "asc", 50, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total, "soft-deleted project matches nothing")
	assert.Empty(t, byProject)
}
