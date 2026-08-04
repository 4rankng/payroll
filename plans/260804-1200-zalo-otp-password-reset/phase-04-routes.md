---
phase: 4
title: "Backend: Admin Zalo Connection Management"
status: pending
priority: P1
dependencies: ["1"]
effort: "M"
---

# Phase 4: Backend: Admin Zalo Connection Management

## Overview

Make the Zalo OA connection **admin-managed at runtime** through the DB and a set of
admin-only HTTP endpoints. This phase delivers: a `zaloconnect.Service` that owns the
DB-backed credential store (implementing Phase 1's `zalo.CredentialSource`), the runtime
`zalo.enabled` toggle, an admin OAuth v4 connect flow with a CSRF-guarded callback, a
connection-status read endpoint, and bootstrap seeding from env vars.

After this phase, an admin can: save `app_id`/`secret_key`/`template_id`, click "Kết nối
Zalo" to run OAuth, see live connection status (`connected` / `expired` / `error`), toggle
the feature on/off, and rotate the connection — all without touching `.env` or restarting
api-server. Phase 5 builds the UI on top of these endpoints.

**Why this is a separate phase from Phase 1:** Phase 1 is the stateless ZNS protocol
client; this phase is the DB + admin + OAuth-orchestration layer. Keeping them separate
means Phase 1's unit tests need no DB, and Phase 4 can change the persistence model (e.g.
move to Vault later) without touching protocol code.

## Requirements

- **Functional**
  - `GET /api/v1/admin/zalo` — return current connection status: `{enabled, configured (has app_id+secret), connected (has valid access_token), template_id, expires_at, last_error}`. **Never** returns `secret_key`, `access_token`, or `refresh_token` (read-only, masked).
  - `PUT /api/v1/admin/zalo/credentials` — save `{app_id, secret_key, template_id}`. Does NOT touch tokens (admin must re-OAuth after rotating keys). Audit-logged.
  - `POST /api/v1/admin/zalo/oauth/start` — generate a random `state`, store it in Redis (5-min TTL), return `{redirect_url}` (Zalo permission URL with `app_id`, `redirect_uri`, `state`). The admin's browser navigates there.
  - `GET /api/v1/admin/zalo/oauth/callback?code=...&state=...` — **admin-authenticated**. Validate `state` (Redis GETDEL, single-use), call `provider.ExchangeCode(ctx, code)`, persist tokens, redirect to `/admin/settings?tab=zalo&zalo_connected=1`.
  - `PUT /api/v1/admin/zalo/enabled` — body `{enabled: bool}`. Flips the runtime toggle. Audit-logged.
  - `POST /api/v1/admin/zalo/refresh` — force a token refresh (manual "Làm mới token" button for debugging). Audit-logged.
- **Non-functional**
  - All five endpoints require **admin role** (`Authorize(admin)`) — including the OAuth callback. A non-admin hitting the callback gets 403, even with a valid `code`.
  - The `state` param is 256-bit, single-use (Redis GETDEL on validation), and bound to the admin's session — defends against OAuth CSRF (the classic login-csrf attack on OAuth flows).
  - `secret_key` and tokens are **never** logged, never returned in any response (masked as `"***"` or omitted). Audit logs record *that* they changed, not the values.
  - Runtime toggle is **hot** — `zaloreset.Service` (Phase 2) reads `zalo.enabled` on each `RequestReset` call via the credential source, not at boot.

## Architecture

### DB storage (reuses existing `settings` table — no migration)

Two `settings` rows, managed by `zaloconnect.Service`:

| `key` | `value_type` | `value` |
|---|---|---|
| `zalo.enabled` | `boolean` | `"true"` / `"false"` |
| `zalo.credentials` | `json` | `{"app_id":"...","secret_key":"...","template_id":"617976","access_token":"...","refresh_token":"...","expires_at":"2026-08-04T...","last_error":"..."}` |

The generic `domain.Settings` + `SettingsRepository` + `SettingsStore` (existing) handle
persistence. `zaloconnect.Service` wraps them with typed accessors. The JSON value row is
written under a `SELECT ... FOR UPDATE` lock (reuse `SettingsStore.ExecuteWrite` pattern)
so concurrent refreshes don't clobber each other.

### `zaloconnect.Service` (implements `zalo.CredentialSource`)

```go
package zaloconnect

type Service struct {
    settingsRepo domain.SettingsRepository
    db           *gorm.DB             // for FOR UPDATE transactions
    provider     *zalo.Provider       // constructed in bootstrap with THIS service as CredentialSource
    callbackURL  string               // https://<host>/api/v1/admin/zalo/oauth/callback
    redis        *redis.Client        // state store
    clk          clock.Clock
    log          *slog.Logger
}

// CredentialSource impl — called by Provider during Send/refresh.
func (s *Service) Get(ctx context.Context) (zalo.Credentials, error)
func (s *Service) Update(ctx context.Context, creds zalo.Credentials) error

// Admin-facing
func (s *Service) GetStatus(ctx context.Context) (Status, error)          // masked, for GET /admin/zalo
func (s *Service) SaveCredentials(ctx, appID, secret, templateID, actor) error
func (s *Service) StartOAuth(ctx, actor) (redirectURL string, err error)  // mints + stores state
func (s *Service) HandleOAuthCallback(ctx, code, state, actor) error       // validates state + ExchangeCode
func (s *Service) SetEnabled(ctx, enabled bool, actor) error
func (s *Service) RefreshNow(ctx, actor) error
func (s *Service) IsEnabled(ctx) (bool, error)                            // hot-read, called by Phase 2/3
```

> **Circular construction:** `Service` needs `*zalo.Provider`, and `zalo.Provider` needs a
> `CredentialSource` (which is `Service`). Bootstrap resolves this with a two-step wire:
> (1) construct `Service` with a nil provider; (2) construct `Provider` with the service;
> (3) `service.SetProvider(provider)`. The provider field is only used for
> `ExchangeCode`/`RefreshNow` (admin actions) — the `CredentialSource` methods (`Get`/`Update`)
> don't need it, so the nil-during-construction is safe.

### OAuth v4 connect flow (sequence)

```
Admin browser                 api-server                    Zalo OAuth
     │                             │                            │
     │ 1. POST /admin/zalo/oauth/start                             │
     │────────────────────────────▶│                            │
     │                             │ state=rand256; SETEX 5m      │
     │ 2. {redirect_url}           │                            │
     │◀────────────────────────────│                            │
     │ 3. window.location = redirect_url                         │
     │──────────────────────────────────────────────────────────▶│
     │                             │                            │
     │ 4. admin consents on Zalo   │                            │
     │◀──────────────────────────────────────────────────────────│
     │ 5. 302 → callback?code=...&state=...                       │
     │────────────────────────────▶│                            │
     │                             │ 6. GETDEL state (single-use) │
     │                             │ 7. provider.ExchangeCode(code)│
     │                             │───────────────────────────▶│
     │                             │    POST oauth/v4 (secret_key)│
     │                             │◀───────────────────────────│
     │                             │ 8. Update settings (tokens)  │
     │                             │ 9. audit event               │
     │ 10. 302 → /admin/settings?tab=zalo&zalo_connected=1       │
     │◀────────────────────────────│                            │
```

The callback carries the admin's session cookie/JWT (step 5 is a same-browser top-level
navigation), so the existing `Authorize(admin)` middleware authenticates it. The `state`
guards against a malicious site initiating the flow (it can't complete step 7 without the
admin's session).

### Bootstrap wiring (`app/bootstrap/services/init.go`)

After Phase 1's package exists, add near the OTP wiring (~line 605):

```go
// Zalo ZNS connection (admin-managed via Settings table). Phase 4.
zaloCallbackURL := cfg.Zalo.CallbackURL // https://<host>/api/v1/admin/zalo/oauth/callback
zaloConnectSvc := zaloconnect.NewService(repos.Settings, db.DB, zaloCallbackURL, redis.Client, clk, logger)

// Seed from env on first boot ONLY (Settings row empty). Thereafter DB is authoritative.
zaloConnectSvc.SeedFromEnvIfEmpty(ctx, zaloconnect.EnvSeed{
    Enabled:    cfg.Zalo.Enabled,
    AppID:      cfg.Zalo.AppID,
    SecretKey:  cfg.Zalo.SecretKey,
    TemplateID: cfg.Zalo.TemplateID, // default "617976"
})

// Two-step wire (Service needs Provider, Provider needs Service as CredentialSource).
zaloCfg := zalo.Config{ /* endpoints + timeouts */ }
zaloProvider := zalo.NewProvider(zaloConnectSvc, zaloCfg, logger)
zaloConnectSvc.SetProvider(zaloProvider)

// Phase 2's zaloreset.Service receives the provider + a hot-enabled check via zaloConnectSvc.
```

`cfg.Zalo` here is a **minimal** bootstrap config (callback URL + seed values only — not
the runtime toggle). The runtime toggle lives in DB. This keeps env as "first-boot seed"
and DB as "authoritative thereafter", matching how the admin UI takes over.

## Related Code Files

- **Create:**
  - `backend/internal/app/services/zaloconnect/{service,doc}.go` + `_test.go`
  - `backend/internal/transport/http/handlers/admin/zalo_handler.go` + `_test.go`
  - `backend/internal/transport/http/dto/zalo.go` (admin DTOs — masked Status, SaveCredentials, etc.)
- **Modify:**
  - `backend/internal/config/config.go` — add **minimal** `ZaloConfig{CallbackURL, Enabled, AppID, SecretKey, TemplateID}` for bootstrap seeding only (not the runtime feature flag).
  - `backend/internal/app/bootstrap/services/init.go` — two-step wire + `SeedFromEnvIfEmpty`.
  - `backend/internal/transport/http/router.go` — register the 5 admin routes under the admin-auth group.
- **Reference (read-only):**
  - `backend/internal/app/services/config/settings_service.go` — `SettingsService` CRUD pattern + cache invalidation.
  - `backend/internal/app/services/infrastructure/settings_store.go` — `SettingsStore.ExecuteWrite` (`FOR UPDATE` lock pattern).
  - `backend/internal/infra/cache/nonce_store.go` — Redis single-use token pattern (reuse for OAuth `state`).
  - `backend/internal/domain/settings.go` — `Settings` entity + `SettingsRepository`.

## Implementation Steps

1. **DTOs** — `ZaloStatusDTO` (masked: `enabled`, `configured`, `connected`, `template_id`, `expires_at`, `last_error`), `ZaloCredentialsDTO` (`app_id`, `secret_key`, `template_id`), `ZaloEnabledDTO` (`enabled`).
2. **`zaloconnect.Service`** — `Get`/`Update` (read/write the `zalo.credentials` JSON row under `FOR UPDATE`), `GetStatus`, `SaveCredentials`, `StartOAuth` (mint `state`, `SETEX 5m`, build Zalo permission URL), `HandleOAuthCallback` (`GETDEL state`, call `provider.ExchangeCode`, `Update`), `SetEnabled`, `RefreshNow`, `IsEnabled`, `SeedFromEnvIfEmpty`.
3. **Admin handler** — 5 thin handler methods mapping to the service; all behind `Authorize(admin)`. Mask all secret fields in responses. Publish audit events for `SaveCredentials`, `HandleOAuthCallback`, `SetEnabled`, `RefreshNow`.
4. **Router** — register under the existing admin group: `adminGroup.GET("/admin/zalo", ...)`, etc. The OAuth callback is **also** admin-authenticated (it's in the admin group, not the public group).
5. **Config + bootstrap** — add minimal `ZaloConfig` (callback URL + seed), two-step wire, `SeedFromEnvIfEmpty`.
6. **Unit tests** — `Service.Get`/`Update` round-trip; `StartOAuth` stores state + builds correct URL; `HandleOAuthCallback` rejects bad/reused `state`, calls `ExchangeCode`, persists; `SaveCredentials` doesn't touch tokens; masking in `GetStatus`.
7. **Local verify** — `cd backend && go build ./... && go test ./internal/app/services/zaloconnect/... ./internal/transport/http/handlers/admin/... -race`.

## Success Criteria

- [ ] `GET /admin/zalo` returns `{enabled, configured, connected, template_id, expires_at, last_error}` and **never** includes `secret_key`, `access_token`, or `refresh_token` (assert in test).
- [ ] `PUT /admin/zalo/credentials` with `app_id`+`secret_key`+`template_id` persists them; a subsequent `GET` shows `configured=true, connected=false` (tokens still empty until OAuth).
- [ ] `POST /admin/zalo/oauth/start` returns a `redirect_url` containing the `app_id`, the registered `callbackURL`, and a `state`; the `state` is single-use (a second callback with the same `state` is rejected).
- [ ] The OAuth callback rejects a non-admin caller with 403 (assert the middleware is applied).
- [ ] `PUT /admin/zalo/enabled {enabled:false}` flips the toggle; `zaloconnect.Service.IsEnabled(ctx)` immediately returns `false` without a restart.
- [ ] `SeedFromEnvIfEmpty` populates the Settings row from env on first boot, and is a **no-op** when the row already exists (DB authoritative).
- [ ] Every mutating endpoint publishes an audit event recording the actor + action (not the secret values).
- [ ] All tests `-race` green; no real network calls (fake `provider.ExchangeCode`).

## Risk Assessment

- **R-Z18 (OAuth CSRF / login-csrf):** Without the `state` guard, a malicious site could initiate the OAuth flow and trick the admin's browser into completing it, connecting the attacker's Zalo OA. **Mitigation:** 256-bit single-use `state` (Redis GETDEL) + admin-session requirement on the callback. Standard OAuth v4 CSRF defense.
- **R-Z19 (secret in URL/logs):** The OAuth `code` and `secret_key` travel in HTTP bodies/redirects. **Mitigation:** TLS everywhere (already enforced); `secret_key` only ever in `PUT` body + Zalo's OAuth endpoint (never in a query string); all `slog` calls redact. Audit log records "credentials saved", not the values.
- **R-Z20 (DB outage breaks OTP):** Since credentials are DB-backed, a DB outage during a refresh means the Provider can't persist new tokens → potential token loss (R-Z1). **Mitigation:** the refresh path catches `Update` errors, logs them, and the old tokens remain valid until their real expiry; the admin sees `last_error` in the status endpoint. Not crash-worthy.
- **R-Z21 (callback URL mismatch):** If the `callbackURL` configured in code doesn't match what's registered in the Zalo OA console, OAuth returns an invalid-redirect error. **Mitigation:** Phase 5's UI shows the exact callback URL to copy into the OA console; `cfg.Zalo.CallbackURL` is env-configurable per environment.
