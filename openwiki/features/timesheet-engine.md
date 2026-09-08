---
type: feature
title: Timesheet Engine
description: Timesheet creation, validation, bulk import (BCC pipeline), and approval lifecycle that feeds salary payout.
tags: [timesheet, bcc-import, validation, approval, payroll]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-69d2f2d90374b15c87ad9570
    resource: repo://backend/internal/app/services/bcc_import_lock.go
  - id: openwiki-source-456ad2e84b6683a3e358fd51
    resource: repo://backend/internal/app/services/bcc_import_pipeline.go
  - id: openwiki-source-7ba637f1f7a4bb9e4ea6b494
    resource: repo://backend/internal/domain/services/timesheet_domain_service_approval.go
  - id: openwiki-source-acdcd3f4304db71e2b2d8ba2
    resource: repo://backend/internal/domain/services/timesheet_validation_service.go
  - id: openwiki-source-214e8850d4cbceb475484f37
    resource: repo://backend/tests/integration/flow_bcc_import.go
  - id: openwiki-source-9cec9d8e66c2c1603b4b7c0b
    resource: repo://backend/tests/integration/flow_bcc_weekly_import.go
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Timesheet Engine

The timesheet engine is the upstream source of every salary payout. Each timesheet records one employee on one project on one day (or a contiguous range) with hours, day type, pay rate, and computed gross amount. Validation enforces pay-period, daily-hours, and assignment rules; the BCC import pipeline feeds weekly batches uploaded from MBank exports; the approval lifecycle gates rows from "draft" through "approved" so they can flow into FlexPay draws and bulk-transfer batches.

The engine has two distinct concerns that should not be conflated:

- **Domain validation** — pure business rules in `backend/internal/domain/services/timesheet_validation_*.go`. Stateless, no I/O side effects, cache-friendly.
- **Import orchestration** — state machine and pipeline plumbing in `backend/internal/app/services/bcc_import_*.go`. Owns transactions, asset references, audit emission, and rate normalization.

## Timesheet domain shape

`Timesheet` (`backend/internal/domain/timesheet.go`, ~13.7K) carries the canonical fields: employee, project, date range, day type, hours breakdown (regular / overtime / night / holiday), assigned pay rate, computed amount, and a status from the timesheet state machine. `timesheet_types.go` (~16.2K) defines the day-type enum (workday, weekend, public holiday, leave, sick) and the overtime-rule helpers.

`ProjectEmployee` (`project_employee.go`, ~16.3K) is the assignment that makes a timesheet valid: an employee must be assigned to the project on the date the timesheet covers, with a pay rate that covers that date.

## Validation services

`TimesheetValidationService` (`backend/internal/domain/services/timesheet_validation_service.go`) composes a chain of single-responsibility validators:

- `timesheet_validation_status.go` — only `draft` timesheets may be edited; only `submitted` may be approved.
- `timesheet_validation_assignment.go` — employee must be assigned to the project covering the timesheet dates.
- `timesheet_validation_daytype.go` — day-type × hours consistency (e.g., a public-holiday row cannot have positive regular hours).
- `timesheet_validation_daily_hours.go` — per-day hours cap, overtime threshold, and night-hour rules.
- `timesheet_validation_zero_rate.go` — pays must have a non-zero rate for the day.
- `timesheet_validation_assignment.go` — re-checks the assignment row at validation time.

Each validator returns a typed domain error with a Vietnamese message. A `validationContext` caches lookups (payrate, assignment, project) within a single validation request to avoid duplicate queries. The `TimesheetDomainService` in `timesheet_domain_service_*.go` orchestrates bulk operations and approval transitions.

## BCC import pipeline

BCC = Bảng Chấm Công (timesheet spreadsheet). Weekly uploads arrive as Excel/CSV from the project accountants. The pipeline has been deliberately decomposed into many small files instead of one monolith:

| File | Responsibility |
|---|---|
| `bcc_import_pipeline.go` | Shared helpers (fail/empty/finalize tails, rate-map builders) |
| `bcc_import_lifecycle.go` | Asset persistence + audit emission + result finalize |
| `bcc_import_process.go` | Row-by-row processor with per-row domain validation |
| `bcc_import_multi_position.go` | Multi-position rows (one employee, multiple roles in a day) |
| `bcc_import_replacement.go` | Replacing prior uploads cleanly without orphaning timesheets |
| `bcc_import_lock.go` | Concurrency guard — only one upload per `(project, for_month)` at a time |
| `bcc_import_stk.go` | Vietnamese STK-specific (bank-account) row handling |
| `bcc_import_weekly_payment.go` | Weekly-payment-rate rows |
| `bcc_import_weekly_bcc.go` | Weekly-bcc rows (the most common shape) |
| `bcc_import_weekly_rates.go` | Rate normalization across week boundaries |
| `bcc_import_weekly_shared.go` | Shared types across weekly processors |
| `bcc_norm_name*.go` | Name normalization (Vietnamese diacritics, alias tables) |
| `bcc_import_result.go` | Result shape (counts, errors, references) |

The decomposition is intentional: the per-format processors share their fail/empty/finalize tails via helpers in `bcc_import_pipeline.go`, but format-specific logic stays in the per-format file. Semantics are frozen — every helper reproduces the processors' original behavior exactly.

## Import lifecycle

1. **Asset upload** — file lands in object storage as an `Asset` row with a content hash.
2. **Lock acquisition** — `bcc_import_lock.go` ensures only one concurrent import per `(project, for_month)`. Subsequent uploads fail fast with a Vietnamese message rather than corrupting prior data.
3. **Format detection** — file extension + first-row sniff pick the per-format processor.
4. **Row iteration** — `bcc_import_process.go` walks rows, normalizing names (`bcc_norm_name.go`) and rates (`bcc_import_weekly_rates.go`). Each row is validated by the domain service chain before being persisted.
5. **Persistence** — timesheet rows go through `timesheet_domain_service_bulk_create.go` inside a single `WithTransaction`; failures roll the whole batch back. Successful batches persist with `import_job_id` linking back to the asset.
6. **Audit emission** — `bcc_import_lifecycle.go` calls `auditService.LogFileImport` non-blocking after the transaction commits.
7. **Result** — a `BCCImportResult` with counts (created / updated / skipped / failed), the original asset reference, and a paginated error list.

## Rate normalization

Weekly pay is not a single number — it is the sum of regular hours × rate plus overtime × OT-multiplier × rate plus any allowances. `bcc_import_weekly_rates.go` resolves the effective rate per row by walking the `payrates` table to find the assignment's active rate at the row's date. If the rate changed mid-week, the row is split at the change boundary.

Multi-position rows (`bcc_import_multi_position.go`) carry a `position_id` per row segment; each segment computes independently against its own rate.

## Approval lifecycle

A timesheet transitions `draft → submitted → approved → (paid)`. The transition lives in `timesheet_domain_service_approval.go`. Once `approved`, the row becomes a candidate for:

- FlexPay draws (see [FlexPay Advance Payments](./flexpay-advance-payments.md)) — capped against the per-month quota.
- Bulk transfer batches (see [Salary Disbursement and Payment Providers](./salary-disbursement-and-payment-providers.md)) — the source lines for the next payout.

Editing an `approved` timesheet is rejected by the validator with a domain error; corrections must go through a reversal/offset flow that creates a paired negative entry rather than mutating the original.

## Payroll calculation

`PayrollCalculationService` (`backend/internal/domain/services/payroll_calculation_service.go`) consumes approved timesheets and produces the per-employee, per-period payout lines: gross, regular, OT, allowances, deductions. The output is what flows into the bulk-transfer batch. Rate resolution and OT multipliers live in `timesheet_types.go`; per-project rules live in `Project` configuration.

## Integration test evidence

Live-API integration tests live in `backend/tests/integration/`:

- `flow_bcc_import.go`, `flow_bcc_weekly_import.go`, `flow_bcc_weekly_payment_import.go` — exercise the per-format processors end-to-end through HTTP, asserting row counts, audit emission, and idempotency on re-upload.
- `flow_timesheet_extended.go` — covers validation chains and assignment edge cases.
- `flow_partner_timesheet.go` — partner-role scoping of timesheet reads.

These tests use the project's clock helpers (`SetServerTime`, `AdvanceServerTime`, `ResetServerTime`) to exercise time-dependent validation without wall-clock waits. Test data is cleaned up via direct DB queries before each run.

## Related pages

- [Architecture Overview](../architecture/overview.md) — where validation services sit in the DDD layers.
- [FlexPay Advance Payments](./flexpay-advance-payments.md) — consumes approved timesheets for the advance cap.
- [Salary Disbursement and Payment Providers](./salary-disbursement-and-payment-providers.md) — consumes approved timesheets for the bulk-transfer batch.
