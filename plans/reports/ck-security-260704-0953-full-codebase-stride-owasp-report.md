# Security Audit Report — payroll (full codebase)

**Date:** 2026-07-04 · **Mode:** Audit-only (STRIDE + OWASP Top 10) · **Scope:** full monorepo (~333K LOC)
**Method:** 7 parallel read-only `security-reviewer` agents across attack-surface slices, then controller-synthesized + git-verified.
**Stack:** Go + Gin + GORM (MySQL) + Casbin RBAC + JWT + asynq/Redis; React 18 + Vite PWA; OnePay disbursement (prod) / 9Pay (sandbox).

> **Verification note:** Two agent findings conflicted on whether `.env` files are git-tracked. Controller verified directly: only `backend/.env.example` is tracked; `frontend/.env`, `backend/.env`, `.env` are all gitignored (no history leak). One CRITICAL claim (DockerHub password "committed") was refuted and downgraded. All secret values below are masked.

---

## Summary

| Severity | Count | Bar |
|---|---|---|
| **Critical** | 0 | Exploitable now → breach/RCE. None confirmed at that bar. |
| **High** | 13 | Exploitable, significant impact — fix this sprint. |
| **Medium** | 25 | Limited exploitability / impact — next sprint. |
| **Low / Info** | ~20 | Defense-in-depth / hygiene. |
| **Verified safe** | 16 | Reconfirmed intact — no action. |

**Headline:** Auth/crypto core is sound (Argon2id + pepper, HS256 alg-confusion prevented, SQLi ORDER BY chokepoint intact, path-traversal blocked, no SSRF, OnePay HMAC core correct, wallet optimistic-locking + 70% cap race-safe). The real risk concentrates in **(a) money-path transaction/idempotency gaps**, **(b) client-trusted GPS on attendance**, **(c) one committed provider credential + weak local secrets**, and **(d) a still-live admin override route that bypasses every gate**.

---

## ⚠️ Fix today (top of High — borderline Critical)

| # | Finding | Why it leads | File:Line |
|---|---|---|---|
| **H1** | OnePay partner key committed in **tracked** `backend/.env.example` — value matches the live `backend/.env` key | Provider credential in the repo (sandbox today, but proves live-secret leakage into tracked files) | `backend/.env.example:143` |
| **H2** | Dev compose binds MySQL/Redis/Adminer/sandbox to `0.0.0.0`; Redis has **no password**, MySQL uses literal `payroll_pass` | Anyone on LAN/VPN/wifi reads/writes the dev DB + asynq queues | `backend/docker-compose.dev.yml:11-21,40-46,64-69,99-104` |
| **H3** | Attendance trusts client-reported lat/lng/accuracy/gps_at with no anti-replay or corroboration | Worker spoofs coords from home → earns shift + 70% advance quota → **payroll fraud**. *Escalates to Critical if `a3f892c`+`bd2b9d1` are not actually deployed to prod* (both confirmed in source) | `attendance_service.go:294-320`; handlers `attendance.go:153-212` |
| **H5** | FlexPay settlement: status UPDATE + ledger create **not in one transaction**, no idempotency key, ledger errors swallowed → returns `Success:true` | Double-credit / out-of-balance books; silent money drift on crash mid-flow | `flex_pay_settlement_service.go:115-188` |
| **H6** | `RetryDisbursement` has no in-flight dedup — each retry mints a new request ID + asynq task | Admin double-click / two admins → **double-pay** the same advance | `retry_disbursement_handler.go:39-92` |

---

## High findings (this sprint)

| # | STRIDE / OWASP | File:Line | Finding | Fix |
|---|---|---|---|---|
| H1 | Disclosure / A05 | `backend/.env.example:143` | OnePay partner key `C2B5<REDACTED>` committed (matches live env). | Remove line; rotate in OnePay sandbox portal; add `gitleaks` to CI; scrub git history if repo is shared externally. |
| H2 | Disclosure / A05 | `backend/docker-compose.dev.yml` | MySQL/Redis/Adminer/sandbox on `0.0.0.0`; Redis no password; MySQL `payroll_pass`. | Bind `127.0.0.1:`; add Redis `--requirepass`; use internal docker network for service-to-service. |
| H3 | Spoofing / A03,A07 | `attendance_service.go:294-320`; `attendance.go:153-212` | Client-trusted GPS, no anti-replay. | Cross-check `gps_at` within ±60s of server clock; flag `accuracy<5m` repeats; sign check-in payload; long-term device attestation. **Verify `a3f892c`+`bd2b9d1` are deployed.** |
| H4 | DoS / A04 | `advance_payment/file_handler.go:187-204` (+lines 449,509,910) | 4 upload handlers `io.ReadAll` with **no** `ValidateFileSize`. Other handlers cap 10MB. | `storage.ValidateFileSize(...,10MB)` before read; or `io.LimitReader`; set global `router.MaxMultipartMemory`. |
| H5 | Tampering / A01,A04 | `flex_pay_settlement_service.go:115-188` | No tx wrapping UPDATE+ledger; no file-hash idempotency; errors swallowed → `Success:true`. | Wrap in `db.Transaction`; SHA-256 file hash w/ unique index; propagate ledger errors; add outbox event. |
| H6 | Tampering / A01 | `retry_disbursement_handler.go:39-92` | Retry mints new request ID + task; no in-flight check. | Query non-terminal `wallet_payments WHERE advance_request_id=?` → 409 if any; bind disbursement ID to `(advance_request_id, amount)`. |
| H7 | Tampering / A01 | `tx_wallet_payment_repository.go:273-292` | `MarkReconciled` bare `WHERE id=?` — bypasses FSM, no version/status precondition. | Add `WHERE id=? AND status IN (...)` or route via `UpdateExpected`. |
| H8 | Spoofing / A07 | `auth_service.go:471-556` (`LoginWithGoogle`) | No `email_verified` check, no `hd` allowlist. | Require `payload.Claims["email_verified"]==true`; enforce hosted-domain if Workspace. |
| H9 | Spoofing / A07 | `rate_limit.go:69-74`; `auth_service.go:96-132` | Login throttle 1000/min/IP, no per-account lockout; multi-identifier resolver widens enumeration. | 5-10/min/IP + Redis per-account lockout (5 fails → 15 min); throttle `/auth/google`, change-password, reset. |
| H10 | EoP / A01 | `casbin_policy.csv:118-119` | `adv_partner` granted `PUT /api/v1/users/*`; user routes have no ownership middleware. | Drop `users/*,PUT` for adv_partner or scope to own user; add `CanUserModifyUser(actor,target)` in handler. |
| H11 | Supply chain / A06,A08 | `backend/Dockerfile:3,30`; `.github/workflows/ci-cd.yml:189,196,238` | Docker base images use mutable tags (no digest); CI Actions pinned by tag (e.g. `appleboy/ssh-action@v1`) not SHA. | Pin by `@sha256:`/commit-SHA; add `trivy image`/`docker scout`; StepSecurity Harden-Runner. |
| H12 | DoS / A05 | `config.go:266-268,345`; `docker-compose.dev.yml` | No container mem/cpu limits; no global `MaxMultipartMemory`. 1GB droplet OOM wedges sshd (prior incident). | `deploy.resources.limits`; `router.MaxMultipartMemory=32<<20`; OOM-adjust sshd. |
| H13 | Disclosure / A02 | `backend/.env:8`; `backend/.env` HASH_SECRET/SALT; perms 0644 | Weak `JWT_SECRET=abc<REDACTED 6>`; `HASH_SECRET`/`HASH_SALT` 8 chars each (weak pepper); `.env`/`backend/.env`/`frontend/.env` world-readable. | Regenerate ≥32-char random (pepper ≥32); force password reset; `chmod 600`. *(gitignored — local only, not in history.)* |

---

## Medium findings (next sprint)

**Attendance / geofence**
- M1 `routes_admin.go:63` + `admin/attendance.go:148-222` — admin override route `POST /admin/attendances/failed-attempts/:id/override` **still live** despite UI button removal; bypasses geofence/window/`CheckInEnabled`, skips auto-reject scheduling. → Gate behind a named permission + freshness bound + verify `reason_category` is system-written, or remove the route.
- M2 `attendance.go:61-97,240-258` — `LogDeviceAttempt` accepts arbitrary client `gps_status` (no allow-list) → worker can seed fake failed-attempts to social-engineer override. → Validate enum; rate-limit per employee.
- M3 `admin/attendance.go:148,228,257` — approve/reject/override have no per-row project-scope check (any admin on any project). → Add `projectID ∈ admin.AccessibleProjects` if multi-tenant.
- M4 `infra/asynq/client.go:342` — auto-reject task payload `{"attendance_id":N}` unsigned (bounded by Redis trust). → HMAC-sign payload; worker honors only past deadline.

**Auth / RBAC**
- M5 `auth_service.go:327-358` — password change blacklists only current JTI; other sessions live up to 14d. → Per-user `token_version` bumped on password change.
- M6 `middleware/auth.go:50-55` — role claim not validated against canonical `domain.Role*` set. → Reject unknown roles with 401.
- M7 `middleware/authorization.go:163-195` — `extractProjectID`/`extractEmployeeID` re-parse `URL.Path` via `strings.Split`, no path-unescape → URL-encoded IDs may skip partner ownership check while Casbin still allows `/*`. → Use `c.Param("id")`; unescape; treat unparseable as unauthorized.

**Money path**
- M8 `onepay/webhook.go:87-94` — IPN amount not cross-checked vs persisted request at adapter; `Int64()` error discarded. → Assert `event.Amount==persisted.Amount`; surface parser error.
- M9 `onepay/webhook.go:60-70` — replay window attacker-controllable via `X-OP-Expires` (±900s, negative skew allowed); no nonce store. → Nonce/JTI TTL store; tighten ±60s; cap expires.
- M10 `wallet_payment_service.go:153-169` — `HasPendingForRecipient` TOCTOU; no DB partial unique index. → Add `UNIQUE(recipient_account_no,recipient_bank,provider) WHERE status NOT IN (terminal)`.
- M11 `admin_bulk_transfer.go:190-283` — `ProcessBankResult` status COMPLETED update + ledger create not in one tx; no event. → Wrap in tx; publish event.
- M12 `fee_schedule_service.go:279-302` — `ResolveFee` fail-open: missing OnePay schedule → fee=0, warn only. → Fail-closed or alert.
- M13 `onepay/client.go:67-70` — `Config.Endpoint` no HTTPS enforcement. → Assert `https://` unless explicit flag.

**PII / logging**
- M14 `onepay/client.go:191-228` + `manual_disbursement_handler.go:293-323,387-393` — Info logs dump full request/response bodies (bank account nos., holder names) + signing debug (canonical_request, signature, auth header). → Drop to Debug / redact PII fields.

**Input / DoS**
- M15 `manual_disbursement_handler.go:107-115` — `amount` only `binding:"required"`, no positive guard. → `binding:"required,gt=0"` + explicit check.
- M16 `transaction_repository.go:407-412`; `common/filter_builder.go:142` — unbounded `LIMIT`/`OFFSET` interpolation (integer-safe, but DoS via huge limit). → Clamp to `MaxPageSize`.
- M17 `backend/go.mod` — Go stdlib CVEs **GO-2026-5039** (net/textproto) + **GO-2026-5037** (crypto/x509), fixed in go1.26.4. → Bump toolchain + Dockerfile to 1.26.4.
- M18 `seed/admin.go:22`; `employee/user_service.go:15`; `password_reset_job.go:166` — hardcoded `Vfic1234@` / `DefaultEmployeePassword="Vfic@1234"`; short prod MySQL root (9 chars). → Env-driven per-user temp passwords; refuse to seed in prod.

**Config / infra**
- M19 `backend/Makefile:152,167-169` — `make restore`/`make demo-db` reset all passwords to `Admin123`. → Random per-reset password printed once; require `RESET_PASSWORDS=1`.
- M20 `config.go:265,453` — silent `DB_DSN` default; `DB_DSN`/`VAPID`/`RESEND` not validated at boot; 14d JWT TTL hardcoded (drifts from `.env.example`'s 7d). → Validate in `validate()`; shorten access TTL + refresh tokens.

**Frontend**
- M21 `lib/auth.ts:37,44`; `AuthContext.tsx:48-56` — JWT in `localStorage` (XSS-exfiltratable). → Migrate to HttpOnly+Secure+SameSite cookie.
- M22 `index.html` + `vite.config.ts` — **no CSP** anywhere. → Strict CSP via nginx `add_header`.
- M23 `lib/realtime.ts:71` — WS `?token=` query leaks JWT via proxy/server logs. → Subprotocol header or one-shot ticket; redact in nginx.
- M24 `Login.tsx:55-66` — Google OAuth implicit flow (`id_token` in fragment). → Auth-code + PKCE.
- M25 `NotificationProvider.tsx:79` — push-payload `url` open-redirect (no same-origin check). → `new URL(url,origin)` origin check before `location.href`.

---

## Low / Info (condensed — backlog/defense-in-depth)

- HSTS only when `c.Request.TLS!=nil` → never set behind nginx (`security_headers.go:23-25`). Move to nginx.
- CSP allows `unsafe-eval` + broad CDN allowlist (cdnjs, sheetjs) — add SRI (`security_headers.go:35`).
- Login-success + failed-attempt writes are fire-and-forget goroutines (`context.Background()`) — lost on shutdown/DB outage (`auth_service.go:154-181,516-548`; `attendance.go:61-97`). Use outbox.
- Auth timing side-channel: user-found runs Argon2, not-found doesn't (`auth.go:42-55`). Dummy-hash for not-found.
- JWT no `iss`/`aud` claims; `logAuthorizationSuccess` is a no-op stub (`auth_service.go:225-243`; `authorization.go:138-147`).
- File storage `0644`/`0755` on shared host (`infra/storage/file_storage.go:87,95,110,142`). → `0600`/`0700`.
- Excel parse no row-count cap (`bcc_import_service.go:135` + 8 sites). Cap `len(rows)`.
- `SanitizeFilename` doesn't strip CR/LF (`storage.go:208`). Strip `\r\n`.
- Webhook wrapped errors form a signature oracle (`onepay/webhook.go:56,80`).
- Reconcile idempotency key uses **sum of IDs** → collisions (`reconcile_service.go:117-118,522-528`). Use sorted join/hash.
- `MarkReconciled` re-fetch error discarded (`wallet_payment_service.go:728-770`).
- Hardcoded bank account `1357210887` / holder in `email_template.go:8-35`. Move to config.
- `admin_export.go:64-78` auto-approves pending requests during export (no separation of duties).
- `onepay/client.go:110` `requestID` not `url.PathEscape`'d; verify canonical URI hardcoded literal (`signing.go:160-178`).
- Frontend triple lockfile drift (pnpm+yarn+bun) — Docker `--frozen-lockfile` hazard (prior incident). Consolidate.
- DockerHub account password in `frontend/.env` (gitignored, **not** committed) — use access token + 2FA.
- Frontend dev-mode console logging (`client.ts:106,131`); sourcemaps emitted in non-prod/demo builds (`vite.config.ts:108`).
- `.env.docker` weak default placeholder (`ROOT_PASSWORD=root_password_change_me`).

---

## Verified safe — no action (reconfirmed under re-audit)

- **ORDER BY SQLi chokepoint intact**: `SanitizeSortColumn`/`SanitizeSortOrder` + `FilterBuilder.ApplySorting` cover all 25+ `.Order()`/`ORDER BY` sites; regex rejects injection; no bypass via intermediate vars. *(systemic fix held)*
- **Path traversal blocked**: `resolveSafePath` clean+prefix-check; uploads UUID-named; `c.File()` uses DB-stored paths; no request-param file paths.
- **No SSRF**: OnePay/9Pay/maps use fixed endpoints; no user-controlled outbound URL.
- **JWT alg-confusion prevented** (explicit `HS256` double-check); **Argon2id strong** (64MB/3/2/32 + pepper + constant-time); **OnePay XFF spoof = false positive** reconfirmed (gin rightmost-IP + `TrustedProxies=[loopback]`).
- **Casbin deny-overrides correct**; partner scope (project + employee chain) sound.
- **Wallet optimistic locking sound** (`UpdateExpected` atomic `WHERE id=? AND version=?`); **70% advance cap race-safe** (`CreateWithBudgetCheck` tx + `FOR UPDATE`); **OnePay HMAC-SHA256 chained signing core correct** (`hmac.Equal`).
- CORS origin-allowlist + credentials (safe); security-headers middleware wired first on router.
- Dockerfile runs as `nobody`, multi-stage `scratch` final (no shell/source/creds baked).
- **Frontend `pnpm audit`: 0 CVEs across 983 deps**; good `overrides` hygiene; VAPID private key server-side only; SW never caches `/api`, same-origin `notificationclick`.
- `MarkReceivableSettled` idempotent (`WHERE receivable_settled_at IS NULL`).

---

## Remediation roadmap

**Week 1 — money + identity integrity**
1. Wrap FlexPay settlement in a DB transaction + file-hash idempotency + propagate ledger errors (closes H5 cluster).
2. In-flight dedup on `RetryDisbursement` + bind disbursement ID to `(advance_request_id, amount)` (H6).
3. `MarkReconciled` status precondition / route via `UpdateExpected` (H7).
4. `LoginWithGoogle` enforce `email_verified` (H8).
5. Add `ValidateFileSize` to the 4 FlexPay upload handlers + global `MaxMultipartMemory` (H4/H12).
6. Verify `a3f892c`+`bd2b9d1` deployed to prod; if not, deploy immediately (H3 gate).

**Week 2 — secrets + supply chain + RBAC**
7. Remove OnePay key from `.env.example`; rotate; add `gitleaks` CI (H1).
8. Bind dev compose ports to loopback + Redis password (H2).
9. Regenerate `JWT_SECRET`/`HASH_SECRET`/`HASH_SALT` (≥32 chars); `chmod 600`; force password reset (H13).
10. Pin Docker images by digest + CI Actions by SHA (H11).
11. Drop `adv_partner` `PUT /users/*` grant; add ownership middleware (H10).
12. Login throttle 5-10/min + per-account lockout (H9).

**Week 3 — attendance hardening + frontend tokens**
13. Decide on admin override route (gate or remove); validate `gps_status` enum (M1/M2).
14. Migrate frontend JWT to HttpOnly cookie; add strict CSP; fix WS token-in-URL (M21-M23).
15. Tighten IPN replay window + amount cross-check; OnePay PII redaction in logs (M8/M9/M14).
16. Bump Go to 1.26.4 (M17).

---

## Unresolved questions

1. **Is `flex_pay_settlement_service.go` the live settlement path?** Memory note `lesson_sao_ke_upload_silent_settlement_drop` describes a *synchronous* `SettlementApplier` (commit 3beb3c6) — possibly a newer replacement. The H5 cluster targets the older-looking service; confirm wiring before remediation.
2. **Are `a3f892c` (accuracy gate) + `bd2b9d1` (checkout gate) actually deployed to prod/demo?** They are confirmed in source. If the running binary predates them, H3 (GPS spoofing) escalates to Critical.
3. **Admin override route (M1):** was the UI removal intended to fully retire the feature, or just hide the button? If retire, the route + `RecordAdminCheckIn` should be deleted; if retain, it needs a named permission + freshness bound.
4. **Is `admin` a single trusted-super-admin role, or multi-tenant?** Determines whether M3 (per-row project scope) is a real gap or by-design.
5. **OnePay key at `.env.example:117` (uncommented) vs `:143` (commented, `C2B5...`):** confirm which is the active value and whether either is a production-usable credential (Agent 7 reported `TESTVFICPO` = sandbox).

---
*Generated by `/ck:security` (audit-only). 7 parallel `oh-my-claudecode:security-reviewer` agents + controller git-verification. No code was modified.*
