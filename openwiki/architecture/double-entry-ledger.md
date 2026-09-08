---
type: architecture
title: Double-Entry Ledger and Chart of Accounts
description: Wallet aggregate, double-entry ledger, chart of accounts, and settlement semantics that enforce financial integrity.
tags: [ledger, wallet, accounting, double-entry, chart-of-accounts, settlement]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-9f8ae5750267e989e29ed553
    resource: repo://backend/internal/domain/accounting_rules.go
  - id: openwiki-source-2832f9d8e7692341156b93f6
    resource: repo://backend/internal/domain/ledger.go
  - id: openwiki-source-ec558c4d7b469552241c6bba
    resource: repo://backend/internal/domain/settlement.go
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-c07a18afa073eea417be0530
    resource: repo://backend/internal/domain/wallet/wallet_balance.go
  - id: openwiki-source-a6c0b9f83f9f2a5c7f013564
    resource: repo://backend/internal/domain/wallet/wallet_topup.go
  - id: openwiki-source-0810c042d4aa7f287c3c4694
    resource: repo://backend/migrations/016_add_settlement_uuid.up.sql
  - id: openwiki-source-6898fb70359e7ca8a8e35e2e
    resource: repo://backend/migrations/019_create_chart_of_accounts.up.sql
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Double-Entry Ledger and Chart of Accounts

The payroll system maintains two complementary financial surfaces: the **wallet aggregate** (operational funds) and the **double-entry ledger** (accounting system of record). Both live in the domain layer with zero framework imports, and together they enforce the invariant that every VND moving through the system is structurally accounted for.

## Why double-entry

<!-- openwiki: broken internal link [../decisions/ADR-009-wallet-double-entry-ledger.md] file "../decisions/ADR-009-wallet-double-entry-ledger.md" does not exist. Fix the href or restore the target, then delete this comment. -->
Financial integrity is non-negotiable for a system handling payroll in VND. Single-entry bookkeeping offers no internal check that inflows equal outflows. The ledger enforces balance at the type level — every transaction creates paired debit/credit entries whose sums must match. This catches split-brain writes, partial failures, and bad import logic before they reach the books. The rationale and rejected alternatives (single-entry, event sourcing, manual `if status == X` checks) are recorded in [ADR-009](../decisions/ADR-009-wallet-double-entry-ledger.md).

## Chart of accounts

The chart of accounts is a small, opinionated hierarchy. Migration `019_create_chart_of_accounts.up.sql` seeds seven canonical accounts under five categories:

| Code | Name              | Category   |
| ---- | ----------------- | ---------- |
| 1000 | Cash              | asset      |
| 1100 | Accounts Receivable | asset    |
| 2000 | Accounts Payable  | liability  |
| 2100 | Loans Payable     | liability  |
| 5000 | Owner's Equity    | equity     |
| 4000 | Service Revenue   | revenue    |
| 3000 | Operating Expenses | expense   |

The Go domain defines a parallel whitelist of seven `LedgerAccount` constants — `cash`, `receivable`, `payable`, `expense`, `revenue`, `equity`, `loan` — in `backend/internal/domain/ledger.go`. Entries outside this whitelist fail `IsValidAccountType` and are rejected before they reach the database.

`Account` rows support a `parent_id` self-reference for hierarchies (`backend/internal/domain/account.go`), although the seven seeded accounts are flat.

## Ledger entry shape

A `LedgerEntry` (`repo://backend/internal/domain/ledger.go#L17-L37`) is one side of a balanced pair:

- `Debit` or `Credit` is set (VND, `int64`), never both, never zero.
- `Account` must be one of the seven whitelisted types.
- `Party` is a non-empty string naming the counterparty (`"Ngân hàng"` for the bank leg).
- `Date` may not be in the future — compared date-only against `clock.Now()` (Asia/Ho_Chi_Minh) to avoid midnight boundary issues.
- Soft-deletable via `DeletedAt`.
- Optionally linked to an `Asset` (proof file), a `Transaction` (user-facing record), or a `Settlement` (clearing event).

`ValidateAmounts` (`repo://backend/internal/domain/ledger.go#L207-L226`) enforces the exclusive-or on debit/credit; `IsValid` composes all four validators.

## Balance invariant

Every financial operation creates a `TransactionGroup` whose entries must balance. `AccountingRules.ValidateTransactionGroup` (`repo://backend/internal/domain/accounting_rules.go#L34-L72`) sums debits and credits and returns a Vietnamese validation error — `"giao dịch không cân bằng: nợ=%s, có=%s"` — when they diverge. Property-based tests in `accounting_rules_test.go` (using `gopter`) fuzz random inputs to verify the invariant across the full state space.

The same invariant is reachable via `ValidateBalancedTransaction` for explicit two-entry operations (`repo://backend/internal/domain/accounting_rules.go#L75-L114`), which also asserts the running amount on each side matches the transaction amount.

## Standard transaction shapes

The accounting rules module ships factory functions that emit pre-balanced groups for the common payroll patterns:

- `CreateBalancedSalaryTransaction` — debit `expense`, credit `cash`. Records that money left the bank to pay an employee.
- `CreateBalancedRevenueTransaction` — debit `cash`, credit `revenue`. Records money received from a client.
- `CreateReversalTransaction` — swaps debit and credit on an existing entry with a mandatory reason, producing the matching offset for correction flows.

These factories call `ValidateTransactionGroup` before returning, so the caller cannot accidentally persist an unbalanced group.

## Wallet aggregate

The wallet is a self-contained aggregate at `repo://backend/internal/domain/wallet/`. It owns its own repository interfaces (`repository.go`), separates operational funds from the accounting ledger, and exposes a `WalletService` interface (`service.go`) for application-layer callers. The aggregate ships three entity groups:

- **WalletTopup** — manual top-ups confirmed on insert, no status field (`wallet_topup.go`).
- **WalletPayment** — outbound transfers carrying `requested_amount`, `fee`, recipient, and a status string updated by the disbursement pipeline (`wallet_payment.go`).
- **WalletIPN** — immutable record of every inbound IPN message from the payment provider, with a `processing_status` of `pending | applied | ignored | error` (`wallet_ipn.go`).

The wallet balance is a derived computation: `Available`, `PendingIn`, `PendingOut`, and `Limbo` (unreconciled failed) — see `wallet_balance.go`. Limbo is visible but does not deduct from `Available`, which makes reconciliation gaps auditable without breaking the operational balance.

## Wallet payment state machine

A wallet payment traverses six states (`repo://backend/internal/domain/transactions/state_machine.go#L17-L24`):

```
pending    --verify-->        verified
pending    --reject-->        failed
verified   --authorise-->     authorised
verified   --reject-->        failed
authorised --ipn_completed--> completed
authorised --ipn_failed-->    failed
authorised --ipn_reversed-->  reversed
completed  --ipn_reversed-->  reversed
```

Triggers are `verify`, `reject`, `authorise`, `ipn_completed`, `ipn_failed`, and `ipn_reversed`. The FSM is built with `qmuntal/stateless` and emits on-entry hooks for `completed`, `failed`, and `reversed` so service code can react (publish events, schedule reconciliation). `IsTerminal` reports whether a state is final.

`Pending` is set on insert; `verified` follows the account check; `authorised` is the provider's synchronous accept; `completed` and `failed` arrive via IPN; `reversed` is a post-settlement reversal.

## Settlement

`Settlement` rows (`repo://backend/internal/domain/settlement.go`) tie ledger clearing events back to user-facing transactions. Each carries:

- A `TransactionID` pointing at the originating `Transaction`.
- An `Amount` and `SettlementDate`.
- An optional `settlement_uuid` (unique-indexed) acting as an idempotency key for async processors — replays of the same settlement UUID are safe.
- An optional proof asset and notes.

`Settlement.Validate()` (`repo://backend/internal/domain/settlement.go#L57-L84`) rejects zero amounts, missing counterparties, and dates in the future. Settlement entries then drive the corresponding ledger writes through the application service layer.

## Settlement entries

`SettlementEntries` (`repo://backend/internal/domain/settlement_entries.go`) describe the per-leg breakdown of a settlement when it clears multiple ledger accounts at once. Settlement entries are the bridge between the wallet's provider-level view (a single IPN settling one payment) and the ledger's account-level view (which Cash, Receivable, or Expense accounts to touch).

## Reconciliation

The wallet service exposes `UploadReconciliation` (matches a 9Pay CSV against `wallet_payments`) and `ExportReconciliationReport` (monthly CSV covering the cycle 16th of the previous month through the 15th of the current month). Both run as async jobs tracked by `ReconciliationJob` (`wallet_balance.go`).

The pipeline that posts completed wallet payments to the ledger lives in `GetCompletedUnsettled` (`repo://backend/internal/domain/wallet/repository.go#L30-L39`), which deliberately buckets by `settled_at` with `created_at` fallback so a missed pickup run can backfill stranded payments by completion day.

## Cross-cutting consequences

<!-- openwiki: broken internal link [../decisions/ADR-003-payment-provider-abstraction.md] file "../decisions/ADR-003-payment-provider-abstraction.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- **Provider abstraction**: the wallet is provider-agnostic (OnePay in production, 9Pay in sandbox) — see `wallet_payment.go` (`Provider` field) and [ADR-003](../decisions/ADR-003-payment-provider-abstraction.md).
- **Forecast surfaces only**: `WalletDemandForecastResponse` (`forecast.go`) is advisory display only. The comment is explicit: it must not feed `SyncBalance` or any auto top-up.
- **Invariant in tests**: the accounting rules are property-tested, but the wallet state machine is enforced by the FSM itself — invalid transitions fail at the trigger call rather than at persist time.
<!-- openwiki: broken internal link [../decisions/ADR-001-ddd-clean-architecture.md] file "../decisions/ADR-001-ddd-clean-architecture.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- **Domain-layer purity**: the wallet aggregate and ledger entities import only `internal/pkg/clock` and `internal/constants` from outside the standard library, satisfying the framework-free domain invariant of [ADR-001](../decisions/ADR-001-ddd-clean-architecture.md).

## Related pages

- [Transaction Manager and Outbox](./transaction-manager-and-outbox.md) — how ledger writes participate in the unit-of-work boundary and when cache invalidation runs.
- [Salary Disbursement and Payment Providers](../features/salary-disbursement-and-payment-providers.md) — wallet payments produced by the disbursement pipeline.
- [FlexPay Advance Payments](../features/flexpay-advance-payments.md) — advance-payment request lifecycle and its ledger settlement path.
