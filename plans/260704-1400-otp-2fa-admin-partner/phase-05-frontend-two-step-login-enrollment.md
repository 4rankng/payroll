---
phase: 5
title: "Frontend two-step login"
status: pending
priority: P1
effort: "S-M (0.5-1d)"
dependencies: [2, 3]
---

# Phase 5: Frontend two-step login

## Overview

The user-facing half: a two-step login flow (password → emailed code entry), with
a resend button. No QR, no enrollment UI (email-OTP has no enrollment). Scoped to
admin/partner; employees see no change.

## Requirements

- **Functional:** when login returns `otp_required`, show the code-entry screen
  instead of navigating. Provide a resend button (Phase 3 endpoint). Show a
  countdown for the resend cooldown.
- **Non-functional:** `otp_session_id` is NOT an auth token — never stored in
  `auth_token` localStorage. The fork must live in **both** the service layer and
  the hook (RT-C3).

## Architecture

### Two-step login fork — MUST be in `auth.service.ts` AND `useAuth.onSuccess`

> ⚠️ **RT-C3.** The original plan forked only `useAuth.onSuccess`. Wrong layer:
> token storage happens in `auth.service.ts:login` (the mutationFn), which runs
> BEFORE `onSuccess`. For an OTP response `access_token` is `""` (omitempty), so
> the `setToken` guard is false — BUT `useAuth.ts:25` then unconditionally calls
> `login(data.access_token, …)` → `setToken("")` → pollutes localStorage + starts
> session monitoring. Fork BOTH layers:

**1. `auth.service.ts:login`** — return early before ANY storage when `otp_required`:
```ts
async login(credentials) {
  const response = await apiClient.post<LoginResponse>(API_ENDPOINTS.auth.login, credentials);
  if (response.data?.otp_required) {
    sessionStorage.setItem('otp_pending', JSON.stringify({
      session_id: response.data.otp_session_id,
    }));
    return response;   // do NOT call authManager.setToken or write user fields
  }
  if (response.data?.access_token) {
    authManager.setToken(response.data.access_token);
    // ...existing localStorage user-field writes
  }
  return response;
}
```

**2. `useAuth.ts:useLogin.onSuccess`** — branch BEFORE calling `login(...)`:
```ts
onSuccess(data) {
  if (data.otp_required) {
    navigate('/login/otp');   // do NOT call authContext.login()
    return;
  }
  queryClient.clear();
  login(data.access_token, data.user);
  navigateByRole(data.user.role);
}
```

### Routes/screens
- `/login/otp` — code-entry screen. Reads `otp_session_id` from
  `sessionStorage['otp_pending']` (NOT router state — RT-M9 from the TOTP plan
  still applies: router `location.state` is lost on refresh; sessionStorage
  survives refresh, clears on tab close). 6-digit input, auto-submit, resend
  button with 30s countdown, "use a different login" link back to `/login`.
- No `/settings/security` enrollment UI (email-OTP has no enrollment).

### Token storage discipline
- `auth_token` localStorage = the real 14-day JWT only.
- `otp_session_id` in `sessionStorage['otp_pending']` — distinct key, cleared on
  verify-success or tab close.
- `AuthManager.isTokenValid()` (`lib/auth.ts:50`) must never see the OTP id.
  Unit-test that an empty/pending value isn't treated as valid.

## Related Code Files

- **Modify:** `frontend/src/types/api/auth.types.ts` — extend `LoginResponse` (`otp_required`, `otp_session_id`); add `VerifyOTPRequest`, `ResendOTPRequest`, `OTPStartedResponse`.
- **Modify:** `frontend/src/services/api/auth.service.ts` — RT-C3: `login` returns early before storage when `otp_required`; add `verifyLoginOtp`, `resendOtp`.
- **Modify:** `frontend/src/hooks/api/useAuth.ts` — RT-C3: `useLogin.onSuccess` branches before `login(...)`.
- **Modify:** `frontend/src/config/api.config.ts` — add endpoints: `verifyOtp: '/auth/login/verify'`, `resendOtp: '/auth/login/resend'`.
- **Create:** `frontend/src/pages/auth/OTPScreen.tsx` — the code-entry + resend UI.
- **Modify:** `frontend/src/App.tsx` (or router) — add the `/login/otp` route (public; requires `sessionStorage['otp_pending']`, else redirect to `/login`).

## Implementation Steps

1. **Types + endpoints + service methods** — freezes the API contract.
2. **RT-C3: fork `auth.service.ts:login` AND `useAuth.onSuccess`.** Two unit
   tests: (a) `otp_required` response does NOT call `authManager.setToken` or
   write `auth_token`; (b) `onSuccess` does not call `authContext.login()` on the OTP branch.
3. **OTPScreen** — 6 separate inputs (shadcn pattern) or a single masked input
   with auto-advance; auto-submit on 6 digits; reuse the frontend `RateLimiter`
   (`lib/auth.ts:197`) for client-side throttling so we don't burn server attempts.
   Resend button with a 30s countdown; on click calls `resendOtp` and restarts
   the countdown. Error states: wrong code, locked account, session expired.
4. **Routing + guard** — `/login/otp` is public but requires
   `sessionStorage['otp_pending']`; redirect to `/login` if absent.
5. **Vietnamese strings** — "Nhập mã xác thực", "Mã đã được gửi đến email của bạn",
   "Gửi lại mã (Ns)", "Mã không đúng", "Tài khoản tạm bị khóa, thử lại sau N phút".

## Success Criteria

- [ ] A non-OTP login (employee, or `OTP_ENABLE=false`) behaves exactly as today.
- [ ] An admin/partner login (OTP on) routes to `/login/otp`; no token stored.
- [ ] Entering the correct code completes login and navigates by role.
- [ ] Resend button is disabled for 30s; clicking it requests a new code.
- [ ] `auth_token` localStorage is empty during the OTP step (verified via devtools).
- [ ] `pnpm lint && pnpm type-check` pass.

## Risk Assessment

- **Risk:** storing `otp_session_id` in `auth_token` by mistake. **Mitigation:**
  distinct `sessionStorage` key + a unit test on `AuthManager`.
- **Risk:** losing the OTP id on page refresh → user must re-login. Acceptable
  (sessionStorage survives refresh; only a tab close loses it, which is the
  desired security behavior).
- **Risk:** mobile usability — the app is mobile-first (AGENTS.md). **Mitigation:**
  reuse existing mobile input patterns; large numeric keypad (`inputMode="numeric"`).
