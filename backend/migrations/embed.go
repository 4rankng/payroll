// Package migrations carries the numbered SQL migration files applied by
// scripts/init-local-db.sh (fresh environments) and, since the boot-time
// runner exists, by persistence.RunPendingSQLMigrations on every server boot.
// Embedding them ships the files inside the api-server binary — deploys no
// longer depend on anyone remembering to apply a migration by hand.
package migrations

import (
	"embed"
)

// UpSQLFiles holds every NN_*.up.sql migration file.
//
//go:embed *_*.up.sql
var UpSQLFiles embed.FS
