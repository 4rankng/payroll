# Runbook: payrates migration 138 — drop `to_date` (from-date-only, phase 2)

Removes the retired `payrates.to_date` column and the `chk_payrates_date_order`
check constraint. Migration files:
`backend/migrations/138_payrate_drop_to_date.{up,down}.sql` — both idempotent
(every statement guarded via INFORMATION_SCHEMA).

## Apply gate — READ FIRST

**Do not run on production until the from-date-only build has served one full
pay cycle.** From-date-only deployed 2026-10-24 (v1.18.x); Kỳ-4 salary lands
~2026-11-01, so the earliest sensible apply is after that payout is confirmed.

Two rollback paths depend on the column surviving until then:

1. Rolling the backend container back to a pre-from-date-only image — old code
   reads `to_date` in every rate-resolution query; with the column gone those
   queries hard-error (payroll down, not degraded).
2. Migration 137's down script re-creates the `open_project_id` generated
   column from `to_date` and cannot run without it.

The application since 137 never reads or writes the column; API responses
derive end dates from each config's successor
(`payrate_service.AttachDisplayEndDates`). The last code-side reference (the
dead `PayrateFilters.ToDate` → `to_date <=` filter) was removed in the commit
carrying this runbook.

## Pre-checks (production)

```sql
-- expect the column and the check constraint to exist
SHOW COLUMNS FROM payrates LIKE 'to_date';
SELECT constraint_name FROM information_schema.table_constraints
WHERE table_schema = DATABASE() AND table_name = 'payrates'
  AND constraint_name = 'chk_payrates_date_order';
-- confirm no code still reads it (repo search for "to_date" under
-- internal/, payrates context) — must come back empty first
```

Backup is mandatory before any schema change: `make backup` (OneDrive dump),
plus a table-level dump for a fast restore:

```bash
docker exec payroll-mysql mysqldump -uroot -prootpassword payroll_db payrates \
  > backups/payrates_pre138_<env>_<date>.sql
```

## Apply (production, after gate)

Run the guarded script via SSH into the prod MySQL container — same statements
as the local verification below:

```bash
ssh <prod> 'docker exec -i payroll-mysql mysql -uroot -p*** payroll_db' \
  < backend/migrations/138_payrate_drop_to_date.up.sql
```

## Verify

```sql
SHOW COLUMNS FROM payrates LIKE 'to_date';        -- must be empty
SHOW INDEX FROM payrates;                          -- PRIMARY + fk_payrates_project
                                                   -- + fk_payrates_created_user
                                                   -- + idx_payrates_deleted_at
                                                   -- + uq_payrates_live_project_from_date
SELECT COUNT(*) FROM payrates;                     -- unchanged vs pre-apply
```

## Rollback

`138_payrate_drop_to_date.down.sql` restores the nullable column and the check
constraint (guarded, idempotent). Rows dropped with the column keep no
`to_date` history — the down script restores structure only; display end dates
continue to be derived from successors.

## Local verification record — 2026-10-25

- Backup: `backups/payrates_pre138_local_20261025.sql` (86 rows).
- Applied `138_payrate_drop_to_date.up.sql` to the local dev DB
  (`payroll-mysql`/`payroll_db`): column and check constraint gone, indexes
  exactly the five expected, row count unchanged.
- Re-ran the same script a second time: silent no-op (idempotency confirmed).
- Full integration suite against a backend built from the same commit:
  434 total / 0 FAIL — the from-date-only code path is healthy without the
  column. Production remains UNTOUCHED pending the Kỳ-4 payout gate.
