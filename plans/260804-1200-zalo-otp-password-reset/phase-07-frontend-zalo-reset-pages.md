---
phase: 7
title: "Frontend: Zalo Reset Pages, Hooks & Routing"
status: pending
priority: P1
dependencies: ["3"]
effort: "M"
---

# Phase 7: Frontend: Zalo Reset Pages, Hooks & Routing

> **Scope note:** This phase consolidates the CLI-scaffolded Phase 8 (Hooks & Routing).
> Pages, hooks, service methods, routing, and the Login entry-point change are one PR.
> See `frontend/AGENTS.md` and the root `AGENTS.md` rule: **desktop + mobile (390px)
> parity is mandatory** for every affected role view.

## Overview

Give employees a self-service path from `/login` → `/forgot-password` (choose Zalo) →
enter mobile → receive ZNS OTP → `/zalo-reset-password` → enter code + new password →
back to `/login`. Reuse the existing `ForgotPassword` / `ResetPassword` / `OTPLogin`
component patterns (`ct-card`, `font-display`, anti-enumeration "always success" UX)
so the visual language is identical to the email flow.

**Employee-first default (Q4 in `plan.md`):** most employees have no email, so the
"Quên mật khẩu?" link on `/login` routes to `/forgot-password` with the **Zalo** tab
pre-selected. Email remains available as the secondary tab.

## Requirements

- **Functional**
  - `/forgot-password` gains a **segmented toggle**: `Đặt lại qua Zalo` | `Đặt lại qua Email`. Default = Zalo (remember last choice in `localStorage`).
  - Zalo tab: a single mobile input + submit. On submit, advance to `/zalo-reset-password?sid=...` (sid from response, **not** in `Referer`-leakable URL fragment — pass via router state, strip from history like the email reset strips its token).
  - `/zalo-reset-password`: mobile (read-only, from state), 6-digit OTP input (6-box OTP UI like `OTPLogin`), new password + confirm, submit. On success → `/login` with success toast.
  - Resend button with cooldown countdown (reuse the `OTPLogin` resend UX, if present).
- **Non-functional**
  - Desktop (≥1280px) **and** mobile (390px; spot-check 320px for the OTP box row + money/density rule) verified per `AGENTS.md`.
  - Anti-enumeration contract honored client-side: the "code sent" success state is shown regardless of network outcome (mirror `ForgotPassword.tsx`'s `try/catch/finally setDone(true)`).
  - Vietnamese copy throughout; matches the `constants/messages.go` strings sent by the backend.
  - `pnpm lint && pnpm type-check` green; existing `OTPLogin.test.tsx` pattern extended to cover the new page.

## Architecture

### File layout

```
frontend/src/
  pages/
    ForgotPassword.tsx          # MODIFY — add Zalo/Email toggle, default Zalo
    ZaloResetPassword.tsx       # NEW  — OTP + new-password form
  components/
    ZaloResetForm.tsx           # NEW  (optional split; or inline in page)
  hooks/api/
    useZaloReset.ts             # NEW  — request / confirm / resend mutations
  services/api/
    auth.service.ts             # MODIFY — add 3 methods
  config/
    api.config.ts               # MODIFY — add 3 endpoint keys
  App.tsx                       # MODIFY — register /zalo-reset-password route
  pages/Login.tsx               # MODIFY — "Quên mật khẩu?" deep-links to /forgot-password (already exists; just verify default tab)
```

### API config (`config/api.config.ts`)

```ts
zaloResetRequest:  '/auth/zalo-reset/request',
zaloResetConfirm:  '/auth/zalo-reset/confirm',
zaloResetResend:   '/auth/zalo-reset/resend',
```

### Service (`services/api/auth.service.ts`)

```ts
requestZaloReset(mobile: string) {
  return apiClient.post<ZaloResetRequestResp>(endpoints.zaloResetRequest, { mobile });
}
confirmZaloReset(payload: { otp_session_id: string; code: string; new_password: string }) {
  return apiClient.post(endpoints.zaloResetConfirm, payload);
}
resendZaloReset(otp_session_id: string) {
  return apiClient.post<{ otp_session_id: string }>(endpoints.zaloResetResend, { otp_session_id });
}
```

### Hooks (`hooks/api/useZaloReset.ts`)

```ts
export const useRequestZaloReset = () =>
  useMutation({
    mutationFn: (mobile: string) => authService.requestZaloReset(mobile),
    meta: { skipGlobalError: true }, // anti-enumeration UX handled in-page
  });

export const useConfirmZaloReset = () =>
  useMutation({
    mutationFn: authService.confirmZaloReset,
    meta: { skipGlobalError: true }, // inline "wrong code" / "expired" messages
  });

export const useResendZaloReset = () =>
  useMutation({ mutationFn: (sid: string) => authService.resendZaloReset(sid) });
```

### Routing (`App.tsx`)

```tsx
<Route path="/zalo-reset-password" element={<ZaloResetPassword />} />
```

`/forgot-password` already exists; only its internals change (toggle).

### Page UX — `ZaloResetPassword.tsx`

- Reads `otp_session_id` + `mobile` from `location.state` (NOT from query string — avoids history/Referer leak, same hardening as `ResetPassword.tsx` stripping the token). If state is missing → redirect to `/forgot-password`.
- OTP input: 6 separate digit boxes, auto-advance, paste-support, numeric keyboard on mobile (`inputMode="numeric"`).
- New password + confirm (reuse `ResetPassword.tsx`'s strength meter if present).
- Submit → `useConfirmZaloReset`; on 401 show inline "Mã không đúng hoặc đã hết hạn"; on 400 show validation; on 200 → `navigate('/login', { state: { resetSuccess: true } })`.
- Resend button disabled for 60s after each send (countdown in state).

### Anti-enumeration client contract

`/forgot-password` Zalo tab:

```tsx
const handleSubmit = async (e) => {
  e.preventDefault();
  const mobile = normalizeMobileInput(input); // digits only, user still sees what they typed
  try {
    const { data } = await requestZaloReset(mobile);
    navigate('/zalo-reset-password', {
      state: { otp_session_id: data.otp_session_id, mobile },
      replace: true, // strip from history
    });
  } catch {
    // Anti-enumeration: still advance to the next page with the dummy session_id
    // the backend returned, so the UX is identical for known/unknown mobiles.
    // The /confirm call will 401 for unknown mobiles — the user sees the same
    // "invalid code" message they'd see for a typo, with no existence leak.
    navigate('/zalo-reset-password', {
      state: { otp_session_id: '', mobile }, // backend returned a dummy sid; pass it through
      replace: true,
    });
  }
};
```

> This mirrors `ForgotPassword.tsx`'s "flip to success regardless of outcome" rule.

## Related Code Files

- **Create:**
  - `frontend/src/pages/ZaloResetPassword.tsx`
  - `frontend/src/hooks/api/useZaloReset.ts`
- **Modify:**
  - `frontend/src/pages/ForgotPassword.tsx` — add Zalo/Email toggle, default Zalo, route to `/zalo-reset-password`.
  - `frontend/src/services/api/auth.service.ts` — 3 methods.
  - `frontend/src/config/api.config.ts` — 3 endpoint keys.
  - `frontend/src/App.tsx` — register the new route.
  - `frontend/src/pages/Login.tsx` — verify the "Quên mật khẩu?" link (line 382) still routes correctly; no change expected.
- **Reference (read-only):**
  - `frontend/src/pages/ForgotPassword.tsx` — visual + anti-enumeration pattern to mirror.
  - `frontend/src/pages/ResetPassword.tsx` — token-stripping + password-strength pattern.
  - `frontend/src/pages/OTPLogin.tsx` + `OTPLogin.test.tsx` — 6-box OTP UI + test pattern.
  - `frontend/src/hooks/api/usePasswordReset.ts` — `skipGlobalError` meta pattern.

## Implementation Steps

1. **API config + service methods** — add the 3 endpoint keys + 3 service methods. Type the response (`ZaloResetRequestResp { otp_session_id: string }`).
2. **Hooks** — 3 mutations with `skipGlobalError` where appropriate.
3. **`ZaloResetPassword.tsx`** — page: read state, OTP boxes, password fields, submit + resend. Reuse `ct-card` / `font-display` classes.
4. **`ForgotPassword.tsx`** — add the segmented toggle (DaisyUI-style `join` or shadcn `Tabs` — match whatever the codebase already uses; check `OTPLogin` for the OTP-toggle precedent). Default Zalo, persist last choice. Route to `/zalo-reset-password` with router state.
5. **`App.tsx`** — register the route (lazy `ZaloResetPassword` to match the existing lazy pattern).
6. **Tests** — extend the `OTPLogin.test.tsx` pattern: render `ZaloResetPassword` with mock state, assert submit calls the confirm mutation with the right payload, assert 401 renders the inline error, assert resend cooldown.
7. **Visual QA (manual, per `AGENTS.md`)** — 1280px + 390px + 320px for both the `/forgot-password` toggle and `/zalo-reset-password` forms. Verify 44px touch targets, no horizontal scroll on the OTP row at 320px, readable Vietnamese wrapping.

## Success Criteria

- [ ] `/forgot-password` shows a Zalo/Email toggle, defaults to Zalo, remembers the choice.
- [ ] Submitting the Zalo tab advances to `/zalo-reset-password` with the session id in router state (not URL); history is replaced.
- [ ] `/zalo-reset-password` 6-box OTP input works on desktop and mobile (390px, 320px), with `inputMode="numeric"` and auto-advance.
- [ ] Wrong-code 401 renders an inline Vietnamese error; success navigates to `/login` with a success toast.
- [ ] Resend button shows a 60s countdown and re-enables.
- [ ] Anti-enumeration: a network error on `/request` still advances the user to `/zalo-reset-password` (no "mobile not found" leak).
- [ ] `pnpm lint && pnpm type-check` green; new test(s) pass.
- [ ] Desktop + 390px + 320px manually verified per `AGENTS.md` for the affected pages.

## Risk Assessment

- **R-Z13 (session_id in URL history):** Using a query string for `sid` would leak it via `Referer` and browser history (the email reset explicitly strips its token for this reason). **Mitigation:** use `navigate(..., { state, replace: true })`; the page reads `location.state`. If a user refreshes `/zalo-reset-password` they lose the state → redirect to `/forgot-password` with a toast ("Phiên đã hết hạn"). This is the same trade-off the email reset makes.
- **R-Z14 (OTP input UX at 320px):** Six boxes + labels can overflow on the narrowest devices. **Mitigation:** size each box to `min(2.75rem, 11vw)`, gap `0.5rem`, allow the row to wrap if needed. Verify at 320px during Step 7.
- **R-Z15 (no email on file for the employee):** If a user picks the **Email** tab but has no email, the backend's email reset silently no-ops (anti-enumeration). Not new; documented here so the page's help link ("Không nhận được email?") points the user at the Zalo tab.
