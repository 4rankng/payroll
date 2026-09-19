---
type: feature
title: Timesheets
description: Timesheet entry, validation (daily hours, daytype, status, zero rate, assignment), edit requests, bulk import, and approval workflow — the input to salary disbursement and the FlexPay quota pool.
tags: [feature, timesheet, validation, approval, edit-request, bulk, payroll]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-f34a1fd765b5250cc0861c61
    resource: repo://backend/internal/app/services/timesheet/bulk_operations.go
  - id: openwiki-source-513983ae15df98a4e52be197
    resource: repo://backend/internal/app/services/timesheet/transaction_orchestrator.go
  - id: openwiki-source-e88001c9ed050e45d26baa0f
    resource: repo://backend/internal/domain/services/timesheet_domain_service.go
  - id: openwiki-source-e81981420a03be5fdd79ea76
    resource: repo://backend/internal/domain/timesheet_edit_request.go
  - id: openwiki-source-2bcd91a7418f042b9a303479
    resource: repo://backend/internal/domain/timesheet.go
  - id: openwiki-source-5a8046a0ac76584e0788025d
    resource: repo://backend/migrations/004_add_timesheet_transaction_id.up.sql
  - id: openwiki-source-98b4fef5bee3b5a0d880f16b
    resource: repo://docs/api.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: Timesheets

Timesheets are the per-day, per-project, per-employee work record that drives payroll. They are created manually (admin/partner entry), imported via the weekly BCC upload, or seeded by attendance for flexi-employees. The full lifecycle covers approval, edit requests, payment-status tracking, and downstream disbursement.

## Lifecycle

```
draft → pending_approval → approved → (paid | failed | cancelled)
                                ↓
                          rejected (terminal)
```

The enum values live in `internal/domain/timesheet.go`:

```
TimesheetStatus:   pending_approval | approved | rejected
PaymentStatus:     pending | paid | failed | cancelled
```

A row is `pending_approval` after creation or import; an approver moves it to `approved` or `rejected`. After approval the row is eligible for salary disbursement (`payment_status: pending`). The bulk-transfer pipeline reads approved rows and books `wallet_payment` rows; the IPN handler drives `payment_status` to `paid` (or `failed` / `cancelled`).

A `force_payroll` flag overrides the typical approval gate for rows the admin wants to push through despite a missing step.

## Validation

`internal/domain/services/timesheet_validation_*.go` and `internal/domain/services/timesheet_domain_service*.go` own the rules:

- **daily hours** (`timesheet_validation_daily_hours.go`) — hours worked must be within the per-day bounds, with overtime handled separately.
- **daytype** (`timesheet_validation_daytype.go`) — `ngày thường` / weekend / holiday classifications that drive the payrate lookup.
- **status** (`timesheet_validation_status.go`) — only valid transitions are allowed (e.g., cannot approve a `rejected` row without an edit request).
- **zero rate** (`timesheet_validation_zero_rate.go`) — a zero-rate row requires explicit confirmation.
- **assignment** (`timesheet_validation_assignment.go`) — the employee must have a `project_employees` assignment that covers the timesheet date.

The bulk-create / update / preview path (`timesheet_domain_service_bulk_*.go`) runs these checks before any write. `reject_unpaid.go` and `reject_unpaid_test.go` cover the rule that an approved row cannot be silently rejected after payment.

## Entity shape (highlights)

`internal/domain/timesheet.go` `Timesheet` struct:

- `PayrateID`, `PayType`, `PayRate` — the rate card in effect at the time (snapshotted, not derived; backdated assignments don't retroactively change rates).
- `AllowedEdit` — whether the timesheet can be edited after approval.
- `RequestEditID` — pending edit-request id (mutually exclusive with the row being edited).
- `TransactionID` — linked revenue transaction (migration 004 added the column with an index, `idx_timesheets_transaction_id`).
- `RevenueReceivable` / `RevenuePaid` / `revenue_paid` — client-billing side of the equation.
- `ApprovedBy`, `ApprovedAt`, `RejectionReason`, `CreatedBy` — audit trail.
- Soft delete via `DeletedAt gorm.DeletedAt`.

## Edit requests

`internal/domain/timesheet_edit_request.go` plus `internal/app/services/timesheet/edit_request_service.go` cover the post-approval edit flow. The flow:

1. User (admin/partner) submits an edit request describing the change and reason.
2. The service validates the request against the current row.
3. On approval the row is updated; the `RequestEditID` is cleared.
4. Cache invalidation runs as an after-commit callback (ADR-007).

The lifecycle ties `AllowedEdit` (whether the row is editable at all), `RequestEditID` (the pending request), and the validation service that enforces what changes are permissible.

## Bulk operations

`internal/app/services/timesheet/bulk_operations.go` plus `timesheet_domain_service_bulk_*.go` cover bulk create, update, preview, and reset. The operations:

- Acquire a per-project lock so a BCC import cannot collide with an in-flight bulk approve.
- Run validation per row inside a transaction.
- Emit per-row errors but commit successful rows (partial success).
- Return a typed result envelope with per-row import errors in Vietnamese.

`cache_key_builder.go` produces the cache keys for per-project summaries; the bulk path invalidates the relevant keys after commit.

## Approval workflow

`workflow.go` plus `transaction_orchestrator.go` orchestrate approval. The transaction orchestrator is the canonical user of `domain.TransactionManager.RunInTransaction` for multi-aggregate writes (timesheet + ledger + event). On approval:

1. The timesheet row transitions to `approved`.
2. The revenue-receivable entry is booked (linked via `transaction_id`).
3. A domain event is emitted (`timesheet_approved`) for cache invalidation, settlement planning, and notification fan-out.
4. The transaction commits; after-commit callbacks fire.

`response_service.go` builds the API response shapes and cache-warm helpers used by the admin UI.

## Creator and per-row CRUD

`creator.go` is the per-row create path. `validator.go` is the per-row validator. `service.go` is the entry point; `operations.go` is the orchestrator.

## Cash-readiness forecast

`/api/v1/timesheets/cash-readiness` returns an advisory target-Kỳ forecast for the next pay date, excluding outstanding-payment backlog. The endpoint keeps legacy percentile fields and adds transparent cash decomposition (`observed_approved`, `pending_target_amount`, `expected_pending_amount`, `expected_future_amount`, `expected_payout`, `recommended_reserve`). All amounts use the configured weekly payable percentage, matching the cash written to the bank-transfer workbook. Reliability fields (`reliability_state`, `calibration_samples`, WAPE, bias, interval coverage, reserve shortfall rate) are measured only from unfiltered company forecasts resolved against distinct timesheets in generated weekly transfer files. The response is advisory and cannot initiate a top-up or payment. See ADR-010 for the statistical-over-ML rationale.

`/api/v1/timesheets/summary` reports the operational payment backlog in `pendingPaymentAmount` and `pendingEmployees` (pending-approval or approved timesheets with `payment_status: pending | failed`). The dashboard uses this to show accrued salary before approval without making it eligible for disbursement. The `status=pending_payment` list filter and the bulk-transfer planner use the stricter payroll-ready cohort (approved with `payment_status: pending | failed`); pending-approval rows never enter payment exports.

## Bank-transfer history (read-only)

`/api/v1/payrolls/bank-transfer-histories` returns read-only completed bank transfers grouped by employee and fixed weekly cycle:

```
Cycle 1 | Days 1-7
Cycle 2 | Days 8-14
Cycle 3 | Days 15-21
Cycle 4 | Days 22-28
```

The response is intentionally read-only and history-only. Partner access is filtered to accessible projects server-side. The `search` filter matches Vietnamese employee names without requiring diacritics and matches transfer codes or bank references case-insensitively.

## Tests

The package ships thorough coverage:

- `operations_test.go`, `validator_test.go`, `workflow_test.go` — service-level coverage.
- `bulk_operations.go` plus the bulk domain-service tests — bulk happy and error paths.
- `reject_unpaid_test.go` — the reject-after-paid invariant.
- `response_service_test.go` — API shape coverage.

## Frontend

- Admin and partner UIs in `frontend/src/pages/admin/`, `frontend/src/pages/partner/`, and the mobile twins in `frontend/src/pages/mobile/admin/`, `frontend/src/pages/mobile/partner/`.
- Reusable components in `frontend/src/components/timesheet/`.
- Bulk import UI drives the BCC import flow.

## Relationships

- **BCC import** — `features/bcc-import.md`. Creates or updates rows in bulk from the weekly Excel.
- **Salary disbursement** — `features/salary-disbursement.md`. Approved rows feed the bulk-transfer pipeline.
- **FlexPay** — `features/flexpay.md`. Attendance earnings bank the quota that constrains FlexPay.
- **Cash readiness** — `features/wallet-ledger.md` and ADR-010.
- **Domain layer** — `architecture/domain-layer.md`. The validation services live there.
