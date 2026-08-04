---
phase: 1
title: "Backend: ZNS Provider"
status: pending
priority: P1
dependencies: []
effort: "M"
---

# Phase 1: Backend: ZNS Provider

## Overview

Port the proven ZNS sender from `tuyennhanvien.vn:wp-content/themes/vfic/inc/functions/functions-zns.php`
into a new Go package `backend/internal/infra/zalo/`. This phase delivers the **stateless
ZNS protocol layer**: phone normalization, param clamping, the `POST /message/template`
send with `-124` single-retry, the OAuth v4 token-refresh + code-exchange functions, and
the error-code map.

**What this phase does NOT own (moved to Phase 4):** credential persistence (the
`Settings`-table-backed credential store), the config struct, boot fail-fast, and the
admin connection service. Phase 1's `Provider` takes credentials + endpoints via
constructor args / a narrow interface, so it is purely a stateless HTTP client — trivially
unit-testable with `httptest.Server` and decoupled from the DB. Phase 4 wires the
DB-backed credential source into it.

This keeps the dependency graph honest: Phase 1 (protocol) ← Phase 2 (service) ← Phase 3
(public handler), and independently Phase 1 ← Phase 4 (admin connection) ← Phase 5 (admin
UI). Phases 2/3 and 4/5 can proceed in parallel once Phase 1 lands.

## Requirements

- **Functional**
  - `NormalizePhone(raw string) string` — strips non-digits, rewrites domestic/`+84`/`84` forms to `84` + 9 digits, returns `""` on mismatch (port of `vfic_zns_normalize_phone`).
  - `ClampParams(templateID, data map[string]string) map[string]string` — per-param cap from a static registry; the `password_reset` (`617976`) entry caps `otp_code`=30, `user_fullname`=30, `otp_valid_in_minutes`=20. Port of `vfic_zns_clamp_params`.
  - `Provider.Send(ctx, phone, templateID, templateData, trackingID)` — refreshes token if needed, clamps params, posts to `business.openapi.zalo.me/message/template`, retries exactly once on `-124`.
  - Token persistence is **not Phase 1's concern** — `Provider` calls the injected `CredentialSource.Update` (Phase 4 wires the DB-backed impl). The refresh_token is **one-shot**, so `Update` must persist both new tokens atomically; the `Provider.mu` serializes concurrent refreshes so two `-124` retries can't double-spend the single-use refresh_token.
  - Boot-time validation: if `ZALO_RESET_ENABLE=true`, fail fast when `ZALO_APP_ID` / `ZALO_SECRET_KEY` are unset (mirrors OTP/Resend fail-fast in `init.go`).
- **Non-functional**
  - All HTTP calls use the shared `http.Client` with a **15s timeout** (matches PHP `timeout => 15` and the OTP email gorilla's `context.WithTimeout(15s)`).
  - No secrets in logs — redact `access_token`/`refresh_token` from any logged response (port `vfic_zns_redact_response`).
  - Never `panic` from a send; surface a typed error to the caller.

## Architecture

### Package layout

```
backend/internal/infra/zalo/
  doc.go                  // package doc + endpoints + error-code reference
  phone.go                // NormalizePhone
  phone_test.go
  params.go               // ClampParams + template registry (617976 caps)
  params_test.go
  provider.go             // Provider (Sender impl) — stateless, takes CredentialSource
  provider_test.go        // httptest-backed; fake CredentialSource
  oauth.go                // refreshToken, exchangeCode (pure functions over Config + CredentialSource)
  oauth_test.go
  errors.go               // ErrorCode map + typed errors
  types.go                // Credentials, Config, CredentialSource interface, Sender interface, SendResult
```

### Credential + config ownership (cross-phase contract)

Phase 1's `Provider` is **stateless** — it does not read the DB or config directly. It
takes its inputs via constructor + a narrow interface:

```go
// CredentialSource is implemented by Phase 4's zaloconnect.Service (DB-backed,
// reads the `zalo.credentials` Settings row). Phase 1 defines the interface;
// Phase 4 provides the implementation. Tests inject a fake.
type CredentialSource interface {
    // Get returns the current credentials. Must be safe for concurrent calls.
    Get(ctx context.Context) (Credentials, error)
    // Update persists rotated tokens atomically (called after a successful refresh).
    Update(ctx context.Context, creds Credentials) error
}

type Credentials struct {
    AppID        string
    SecretKey    string
    TemplateID   string
    AccessToken  string
    RefreshToken string
    ExpiresAt    *time.Time
}

type Config struct {
    SendURL       string // https://business.openapi.zalo.me/message/template
    OAuthURL      string // https://oauth.zaloapp.com/v4/oa/access_token
    RefreshBuffer time.Duration // default 2h
    HTTPTimeout   time.Duration // default 15s
}
```

This keeps Phase 1 free of GORM/Settings/`config.go` imports. Phase 4 implements
`CredentialSource` against the `Settings` table and constructs `Provider` with it.
Phase 2's `zaloreset.Service` receives a `Provider` (which internally holds the
`CredentialSource`) — neither knows nor cares where creds come from.

> **Why the split:** it lets Phase 1 ship + unit-test in isolation (no DB, no config
> loading), and lets Phase 4 (admin UI) change the persistence model without touching the
> protocol code. The earlier draft had Phase 1 owning a `zalo_oauth_credentials` singleton
> table + GORM repo + `ZaloConfig` struct — that created a circular coupling between
> "protocol" and "admin persistence" and is now removed.

> **Note on the OA console "date" type for `otp_valid_in_minutes`:** the OA UI labels the
> param `Loại dữ liệu = date`, but the cap is 20 chars and the sample slot is numeric. The
> service renders it as the integer minutes string (e.g. `"10"`). If a sandbox smoke-test
> returns `-1121`/`-1122`, this is the first thing to revisit (Open Question Q5).

### Provider (Go port of `functions-zns.php` §4–§5)

```go
package zalo

type Sender interface {
    Send(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (SendResult, error)
}

type SendResult struct {
    MsgID      string
    ErrorCode  int
    ErrorMsg   string
    HTTPStatus int
}

type Provider struct {
    creds CredentialSource   // Phase 4 wires the DB-backed impl; tests inject a fake
    cfg   Config             // endpoints + timeouts + refresh buffer
    http  *http.Client
    log   *slog.Logger
    mu    sync.Mutex          // serialize refresh; two concurrent -124 retries must not double-spend the one-shot refresh_token
}

func NewProvider(creds CredentialSource, cfg Config, log *slog.Logger) *Provider
```

**`Send` flow** (directly maps to `vfic_zns_send`):

1. `phone = NormalizePhone(phone)`; return error if empty.
2. `templateData = ClampParams(templateID, templateData)` — enforce per-param caps from the static registry (the `617976` entry: `otp_code`/`user_fullname`=30, `otp_valid_in_minutes`=20). Prevents Zalo `-1121`.
3. `creds = p.creds.Get(ctx)`; `token = p.getAccessToken(ctx, creds)` — refreshes (via `p.refreshToken`) if `expires_at < now+refreshBuffer`.
4. POST template; parse `{error, data:{msg_id}}`.
5. If `error == -124`: call `p.refreshToken(ctx)` under `mu`; on success retry the POST exactly once.
6. Return `SendResult`; never return an error for a Zalo business error (`-118`, `-115`, `-127`, …) — only for transport/Go-level failures. The caller (service) decides what to do with `ErrorCode != 0`.

**`refreshToken` flow** (maps to `vfic_zns_refresh_token`):

1. Under `p.mu`, re-read creds via `p.creds.Get` (another goroutine may have refreshed).
2. POST `cfg.OAuthURL` form body `{grant_type=refresh_token, app_id, refresh_token}` with `secret_key` header.
3. Require both `access_token` and `refresh_token` in the response (Zalo rotates the refresh_token each call).
4. `p.creds.Update(ctx, newCreds)` — Phase 4's impl persists atomically to the `Settings` row.
5. On any failure: do **not** clobber the existing refresh_token with `""` (PHP bug-fix `// không clobber token cũ bằng ''`). Return error; caller short-circuits.

**`ExchangeCode` flow** (maps to `vfic_zns_exchange_code`; called only from Phase 4's admin OAuth callback):

1. POST `cfg.OAuthURL` form body `{grant_type=authorization_code, app_id, code, code_verifier?}` with `secret_key` header.
2. Require `access_token`; if `refresh_token` is present in the response, keep it (don't clobber existing with empty).
3. Build `Credentials` with the new tokens + `expires_at = now+expires_in`; `p.creds.Update(ctx, creds)`.
4. Return `(Credentials, error)`. Phase 4's handler flips the connection status to "connected".

## Related Code Files

- **Create:**
  - `backend/internal/infra/zalo/{doc,types,phone,params,provider,oauth,errors}.go` + `_test.go`
- **Modify:**
  - `backend/internal/constants/messages.go` — add the Zalo-reset message constants (list below).
- **NOT in this phase (moved to Phase 4):**
  - ~~`backend/migrations/NNNN_create_zalo_oauth_credentials.{up,down}.sql`~~ — dropped; Phase 4 uses the existing `settings` table.
  - ~~`backend/internal/infra/zalo/credential_repo.go`~~ — Phase 4 implements `CredentialSource` against `Settings`.
  - ~~`backend/internal/config/config.go` `ZaloConfig`~~ — Phase 4 owns the runtime config (DB-driven; env is seed-only).
- **Reference (read-only):**
  - `wp-content/themes/vfic/inc/functions/functions-zns.php` (tuyennhanvien.vn) — source of truth for endpoints, error map, phone regex, `-124` retry.
  - `backend/internal/infra/email/resend_provider.go` — Go provider style to match (struct + `NewXxxProvider` constructor + `Send`).

### New constants (`constants/messages.go`)

```go
// Zalo OTP password reset
MsgZaloResetRequestedVN    = "Nếu số điện thoại tồn tại trong hệ thống, mã đặt lại mật khẩu đã được gửi qua Zalo."
MsgZaloResetCodeInvalidVN  = "Mã đặt lại không đúng hoặc đã hết hạn. Vui lòng yêu cầu mã mới."
MsgZaloResetStoreDownVN    = "Đã có lỗi xảy ra, vui lòng thử lại."
MsgZaloResetSuccessVN      = "Đặt lại mật khẩu thành công. Vui lòng đăng nhập bằng mật khẩu mới."
MsgZaloResetMobileRequiredVN = "Vui lòng nhập số điện thoại"
```

## Implementation Steps

1. **`phone.go`** — `NormalizePhone`. Port the PHP regex/branch logic byte-for-byte (`84\d{9}` final check). Table-test: `0987654321`, `+84987654321`, `84987654321`, `"0987 654 321"`, `"0123"` (invalid → `""`), `""` → `""`.
2. **`params.go`** — `ClampParams(templateID, data)`. Static registry with the `617976` entry: `{otp_code:30, user_fullname:30, otp_valid_in_minutes:20}`. Use `utf8` rune-count (Vietnamese diacritics), matching `vfic_zns_clamp_params`'s `mb_substr`. Test: a 35-char `user_fullname` → 30 chars; a numeric `otp_valid_in_minutes` passes through.
3. **`errors.go`** — port `vfic_zns_error_message` map (`-108`, `-115`, `-118`, `-124`, `-127`, `-135`, `-139`, `-140`, `-141`, `-144`, `-147`, `-1121`, `-1122`, …). Expose `ErrorMessage(code int) string`.
4. **`types.go`** — `Credentials`, `Config`, `CredentialSource` interface, `Sender` interface, `SendResult`.
5. **`oauth.go`** — `refreshToken(ctx)` and `ExchangeCode(ctx, code, verifier)` as pure functions over `(Provider)` that use `p.creds` for read/update. Phase 4's admin callback calls `ExchangeCode`.
6. **`provider.go`** — `Provider`, `NewProvider(creds, cfg, log)`, `Send`, `getAccessToken`. Wire `http.Client{Timeout: cfg.HTTPTimeout}` (default 15s). Redact tokens in all `slog` calls.
7. **`constants/messages.go`** — append the five Zalo-reset message constants (no config/migration work in this phase).
8. **Local verify** — `cd backend && go build ./... && go test ./internal/infra/zalo/... -v -race -cover`. The tests inject a fake `CredentialSource` (in-memory map) so no DB is needed.

## Success Criteria

- [ ] `NormalizePhone` passes the table-test covering domestic/`+84`/`84`/whitespace/invalid.
- [ ] `ClampParams("617976", {otp_code:"123456", user_fullname:<35 chars>, otp_valid_in_minutes:"10"})` returns `user_fullname` truncated to 30 runes; the other two unchanged.
- [ ] `Provider.Send` against an `httptest` mock returns `ErrorCode=0, MsgID="m1"` on a 200 `{error:0,data:{msg_id:"m1"}}`.
- [ ] `Provider.Send` retries exactly once when the first response is `-124` and the refresh endpoint returns fresh tokens; the **second** POST uses the new access_token. Verify the mock saw two POSTs and one refresh.
- [ ] `Provider.Send` does **not** retry on `-118`; returns `ErrorCode=-118`.
- [ ] A refresh response missing `refresh_token` is rejected, the stored refresh_token is **unchanged**, and the error is logged with both tokens redacted.
- [ ] `go test ./internal/infra/zalo/... -race` green; coverage ≥ 80% for the package. All tests use a fake `CredentialSource` (in-memory) — no DB or env required.

## Risk Assessment

- **R-Z1 (token clobber):** Zalo refresh_tokens are single-use. If a refresh succeeds but we crash before persisting, the old refresh_token is dead and we cannot refresh again → full admin re-OAuth. **Mitigation:** call `p.creds.Update(ctx, newCreds)` immediately on a successful refresh response, before returning to `Send`'s retry. The `CredentialSource.Update` contract (Phase 4) must be atomic. The PHP code documents this same risk.
- **R-Z2 (production template not approved):** If `ZALO_RESET_TEMPLATE_ID` points at a template Zalo hasn't approved for production sends, every send returns `-131`/`-109`. **Mitigation:** Phase 9 has a go-live checklist; fail-fast does **not** cover this (template id is structurally valid). Surface `-131` in the service log as "admin action required".
- **R-Z3 (concurrent refresh):** Two simultaneous sends that both hit `-124` could race the refresh. The refresh_token is single-use, so the second refresh burns the just-issued token. **Mitigation:** `Provider.mu` serializes all refreshes; the second caller re-reads creds under the lock and skips refresh if the token has already rotated.
- **R-Z4 (clock skew):** `expires_at` comparison uses `clock.Now()` (Asia/Ho_Chi_Minh, per project rule). Production server is UTC internally; the 2h buffer absorbs skew.
