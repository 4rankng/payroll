<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# migrations — Database Schema Migrations

## Purpose
Contains 83 SQL migration files that define and evolve the database schema. Migrations are numbered sequentially (001–073) and include both up migrations and occasional down migrations. The schema supports payroll processing, double-entry ledger accounting, wallet operations, payment disbursement, timesheet management, and audit logging.

## Key Files
| File | Description |
|------|-------------|
| `001_init_db.up.sql` | Initial schema — all core tables (employees, projects, timesheets, transactions, etc.). 35.5K. |
| `015_create_outbox_events.up.sql` | Outbox pattern table for reliable event publishing |
| `057_wallet_provider.up.sql` | Wallet provider abstraction (OnePay/9Pay) |
| `061_recover_orphaned_settlements.sql` | Idempotent data repair for orphaned settlements |
| `062_drop_outbox_tables.up.sql` | Removed outbox tables (migrated to Redis streams) |
| `067_add_flexi_checkin_fields.up.sql` | Flexible project check-in/out fields |
| `068_add_geofence_gates.up.sql` | Geofence gate definitions for attendance |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- Migrations are raw SQL files, not managed by a migration tool framework
- **Always prefer code-level fixes over schema migrations** when possible (project convention)
- Migration files are applied via `go run cmd/migrate/main.go up`
- When creating new migrations, use the next sequential number
- Some migrations have corresponding `.down.sql` files for rollback
- **Recovery/repair scripts must be idempotent** — re-runnable without side effects
- Migrations 062+ dropped the outbox tables; event publishing now uses Redis streams

### Testing Requirements
- After schema changes, run `make api-test` to verify no regressions
- Test data pollution may require direct DB cleanup: `docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "DELETE FROM timesheets WHERE ..."`

### Common Patterns
```sql
-- Typical migration pattern
ALTER TABLE table_name ADD COLUMN new_column VARCHAR(255);

-- Idempotent recovery script pattern (see 061)
INSERT IGNORE INTO settlements (...) SELECT ... FROM ... WHERE ...;
```

## Dependencies

### Internal
- `internal/infra/persistence/database.go` — GORM database initialization
- Schema changes may require updates to domain entities and repository queries

### External
- MySQL 8.x — target database

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
