---
phase: 3
title: "Backend: Handler, Routes, Rate-Limit & Wiring"
status: pending
priority: P1
dependencies: ["1", "2"]
effort: "S"
---

# Phase 3: Backend: Handler, Routes, Rate-Limit & Wiring

> **Scope note:** This phase consolidates the CLI-scaffolded phases 4 (Routes) and 5
> (Rate-Limit & Wiring). Those stub files are kept only to preserve the CLI phase-id
> sequence; their bodies point here and contain **no independent work**. Do not split
> this across PRs — handler + route + rate-limit + bootstrap wiring are one coherent
> change and share a single red-team surface (anti-enumeration, body-restore).

## Overview

Expose the Phase 2 service as two unauthenticated HTTP endpoints, gate them with a
per-mobile rate limiter (body-restore pattern, identical contract to
`CreatePasswordResetRateLimit`), wire the service into the bootstrap container, and add
the fail-fast env validation. Mirrors the email reset's handler shape at
`transport/http/handlers/auth.go:170-220` — the anti-enumeration and error-mapping
behavior is copied verbatim, only the DTO and service call differ.

## Requirements

- **Functional**
  - `POST /api/v1/auth/zalo-reset/request` body `{ "mobile": "..." }` → always 200 with `MsgZaloResetRequestedVN`. Never reveals mobile existence.
  - `POST /api/v1/auth/zalo-reset/confirm` body `{ "otp_session_id": "...", "code": "123456", "new_password": "..." }` → 200 on success, 401 on wrong/expired code, 400 on validation, 500 on store-down.
  - `POST /api/v1/auth/zalo-reset/resend` body `{ "otp_session_id": "..." }` → 200 with new session id, 429 on cooldown.
- **Non-functional**
  - All three endpoints are **public** (no `authMiddleware`) — they exist precisely because the user cannot authenticate. They sit in the same route group as `/auth/login`, `/auth/password-reset/*`.
  - Rate limit keyed on **normalized mobile** (digits only, leading-0/`+84` collapsed) for `/request`, and on `otp_session_id` for `/confirm` and `/resend` — exactly how the OTP login limiter keys `/login/verify` on `otp_session_id` (see `middleware/rate_limit.go:119-121`).
  - Body-restore after reading in the key-getter — copy `passwordResetEmailKeyGetter`'s `c.Request.Body = io.NopCloser(bytes.NewReader(raw))` or the downstream `BindJSON` 400s every request.

## Architecture

### Handler additions (`transport/http/handlers/auth.go`)

Extend `AuthHandler` with a `zaloResetService *zaloreset.Service` field and constructor
param. Add three methods next to the existing `RequestPasswordReset`/`ConfirmPasswordReset`:

```go
// @Summary Request Zalo OTP password reset
// @Description Send a 6-digit OTP via Zalo ZNS to the employee's mobile. Always 200.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ZaloResetRequestDTO true "Mobile number"
// @Success 200 {object} response.SuccessResponse
// @Router /auth/zalo-reset/request [post]
func (h *AuthHandler) RequestZaloReset(c *gin.Context) {
    var req dto.ZaloResetRequestDTO
    if !helpers.BindJSON(c, &req) { return }
    // Always 200. Service returns nil regardless of mobile existence.
    _ = h.zaloResetService.RequestReset(c.Request.Context(), req.Mobile)
    response.Success(c, gin.H{
        "otp_session_id": "", // placeholder — populated only for the dummy path so the
                              // shape is stable; real session id is opaque and we DO NOT
                              // echo it here for not-found mobiles (would leak existence)
    }, constants.MsgZaloResetRequestedVN)
}
```

> **Anti-enumeration subtlety:** the email reset does **not** return any per-request
> value, because a not-found email has no token to echo. The Zalo flow needs to return a
> `session_id` to the client so the user can submit `/confirm`. **Resolution:** always
> generate a session_id (even for not-found — store.CreateDummy returns a real, opaque,
> but userID-less id whose Consume always returns `ErrSessionNotFound`). The client sees
> an identical `{otp_session_id}` for known and unknown mobiles. The Phase 2 store's
> `CreateDummy` therefore must return a `(sessionID string, err error)` whose sessionID
> is structurally indistinguishable from a real one. **This is a Phase 2 contract
> adjustment — flag it back to Phase 2 if not already implemented that way.**

```go
// @Summary Confirm Zalo OTP password reset
// @Router /auth/zalo-reset/confirm [post]
func (h *AuthHandler) ConfirmZaloReset(c *gin.Context) {
    var req dto.ZaloResetConfirmDTO
    if !helpers.BindJSON(c, &req) { return }
    if err := h.zaloResetService.ConfirmReset(c.Request.Context(),
        req.OTPSessionID, req.Code, req.NewPassword); err != nil {
        response.HandleDomainError(c, err)
        return
    }
    response.Success(c, nil, constants.MsgZaloResetSuccessVN)
}
```

### DTOs (`transport/http/dto/`)

```go
type ZaloResetRequestDTO struct {
    Mobile string `json:"mobile" binding:"required"`
}
type ZaloResetConfirmDTO struct {
    OTPSessionID string `json:"otp_session_id" binding:"required"`
    Code         string `json:"code" binding:"required,len=6"`
    NewPassword  string `json:"new_password" binding:"required,min=8"`
}
type ZaloResetResendDTO struct {
    OTPSessionID string `json:"otp_session_id" binding:"required"`
}
```

> `binding:"len=6"` on `Code` rejects non-6-char input before it reaches the service
> (cheap pre-filter). Real validation is the SHA compare in the Lua Consume.

### Routes (`transport/http/router.go` — wherever the existing `/auth/password-reset/*` group lives)

```go
authGroup.POST("/auth/zalo-reset/request",
    middleware.CreateZaloResetRateLimit(redisURL, cfg.Zalo.RatePerHour),
    authHandler.RequestZaloReset)
authGroup.POST("/auth/zalo-reset/confirm",
    middleware.CreateZaloResetConfirmRateLimit(redisURL),
    authHandler.ConfirmZaloReset)
authGroup.POST("/auth/zalo-reset/resend",
    middleware.CreateZaloResetConfirmRateLimit(redisURL),
    authHandler.ResendZaloReset)
```

All three are **public** (same group as `/auth/login`, no `Authorize` middleware).

### Rate limit (`transport/http/middleware/rate_limit.go`)

Add two constructors alongside `CreatePasswordResetRateLimit`:

```go
// CreateZaloResetRateLimit caps /auth/zalo-reset/request per normalized MOBILE number
// (read from the body), falling back to client IP. Mirrors CreatePasswordResetRateLimit.
func CreateZaloResetRateLimit(redisURL string, perHour int) gin.HandlerFunc {
    if perHour <= 0 { perHour = 3 }
    return createRateLimiterWithKey(RateLimitConfig{
        Rate: fmt.Sprintf("%d-H", perHour), RedisURL: redisURL,
    }, zaloResetMobileKeyGetter)
}

// CreateZaloResetConfirmRateLimit caps /confirm and /resend per otp_session_id.
func CreateZaloResetConfirmRateLimit(redisURL string) gin.HandlerFunc {
    return createRateLimiterWithKey(RateLimitConfig{
        Rate: "10-H", RedisURL: redisURL, // 10 confirm+resend attempts/hour per session
    }, zaloResetSessionKeyGetter)
}
```

**`zaloResetMobileKeyGetter`** — body-restore (CRITICAL, port the email getter's
restore-or-BindJSON-400s comment), then return `"zreset-mobile:" + normalizePhoneForRL(mobile)`.
Normalization: strip non-digits, strip leading `0` or `84`, prefix `m`. This collapses
`0987...`, `+8498...`, `8498...`, `"0987 654 321"` to one bucket — same spirit as the
email getter's NFKC fold but for phone digits.

**`zaloResetSessionKeyGetter`** — read `otp_session_id` from the body (same restore
pattern), return `"zreset-sid:" + sessionID`.

### Bootstrap wiring (`app/bootstrap/services/init.go`, near line 621)

After `emailPasswordResetService := passwordreset.NewService(...)` and after Phase 4's
`zaloConnectSvc` + `zaloProvider` are constructed:

```go
// ZNS-based employee password reset (Phase 2/3). The enabled-check is the live
// admin toggle (Phase 4's zaloconnect.Service), NOT the boot-time env flag.
var zaloResetService *zaloreset.Service
zaloResetStore := cache.NewZaloResetStore(redis.Client, zaloCodeTTL) // 10m
zaloResetService = zaloreset.NewService(
    zaloResetStore, repos.User, userService, zaloProvider,
    zaloConnectSvc.TemplateID(ctx),   // read from Settings (default "617976"); or pass a getter
    zaloCodeTTL,
    zaloConnectSvc,                   // EnabledChecker (hot-read of zalo.enabled)
    eventBus, clk, logger,
)
```

Then add `ZaloPasswordReset *zaloreset.Service` to `Services` and pass it to
`NewAuthHandler(authService, userService, passwordResetService, zaloResetService)`.

> **Constructor signature change is fine** — `NewAuthHandler` is called from exactly one
> place (the bootstrap), and its existing tests will need the new arg (provide a Noop).

## Related Code Files

- **Modify:**
  - `backend/internal/transport/http/handlers/auth.go` — 3 methods + constructor field.
  - `backend/internal/transport/http/dto/` (or wherever `PasswordResetRequestDTO` lives) — add 3 DTOs.
  - `backend/internal/transport/http/router.go` — 3 routes.
  - `backend/internal/transport/http/middleware/rate_limit.go` — 2 constructors + 2 key-getters + 1 phone-norm helper.
  - `backend/internal/app/bootstrap/services/init.go` — wire `ZaloPasswordReset` into `Services` + `NewAuthHandler`.
- **Reference (read-only):**
  - `transport/http/handlers/auth.go:177-219` — `RequestPasswordReset` / `ConfirmPasswordReset` to copy the response/error-mapping shape.
  - `middleware/rate_limit.go:191-243` — `CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter` + the body-restore comment.

## Implementation Steps

1. **DTOs** — add the three structs; verify `binding:"len=6"` rejects `"12345"` and `"1234567"` at the BindJSON layer.
2. **Rate-limit getters** — `zaloResetMobileKeyGetter` (with body-restore) and `zaloResetSessionKeyGetter`. Unit-test the phone normalization for the same set as Phase 1's `NormalizePhone`, plus verify body is restorable (a second `c.Request.Body` read in a fake handler returns the original JSON).
3. **Handler methods** — three methods; assert the service is called with the right args and the response shape is `{otp_session_id, message}` on `/request`.
4. **Routes** — register all three as public. Verify with `curl` locally that `/request` returns 200 for both a known and an unknown mobile (no auth header).
5. **Bootstrap wiring** — add the field, the conditional construction, the Noop fallback, and update `NewAuthHandler`. Update `auth_test.go` callers to pass the Noop.
6. **Local verify** — `cd backend && go build ./... && go test ./internal/transport/... ./internal/app/bootstrap/... -race`.

## Success Criteria

- [ ] `/request` returns 200 + `{otp_session_id: <43-char-opaque>, message: MsgZaloResetRequestedVN}` for a known employee mobile **and** an unknown mobile — byte-identical response shape.
- [ ] `/request` for an admin mobile returns the same generic 200 (role gate, no ZNS send).
- [ ] `/confirm` with a wrong code → 401 `MsgZaloResetCodeInvalidVN`; with an expired/consumed session → 401; with a weak password → 400 validation.
- [ ] `/confirm` happy path → 200 `MsgZaloResetSuccessVN`; the user's password is changed and prior JWTs are invalid (`tokens_invalid_before` set).
- [ ] The rate limiter blocks the 4th `/request` per mobile per hour with 429 (or the existing limiter's too-many-requests shape).
- [ ] `BindJSON` works after the key-getter has read the body (regression test for the body-restore bug the email getter documents).
- [ ] `NewAuthHandler` compiles with the new param; existing `auth_test.go` updated and green.

## Risk Assessment

- **R-Z8 (session_id leak via timing):** Returning a `session_id` for an unknown mobile is necessary for shape-parity, but a real session_id encodes a Redis key whose TTL was set by a `Create(userID)` that ran for known mobiles. If `Create` is slower than `CreateDummy`, the response time leaks existence. **Mitigation:** Phase 2's `CreateDummy` must do the same Redis `SET ... PX` work (just with a sentinel userID); the email reset's H2 timing defense is the precedent. Phase 6 includes a timing assertion.
- **R-Z9 (DoS via `/confirm` without rate limit):** An attacker with a harvested `session_id` could brute-force 6-digit codes (10⁶ space). The 10/hour/session limiter bounds this to 10 guesses — safe. **Verify** the session-key getter is applied to `/confirm`, not just `/resend`.
- **R-Z10 (constructor churn):** Adding a param to `NewAuthHandler` touches its test surface. Low risk; isolated to bootstrap + auth tests. Done first in the phase to unblock the rest.
