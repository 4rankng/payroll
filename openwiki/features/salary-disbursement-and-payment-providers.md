---
type: feature
title: Salary Disbursement and Payment Providers
description: Bulk transfer pipeline, provider-agnostic disbursement abstraction (OnePay prod, 9Pay sandbox), IPN handling, and asynq-driven background workers.
tags: [disbursement, onepay, ninepay, bulk-transfer, ipn, asynq]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-238358ccfe4c6654aa4041a4
    resource: repo://backend/internal/domain/wallet/wallet_ipn.go
  - id: openwiki-source-49857e2784126dfe96ada34e
    resource: repo://docs/decisions/ADR-003-payment-provider-abstraction.md
  - id: openwiki-source-28d4ebbc4a3f54433e091cbe
    resource: repo://docs/decisions/ADR-005-asynq-background-jobs.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Salary Disbursement and Payment Providers

The salary disbursement flow takes approved timesheets (and FlexPay requests) and moves VND from the company wallet into employee bank accounts through a third-party payment provider. The flow is provider-agnostic by design (ADR-003): production runs on OnePay, sandbox and development on 9Pay, and the active provider is selected at request time from `Settings.disbursement_provider`. Background work — bulk execution, IPN processing, status polling — is driven by asynq tasks on Redis-backed queues (ADR-005).

## Provider abstraction

`DisbursementProvider` lives in `domain/ports/infrastructure/disbursement.go` and exposes the minimum surface every provider must implement:

```go
type DisbursementProvider interface {
    Name() string
    InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResult, error)
    VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*WebhookEvent, error)
}
```

Two implementations ship in `internal/infra/disbursement/`:

- `onepay/` — production. Includes OnePay-specific client, signing, error codes, portal auth, and the `ReportExporter` capability for reconciliation CSV export.
- `ninepay/` — sandbox/development.

The active provider is selected at request time from `Settings.disbursement_provider`. Switching providers requires only a settings change — no recompile, no code path change.

### Optional capabilities

Provider-specific features are exposed as optional interfaces discovered via type assertion rather than baked into the core interface. This keeps 9Pay from having to stub methods it does not support:

| Interface | Purpose | OnePay | 9Pay |
|---|---|---|---|
| `StatusPoller` | Poll transfer status | yes | no (IPN-only) |
| `AccountVerifier` | Pre-flight bank account check | yes | no |
| `BalanceReporter` | Partner balance query | yes | no |
| `ReportExporter` | Reconciliation CSV export | yes | no |
| `ErrorTranslator` | Vietnamese error code translation | yes | yes |
| `TransferLimiter` | Per-transfer amount bounds | yes | yes |

The runtime check pattern:

```go
if poller, ok := provider.(StatusPoller); ok {
    status, err := poller.PollStatus(ctx, transferID)
}
```

### Webhook security

Inbound IPNs arrive at the HTTP webhook handlers and pass through `ip_whitelist.go` middleware that enforces the provider's published IP ranges. The disbursement provider's `VerifyAndParseWebhook` checks the request signature before any state change. IPN timing is roughly 3 seconds after batch completion on the provider side.

## Bulk transfer pipeline

A bulk transfer batch (`bulk_transfer_batch.go`) is the input to the pipeline. It groups the lines that the admin uploaded from a weekly BCC/MBank export, plus any FlexPay-approved advances scheduled for the same payout window. The pipeline:

1. **Batch creation** persists `BulkTransferBatch` and one `BulkTransferFile` per uploaded file.
2. **Asynq enqueue** (`backend/internal/infra/asynq/`) hands the batch to a worker via the bulk-transfer task type.
3. **Per-line wallet payment** — for each line, the worker calls `DisbursementProvider.InitiateTransfer`, which produces a `WalletPayment` row in `pending` state with `provider` set to `onepay` or `ninepay`.
4. **Status updates** flow back via IPN (`provider → webhook → mux → handlers → wallet_payment_service`). The provider IPN includes the invoice number, status, and amount; the service matches it to the matching `WalletPayment` and transitions the FSM.
5. **Status polling** — OnePay supports `StatusPoller.PollStatus`; this is the fallback path when an IPN is missed. 9Pay does not poll, so missed IPNs surface as `pending` payments until the operator investigates.
6. **Settlement posts to the ledger** — when a payment transitions to `completed`, the wallet settlement worker picks it up and posts the corresponding `expense` / `cash` pair to the ledger (see [Double-Entry Ledger and Chart of Accounts](../architecture/double-entry-ledger.md)).

The FSM for each `WalletPayment` is described in the architecture page (`pending → verified → authorised → completed | failed | reversed`).

## IPN handling

`WalletIPN` (`repo://backend/internal/domain/wallet/wallet_ipn.go`) records every inbound IPN message. Each row is immutable once `processing_status` is set to its final value (`applied | ignored | error`), so the audit trail cannot be rewritten. The row carries:

- `Provider`, `InvoiceNo`, `RequestID` — the provider's identifiers, used to match the IPN back to a wallet payment.
- `Status`, `Amount`, `RawErrorCode`, `FailureReason` — the provider-reported outcome.
- `RawPayload` — the full provider payload (binary), for forensics.
- `ProcessingStatus`, `ProcessingError` — what the service did with it.
- `WalletPaymentID` — the matched payment, set on `applied`.

The IPN handler in `infra/events/wallet_ipn_handler.go` is the canonical "provider spoke, what do we do" entry point. It runs the matched payment through the FSM (via the `transactions.Build` state machine), which then fires on-entry hooks for `completed`, `failed`, and `reversed`.

## Asynq-driven workers

ADR-005 records the design. The asynq setup is split:

- `backend/internal/infra/asynq/client.go` — enqueue side, with retry policies, scheduling, and unique-task enforcement (no duplicate bulk transfer dispatch).
- `backend/internal/infra/asynq/handlers.go` — dispatches task types to worker functions.
- `backend/internal/infra/asynq/mux.go` — registers task-type → handler mappings (`Mux.HandleFunc(...)`).
- `backend/internal/infra/asynq/server.go` — worker pool, concurrency, queue priorities, and the HTTP monitoring endpoint.
- `backend/internal/app/workers/` — the worker functions themselves, where the actual disbursement logic runs.

Task types include bulk-transfer execution, disbursement processing, IPN handling, status polling (OnePay), employee imports, and audit log writing. Redis AOF persistence is required for task durability.

The asynq server starts as a goroutine from `bootstrap/server.go`. Task uniqueness prevents duplicate processing — uploading the same batch file twice does not enqueue two workers.

## Audit and reconciliation

The wallet reconciliation CSV upload (`WalletService.UploadReconciliation`) matches a provider CSV against `wallet_payments` rows, producing a `ReconciliationJob` with matched/unmatched counts. The monthly report (`ExportReconciliationReport`) covers the cycle 16th of the previous month through 15th of the current month. The OnePay `ReportExporter` capability is the only implementation today; 9Pay relies on the wallet-side upload.

Each completed payment that settles posts a ledger entry; the wallet-vs-ledger reconciliation is the `expense` total against the `cash` total over the same range. Drift surfaces in the admin dashboard as a flag, not a failure — the system continues to operate, but the discrepancy is auditable.

## Production invariants

These are non-negotiable and appear as Claims:

- **Production uses OnePay.** `Settings.disbursement_provider` must point at OnePay in production deployments. 9Pay is sandbox-only.
- **The provider abstraction is the only sanctioned extension point.** New providers implement `DisbursementProvider` (and any optional capabilities they support). They never reach into OnePay or 9Pay internals.
- **IPN processing is the source of truth for terminal state.** Status polling (where available) is a backstop, not the primary update path.

## Related pages

- [Double-Entry Ledger and Chart of Accounts](../architecture/double-entry-ledger.md) — what the completed payments post to.
- [Transaction Manager and Outbox](../architecture/transaction-manager-and-outbox.md) — how disbursement writes participate in the unit-of-work.
- [Timesheet Engine](./timesheet-engine.md) — the upstream source of payout lines.
- [FlexPay Advance Payments](./flexpay-advance-payments.md) — the other source of wallet payment rows.
