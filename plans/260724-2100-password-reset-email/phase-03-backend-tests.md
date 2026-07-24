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

`backend/tests/integration/auth_password_reset_test.go` — mirrors the structure of the existing OTP integration test (look at `auth_otp_test.go` / `auth_login_test.go` in the same dir). It hits the live API.

**Red Team C3 (concrete token-retrieval strategy):** The earlier draft hand-waved "intercept the token from the SandboxProvider." This does NOT work: integration tests run cross-process (`tests/integration/client.go:195` is a pure HTTP client against `PAYROLL_BASE_URL`; `make api-test` = `go run ./tests/integration/`). The server's `SandboxProvider` instance is unreachable across the process boundary — `LastEmail()` (`sandbox_provider.go:81`) is only callable by code holding the provider pointer. No existing integration test retrieves sandboxed emails.

**Chosen strategy (two-layer):**
1. **Unit tests (in-process, `passwordreset/service_test.go`)** cover the single-use, session-invalidation, and transactional guarantees by injecting a **capturing `EmailSender` mock** that records the `EmailMessage` (and thus the token, parsed from the CTA link). This is where the token-consuming logic is verified deterministically.
2. **Integration test (cross-process, `tests/integration/`)** covers ONLY the contracts that don't require reading the token:
   - Anti-enumeration (200 for known AND unknown email, identical body).
   - Rate limit (429 after 3 requests/hour/email).
   - Confirm with a garbage token → 401.
   - Confirm with a well-formed but non-existent token → 401.
   - Feature flag off → endpoints 404.
3. **For the end-to-end happy path** (request → real token → confirm → login-with-new), add a **dev-only HTTP endpoint** `GET /dev/sandbox-emails` (gated on `cfg.App.Env != "production"`) that returns captured sandbox messages as JSON. The integration test calls it to retrieve the token. This matches the existing sandbox-provider pattern (dev-only, never registered in prod) and requires no production code change beyond a small dev-only route. **Alternative if the team rejects a dev endpoint:** write the happy-path test as an in-process Go test using `httptest` + direct DI (not in `tests/integration/`), reserving the cross-process suite for the contract checks above.

Integration test flow (revised):
1. Seed a user with a known email + password.
2. `POST /auth/password-reset/request {email}` → assert 200 + generic message.
3. `GET /dev/sandbox-emails?to={email}` (dev-only) → extract `token` from the CTA link in the captured email. (Or skip the happy path if dev endpoint rejected.)
4. `POST /auth/password-reset/confirm {token, new_password}` → assert 200.
5. `POST /auth/login {username, new_password}` → assert 200 (new password works).
6. `POST /auth/login {username, old_password}` → assert 401 (old password dead).
7. `POST /auth/password-reset/confirm {token, new_password2}` → assert 401 (token single-use).
8. Anti-enumeration: `POST /auth/password-reset/request {unknown@email.com}` → assert **same** 200 + message.
9. Rate limit: 4th `POST /auth/password-reset/request {email2}` within the hour → assert 429.

## Related Code Files

- **Create:** `backend/internal/infra/cache/password_reset_token_store_test.go`
- **Create:** `backend/internal/app/services/passwordreset/service_test.go`
- **Create:** `backend/tests/integration/auth_password_reset_test.go`
- **Possibly modify:** `backend/internal/app/bootstrap/routes_dev.go` (or a new dev-only route file) — add `GET /dev/sandbox-emails` gated on `cfg.App.Env != "production"` to expose captured sandbox messages for the integration test (Red Team C3). Do NOT add retention code to the production `SandboxProvider` — it already has `LastEmail()` (`sandbox_provider.go:81`).

## Implementation Steps

1. **Confirm miniredis availability:** `grep miniredis backend/go.mod`. If present, use it for `password_reset_token_store_test.go`. If absent, either add it (`go get github.com/alicebob/miniredis/v2`) or test against the live Redis started by `make db`.

2. **Write `password_reset_token_store_test.go`:**
   - `TestCreate_ReturnsHighEntropyToken`: token is ~43 chars, base64url-charset, two `Create` calls produce different tokens.
   - `TestConsume_HappyPath`: `Create` then `Consume` returns the same `userID`.
   - `TestConsume_SingleUse`: `Create` once, `Consume` twice → second returns `ErrPasswordResetTokenNotFound`.
   - `TestConsume_Concurrent`: `Create` once, fire 10 goroutines calling `Consume` → exactly one succeeds, nine get the error. (This is the critical race test — run with `-race`.)
   - `TestConsume_UnknownToken`: `Consume("garbage")` → `ErrPasswordResetTokenNotFound`.
   - `TestConsume_ExpiredToken`: (miniredis) `Create`, advance the miniredis clock past TTL, `Consume` → error.

3. **Write `service_test.go`** with table-driven cases (using a **capturing EmailSender mock** that records the `EmailMessage` so the token can be parsed from the CTA link — this is how the single-use happy path is verified in-process per Red Team C3):
   - `RequestReset_KnownUser_SendsEmail`: mock records the call; assert `EmailSender.Send` invoked once with a message containing the reset URL + token.
   - `RequestReset_UnknownEmail_ReturnsNilNoEmail_TimingEqualized`: assert `EmailSender.Send` never called, no error returned, AND a dummy `tokenStore.Create` WAS called (Red Team H2 timing equalization).
   - `RequestReset_TransientDBError_ReturnsNilNoEmail_TimingEqualized`: `GetByEmail` returns an internal error → same behavior as not-found (no enumeration leak).
   - ~~`RequestReset_GoogleOnlyUser_ReturnsNilNoEmail`~~ **REMOVED (Red Team C2 — no such accounts exist; the check is dead code).**
   - `ConfirmReset_HappyPath`: token valid → `UpdatePasswordAndInvalidateSessions` called ONCE with (userID, hashed, ~now) in a single transaction (Red Team C1 — verify the repo mock's `UpdatePasswordAndInvalidateSessions` is called, NOT separate `Update` + `UpdateTokensInvalidBefore`).
   - `ConfirmReset_InvalidToken`: `ErrPasswordResetTokenNotFound` → `domain.UnauthorizedError` (401).
   - `ConfirmReset_StoreUnavailable`: `ErrPasswordResetStoreUnavailable` → `domain.InternalError` (500, NOT 401) (Red Team H5).
   - `ConfirmReset_WeakPassword`: validator returns error → `domain.ValidationError`, password NOT updated.
   - `ConfirmReset_DeletedUser`: token valid but `GetByID` not-found → `UnauthorizedError`, token already consumed.
   - `ConfirmReset_AuditAttribution`: verify `NewPasswordChangedEvent` published with `actorUserID = user.ID` (Red Team Security-5, not 0).
   - `RequestReset_GoroutinePanicRecovered`: inject an `EmailSender` whose `Send` panics; assert the panic is recovered (Red Team H4) and the request still returns nil. (Use a sync primitive to ensure the goroutine ran before the assertion.)
   - Use `testify/mock` or hand-rolled mocks — match whatever the existing `otp_service_test.go` uses (read it first and mirror the style).

4. **Red Team C3 / Scope-7 (SandboxProvider accessor):** `sandbox_provider.go:81` already has `LastEmail() *CapturedEmail` — do NOT add a duplicate `LastSent()`. For the cross-process integration test, use the dev-only `GET /dev/sandbox-emails` endpoint (Phase 3 Architecture) OR limit the integration test to the no-token contracts. Do not add dev-only retention code to the production `SandboxProvider` without a build tag.

5. **Write `auth_password_reset_test.go`** following the 8-step flow above. Reuse the integration test helper that creates a user + logs in (find it in `tests/integration` — likely a `seedUser` or similar helper).

6. **Run everything:**
   - `cd backend && go test ./internal/infra/cache/... -v -race -run PasswordReset`
   - `cd backend && go test ./internal/app/services/passwordreset/... -v -race`
   - `make api-test` (after the backend is running via `make dev` / `make db`)

## Success Criteria

- [ ] Token store tests pass under `-race`, including the concurrent-consume test proving single-use.
- [ ] Service unit tests cover: known user, unknown email (anti-enumeration, timing-equalized), transient DB error (treated as not-found), happy confirm, invalid token, store-unavailable (500 not 401), weak password, deleted user, goroutine-panic-recovered, audit-actor-is-user.
- [ ] Integration test passes end-to-end: request → confirm → login-with-new → old-password-dead → token-single-use.
- [ ] `make api-test` is green (no regressions in the existing 30 flow files).

## Risk Assessment

- **miniredis not in go.mod:** Discovered at step 1. Mitigation: add the dep (it's a well-maintained, widely-used test-only Redis mock) OR gate the TTL/concurrency tests behind `make db`-provided Redis. Prefer miniredis for unit-level determinism.
- **Integration test can't read the emailed token cross-process:** Mitigated by the dev-only `GET /dev/sandbox-emails` endpoint (step 4 / Architecture). Fallback: cover the single-use + transactional guarantees in unit tests with a capturing mock, limit the integration test to the no-token contracts.
- **Flaky concurrency test:** The concurrent-consume test asserts exactly-one-wins; with `GETDEL` this is guaranteed, but the assertion must use a `sync.WaitGroup` + `atomic.Counter`, not sleep-based timing.
