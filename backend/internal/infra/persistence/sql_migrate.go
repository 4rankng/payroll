package persistence

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// sqlMigrationsBaseline is the highest migration version shipped before the
// boot-time runner existed (145 = dashboard_bank_usage_summaries, applied by
// hand on every live environment because deploys never ran migrations). The
// runner records versions <= baseline as applied WITHOUT executing them; the
// two historical files that use DELIMITER client directives (103, 108) sit
// below the baseline, so the runner never needs a DELIMITER-aware parser.
// Fresh environments keep provisioning through scripts/init-local-db.sh,
// whose mysql client applies the full history natively — recording <= baseline
// stays truthful there too.
//
// Bump this baseline only if a hot migration had to be applied manually on
// every environment. Test-overridable: fixtures must sit above it to be
// executed by the runner.
var sqlMigrationsBaseline int64 = 145

var sqlMigrationVersionRe = regexp.MustCompile(`^0*(\d+)_`)

type sqlMigration struct {
	version int64
	name    string
}

// RunPendingSQLMigrations creates the schema_migrations ledger, records every
// version <= baseline as applied, then executes pending *.up.sql files in
// version order — each file inside one transaction together with its ledger
// row, so a half-applied migration can never be recorded.
//
// Statement splitting is deliberately simple: a statement ends at the first
// `;` that terminates a line. Migration files must not rely on DELIMITER or
// stored routines; comment lines (`--`) pass through to the server untouched.
// Opt out per environment with DB_AUTO_MIGRATE=false.
func RunPendingSQLMigrations(ctx context.Context, db *gorm.DB, files fs.FS, logger *slog.Logger) error {
	if err := db.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT UNSIGNED PRIMARY KEY,
			applied_at DATETIME(3) NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.Glob(files, "*_*.up.sql")
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	migrations := make([]sqlMigration, 0, len(entries))
	for _, entry := range entries {
		match := sqlMigrationVersionRe.FindStringSubmatch(entry)
		if match == nil {
			continue
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse version of %s: %w", entry, err)
		}
		migrations = append(migrations, sqlMigration{version: version, name: entry})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })

	applied := map[int64]bool{}
	var recorded []int64
	if err := db.WithContext(ctx).Raw(`SELECT version FROM schema_migrations`).Scan(&recorded).Error; err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	for _, version := range recorded {
		applied[version] = true
	}

	now := time.Now()
	for _, migration := range migrations {
		if applied[migration.version] {
			continue
		}
		if migration.version <= sqlMigrationsBaseline {
			// Shipped before the runner: assumed applied by the manual flows.
			if err := db.WithContext(ctx).Exec(
				`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
				migration.version, now,
			).Error; err != nil {
				return fmt.Errorf("record baseline migration %d: %w", migration.version, err)
			}
			continue
		}

		content, err := fs.ReadFile(files, migration.name)
		if err != nil {
			return fmt.Errorf("read %s: %w", migration.name, err)
		}
		statements := splitSQLStatements(string(content))
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, statement := range statements {
				if err := tx.Exec(statement).Error; err != nil {
					return fmt.Errorf("%s: %w", migration.name, err)
				}
			}
			return tx.Exec(
				`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
				migration.version, time.Now(),
			).Error
		})
		if err != nil {
			return err
		}
		logger.Info("Applied SQL migration", "version", migration.version, "file", migration.name)
	}

	logger.Info("SQL migrations up to date", "baseline", sqlMigrationsBaseline, "tracked", len(applied))
	return nil
}

// splitSQLStatements breaks a migration file into statements on the first `;`
// that terminates a line, dropping empty results. Quoted `;` inside a string
// literal on the final line of a statement would split early — historical
// files don't do this and the convention is one statement per `;` line.
func splitSQLStatements(content string) []string {
	statements := make([]string, 0, 8)
	var current strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, " \t\r")
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			// The statement is the whole accumulated buffer up to (and
			// including) this terminating `;` — interior lines of a
			// multi-line statement end with `,` or `(`, not `;`.
			statement := strings.TrimSuffix(strings.TrimRight(current.String(), " \t\r\n"), ";")
			statement = strings.TrimRight(statement, " \t\r\n")
			if strings.TrimSpace(statement) != "" {
				statements = append(statements, statement+";")
			}
			current.Reset()
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		statements = append(statements, tail)
	}
	return statements
}
