---
title: "Employee Password Reset via Zalo OTP (ZNS)"
description: "Self-service password reset for employee-role users whose only contact channel is a mobile number. The user enters their phone, receives a 6-digit OTP via Zalo ZNS (template message), and sets a new password in a single confirm call. Mirrors the existing email magic-link reset's security contract (anti-enumeration, atomic password+session-invalidation transaction, per-channel rate limit, single-use code). Ports the proven ZNS sender from the tuyennhanvien.vn PHP codebase to a Go provider behind the same EmailDeliveryPort-style abstraction."
status: pending
priority: P2
branch: "main"
tags: ["auth", "security", "zalo", "zns", "otp", "frontend", "backend", "employee"]
blockedBy: []
blocks: []
created: "2026-08-04T07:00:43.619Z"
createdBy: "ck:plan"
source: skill
---

# Employee Password Reset via Zalo OTP (ZNS)

## Overview

Add a **self-service, phone-channel password reset** for `employee`-role users.
Employees in this system frequently have a `users.mobile` on file but **no**
`users.email` (the existing `/auth/password-reset/request` flow is email-only and
silently cannot serve them). The new flow delivers a 6-digit OTP through
**Zalo ZNS** (Zalo Notification Service — template messages billed per send and
delivered to the phone's linked Zalo account), then lets the user set a new
password in a single `confirm` call.

**Why ZNS and not SMS:** the org already runs a Zalo OA with approved ZNS
templates in the sister codebase `tuyennhanvien.vn`. ZNS is markedly cheaper
than transactional SMS in Vietnam and reaches the same phone-number identity
(Zalo is keyed by phone). One new **transactional** template (`password_reset`)
must be registered/approved in the Zalo OA console before go-live.

**Flow (2 steps, mirrors ZNS delivery + existing OTP shape):**

1. User clicks **"Quên mật khẩu?"** on `/login`, picks **"Đặt lại qua Zalo"** on `/forgot-password`.
2. Enters mobile → `POST /api/v1/auth/zalo-reset/request` `{ mobile }`.
3. Backend looks up `users WHERE mobile = ? AND role = 'employee' AND deleted_at IS NULL`.
   If found, generate a 6-digit code (reuse `otp.GenerateCode`), store its SHA-256 hash
   in Redis under an opaque `session_id` (10-min TTL), and **asynchronously** send a ZNS
   template message (`617976`) carrying **three** params: `otp_code`, `user_fullname`
   (clamped to 30 chars from `users.fullname`), and `otp_valid_in_minutes` (derived from
   `ZaloConfig.CodeTTL`, e.g. `"10"` — where `ZaloConfig` here is the **bootstrap tuning**
   value `ZALO_RESET_CODE_TTL`, distinct from the DB-driven admin connection). Response **always** returns a `session_id` with a
   generic message whether or not the mobile exists (anti-enumeration — identical to the
   email reset's `RequestReset` contract). **Not-found mobiles do NOT trigger a paid ZNS
   send** — only a dummy Redis write (0 ₫ per probe; enumeration is cost-bounded).
4. User types the 6 digits + new password on `/zalo-reset-password` →
   `POST /api/v1/auth/zalo-reset/confirm` `{ otp_session_id, code, new_password }`.
5. Backend consumes the code (atomic), enforces password strength, and in a **single DB
   transaction** updates the password + invalidates all sessions — reusing the exact
   `UserRepository.UpdatePasswordAndInvalidateSessions` + `NewPasswordChangedEvent` path
   the email reset uses.
6. User is redirected to `/login` with a success toast.

**Scope guardrails (owner decisions baked in, revisit at validate):**

- **Employees only.** Admin/partner already have email-OTP 2FA + email reset; routing
  them through ZNS would split the security surface. The handler rejects non-employee
  roles with the same generic anti-enumeration response.
- **ZNS is the sole OTP channel here** (no SMS fallback). If the phone has no linked Zalo
  account, Zalo returns `-118`; the user sees a generic "if the number exists, a code was
  sent" message and must use the email channel or contact admin. SMS is explicitly out of
  scope (cost + new vendor).
- **Separate payroll-only OA** "Ting Ting Software Solution" / app "TingTing Soft" —
  **not** the `tuyennhanvien.vn` recruitment OA. This plan uses its own
  `ZALO_APP_ID`/`ZALO_SECRET_KEY`. The tuyennhanvien.vn PHP code is a **reference
  implementation only** (port the protocol, not the credentials).
- **One template, `617976` (`OTP-ZNS-v1`)** — transactional (`Mục đích gửi: Giao dịch`,
  tag=1), 3 params (`otp_code`, `user_fullname`, `otp_valid_in_minutes`), 300 ₫/send via
  phone. Status **Đang duyệt** (pending Zalo review, 2–3 business days) as of 2026-08-04;
  **prod go-live is blocked until approved**, but dev/test can proceed against the sandbox.

## Background — what already exists (REUSE, do not reinvent)

| Concern | Existing asset | How this plan uses it |
|---|---|---|
| 6-digit code gen + SHA-256 hash | `app/services/otp/code.go` (`GenerateCode`, `HashCode`, `EqualCodeHash`) | Called verbatim by the new service. |
| Pending-session Redis store pattern | `infra/cache/otp_pending_store.go` (`OTPPendingStore`, per-user index, opaque id) | **Mirror**, not import — the OTP store binds to IP/UA for login; password reset has no prior login so binding differs. New `ZaloResetStore` is a focused sibling. |
| Atomic password + session invalidation | `domain.UserRepository.UpdatePasswordAndInvalidateSessions` | Called from `ConfirmReset` — identical to email reset. |
| Password strength + hashing | `app/services/user.UserService.ValidatePassword` / `HashNewPassword` | Reused as-is. |
| Password-changed audit event | `domain.NewPasswordChangedEvent(ctx, uid, username, method, actorID, actorName)` | Called with `method = "zalo_reset"`. |
| Anti-enumeration handler pattern | `transport/http/handlers/auth.go::RequestPasswordReset` + `passwordreset.Service::RequestReset` | Copied shape: always-200 + dummy Redis write on not-found to equalize timing. |
| Per-channel rate limit w/ body-restore | `middleware/rate_limit.go::CreatePasswordResetRateLimit` + `passwordResetEmailKeyGetter` | New `zaloResetMobileKeyGetter` sibling; same NFKC/case-fold normalization adapted for phone digits. |
| Bootstrap wiring shape | `app/bootstrap/services/init.go` (services struct field + `NewService(...)` near line 621) | Add `ZaloPasswordReset *zaloreset.Service` field; construct after `emailPasswordResetService`. |
| ZNS send/OAuth/refresh/phone-norm/error-map | `tuyennhanvien.vn:wp-content/themes/vfic/inc/functions/functions-zns.php` | **Ported to Go** in Phase 1 as `infra/zalo` provider. Endpoint, OAuth v4, `-124` one-retry, `84xxxxxxxxx` normalization, tracking_id, error-code map all originate here. |

## Phases

| Phase | Name | Status | Effort | Owner file |
|-------|------|--------|--------|------------|
| 1 | [Backend: ZNS Provider](./phase-01-backend-zns-provider-config.md) | Pending | M | `infra/zalo/`, `constants/` |
| 2 | [Backend: Zalo OTP Reset Store & Service](./phase-02-backend-zalo-otp-reset-store-service.md) | Pending | M | `infra/cache/zalo_reset_store.go`, `app/services/zaloreset/` |
| 3 | [Backend: Public Reset Handler, Routes & Wiring](./phase-03-backend-handler.md) | Pending | S | `transport/http/handlers/auth.go`, `router.go`, `middleware/rate_limit.go` |
| 4 | [Backend: Admin Zalo Connection Management](./phase-04-routes.md) | Pending | M | `app/services/zaloconnect/`, `transport/http/handlers/admin/`, OAuth callback |
| 5 | [Frontend: Admin Zalo Settings Tab](./phase-05-rate-limit-wiring.md) | Pending | M | `pages/admin/SettingsPage/`, `components/settings/`, hooks |
| 6 | [Backend: Tests](./phase-06-backend-tests.md) | Pending | M | `*_test.go` across the above |
| 7 | [Frontend: Employee Reset Pages, Hooks & Routing](./phase-07-frontend-zalo-reset-pages.md) | Pending | M | `pages/`, `hooks/api/`, `App.tsx`, `Login.tsx` |
| 8 | [Hooks & Routing (folded into Phase 7)](./phase-08-hooks-routing.md) | Pending | 0 | — |
| 9 | [Docs & Migration](./phase-09-docs-migration.md) | Pending | S | `docs/`, `.env.example`, `backend/migrations/` |

> **Scope expansion (2026-08-04):** the admin must be able to **(a)** enter Zalo OA keys
> (`app_id`, `secret_key`), **(b)** run the OAuth v4 connect flow to obtain
> `access_token` + `refresh_token`, **(c)** toggle the whole Zalo OTP feature on/off at
> runtime, and **(d)** see connection status + last-refresh/error — all from the existing
> `/admin/settings` page. This adds Phase 4 (admin connection backend + OAuth callback)
> and Phase 5 (admin settings UI tab). The two were originally folded stubs; they are now
> first-class phases. The connection is **DB-backed via the existing key-value `Settings`
> table** (not a new singleton table and not env-only), so toggling on/off does not require
> a redeploy. `ZALO_*` env vars become **bootstrap-only seed values**, superseded by DB
> settings once an admin saves the connection.
>
> **Phase 8 remains folded** into Phase 7 (employee frontend is one unit).

## Architecture

```
 ┌────────────┐  POST /auth/zalo-reset/request {mobile}
 │  Browser   │ ─────────────────────────────────────────┐
 │ (employee) │  POST /auth/zalo-reset/confirm {sid,code,pwd}
 └─────┬──────┘ ─────────────────────────────────────────┐
       │gin                                            │
       ▼                                               ▼
 ┌─────────────────────────────┐         ┌───────────────────────────────┐
 │ rate_limit.go               │         │ auth.go handlers              │
 │ zaloResetMobileKeyGetter    │         │ ZaloRequestReset /            │
 │ (per-mobile NFKC/digit norm)│         │ ZaloConfirmReset              │
 └────────────┬────────────────┘         └────────────┬──────────────────┘
              │                                       ▼
              │                  ┌────────────────────────────────────────┐
              │                  │ app/services/zaloreset.Service         │
              │                  │  RequestReset(mobile)                   │
              │                  │   └─ userRepo.GetByMobile(+role check)  │
              │                  │   └─ GenerateCode + HashCode            │
              │                  │   └─ store.Create(userID, codeHash) ────┼──┐
              │                  │   └─ go zalo.Send(...)  (async, 15s)    │  │
              │                  │  ConfirmReset(sid, code, newPassword)   │  │
              │                  │   └─ store.Consume (GETDEL-equivalent)  │  │
              │                  │   └─ ValidatePassword / HashNewPassword│  │
              │                  │   └─ UpdatePasswordAndInvalidateSessions│ │
              │                  │   └─ Publish PasswordChangedEvent      │  │
              │                  └────────────┬───────────────────────────┘  │
              │                               │                              │
              │                               ▼                              ▼
              │                  ┌──────────────────────┐   ┌──────────────────────┐
              │                  │ infra/cache/         │   │ infra/zalo/          │
              │                  │ ZaloResetStore       │   │ Provider (Go port of │
              │                  │ Redis: zreset:<hash> │   │ functions-zns.php)   │
              │                  │ TTL 10m, atomic      │   │  OAuth v4 + refresh   │
              │                  │ consume via Lua      │   │  Send(template_id,    │
              │                  └──────────────────────┘   │   {code})            │
              │                                             │  -124 single retry   │
              │                                             └──────────┬───────────┘
              │                                                        │ HTTPS
              │                                                        ▼
              │                                  https://business.openapi.zalo.me
              │                                        /message/template
              │                                                        │
              │                                                        ▼
              │                                              OAuth v4 token refresh
              │                                              https://oauth.zaloapp.com/v4
              └─ Redis also backs the rate limiter (existing infra, shared client).
```

### Key contracts

- **`session_id` is opaque** (256-bit, base64url, same generator as OTP store's
  `newOpaqueID`). The client round-trips it; the code is **never** returned to the
  client (only the ZNS message carries it).
- **`store.Create` returns `(sessionID, error)`** — the code is hashed before the
  store ever sees it; Redis never holds the plaintext code.
- **`store.Consume` is atomic** (Lua GETDEL) so a double-submit / replay can succeed
  at most once — same guarantee the email token store provides.
- **ZNS send is fire-and-forget** from the request path (15s background ctx, panic
  recovery), exactly like `passwordreset.Service.RequestReset`. A failed send does NOT
  fail the HTTP request; the user can re-request.
- **No IP/UA binding** on the reset session (unlike login OTP) — the user is by
  definition not authenticated. Brute-force is bounded by the rate limiter + the
  store's per-`session_id` single consume + the 6-digit hash compare (no per-account
  attempt counter is added in v1; see Risk R-Z3 in Phase 2).

### Data flow for the ZNS message (Phase 1, ported from PHP)

```
mobile "0987 654 321"
   │  NormalizePhone  →  "84987654321"   (84 + 9 digits, regex-validated)
   │
   ▼
Provider.Send(ctx, phone, "617976", trackingID, {
     "otp_code":             "123456",                       // string, cap 30
     "user_fullname":        clamp(u.Fullname, 30),           // string, cap 30
     "otp_valid_in_minutes": "10",                            // from ZaloConfig.CodeTTL → minutes
   })
   │
   ├─ get access_token (cached; refresh if expires_at < now+2h)
   │     └─ POST oauth.zaloapp.com/v4/oa/access_token
   │        {grant_type: refresh_token, app_id, refresh_token}  secret_key hdr
   │     └─ persist new access + refresh + expires_at  (refresh token is ONE-SHOT)
   │
   ├─ POST business.openapi.zalo.me/message/template
   │     headers: access_token, Content-Type: application/json
   │     body: {phone, template_id: "617976",
   │            template_data: {otp_code, user_fullname, otp_valid_in_minutes},
   │            tracking_id}
   │
   └─ if error == -124 (token dead) → refresh once → retry exactly once
      (any other error → return to caller; service logs + audit-event only)
```

> **Template param contract (from OA console, 2026-08-04):** `OTP-ZNS-v1` (id `617976`)
> requires exactly `otp_code` (string/30), `user_fullname` (string/30), and
> `otp_valid_in_minutes` (date/20). Missing or over-cap params return Zalo error `-1122`
> (missing) or `-1121` (over cap). Phase 1's `Provider.Send` clamps each string param to
> its cap before posting (port `vfic_zns_clamp_params`); `otp_valid_in_minutes` is rendered
> as a plain string of the integer minutes (the OA console labels it "date" but the cap is
> 20 chars and the sample slot is numeric — send `"10"`, not a timestamp).

## Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | New `app/services/zaloreset` package, **not** an option on the email `passwordreset.Service` | Different OTP medium (push template vs magic-link email), different lookup key (mobile vs email), different "found vs not-found" timing-equalization (dummy `store.Create` vs dummy token). Mixing them produces a god-service. Same audit/security contract, separate type. |
| D2 | New `infra/cache/ZaloResetStore`, **not** reuse `OTPPendingStore` | OTP store binds session to IP/UA + enforces per-account lockout — both wrong for an unauthenticated reset. A focused 120-line store is cheaper than parameterizing the OTP store and risk-breaking login 2FA. |
| D3 | ZNS provider implements `domain.EmailDeliveryPort`-shaped **narrow port** `zalo.Sender` (not the email port itself) | Keeps the ZNS dependency behind an interface so the service is unit-testable with a fake. Avoids dragging the `domain.EmailMessage` shape into a phone channel. |
| D4 | One ZNS template `OTP-ZNS-v1` (id `617976`, tag=1 TRANSACTION) with 3 params | Approved-and-registered in the payroll OA. Hardcoded as the `ZALO_RESET_TEMPLATE_ID` default; env override kept for sandbox. All three params (`otp_code`, `user_fullname`, `otp_valid_in_minutes`) are populated; missing any returns Zalo `-1122`. |
| D5 | Credential + connection state stored in the **existing generic `Settings` key-value table**, not a new singleton table and not env-only | Two settings keys: `zalo.enabled` (bool) and `zalo.credentials` (JSON: `app_id`, `secret_key`, `access_token`, `refresh_token`, `expires_at`, `template_id`). Admin UI writes them; `Provider` reads them at runtime via `SettingsRepository.GetByKey`. Env vars (`ZALO_*`) become **bootstrap-only seed values** — on first boot, if the settings row is empty, seed it from env; thereafter DB is authoritative. This matches the existing OTP/notification settings pattern and lets the admin toggle without a redeploy. |
| D8 | OAuth v4 connect is an **admin-initiated flow with a backend callback**, not a CLI ritual | Admin clicks "Kết nối Zalo" → backend redirects to Zalo's `permission` URL → Zalo redirects to `/api/v1/admin/zalo/oauth/callback?code=...` → backend exchanges code for tokens, persists to `zalo.credentials`, redirects back to the settings tab. The callback route is **admin-authenticated** — it requires an admin session/JWT, so a harvested `code` can't be replayed by a third party. |
| D6 | `role = 'employee'` hard gate in the lookup | Admin/partner reset MUST stay on the audited email + 2FA path. Splitting channels per role keeps each role's blast-radius small. |
| D7 | **No per-account failed-attempt lockout in v1** | The rate limiter (per-mobile) + 10-min TTL + single-consume already cap brute force at ~3 requests/hour/mobile → 3 guesses, not 3×10⁶. Adding `OTPFailedAttempts` reuse would require schema changes on `users` for a non-login flow. Deferred — flagged in Risk R-Z3. |

## Open Questions

| # | Question | Status |
|---|---|---|
| ~~Q1~~ | ~~template_id for `password_reset`~~ | **RESOLVED:** `617976` (`OTP-ZNS-v1`), 3 params (`otp_code`, `user_fullname`, `otp_valid_in_minutes`). Hardcoded as the default for `ZALO_RESET_TEMPLATE_ID`; env override retained for sandbox. **Pending Zalo approval (Đang duyệt, 2–3 days) — blocks prod, not dev.** |
| ~~Q2~~ | ~~Same OA as tuyennhanvien.vn?~~ | **RESOLVED:** **Separate** payroll OA "Ting Ting Software Solution" / app "TingTing Soft". Phase 1 uses its own `ZALO_APP_ID`/`ZALO_SECRET_KEY`. |
| ~~Q3~~ | ~~Cost / budget cap?~~ | **RESOLVED:** 300 ₫/send via phone, 210 ₫/UID. No in-app budget cap (rely on Zalo OA quota `-144`/`-147` surfacing as "try again later"). Enumeration probes cost 0 ₫ (not-found = no send). At 300 ₫ and a 3/hour/mobile rate cap, worst-case friendly-fire cost is trivial; monitor the `zns_log`-equivalent for spikes. |
| Q4 | Should `/forgot-password` default the toggle to **Zalo** (most employees have no email) or **Email**? | **Default: Zalo** for the `/login` deep link; remember last choice in `localStorage`. Confirm at validate. |
| Q5 (new) | The OA console labels `otp_valid_in_minutes` as type **"date"** but the cap is 20 chars and the sample is `123456`. Confirm whether Zalo expects a numeric string (`"10"`) or a date. | **Default:** send the integer-minutes as a string (`"10"`). If Zalo rejects with `-1121`/`-1122` during sandbox smoke-test, revisit. This is the single most likely spec ambiguity to surface in Phase 1 testing. |
| Q6 (new) | What is the **production API host** the Zalo OAuth callback should target? (e.g. `https://api.tingting.vip` vs `https://tingting.vip/api`). | The callback URL is `https://<host>/api/v1/admin/zalo/oauth/callback`. Default to the existing backend host pattern; confirm with ops at Phase 4 implementation. This must match the redirect_uri registered in the Zalo OA console exactly (scheme + host + path). |
| Q7 (new) | Should the admin "Kết nối Zalo" (OAuth) flow be available **before** `zalo.enabled` is toggled on? (i.e., can the admin save keys + connect while the feature is still off, then flip the toggle?) | **Default: yes** — saving keys and running the OAuth connect are allowed while `zalo.enabled=false`; only the `/auth/zalo-reset/*` public endpoints check the toggle. This lets the admin verify the connection works before exposing the feature to employees. |

## Dependencies

- **External:** Zalo OA console — template `617976` (`OTP-ZNS-v1`) is **Đăng duyệt** (pending review, 2–3 business days as of 2026-08-04). **Blocks prod go-live only.** Dev/test can proceed immediately against the sandbox OA using the same template_id (Zalo returns `-127` "template test only sends to OA admins" in dev mode, which the service treats as a non-fatal send outcome). Re-check approval status before the Phase 9 go-live checklist sign-off.
- **External:** OAuth callback URL must be **registered in the Zalo OA console** as a valid redirect_uri. The callback is `https://<api-host>/api/v1/admin/zalo/oauth/callback`. The admin enters it once in the OA console (Phase 5 settings tab shows the exact URL to copy). Without this, Zalo rejects the connect flow with an invalid-redirect error. **Blocks the connect flow, not the code/tests.**
- **No external CLI ritual:** unlike the earlier draft, the OAuth handshake is done **through the admin UI** (Phase 4 + 5), not a separate `make` target. Phase 9 keeps a fallback runbook only for disaster recovery (lost refresh_token).
- **Cross-plan:** `plans/260724-2100-password-reset-email/` (pending, email magic-link). **No code conflict** — different package (`zaloreset` vs `passwordreset`), different route prefix (`/auth/zalo-reset/*` vs `/auth/password-reset/*`), different lookup column. Both can ship independently. Marked `blockedBy: []`.
- **No schema migration** to `users` — reuses `users.mobile` (existing, indexed, `uniqueIndex:unique_user_mobile_deleted_at`).
- **No new table** — reuses the existing generic `settings` key-value table (two new rows: `zalo.enabled`, `zalo.credentials`). The earlier draft's `zalo_oauth_credentials` singleton table is **dropped** in favor of this.

## Success Criteria (whole plan)

- [ ] An employee with a verified `users.mobile` can reset their password end-to-end from `/forgot-password` → ZNS OTP → new password → login, on desktop **and** mobile (390px).
- [ ] A mobile that is **not** in the DB returns the same `{session_id, message}` shape and approximate latency as a known mobile (anti-enumeration; verified by a timing test in Phase 6).
- [ ] A non-employee (admin/partner) mobile is rejected with the same generic response — never enrolled in the Zalo flow.
- [ ] Confirm consumes the code exactly once; a replay returns `401 invalid/expired`.
- [ ] On confirm success: password is hashed, all existing JWTs are invalidated in the same transaction, a `PasswordChangedEvent(method="zalo_reset")` is published, and the user must log in again.
- [ ] ZNS send failure (-118 no Zalo account, -115 quota, -127 sandbox-only, network) does **not** surface a specific error to the client and does **not** crash the process.
- [ ] Every ZNS send populates **all three** template params (`otp_code`, `user_fullname`, `otp_valid_in_minutes`); `user_fullname` is clamped to 30 chars; `otp_valid_in_minutes` reflects the configured `CodeTTL` in whole minutes. (Verified by the Phase 6 integration test asserting the stub received all three keys.)
- [ ] An admin can enter Zalo OA keys (`app_id`, `secret_key`, `template_id`) in `/admin/settings?tab=zalo`, click "Kết nối Zalo", complete the OAuth flow, and see the connection flip to "Đã kết nối" with a live `expires_at` — all from the UI, no env edits or redeploy.
- [ ] An admin can toggle `zalo.enabled` off at runtime; the public `/auth/zalo-reset/*` endpoints immediately stop dispatching ZNS (return the generic anti-enumeration 200 without sending), while the admin connection settings remain intact.
- [ ] The OAuth callback route rejects non-admin callers (403); a replayed `code` is rejected (single-use, state-param CSRF guard).
- [ ] Toggling `zalo.enabled` or rotating tokens does not require an api-server restart.
- [ ] `make api-test` green; new unit tests for store (atomic consume), service (request/confirm happy + error paths), provider (phone-norm, -124 retry), rate-limit key getter, and handler (anti-enumeration) all pass.
- [ ] `cd frontend && pnpm lint && pnpm type-check` green; desktop + 390px mobile verified per `AGENTS.md`.
