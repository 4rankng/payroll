---
phase: 6
title: "Hardening & verification"
status: pending
priority: P1
effort: "M (1d)"
dependencies: [2, 3, 4, 5]
---

# Phase 6: Hardening & verification

## Overview

The gate before deploy: per-account brute-force defenses (the shared counter,
exponential lockout backoff, lockout notification), the test matrix that proves
the threat model holds, the red-team checklist, and the rollout/rollback runbook.
Nothing ships until this phase passes.

## Requirements

- **Functional:** per-account lockout (not just per-session/IP), one-session-per-user
  verified, the `UpdateProfile` password-reset hole closed (RT-H3).
- **Non-functional:** the audit demonstrates that a known password + no inbox
  access = no login; coverage of brute-force, lockout, and the money-route gate.

## Architecture

This phase is enforcement refinements deferred from Phase 2 + the test suite +
the adversarial checklist. No new components.

### Per-account lockout (refining Phase 2's counter)
Phase 2 bumps `users.otp_failed_attempts` on each verify fail and locks at 5.
Phase 6 adds:
- **RT-M2 (lockout DoS):** exponential backoff on repeated locks — track a
  `otp_lock_count` (or derive from a timestamp window); each consecutive lock
  doubles `locked_until` up to a cap (e.g. max 2h). Promote a lockout-notification
  email to v1 for admin accounts (so the real user learns of an attack/DoS).
  The existing per-IP `LoginRateLimit` (10/min) caps how fast an attacker can
  drive locks.
- On successful verify: reset `otp_failed_attempts = 0` AND `otp_locked_until = NULL`.

### Tests (Go)
| Test | Proves |
|------|--------|
| `TestLogin_OTPOff_IssuesTokenImmediately` | flag off = no regression |
| `TestLogin_OTPOn_Admin_ReturnsOTPSession` | the gate fires |
| `TestLogin_OTPOn_Employee_NotGated` | only admin/partner |
| `TestLogin_OTPOn_NoEmail_Returns401` | RT-M8 (strict email requirement) |
| `TestLogin_OTPOn_RepoError_Returns500` | RT-C2 fail-closed (never password-only) |
| `TestLogin_OTPOn_GoogleLogin_Admin_Gated` | RT-H5 (Google login gated too) |
| `TestVerifyLoginOTP_ValidCode_IssuesToken_ClaimTrue` | happy path + claim set |
| `TestVerifyLoginOTP_WrongCode_IncrementsPerAccount` | RT-H1 shared counter |
| `TestVerifyLoginOTP_ParallelSessions_SharedLockout` | RT-H1 (N sessions ≠ N×attempts) |
| `TestVerifyLoginOTP_FiveFails_Locks_PerAccount` | brute-force defense |
| `TestVerifyLoginOTP_LockedAccount_CannotStartSession` | lockout checked at login |
| `TestVerifyLoginOTP_SecondLoginInvalidatesFirstSession` | RT-H1 one-session/user |
| `TestVerifyLoginOTP_IPUAMismatch_Rejected` | RT-M5 session binding |
| `TestVerifyLoginOTP_RevokedDuringSession_Rejected` | RT-M4 (tokens_invalid_before re-check) |
| `TestGenerateCode_Distribution` | unbiased 6-digit over 10k samples |
| `TestResend_Cooldown30s` | resend throttled |
| `TestResend_InvalidatesPriorCode` | new code, old won't verify |
| `TestAuthorize_OTPVerifiedFalse_WalletRoute_403` | RT-C1 (money route covered) |
| `TestAuthorize_OTPVerifiedFalse_ManualDisbursement_403` | RT-C1 (money route covered) |
| `TestAuthorize_OTPOff_NoEnforcement` | kill-switch |
| `TestRouteCoverage_PrivilegedPathsHaveOTPGate` | RT-C1 build-time guard |
| `TestUpdateProfile_RejectsPasswordField` | RT-H3 (takeover persistence primitive) |
| `TestRevokeUserTokens_DeletesPendingOTPSession` | RT-M4 |

### Frontend tests
- `useLogin` forks correctly on `otp_required` (RTL): no `setToken`, no `login()`.
- `AuthManager` does not treat `otp_session_id`/empty as a valid token.

## Related Code Files

- **Modify:** `internal/app/services/otp/otp_service.go` — exponential lockout backoff + lockout email.
- **Modify:** `internal/app/services/auth/auth_service.go` — `UpdateProfile` rejects the `Password` field (RT-H3, at `:473`).
- **Create:** `internal/app/services/otp/otp_service_test.go` — the test matrix.
- **Create:** `internal/app/services/auth/auth_otp_test.go` — login-gate tests.
- **Create:** `backend/tests/integration/otp_flow_test.go` — end-to-end (per `backend/AGENTS.md` patterns).
- **Modify:** `frontend/src/hooks/api/useAuth.test.ts` (or create) — OTP fork test.

## Implementation Steps

1. **Exponential lockout** in `OTPService`: on the Nth consecutive lock within a
   window, `locked_until = now + min(15m * 2^(N-1), 2h)`. Send a lockout email to
   the user (admin accounts especially).
2. **RT-H3: reject `Password` on `UpdateProfileRequest`.** Force password changes
   through `/auth/change-password` (which re-auths). File `auth_service.go:473`.
   Also note in the sibling security-remediation plan.
3. **Write the Go test matrix** via the integration-test harness (`make api-test`).
4. **Concurrent-session test** (`TestVerifyLoginOTP_ParallelSessions_SharedLockout`):
   start one login (session A), start another for the same user (invalidates A —
   RT-H1), submit wrong codes against any session and confirm the per-account
   counter climbs once per attempt regardless of which session id is presented.
5. **Red-team checklist** (run against local stack):
   - [ ] Known password (`Admin123`) + no inbox access → cannot reach dashboard.
   - [ ] **Money routes covered (RT-C1):** known-password token with
         `otp_verified:false` → `POST /api/v1/admin/manual-disbursement` and
         `/api/v1/wallet/...` return 403.
   - [ ] **Google-login admin gated (RT-H5):** Google login for admin returns
         `otp_required`, not a token.
   - [ ] 5 wrong codes → account locked 15 min; N parallel sessions do NOT raise
         the budget (RT-H1).
   - [ ] `OTP_ENABLE=false` → full bypass (kill-switch works).
   - [ ] **UpdateProfile password-field rejected (RT-H3).**
   - [ ] **No-email admin blocked with a clear message (RT-M8).**
   - [ ] Resend cooldown prevents inbox flooding.
6. **Rollout + ops runbook** (append to `docs/`):
   - Deploy code with `OTP_ENABLE=false` (no behavior change).
   - Run migration 087 (additive, safe). Apply via `docker exec ... mysql` on dev;
     demo gets it via `make demo-db` mysqldump (there is no `cmd/migrate`).
   - **RT-M8 precondition:** before flipping, run
     `SELECT id, username, email FROM users WHERE role IN ('admin','partner') AND (email IS NULL OR email = '')`
     and ensure every privileged user has an email. Any without → they CANNOT log
     in once the flag flips (owner decision: hard block).
   - Set `OTP_ENABLE=true` (+ ensure `RESEND_API_KEY`, `EMAIL_FROM` set), restart.
   - Verify: a known-password login now demands OTP; money routes 403 without it.
   - **Lockout recovery (runbook):** `docker exec <mysql> mysql -e "UPDATE users
     SET otp_failed_attempts=0, otp_locked_until=NULL WHERE id=<uid>" payroll_db`.
   - **Rollback:** set `OTP_ENABLE=false`, restart. Pending Redis sessions expire in 5 min.

## Success Criteria

- [ ] All Go tests in the matrix pass (`-race`), incl. money-route coverage
      (RT-C1), parallel-session lockout (RT-H1), and no-email gate (RT-M8).
- [ ] `TestRouteCoverage_PrivilegedPathsHaveOTPGate` passes and fails when a
      privileged route is added without the gate.
- [ ] Integration test `otp_flow_test.go` passes end-to-end.
- [ ] Red-team checklist (step 5) — every box checked, no bypass found.
- [ ] `make api-test` green; `pnpm lint && pnpm type-check` green.
- [ ] Rollout runbook reviewed; a dry-run on demo succeeds (flip → known-password
      demands OTP → money routes 403 without OTP → flip back).

## Risk Assessment

- **Risk:** per-account lockout creates a DoS vector (attacker locks the admin).
  **Mitigation:** exponential backoff caps the lock; lockout-notification email
  alerts the real user; per-IP rate limit caps the lock-trigger rate. Accepted residual.
- **Risk:** email deliverability failure (spam filter) → legit user can't get the
  code. **Mitigation:** resend button + cooldown; runbook covers Resend domain
  verification; SPF/DKIM on the `1stop.app` sending domain (ops check).
- **Risk:** RT-H3 fix (`UpdateProfile` rejects password) breaks a frontend flow
  that currently uses it. **Mitigation:** grep the frontend for password changes
  via `PUT /auth/me`; route any through `/auth/change-password` instead. Phase 6
  includes this check.
- **Risk:** inbox compromise is the residual bypass (accepted by owner).
  **Document:** the runbook notes that admin/partner emails should be on
  accounts with their own strong passwords / 2FA; a future WebAuthn v2 closes this.
