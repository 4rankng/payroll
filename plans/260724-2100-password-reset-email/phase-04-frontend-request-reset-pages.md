---
phase: 4
title: "Frontend: Request & Reset Pages"
status: pending
priority: P2
dependencies: [2]
---

# Phase 4: Frontend: Request & Reset Pages

## Overview

Two new pages, both matching the visual language of the existing `Login.tsx` and `OTPLogin.tsx`: DaisyUI (`data-admin-ui` + `data-theme="congtruong"`) components, mobile-first, all Vietnamese.

- **`/forgot-password`** (`ForgotPassword.tsx`): single email input → submit → show a success state (the same anti-enumeration message the backend returns). Includes a "back to login" link.
- **`/reset-password`** (`ResetPassword.tsx`): reads `?token=...` from the URL, shows a new-password + confirm-password form with a live strength meter (reuse the existing pattern), submits, and on success redirects to `/login` with a toast.

## Requirements

- **Functional:** Both pages call the new backend endpoints via TanStack Query mutations (Phase 5 adds the hooks; this phase builds the presentational UI). Forms validate client-side (email format; password min length + match). The reset page handles missing/expired tokens with a clear Vietnamese message and a link back to `/forgot-password`.
- **Non-functional:** Mobile-responsive (the `Login.tsx` hero-card pattern is the reference). Accessible labels (`htmlFor`, `aria-label`). No English strings. Reuse existing DaisyUI input/button/alert classes (`ct-input`, `ct-btn`, `ct-alert`).

## Architecture

### Page states

**`ForgotPassword.tsx`:**
```
[initial]      → email input + "Gửi liên kết đặt lại" button
[submitting]   → button shows spinner + "Đang gửi..."
[success]      → green alert: "Nếu email tồn tại..." + "Quay lại đăng nhập" link
                (shown regardless of API outcome — matches backend anti-enumeration)
```

**`ResetPassword.tsx`:**
```
[no token]     → red alert: "Liên kết không hợp lệ" + link to /forgot-password
[form]         → new password (with strength meter) + confirm password + submit
[submitting]   → spinner
[success]      → green alert: "Đặt lại thành công" → auto-redirect /login after 2s
[error]        → red alert with VN message (invalid/expired token → link to /forgot-password)
```

### Strength meter

**Red Team H7:** The existing `change-password` UI has a **client-side** strength indicator at `frontend/src/components/ui/password-strength-indicator.tsx:12` (used by `ChangePasswordModal.tsx:19` and `ResetPasswordModal.tsx:6`). It uses `validatePassword` from `@/lib/validation` — **no backend call**. There is NO `GET /auth/password-strength` endpoint (`routes_auth.go` has no such route — the earlier draft's reference was a phantom). Reuse this component directly; do not add a backend strength call.

## Related Code Files

- **Create:** `frontend/src/pages/ForgotPassword.tsx`
- **Create:** `frontend/src/pages/ResetPassword.tsx`
- **Modify:** `frontend/src/pages/Login.tsx` — add "Quên mật khẩu?" link below the password field, linking to `/forgot-password`.
- **Read (for reference):** `frontend/src/pages/Login.tsx`, `frontend/src/pages/OTPLogin.tsx`, and **`frontend/src/components/ui/password-strength-indicator.tsx`** (Red Team H7 — the client-side strength component to reuse; used by `ChangePasswordModal.tsx:19` and `ResetPasswordModal.tsx:6`).

## Implementation Steps

1. **Read the reference files first:** `Login.tsx` (already read during planning), `OTPLogin.tsx` (partially read), and grep for the password-strength component used by the change-password flow (`grep -rl "password-strength\|PasswordStrength\|getPasswordStrength" frontend/src`). Copy the shell layout (hero section + card) from `Login.tsx` so the visual identity matches.

2. **Write `ForgotPassword.tsx`:**
   - State: `email`, `submitting`, `done` (boolean, flips true after submit).
   - On submit: call the `useRequestPasswordReset()` hook (Phase 5). On success OR error, flip `done = true` — the success message is shown either way (anti-enumeration contract must hold client-side too, so a network error doesn't reveal whether the email is valid).
   - Layout: centered card, logo, heading "Quên mật khẩu?", subtext "Nhập email đăng ký, chúng tôi sẽ gửi liên kết đặt lại mật khẩu.", the email input (same `ct-input` styling), submit button, and a "Quay lại đăng nhập" link.
   - When `done`, replace the form with the success alert + back-to-login link.

3. **Write `ResetPassword.tsx`** with **Red Team H3 (token-URL hardening)** applied:
   - Read token on mount: `const token = new URLSearchParams(window.location.search).get("token")`.
   - **Strip the token from the URL immediately on mount** (keep it in component state) so it doesn't linger in browser history, leak via Referer on sub-resource loads, or persist in screenshots:
     ```tsx
     useEffect(() => {
       if (token) {
         window.history.replaceState({}, "", "/reset-password");  // clean URL
       }
     }, [token]);
     ```
   - **Add a `<meta name="referrer" content="no-referrer" />` tag** (or set it via a `useEffect` that manipulates `document.head`) for the duration of this page, so no Referer header leaks the token to any third-party resource. The backend's global `Referrer-Policy: strict-origin-when-cross-origin` (`security_headers.go:20`) is not sufficient — it strips path/query on cross-origin but not same-origin, and older browsers ignore it.
   - If no token → render the "invalid link" state immediately.
   - State: `newPassword`, `confirmPassword`, `showPassword`, `submitting`, `success`, `error` (strength is computed **client-side** via the `PasswordStrengthIndicator` component — Red Team H7, no backend call).
   - Form: new password input (with show/hide toggle + `PasswordStrengthIndicator`), confirm password input (with match validation), submit button "Đặt lại mật khẩu".
   - Client validation: passwords match, min length 8 (matches the DTO binding). Show inline VN errors.
   - On submit: call `useConfirmPasswordReset()` hook with `{ token, new_password }` (token from state, not from URL).
   - On success: show green alert "Đặt lại mật khẩu thành công", then `setTimeout(() => navigate('/login', { replace: true }), 2000)`.
   - On error: if message contains the token-invalid constant → show a "link expired" alert with a button to `/forgot-password`. Otherwise show the raw VN error message.

4. **Add "Quên mật khẩu?" link to `Login.tsx`:** Below the password field's container (around the `needsCaptcha` block), add a small right-aligned link:
   ```tsx
   <div className="flex justify-end">
     <Link to="/forgot-password" className="text-xs font-bold text-primary hover:underline">
       Quên mật khẩu?
     </Link>
   </div>
   ```
   Import `Link` from `react-router-dom` (already imported in the file via `react-router-dom` usage).

5. **Visual check:** Run `make dev`, navigate to `/login` → click "Quên mật khẩu?" → `/forgot-password`. Submit a test email → see the success state. (The actual email send is exercised in Phase 5 / integration; here just confirm the UI renders and transitions states.)

## Success Criteria

- [ ] `/forgot-password` renders, validates email format, submits, and shows the anti-enumeration success state for both known and unknown emails.
- [ ] `/reset-password?token=valid` renders the new-password form with a strength meter and confirm-match validation.
- [ ] `/reset-password` (no token) renders the invalid-link state with a link back to `/forgot-password`.
- [ ] A token-invalid API response renders the expired-link state with a retry link.
- [ ] Both pages are mobile-responsive (test at 375px width) and match the login page's DaisyUI theme.
- [ ] `pnpm lint` and `pnpm type-check` pass for the new files.

## Risk Assessment

- **Strength meter:** Red Team H7 — reuse the existing **client-side** `PasswordStrengthIndicator` (`components/ui/password-strength-indicator.tsx:12`); there is no backend strength endpoint. No rebuild needed.
- **Anti-enumeration leak via timing:** The `done` flag flips regardless of success/error. Mitigation: backend always returns 200 (Phase 2) + performs timing equalization (Red Team H2), so the client only ever sees 200 or network-error.
- **Red Team H3 (token-in-URL leakage):** The earlier draft dismissed this as "standard magic-link pattern." That was wrong — the token is live for 30 minutes before first use, so any leak (Referer header, proxy log, browser history, inbox screenshot) gives an attacker a takeover window. Mitigations applied: (a) strip token from URL on mount via `replaceState`, (b) `Referrer-Policy: no-referrer` meta on the page, (c) clear on success. **Residual risk** (proxy logs capturing the click before the page strips it, inbox malware) is documented and accepted given the single-use + 30-min TTL constraint — this matches industry magic-link implementations (e.g. GitHub, Notion).
