---
phase: 2
title: "Backend: Service & Handler & Wiring"
status: pending
priority: P1
dependencies: [1]
---

# Phase 2: Backend: Service & Handler & Wiring

## Overview

Compose the Phase 1 building blocks into a `PasswordResetService`, wire it into the auth handler, add two public routes (gated on `cfg.PasswordReset.Enabled`), and write the branded Vietnamese reset-link email template.

## Requirements

- **Functional:**
  - `RequestReset(ctx, email)`: look up user; if found, create a token, email a magic link async. **Always return nil** (generic success) regardless of whether the email exists — the handler maps nil → 200. **Timing equalization** (Red Team H2): on the not-found path, perform a dummy `tokenStore.Create` against a throwaway key so both paths have identical wall-clock profiles.
  - `ConfirmReset(ctx, token, newPassword)`: atomically consume the token, fetch the user, validate password strength, and call **`UpdatePasswordAndInvalidateSessions`** (Red Team C1 — single transaction so password + session-kill commit atomically). Publish `PasswordChangedEvent` with `method="email_reset"` and `actorUserID = user.ID` (self-service — the user IS the actor, matching `NewUserLogoutEvent`'s zero-actor fix at `event_factory_user.go:136`).
- **Non-functional:**
  - Email send is fire-and-forget in a goroutine (15s background ctx) **wrapped in `defer recover()`** (Red Team H4 — a panic in `Send` must not crash the process) so the request returns immediately.
  - All user-facing errors are `domain.NewValidationError` / `domain.NewUnauthorizedError` / `domain.NewInternalError` so the existing `response.HandleDomainError` maps them to correct HTTP codes.
  - **Feature flag is functional** (Red Team H6): routes are registered only when `cfg.PasswordReset.Enabled` is true; when false the endpoints 404 rather than silently returning a misleading "email coming" message.

## Architecture

### Service struct

```go
package passwordreset

type Service struct {
    tokenStore    *cache.PasswordResetTokenStore
    userRepo      domain.UserRepository
    userService   *user.UserService     // Red Team H8: inject directly (ValidatePassword is already exported; hashPassword is private so add ONE delegator only)
    emailSender   domain.EmailDeliveryPort  // Red Team C4: reuse the existing otpEmailSender variable from init.go — do NOT re-instantiate a provider
    fromEmail     string                 // constants.DefaultEmailSenderAddress
    resetURL      string                 // from cfg.PasswordReset.ResetURL
    clk           clock.Clock
    logger        *slog.Logger
}
```

**Red Team H8 (drop gold-plating):** The earlier draft proposed 3 narrow ports (`EmailSender`, `PasswordHasher`, `PasswordValidator`) + 2 delegator methods. Rejected:
- `UserService.ValidatePassword` is **already exported** (`user/password.go:144`) — no delegator needed; call it directly.
- `domain.EmailDeliveryPort` is **already the established port** for email sending (`domain/email.go:174`) and is what `OTPService` accepts — reuse it, don't redefine.
- Only `hashPassword` is private, so add exactly **one** thin delegator (`HashNewPassword`) on `UserService`. This matches the codebase precedent the plan originally cited (`AuthService.userService.VerifyPasswordHash` at `auth_service.go:290` is a direct reach-in).

**Red Team C4 (correct provider wiring):** Do NOT call `email.NewResendProvider(...)` inline. Instead pass the **existing `otpEmailSender` variable** from `services/init.go:607-611` as the `emailSender` argument to `NewService`. That variable already resolves Sandbox (dev) vs Resend (prod/OTP-enabled) and guarantees reset emails reach real inboxes when they need to, without re-instantiating a provider or duplicating the env-gating logic.

### Email template

A new `template.go` in the `passwordreset` package, structurally identical to `otp/template.go`:
- `BuildResetEmailMessage(token, recipientEmail, recipientName, fromEmail, resetURL string) (*domain.EmailMessage, error)`.
- Vietnamese subject: `"Đặt lại mật khẩu TingTing"`.
- Body: a green CTA button linking to `{resetURL}?token={token}`, plus the 30-min expiry notice and the "if you didn't request this, ignore this email" line.
- Reuses the public banner image URL (`https://tingting.vip/email-banner.jpg?v=20260709`) — same as the OTP template.
- **Red Team Scope-3:** Use a new `EmailKindPasswordReset` (add it to the `EmailMessageKind` enum in `domain/email.go` alongside `EmailKindOTP`, rather than the rejected `Metadata["purpose"]` tag which had zero consumers). This matches the established observability enum pattern.

### Handler

Two new methods on `AuthHandler`:

```go
// RequestPasswordReset — POST /auth/password-reset/request
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
    var req dto.PasswordResetRequestDTO
    if !helpers.BindJSON(c, &req) { return }
    // Service ALWAYS returns nil (generic success). Any internal error is
    // logged but NOT surfaced distinctly, to avoid leaking which emails exist.
    _ = h.passwordResetService.RequestReset(c.Request.Context(), req.Email)
    response.Success(c, nil, constants.MsgPasswordResetRequestedVN)
}

// ConfirmPasswordReset — POST /auth/password-reset/confirm
func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
    var req dto.PasswordResetConfirmDTO
    if !helpers.BindJSON(c, &req) { return }
    if err := h.passwordResetService.ConfirmReset(c.Request.Context(), req.Token, req.NewPassword); err != nil {
        response.HandleDomainError(c, err)
        return
    }
    response.Success(c, nil, constants.MsgPasswordResetSuccessVN)
}
```

`AuthHandler` gets a new `passwordResetService *passwordreset.Service` field + constructor param.

### Routes

In `backend/internal/app/bootstrap/routes_auth.go`, inside the `auth := v1.Group("/auth")` block. **Red Team M3:** `routes_auth.go` has no `auth.Use(...)` block — middleware is applied per-route inline (e.g. `/change-password` mounts `Authenticate()` directly). Add the two routes as siblings to the other public routes (`/login`, `/captcha`):

```go
// Red Team H6: only register the reset routes when the feature is enabled.
// When disabled, the endpoints 404 (honest) instead of returning a misleading
// "email coming" 200. Mirrors the OTP fail-fast-at-boot pattern (init.go:608).
if container.Config.PasswordReset.Enabled {
    auth.POST("/password-reset/request", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.RequestPasswordReset)
    auth.POST("/password-reset/confirm", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.ConfirmPasswordReset)
}
```

This requires `Container` to expose `Config` (or specifically `PasswordReset.Enabled`) — check `container.go`; if it only holds `Services`/`Handlers`/`Middleware`, add a `Config *config.Config` field or pass the boolean through. Prefer reading the flag directly to avoid stale snapshots.

### Rate limiter

New limiter in `backend/internal/transport/http/middleware/rate_limit.go`:

```go
// CreatePasswordResetRateLimit caps password-reset requests per EMAIL ADDRESS
// (read from the request body), not per IP — a rotating-IP attacker must not
// bypass the cap. Falls back to IP when the body has no email.
func CreatePasswordResetRateLimit(redisURL string, perHour int) gin.HandlerFunc {
    rate := fmt.Sprintf("%d-H", perHour)
    return createRateLimiterWithKey(RateLimitConfig{Rate: rate, RedisURL: redisURL}, passwordResetEmailKeyGetter)
}
```

**Red Team H1 (unicode normalization) + M1 (body-restore footgun):** `passwordResetEmailKeyGetter` must normalize the email the same way the DB's `utf8mb4_unicode_ci` collation does — `strings.ToLower` is ASCII-only, but `Alice@`, `álice@`, `alice@\u200b` are the same DB row under unicode folding and must collapse to the same limiter key. The implementation:

```go
func passwordResetEmailKeyGetter(c *gin.Context) string {
    const maxBody = 4 << 10
    if c.Request != nil && c.Request.Body != nil {
        raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody))
        // CRITICAL (Red Team M1): restore the body or the downstream handler's
        // BindJSON sees EOF and 400s every reset request.
        c.Request.Body = io.NopCloser(bytes.NewReader(raw))
        if err == nil {
            var payload struct {
                Email string `json:"email"`
            }
            if json.Unmarshal(raw, &payload) == nil {
                if e := strings.TrimSpace(payload.Email); e != "" {
                    // Red Team H1: Unicode case-fold + NFKC normalize to match
                    // the DB's utf8mb4_unicode_ci comparison. Also strip
                    // zero-width chars that TrimSpace misses (\u200b etc).
                    normalized := stripZeroWidth(unicodeFold(e))
                    if len(normalized) > 255 {
                        normalized = normalized[:255]
                    }
                    return "pwreset-email:" + normalized
                }
            }
        }
    }
    return "ip:" + c.ClientIP()
}
```
- `unicodeFold`: use `golang.org/x/text/cases` / `unicode/norm.NFKC` (check `go.mod`; if `golang.org/x/text` isn't present, add it — it's already a transitive dep of most Go web projects).
- `stripZeroWidth`: remove `\u200b` (ZWSP), `\u200c` (ZWNJ), `\u200d` (ZWJ), `\ufeff` (BOM) — `strings.TrimSpace` does not catch these.
- Prefix is `pwreset-email:` (distinct from `account:`/`otp-session:`) so the bucket never collides with the login limiter.

## Related Code Files

- **Create:** `backend/internal/app/services/passwordreset/service.go`
- **Create:** `backend/internal/app/services/passwordreset/template.go`
- **Modify:** `backend/internal/app/services/user/password.go` — add **one** thin public delegator `HashNewPassword` (Red Team H8: only `hashPassword` is private; `ValidatePassword` is already exported, no delegator needed)
- **Modify:** `backend/internal/domain/email.go` — add `EmailKindPasswordReset` to the `EmailMessageKind` enum (Red Team Scope-3)
- **Modify:** `backend/internal/transport/http/handlers/auth.go` — add the two handler methods + the field/constructor param
- **Modify:** `backend/internal/transport/http/middleware/rate_limit.go` — add `CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter` (with unicode normalization)
- **Modify:** `backend/internal/app/bootstrap/services/init.go` — construct `passwordreset.Service` (Red Team C4: pass existing `otpEmailSender` as the email sender), add field `EmailPasswordReset` to `Services` struct (**Red Team M2**: do NOT name it `PasswordReset` — collides with existing `PasswordResetJobManager` at `init.go:60`)
- **Modify:** `backend/internal/app/bootstrap/container.go` — wire into `AuthHandler` constructor; add `PasswordResetRateLimit` to middleware; expose `PasswordReset.Enabled` for the route gate
- **Modify:** `backend/internal/app/bootstrap/routes_auth.go` — add the two routes behind `if cfg.PasswordReset.Enabled` (Red Team H6)
- **Modify:** `backend/internal/app/bootstrap/middleware.go` — construct the new limiter (mirrors how `LoginRateLimit` is built)

## Implementation Steps

1. **Write `passwordreset/template.go`** first (no deps):
   - `BuildResetEmailMessage(...)` following the OTP template structure. Use a CTA `<a>` button styled with the brand green. Subject `"Đặt lại mật khẩu TingTing"`. Body text in Vietnamese. Build the link as `resetURL + "?token=" + token` (URL-encode the token — base64url has no chars needing encoding, but be safe).
   - Call `msg.Validate()` before returning, like `BuildOTPEmailMessage` does.

2. **Write `passwordreset/service.go`**:
   - Define the struct + `NewService(...)` constructor per the Architecture section. Inject `*user.UserService` directly (Red Team H8).
   - `RequestReset(ctx, email string) error`:
     - `user, err := s.userRepo.GetByEmail(ctx, email)`.
     - **If `err != nil` (not found or DB error)** → perform a **dummy `tokenStore.Create` against a throwaway key** (Red Team H2 timing equalization) so the not-found path has the same Redis-RTT + hashing profile as the found path, then **return nil**. Log at debug level only. (Red Team Assumption-1: `GetByEmail` returns distinct NotFound vs InternalError kinds — treat both as "act like not-found" here to preserve anti-enumeration; a transient DB blip should not leak.)
     - ~~If `user.Password == ""` → Google-only account~~ **(Red Team C2: REMOVED — dead code. `users.password` is NOT NULL per migration 001, and `LoginWithGoogle` only reads users by email, never creates them. The check can never fire. Do not add it.)**
     - `token, err := s.tokenStore.Create(ctx, user.ID)`. On error, log and **return nil** (don't leak via 500).
     - Fire goroutine with **`defer recover()`** (Red Team H4):
       ```go
       go func() {
           defer func() {
               if r := recover(); r != nil {
                   s.logger.Error("password reset email goroutine panicked", "panic", r, "user_id", user.ID)
               }
           }()
           bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
           defer cancel()
           msg, err := BuildResetEmailMessage(token, *user.Email, user.Fullname, s.fromEmail, s.resetURL)
           if err != nil { s.logger.Error("build reset email failed", "error", err); return }
           if _, err := s.emailSender.Send(bg, msg); err != nil {
               s.logger.Error("send reset email failed", "error", err, "user_id", user.ID)
           }
       }()
       ```
     - Return `nil`.
   - `ConfirmReset(ctx, token, newPassword string) error`:
     - `userID, err := s.tokenStore.Consume(ctx, token)`.
     - **Red Team H5 (Redis error mapping):** If `err` is `ErrPasswordResetTokenNotFound` → return `domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)` (401, honest "invalid/expired"). If `err` is `ErrPasswordResetStoreUnavailable` (Redis down) → return `domain.NewInternalError(constants.MsgPasswordResetStoreUnavailableVN, err)` (500 — do NOT tell the user their valid token is expired).
     - `user, err := s.userRepo.GetByID(ctx, userID)`. If not found → return `domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)` (token pointed at a deleted user).
     - `if err := s.userService.ValidatePassword(newPassword); err != nil` → return `domain.NewValidationError(err.Error())` (Red Team H8: call the already-exported method directly).
     - `hashed, err := s.userService.HashNewPassword(newPassword)`; on err → `domain.NewInternalError(...)` (Red Team H8: the one delegator).
     - **Red Team C1 (transactional atomicity):** `s.userRepo.UpdatePasswordAndInvalidateSessions(ctx, user.ID, hashed, s.clock.Now())` — single DB transaction, NOT two separate `Update` + `UpdateTokensInvalidBefore` calls. If this fails, the token is already consumed (GETDEL) but the account state is unchanged; the user requests a new link. That's far less bad than a half-applied state (password changed but sessions alive).
     - Publish `domain.NewPasswordChangedEvent(ctx, user.ID, user.Username, "email_reset", user.ID, "")` — **Red Team Security-5: `actorUserID = user.ID`** (self-service: the user is the actor, matching `NewUserLogoutEvent`'s zero-actor fix at `event_factory_user.go:136`; this keeps the audit trail forensically useful). Best-effort, log on failure.
     - Return `nil`.

3. **Add `HashNewPassword` delegator to `user/password.go`** (Red Team H8 — only ONE delegator; `ValidatePassword` is already exported):
   ```go
   // HashNewPassword exposes the internal hasher for use by passwordreset.Service.
   // (ValidatePassword is already exported and called directly — no delegator needed.)
   func (s *UserService) HashNewPassword(password string) (string, error) {
       return s.hashPassword(password)
   }
   ```

4. **Wire the handler** (`auth.go`): add field `emailPasswordReset *passwordreset.Service`, update `NewAuthHandler` signature, add the two methods. Update the **single** call site in `container.go:386` (`handlers.NewAuthHandler(...)`).

5. **Add the rate limiter** (`rate_limit.go`): `CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter`. Construct it in `middleware.go` alongside the other limiters (read `cfg.PasswordReset.RateLimitPerHour`).

6. **Construct the service** in `services/init.go`, near the `otpService` construction (~line 613). **Red Team C4:** pass the **existing `otpEmailSender` variable** (resolved at `init.go:607-611`) as the email sender — do NOT re-instantiate `email.NewResendProvider`. That variable already resolves Sandbox (dev) vs Resend (prod/OTP-enabled) correctly.
   ```go
   pwresetTokenStore := cache.NewPasswordResetTokenStore(redis.Client, cfg.PasswordReset.TokenTTL)
   emailPasswordResetService := passwordreset.NewService(
       pwresetTokenStore, repos.User, userService, otpEmailSender,   // ← reuse otpEmailSender
       constants.DefaultEmailSenderAddress, cfg.PasswordReset.ResetURL, clk, logger,
   )
   ```
   - **Red Team M2:** Add field `EmailPasswordReset *passwordreset.Service` to the `Services` struct — do NOT name it `PasswordReset` (collides with existing `PasswordResetJobManager *user.PasswordResetJobManager` at `init.go:60`).

7. **Add the routes** in `routes_auth.go` behind `if cfg.PasswordReset.Enabled` (Red Team H6, see Routes section). The middleware field name: `container.Middleware.PasswordResetRateLimit`.

8. **Compile + smoke:** `cd backend && go build ./...`. Then manually `curl -X POST localhost:8080/api/v1/auth/password-reset/request -d '{"email":"nonexistent@example.com"}'` → expect 200 with the generic message.

## Success Criteria

- [ ] `passwordreset.Service` implements `RequestReset` (always-generic) and `ConfirmReset` (full validation + session kill).
- [ ] The magic-link email uses the branded banner and a visible CTA button.
- [ ] Both routes are public (no auth middleware) and rate-limited per-email.
- [ ] `AuthHandler` constructor compiles with the new dependency; no other call sites break.
- [ ] `cd backend && go build ./...` passes.
- [ ] Manual `curl` of the request endpoint returns 200 for both known and unknown emails with an identical body.

## Risk Assessment

- **Forgetting to update `NewAuthHandler` call site:** The only call is in `container.go:386`. A compile error will catch it immediately.
- **Email send blocking the request:** Mitigated by the goroutine + 15s background ctx + **`defer recover()`** (Red Team H4 — a panic in `Send` no longer crashes the process).
- **`GetByEmail` semantics:** Returns an error (not nil user) when not found — confirmed by reading `internal/domain/user.go:68` interface + the impl at `user_repository.go:107-117`. The service treats both NotFound and transient DB errors as "act like not-found" to preserve anti-enumeration (Red Team Assumption-1).
- **Red Team M3 (corrected):** This flow is NOT "mirroring `ChangePasswordAndBlacklistToken`." The existing `ChangePassword` (`user/auth.go:40-90`) does NOT call `UpdateTokensInvalidBefore` — it only blacklists the current JTI via `blacklistedTokenRepo.BlacklistToken` (`auth_service.go:551-570`). The reset flow introduces session-wide invalidation on a public, token-gated endpoint for the FIRST time. `tokens_invalid_before` is enforced inside `AuthService.ValidateToken` (`auth_service.go:411`), not in middleware. The transactional guarantee in C1 is therefore essential — there is no existing safety net to lean on.
- **Token-in-URL leakage (Red Team H3):** Mitigated on the frontend (Phase 4) by stripping the token from the URL on mount + `Referrer-Policy: no-referrer` meta tag. Residual risk (proxy logs, inbox screenshots) documented and accepted given the 30-min single-use TTL.
