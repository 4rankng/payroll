<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# disbursement — Payment Provider Adapters

## Purpose
Implements the payment disbursement provider interfaces for transferring funds to employees. Contains two provider adapters: **OnePay** (used in production) and **9Pay** (used in sandbox/development). Each provider implements a common interface for initiating transfers, handling webhooks (IPN), signing requests, and querying transfer status. The system is provider-agnostic at the service layer.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `onepay/` | OnePay provider — production payment disbursement adapter |
| `ninepay/` | 9Pay provider — sandbox/development payment disbursement adapter |

## For AI Agents

### Working In This Directory
- **Production uses OnePay** — never confuse with 9Pay for production issues
- 9Pay is for sandbox/development only
- Both providers implement a common `Provider` interface defined in the domain layer
- IPN (Instant Payment Notification) arrives asynchronously (~3s after batch completion)
- Tests must poll for `payment_status='paid'` rather than checking immediately
- IPN webhook IPs must be whitelisted for security (see `internal/transport/http/middleware/ip_whitelist.go`)

### Testing Requirements
- Each provider has comprehensive tests: `provider_test.go`, `signing_test.go`, `webhook_test.go`, `queue_test.go`
- OnePay tests were removed (production provider — tested in staging)
- 9Pay has stubs for sandbox testing (`stubs.go`)
- Webhook tests verify signature validation and payload parsing

### Common Patterns
```go
// Provider interface pattern
type Provider interface {
    InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error)
    HandleWebhook(ctx context.Context, payload []byte) (*WebhookResult, error)
    QueryStatus(ctx context.Context, refID string) (*StatusResult, error)
}

// Webhook IPN handling
// POST /api/v1/webhooks/disbursement/{provider}
// Validated by IP whitelist middleware + signature verification
```

## Dependencies

### Internal
- `internal/domain` — disbursement interfaces and types
- `internal/infra/observability` — logging
- `internal/pkg/retry` — retry logic for provider API calls

### External
- HTTP client for provider API calls
- Cryptographic signing for request/response verification

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
