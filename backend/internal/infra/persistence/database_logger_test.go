package persistence

import (
	"bytes"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGORMLoggerDoesNotRenderBoundSecrets(t *testing.T) {
	var output bytes.Buffer
	db, err := gorm.Open(
		sqlite.Open(":memory:"),
		&gorm.Config{Logger: newGORMLogger(&output)},
	)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec("CREATE TABLE secret_log_test (value TEXT)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	const secret = "zalo-refresh-token-must-never-reach-logs"
	if err := db.Exec("INSERT INTO secret_log_test (value) VALUES (?)", secret).Error; err != nil {
		t.Fatalf("insert secret: %v", err)
	}

	logs := output.String()
	if strings.Contains(logs, secret) {
		t.Fatalf("SQL logger exposed a bound secret: %s", logs)
	}
	if !strings.Contains(logs, "VALUES (?)") {
		t.Fatalf("SQL logger did not retain the parameterized query shape: %s", logs)
	}
}
