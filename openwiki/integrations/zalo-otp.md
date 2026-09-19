---
type: integration
title: Zalo OTP & Password Reset
description: Zalo OA-based OTP login, employee-mobile password reset via ZNS, the stateless Zalo client, the credentials-via-settings contract, and the admin-managed connect flow.
tags: [integration, zalo, otp, password-reset, zns, credentials]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-8c38cb85765bb48747a83874
    resource: repo://docs/decisions/ADR-011-zalo-otp-password-reset.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Integration: Zalo OTP & Password Reset

The Zalo channel is the primary authentication path for the employee mobile app. It uses the Zalo Official Account (OA) API to deliver OTP codes for login and password reset. ADR-011 is the source of truth.

## Two services, one OA

- `internal/app/services/otp/` — the OTP code-generation and verification service.
- `internal/app/services/passwordreset/` — the email-based password reset (magic link).
- `internal/app/services/zaloconnect/` — admin-managed OA connection. Owns the OAuth v4 connect flow, credential refresh, and the runtime enable/disable toggle.
- `internal/app/services/zaloreset/` — the employee-mobile password reset via ZNS. Role-gated to `employee`; anti-enumeration via dummy sessions.

## Stateless Zalo client

`internal/infra/zalo/` is a DB-agnostic ZNS / OAuth v4 client, ported from a prior PHP implementation. It owns:

- Phone normalization (Vietnamese mobile format).
- Param clamping for ZNS template parameters.
- OAuth v4 with `- `-` `-124` retry semantics.
- Credential resolution via a `CredentialSource` interface (see below).

It is stateless — no DB connection, no cache. All state lives in the calling services and in the `settings` table.

## Credentials via settings

The `CredentialSource` interface lets the client pull live credentials without itself talking to the DB:

- `zalo.enabled` (`BOOLEAN`) — runtime kill switch. When `false`, every ZNS send is rejected before the network call.
- `zalo.credentials` (`JSON`) — the OA app_id, secret, and access/refresh token bundle.

`zaloconnect` is the only writer to these rows. `ZALO_*` env vars are **bootstrap-time seed only**: on first boot they populate the initial values, thereafter the admin UI is authoritative. The admin UI lives at `/admin/settings?tab=zalo` and supports credential entry, the OAuth v4 connect flow, and the enable/disable toggle. Hot reload — no redeploy required. See `docs/runbooks/zalo-oa-connect.md`.

## OTP login flow

`frontend/src/pages/OTPLogin.tsx` is the entry. End-to-end:

1. User enters phone number → `POST /api/v1/auth/login` (step 1, sends OTP).
2. The OTP service generates a code and calls `internal/infra/zalo/` to send a ZNS template with the code.
3. User receives the code on Zalo → `POST /api/v1/auth/login/verify` (step 2).
4. The OTP service verifies the code against the Redis pending store (`internal/infra/cache/otp_pending_store.go`) and issues a JWT on success.
5. Resend: `POST /api/v1/auth/login/resend` (rate-limited).

The Redis pending store uses a short TTL keyed by normalized phone; the row is consumed atomically on verify.

## Zalo password reset

`frontend/src/pages/ZaloResetPassword.tsx` is the employee-mobile reset flow. The `zaloreset` service:

1. Looks up the employee by phone (only `employee` role is allowed; admin/partner use the email reset path).
2. Generates a one-time reset code and sends it via ZNS.
3. Anti-enumeration: always returns success-shaped response even when the phone is not found, and emits a dummy session to mask timing.
4. On verify, atomically updates the password and invalidates every existing session (`tokens_invalid_before`).

The email magic-link path (`/api/v1/auth/password-reset/request` + `/confirm`) is the parallel flow for users with email on file. Same anti-enumeration + atomic-consume discipline. See `integrations/auth-rbac.md`.

## OAuth v4 connect flow

`zaloconnect` implements the OAuth v4 connect handshake:

1. Admin clicks "Kết nối Zalo" in the settings UI.
2. The UI redirects to Zalo's authorize URL with the registered callback.
3. Zalo redirects back with `code`; `zaloconnect` exchanges it for `access_token` + `refresh_token` via the OAuth v4 token endpoint.
4. Credentials are stored in `settings.zalo.credentials`. `access_token` expiry triggers an automatic refresh on the next request.

The refresh path uses the OAuth v4 refresh endpoint, retries on `-124` (rate-limit / transient) with exponential backoff.

## Operational notes

- Disabling Zalo (`zalo.enabled=false`) does not disable the OTP login or password reset endpoints — it only blocks the actual ZNS send. The UI surfaces a clear error so users know to use the email path.
- Tokens that expire are refreshed lazily on the next send, not by a background sweeper, so a fully idle OA stays connected without a heartbeat job.
- The ZALO_* env seed values are read once at bootstrap; runtime changes go through the settings table.

## Relationships

- Auth flow — `integrations/auth-rbac.md`.
- Employee mobile entry points — `frontend/employee-mobile.md`.
- Password reset email path — `integrations/auth-rbac.md` and `docs/runbooks/zalo-oa-connect.md`.
- Settings table schema and runtime-config conventions — `architecture/domain-layer.md` and `architecture/application-services.md`.
