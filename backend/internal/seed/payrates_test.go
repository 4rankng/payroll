package seed

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedPayratesPersistsUsableConfiguration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, statement := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, deleted_at DATETIME)`,
		`CREATE TABLE projects (id INTEGER PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE project_employees (id INTEGER PRIMARY KEY, project_id INTEGER, position TEXT, deleted_at DATETIME)`,
		`CREATE TABLE payrates (id INTEGER PRIMARY KEY, project_id INTEGER, payrate_json TEXT, from_date DATETIME, to_date DATETIME, deleted_at DATETIME, created_by INTEGER, created_at DATETIME, updated_at DATETIME)`,
		`INSERT INTO users (id, username) VALUES (1, 'admin')`,
		`INSERT INTO projects (id, name) VALUES (1, 'Synthetic project')`,
		`INSERT INTO project_employees (id, project_id, position) VALUES (1, 1, 'phổ thông')`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}

	require.NoError(t, NewSeeder(db).seedPayrates(context.Background(), db))
	var stored string
	require.NoError(t, db.Raw("SELECT payrate_json FROM payrates WHERE project_id = 1").Scan(&stored).Error)
	rates, err := domain.PayrateConfiguration(stored).Flatten()
	require.NoError(t, err)
	require.NotEmpty(t, rates)
	for _, rate := range rates {
		require.Positive(t, rate)
	}
}
