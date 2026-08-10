# ADR-011: Zalo OTP Password Reset for Employees

- **Status:** Accepted
- **Date:** 2026-08-04
- **Related:** ADR for email password reset (plans/260724-2100-password-reset-email), `docs/standards/context-engineering.md`

## Context

Employees in this system frequently have a `users.mobile` on file but **no**
`users.email`. The existing self-service password reset
(`/auth/password-reset/request`) is email-only and silently cannot serve them —
they must contact an admin to reset their password, creating an operational
bottleneck.

The organization already runs a Zalo Official Account (OA) with ZNS (Zalo
Notification Service) template messaging approved for the sister recruitment
codebase `tuyennhanvien.vn`. ZNS delivers to the phone number's linked Zalo
account and is markedly cheaper than transactional SMS in Vietnam. A new
transactional template `OTP-ZNS-v2` (id `619684`, one `otp` parameter) was
registered in the payroll OA
("Ting Ting Software Solution" / app "TingTing Soft") for this purpose.

## Decision

Add a **self-service, mobile-channel password reset** for `employee`-role users
via Zalo ZNS OTP. The admin manages the OA connection (keys, OAuth, on/off
toggle) entirely through a settings UI — no env edits or redeploy for routine
operations.

### Architectural decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | New `app/services/zaloreset` package, separate from the email `passwordreset` | Different OTP medium (push template vs magic-link email), different lookup key (mobile vs email), different timing-equalization. Mixing them produces a god-service. |
| D2 | New `infra/cache/ZaloResetStore`, not reuse `OTPPendingStore` | The OTP store binds sessions to IP/UA + enforces per-account lockout — both wrong for an unauthenticated reset. A focused store is cheaper than parameterizing the OTP store and risking login 2FA. |
| D3 | ZNS provider implements a narrow `zalo.Sender` interface | Keeps the ZNS dependency behind an interface so the reset service is unit-testable with a fake. |
| D4 | Template `619684` (`OTP-ZNS-v2`) with one `otp` parameter | Current approved payroll OA template. The retired `617976` template is not used for new reset requests. |
| D5 | Credentials + connection state in the **existing generic `settings` table** | Two rows: `zalo.enabled` (bool) + `zalo.credentials` (JSON). Admin UI is authoritative after first boot; env vars are seed-only. Lets admin toggle without a redeploy. |
| D6 | `role = 'employee'` hard gate | Admin/partner reset stays on the audited email + 2FA path. Splitting channels per role keeps each role's blast-radius small. |
| D7 | No per-account failed-attempt lockout in v1 | The rate limiter (3/hour/mobile) + 10-min TTL + single-consume already cap brute force. Adding `OTPFailedAttempts` reuse would require schema changes for a non-login flow. Deferred. |
| D8 | OAuth v4 connect is an **admin-initiated flow with an SPA-mediated callback** | Admin clicks "Kết nối Zalo" → backend mints a `state` + returns the Zalo permission URL → browser redirects to Zalo → Zalo redirects to the **frontend** route (`/admin/settings?tab=zalo&code=&state=`) → the SPA extracts `code`+`state` and POSTs to `/api/v1/admin/zalo/oauth/callback` via axios (attaching the admin JWT). The SPA intermediary is required because the API uses header-based JWT auth, not cookies — a direct browser redirect to the API would arrive with no auth header and 401. The backend callback validates the single-use `state` (Redis GETDEL) + admin session (Authorize middleware) before exchanging the code. |

## Consequences

**Positive:**
- Employees can self-reset without admin intervention, reducing support load.
- The admin UI makes the Zalo connection fully operable without shell/SQL access.
- The DB-driven toggle means disabling the feature (e.g. during a Zalo outage) is instant.
- Reuses the existing `settings` table — no new migration.
- The `infra/zalo` Provider is stateless + DB-agnostic, making it trivially testable and reusable for future ZNS use cases (notifications, etc.).

**Negative:**
- Adds a runtime dependency on Zalo ZNS availability (mitigated: send failures are non-fatal; the user sees a generic "try again" message).
- ZNS charges 300 ₫/send via phone (mitigated: not-found mobiles do a dummy Redis write, 0 ₫; enumeration is cost-bounded).
- The `infra/zalo` Provider ↔ `zaloconnect.Service` construction cycle requires a two-step bootstrap wire (`SetProvider` after construction). Mildly unusual but well-documented.
- No per-account brute-force lockout in v1 (R-Z3). The rate limiter bounds guesses to ~3/hour/mobile, but a distributed attacker with many mobiles could attempt more. Revisit if abuse appears.

## Security contract

Inherited from the email reset's red-team review:
- **Anti-enumeration:** `/request` always returns 200 + a structurally identical session id for known and unknown mobiles. Not-found + disabled-toggle paths perform a dummy Redis write to equalize timing.
- **Single-use:** Consume uses a Lua compare-and-delete; a code can succeed at most once.
- **Wrong-code survival:** A wrong code does NOT consume the session (user may retry within the TTL). This differs from the email reset (which GETDELs the token regardless) because OTP codes are user-typed 6-digit values where a typo should not burn the session.
- **Atomic password update:** password + `tokens_invalid_before` commit in one DB transaction.
- **Role gate:** only `role='employee'` is eligible.
- **Hot toggle:** the admin toggle is read on every request; disabling takes effect immediately.
- **OAuth CSRF defense:** the connect flow uses a 256-bit single-use `state` (Redis GETDEL) + admin-session requirement on the callback.

## Residual risks

- **R-Z1:** Zalo refresh_tokens are single-use; a crash between refresh and persist forces a full admin re-OAuth. Mitigated by persisting `Update` before returning to the retry path.
- **R-Z3:** No per-account attempt lockout (see D7).
- **R-Z5:** Wrong-code session survival removes one anti-brute-force lever; the per-mobile rate limiter is the compensating control.
- **R-Z16:** The current `619684` template must remain approved in the OA console; sandbox delivery may return `-127`, which is treated as non-fatal.

## Verification

- `go test ./internal/infra/zalo/... -race -cover` → 83% coverage.
- `go test ./internal/app/services/zaloreset/... -race` → all service paths covered.
- `go test ./internal/app/services/zaloconnect/... -race` → 74.6% coverage.
- `go test ./internal/infra/cache/... -race -run ZaloReset` → 8 tests including concurrent-consume-only-one-wins.
- `go test ./internal/transport/http/middleware/... -race -run Zalo` → key-getter normalization + body-restore.
- `pnpm lint && pnpm build && pnpm test:run` → 331 frontend tests pass.
- `go build ./...` → clean.
