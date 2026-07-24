---
phase: 3
title: "Backend: Tests"
status: pending
priority: P2
dependencies: [2]
---

# Phase 3: Backend: Tests

## Overview

Unit tests for the token store (Redis-backed) and the reset service (mocked deps), plus an integration test exercising the full HTTP flow against the live backend. Tests are written **after** the implementation compiles (Phase 2 done) so they exercise real code, not speculation.

## Requirements

- **Functional:** Verify token create/consume atomicity, single-use enforcement, TTL expiry, the anti-enumeration behavior of `RequestReset`, the full happy path of `ConfirmReset`, and the HTTP-level 200-for-unknown-email contract.
- **Non-functional:** Unit tests use `miniredis` (already a dependency? — check; if not, use a real Redis via testcontainers or skip the TTL test). Integration test follows the existing `backend/tests/integration` pattern (see `auth_otp_test.go` for the template).

## Architecture

### Unit test layout

- `password_reset_token_store_test.go` — uses `miniredis` (`github.com/alicebob/miniredis/v2`) if already in `go.mod`; otherwise spin a real Redis in the integration test only and keep unit tests on the atomicity paths that `miniredis` supports. **Check `go.mod` first** — if miniredis is present (it is commonly used with go-redis projects), prefer it; if not, add it.
- `service_test.go` — mocks `UserRepository`, `EmailSender`, `PasswordHasher`, `PasswordValidator` via table-driven tests.

### Integration test layout

`backend/tests/integration/auth_password_reset_test.go` — mirrors the structure of the existing OTP integration test (look at `auth_otp_test.go` / `auth_login_test.go` in the same dir). It hits the live API:

1. Seed a user with a known email + password.
2. `POST /auth/password-reset/request {email}` → assert 200 + generic message.
3. (Email is sandboxed in dev, so intercept the token: either read it from the sandbox provider's captured messages, OR add a test-only hook / env var that logs the token. **Simplest:** in dev/sandbox mode, the `SandboxProvider` already logs the message — grep the server logs, or extend `SandboxProvider` to store last-sent message in memory for tests. Confirm by reading `sandbox_provider.go`.)
4. `POST /auth/password-reset/confirm {token, new_password}` → assert 200.
5. `POST /auth/login {username, new_password}` → assert 200 (new password works).
6. `POST /auth/login {username, old_password}` → assert 401 (old password dead).
7. `POST /auth/password-reset/confirm {token, new_password2}` → assert 401 (token single-use).
8. Anti-enumeration: `POST /auth/password-reset/request {unknown@email.com}` → assert **same** 200 + message.

## Related Code Files

- **Create:** `backend/internal/infra/cache/password_reset_token_store_test.go`
- **Create:** `backend/internal/app/services/passwordreset/service_test.go`
- **Create:** `backend/tests/integration/auth_password_reset_test.go`
- **Possibly modify:** `backend/internal/infra/email/sandbox_provider.go` — if it doesn't already expose the last-sent message for test retrieval, add a thread-safe `LastSent() *domain.EmailMessage` accessor (read-only, dev-only).

## Implementation Steps

1. **Confirm miniredis availability:** `grep miniredis backend/go.mod`. If present, use it for `password_reset_token_store_test.go`. If absent, either add it (`go get github.com/alicebob/miniredis/v2`) or test against the live Redis started by `make db`.

2. **Write `password_reset_token_store_test.go`:**
   - `TestCreate_ReturnsHighEntropyToken`: token is ~43 chars, base64url-charset, two `Create` calls produce different tokens.
   - `TestConsume_HappyPath`: `Create` then `Consume` returns the same `userID`.
   - `TestConsume_SingleUse`: `Create` once, `Consume` twice → second returns `ErrPasswordResetTokenNotFound`.
   - `TestConsume_Concurrent`: `Create` once, fire 10 goroutines calling `Consume` → exactly one succeeds, nine get the error. (This is the critical race test — run with `-race`.)
   - `TestConsume_UnknownToken`: `Consume("garbage")` → `ErrPasswordResetTokenNotFound`.
   - `TestConsume_ExpiredToken`: (miniredis) `Create`, advance the miniredis clock past TTL, `Consume` → error.

3. **Write `service_test.go`** with table-driven cases:
   - `RequestReset_KnownUser_SendsEmail`: mocks record the call; assert `EmailSender.Send` invoked once with a message containing the reset URL.
   - `RequestReset_UnknownEmail_ReturnsNilNoEmail`: assert `EmailSender.Send` never called, no error returned.
   - `RequestReset_GoogleOnlyUser_ReturnsNilNoEmail`: user with `Password == ""` → no email.
   - `ConfirmReset_HappyPath`: token valid → password hashed + updated, `UpdateTokensInvalidBefore` called with ~now, event published.
   - `ConfirmReset_InvalidToken`: `ErrPasswordResetTokenNotFound` → `domain.UnauthorizedError`.
   - `ConfirmReset_WeakPassword`: validator returns error → `domain.ValidationError`, password NOT updated.
   - `ConfirmReset_DeletedUser`: token valid but `GetByID` not-found → `UnauthorizedError`, token already consumed.
   - Use `testify/mock` or hand-rolled mocks — match whatever the existing `otp_service_test.go` uses (read it first and mirror the style).

4. **Check `sandbox_provider.go`** for a test-retrievable last-message accessor. If absent, add:
   ```go
   func (p *SandboxProvider) LastSent() *domain.EmailMessage {
       p.mu.Lock(); defer p.mu.Unlock()
       return p.last
   }
   ```
   And store `p.last = msg` in `Send`. This is dev-only code; in production the Resend provider is used.

5. **Write `auth_password_reset_test.go`** following the 8-step flow above. Reuse the integration test helper that creates a user + logs in (find it in `tests/integration` — likely a `seedUser` or similar helper).

6. **Run everything:**
   - `cd backend && go test ./internal/infra/cache/... -v -race -run PasswordReset`
   - `cd backend && go test ./internal/app/services/passwordreset/... -v -race`
   - `make api-test` (after the backend is running via `make dev` / `make db`)

## Success Criteria

- [ ] Token store tests pass under `-race`, including the concurrent-consume test proving single-use.
- [ ] Service unit tests cover: known user, unknown email (anti-enumeration), Google-only user, happy confirm, invalid token, weak password, deleted user.
- [ ] Integration test passes end-to-end: request → confirm → login-with-new → old-password-dead → token-single-use.
- [ ] `make api-test` is green (no regressions in the existing 30 flow files).

## Risk Assessment

- **miniredis not in go.mod:** Discovered at step 1. Mitigation: add the dep (it's a well-maintained, widely-used test-only Redis mock) OR gate the TTL/concurrency tests behind `make db`-provided Redis. Prefer miniredis for unit-level determinism.
- **Integration test can't read the emailed token in sandbox mode:** Mitigated by the `SandboxProvider.LastSent()` accessor (step 4). If the provider is already structured to capture sent messages, even easier.
- **Flaky concurrency test:** The concurrent-consume test asserts exactly-one-wins; with `GETDEL` this is guaranteed, but the assertion must use a `sync.WaitGroup` + `atomic.Counter`, not sleep-based timing.
