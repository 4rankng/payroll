---
type: feature
title: BCC Weekly Import
description: Bank Confirmation Certificate upload — the multi-format pipeline that parses weekly Excel files, deduplicates, applies payrate label mapping, and dispatches rows into the timesheet system.
tags: [feature, bcc, import, timesheet, weekly, payroll, excel]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-69d2f2d90374b15c87ad9570
    resource: repo://backend/internal/app/services/bcc_import_lock.go
  - id: openwiki-source-456ad2e84b6683a3e358fd51
    resource: repo://backend/internal/app/services/bcc_import_pipeline.go
  - id: openwiki-source-0a855c82f0a5d4b5c2d197b8
    resource: repo://backend/internal/app/workers/bcc_import_worker.go
  - id: openwiki-source-e5e315ee6d1053411e0b5b4e
    resource: repo://backend/migrations/098_add_flexible_bcc_import_option.up.sql
  - id: openwiki-source-98b4fef5bee3b5a0d880f16b
    resource: repo://docs/api.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: BCC Weekly Import

BCC (Bank Confirmation Certificate / "bảng xác nhận ngân hàng") files are weekly Excel uploads that capture which employees worked which positions on which days at which rates. They are the source of truth for payroll generation: the import creates or updates timesheet rows, which downstream services then approve, batch, and disburse. The pipeline runs as a single asynq task (`import:job`) driven by `bcc_import_worker.go`.

The reference file formats live under `docs/WeeklyBCC/` (see the BCC upload reference docs).

## Pipeline

```
upload ──► parse ──► dedup ──► label-rate ──► dispatch ──► finalize
                                       │
                                       └─► fail (per-format processor)
```

Shared pipeline primitives live in `internal/app/services/bcc_import_pipeline.go` (`failBCCImport`, `bccFailer`, `buildRateToTarget`, etc.). Format-specific logic lives in the per-format processors:

- `bcc_import_service.go` — legacy date-row format (the largest file).
- `bcc_import_weekly_bcc.go` — weekly BCC format.
- `bcc_import_weekly_payment.go` — weekly payment format.
- `bcc_import_multi_position.go` — multi-position format (one row per position-day).
- `bcc_import_weekly_rates.go` — weekly rates variant.

The shared helpers exist because the per-format processors used to copy-paste the same fail/empty/finalize tails and rate-map builders; the helpers now reproduce the original behavior exactly while keeping format-specific logic in the per-format files.

### Backdate

`bcc_import_backdate*.go` handles the case where the upload proves an employee worked earlier days than their current project assignment covers. The import backdates the existing assignment's `start_date` to the file's earliest entry, so subsequent assignment validation does not reject same-month uploads. (This rule is also referenced in `docs/api.md` for the `/api/v1/project-employees` endpoint.)

### Label-rate mapping

`buildRateToTarget` (in `bcc_import_pipeline.go`) maps a VND rate to a `(dayType, hourType)` pair from flattened payrate paths. The rule: **rate is king** — as long as the rate matches, the entry is valid. On collision (same rate, multiple paths), the lowest day-type priority wins (prefer "ngày thường"). `posPrefix` scopes the map to one position (`"position."`) for position-keyed formats; empty builds from every path (legacy behavior).

### Replacement

`bcc_import_replacement.go` handles rows that need to overwrite an existing timesheet (rate or status change). Replacement is opt-in per format and respects the bulk-operations lock.

### Lock and dispatch

`bcc_import_lock.go` serializes BCC imports against concurrent bulk operations on the same project so a half-applied import does not collide with an in-flight bulk approve. `bcc_import_dispatch*.go` writes the parsed rows into the timesheet table inside a transaction; on commit, after-commit callbacks fire the cache invalidation and event emission per ADR-007.

### Result & errors

`bcc_import_result.go` and `bcc_import_types.go` define the result envelope returned to the admin UI: counts of created / updated / skipped / failed rows, the original filename, the for-month tag, the project ID, and per-row import errors with Vietnamese messages. The result is also stored on the import job row so the UI can render a persistent summary.

### Lifecycle and helpers

`bcc_import_lifecycle.go` owns the state transitions for `timesheet_import_jobs` (queued → processing → succeeded / failed). `bcc_import_helpers.go` carries cross-processor helpers (STK parsing for bank account imports, payrate normalization, name normalization). `bcc_import_stk.go` parses the STK (số tài khoản) bank account file used when seeding employee payment details.

## Flexible-employee option (migration 098)

`timesheet_import_jobs.include_flexible_employees` (BOOLEAN, default FALSE) lets the admin include flexi-employees in the import. Without this flag, only regular employees are imported. The flag is set at upload and stored on the job for audit.

## Vietnamese name normalization

`bcc_norm_name_test.go` (with the matching helper) covers the Vietnamese-specific name normalization rules used when matching BCC rows to existing employees (diacritics, middle-name handling, common abbreviations). Property-style test patterns cover a range of inputs.

## Tests

The BCC import has dedicated unit tests alongside each module:

- `bcc_import_service_test.go` — full service coverage.
- `bcc_import_pipeline_test.go` — pipeline helpers and rate-map logic.
- `bcc_import_backdate_test.go` — assignment backdate.
- `bcc_import_dedup_test.go` — duplicate detection.
- `bcc_import_dispatch_test.go` — dispatch into timesheets.
- `bcc_import_helpers_test.go` — STK and payrate helpers.
- `bcc_import_label_rate_test.go` — rate → (dayType, hourType) mapping.
- `bcc_import_weekly_test.go`, `bcc_import_weekly_payrate_test.go` — weekly variants.
- `bcc_norm_name_test.go` — Vietnamese name normalization.

Integration scenarios in `backend/tests/integration/` exercise the full upload → timesheet row pipeline.

## Relationships

- **Timesheets** — successful dispatch creates or updates `timesheets` rows. See `features/timesheet.md`.
- **Salary disbursement** — imported timesheets are the input to the weekly payroll run; `bulk_transfer_batch` and `bulk_transfer_file` are populated from the imported rows. See `features/salary-disbursement.md`.
- **Admin/Partner UI** — the upload UI lives in `frontend/src/pages/admin` (admin role) and `frontend/src/pages/partner` (partner role); the result summary is rendered from `BCCImportResult`.
