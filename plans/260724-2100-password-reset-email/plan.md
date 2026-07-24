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
3. Backend looks up the user by email. If found **and** the user has a password (not a Google-only account), generate a **single-use, high-entropy token** (32 bytes, base64url), store its SHA-256 hash in Redis with a 30-minute TTL, and email a branded magic link: `https://tingting.vip/reset-password?token=...`.
4. User clicks the link → lands on `/reset-password`, which reads the token from the query string and calls `POST /api/v1/auth/password-reset/confirm` with `{ token, new_password }`.
5. Backend validates the token (exists, not expired, not already consumed), enforces password-strength rules, re-checks the user still exists, hashes the new password, **invalidates all existing sessions** (sets `tokens_invalid_before`), and deletes the token.
6. User is redirected to `/login` with a success toast.

**Design decision — magic link, not 6-digit code:** Password reset is the industry-standard use case for a clickable magic link. It is more secure than a 6-digit code (256-bit entropy vs. ~20 bits), works seamlessly cross-device (open email on phone, reset on desktop), and doesn't force the user to copy-paste. The codebase's existing 6-digit OTP infra is a *login* second-factor and is intentionally not reused here to keep the two flows' security semantics distinct.

**Scope & non-goals:**
- ✅ Users with an email (`admin`, `partner`, `adv_partner`, `employee` — anyone with `users.email` set).
- ✅ Works alongside the existing admin `/users/:id/reset-password` and self-service `/auth/change-password` flows.
- ❌ No SMS/OTP-based reset. Email-only.
- ❌ No "reset link sent" email when the email is unknown — the response is always generic success to prevent email enumeration (but the email is only actually sent to known accounts).
- ❌ No migration of existing tokens — greenfield.

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
| **Atomic consume** via Lua `GETDEL` | Eliminates the race where two concurrent confirm requests both read a valid token before either deletes it. |
| **Per-email rate limit** (3/hour) | Prevents an attacker from triggering mass emails against a known address (email-bomb / annoyance) or enumerating via timing. Per-email, not per-IP, so a rotating-IP attacker is still capped. |
| **Always 200 on request, regardless of email existence** | Standard anti-enumeration: the response body is identical whether the email exists or not. Email only actually dispatched for known accounts. |
| **Invalidate all sessions on reset** | Sets `users.tokens_invalid_before = now`, which the auth middleware already enforces. Mirrors `ChangePasswordAndBlacklistToken` semantics but broader (kills *all* JWTs, not just the current one). |
| **30-minute TTL** | Balances security window with email-delivery latency and user attention span. |
| **Google-only accounts excluded** | Users created via Google OIDC have no password. Resetting is meaningless — they'd get a password they didn't ask for. Skip if `user.Password == ""`. |

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
- `backend/internal/domain/event_factory_user.go` — add `"email_reset"` to the `method` enum doc/usage
- `backend/internal/app/bootstrap/services/init.go` — construct + inject `PasswordResetService`
- `backend/internal/app/bootstrap/container.go` — wire into handler
- `backend/internal/app/bootstrap/routes_auth.go` — add the two new public routes
- `backend/internal/transport/http/handlers/auth.go` — `RequestPasswordReset` + `ConfirmPasswordReset` handlers
- `backend/internal/transport/http/middleware/rate_limit.go` — add `CreatePasswordResetRateLimit`
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
- [ ] Requesting a reset for an unknown email returns the **same** success response as a known email (no enumeration).
- [ ] After reset, all previously-issued JWTs for that user are invalid (they must log in again with the new password).
- [ ] Password strength validation runs on the new password (reuses `passwordValidator.Validate`).
- [ ] Per-email rate limiting caps reset requests at 3/hour/email.
- [ ] A Google-only account (no password) silently receives no email but still gets a 200.
- [ ] Frontend pages are mobile-responsive, Vietnamese, DaisyUI-styled, and match the login page's look.
- [ ] `make api-test` passes including the new integration test.
- [ ] `cd backend && go test ./... -race` passes.

## Open Questions

None — design decisions resolved during research (magic link, 30-min TTL, Redis store, anti-enumeration).

## Dependencies

No cross-plan dependencies. This feature is self-contained. The existing email infrastructure (`ResendProvider`, `EmailDeliveryPort`, `branding.go`) and Redis client are reused as-is.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Token enumeration / brute force | High | 256-bit tokens (32 bytes base64url); per-token lookup is by hash, not value; 30-min TTL. |
| Email enumeration via request endpoint | Medium | Generic 200 response regardless of email existence; no timing differential (email send is async). |
| Account takeover via stolen token | High | Single-use (atomic GETDEL); 30-min TTL; session invalidation on reset forces re-login everywhere. |
| Rate-limit bypass to spam emails | Medium | Per-email rate limit (3/hr), not per-IP. |
| Existing tests break | Low | New routes are additive; no existing route signatures change. |
| Frontend build breaks | Low | New pages are lazy-loaded; no changes to existing route components. |
