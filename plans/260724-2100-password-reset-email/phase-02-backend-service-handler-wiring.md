---
phase: 2
title: "Backend: Service & Handler & Wiring"
status: pending
priority: P1
dependencies: [1]
---

# Phase 2: Backend: Service & Handler & Wiring

## Overview

Compose the Phase 1 building blocks into a `PasswordResetService`, wire it into the auth handler, add two public routes with a dedicated per-email rate limiter, and write the branded Vietnamese reset-link email template.

## Requirements

- **Functional:**
  - `RequestReset(ctx, email)`: look up user; if found and has a password and `PasswordResetConfig.Enabled`, create a token, email a magic link async. **Always return nil** (generic success) regardless of whether the email exists — the handler maps nil → 200.
  - `ConfirmReset(ctx, token, newPassword)`: atomically consume the token, fetch the user, validate password strength, hash + update, invalidate all sessions (`UpdateTokensInvalidBefore(now)`), publish `PasswordChangedEvent` with `method="email_reset"`.
- **Non-functional:**
  - Email send is fire-and-forget in a goroutine (15s background ctx) so the request returns immediately — mirrors `OTPService.StartLogin`.
  - All user-facing errors are `domain.NewValidationError` / `domain.NewUnauthorizedError` / `domain.NewInternalError` so the existing `response.HandleDomainError` maps them to correct HTTP codes.

## Architecture

### Service struct

```go
package passwordreset

type Service struct {
    tokenStore       *cache.PasswordResetTokenStore
    userRepo         domain.UserRepository
    emailSender      EmailSender          // = domain.EmailDeliveryPort (ResendProvider)
    passwordHasher   PasswordHasher       // narrow port over *user.UserService
    passwordValidator PasswordValidator   // narrow port over *user.UserService
    fromEmail        string               // constants.DefaultEmailSenderAddress
    resetURL         string               // e.g. "https://tingting.vip/reset-password"
    clk              clock.Clock
    logger           *slog.Logger
}

// Narrow ports so the service is testable without the full UserService.
type EmailSender interface {
    Send(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error)
}
type PasswordHasher interface {
    HashPassword(password string) (string, error)
}
type PasswordValidator interface {
    ValidatePassword(password string) error
}
```

**Adapters:** `*user.UserService` already implements `HashPassword` (private) — so add **two thin public methods** on `UserService`:
- `HashNewPassword(password string) (string, error)` — wraps the private `hashPassword`.
- `ValidatePasswordRules(password string) error` — wraps `ValidatePassword`.

These are trivial delegators (one-line) but keep `PasswordResetService` decoupled from `UserService` internals. Alternative: pass `*user.UserService` directly and add an interface above it. Chosen approach: thin delegators, because the rest of the codebase already reaches into `UserService` this way (e.g. `AuthService.userService.VerifyPasswordHash`).

### Email template

A new `template.go` in the `passwordreset` package, structurally identical to `otp/template.go`:
- `BuildResetEmailMessage(token, recipientEmail, recipientName, fromEmail, resetURL string) (*domain.EmailMessage, error)`.
- Vietnamese subject: `"Đặt lại mật khẩu TingTing"`.
- Body: a green CTA button linking to `{resetURL}?token={token}`, plus the 30-min expiry notice and the "if you didn't request this, ignore this email" line.
- Reuses the public banner image URL (`https://tingting.vip/email-banner.jpg?v=20260709`) — same as the OTP template.
- `Kind: domain.EmailKindGeneric` (no new email kind needed; reset emails are not differentiated for observability in v1). Add a `Metadata["purpose"] = "password_reset"` tag for log/filter use.

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

In `backend/internal/app/bootstrap/routes_auth.go`, inside the `auth := v1.Group("/auth")` block (the **public** block, before `auth.Use(...Authenticate)`):

```go
auth.POST("/password-reset/request", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.RequestPasswordReset)
auth.POST("/password-reset/confirm", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.ConfirmPasswordReset)
```

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

func passwordResetEmailKeyGetter(c *gin.Context) string {
    // Read body (capped at 4 KiB), restore it, extract JSON "email".
    // Mirror loginAccountKeyGetter exactly. Fall back to "ip:" + c.ClientIP().
}
```

## Related Code Files

- **Create:** `backend/internal/app/services/passwordreset/service.go`
- **Create:** `backend/internal/app/services/passwordreset/template.go`
- **Modify:** `backend/internal/app/services/user/password.go` — add `HashNewPassword` + `ValidatePasswordRules` thin public delegators
- **Modify:** `backend/internal/transport/http/handlers/auth.go` — add the two handler methods + the field/constructor param
- **Modify:** `backend/internal/transport/http/middleware/rate_limit.go` — add `CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter`
- **Modify:** `backend/internal/app/bootstrap/services/init.go` — construct `passwordreset.Service`, add to `Services` struct
- **Modify:** `backend/internal/app/bootstrap/container.go` — wire into `AuthHandler` constructor; add `PasswordResetRateLimit` to middleware
- **Modify:** `backend/internal/app/bootstrap/routes_auth.go` — add the two routes
- **Modify:** `backend/internal/app/bootstrap/middleware.go` — construct the new limiter (mirrors how `LoginRateLimit` is built)

## Implementation Steps

1. **Write `passwordreset/template.go`** first (no deps):
   - `BuildResetEmailMessage(...)` following the OTP template structure. Use a CTA `<a>` button styled with the brand green. Subject `"Đặt lại mật khẩu TingTing"`. Body text in Vietnamese. Build the link as `resetURL + "?token=" + token` (URL-encode the token — base64url has no chars needing encoding, but be safe).
   - Call `msg.Validate()` before returning, like `BuildOTPEmailMessage` does.

2. **Write `passwordreset/service.go`**:
   - Define the struct + narrow ports + `NewService(...)` constructor.
   - `RequestReset(ctx, email string) error`:
     - `user, err := s.userRepo.GetByEmail(ctx, email)`. If `err != nil` (not found) → **return nil** (generic success, no email sent). Log at debug level only.
     - If `user.Password == ""` → Google-only account → **return nil** (no email, no error).
     - If `!s.enabled` → return `nil` (feature off; handler still 200s with the generic message — but actually the route shouldn't be hit; belt-and-suspenders).
     - `token, err := s.tokenStore.Create(ctx, user.ID)`. On error, log and **return nil** (don't leak via 500).
     - Fire goroutine: `go func() { ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second); defer cancel(); msg, _ := BuildResetEmailMessage(token, *user.Email, user.Fullname, s.fromEmail, s.resetURL); s.emailSender.Send(ctx, msg) }()`. Log errors, don't surface.
     - Return `nil`.
   - `ConfirmReset(ctx, token, newPassword string) error`:
     - If `!s.enabled` → return `domain.NewValidationError(constants.MsgPasswordResetDisabledVN)`.
     - `userID, err := s.tokenStore.Consume(ctx, token)`. If `ErrPasswordResetTokenNotFound` → return `domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)`.
     - `user, err := s.userRepo.GetByID(ctx, userID)`. If not found → return `domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)` (token pointed at a deleted user; treat as invalid).
     - `if err := s.passwordValidator.ValidatePassword(newPassword); err != nil` → return `domain.NewValidationError(err.Error())`.
     - `hashed, err := s.passwordHasher.HashPassword(newPassword)`; on err → `domain.NewInternalError(...)`.
     - `user.Password = hashed; s.userRepo.Update(ctx, user)`.
     - `s.userRepo.UpdateTokensInvalidBefore(ctx, user.ID, s.clock.Now())` — **kills all JWTs**.
     - Publish `domain.NewPasswordChangedEvent(ctx, user.ID, user.Username, "email_reset", 0, "")` (actorUserID=0 since this is anonymous; actorFullName empty). Best-effort, log on failure.
     - Return `nil`.

3. **Add `HashNewPassword` + `ValidatePasswordRules` to `user/password.go`**:
   ```go
   // HashNewPassword exposes the internal hasher for use by passwordreset.Service.
   func (s *UserService) HashNewPassword(password string) (string, error) {
       return s.hashPassword(password)
   }
   // ValidatePasswordRules exposes the validator for use by passwordreset.Service.
   func (s *UserService) ValidatePasswordRules(password string) error {
       return s.ValidatePassword(password)
   }
   ```

4. **Wire the handler** (`auth.go`): add field, update `NewAuthHandler` signature, add the two methods. Update the **single** call site in `container.go` (`handlers.NewAuthHandler(...)`).

5. **Add the rate limiter** (`rate_limit.go`): `CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter`. Construct it in `middleware.go` alongside the other limiters (read `cfg.PasswordReset.RateLimitPerHour`).

6. **Construct the service** in `services/init.go`, near the `otpService` construction (~line 613). It needs:
   - The Redis-backed token store: `cache.NewPasswordResetTokenStore(redis.Client, cfg.PasswordReset.TokenTTL)`.
   - `repos.User`, the email sender (`emailProvider` is already in scope — but use `email.NewResendProvider(cfg.Notification.ResendAPIKey)` if production, same gating logic as OTP, so reset emails reach real inboxes even in dev sandbox mode), `userService` (for the hasher/validator ports), `constants.DefaultEmailSenderAddress`, a reset URL from config (add `PASSWORD_RESET_URL` env, default `https://tingting.vip/reset-password`), `clk`, `logger`.
   - Add field `PasswordReset *passwordreset.Service` to the `Services` struct.

7. **Add the routes** in `routes_auth.go` (see Architecture). The middleware field name: `container.Middleware.PasswordResetRateLimit`.

8. **Compile + smoke:** `cd backend && go build ./...`. Then manually `curl -X POST localhost:8080/api/v1/auth/password-reset/request -d '{"email":"nonexistent@example.com"}'` → expect 200 with the generic message.

## Success Criteria

- [ ] `passwordreset.Service` implements `RequestReset` (always-generic) and `ConfirmReset` (full validation + session kill).
- [ ] The magic-link email uses the branded banner and a visible CTA button.
- [ ] Both routes are public (no auth middleware) and rate-limited per-email.
- [ ] `AuthHandler` constructor compiles with the new dependency; no other call sites break.
- [ ] `cd backend && go build ./...` passes.
- [ ] Manual `curl` of the request endpoint returns 200 for both known and unknown emails with an identical body.

## Risk Assessment

- **Forgetting to update `NewAuthHandler` call site:** The only call is in `container.go`. A compile error will catch it immediately.
- **Email send blocking the request:** Mitigated by the goroutine + 15s background ctx, exactly as `OTPService.StartLogin` does.
- **`GetByEmail` semantics:** Returns an error (not nil user) when not found — confirmed by reading `internal/domain/user.go` interface + existing usage in `crud.go`. The service must `return nil` on that error path.
- **Session invalidation scope:** `UpdateTokensInvalidBefore` is already enforced by the auth middleware (reads `user.TokensInvalidBefore`). No new middleware logic needed.
