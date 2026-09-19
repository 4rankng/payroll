---
type: integration
title: Payment Providers (OnePay / 9Pay)
description: Provider-agnostic abstraction, OnePay in production, 9Pay in sandbox, IPN contract and signature verification, status inquiry, balance reporting, and the registry that selects the active provider at boot.
tags: [integration, onepay, ninepay, ipn, webhook, provider-abstraction]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-2afe96c9e3e7f78c56afa2bd
    resource: repo://backend/internal/app/services/disbursement/registry.go
  - id: openwiki-source-49d8b6aea4e63ed7b3039ca1
    resource: repo://backend/internal/infra/disbursement/AGENTS.md
  - id: openwiki-source-0e339d0fbfd848d3c007e835
    resource: repo://backend/internal/infra/disbursement/onepay/provider.go
  - id: openwiki-source-49857e2784126dfe96ada34e
    resource: repo://docs/decisions/ADR-003-payment-provider-abstraction.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Integration: Payment Providers (OnePay / 9Pay)

The system is provider-agnostic per ADR-003. In production it routes all disbursements through OnePay; in sandbox/dev it uses 9Pay. The application layer never imports a concrete provider package — it talks to the active provider through `disbursement.Registry`. The provider abstraction is the runtime seam that lets the system swap implementations via config.

## Production invariant

**OnePay in production, 9Pay in sandbox/dev.** Both adapters live under `internal/infra/disbursement/`. The registry selects the active one at boot via env vars (e.g. `ENABLE_NINEPAY`); bootstrap registers a provider only when its master switch is on AND its credentials are present. `Registry.Active(ctx)` returns `ErrNoActiveProvider` when nothing is registered — the natural "disbursements are off" state.

The system never accidentally calls 9Pay in production because:

- The 9Pay env vars are not set in the production `.env`.
- The 9Pay registration branch in `bootstrap/` is gated on `ENABLE_NINEPAY=true`.
- The OnePay registration branch is gated on its own credentials.
- The `Registry` enforces a single active provider; both cannot be active simultaneously in practice.

## Port — `domain/ports/infrastructure/disbursement.go`

The application layer talks to a small set of port interfaces:

```
DisbursementProvider     Top-level: Initiate, StatusInquiry, Cancel, IPNVerification
AccountVerifier          Verify a recipient's bank account before disbursement
BalanceReporter          Read provider-side wallet balance
StatusPoller             Periodic status check
TransferLimiter          Per-transfer amount bounds (for batch validation)
ErrorTranslator          Provider-specific error code → Vietnamese user-facing message
```

The OnePay provider (`internal/infra/disbursement/onepay/provider.go`) implements all six via compile-time assertions:

```go
var (
    _ infrastructure.DisbursementProvider = (*Provider)(nil)
    _ infrastructure.AccountVerifier      = (*Provider)(nil)
    _ infrastructure.BalanceReporter      = (*Provider)(nil)
    _ infrastructure.ErrorTranslator      = (*Provider)(nil)
    _ infrastructure.StatusPoller         = (*Provider)(nil)
    _ infrastructure.TransferLimiter      = (*Provider)(nil)
)
```

The 9Pay adapter implements the same set.

## Wire-state mapping

The OnePay provider maps wire states to abstract `TransferStatus` and then to FSM triggers:

```
Wire state   → TransferStatus   → FSM trigger
created      → Pending          → (no trigger)
pending      → Pending          → (no trigger)
approved     → Success          → ipn_completed
failed       → Failed           → ipn_failed
reverted     → Reversed         → ipn_failed + flag
```

The wallet state machine (`internal/domain/transactions/state_machine.go`) consumes the triggers; the provider layer does not know about the FSM. This keeps the abstraction clean.

## Registry and selector

`internal/app/services/disbursement/registry.go`:

- `Registry.Register(p)` — adds a provider under `p.Name()`. Bootstrap-only; not concurrency-safe. Duplicate names panic.
- `Registry.Active(ctx)` — returns the registered provider, consulting the optional `Selector` first (for future use), then falling back to first-registered. Returns `ErrNoActiveProvider` when nothing is registered.
- `Registry.ByName(name)` — looks up a specific provider regardless of which one is active. Used by per-provider webhook routes.
- `Registry.ActiveTransferLimits(ctx)` — convenience for batch validation.
- `Registry.GetProviderBalance(ctx)` — asserts `BalanceReporter` and returns provider balance. Returns `ErrBalanceNotSupported` when the active provider doesn't implement it.
- `Registry.TranslateError(name, code)` and `TranslateActiveError(code)` — map provider error codes to Vietnamese messages via the `ErrorTranslator` interface, falling through to a passthrough when the provider lacks the capability.

The pre-IPN service layer uses these helpers to avoid importing concrete provider packages just to translate codes.

## Per-provider adapter shape

Each adapter (`onepay/`, `ninepay/`) carries:

| File | Purpose |
|------|---------|
| `client.go` | HTTP client (provider-specific transport, signing headers) |
| `provider.go` | Implements `domain.DisbursementProvider` |
| `signing.go` | Request signing (RSA / HMAC) |
| `types.go` | Provider-specific request/response types |
| `error_codes.go` | Provider-specific error catalog |
| `webhook.go` | IPN signature verification + payload parsing |
| `queue.go` | In-flight request queue with retries |
| `stubs.go` (9Pay only) | Sandbox stubs for offline tests |
| `*_test.go` | Coverage |

OnePay provider tests were removed (production provider — tested in staging). 9Pay keeps comprehensive tests against the sandbox mock.

## IPN contract

OnePay and 9Pay POST an IPN (Instant Payment Notification) webhook to `/api/v1/webhooks/disbursement/{provider}` typically ~3s after a batch settles. The endpoint:

- Is mounted with `IP whitelist` middleware (`middleware/ip_whitelist.go`) so only the provider's IPs can post.
- Verifies the request signature using the per-provider `webhook.go` implementation.
- Enqueues an `ipn:process` asynq task (`backend/internal/app/workers/ipn_process_worker.go`).
- Returns 200 quickly.

The `ipn:process` worker:

1. Looks up the wallet payment by the provider's invoice number.
2. Verifies the IPN amount matches `wallet_payment.RequestedAmount` (`ErrIPNAmountMismatch` — a forgery signal; asynq must not retry).
3. Translates the wire state to the matching FSM trigger (`ipn_completed`, `ipn_failed`, `ipn_reversed`).
4. Drives the state machine; the post-commit handler writes ledger entries and emits settlement events.

The wire state → FSM trigger mapping is the same table above; the registry's `ErrorTranslator` translates any non-success code to a Vietnamese user-facing message.

## Status inquiry and poller

`status_inquiry_worker.go` queries a single payment by ID; `disbursement_poller_worker.go` periodically checks rows stuck between `authorised` and the next IPN. Both use the `StatusPoller` capability. They are the safety net for missed webhooks (provider outage, network partition, transient 5xx). When a poll returns a terminal state, the worker feeds the matching trigger into the FSM.

## Bulk-transfer execution

The bulk-transfer pipeline (`features/salary-disbursement.md`) calls `Registry.Active(ctx)` for each row, then `DisbursementProvider.InitiateTransfer(ctx, req)`. The row's wallet_payment is updated with the returned invoice number and provider-side status.

The `TransferLimiter.Limits()` capability is consulted by the bulk pipeline to enforce per-transfer amount bounds before enqueueing.

## Error sentinel contract

The registry exposes sentinel errors that the application layer maps to terminal failures:

- `ErrDuplicatePaymentInProgress` — a pending or authorised wallet_payment already exists for the same recipient. Prevents double-disbursement when auto-poller and manual admin flows target the same employee simultaneously.
- `ErrIPNAmountMismatch` — IPN amount does not match `RequestedAmount`. Terminal; asynq must not retry (a forgery won't self-heal).
- `ErrFeeResolution` — fee schedule could not be resolved in fail-closed mode (`DISBURSEMENT_FEE_FAIL_OPEN=false`) and the provider is not on the zero-fee allowlist. Terminal; the worker surfaces it as `SkipRetry`.
- `ErrNoActiveProvider` — no provider is registered (disbursements are off).
- `ErrBalanceNotSupported` — provider does not implement `BalanceReporter`.

These are the contract that lets the application layer decide retry vs terminal without knowing the provider.

## Local development

Both providers have local mocks for sandbox testing:

- 9Pay mock at `localhost:9001` (dev compose).
- OnePay mock at `localhost:9002` (dev compose).

Tests that poll for `payment_status='paid'` rather than checking immediately because IPN arrives ~3s after a batch completes in real life.

## References

- `docs/onepay/` — OnePay-specific contract docs.
- `docs/decisions/ADR-003-payment-provider-abstraction.md` — the rationale.
- `docs/deployment-guide.md` — env wiring.
- Wallet state machine — `features/wallet-ledger.md`.
- Bulk-transfer pipeline — `features/salary-disbursement.md`.
- asynq runtime — `integrations/background-jobs.md`.
