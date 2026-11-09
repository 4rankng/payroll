package persistence

import (
	"context"
	"log/slog"
	"testing"
	"testing/fstest"

	gormlogger "gorm.io/gorm/logger"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMigrationTestDB(t *testing.T) *gorm.DB {
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
	return db
}

// Fixtures sit above sqlMigrationsBaseline (145) so the runner executes them;
// one version below the baseline pins the record-but-skip behavior.
func TestRunPendingSQLMigrations(t *testing.T) {
	db := newMigrationTestDB(t)
	files := fstest.MapFS{
		"900_create_gadget.up.sql": &fstest.MapFile{
			Data: []byte("-- a gadget table\nCREATE TABLE gadgets (\n  id INTEGER PRIMARY KEY,\n  name TEXT\n);\n\nCREATE INDEX idx_gadgets_name ON gadgets (name);\n"),
		},
		"901_seed_gadget.up.sql": &fstest.MapFile{
			Data: []byte("INSERT INTO gadgets (id, name) VALUES (1, 'half; truth');\n"),
		},
		"899_manual_history.up.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE should_not_exist (id INTEGER);\n"),
		},
	}

	// 899_manual_history.up.sql sits at the override baseline: recorded, never
	// executed (version <= baseline is the skip rule). 900/901 run for real.
	sqlMigrationsBaseline = 899
	t.Cleanup(func() { sqlMigrationsBaseline = 145 })

	err := RunPendingSQLMigrations(context.Background(), db, files, slog.Default())
	if err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// 899 sits below the (raised) baseline: recorded, never executed.
	var count int
	if err := db.Raw(`SELECT COUNT(*) FROM should_not_exist`).Scan(&count).Error; err == nil {
		t.Fatal("baseline migration was executed, want skipped")
	}

	// Statement split kept the quoted semicolon intact.
	var seeded name
	if err := db.Raw(`SELECT name FROM gadgets WHERE id = 1`).Scan(&seeded).Error; err != nil {
		t.Fatalf("seeded gadget missing: %v", err)
	}
	if seeded.Name != "half; truth" {
		t.Fatalf("seeded name %q, want %q", seeded.Name, "half; truth")
	}

	var applied []int64
	if err := db.Raw(`SELECT version FROM schema_migrations ORDER BY version`).Scan(&applied).Error; err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(applied) != 3 || applied[0] != 899 || applied[1] != 900 || applied[2] != 901 {
		t.Fatalf("ledger wrong: %+v", applied)
	}

	// Idempotent: a second run applies nothing and still succeeds.
	if err := RunPendingSQLMigrations(context.Background(), db, files, slog.Default()); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if err := db.Raw(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count).Error; err != nil || count != 3 {
		t.Fatalf("ledger grew on re-run: count=%d err=%v", count, err)
	}
}

type name struct {
	Name string
}
