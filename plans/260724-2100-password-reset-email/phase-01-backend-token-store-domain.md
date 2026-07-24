---
phase: 1
title: "Backend: Token Store & Domain"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Backend: Token Store & Domain

## Overview

Build the foundational layer: a Redis-backed single-use token store, the DTOs, config struct, and message constants. No business logic yet — just the building blocks Phase 2 composes.

## Requirements

- **Functional:** A store that can create a high-entropy reset token (mapped to a user ID), look it up by its SHA-256 hash, and atomically consume (read-then-delete) it.
- **Non-functional:** Tokens are 256-bit entropy (32 random bytes, base64url-encoded ≈ 43 chars). Stored as SHA-256 hash so a Redis dump doesn't leak usable tokens. TTL enforced by Redis `PX` argument (no cleanup cron needed). Atomic consume via Lua `GETDEL` to eliminate the double-use race.

## Architecture

### Token lifecycle

```
Create(userID) ──▶ generate 32 random bytes
                ──▶ base64url(token) = "Ab3x...43 chars"   (returned to caller, emailed to user)
                ──▶ sha256(token) = [32]byte               (stored in Redis under key pwreset:<hexhash>)
                ──▶ SET pwreset:<hexhash> <userID> PX 1800000
                ──▶ return token string

Consume(token) ──▶ sha256(token) = [32]byte
                ──▶ GETDEL pwreset:<hexhash>                (Lua: atomic read + delete)
                ──▶ if nil → expired/unknown/already-used
                ──▶ else → parse userID, return it
```

**Key naming:** `pwreset:<sha256-hex>` — consistent with `otp:pending:*` / `otp:user:*` namespace convention in `internal/infra/cache/otp_pending_store.go`.

**Why GETDEL and not GET+DEL:** Two concurrent confirm requests with the same token (e.g. user double-clicks, or an attacker replays) could both `GET` the token before either `DEL`etes, then both reset the password. `GETDEL` (Redis 6.2+, available on Redis 7 / Redis Stack — the project uses Redis 7 per `docker-compose.yml`) is a single atomic operation: only one caller receives the value.

### Config

```go
// PasswordResetConfig backs the self-service email password reset feature.
type PasswordResetConfig struct {
    Enabled          bool          // PASSWORD_RESET_ENABLE (default true — low-risk self-service)
    TokenTTL         time.Duration // PASSWORD_RESET_TOKEN_TTL (default 30m)
    RateLimitPerHour int           // PASSWORD_RESET_RATE_LIMIT (default 3)
    ResetURL         string        // PASSWORD_RESET_URL (default "https://tingting.vip/reset-password")
}
```

### DTOs

```go
// PasswordResetRequestDTO — the "I forgot my password" form body.
type PasswordResetRequestDTO struct {
    Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirmDTO — the "set new password" form body.
type PasswordResetConfirmDTO struct {
    Token       string `json:"token" binding:"required"`
    NewPassword string `json:"new_password" binding:"required,min=8"`
}
```

## Related Code Files

- **Create:** `backend/internal/infra/cache/password_reset_token_store.go`
- **Create:** `backend/internal/infra/cache/password_reset_token_store_test.go` (defer test bodies to Phase 3, but create the file shell now if convenient)
- **Modify:** `backend/internal/config/config.go` — add `PasswordResetConfig` struct + env parsing
- **Modify:** `backend/internal/app/dto/user.go` — add the two DTOs
- **Modify:** `backend/internal/constants/messages.go` — add VN message constants

## Implementation Steps

1. **Create `password_reset_token_store.go`** in `backend/internal/infra/cache/`:
   - Define `PasswordResetTokenStore` struct holding `*redis.Client` + `ttl time.Duration`.
   - `NewPasswordResetTokenStore(client *redis.Client, ttl time.Duration) *PasswordResetTokenStore` — default TTL 30m if `ttl <= 0`.
   - `Create(ctx, userID uint) (token string, err error)`:
     - `crypto/rand.Read(32 bytes)` → base64url (no padding) → `token`.
     - `sha256.Sum256(token)` → hex string → `hashHex`.
     - `SET pwreset:<hashHex> <userID> PX <ttl.ms>`.
     - Return `token`.
   - `Consume(ctx, token string) (userID uint, err error)`:
     - `sha256.Sum256(token)` → `hashHex`.
     - Lua script: `return redis.call("GETDEL", KEYS[1])` with `KEYS[1] = pwreset:<hashHex>`.
     - If result is nil (`redis.Nil` or empty) → return `ErrPasswordResetTokenNotFound`.
     - Else parse `strconv.ParseUint(result, 10, 64)` → `uint` → return.
   - `Delete(ctx, token string) error` — non-atomic delete for admin/teardown use; same hash path.
   - Define sentinel errors: `ErrPasswordResetTokenNotFound = errors.New("password reset token not found or expired")`.
   - Mirrors the package-level comment style of `otp_pending_store.go` (package doc explaining purpose + RT notes).

2. **Modify `backend/internal/config/config.go`**:
   - Add `PasswordReset PasswordResetConfig` field to the main `Config` struct (near `OTP OTPConfig`).
   - Add the `PasswordResetConfig` struct definition (see Architecture above).
   - In `Load()` / the env-parsing block, add:
     ```go
     PasswordReset: PasswordResetConfig{
         Enabled:          parseBool(getEnv("PASSWORD_RESET_ENABLE", "true")),
         TokenTTL:         parseDuration(getEnv("PASSWORD_RESET_TOKEN_TTL", "30m")),
         RateLimitPerHour: parseInt(getEnv("PASSWORD_RESET_RATE_LIMIT", "3")),
         ResetURL:         getEnv("PASSWORD_RESET_URL", "https://tingting.vip/reset-password"),
     },
     ```
   - Note: `parseBool`, `parseDuration`, `parseInt` helpers already exist in this file.

3. **Modify `backend/internal/app/dto/user.go`** — append the two DTOs near the existing password DTOs (after `ResetUserPasswordRequest`):
   ```go
   // PasswordResetRequestDTO is the body of POST /auth/password-reset/request.
   // The endpoint returns the SAME success response whether or not the email
   // exists, to prevent email enumeration.
   type PasswordResetRequestDTO struct {
       Email string `json:"email" binding:"required,email"`
   }

   // PasswordResetConfirmDTO is the body of POST /auth/password-reset/confirm.
   // Token is the opaque value from the magic-link query string.
   type PasswordResetConfirmDTO struct {
       Token       string `json:"token" binding:"required"`
       NewPassword string `json:"new_password" binding:"required,min=8"`
   }
   ```

4. **Modify `backend/internal/constants/messages.go`** — add to the auth/password section (near line 665-670):
   ```go
   // Password reset (self-service email flow)
   MsgPasswordResetRequestedVN          = "Nếu email tồn tại trong hệ thống, bạn sẽ nhận được hướng dẫn đặt lại mật khẩu trong vài phút."
   MsgPasswordResetSuccessVN            = "Đặt lại mật khẩu thành công. Vui lòng đăng nhập bằng mật khẩu mới."
   MsgPasswordResetTokenInvalidVN       = "Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn. Vui lòng yêu cầu liên kết mới."
   MsgPasswordResetDisabledVN           = "Tính năng đặt lại mật khẩu đang bị tắt. Vui lòng liên hệ quản trị viên."
   MsgPasswordResetRateLimitedVN        = "Bạn đã yêu cầu đặt lại mật khẩu quá nhiều lần. Vui lòng thử lại sau."
   MsgPasswordResetTokenMissingVN       = "Thiếu mã đặt lại mật khẩu."
   ```

## Success Criteria

- [ ] `password_reset_token_store.go` compiles, with `Create`, `Consume`, `Delete`, and `ErrPasswordResetTokenNotFound`.
- [ ] `Create` returns a base64url token of ~43 chars; the same input is **not** stored in Redis (only its SHA-256 hex).
- [ ] `Consume` is atomic: two concurrent calls with the same token return exactly one `userID` and one `ErrPasswordResetTokenNotFound`.
- [ ] `Config.PasswordReset` parses from env with the documented defaults.
- [ ] DTOs and message constants exist and are referenced (compile check).
- [ ] `cd backend && go build ./...` passes.

## Risk Assessment

- **Redis version:** `GETDEL` requires Redis ≥ 6.2. The project runs Redis 7 (`docker-compose.yml`). If an older Redis were in use, fall back to a `WATCH/MULTI/GET/DEL` Lua block — but that is not needed here.
- **Token length vs. hash collision:** 32-byte tokens have 256 bits of entropy; SHA-256 collisions are computationally infeasible. No collision-handling code is needed.
