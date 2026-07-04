---
phase: 4
title: "Config & enforcement"
status: pending
priority: P1
effort: "S (0.5d)"
dependencies: [1]
---

# Phase 4: Config & enforcement

## Overview

Wire the `OTP_ENABLE` env flag + `OTPConfig` into boot config, and add the
middleware-layer enforcement so a token lacking a verified 2FA step cannot reach
privileged routes. This phase is what makes the flag a real kill-switch — and
the **RT-C1 fix lives here**: enforcement is a property of `Authorize()`, not a
route-group middleware, because money routes are mounted on `v1` not `protected`.

## Requirements

- **Functional:** `OTP_ENABLE` (default `false`) gates the feature. When on,
  protected routes for admin/partner require a token issued post-OTP. When off,
  behavior is identical to today.
- **Non-functional:** flag read at boot (fail-fast on misconfig); enforcement is
  a single check inside the RBAC layer; a build-time route-coverage test prevents
  regressions; fail-closed.

## Architecture

### Config — new `OTPConfig` on `Config`
```go
type OTPConfig struct {
    Enabled       bool          // OTP_ENABLE
    CodeTTL       time.Duration // OTP_CODE_TTL (default 5m) — pending-session TTL
    MaxAttempts   int           // OTP_MAX_ATTEMPTS (default 5)
    LockDuration  time.Duration // OTP_LOCK_DURATION (default 15m)
    ResendCooldown time.Duration // OTP_RESEND_COOLDOWN (default 30s)
}
```
Populated in `config.Load()` via `getEnv`/`parseBool`/`parseDuration`
(existing pattern at `config.go:492-517`). **No `OTP_SECRET_KEY` needed**
(email-OTP has no stored secret — retired RT-M7).

### Boot validation (fail-fast) — `Config.validate()` (`config.go:458-490`)
Mirror the `JWT_SECRET` check at `:459`:
```go
if c.OTP.Enabled {
    if c.Email.FromEmail == "" {
        return fmt.Errorf("EMAIL_FROM must be set when OTP_ENABLE=true (needed to deliver codes)")
    }
    if c.Email.ResendAPIKey == "" {
        return fmt.Errorf("RESEND_API_KEY must be set when OTP_ENABLE=true")
    }
}
```
Demo can run `OTP_ENABLE=false` with no changes; prod must have a working email
config or the server won't start. This is the recovery for the retired RT-M7's
"boot loop" concern — the failure is now about email config, not a missing key.

### Enforcement: `otp_verified` claim, checked INSIDE `Authorize()` (RT-C1)

> ⚠️ **RT-C1.** The original design attached a middleware to the `protected`
> group in `routes.go:49`. That structurally cannot work: `/wallet`,
> `/admin/manual-disbursement`, `/settings`, `/admin/attendances` are mounted on
> `v1` directly and apply `Authenticate()`+`Authorize()` themselves. Enforcement
> must travel with RBAC.

Add a claim to the real access token:
```go
type Claims struct {
    // ...existing...
    OTPVerified bool `json:"otp_verified,omitempty"`   // defaults false (RT-C2)
}
```
**Defaults `false`.** Set `true` ONLY inside `VerifyLoginOTP` (Phase 2) and on
non-gated paths (employee, or admin/partner when `OTP_ENABLE=false`).

**Enforcement inside `Authorize()`** (after the Casbin allow decision):
```go
if m.otpConfig.Enabled && (role == RoleAdmin || role == RolePartner) {
    if !claims.OTPVerified {
        response.Forbidden(c, "OTP verification required")
        c.Abort()
        return
    }
}
```
This runs for EVERY route that does RBAC, regardless of mount point — covering
`/wallet` and `/admin/manual-disbursement` automatically.

### Build-time route-coverage test (mandatory)
`bootstrap/routes_otp_coverage_test.go`: enumerate `engine.Routes()`; fail if any
path matching a privileged prefix (`/admin`, `/wallet`, `/settings`, `/lenders`,
`/loans`, `/ledger`, `/transactions`, `/payrolls`, `/dashboard`, `/cron-jobs`,
`/db-export`, `/email`, `/audit`, `/users`, `/metrics`) lacks the OTP gate when
`OTP_ENABLE=true`. Without this, the next route added to `v1` re-opens RT-C1.

## Related Code Files

- **Modify:** `internal/config/config.go` — add `OTPConfig`, populate in `Load()`, validate.
- **Modify:** `internal/app/services/auth/auth_service.go` — `Claims.OTPVerified` (default false); `generateAccessToken` takes `otpVerified bool`.
- **Modify:** `internal/transport/http/middleware/auth.go` — set `otp_verified` in context from the claim in `Authenticate()`.
- **Modify:** `internal/transport/http/middleware/authorization.go` — **RT-C1:** the OTP check inside `Authorize()`, after the Casbin allow.
- **Create:** `internal/app/bootstrap/routes_otp_coverage_test.go` — the build-time guard.
- **Modify:** `backend/.env.example` — document `OTP_ENABLE`, `OTP_CODE_TTL`, `OTP_MAX_ATTEMPTS`, `OTP_LOCK_DURATION`, `OTP_RESEND_COOLDOWN` (commented, default-off).
- **Modify:** `internal/app/bootstrap/infrastructure/init.go` — pass `OTPConfig` into `AuthService` + `Authorization` middleware.

## Implementation Steps

1. **`config.go`** — add `OTPConfig`, populate (defaults above), add the
   `validate()` block. Unit-test: enabled-without-email → error; enabled-with-email → ok.
2. **`Claims` + `generateAccessToken`** — add the field (default false); change
   `generateAccessToken` to accept `otpVerified bool`; set `true` only at the
   two correct call sites (Phase 2 `VerifyLoginOTP` success; non-gated logins).
3. **`middleware/auth.go`** — in `Authenticate()`, `c.Set("otp_verified", claims.OTPVerified)`.
4. **`middleware/authorization.go`** — add the OTP check inside `Authorize()`
   after the Casbin allow decision, gated on `Enabled && role∈{admin,partner}`.
5. **`routes_otp_coverage_test.go`** — enumerate routes, assert privileged paths
   are gated when `OTP_ENABLE=true`.
6. **`.env.example`** — the documented (commented) block.
7. **Smoke test the kill-switch:** flag-off → all routes work; flag-on + an
   unverified token → admin/partner routes 403 **including `/wallet/...` and
   `/admin/manual-disbursement`** (RT-C1 regression-proof).

## Success Criteria

- [ ] `OTP_ENABLE` unset/false → server boots; all routes work as today.
- [ ] `OTP_ENABLE=true` + missing `RESEND_API_KEY`/`EMAIL_FROM` → server **fails to boot** with a clear error.
- [ ] `OTP_ENABLE=true` + a token with `otp_verified:false` → **every** admin/partner route returns 403, **including `/wallet/...` and `/admin/manual-disbursement`** (RT-C1).
- [ ] `/auth/*` (login/verify/resend) and employee routes remain reachable.
- [ ] The route-coverage test passes and fails if a privileged route is added without the gate.
- [ ] Flipping the flag requires only a restart (documented).

## Risk Assessment

- **Risk:** applying enforcement too broadly → breaks employee login or the
  OTP endpoints. **Mitigation:** gate on `role∈{admin,partner}` inside the check;
  `/auth/*` is never behind `Authorize()` for the login-flow methods.
- **Risk:** forgetting to set `otp_verified:true` on legitimate non-gated paths.
  **Mitigation:** set it true whenever the login path isn't OTP-gated; unit-test both branches.
- **Risk:** boot validation too strict for dev. **Mitigation:** only validates
  when `Enabled=true`; dev stays default-off.
