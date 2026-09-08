---
type: feature
title: FlexPay Advance Payments
description: Advance-payment request lifecycle, tiered fee schedule with effective-date history, and reconciliation against the ledger.
tags: [flexpay, advance-payment, fee-schedule, reconciliation, ledger]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-0561ebb6fa3a9d3f78919d06
    resource: repo://backend/internal/domain/advance_payment_fee_schedule.go
  - id: openwiki-source-c63aac3b5b210a2bb74f41b6
    resource: repo://backend/internal/domain/advance_payment_request.go
  - id: openwiki-source-547ebfab24a9ad734665d185
    resource: repo://backend/internal/domain/advance_payment.go
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# FlexPay Advance Payments

FlexPay is the employee-facing advance-payment flow. An employee reads their monthly quota (`AdvancePayment`), submits a request (`AdvancePaymentRequest`), the system resolves a fee from the active fee schedule, and on settlement the request becomes a transaction that posts to the wallet and the ledger. Reconciliation joins FlexPay requests to ledger entries and bulk-transfer settlements so the admin can audit what was advanced, what was settled, and what remains outstanding.

## Quota

`AdvancePayment` (`repo://backend/internal/domain/advance_payment.go#L20-L38`) is one row per `(project, employee, for_month)`. It carries:

- `MaxAdvAmount` — the cap, in VND. For the admin-upload (BCC) flow this is set directly from the upload. For the self-check-in flow it is computed as `floor(Salary × configured_percent / 100)` from the employee's earned wages.
- `Salary` — total earned wages (100%) for the month from check-in/out. Used only by the self-check-in flow.
- `LastAppliedAssetID` — the most recent attendance asset applied to this quota.

`DefaultSelfCheckInAdvancePercentage = 70` (`repo://backend/internal/domain/advance_payment.go#L11`) is the fallback percent when the Admin setting is missing or invalid. `QuotaCreditHoldDuration = 24 * time.Hour` (`repo://backend/internal/domain/advance_payment.go#L17`) is the deferred-credit window: self-check-out enqueues a credit task after the configured hold, while admin manual approvals credit immediately and bypass it.

`AdvancePaymentRepository.SumPendingEarningsByEmployeeMonth` returns earnings held in the credit-hold window (`quota_credited_at IS NULL, earning_amount > 0`). This is shown separately from `Salary` on the advance screen so the worker can see money is coming; it is intentionally NOT part of the configurable advanceable cap.

`GetQuotaAnomalies` enumerates rows violating named invariants: `drift` (`max_adv != floor(salary × configured_percent / 100)`), `missing` (earning attendance but no advance row), or `stale` (`salary > 0` but assignment check-in disabled). The admin dashboard surfaces these counts so quota-drift bugs do not silently inflate caps.

## Request lifecycle

`AdvancePaymentRequest` (`repo://backend/internal/domain/advance_payment_request.go#L20-L42`) is one submitted advance. It carries `RequestAmount`, `Fee`, `ProviderFee`, `NetAmount`, and a `Status` from the enum:

| Status | Vietnamese | Meaning |
|---|---|---|
| `PENDING` | — | submitted, awaiting approval |
| `APPROVED` | — | approved, pending payment |
| `CANCELLED` | — | cancelled before payment |
| `COMPLETED` | — | paid out and ledger-settled |
| `FAILED` | — | payout failed |

`ValidateRequestAmount` enforces `request_amount >= 10000 VND` and produces Vietnamese validation messages on failure. The minimum is hard-coded at the domain layer because changing it would affect every active fee tier.

The request rows link forward to their originating `AdvancePayment` (quota), their `Project` and `Employee`, and `SettlementTransaction` (a `Transaction` row that holds the corresponding receivable). `ReceivableSettledAt` is set when the client pays the receivable — the moment the receivable is cleared.

## Fee schedule

`FeeScheduleEntry` (`repo://backend/internal/domain/advance_payment_fee_schedule.go#L33-L41`) is one historical schedule row stored as a JSON array under the single Settings row whose key is `advance_payment_fee_schedules`. Each entry has:

- `ID` — UUID, stable per entry so PATCH/DELETE can target a single row.
- `EffectiveDate` — YYYY-MM-DD, the canonical wire format that string-compares the same way it sorts chronologically.
- `Tiers` — sorted ascending by `MinAmount`; the first tier must have `MinAmount == 0`.
- `MinFeeVND` — minimum fee floor (0 disables it, allowing a fully free advance).
- `Notes` and `CreatedByUserID` — audit metadata.

`Validate` (`repo://backend/internal/domain/advance_payment_fee_schedule.go#L47-L71`) enforces:

- At least one tier.
- First tier has `MinAmount == 0`.
- Tiers strictly increasing by `MinAmount`.
- Percentages in `[0, 100]`.
- `MinFeeVND >= 0`.
- `EffectiveDate` parses as YYYY-MM-DD.

`ResolveFee` picks the highest-`MinAmount` tier whose `MinAmount <= request_amount` and returns `max(amount × percentage/100, MinFeeVND)`. `ActiveFeeScheduleAt(entries, at)` returns the entry whose `EffectiveDate` is the latest `<= at`; nil if all entries are future-dated.

Per-entry CRUD on the JSON array is done by reading the row, mutating the slice, and writing back inside a `SELECT … FOR UPDATE` so concurrent edits cannot interleave.

## Reconciliation

`FlexPayReconciliationService` (`repo://backend/internal/domain/services/flex_pay_reconciliation_service.go#L13-L36`) joins `AdvancePaymentRequest` against `AdvancePayment` (quota), `Project`, and `Employee` to produce per-project, per-employee report rows. The report covers a salary period (`SalaryPeriodFrom`, `SalaryPeriodTo`), sums `RequestAmount` and `Fee`, and emits the request IDs so an admin can drill from the report to the originating requests.

The reconciliation output is consumed by the export flow and feeds the dashboard tiles `B4` (throughput) and `K` (outstanding receivables).

## Linkage to ledger and wallet

A `COMPLETED` FlexPay request is the entry point to the financial surfaces:

- `SettlementTransactionID` links the request to the `Transaction` row that carries the receivable.
- The wallet payment that actually disburses the net amount is produced by the disbursement pipeline (see [Salary Disbursement and Payment Providers](./salary-disbursement-and-payment-providers.md)) and reconciles to the request via the `WalletPayment.RequestID` field.
- When the client settles the receivable, `ReceivableSettledAt` is set and the ledger entries tied to the original transaction are reversed.

This three-way linkage (FlexPay request → wallet payment → ledger transaction) is what makes "did we actually pay this advance?" a single SQL query away.

## Vietnamese UI strings

The domain validation messages are kept in Vietnamese because they surface directly to the user through the API:

- `"Tháng là bắt buộc"` — `for_month` is required.
- `"Định dạng tháng không hợp lệ (YYYY-MM)"` — invalid month format.
- `"Số tiền yêu cầu là bắt buộc"` — request amount required.
- `"Số tiền yêu cầu tối thiểu là 10,000 VND"` — minimum request amount.

Product-facing copy ("Yêu cầu tạm ứng", "Hạn mức tháng này", etc.) lives inline in the React components — there is no i18n layer.

## Related pages

- [Double-Entry Ledger and Chart of Accounts](../architecture/double-entry-ledger.md) — where FlexPay request settlements post.
- [Salary Disbursement and Payment Providers](./salary-disbursement-and-payment-providers.md) — wallet payments produced from FlexPay requests.
- [Timesheet Engine](./timesheet-engine.md) — self-check-in earnings drive the quota for the self-check-in advance flow.
