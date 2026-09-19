---
type: feature
title: Wallet & Double-Entry Ledger
description: The wallet aggregate (balance, top-ups, payments, IPN), the qmuntal/stateless state machine, the double-entry ledger with property-based accounting invariants, settlement, and the statistical demand forecast that drives top-up timing.
tags: [feature, wallet, ledger, double-entry, state-machine, ipn, forecast, accounting]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-a16ad93c72e763068e8b900c
    resource: repo://backend/internal/app/workers/disbursement_poller_worker.go
  - id: openwiki-source-88d820f8ba65016c66382371
    resource: repo://backend/internal/app/workers/status_inquiry_worker.go
  - id: openwiki-source-9f8ae5750267e989e29ed553
    resource: repo://backend/internal/domain/accounting_rules.go
  - id: openwiki-source-a5c9de3ea4280bad6936ca7f
    resource: repo://backend/internal/domain/AGENTS.md
  - id: openwiki-source-ec558c4d7b469552241c6bba
    resource: repo://backend/internal/domain/settlement.go
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-cb5b90656b301b764e1fb4b6
    resource: repo://backend/internal/domain/wallet/forecast.go
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-8e89964ec2029eb23df68d1e
    resource: repo://backend/internal/domain/wallet/service.go
  - id: openwiki-source-a6c0b9f83f9f2a5c7f013564
    resource: repo://backend/internal/domain/wallet/wallet_topup.go
  - id: openwiki-source-98b4fef5bee3b5a0d880f16b
    resource: repo://docs/api.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: Wallet & Double-Entry Ledger

The wallet holds the funds used to pay salaries and FlexPay advances. Every VND is accounted for in a double-entry ledger, and every payment flows through a formally specified state machine that is fed by the payment provider's IPN webhooks. Financial integrity is structural: invalid transitions and unbalanced entries are rejected by code, not by convention.

ADR-009 is the source of truth for the wallet design.

## The wallet aggregate

`internal/domain/wallet/` is a self-contained bounded context:

| File | Purpose |
|------|---------|
| `wallet_payment.go` | `WalletPayment` type and filter — a payment to an employee's bank |
| `wallet_topup.go` | `WalletTopup` type — manual top-ups via bank app (no status; confirmed on insert) |
| `wallet_balance.go` | Balance computation (provider + local reconciliation) |
| `wallet_ipn.go` | IPN handling types and verification entry points |
| `service.go` | `WalletService` interface — the aggregate's use-case surface |
| `repository.go` | Repository interfaces scoped here (not in the global ports directory) |
| `forecast.go` | Statistical demand forecast types |

The `WalletService` interface (`internal/domain/wallet/service.go`) is the boundary between the aggregate and the application layer. It owns: balance, sync balance (provider), unified transaction view, top-up CRUD, payment reads, payment resolve (admin action), reconciliation (9Pay CSV matching against wallet payments only), and monthly reconciliation report export.

The repository interfaces (`WalletTopupRepository`, `WalletPaymentRepository`, `WalletIPNRepository`) live alongside the aggregate, not in `internal/domain/ports/`, so the wallet is portable as a unit.

## State machine

`internal/domain/transactions/state_machine.go` implements the wallet payment FSM using `github.com/qmuntal/stateless`. States and triggers:

```
States:    pending → verified → authorised → completed | failed | reversed
Triggers:  verify, reject, authorise, ipn_completed, ipn_failed, ipn_reversed
Terminal:  completed, failed, reversed
```

The transitions:

```
pending     --verify-->      verified
pending     --reject-->      failed
verified    --authorise-->   authorised
verified    --reject-->      failed
authorised  --ipn_completed--> completed
authorised  --ipn_failed-->    failed
authorised  --ipn_reversed-->  reversed
completed   --ipn_reversed-->  reversed
```

The FSM exposes `OnEnterCompleted`, `OnEnterFailed`, `OnEnterReversed` hook callbacks. The wallet service wires these to write ledger entries and emit post-commit settlement events.

Invalid transitions return an error rather than silently failing. There is no manual `if status == X` check anywhere; the FSM is the contract.

## Top-up, payment, IPN, settlement

### Top-up flow

`WalletTopup` represents a manual wallet top-up performed by an admin via a bank app. The top-up is confirmed on insert — there is no status field because there is no asynchronous confirmation. The admin records the amount, bank reference (`BankRef`), and a note; the row is the audit record.

### Payment flow

1. **Booking** — the disbursement service creates a `wallet_payment` row with status `pending`. The wallet's available balance is checked at this point.
2. **Authorisation** — `disbursement_execute_worker` calls the active provider (OnePay/9Pay) with the recipient, amount, and the row's `TxnID`. The provider returns an invoice number; the row transitions `pending → verified → authorised`.
3. **IPN confirmation** — the provider POSTs an IPN to `/api/v1/webhooks/{provider}/ipn`. The handler verifies the signature and enqueues an `ipn:process` asynq task. The worker looks up the payment by invoice number and feeds the matching trigger into the FSM (`ipn_completed` / `ipn_failed` / `ipn_reversed`).
4. **Post-commit** — on terminal transition, ledger entries are written, the settlement event fires, and the cache invalidates.

Tables touched:

```
wallet_payments   -- one row per disbursement, lives through FSM
wallet_topups     -- immutable manual top-up audit
wallet_ipn        -- durable IPN delivery record (idempotency, audit)
```

### Settlement

`Settlement` (`internal/domain/settlement.go`) is the persistent record that an external cash event settled a transaction. It carries a `SettlementUUID` (idempotency key, unique index) for safe async processing. `Validate()` enforces:

- `transaction_id` required.
- `amount > 0`.
- `settlement_date` not in the future (date-only comparison to avoid timezone edges).
- `created_by` required.

`SettlementRepository` supports `Create`, `GetByID`, `GetByTransactionID`, `GetTotalSettledAmount`, `List`, `Count`. Soft-deleted via `DeletedAt`.

The settlement path runs in two flavors:

1. **Wallet-driven** — `wallet_settlement_worker.go` reconciles each wallet payment against the employee's open advance_payment_requests. See `features/flexpay.md`.
2. **Manual / external** — admin records a settlement against an outstanding transaction (cash receipt, bank transfer); the row writes through `SettlementRepository.Create` inside a transaction.

## Double-entry ledger

`internal/domain/ledger.go` defines `LedgerEntry` and `Transaction` aggregates. `internal/domain/accounting_rules.go` enforces the rules.

`AccountingRules.ValidateTransactionGroup` sums debits and credits across a group of entries and rejects any group where they do not match. `ValidateBalancedTransaction` enforces the simpler debit/credit pair shape. The accounting helpers live in `internal/app/accounting/` (a `Money` type with safe arithmetic).

The rules are verified by property-based tests (`accounting_rules_test.go` using `gopter`): random balanced and unbalanced groups are generated, balanced groups must validate and unbalanced groups must fail. This catches off-by-one and rounding regressions that table-driven tests miss.

## Cash-readiness and demand forecast

The wallet's demand forecast (`internal/domain/wallet/forecast.go`) is statistical per ADR-010 (statistical over ML). It powers `/api/v1/wallet/demand-forecast` and `/api/v1/timesheets/cash-readiness` — both are advisory and never feed `SyncBalance` or auto top-up.

`WalletDemandForecastResponse` carries:

- A cohort series (`WalletDemandPeriod` with cumulative `WalletDemandPoint` rows) for the chart.
- A balance prediction (`WalletDemandPrediction`) for the card, with newsvendor / tail-risk fields (`P50Reference`, `P90Reference`, `P99Reference`, `CoverageProbability`).

`CashReadinessForecast` (`internal/app/services/cash_readiness_forecast*.go`) is the application-layer implementation with backtest (`cash_readiness_backtest_test.go`) and forecast-cache (`cash_readiness_forecast_cache_test.go`) coverage. The forecast uses Monte Carlo or gamma-fit depending on history depth; the response carries `Method` and `Confidence`.

Reliability fields (`reliability_state`, `calibration_samples`, WAPE, bias, interval coverage, reserve shortfall rate) are measured only from unfiltered company forecasts resolved against distinct timesheets in generated weekly transfer files. Filtered forecasts are never mixed into company accuracy.

## Reconciliation

Two reconciliation paths coexist:

- **9Pay reconciliation CSV** — the wallet service exposes `UploadReconciliation(ctx, csvData, userID)`. The CSV is matched against `wallet_payments` only (not against manual settlements).
- **Monthly reconciliation report** — `ExportReconciliationReport(month)` returns the cycle-window report (16th of previous month through 15th of current month) for finance review.

## Settlement event handler

`internal/infra/events/handlers/settlement_event_handler.go` consumes wallet settlement events and reconciles any out-of-band updates. Combined with `cache_invalidation_handler.go` and `audit_event_handler.go`, the wallet publishes a small set of events that downstream consumers react to without coupling to the wallet's internal state machine.

## Where it lives

- Domain: `internal/domain/wallet/`, `internal/domain/transactions/`, `internal/domain/ledger.go`, `internal/domain/accounting_rules.go`, `internal/domain/settlement.go`.
- Application: `internal/app/services/wallet_bulk/`, `internal/app/services/wallet_service.go`, `internal/app/services/cash_readiness_forecast*.go`, `internal/app/services/settlement/`.
- Infra: `internal/infra/disbursement/onepay/`, `internal/infra/disbursement/ninepay/`, `internal/infra/events/handlers/settlement_event_handler.go`.
- Workers: `internal/app/workers/ipn_process_worker.go`, `wallet_settlement_worker.go`, `verified_transfer_recovery.go`, `transfer_paid_amounts.go`.

## Relationships

- **Salary disbursement** — `features/salary-disbursement.md`. Books the wallet payments and drives them to terminal.
- **FlexPay** — `features/flexpay.md`. Settlement reconciles advances against salary.
- **Payment providers** — `integrations/payment-providers.md`. The IPN contract.
- **Architecture / domain** — `architecture/domain-layer.md`. The wallet aggregate and FSM in detail.
- **Architecture / application** — `architecture/application-services.md`. The transaction manager that owns multi-aggregate writes.
