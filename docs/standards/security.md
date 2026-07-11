# Security Standards

Security practices, threat models, and mitigations for the payroll system. This is a financial system handling hundreds of millions of VND — security is non-negotiable. See [Review Checklist](review-checklist.md) for the pre-merge security checklist.

## Authentication

### JWT + Casbin RBAC

- **Authentication:** JWT tokens extracted and validated in `transport/http/middleware/auth.go`. User role is set in the request context.
- **Authorization:** Casbin RBAC enforced in `transport/http/middleware/authorization.go`. Deny-override model. See [ADR-008](../decisions/ADR-008-casbin-rbac-authorization.md).
- **Token revocation:** `blacklisted_tokens` table + `invalid_before` timestamp (migration 086). Both logout and password reset revoke the token.

### Password Hashing

- **Argon2id** via `golang.org/x/crypto`. Never MD5, never SHA-256 for passwords.

### Login Protection

- **Self-hosted image CAPTCHA** after 3 failed login attempts (commit `6327168`). Uses `base64Captcha` with Redis storage.
- **Per-account login rate limiting** (commit `473a49e`). Redis-backed, returns Vietnamese message with actual `retry_after` seconds.
- **Per-IP rate limiting** via `middleware/rate_limit.go`.

### Google OAuth

- Google OAuth login skips OTP (commit `e5dd9f7`), with replay/issuer hardening.
- Residual risk documented (commit `63c72fb`). See [Lesson: Google OAuth no-OTP](../lessons/2026-07-04-google-oauth-no-otp-residual-risk.md).

## Authorization (IDOR Prevention)

**IDOR (Insecure Direct Object Reference)** was identified and fixed in a red-team exercise (commit `619f9cd`). See [Lesson: Security IDOR fix](../lessons/2026-07-04-security-idor-spray-token-revocation.md).

**Rule:** Every endpoint that takes a resource ID must validate that the requesting user owns or is authorized to access that resource. Never trust client-provided IDs without server-side ownership checks.

```go
// WRONG — no ownership check
func (h *Handler) GetTimesheet(c *gin.Context) {
    id := c.Param("id")
    timesheet := h.repo.FindByID(id) // Anyone can access any timesheet
}

// CORRECT — validate ownership
func (h *Handler) GetTimesheet(c *gin.Context) {
    userID := c.GetInt("user_id")
    role := c.GetString("role")
    id := c.Param("id")
    timesheet := h.repo.FindByIDForUser(id, userID, role) // Ownership enforced
}
```

## Webhook Security

- **IP whitelisting** — Payment provider webhooks (`/webhooks/disbursement/{provider}`) are protected by `ip_whitelist.go` middleware. Only known provider IPs are accepted.
- **Signature verification** — `VerifyAndParseWebhook()` in the `DisbursementProvider` interface validates provider signatures. See [ADR-003](../decisions/ADR-003-payment-provider-abstraction.md).

## Attendance Security

- **Geofencing** — Check-in/out validated against project geofence.
- **GPS anti-replay** — Commit `2ae0bc8` added GPS anti-replay for attendance. Prevents replaying old GPS coordinates.
- **Failed attempt tracking** — `attendance_failed_attempts` table (migration 079) logs GPS failures for audit.

## Input Validation

### Backend

- Domain validation in `internal/domain/` — entities validate their own invariants in constructors.
- Domain errors: `NewValidationError()` for invalid input.
- DTOs in `internal/app/dto/` define request/response shapes.

### Frontend

- **Zod** schema validation via `@hookform/resolvers/zod` with `react-hook-form`.
- TypeScript strict mode — no `any` types.

## Secrets Management

- All secrets come from `.env` files. See [Deployment Guide](../deployment-guide.md) → Environment Files.
- `backend/.env.example` and `frontend/.env.example` are templates with all required variables.
- Never default secrets in `getEnv()` functions.
- `.env` files are gitignored.

## Logging Security

- Use `slog` via `internal/infra/observability/logger.go`.
- **Never log:** JWT tokens, passwords, bank account numbers, payment provider API keys.
- Log rotation via `lumberjack.v2`.

## CI Security Scanning

The CI pipeline (`.github/workflows/ci-cd.yml`) includes a `security` job:

| Tool | Scope | Action |
|------|-------|--------|
| `gosec` | Backend Go code | SARIF report uploaded to GitHub Security tab |
| `npm audit --audit-level=high` | Frontend dependencies | Fails on high-severity vulnerabilities |

## Zero-Trust for External Repositories

When working with external repositories or documentation:
- **Review commands before execution** — Don't blindly run setup scripts.
- **Avoid granting unnecessary permissions** — Least privilege.
- **Be cautious with setup instructions from unknown repositories** — Indirect prompt injection can manipulate AI agents into executing harmful commands.

## Security Incident History

| Date | Commit | Issue | Fix |
|------|--------|-------|-----|
| 2026-07-04 | `619f9cd` | IDOR, spray, token-revocation gaps | Red-team fix: ownership validation, rate limiting, token blacklist |
| 2026-07-04 | `e5dd9f7` | Google OAuth replay/issuer weakness | Replay protection, issuer validation |
| 2026-07-04 | `2ae0bc8` | GPS anti-replay for attendance | Geofence + timestamp validation |
| 2026-07-04 | `6327168` | Brute-force login | Self-hosted CAPTCHA after 3 failures |
| 2026-07-04 | `473a49e` | Per-account login spray | Per-account rate limiting |
