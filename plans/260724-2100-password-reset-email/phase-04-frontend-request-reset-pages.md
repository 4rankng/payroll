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

The existing `change-password` UI has a strength meter component (search `frontend/src` for `PasswordStrength` / `GetPasswordStrength`). Reuse the same component/hook rather than rebuilding. The backend already exposes `GET /auth/password-strength` — the change-password page calls it; mirror that.

## Related Code Files

- **Create:** `frontend/src/pages/ForgotPassword.tsx`
- **Create:** `frontend/src/pages/ResetPassword.tsx`
- **Modify:** `frontend/src/pages/Login.tsx` — add "Quên mật khẩu?" link below the password field, linking to `/forgot-password`.
- **Read (for reference):** `frontend/src/pages/Login.tsx`, `frontend/src/pages/OTPLogin.tsx`, and the change-password component for the strength-meter pattern.

## Implementation Steps

1. **Read the reference files first:** `Login.tsx` (already read during planning), `OTPLogin.tsx` (partially read), and grep for the password-strength component used by the change-password flow (`grep -rl "password-strength\|PasswordStrength\|getPasswordStrength" frontend/src`). Copy the shell layout (hero section + card) from `Login.tsx` so the visual identity matches.

2. **Write `ForgotPassword.tsx`:**
   - State: `email`, `submitting`, `done` (boolean, flips true after submit).
   - On submit: call the `useRequestPasswordReset()` hook (Phase 5). On success OR error, flip `done = true` — the success message is shown either way (anti-enumeration contract must hold client-side too, so a network error doesn't reveal whether the email is valid).
   - Layout: centered card, logo, heading "Quên mật khẩu?", subtext "Nhập email đăng ký, chúng tôi sẽ gửi liên kết đặt lại mật khẩu.", the email input (same `ct-input` styling), submit button, and a "Quay lại đăng nhập" link.
   - When `done`, replace the form with the success alert + back-to-login link.

3. **Write `ResetPassword.tsx`:**
   - Read token on mount: `const token = new URLSearchParams(window.location.search).get("token")`.
   - If no token → render the "invalid link" state immediately.
   - State: `newPassword`, `confirmPassword`, `showPassword`, `strength` (from backend hook), `submitting`, `success`, `error`.
   - Form: new password input (with show/hide toggle + strength meter), confirm password input (with match validation), submit button "Đặt lại mật khẩu".
   - Client validation: passwords match, min length 8 (matches the DTO binding). Show inline VN errors.
   - On submit: call `useConfirmPasswordReset()` hook with `{ token, new_password }`.
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

- **Strength meter differs from change-password page:** Mitigation: locate and reuse the exact same component/hook rather than reimplementing. If it's not reusable, build a minimal meter but keep the visual consistent.
- **Anti-enumeration leak via timing:** The `done` flag flips regardless of success/error, but if a 404 (should never happen — backend always 200s) had different timing than a 200, a sophisticated attacker could distinguish. Mitigation: backend always returns 200 (Phase 2), so the client only ever sees 200 or network-error.
- **Token in URL is sensitive:** It's a single-use token with 30-min TTL; being in the URL is the standard magic-link pattern. The reset page clears it from the URL on success (`navigate('/login', { replace: true })`).
