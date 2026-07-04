---
phase: 2
title: "OTP service & login gate"
status: pending
priority: P1
effort: "M (1-1.5d)"
dependencies: [1]
---

# Phase 2: OTP service & login gate

## Overview

The core: an `OTPService` (code generation, email dispatch via `EmailDeliveryPort`,
Redis pending-session lifecycle) and the modification of `AuthService.Login`
(and `LoginWithGoogle`) to gate admin/partner accounts behind a second step.
After this phase, login either returns a full JWT (OTP off / employee) or
`{otp_required, otp_session_id}` (admin/partner with OTP on). A new
`POST /auth/login/verify` consumes the code and issues the real JWT.

## Requirements

- **Functional:** when `OTP_ENABLE=true` and the user is `admin`/`partner`,
  login does NOT issue a JWT — it generates a 6-digit code, emails it, stores a
  hash in Redis, and returns `otp_session_id`. `/auth/login/verify` validates the
  code and issues the 14-day JWT. `LoginWithGoogle` is gated identically (RT-H5).
- **Non-functional:** codes are `crypto/rand` 6-digit, single-use, 5-min TTL;
  pending sessions are opaque, high-entropy, bound to IP+UA (RT-M5), one-per-user
  (RT-H1); the real JWT is minted only after verify success.

## Architecture

### Two-step login flow (Option B — Redis temp session)

```
POST /auth/login {username, password}
  ├─ password OK, role∈{admin,partner}, OTP_ENABLE?
  │    YES → RT-M8: if user.Email == nil → 401 "email required for OTP, contact admin"
  │         check otp_locked_until → if locked → 401 "try again in Nm"
  │         generate 6-digit code (crypto/rand)
  │         hash code (SHA-256); CreateSession(userID, hash, ip, ua) → otp_session_id
  │         send code via EmailDeliveryPort (async; non-blocking goroutine w/ error log)
  │         return 200 {otp_required:true, otp_session_id, expires_in:300}
  │    NO  → (existing path) return 200 {access_token, ...}
  └─ password FAIL → 401 (unchanged)

POST /auth/login/verify {otp_session_id, code}
  ├─ GetSession → {user_id, code_hash, ip, ua, attempts}; validate ip+ua match (RT-M5)
  ├─ RT-M4: re-check users.tokens_invalid_before and otp_locked_until (may have changed)
  ├─ constant-time compare code vs code_hash
  ├─ ALL PASS → DeleteSession, reset otp_failed_attempts=0, issue 14-day JWT (OTPVerified=true),
  │             stamp last_login, publish login audit → return LoginResponse
  └─ FAIL → IncrementSessionAttempts; bump users.otp_failed_attempts (shared, RT-H1);
            at 5 → otp_locked_until = now+15m; return 401
```

### OTPService responsibilities
- `GenerateCode() (string, error)` — 6-digit `crypto/rand`, unbiased (rejection sampling to avoid modulo bias).
- `HashCode(code) []byte` — SHA-256.
- `StartLogin(ctx, user, ip, ua) (sessionID string, err error)` — the gate body above.
- `VerifyLogin(ctx, sessionID, code, ip, ua) (*domain.User, error)` — load session, compare, enforce lockout/revocation re-check.
- `SendOTPEmail(ctx, user, code) error` — builds `domain.EmailMessage` (Phase 3 template), calls `EmailDeliveryPort.Send`.

### Why RT-C2 (claim defaults false) matters here
`generateAccessToken` must take a new `otpVerified bool` param. Set it `true`
**only** inside `VerifyLoginOTP`'s success branch and on non-gated paths
(employee, or admin/partner when `OTP_ENABLE=false`). The password-only branch
through `Login` (when it falls through, e.g. employee) sets it according to
whether THAT path was OTP-gated — for employees, `true` (they're not gated); for
admin/partner under flag-on, that branch is never reached (the gate returns early).

## Related Code Files

- **Create:** `internal/app/services/otp/otp_service.go` — the service.
- **Create:** `internal/app/services/otp/code.go` — `GenerateCode`/`HashCode` (small, testable).
- **Modify:** `internal/app/services/auth/auth_service.go:96-189` (`Login`) — insert the OTP gate at ~line 137 (after password OK, before `generateAccessToken`). See §Implementation.
- **Modify:** `internal/app/services/auth/auth_service.go:493-601` (`LoginWithGoogle`) — RT-H5: same gate when `Enabled && role∈{admin,partner}`.
- **Modify:** `internal/app/services/auth/auth_service.go` — `Claims` gets `OTPVerified bool` (default false); `generateAccessToken` signature changes; add `VerifyLoginOTP(ctx, sessionID, code, ip, ua) (*dto.LoginResponse, error)`.
- **Modify:** `internal/transport/http/handlers/auth.go` — add `VerifyLoginOTP` handler.
- **Modify:** `internal/app/dto/user.go:46` (`LoginResponse` — RT-L2: it lives here, NOT `dto/auth.go`) — union-ish shape via `omitempty` (see below).
- **Modify:** `internal/app/bootstrap/routes_auth.go` — add `auth.POST("/login/verify", LoginRateLimit, Auth.VerifyLoginOTP)`.
- **Modify:** `internal/app/services/auth/auth_service.go:298-328` (`RevokeUserTokens`) — RT-M4: also call `OTPPendingStore.DeleteSessionsForUser(userID)`.

### DTO shape (`LoginResponse`)
```go
type LoginResponse struct {
    User        *UserResponse `json:"user,omitempty"`
    AccessToken string        `json:"access_token,omitempty"`
    TokenType   string        `json:"token_type,omitempty"`
    ExpiresIn   int64         `json:"expires_in,omitempty"`
    OTPRequired bool   `json:"otp_required,omitempty"`
    OTPSessionID string `json:"otp_session_id,omitempty"`
}
```
`omitempty` keeps the existing shape for non-OTP logins (frontend compat).

## Implementation Steps

1. **`otp/code.go`** — `GenerateCode`: 6-digit, `crypto/rand`, rejection sampling
   (draw bytes, reject values that would introduce modulo bias). Unit-test:
   distribution over 10k samples, all outputs ∈ [0, 999999], zero-padded 6-char string.
2. **`OTPService.StartLogin`** — per the architecture: email-presence check
   (RT-M8), lockout check, code gen, hash, `CreateSession`, fire-and-forget email
   send. Return `sessionID`. **RT-M5:** capture `ip` + `ua` into the session.
3. **Gate `AuthService.Login`** at `auth_service.go:~137`:
   ```go
   if s.otpConfig.Enabled && (user.IsAdmin() || user.IsPartner()) {
       if user.Email == nil || *user.Email == "" {
           return nil, domain.NewUnauthorizedError("tài khoản chưa có email, không thể bật OTP — liên hệ quản trị")
       }
       if user.OTPLockedUntil != nil && user.OTPLockedUntil.After(clock.Now()) {
           return nil, domain.NewUnauthorizedError("tài khoản tạm bị khóa, thử lại sau")
       }
       sessionID, err := s.otpService.StartLogin(ctx, user, ip, ua)
       if err != nil { return nil, err }
       return &dto.LoginResponse{OTPRequired: true, OTPSessionID: sessionID, ExpiresIn: 300}, nil
   }
   ```
   **RT-C2:** no `generateAccessToken` on this branch — the JWT is never minted here.
4. **Gate `LoginWithGoogle`** (RT-H5) — same logic after the Google user is
   resolved, when `Enabled && role∈{admin,partner}`. Google login is NOT a free
   pass into admin/partner; it gets an OTP challenge just like password login.
5. **`VerifyLoginOTP`** — per architecture: load session, IP/UA check, revocation
   re-check (RT-M4), constant-time compare, on success reset lockout + issue JWT
   with `OTPVerified=true`; on fail bump shared per-account counter + lock at 5.
6. **`RevokeUserTokens`** — add `s.otpPendingStore.DeleteSessionsForUser(userID)`
   so revoking a user also kills their pending OTP sessions (RT-M4).
7. **Wire route** — `/auth/login/verify` with `LoginRateLimit` (10/min/IP from 619f9cd).

## Success Criteria

- [ ] With `OTP_ENABLE=false`, login is **byte-identical** to today (regression-safe).
- [ ] With `OTP_ENABLE=true`, admin/partner login returns `{otp_required, otp_session_id}` and NO `access_token`.
- [ ] **RT-M8:** admin/partner with `email == nil` → 401 with the clear VN message.
- [ ] **RT-H5:** `LoginWithGoogle` for an admin/partner also returns an OTP challenge.
- [ ] `/auth/login/verify` with the correct code returns a valid 14-day JWT with `otp_verified:true`.
- [ ] **RT-H1:** a second `/auth/login` for the same user invalidates the first `otp_session_id`.
- [ ] **RT-M5:** verifying from a different IP/UA than the session was created with → 401.
- [ ] **RT-M4:** after `RevokeUserTokens`, a pending `otp_session_id` no longer verifies.
- [ ] 5 failed verifies lock the account for 15 min (per-account, not per-session).
- [ ] `go test ./internal/app/services/otp/... ./internal/app/services/auth/...` passes.

## Risk Assessment

- **Risk:** email send latency/failure blocks the login response. **Mitigation:**
  fire-and-forget goroutine with error logging; the response returns immediately
  with `otp_session_id`. If the email truly failed, the user hits "resend" (Phase 3).
- **Risk:** `crypto/rand` modulo bias on the 6-digit code. **Mitigation:**
  rejection sampling in `GenerateCode`; unit test the distribution.
- **Risk:** failing closed incorrectly when `EmailDeliveryPort` is misconfigured.
  **Mitigation:** that's a deploy-time concern surfaced by Phase 4 boot validation
  (`OTP_ENABLE=true` requires a working email config); documented in runbook.
