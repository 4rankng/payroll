# ADR-003: Payment Provider Abstraction

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system disburses salary payments via third-party payment providers. In production, it uses **OnePay**; in sandbox/development, it uses **9Pay**. The two providers have different APIs, webhook formats, and capabilities. The system must switch between them without code changes, and new providers may be added in the future.

Additionally, providers have different optional capabilities: OnePay supports status polling; 9Pay relies solely on IPN webhooks. OnePay supports account verification; 9Pay does not.

## Decision

Implement a **provider-agnostic disbursement interface** in `domain/ports/infrastructure/disbursement.go`:

```go
type DisbursementProvider interface {
    Name() string
    InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResult, error)
    VerifyAndParseWebhook(ctx context.Context, payload map[string]any) (*WebhookEvent, error)
}
```

Two implementations:
- **OnePay** (`internal/infra/disbursement/onepay/`) — used in **production**
- **9Pay** (`internal/infra/disbursement/ninepay/`) — used in **sandbox/development**

The active provider is selected at request time based on `Settings.disbursement_provider`.

### Optional Capability Interfaces

For provider-specific features, optional capability interfaces are defined and accessed via **type assertion**:

| Interface | Purpose | OnePay | 9Pay |
|-----------|---------|--------|------|
| `StatusPoller` | Poll transfer status | ✅ | ❌ (relies on IPN) |
| `AccountVerifier` | Pre-flight bank account check | ✅ | ❌ |
| `BalanceReporter` | Partner balance query | ✅ | ❌ |
| `ReportExporter` | Reconciliation CSV export | ✅ | ❌ |
| `ErrorTranslator` | Vietnamese error code translation | ✅ | ✅ |
| `TransferLimiter` | Per-transfer amount bounds | ✅ | ✅ |

The service checks for optional capabilities at runtime:
```go
if poller, ok := provider.(StatusPoller); ok {
    status, err := poller.PollStatus(ctx, transferID)
}
```

### Webhook Security

Webhook IPs are whitelisted via `ip_whitelist.go` middleware. IPN arrives asynchronously (~3s after batch completion).

## Consequences

**Positive:**
- Switching providers requires only a settings change — no code modification.
- New providers can be added by implementing the interface.
- Optional capabilities don't pollute the core interface.
- Provider-specific logic is isolated in its own package.

**Negative:**
- Type assertion pattern is less type-safe than a full capability registry.
- Testing requires mocking both providers.
- OnePay tests were removed (production provider — tested in staging only). 9Pay sandbox + mocks are used in CI.

## Alternatives Considered

1. **Single provider with environment branching** — Rejected. Would couple business logic to provider-specific APIs and make future provider changes expensive.
2. **Strategy pattern with all methods in one interface** — Rejected. Would force providers to implement methods they don't support (e.g., 9Pay implementing `PollStatus` as a no-op).
3. **Plugin system** — Rejected. Over-engineered for two providers. The interface + type assertion pattern is simpler and sufficient.
