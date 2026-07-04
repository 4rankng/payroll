---
phase: 3
title: "Resend code & email template"
status: pending
priority: P2
effort: "S (0.5d)"
dependencies: [2]
---

# Phase 3: Resend code & email template

## Overview

Email-OTP has no enrollment lifecycle (unlike TOTP). This phase covers the two
remaining pieces: (1) the Vietnamese-localized OTP email template rendered by
`EmailDeliveryPort`, and (2) a `/auth/login/resend` endpoint so a user whose
email didn't arrive (latency, spam filter) can request a new code without
re-entering their password.

## Requirements

- **Functional:** a clear, branded, VN-language OTP email; a resend endpoint
  that re-issues a code (invalidating the prior pending session — RT-H1's
  one-session-per-user rule) and respects the per-account lockout.
- **Non-functional:** resend is rate-limited (per-session + per-IP); email body
  does not leak whether the account exists beyond what login already reveals.

## Architecture

### Email template
Reuse `domain.EmailMessage` + the existing `FromEmail` (`payroll@1stop.app`,
`config.go:307`). Build the message in `OTPService.SendOTPEmail` (Phase 2):
- `Subject`: "Mã xác thực đăng ký — Payroll" (or similar VN).
- `TextBody`: short — "Mã đăng nhập của bạn là: 123456. Mã có hiệu lực 5 phút.
  Nếu bạn không yêu cầu, vui lòng bỏ qua email này."
- `HTMLBody`: a small branded template (table, logo placeholder, large code,
  expiry note). Keep it simple — no external images (CSP + deliverability).
- **Never** include the `otp_session_id` in the email (it's a session token, not
  a user-facing value). Only the 6-digit code goes in the email.

### `/auth/login/resend` flow
```
POST /auth/login/resend {otp_session_id}
  ├─ GetSession(otp_session_id) → must exist & not expired
  ├─ RT-M5: validate ip+ua match the session
  ├─ per-session resend cooldown (e.g. 30s; stored in the session JSON)
  ├─ re-check otp_locked_until on the user
  ├─ generate new code → overwrite the session (RT-H1: one per user)
  └─ send email; return 200 {otp_session_id (may be new), expires_in:300}
```
The endpoint is **unauthenticated** (carries `otp_session_id` like `/verify`).
Apply `LoginRateLimit`. The resend cooldown prevents email-flooding a victim's
inbox (a mild DoS/amplification concern).

## Related Code Files

- **Create:** `internal/app/services/otp/template.go` — `BuildOTPEmailMessage(code, user, fromEmail) *domain.EmailMessage`.
- **Modify:** `internal/app/services/otp/otp_service.go` — `SendOTPEmail` uses the template; add `ResendCode(ctx, sessionID, ip, ua) (newSessionID string, err error)`.
- **Modify:** `internal/transport/http/handlers/auth.go` — add `ResendOTPCode` handler.
- **Modify:** `internal/app/dto/user.go` — `VerifyOTPRequest{OTPSessionID, Code string}`, `ResendOTPRequest{OTPSessionID string}`, `OTPStartedResponse{OTPRequired, OTPSessionID string, ExpiresIn int64}`.
- **Modify:** `internal/app/bootstrap/routes_auth.go` — `auth.POST("/login/resend", LoginRateLimit, Auth.ResendOTPCode)`.
- **Modify:** `internal/infra/cache/otp_pending_store.go` — add `last_resend_at` to the session JSON + a `TouchResend(ctx, sessionID, newCodeHash) error` helper.

## Implementation Steps

1. **Template** (`template.go`) — VN strings, simple HTML, no external assets.
   Unit-test the builder produces a valid `EmailMessage` (Validate passes) and
   contains the code.
2. **`ResendCode`** — load session, IP/UA check, cooldown check (≥30s since last
   resend), lockout re-check, generate new code, overwrite session hash +
   `last_resend_at`, send email. Returns the (possibly new) `sessionID`.
3. **Handler + route** — bind `ResendOTPRequest`, call service, return
   `OTPStartedResponse`. Apply `LoginRateLimit`.
4. **Cooldown** — store `last_resend_at` in the Redis session JSON; reject if
   `now - last_resend_at < 30s` with a "thử lại sau" message.

## Success Criteria

- [ ] The OTP email arrives (Resend sandbox) with the 6-digit code in both text + HTML.
- [ ] The email does NOT contain `otp_session_id` or any session token.
- [ ] `/auth/login/resend` issues a new code; the prior `otp_session_id`'s code no longer verifies.
- [ ] Resend within the 30s cooldown → 429/400 with the cooldown message.
- [ ] Resend for an expired/invalid `otp_session_id` → 401.
- [ ] Resend from a mismatched IP/UA → 401 (RT-M5).
- [ ] A locked account cannot resend.

## Risk Assessment

- **Risk:** email-flooding via rapid resends → inbox DoS on the victim.
  **Mitigation:** 30s cooldown + per-IP `LoginRateLimit`.
- **Risk:** resend reveals account existence. **Mitigation:** `/resend` requires
  a valid `otp_session_id` (which only exists after a successful password login),
  so it leaks nothing login didn't already reveal.
- **Risk:** template breaks in odd email clients. **Mitigation:** keep it table-based,
  no external images; test in Gmail + the Resend preview.
