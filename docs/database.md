# Database

Schema overview, migration conventions, and GORM patterns for the payroll backend. MySQL 8 is the primary database, accessed exclusively through GORM. See [Code Standards](code-standards.md) for repository conventions and [ADR-002](decisions/ADR-002-go-gin-gorm-stack.md) for the stack decision.

## Migrations

**Location:** `backend/migrations/` — 103 SQL `.up.sql` files (numbered 001-103; some numbers have down migrations).

**Apply locally:**
```bash
cd backend && go run cmd/migrate/main.go up
```

**Apply to demo/prod:** See [Deployment Guide](deployment-guide.md) → Database Migrations.

### Migration Rules

1. **Always prefer code-level fixes over schema migrations** when possible.
2. Recovery/repair scripts must be **idempotent** (use `INSERT IGNORE`, `ON DUPLICATE KEY UPDATE`).
3. Migrations are raw SQL `.up.sql` files — no migration framework.
4. `schema_migrations` table may be empty on dump-seeded environments. Manual DDL application does not require a version row.
5. When a migration adds columns the deployed code expects, apply the migration **before** the backend deploy.

## Core Tables (Migration 001)

25 core tables created in `001_init_db.up.sql`:

| Table | Purpose |
|-------|---------|
| `users` | Authentication, user management (role: admin/partner/employee) |
| `blacklisted_tokens` | JWT token revocation |
| `banks` | Bank branch information |
| `lenders` | Loan lenders |
| `projects` | Project entities |
| `project_users` | Project-user access |
| `employees` | Employee records |
| `employee_users` | Employee-user account linking |
| `project_employees` | Employee-project assignments |
| `payrates` | Payrate configuration |
| `timesheets` | Check-in/out records |
| `timesheet_edit_requests` | Timesheet edit workflow |
| `assets` | Asset tracking |
| `loans` | Loan management |
| `loan_repayment_schedules` | Loan repayment schedules, including persisted `principal_amount` and `interest_amount` components (migration 103) |
| `transactions` | Financial transactions |
| `ledger_entries` | Double-entry accounting |
| `settlements` | Payment settlement tracking |
| `bulk_transfer_files` | Batch payment files |
| `bulk_transfer_histories` | Batch payment history |
| `notifications` | Notification delivery |
| `settings` | System configuration |
| `audit_logs` | Audit trail |
| `api_metrics` | API performance metrics |
| `migration_locks` / `migration_history` | Migration tracking |

## Tables Added in Later Migrations

| Migration | Table(s) | Purpose |
|-----------|----------|---------|
| 009 | `api_endpoints` | Normalized API endpoint definitions |
| 015 | `outbox_events` | Outbox pattern (later dropped in 062) |
| 018 | `outbox_event_handlers` | Outbox event handler registry (dropped in 062) |
| 019 | `accounts` | Chart of accounts for ledger |
| 028 | `advance_payments`, `advance_payment_requests`, `transaction_codes` | FlexPay advance payment system |
| 039 | `push_subscriptions` | Web Push subscription management |
| 040 | `cron_job_status` | Cron job execution tracking |
| 046 | `provider_transactions` | Payment provider transaction tracking |
| 054 | `wallet_payments`, `wallet_topups` | Wallet operations |
| 055 | `wallet_ipn` | Wallet instant payment notifications |
| 067 | `attendances` | Flexible project check-in/out (geofence) |
| 079 | `attendance_failed_attempts` | GPS attendance failure tracking |
| 085 | `settlement_uploads` | Settlement file upload tracking |

### Notable Migration Events

- **015 → 062:** Outbox tables created then dropped. Event publishing migrated to Redis streams. See [ADR-004](decisions/ADR-004-redis-streams-event-bus.md).
- **103:** Persists principal/interest components for scheduled loan repayments and their transactions, backfills historical schedules, rebuilds paid loan aggregates, and repairs the corresponding scheduled-payment ledger entries. It stops before DDL if an existing schedule totals less than its loan principal, because that split cannot be inferred safely. Apply it before deploying code that reads or writes these columns.
- **061:** Idempotent recovery script for orphaned settlements (`INSERT IGNORE`).
- **064:** Drop `advance_payments` salary column.
- **086:** Add `invalid_before` to user tokens for token revocation.
- **087:** Add OTP lockout fields.
- **088:** Performance indexes (`add_query_perf_indexes`).

## Domain Entities

90+ domain entity files in `backend/internal/domain/`. Key entities:

| Entity | File(s) | Size |
|--------|---------|------|
| Employee | `employee.go` | 15.6K |
| Project | `project.go` | 13.2K |
| ProjectEmployee | `project_employee.go` | 16.3K |
| Payrate | `payrate.go` | 18.5K |
| Timesheet | `timesheet.go` + `timesheet_types.go` | 13.7K + 16.2K |
| Transaction | `transaction.go` + `transaction_code.go` | 11.9K + 4.3K |
| Ledger | `ledger.go` + `accounting_rules.go` | 18.6K + 7.6K |
| AdvancePayment | `advance_payment.go` + `advance_payment_request.go` + `advance_payment_fee_schedule.go` | 7K + 11.7K + 4.9K |
| Wallet (aggregate) | `wallet/` subdirectory (7 files) | Self-contained |
| Loan | `loan.go` + `loan_strategy.go` + `loan_repayment_schedule.go` | 4.9K + 3.5K |
| Settlement | `settlement.go` + `settlement_entries.go` + `settlement_upload.go` | 3.4K |
| Security | `security.go` | 25.1K |
| Dashboard | `dashboard.go` | 25.5K |
| Notification | `notification.go` | 5.3K |
| User | `user.go` | 5.6K |
| Attendance | `attendance.go` | 3.8K |

### Wallet Aggregate

Self-contained in `domain/wallet/` (7 files): `forecast.go`, `repository.go`, `service.go`, `wallet_balance.go`, `wallet_ipn.go`, `wallet_payment.go`, `wallet_topup.go`. Uses `qmuntal/stateless` for state machine. See [ADR-009](decisions/ADR-009-wallet-double-entry-ledger.md).

## GORM Conventions

### Repository Pattern

Repositories implement interfaces from `domain/ports/`. Key patterns:

- Use `db.WithContext(ctx)` for all GORM operations.
- Handle `gorm.ErrRecordNotFound` → translate to `domain.NewNotFoundError()`.
- Map GORM models to domain types via `model.ToDomain()` method.
- Use `common/filter_builder.go` for dynamic WHERE clauses.
- Use `common/batch_processor.go` for batch insert/update.
- Complex queries use dedicated query builders in `query_builders/`.
- The timesheet repository is split into 3 files (analytics, command, query) due to size.
- Cache invalidation must happen **after** transaction commit, not before. See [ADR-007](decisions/ADR-007-transaction-manager-unit-of-work.md).

### Connection

Database connection setup in `internal/infra/persistence/database.go` (10K). Shared CRUD in `base_repository.go` (6.5K). Configured via `DB_DSN` environment variable.

### Transaction Manager

`GormTransactionManager` implements `domain.TransactionManager`:

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.repo.Save(txCtx, entity); err != nil { return err }
    if err := s.otherRepo.Update(txCtx, other); err != nil { return err }
    return nil // auto-commit
})
```

Automatic rollback on panic via deferred recovery. See [ADR-007](decisions/ADR-007-transaction-manager-unit-of-work.md).

## Domain Ports (Interfaces)

**Infrastructure ports** (`domain/ports/`): `audit.go`, `cache.go`, `disbursement.go`, `notification.go`
**Service ports** (`domain/ports/`): `ledger.go`, `loan.go`, `payroll.go`, `settlement.go`, `timesheet.go`, `transaction.go`

## Performance Indexes

Multiple rounds of query optimization through migrations:

| Migrations | Focus |
|-----------|-------|
| 010, 013, 014 | Dashboard and query performance indexes |
| 022, 023, 024, 025, 026 | API metrics optimization, slow query fixes, duplicate index cleanup, critical performance indexes, timesheet performance |
| 030 | Timesheet query optimization |
| 037 | Analytics query indexes |
| 088 | Query performance indexes (latest) |

See [Performance Standards](standards/performance.md) for ongoing optimization practices.
