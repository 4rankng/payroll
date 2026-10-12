package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCountActiveEmployeesByProject covers the per-project active-login
// aggregation: distinct employees per current project assignment, excluded
// when the login is outside the window, the assignment has ended, or the
// project is soft-deleted. An employee active in two projects counts once in
// each.
func TestCountActiveEmployeesByProject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	inWindow := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	outWindow := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)

	users := []struct {
		id        uint
		role      string
		lastLogin time.Time
	}{
		{1, "employee", inWindow},
		{2, "employee", inWindow.Add(24 * time.Hour)},
		{3, "employee", inWindow.Add(48 * time.Hour)},
		{4, "employee", inWindow.Add(72 * time.Hour)},
		{5, "employee", outWindow},
	}
	for _, u := range users {
		if err := db.Exec(
			`INSERT INTO users (id, role, last_login) VALUES (?, ?, ?)`,
			u.id, u.role, u.lastLogin,
		).Error; err != nil {
			t.Fatalf("insert user %d: %v", u.id, err)
		}
	}

	projects := []struct {
		id        uint
		name      string
		deletedAt *time.Time
	}{
		{1, "Nha may VERICO", nil},
		{2, "San xuat xa phong", nil},
		{3, "Da ket thuc", nil},
		{4, "Da xoa", new(now)},
	}
	for _, p := range projects {
		if err := db.Exec(
			`INSERT INTO projects (id, name, deleted_at) VALUES (?, ?, ?)`,
			p.id, p.name, p.deletedAt,
		).Error; err != nil {
			t.Fatalf("insert project %d: %v", p.id, err)
		}
	}

	for id := uint(1); id <= 5; id++ {
		if err := db.Exec(
			`INSERT INTO employees (id, user_id) VALUES (?, ?)`, id, id,
		).Error; err != nil {
			t.Fatalf("insert employee %d: %v", id, err)
		}
	}

	assignments := []struct {
		id         uint
		employeeID uint
		projectID  uint
		startDate  time.Time
		lastDate   *time.Time
	}{
		{1, 1, 1, now.Add(-30 * 24 * time.Hour), nil},
		{2, 1, 2, now.Add(-30 * 24 * time.Hour), nil},
		{3, 2, 1, now.Add(-30 * 24 * time.Hour), nil},
		{4, 3, 3, now.Add(-30 * 24 * time.Hour), new(now.Add(-24 * time.Hour))},
		{5, 4, 4, now.Add(-30 * 24 * time.Hour), nil},
		{6, 5, 1, now.Add(-30 * 24 * time.Hour), nil},
	}
	for _, a := range assignments {
		if err := db.Exec(
			`INSERT INTO project_employees (id, employee_id, project_id, payment_schedule, start_date, last_date)
			 VALUES (?, ?, ?, 'weekly', ?, ?)`,
			a.id, a.employeeID, a.projectID, a.startDate, a.lastDate,
		).Error; err != nil {
			t.Fatalf("insert assignment %d: %v", a.id, err)
		}
	}

	repo := NewUserRepository(&Database{DB: db})
	counts, err := repo.CountActiveEmployeesByProject(context.Background(), since, until)
	if err != nil {
		t.Fatalf("CountActiveEmployeesByProject: %v", err)
	}

	assert.Equal(t, []*domain.ProjectActiveCount{
		{ProjectID: 1, ProjectName: "Nha may VERICO", Count: 2},
		{ProjectID: 2, ProjectName: "San xuat xa phong", Count: 1},
	}, counts)
}
