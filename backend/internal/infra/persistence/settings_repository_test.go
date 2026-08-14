package persistence

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"api-server/internal/domain"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mysqlNamedDialector struct {
	gorm.Dialector
}

func (mysqlNamedDialector) Name() string { return "mysql" }

func TestSettingsRepositoryCompareAndSwapValue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT,
			value_type TEXT NOT NULL,
			deleted_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create settings table: %v", err)
	}
	repo := &SettingsRepository{BaseRepository: NewBaseRepository(&Database{DB: db})}
	original := `{"access_token":"old","refresh_token":"spent"}`
	row := &domain.Settings{Key: "zalo.credentials", Value: &original, ValueType: domain.ValueTypeJSON}
	if err := repo.Create(context.Background(), row); err != nil {
		t.Fatalf("create setting: %v", err)
	}

	rotated := `{"access_token":"new","refresh_token":"successor"}`
	updated, err := repo.CompareAndSwapValue(context.Background(), row.Key, original, rotated, domain.ValueTypeJSON)
	if err != nil || !updated {
		t.Fatalf("first compare-and-swap = %v, %v; want true, nil", updated, err)
	}
	staleWrite := `{"access_token":"old","refresh_token":"spent","last_error":"late"}`
	updated, err = repo.CompareAndSwapValue(context.Background(), row.Key, original, staleWrite, domain.ValueTypeJSON)
	if err != nil {
		t.Fatalf("stale compare-and-swap: %v", err)
	}
	if updated {
		t.Fatal("stale compare-and-swap unexpectedly overwrote rotated credentials")
	}
	stored, err := repo.GetByKey(context.Background(), row.Key)
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if stored.Value == nil || *stored.Value != rotated {
		t.Fatalf("stored value = %v, want rotated credentials", stored.Value)
	}
}

func TestSettingsRepositoryCompareAndSwapUsesBinaryComparisonOnMySQL(t *testing.T) {
	var logs bytes.Buffer
	db, err := gorm.Open(
		mysqlNamedDialector{Dialector: sqlite.Open(":memory:")},
		&gorm.Config{DryRun: true, Logger: newGORMLogger(&logs)},
	)
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}
	repo := &SettingsRepository{BaseRepository: NewBaseRepository(&Database{DB: db})}
	_, err = repo.CompareAndSwapValue(
		context.Background(),
		"zalo.credentials",
		`{"refresh_token":"CaseSensitive"}`,
		`{"refresh_token":"successor"}`,
		domain.ValueTypeJSON,
	)
	if err != nil {
		t.Fatalf("compare-and-swap dry run: %v", err)
	}
	if sql := logs.String(); !strings.Contains(sql, "BINARY `value` = BINARY ?") {
		t.Fatalf("MySQL CAS is not byte-exact: %s", sql)
	}
}
