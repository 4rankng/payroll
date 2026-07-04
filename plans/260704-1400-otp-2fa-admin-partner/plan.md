---
title: "Email-OTP 2FA for admin and partner accounts (OTP_ENABLE flag)"
description: "Adds email-delivered one-time-code two-factor authentication for admin and partner roles, gated behind an OTP_ENABLE env flag (default off). Fixes the account-takeover gap exposed by the 2026-07-04 red-team audit: a leaked default password (Admin123) lets a stranger log in as admin. The email OTP makes a known password alone insufficient — the attacker must also control the victim's inbox. Threat model is ACCEPTED by the owner: email-OTP is the chosen v1 mechanism; email-account compromise is acknowledged as the residual bypass vector (mirrors the Google-login risk). Scope: backend OTP service + login gate, verify endpoint, middleware enforcement, and a frontend two-step login + code-entry UI. Employee and adv_partner roles are NOT in scope for v1."
status: pending
priority: P1
branch: "main"
tags: [security, auth, mfa, otp, email, audit-2026-07-04]
blockedBy: []
blocks: []
created: "2026-07-04T08:15:07.953Z"
createdBy: "ck:plan"
source: skill
---

# Email-OTP 2FA for admin and partner accounts (OTP_ENABLE flag)

## Overview

A red-team audit of `demo.tingting.vip` (2026-07-04) demonstrated that the
documented demo default password `Admin123` lets a member of the public log in
as the admin (`frankng`, user_id 1) and receive a valid 14-day JWT. Existing
defenses — password policy, IP rate limiting, the IDOR/token-revocation fixes in
commit `619f9cd` — do **not** stop this, because `Admin123` satisfies the policy
and is a *known* constant, not a guessed one.

This plan closes that gap with a second factor: a **one-time code emailed to the
user at login** (delivered via the existing Resend integration), required for
`admin` and `partner` roles when the `OTP_ENABLE` flag is on.

**Threat model answered (and accepted):** once a privileged account is
OTP-enabled, a known/stolen password alone is insufficient to authenticate — the
attacker also needs access to the victim's email inbox. The owner accepts the
residual risk that **inbox compromise (phishing, weak email password, no 2FA on
the email account itself) remains a bypass vector.** This is the same vector
that applies to Google-login; see RT-H5 in the red-team history below.

### Why email-OTP (not TOTP / SMS / WebAuthn)
- **Email** — chosen: zero enrollment friction (everyone has email), reuses the
  existing Resend integration, simplest UX for a VN construction SaaS.
- **TOTP** — rejected for v1: enrollment friction (QR + authenticator app) for a
  non-technical user base. (Original plan was TOTP; owner redirected to email.)
- **SMS** — rejected: SIM-swap vulnerable, NIST-restricted, cost, VN deliverability.
- **WebAuthn/passkeys** — phishing-resistant v2; tracked separately.

### Key technical decisions
- **Delivery:** async via Resend. Generate code → hash-store in Redis (5-min
  TTL) → send via `EmailDeliveryPort` → return `otp_session_id` immediately.
  Client submits code → server compares stored hash (constant-time).
- **Code:** 6-digit numeric, `crypto/rand`, single-use, 5-min TTL, per-account
  attempt cap (5) + 15-min lockout.
- **Pending-session model:** **Option B — Redis temp session** (NOT a pre-auth
  JWT). Login returns `{otp_required, otp_session_id}`; the real 14-day JWT is
  minted **only** after OTP verifies.
- **OTP data home:** NO persistent per-user secret table (unlike TOTP — no seed).
  Per-account lockout state (`otp_failed_attempts`, `otp_locked_until`) lives on
  the `users` table (mirrors the `tokens_invalid_before` pattern from 086). The
  code + attempts + expiry live in Redis only.
- **Gating:** global `OTP_ENABLE` env flag (default off) AND role∈{admin,partner}.
  Both must hold to require OTP.
- **Strict email requirement (owner decision):** once `OTP_ENABLE=true`, any
  admin/partner **without** an email on file **cannot log in** (clear error, not
  silent). App-level email verification is NOT checked for password-login users
  (no such flag exists; only Google-login checks `email_verified`).

## Phases

| Phase | Name | Status | Goal |
|-------|------|--------|------|
| 1 | [Lockout store & Redis pending-session](./phase-01-domain-migration.md) | Pending | per-account lockout fields (migration 087), Redis session helpers |
| 2 | [OTP service & login gate](./phase-02-backend-otp-service-login-gate.md) | Pending | code gen/send, `Login` gate + `/auth/login/verify`, gate Google login |
| 3 | [Resend / email template](./phase-03-enrollment-recovery-endpoints.md) | Pending | VN email template, `/auth/login/resend` endpoint |
| 4 | [Config & enforcement](./phase-04-middle-layer-enforcement-config.md) | Pending | `OTPConfig`, `OTP_ENABLE`, enforce-inside-`Authorize` + coverage test |
| 5 | [Frontend two-step login](./phase-05-frontend-two-step-login-enrollment.md) | Pending | code-entry screen + resend, types/endpoints, fork BOTH layers |
| 6 | [Hardening & verification](./phase-06-hardening-verification.md) | Pending | brute-force/lockout defenses, test matrix, red-team checklist, runbook |

## Critical path / dependency order

```
1 (lockout store) ──▶ 2 (service+gate) ──▶ 3 (resend/template) ─┐
                                                               ├──▶ 6 (hardening+tests)
                      4 (config+middleware) ────────────────────┘
                      5 (frontend) ── depends on 2&3 contract ──▶ 6
```

- **1 → 2 → 3** strictly sequential.
- **4** parallel with 2–3 but lands before 6.
- **5** after the API contract in 2–3 is frozen.
- **6** is the gate: no deploy until hardening + tests pass.

## Out of scope (deliberate)

- **Employee & adv_partner roles** — low-value, high rollout friction. v2.
- **WebAuthn / FIDO2 (passkeys)** — phishing-resistant v2; separate plan.
- **App-level email verification for password-login users** — only Google-login
  checks `email_verified` today; adding app-side verification is its own feature.
  For v1, "has an email on file" is the gate (owner decision).
- **TOTP / authenticator-app / backup codes** — superseded by email-OTP.
- **Per-account "force MFA" admin toggle UI** — the role+flag gate covers this.

## Red Team Review — Session 2026-07-04 (history; carried into email design)

A TOTP-version plan was red-teamed by two adversarial reviewers. The owner then
redirected to email-OTP. The findings below are **re-evaluated for the email
design** — most still apply (they concern login-gate/enforcement/frontend
discipline, not the TOTP mechanism), and remain folded in.

### Carried over to email-OTP (still apply — SAME fix)
| ID | Finding | Fix carried into |
|----|---------|------------------|
| RT-C1 | Money routes (`/wallet`, `/admin/manual-disbursement`) mounted on `v1`, not `protected` → group-middleware can't reach them | Phase 4: enforce INSIDE `Authorize()` + route-coverage test |
| RT-C2 | `otp_verified:true` stamped on password-only logins → silent downgrade | Phase 2/4: claim defaults false; set true ONLY in verify |
| RT-C3 | Frontend forks wrong layer (`useAuth.onSuccess` vs `auth.service.ts`) | Phase 5: fork in `auth.service.ts:login` AND `onSuccess` |
| RT-H1 | Per-session cap × N sessions = brute-force amplifier | Phase 2/6: one session/user + shared per-account counter |
| RT-H3 | `PUT /auth/me` resets password w/o current-password re-auth | Phase 6: reject `Password` field on `UpdateProfileRequest` |
| RT-H4 | Enrollment hijack via stolen pre-OTP token | N/A for v1 — email-OTP has no enrollment; if added later, require verified token |
| RT-H5 | Google-login permanent OTP bypass | Phase 2: gate `LoginWithGoogle` identically |
| RT-M1 | Rolling-deploy partial flag state | Fixed by RT-C2 (claim never true on password path) |
| RT-M2 | Lockout DoS on sole admin | Phase 6: exponential backoff + notify |
| RT-M4 | `RevokeUserTokens` doesn't kill Redis sessions | Phase 2/6: delete `otp:pending:user:<id>`; `/verify` re-checks revocation |
| RT-M5 | `otp_session_id` not bound to IP/UA | Phase 2: bind to IP+UA |
| RT-M8 | Force-enroll + no precondition → partial lockout | Reframed: "no email → cannot log in" is the gate; runbook checks all have email before flip (Phase 6) |

### Retired (TOTP-specific, no longer relevant)
- RT-C4 (`.UTC()` no-op), RT-M3 (skew replay), RT-M6 (`user_mfas` migration fiction),
  RT-M7 (AES-GCM key rotation), RT-M9 (backup-code leak), RT-H2 (backup-code defenses).

## Open questions (resolve before Phase 1)

1. **Google-login policy** — resolved by red-team: privileged roles require OTP
   even via Google login (gate `LoginWithGoogle` identically).
2. **Lockout storage** — extend `users` table (migration 087: `otp_failed_attempts`,
   `otp_locked_until`) vs. a new `user_otp_lockout` table. Recommend `users` (mirrors
   the `tokens_invalid_before` pattern from commit `619f9cd`). Confirm.
3. **"No email" handling at flip** — owner decided: hard-block login with a clear
   error. Runbook must ensure all admin/partner have an email before flipping.

## Deploy & rollback

- **Deploy:** ship with `OTP_ENABLE=false`; behavior is identical to today.
  The only schema change (lockout fields, migration 087) is additive.
- **Rollback:** flip `OTP_ENABLE=false` (requires restart; documented). Redis
  pending keys expire naturally (5-min TTL).
- **Lockout recovery:** admin resets `otp_locked_until=NULL,
  otp_failed_attempts=0` via SQL, or waits 15 min.

## Related

- **Trigger:** red-team audit 2026-07-04 (commit `619f9cd` closed the network
  IDOR/spray/revocation gaps; this plan closes the *password-alone* gap).
- **Sibling plan:** `260704-1027-security-remediation` — disjoint scope
  (money-path / RBAC / attendance). No file overlap.
- **Email infra (reuse):** `internal/infra/email/resend_provider.go`
  (production-wired), `internal/domain/email.go` (`EmailDeliveryPort`,
  `EmailMessage`), `internal/app/services/notification/email_service.go`,
  `FromEmail` default `payroll@1stop.app` (`config.go:307`).
