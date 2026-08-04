---
phase: 2
title: "Backend: Zalo OTP Reset Store & Service"
status: pending
priority: P1
dependencies: ["1"]
effort: "M"
---

# Phase 2: Backend: Zalo OTP Reset Store & Service

## Overview

Build the business layer that turns a mobile number into a single-use OTP session and
later consumes that session to set a new password. Two artifacts:

1. **`infra/cache/ZaloResetStore`** — a focused Redis store (mirrors the email
   `PasswordResetTokenStore` shape, not the login-OTP store — see D2 in `plan.md`).
2. **`app/services/zaloreset.Service`** — orchestrates lookup → code-gen → store → async
   ZNS send (request), and consume → validate → atomic password update + event (confirm).
   Its security contract is copied line-for-line from `passwordreset.Service`:
   anti-enumeration, timing equalization, single-consume, single-transaction password +
   session invalidation, `PasswordChangedEvent(method="zalo_reset")`.

## Requirements

- **Functional**
  - `RequestReset(ctx, mobile) error` — always returns `nil`; never reveals whether the mobile exists. Looks up `users.mobile` **scoped to role='employee'**, generates + stores code, dispatches ZNS async.
  - `ConfirmReset(ctx, sessionID, code, newPassword) error` — consumes the code, validates password strength, hashes, updates password + invalidates sessions in one transaction, publishes event.
  - `ResendCode(ctx, sessionID) (string, error)` — re-issues a code for an existing session with a cooldown (reuse `OTPConfig.ResendCooldown` shape; see Open Question Q-Phase2-1).
- **Non-functional**
  - The plaintext 6-digit code is **never** stored in Redis — only its SHA-256 hash (`otp.HashCode`).
  - `session_id` is 256-bit base64url (32 bytes → 43 chars), generated exactly like `otp_pending_store.go:newOpaqueID`.
  - TTL **10 minutes** (configurable via `ZALO_RESET_CODE_TTL` env at boot; passed into the store + service constructors — this is a *tuning* value, not part of the DB-driven admin connection, so it stays env-only like `OTPConfig.CodeTTL`).
  - `Consume` is **atomic** (Redis Lua `GETDEL`) — a double-submit or replay succeeds at most once.
  - Request path latency is **equalized** between found and not-found via a dummy `store.Create` (port the email reset's H2 timing defense).

## Architecture

### Redis key schema

```
zreset:<session_id_hash>   →  <userID>            (TTL 10m)
zreset:idx:<userID>        →  <session_id_hash>   (TTL 10m)  [optional, for per-user single-live-session]
```

- `session_id_hash` = SHA-256 hex of the opaque `session_id` (so a Redis dump can't be replayed).
- We **do not** store the code in Redis at all — see *Why hash-only* below.

> **Why hash-only (no code-in-Redis)?** The `OTPPendingStore` stores the code's hash and
> does the compare in Go after reading the session. That works for login because the
> session is bound to IP/UA. For an unauthenticated reset the only secret is the code
> itself, so we collapse the session into a single value: `GETDEL zreset:<hash>` returns
> the userID if the caller supplied a `sessionID` that hashes to a live key. The code is
> verified by re-hashing: we store `codeHash` as the **value** and compare in Lua. This
> keeps everything in one atomic round-trip.

**Revised (single-key, atomic compare-and-delete):**

```
KEY:   zreset:<sha256_hex(session_id)>
VALUE: <userID>:<sha256_hex(code)>           (TTL CodeTTL)

Consume Lua:
  local v = redis.call('GET', KEYS[1])
  if not v then return '-1' end                -- not found / expired / consumed
  local uid, hash = string.match(v, '^(%d+):(.+)$')
  if hash ~= ARGV[1] then return '-2' end      -- wrong code (key survives for retry)
  redis.call('DEL', KEYS[1])
  return uid                                   -- success
```

This gives us: atomic consume-on-correct-code, **wrong code does not consume** (lets the
user retry within the same session's TTL), and a single round-trip. The `OTPFailedAttempts`
lockout question (R-Z3 in `plan.md`) is what would change the `-2` branch; deferred.

> **Trade-off documented:** keeping the session alive on wrong-code is **intentional** and
> different from the email reset (which GETDELs the token regardless). Email reset tokens
> are high-entropy 256-bit magic links — a wrong submission is attacker noise, consume it.
> OTP sessions are 6-digit user-typed codes — a typo should not burn the session. The rate
> limiter still caps requests per mobile.

### Service shape

```go
package zaloreset

// EnabledChecker is implemented by zaloconnect.Service (Phase 4). It lets the
// reset service hot-check the admin toggle on every request without depending
// on env or config. Returns false if the Settings row is missing/disabled.
type EnabledChecker interface {
    IsEnabled(ctx context.Context) (bool, error)
}

type Service struct {
    store      *cache.ZaloResetStore
    userRepo   domain.UserRepository
    userService *user.UserService
    zalo       zalo.Sender
    templateID string
    codeTTL    time.Duration // drives otp_valid_in_minutes param + store TTL alignment
    enabled    EnabledChecker // hot-read of admin toggle (Phase 4); nil = always-on (dev)
    eventBus   domain.EventBus
    clk        clock.Clock
    logger     *slog.Logger
    resendCooldown time.Duration
}

func NewService(store *cache.ZaloResetStore, userRepo domain.UserRepository,
    userService *user.UserService, zalo zalo.Sender, templateID string,
    codeTTL time.Duration, enabled EnabledChecker, eventBus domain.EventBus,
    clk clock.Clock, logger *slog.Logger) *Service
```

> `codeTTL` is passed explicitly (rather than read from a config struct) so the service
> is testable without loading config. It must equal the store's TTL — Phase 3 wiring
> passes the same value to both. The integer-minutes rendering of this value becomes
> the `otp_valid_in_minutes` template param.
>
> **`enabled` is a hot-check, not a boot flag.** `RequestReset` calls
> `enabled.IsEnabled(ctx)` on every request. If the admin toggled Zalo off via Phase 4's
> `PUT /admin/zalo/enabled`, the very next `/auth/zalo-reset/request` returns the generic
> anti-enumeration 200 **without** dispatching ZNS — no redeploy. (Anti-enumeration is
> preserved: the not-found path already doesn't dispatch; the disabled path now behaves
> identically, so timing doesn't leak the toggle state to a prober.)

### `RequestReset` flow (mirrors `passwordreset.Service.RequestReset`)

```
// Hot-check the admin toggle (Phase 4). If off, behave like a not-found: dummy
// Redis write + return nil. Anti-enumeration preserved — disabled path is
// indistinguishable from not-found-mobile path.
if s.enabled != nil {
    if on, _ := s.enabled.IsEnabled(ctx); !on {
        store.CreateDummy(ctx)
        return nil
    }
}

u, err := userRepo.GetByMobile(ctx, NormalizeMobile(mobile))
if err != nil or u.Role != "employee":
    // Anti-enumeration H2: equalize timing with a dummy Create so a not-found
    // has the same Redis RTT + hash profile as a found path.
    store.CreateDummy(ctx)
    return nil

code, _ := otp.GenerateCode()           // reuse existing
sessionID, _ := store.Create(ctx, u.ID, otp.HashCode(code))

go func() {
    defer recover()/log on panic        // H4: never crash the process on send
    bg, cancel := context.WithTimeout(background, 15s)
    defer cancel()
    trackingID := buildTrackingID(u.ID)
    // Template 617976 (OTP-ZNS-v1) requires all three params; ClampParams
    // (Phase 1) enforces per-param caps before the POST.
    res, err := zalo.Send(bg, mobile, templateID, trackingID, map[string]string{
        "otp_code":             code,
        "user_fullname":        u.Fullname,                          // capped to 30 by ClampParams
        "otp_valid_in_minutes": strconv.Itoa(int(s.codeTTL.Minutes())), // "10" for a 10m TTL
    })
    // log result + audit; NEVER return an error path that changes the HTTP response
}()
return nil
```

> **`user_fullname` source:** `users.fullname` (NOT NULL per schema). ClampParams truncates
> to the 30-rune template cap, so long names are safe. The service does **not** fall back to
> `username` — the ZNS message is user-facing and the OA-approved template expects a display
> name. If `fullname` is empty for some legacy row, `ClampParams` passes `""` through and
> Zalo renders the slot blank ( preferable to inventing a value).

**Role-gate (D6):** the `GetByMobile` result is checked for `Role == RoleEmployee`.
Admin/partner mobiles go down the not-found (dummy create) path — they are never
enrolled in the Zalo reset flow, and the response is byte-identical.

### `ConfirmReset` flow (mirrors `passwordreset.Service.ConfirmReset`)

```
userID, err := store.Consume(ctx, sessionID, otp.HashCode(code-in-clear-from-request))
if err: return mapError(err)   // -1 → invalid/expired, -2 → wrong code

u, _ := userRepo.GetByID(ctx, userID)
if err: return invalid          // token pointed at a deleted user

if err := userService.ValidatePassword(newPassword); err != nil:
    return ValidationError(err)

hashed, _ := userService.HashNewPassword(newPassword)

// C1 atomic transaction: password + tokens_invalid_before commit together.
userRepo.UpdatePasswordAndInvalidateSessions(ctx, u.ID, hashed, clk.Now())

eventBus.Publish(ctx, NewPasswordChangedEvent(ctx, u.ID, u.Username, "zalo_reset", u.ID, ""))
```

> **Reuse note:** `UpdatePasswordAndInvalidateSessions`, `ValidatePassword`,
> `HashNewPassword`, and `NewPasswordChangedEvent` are **all** existing — Phase 2 calls
> them, does not modify them. The actor is the user themselves (`actorUserID = u.ID`,
> `actorFullName = ""`), exactly as the email reset does (its `// Red Team Security-5`
> comment documents why).

## Related Code Files

- **Create:**
  - `backend/internal/infra/cache/zalo_reset_store.go` + `_test.go`
  - `backend/internal/app/services/zaloreset/{service,doc}.go` + `_test.go`
- **Modify:** none (composition-only — existing repos/services are passed in).
- **Reference (read-only):**
  - `backend/internal/app/services/passwordreset/service.go` — copy the security-contract comments verbatim, adapt medium.
  - `backend/internal/infra/cache/password_reset_token_store.go` — store shape (Create/Consume, Lua GETDEL).
  - `backend/internal/app/services/otp/code.go` — `GenerateCode`, `HashCode`, `EqualCodeHash` (only `GenerateCode` + `HashCode` needed here; the Lua compare replaces `EqualCodeHash`).
  - `backend/internal/domain/user.go:80` — `UpdatePasswordAndInvalidateSessions`.
  - `backend/internal/domain/event_factory_user.go:61` — `NewPasswordChangedEvent`.

## Implementation Steps

1. **`ZaloResetStore`** — `Create(ctx, userID uint, codeHash []byte) (sessionID string, err error)`, `CreateDummy(ctx) error`, `Consume(ctx, sessionID string, codeHash []byte) (userID uint, err error)`. Lua script for Consume (above). Use the existing redis client; key prefix `zreset:`. TTL from constructor arg.
2. **`ZaloResetStore` tests** — happy Create→Consume; Consume-unknown → `ErrSessionNotFound`; Consume-wrong-code → `ErrInvalidCode` (key survives, second Consume with right code works); TTL expiry (use `clock` + manual `redis.EXPIRE` in test); concurrent Consume (only one wins, `-race`).
3. **`Service.RequestReset`** — implement; unit-test with a fake `userRepo` + `zalo` sender + spy `store`. Assert: not-found → no Send call, dummy Create called, returns nil; found-employee → exactly one `zalo.Send` (eventually, via a sync fake), code passed via `template_data["otp_code"]`; found-admin → no Send (role gate), dummy Create called.
4. **`Service.ConfirmReset`** — implement; unit-test: happy path calls `UpdatePasswordAndInvalidateSessions` with the hashed password and publishes the event; wrong code → `ErrInvalidCode` and **no** password update; consume-twice → second returns `ErrSessionNotFound`.
5. **`Service.ResendCode`** — implement with `resendCooldown`; assert resend inside cooldown → `ErrResendCooldown`; outside → new code, new Send.
6. **Timing test** — `RequestReset("known")` vs `RequestReset("unknown")` measured wall-clock; assert the delta is `< 5ms` after warm-up (port the spirit of the email reset's H2 defense).

## Success Criteria

- [ ] `store.Consume` is atomic under `-race`: two concurrent calls with the same correct code → exactly one returns the userID, the other `ErrSessionNotFound`.
- [ ] A wrong-code `Consume` does **not** consume the session — a subsequent correct `Consume` still succeeds.
- [ ] `RequestReset` for an admin-role mobile dispatches **zero** ZNS sends and returns `nil` (anti-enumeration + role gate).
- [ ] `RequestReset` for a known-employee mobile calls `zalo.Send` with all three template params populated: `otp_code` (6 digits), `user_fullname` (= `u.Fullname`), `otp_valid_in_minutes` (= `strconv.Itoa(int(codeTTL.Minutes()))`). Verified by the fake `zalo.Sender` recording the last `template_data`.
- [ ] When `EnabledChecker.IsEnabled` returns `false`, `RequestReset` performs a dummy `store.CreateDummy` and returns `nil` **without** dispatching ZNS — identical behavior to a not-found mobile (anti-enumeration + hot-toggle).
- [ ] `RequestReset("unknown")` and `RequestReset("known-employee")` produce the same response shape from the caller's perspective (handler-level assertion is in Phase 3; here assert the service returns `nil` for both and the dummy Create is invoked on not-found).
- [ ] `ConfirmReset` happy path calls `UpdatePasswordAndInvalidateSessions` exactly once and publishes `PasswordChangedEvent` with `method="zalo_reset"`.
- [ ] A send goroutine panic is recovered and logged; the process does not crash.
- [ ] `go test ./internal/app/services/zaloreset/... ./internal/infra/cache/... -race` green.

## Risk Assessment

- **R-Z5 (wrong-code key survival):** Allowing the session to survive a wrong code is user-friendly but removes one anti-brute-force lever. **Mitigation:** the per-mobile rate limiter (Phase 3) caps requests at `RatePerHour`/hour, bounding total wrong-code attempts to ≤ that cap per session. If this proves insufficient, add `OTPFailedAttempts`-style lockout in a follow-up (R-Z3 in `plan.md`).
- **R-Z6 (Zalo `-118` "no Zalo account"):** ~10–20% of Vietnamese phones may not have a linked Zalo account. The send will succeed from our side (Zalo accepted the request) but the user gets nothing. **Mitigation:** the service logs `-118` at `Info` (not Error); the `/forgot-password` page shows a secondary "Không nhận được mã? Liên hệ quản trị viên" help link (Phase 7). Not a code risk; an UX/comms one.
- **R-Z7 (mobile format in DB):** `users.mobile` is `varchar(15)` and historically may be stored as `0987...` or `+8498...`. `NormalizePhone` (Phase 1) handles all forms for the ZNS call. The **lookup** (`GetByMobile`) must use the same normalization the write-path uses — verify against existing admin "create user" code before finalizing. If DB stores raw `0987...` and the user types `+8498...`, the lookup misses and we silently no-op (safe but confusing). **Action item:** in Phase 2, check `UserRepository.GetByMobile` query; if it doesn't normalize, add a `NormalizeMobileForLookup` helper and use it on both the request path and the existing user-create path.
