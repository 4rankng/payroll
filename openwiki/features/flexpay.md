---
type: feature
title: FlexPay (Advance Payments)
description: Employee advance-request lifecycle, the tiered fee schedule, per-employee kill switch, the wallet-backed funds flow, and the salary-notification hook that tells employees what their next salary will deduct.
tags: [feature, flexpay, advance-payment, fee-schedule, salary-notification, zns]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-64a777566054a9461a71ef47
    resource: repo://backend/internal/app/services/flex_pay/flex_pay_settlement_service.go
  - id: openwiki-source-8a3b339ddf337ea9fa11ee6e
    resource: repo://backend/internal/app/workers/flexpay_salary_notification_worker.go
  - id: openwiki-source-0561ebb6fa3a9d3f78919d06
    resource: repo://backend/internal/domain/advance_payment_fee_schedule.go
  - id: openwiki-source-43c18e617136f11e46e05ed9
    resource: repo://backend/internal/domain/flexpay_salary_notification.go
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-9bf61dc2d18ff0e4c789f4e6
    resource: repo://backend/migrations/100_add_flexpay_salary_notifications.up.sql
  - id: openwiki-source-22c979da92ff10b8f954b8e0
    resource: repo://backend/migrations/105_project_employees_advance_request_enabled.up.sql
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: FlexPay (Advance Payments)

FlexPay lets an employee request an advance against their next salary. The system computes the fee from an admin-managed tiered schedule, debits the wallet, transfers the net amount to the employee's bank, and at salary time reconciles the advance against the disbursed salary so the employee sees a clear deduction. Every step is auditable; the financial critical path uses the wallet state machine and double-entry ledger.

## Lifecycle

```
requested → approved → disbursing → disbursed → reconciled
                ↓
              rejected (terminal)
```

- **requested** — employee submits a request via the mobile app (or the admin submits on their behalf). The request amount and the eligible advance quota (from attendance earnings) are validated.
- **approved** — admin or partner approves. `advance_request_enabled` on `project_employees` (migration 105) is the per-employee kill switch that blocks new requests; existing pending/approved rows are untouched.
- **disbursing** — the disbursement service books a `wallet_payment` and enqueues the `disbursement:execute` task to the payment provider. See `features/salary-disbursement.md`.
- **disbursed** — provider IPN confirms settlement (`ipn_completed` trigger); wallet state advances to `completed`.
- **reconciled** — at weekly payroll time, the FlexPay balance is deducted from the salary disbursement. The reconciliation exporter (`flex_pay_reconciliation_exporter.go`) emits the matching report.

Rejection can happen from validation (amount, quota, missing bank details) or admin/partner action.

## Fee schedule

The fee for a request is computed from a per-day-effective tiered schedule stored in a single settings row:

```
settings.key = "advance_payment_fee_schedules"
settings.value = JSON array of FeeScheduleEntry
```

The whole list lives in one settings row; per-entry CRUD parses the JSON, mutates the slice, and writes the row back inside a `SELECT … FOR UPDATE` (`fee_schedule_service.go`). `FeeScheduleEntry.Validate()` enforces:

- At least one tier.
- First tier has `MinAmount == 0`.
- Tiers strictly increasing by `MinAmount`.
- Percentages in `[0, 100]`.
- `MinFeeVND >= 0` (`0` disables the minimum-fee floor, allowing a fully free advance).
- `effective_date` in `YYYY-MM-DD` (string-comparison matches chronological).

`ResolveFee(requestAmount)` picks the highest-`MinAmount` tier whose `MinAmount <= amount`, then returns `max(amount * percentage / 100, MinFeeVND)`. `ActiveFeeScheduleAt(entries, at)` returns the entry with the latest `EffectiveDate <= at`; the same input is used by both ad-hoc preview and the disbursement-time fee calculation so the customer-visible fee always matches what is actually charged.

## Per-employee kill switch

`project_employees.advance_request_enabled` (migration 105, `TINYINT(1) NOT NULL DEFAULT 1`) is the per-employee "tạm ngừng ứng lương" kill switch:

- `0` blocks new advance requests for the employee (both the regular flow and the self-check-in flow).
- Existing pending or approved requests are untouched.
- Default `1` preserves current behavior.

The frontend label is "tạm ngừng ứng lương"; see `features/attendance-geofence.md` for the parallel `pending_check_in_enabled` deferred activation.

## Funds flow

FlexPay advances are paid out of the wallet. The wallet holds pre-funded balance from top-ups (admin-issued top-ups via OnePay/9Pay). When an advance is disbursed:

1. The disbursement service creates a `wallet_payment` row with status `pending`.
2. The `disbursement_execute_worker` calls the active provider (OnePay in production, 9Pay in sandbox) to push funds to the employee's bank account.
3. The IPN handler processes the provider webhook and feeds the matching `Trigger` into the wallet state machine: `ipn_completed` → `completed`, `ipn_failed` → `failed`, `ipn_reversed` → `reversed`.
4. On terminal completion, the service writes ledger entries and emits the post-commit events (settlement, audit, notification).

See `features/wallet-ledger.md` and `integrations/payment-providers.md` for the wallet and provider side.

## Salary notification

Migration 100 introduced `flexpay_salary_notifications` — a durable delivery record that tells each employee, before payday, what their next salary will be and how much will be deducted for FlexPay advances. The notification goes out via Zalo ZNS (the same channel used for password reset).

The table:

```
flexpay_salary_notifications(
  id, asset_id, project_id, employee_id, template_id,
  phone, customer_name, max_amount, expiry_date,
  status, attempt, lease_expires_at, enqueued_at, sent_at,
  provider_msg_id, provider_code, last_error, created_at, updated_at
)
```

- `UNIQUE KEY (asset_id, project_id, employee_id, template_id)` — import retry is safe while retaining a complete delivery audit.
- `INDEX (status, lease_expires_at)` — the recovery sweeper finds work to claim.

Status values: `pending`, `processing`, `sent`, `failed`, `suppressed` (the last three are terminal via `IsTerminal()`). The repository interface (`FlexPaySalaryNotificationRepository`) implements a lease-based claim model:

- `CreateIfAbsent` — insert if no row exists for the (asset, project, employee, template) tuple.
- `Claim(id, now, leaseUntil)` — atomically move `pending` → `processing` with a lease.
- `ReleaseForRetry` — return to retryable state with a recorded reason.
- `MarkSent` / `MarkFailed` / `MarkSuppressed` — terminal transitions.
- `ListRecoverable(now, limit)` — sweep for stale leases or stuck `pending` rows.

The asynq worker `flexpay_salary_notification_worker.go` consumes the queue, calls `Claim`, invokes the ZNS adapter, and writes the terminal state. The recovery sweeper (also asynq) calls `ListRecoverable` to handle leases that expired mid-flight.

## Reconciliation

`flex_pay_reconciliation_exporter.go` (in `internal/app/services/flex_pay/`) emits the matching report for a given pay-period window: which employees had advances, what was disbursed, what was reconciled against the salary. The report is consumed by the admin reconciliation UI and is the basis for the salary deduction line item employees see.

`flex_pay_settlement_service.go` performs the per-employee settlement: it walks the employee's open `advance_payment_request`s, marks each as reconciled against the salary disbursement, and updates the running balance. The service runs inside a transaction so the request, the ledger entries, and the salary batch row all commit atomically.

## Service packages

`internal/app/services/advance_payment/` and `internal/app/services/flex_pay/` together cover the lifecycle:

- `admin_flexpay_employees.go`, `admin_flexpay_import.go` — admin-side queries and the bulk import.
- `calculator.go` — wraps `domain.ResolveFee` and the quota check.
- `admin_bulk_transfer.go`, `admin_export.go`, `admin_queries.go`, `bank_result_rows.go` — admin and partner flows.
- `email_template.go` — salary-notification email template.
- `flex_pay_reconciliation_exporter.go` — the report emitter.
- `flex_pay_settlement_service.go` — the settlement runner.

## Tests

- `advance_payment_fee_schedule_test.go` (domain) — tier validation, `ResolveFee`, `ActiveFeeScheduleAt`.
- `admin_flexpay_import_test.go`, `calculator_test.go`, `bank_result_rows_test.go` — service tests.
- `flex_pay_reconciliation_exporter_test.go`, `flex_pay_settlement_service_test.go` — settlement/reporting tests.

## Relationships

- **Wallet & ledger** — funds source and state machine. See `features/wallet-ledger.md`.
- **Salary disbursement** — the consumer pipeline. See `features/salary-disbursement.md`.
- **Timesheets** — attendance earnings bank the quota that constrains `max_adv_amount`. See `features/timesheet.md` and `features/attendance-geofence.md`.
- **Zalo ZNS** — the notification channel. See `integrations/zalo-otp.md`.
- **Frontend** — the employee mobile flow (`/advance-payment`, see `frontend/employee-mobile.md`) and the admin reconciliation UI (`/admin/flexpay`).
