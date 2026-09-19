---
type: "Reference"
title: "Migrations: Schema Evolution"
openwiki_generated: true
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-582d96b5d108bf6aae1e2c21
    resource: repo://backend/internal/domain/bulk_transfer_batch.go
  - id: openwiki-source-43c18e617136f11e46e05ed9
    resource: repo://backend/internal/domain/flexpay_salary_notification.go
  - id: openwiki-source-4887d9be4988ded784668fd2
    resource: repo://backend/internal/domain/wallet/wallet_payment.go
  - id: openwiki-source-e5e315ee6d1053411e0b5b4e
    resource: repo://backend/migrations/098_add_flexible_bcc_import_option.up.sql
  - id: openwiki-source-9bf61dc2d18ff0e4c789f4e6
    resource: repo://backend/migrations/100_add_flexpay_salary_notifications.up.sql
  - id: openwiki-source-da0867ee883cf3b9195fa043
    resource: repo://backend/migrations/101_seed_self_check_in_advance_hold_hours.up.sql
  - id: openwiki-source-4c5117867fd7bf51d2bf86a4
    resource: repo://backend/migrations/102_add_attendance_quota_credit_eligible_at.up.sql
  - id: openwiki-source-7814a631d4046f6bc84c871c
    resource: repo://backend/migrations/104_project_employees_pending_check_in.up.sql
  - id: openwiki-source-22c979da92ff10b8f954b8e0
    resource: repo://backend/migrations/105_project_employees_advance_request_enabled.up.sql
  - id: openwiki-source-156ad5a7f27e5758f4a19efd
    resource: repo://backend/migrations/AGENTS.md
  - id: openwiki-source-675cdb8aa785fd3b58b5747f
    resource: repo://docs/decisions/ADR-004-redis-streams-event-bus.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---


# Migrations: Schema Evolution

`backend/migrations/` carries 130+ SQL files that built the current schema. Most migrations are incremental and uninteresting in isolation; this page records the migrations that changed behavior or contract — new aggregates, breaking constraint changes, the outbox removal, and the kill switches. The full file list lives in `docs/database.md`; this page is the curated index.

## Wallet foundation (054, 055, 057)

- **054** — `wallet_payments`, `wallet_topups` tables. Double-entry accounting surface; one row per disbursement.
- **055** — `wallet_ipn` table. Durable IPN delivery record; idempotency key by `(provider, invoice_no)`.
- **057** — `wallet_provider` table (provider abstraction). Renames/normalizes the provider column ahead of the OnePay/9Pay split.

These three migrations turned the wallet into a self-contained aggregate. See `features/wallet-ledger.md`.

## Attendance quota + kill switches (101, 102, 104, 105)

- **101** — Seed `settings.self_check_in_advance_hold_hours = 24`. Idempotent INSERT preserves any admin-configured value.
- **102** — Add `attendances.quota_credit_eligible_at` (immutable per-row) + `idx_attendances_quota_credit_eligible`. The credit quota worker and the recovery sweep read only this column so admin changes to the hold setting cannot accelerate already-waiting earnings.
- **104** — Add `project_employees.pending_check_in_enabled` + `check_in_effective_from`. Enable defers to day 1 of next month; disable is instant; DELETE cancels a pending enable.
- **105** — Add `project_employees.advance_request_enabled` (default 1). The per-employee kill switch ("tạm ngừng ứng lương") for new advance requests; existing pending/approved rows are untouched.

See `features/attendance-geofence.md`.

## FlexPay notifications (100)

- **100** — `flexpay_salary_notifications` table. Durable delivery record with `UNIQUE (asset_id, project_id, employee_id, template_id)` for safe import retry and `INDEX (status, lease_expires_at)` for the recovery sweeper. Status values: pending, processing, sent, failed, suppressed (last three terminal).

See `features/flexpay.md`.

## BCC flexible import (098)

- **098** — Add `timesheet_import_jobs.include_flexible_employees` (BOOLEAN, default FALSE). Lets admins opt flexi-employees into a BCC import.

See `features/bcc-import.md`.

## OnePay bank code correction (099)

- **099** — Correct the OnePay bank code map. Provider-side bank codes were re-aligned with the canonical Vietnamese bank list; rows using stale codes are remapped.

## Loan schedule split (103)

- **103** — Split the loan repayment schedule into `principal` and `interest` columns. Replaces the prior combined-amount column so reports can show the principal/interest split per installment.

## Outbox removal (062)

- **062** — Drop the database-backed outbox tables. Events now flow through Redis Streams (ADR-004). The drop is irreversible — never reintroduce a database outbox.

## Ad banners (106)

- **106** — Add `ad_banners` table and the supporting event factory. Used for in-app promotional cards.

## Weekly payment fees (107)

- **107** — Seed `weekly_payment_fee_schedules` with the default weekly fee tiers. Idempotent; admin can override.

## Schema reconciliation (108)

- **108** — `reconcile_runtime_update.up.sql` repairs runtime drift between MySQL and the expected schema after deploys. Idempotent by design.

## Ledger unique index drop (109)

- **109** — Drop a legacy `ledger_asset_unique` index that prevented legitimate re-bookings. The drop is required so the bulk-transfer pipeline can correct itself via `ProcessBookBatchLedger`.

## Conventions

- Migrations are raw SQL, not a framework-managed migration tool; applied via `go run cmd/migrate/main.go up`.
- Always prefer code-level fixes over schema migrations.
- Recovery/repair migrations must be idempotent.
- Down migrations exist for some but not all — assume no downgrade without testing.
- `backend/migrations/AGENTS.md` carries the full discipline.

## Relationships

<!-- openwiki: broken internal link [../docs/database.md] file "../docs/database.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- Schema overview — [`docs/database.md`](../docs/database.md).
- Wallet & ledger — `features/wallet-ledger.md`.
- Attendance — `features/attendance-geofence.md`.
- FlexPay — `features/flexpay.md`.
- BCC import — `features/bcc-import.md`.
- Bulk-transfer pipeline — `features/salary-disbursement.md`.
