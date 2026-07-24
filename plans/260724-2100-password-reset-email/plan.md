---
title: "Password Reset via Email (Magic Link)"
description: "Self-service password reset for users with an email on file. User requests a reset link from the login page, receives a branded email with a high-entropy token, and sets a new password on a dedicated reset page."
status: pending
priority: P2
branch: "main"
tags: ["auth", "security", "email", "frontend", "backend"]
blockedBy: []
blocks: []
created: "2026-07-24T13:38:04.027Z"
createdBy: "ck:plan"
source: skill
---

# Password Reset via Email (Magic Link)

## Overview

Add a **self-service password reset** flow for users who have an email address on record (`users.email IS NOT NULL`). This covers the common "I forgot my password" case without requiring admin intervention.

**Flow:**
1. User clicks **"Quên mật khẩu?"** (Forgot password?) on `/login`.
2. Enters their email → `POST /api/v1/auth/password-reset/request`.
3. Backend looks up the user by email. If found, generate a **single-use, high-entropy token** (32 bytes, base64url), store its SHA-256 hash in Redis with a 30-minute TTL, and email a branded magic link: `https://tingting.vip/reset-password?token=...`.
4. User clicks the link → lands on `/reset-password`, which reads the token from the query string (then **strips it from the URL** to prevent Referer/history leakage — Red Team H3) and calls `POST /api/v1/auth/password-reset/confirm` with `{ token, new_password }`.
5. Backend validates the token (exists, not expired, not already consumed), enforces password-strength rules, re-checks the user still exists, hashes the new password, and **atomically** updates the password + invalidates all existing sessions (sets `tokens_invalid_before`) in a **single DB transaction** (Red Team C1).
6. User is redirected to `/login` with a success toast.

> **Red Team C2 (corrected):** An earlier draft gated step 3 on "the user has a password (not a Google-only account)." This check is **dead code** — `users.password` is `NOT NULL` (migration `001:16`), and `LoginWithGoogle` (`auth_service.go:760`) only *reads* users by email, never creates them. No passwordless accounts exist. The gate was removed.

**Design decision — magic link, not 6-digit code:** Password reset is the industry-standard use case for a clickable magic link. It is more secure than a 6-digit code (256-bit entropy vs. ~20 bits), works seamlessly cross-device (open email on phone, reset on desktop), and doesn't force the user to copy-paste. The codebase's existing 6-digit OTP infra is a *login* second-factor and is intentionally not reused here to keep the two flows' security semantics distinct.

**Scope & non-goals:**
- ✅ Users with an email (`admin`, `partner`, `adv_partner`, `employee` — anyone with `users.email` set).
- ✅ Works alongside the existing admin `/users/:id/reset-password` and self-service `/auth/change-password` flows.
- ❌ No SMS/OTP-based reset. Email-only.
- ❌ No "reset link sent" email when the email is unknown — the response is always generic success to prevent email enumeration (but the email is only actually sent to known accounts). **Timing is equalized** (Red Team H2) so the known vs unknown paths are indistinguishable by wall-clock.
- ❌ No migration of existing tokens — greenfield.

**Pre-existing surfaces (Red Team Security-7):** The codebase already has `backend/internal/app/services/user/password_reset_job.go` (`PasswordResetJobManager`) and the route `/users/reset-first-time-login-password` — an **admin-initiated bulk first-time-login reset job**, unrelated to this self-service email flow. To avoid namespace confusion, the new field on the `Services` struct is named `EmailPasswordReset` (not `PasswordReset`, which would collide with `PasswordResetJobManager` at `init.go:60`), and the new package is `passwordreset` (distinct from the `user` package's job).

## Architecture

```
┌─────────────┐   POST /auth/password-reset/request      ┌──────────────┐
│  /login     │ ───────────────────────────────────────▶ │  AuthHandler │
│  (Forgot?)  │   { email }                              │              │
└─────────────┘                                          └──────┬───────┘
                                                                │ PasswordResetService.RequestReset
                                                                ▼
                         ┌──────────────────────────────────────────┐
                         │ 1. UserRepository.GetByEmail              │
                         │ 2. (if found) TokenStore.Create(userID)   │
                         │ 3. (if found) EmailSender.Send(magic link)│
                         │ 4. ALWAYS return generic success          │
                         └──────────────────────────────────────────┘
                                                  │
                                                  ▼  (async goroutine, 15s ctx)
                                         ┌─────────────────┐
                                         │ Resend Provider │ ──▶ inbox
                                         └─────────────────┘

┌───────────────┐  POST /auth/password-reset/confirm       ┌──────────────────┐
│ /reset-password│ ──────────────────────────────────────▶ │  AuthHandler     │
│  ?token=...   │  { token, new_password }                │                  │
└───────────────┘                                          └──────┬───────────┘
                                                                  │ PasswordResetService.ConfirmReset
                                                                  ▼
                         ┌────────────────────────────────────────────────┐
                         │ 1. TokenStore.Consume(token) → userID          │
                         │    (SHA-256 lookup, atomic delete, single-use) │
                         │ 2. UserRepository.GetByID(userID)              │
                         │ 3. ValidatePassword(newPassword)               │
                         │ 4. Hash + Update password                      │
                         │ 5. UpdateTokensInvalidBefore(now)  ◀─ kill JWTs│
                         │ 6. Publish PasswordChangedEvent("email_reset") │
                         └────────────────────────────────────────────────┘
```

### Key design choices

| Decision | Rationale |
|----------|-----------|
| **Redis-backed token store** (not MySQL) | Tokens are ephemeral (30-min TTL), single-use, and high-volume-potential. Redis gives automatic expiry, atomic consume-and-delete via Lua, and avoids polluting the relational schema. Mirrors the existing `OTPPendingStore` pattern. |
| **Store SHA-256 hash of token, not plaintext** | If Redis is dumped, tokens are not directly usable. Same hygiene as `OTPPendingStore` storing `code_hash`. |
| **Atomic consume** via Lua `GETDEL` | Eliminates the race where two concurrent confirm requests both read a valid token before either deletes it. (Redis 7 confirmed; miniredis v2.38.0 supports GETDEL for tests.) |
| **Per-email rate limit** (3/hour), **unicode-normalized key** | Prevents an attacker from triggering mass emails against a known address (email-bomb / annoyance) or enumerating via timing. Per-email, not per-IP. **Red Team H1:** the email is Unicode case-folded + NFKC-normalized + zero-width-stripped before keying, so `Alice@`, `Álice@`, `alice@\u200b` collapse to the same bucket (matching the DB's `utf8mb4_unicode_ci`). |
| **Always 200 + timing equalization** | Standard anti-enumeration: the response body is identical whether the email exists or not. **Red Team H2:** the not-found path performs a dummy `tokenStore.Create` so both paths have identical Redis-RTT + hashing profiles, defeating timing oracles. |
| **Transactional password + session-kill** (Red Team C1) | The password hash and `tokens_invalid_before` are written in a **single DB transaction** via `UpdatePasswordAndInvalidateSessions` — NOT two separate calls. A partial commit (password changed but sessions alive) would leave stolen JWTs valid for up to 14 days. |
| **Honest error mapping** (Red Team H5) | A Redis outage during `Consume` returns a distinct `ErrPasswordResetStoreUnavailable` → HTTP 500 ("try again"), NOT a 401 "invalid token." The user isn't lied to about their valid link being expired. |
| **Functional feature flag** (Red Team H6) | Routes are registered only when `cfg.PasswordReset.Enabled`. When false, endpoints 404 (honest) instead of returning a misleading "email coming" 200. |
| **Goroutine panic recovery** (Red Team H4) | The async email send is wrapped in `defer recover()` so a panic in `Send` logs instead of crashing the whole backend. |
| **30-minute TTL** | Balances security window with email-delivery latency and user attention span. |
| **New trust-boundary behavior** (Red Team M3, corrected) | Session-wide invalidation on a public endpoint is NEW — the existing `ChangePassword` (`user/auth.go:40-90`) only blacklists the current JTI and does NOT call `UpdateTokensInvalidBefore`. `tokens_invalid_before` is enforced inside `AuthService.ValidateToken` (`auth_service.go:411`). The C1 transaction is therefore essential — there is no existing safety net. |

## Phases

| Phase | Name | Status | Summary |
|-------|------|--------|---------|
| 1 | [Backend: Token Store & Domain](./phase-01-backend-token-store-domain.md) | Pending | `PasswordResetTokenStore` (Redis), DTOs, config, constants, domain event variant |
| 2 | [Backend: Service & Handler & Wiring](./phase-02-backend-service-handler-wiring.md) | Pending | `PasswordResetService`, handler methods, routes, DI wiring, email template |
| 3 | [Backend: Tests](./phase-03-backend-tests.md) | Pending | Unit tests for store + service; integration test for the full flow |
| 4 | [Frontend: Request & Reset Pages](./phase-04-frontend-request-reset-pages.md) | Pending | `ForgotPassword.tsx` + `ResetPassword.tsx` pages, DaisyUI-styled, Vietnamese |
| 5 | [Frontend: Hooks & Routing & Tests](./phase-05-frontend-hooks-routing-tests.md) | Pending | `usePasswordReset` hooks, routes in `App.tsx`, component tests |
| 6 | [Docs & Migration](./phase-06-docs-migration.md) | Pending | Update `api.md`, `system-architecture.md`; no DB migration needed (Redis-only) |

## Related Code Files

### Create
- `backend/internal/infra/cache/password_reset_token_store.go` — Redis token store
- `backend/internal/app/services/passwordreset/service.go` — orchestration service
- `backend/internal/app/services/passwordreset/template.go` — branded reset email HTML
- `backend/internal/app/services/passwordreset/service_test.go` — unit tests
- `backend/internal/infra/cache/password_reset_token_store_test.go` — store tests
- `backend/tests/integration/auth_password_reset_test.go` — E2E flow test
- `frontend/src/pages/ForgotPassword.tsx` — request page
- `frontend/src/pages/ResetPassword.tsx` — confirm page
- `frontend/src/hooks/api/usePasswordReset.ts` — TanStack Query hooks
- `frontend/src/pages/__tests__/ForgotPassword.test.tsx` — component test
- `frontend/src/pages/__tests__/ResetPassword.test.tsx` — component test

### Modify
- `backend/internal/app/dto/user.go` — `PasswordResetRequestDTO`, `PasswordResetConfirmDTO`
- `backend/internal/config/config.go` — `PasswordResetConfig` struct + env parsing
- `backend/internal/constants/messages.go` — Vietnamese success/error messages
- `backend/internal/domain/user.go` — add `UpdatePasswordAndInvalidateSessions` to `UserRepository` interface (Red Team C1)
- `backend/internal/infra/persistence/user_repository.go` — implement `UpdatePasswordAndInvalidateSessions` (transactional, column-scoped)
- `backend/internal/domain/email.go` — add `EmailKindPasswordReset` to the enum (Red Team Scope-3)
- `backend/internal/domain/event_factory_user.go` — `NewPasswordChangedEvent` called with `method="email_reset"` + `actorUserID=user.ID` (Red Team Security-5)
- `backend/internal/app/services/user/password.go` — add ONE delegator `HashNewPassword` (Red Team H8 — `ValidatePassword` is already exported)
- `backend/internal/app/bootstrap/services/init.go` — construct + inject `EmailPasswordReset` service (Red Team C4 reuse otpEmailSender; Red Team M2 field name)
- `backend/internal/app/bootstrap/container.go` — wire into handler; expose `PasswordReset.Enabled` for route gate
- `backend/internal/app/bootstrap/routes_auth.go` — add the two new public routes behind `if cfg.PasswordReset.Enabled` (Red Team H6)
- `backend/internal/transport/http/handlers/auth.go` — `RequestPasswordReset` + `ConfirmPasswordReset` handlers
- `backend/internal/transport/http/middleware/rate_limit.go` — add `CreatePasswordResetRateLimit` + unicode-normalizing `passwordResetEmailKeyGetter` (Red Team H1, M1)
- `frontend/src/services/api/auth.service.ts` — `requestPasswordReset`, `confirmPasswordReset`
- `frontend/src/types/api/auth.types.ts` — request/response types
- `frontend/src/App.tsx` — add `/forgot-password` and `/reset-password` routes
- `frontend/src/pages/Login.tsx` — add "Quên mật khẩu?" link

### Delete
- _(none)_

## Acceptance Criteria

- [ ] A user with an email can request a reset link from `/forgot-password` and receive a branded email within ~30 seconds.
- [ ] The reset link works exactly once; a second use is rejected with a clear Vietnamese error.
- [ ] The reset link expires after 30 minutes and the user gets a clear "link expired, request a new one" message.
- [ ] Requesting a reset for an unknown email returns the **same** success response as a known email (no enumeration), AND the response times are equalized so a timing oracle can't distinguish them (Red Team H2).
- [ ] Password + session invalidation commit **atomically** in one DB transaction (Red Team C1) — verified by a unit test asserting `UpdatePasswordAndInvalidateSessions` is called once (not separate `Update` + `UpdateTokensInvalidBefore`).
- [ ] After reset, all previously-issued JWTs for that user are invalid (they must log in again with the new password).
- [ ] Password strength validation runs on the new password (reuses `UserService.ValidatePassword` — Red Team H8, already exported).
- [ ] Per-email rate limiting caps reset requests at 3/hour/email, with Unicode-normalized keys so `Alice@`/`Álice@`/`alice@\u200b` share a bucket (Red Team H1).
- [ ] A Redis outage during confirm returns HTTP 500 ("try again"), NOT a 401 "invalid token" (Red Team H5).
- [ ] `PASSWORD_RESET_ENABLE=false` makes the endpoints 404 (Red Team H6).
- [ ] A panic in the async email sender is recovered and logged, not crashed (Red Team H4).
- [ ] The reset token is stripped from the URL on page mount (Red Team H3).
- [ ] The audit event for a reset carries `actor_user_id = <the user>` (Red Team Security-5).
- [ ] Frontend pages are mobile-responsive, Vietnamese, DaisyUI-styled, and match the login page's look.
- [ ] `make api-test` passes including the new integration test.
- [ ] `cd backend && go test ./... -race` passes.

## Open Questions

- ~~Naming collision with existing `PasswordResetJobManager`?~~ **Resolved (Red Team Security-7 / M2):** new field is `EmailPasswordReset`, new package is `passwordreset`. Documented in Scope.
- ~~How does the integration test retrieve the emailed token cross-process?~~ **Resolved (Red Team C3):** two-layer strategy — unit tests use a capturing mock for the single-use/transactional guarantees; integration test covers the no-token contracts (anti-enumeration, rate limit, 404-on-disabled) and uses a dev-only `GET /dev/sandbox-emails` endpoint for the happy path (or falls back to an in-process `httptest` test). See Phase 3.

No other open questions — remaining design decisions resolved during research + red-team review.

## Dependencies

No cross-plan dependencies. This feature is self-contained. The existing email infrastructure (`ResendProvider`, `EmailDeliveryPort`, `branding.go`) and Redis client are reused as-is.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Token enumeration / brute force | High | 256-bit tokens (32 bytes base64url); per-token lookup is by hash, not value; 30-min TTL. |
| Email enumeration via request endpoint | High | Generic 200 + **timing equalization** (dummy `tokenStore.Create` on not-found path — Red Team H2). Email send is async + recovered (Red Team H4). |
| Email enumeration via rate-limit bypass | High | Per-email rate limit (3/hr) with **Unicode-normalized keys** (Red Team H1) so case/accent/zero-width variants share a bucket. |
| Account takeover via stolen token (pre-click) | Medium | Single-use (atomic GETDEL); 30-min TTL; session invalidation on reset. **Red Team H3:** token stripped from URL on mount + `Referrer-Policy: no-referrer`. Residual risk (proxy logs pre-strip, inbox malware) documented and accepted. |
| Partial-failure: password changed but sessions alive | Critical | **Red Team C1:** transactional `UpdatePasswordAndInvalidateSessions` — both writes commit atomically or neither does. |
| Redis outage lies to user ("link expired") | High | **Red Team H5:** distinct `ErrPasswordResetStoreUnavailable` → 500, not 401. |
| Feature flag is cosmetic (can't actually disable) | High | **Red Team H6:** routes register only when `cfg.PasswordReset.Enabled`. |
| Goroutine panic crashes process | High | **Red Team H4:** `defer recover()` on every fire-and-forget email send. |
| Phantom `GET /auth/password-strength` blocks frontend | Medium | **Red Team H7:** use the existing **client-side** `PasswordStrengthIndicator`; no backend call. |
| Existing tests break | Low | New routes are additive; no existing route signatures change. |
| Frontend build breaks | Low | New pages are lazy-loaded; no changes to existing route components. |

## Red Team Review

### Session — 2026-07-24
**Reviewers:** Security Adversary (Fact Checker), Failure Mode Analyst (Flow Tracer), Assumption Destroyer (Scope Auditor), Scope & Complexity Critic (Contract Verifier) — Full tier (6 phases).
**Findings:** 15 (4 Critical, 8 High, 3 Medium) — 13 accepted, 2 rejected.
**Severity breakdown:** 4 Critical, 8 High, 3 Medium.

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| C1 | No DB transaction in `ConfirmReset` — password update + session-kill are independent writes; partial failure leaves sessions alive for up to 14 days | Critical | Accept | Phase 1, 2, plan.md |
| C2 | `user.Password == ""` Google-only check is dead code — `users.password` is NOT NULL; `LoginWithGoogle` only reads, never creates | Critical | Accept | Phase 2, plan.md |
| C3 | Integration test cannot read emailed token cross-process — `SandboxProvider.LastEmail()` unreachable across process boundary | Critical | Accept | Phase 3, plan.md |
| C4 | Email sender gating vague — "same as OTP" references `cfg.OTP.Enabled` (default false); plan must reuse existing `otpEmailSender` var | Critical | Accept | Phase 2 |
| H1 | Rate-limit bypass via email case/unicode — `strings.ToLower` is ASCII-only but DB is `utf8mb4_unicode_ci` | High | Accept | Phase 2 |
| H2 | Timing-based email enumeration — known path does Redis SET, unknown path returns immediately | High | Accept | Phase 2 |
| H3 | Token in URL leaks via Referer/header/history — `replace()` only on success; `Referrer-Policy` is `strict-origin-when-cross-origin` not `no-referrer` | High | Accept | Phase 4, plan.md |
| H4 | Goroutine panic crashes process — no `defer recover()` in async email send | High | Accept | Phase 2 |
| H5 | Redis outage maps to "invalid token" — `Consume` conflates `redis.Nil` with connection errors | High | Accept | Phase 1, 2 |
| H6 | `PASSWORD_RESET_ENABLE=false` is cosmetic — routes register unconditionally; `RequestReset` returns nil 200 not the disabled message | High | Accept | Phase 2, plan.md |
| H7 | Phantom `GET /auth/password-strength` — no such route; strength indicator is client-side | High | Accept | Phase 4 |
| H8 | Gold-plating: 3 ports + 2 delegators — `ValidatePassword` already exported; `HashPassword` is a free function | High | Accept | Phase 2 |
| M1 | Body-restore footgun in `passwordResetEmailKeyGetter` — left as one-line comment but is 25+ non-obvious lines | Medium | Accept | Phase 2 |
| M2 | `passwordreset` package + `PasswordReset` field name collides with existing `PasswordResetJobManager` | Medium | Accept | Phase 2, plan.md |
| M3 | "Mirrors `ChangePasswordAndBlacklistToken`" claim is false — existing `ChangePassword` does NOT call `UpdateTokensInvalidBefore`; `tokens_invalid_before` enforced in `ValidateToken` not middleware | Medium | Accept | Phase 2, plan.md |
| — | `Metadata["purpose"]` has zero consumers (speculative observability) | Medium | Reject | — |
| — | 6 phases is over-planned | Medium | Reject | — |

**Noted (addressed by other fixes, not separate findings):** `Update(user)` via `Save` writes all columns → lost-update (addressed by C1's column-scoped `UpdatePasswordAndInvalidateSessions`); `PasswordChangedEvent` with `actorUserID=0` un-attributable (addressed: set to `user.ID`); CSRF surface on public POSTs (noted: add Origin/Referer check during implementation); ConfirmReset doesn't re-verify email binding if changed mid-flight (accepted risk, documented); base64url encoding nit (pinned `RawURLEncoding`).

### Whole-Plan Consistency Sweep
- **Files reread:** plan.md, phase-01 through phase-06 (all 7 files).
- **Decision deltas checked:** 13 (C1-C4, H1-H8, M1-M3).
- **Reconciled stale references:** Google-only check removed from plan.md Flow + Scope + Key design choices + Acceptance criteria; "mirrors ChangePassword" reworded in plan.md + Phase 2; phantom `/auth/password-strength` removed from Phase 4; `PasswordReset` field renamed to `EmailPasswordReset` in plan.md + Phase 2; test-token-retrieval strategy rewritten in Phase 3 + plan.md Open Questions.
- **Unresolved contradictions:** 0.

**Reports:** `/Users/dev/Documents/projects/payroll/plans/260724-2100-password-reset-email/reports/` (assumption-destroyer report; others returned inline).
