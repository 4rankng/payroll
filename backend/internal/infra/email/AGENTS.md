<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# email — Email Sending Service

## Purpose
Implements email delivery with a provider abstraction. The `ResendProvider` sends emails via the Resend API in production. The `SandboxProvider` logs emails to stdout in development/testing environments. Both implement a common email sender interface.

## Key Files
| File | Description |
|------|-------------|
| `resend_provider.go` | Resend email provider — production email sending via Resend API (2.1K) |
| `sandbox_provider.go` | Sandbox email provider — logs emails instead of sending (dev/test) (2.4K) |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- Provider selection happens in `bootstrap/services/init.go` based on environment
- Email service scope is **email only** — do not add non-email concerns here
- Email templates and content are defined in `internal/domain/email.go`
- The email service is a thin adapter — business logic for when/what to email lives in application services

### Testing Requirements
- Sandbox provider is automatically used in test/dev environments
- Integration tests verify email flow without actually sending

### Common Patterns
```go
// Email provider interface
type EmailProvider interface {
    Send(ctx context.Context, to, subject, htmlBody string) error
}

// Provider selection in bootstrap
if cfg.IsProduction() {
    emailProvider = email.NewResendProvider(cfg.ResendAPIKey)
} else {
    emailProvider = email.NewSandboxProvider()
}
```

## Dependencies

### Internal
- `internal/domain` — email entity and template types
- `internal/config` — API keys and environment detection

### External
- `resend/resend-go` — Resend email API client

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
