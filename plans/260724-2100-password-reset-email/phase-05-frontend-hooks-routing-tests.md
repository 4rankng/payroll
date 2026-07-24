---
phase: 5
title: "Frontend: Hooks & Routing & Tests"
status: pending
priority: P2
dependencies: [4]
---

# Phase 5: Frontend: Hooks & Routing & Tests

## Overview

Wire the two new pages into the app: add the API client methods, TypeScript types, TanStack Query hooks, and the two public routes in `App.tsx`. Add component tests covering the happy and error paths.

## Requirements

- **Functional:** `authService.requestPasswordReset(email)` and `authService.confirmPasswordReset({token, new_password})` call the right endpoints; `useRequestPasswordReset` and `useConfirmPasswordReset` mutations wrap them with proper `onSuccess`/`onError` handling (including the anti-enumeration contract).
- **Non-functional:** Routes are public (no `ProtectedRoute` wrapper), lazy-loaded like the other auth pages. Types live in `types/api/auth.types.ts` alongside the existing auth types.

## Architecture

### API client methods (`auth.service.ts`)

```ts
async requestPasswordReset(email: string): Promise<ApiResponse<void>> {
  return apiClient.post<void>(API_ENDPOINTS.auth.passwordResetRequest, { email });
}

async confirmPasswordReset(payload: { token: string; new_password: string }): Promise<ApiResponse<void>> {
  return apiClient.post<void>(API_ENDPOINTS.auth.passwordResetConfirm, payload);
}
```

### Endpoint config

Add to `frontend/src/config/api.config.ts` (find the `auth` block — it already has `login`, `loginVerify`, `loginResend`, `changePassword`):

```ts
auth: {
  // ...existing...
  passwordResetRequest: '/auth/password-reset/request',
  passwordResetConfirm: '/auth/password-reset/confirm',
},
```

### Types (`types/api/auth.types.ts`)

```ts
export interface PasswordResetRequest {
  email: string;
}
export interface PasswordResetConfirm {
  token: string;
  new_password: string;
}
```

### Hooks (`hooks/api/usePasswordReset.ts`)

```ts
export const useRequestPasswordReset = () =>
  useMutation({
    mutationFn: (email: string) => authService.requestPasswordReset(email),
    // No global error toast — the page handles the anti-enumeration UX itself.
    meta: { skipGlobalError: true },
  });

export const useConfirmPasswordReset = () =>
  useMutation({
    mutationFn: (payload: PasswordResetConfirm) => authService.confirmPasswordReset(payload),
    meta: { skipGlobalError: true }, // page shows inline errors
  });
```

**Why `skipGlobalError: true`:** The global error handler would show a toast for any non-2xx, which (a) for the request endpoint should never happen (always 200), and (b) for the confirm endpoint the page wants to show a tailored expired-link message, not a generic toast.

### Routes (`App.tsx`)

Add to the public routes block, near `<Route path="/login" element={<Login />} />`:

```tsx
const ForgotPassword = lazy(() => import("./pages/ForgotPassword"));
const ResetPassword = lazy(() => import("./pages/ResetPassword"));

// inside <Routes>:
<Route path="/forgot-password" element={<ForgotPassword />} />
<Route path="/reset-password" element={<ResetPassword />} />
```

These sit **outside** any `ProtectedRoute` — they are pre-auth screens.

## Related Code Files

- **Create:** `frontend/src/hooks/api/usePasswordReset.ts`
- **Modify:** `frontend/src/services/api/auth.service.ts` — add the two methods
- **Modify:** `frontend/src/config/api.config.ts` — add the two endpoint constants
- **Modify:** `frontend/src/types/api/auth.types.ts` — add the two interfaces
- **Modify:** `frontend/src/App.tsx` — add lazy imports + routes
- **Create:** `frontend/src/pages/__tests__/ForgotPassword.test.tsx`
- **Create:** `frontend/src/pages/__tests__/ResetPassword.test.tsx`

## Implementation Steps

1. **Add the endpoint constants** in `api.config.ts` (read the existing `auth` block first to match the exact nesting/structure).

2. **Add the types** in `auth.types.ts`.

3. **Add the service methods** in `auth.service.ts` (the `AuthService` class).

4. **Write `usePasswordReset.ts`** with the two hooks above.

5. **Update Phase 4's pages** to import and use these hooks (if not already wired during Phase 4 — confirm the page components reference `useRequestPasswordReset` / `useConfirmPasswordReset`).

6. **Add the routes** in `App.tsx`.

7. **Write component tests.** First check the existing test setup: read `frontend/src/pages/OTPLogin.test.tsx` (already referenced in the repo) to mirror the testing-library + vitest + msw pattern.
   - `ForgotPassword.test.tsx`:
     - Renders the form.
     - Submitting a valid email calls `authService.requestPasswordReset` (mock with msw or `vi.spyOn`) and shows the success state.
     - Submitting an unknown email shows the **same** success state (anti-enumeration).
     - Invalid email shows inline validation.
   - `ResetPassword.test.tsx`:
     - Renders the form when `?token=abc` is present.
     - Shows the invalid-link state when no token.
     - Submitting with mismatched passwords shows an inline error.
     - A successful confirm call shows the success state.
     - An expired-token error shows the "link expired" state with a retry link.

8. **Run frontend checks:**
   - `cd frontend && pnpm lint`
   - `cd frontend && pnpm type-check`
   - `cd frontend && pnpm test` (unit/component tests)

## Success Criteria

- [ ] `useRequestPasswordReset` and `useConfirmPasswordReset` exist and call the correct endpoints.
- [ ] `/forgot-password` and `/reset-password` routes render their pages without auth.
- [ ] Navigating to `/login` → "Quên mật khẩu?" → `/forgot-password` works.
- [ ] Component tests pass for: valid email, unknown email (same UI), missing token, mismatched passwords, success, expired token.
- [ ] `pnpm lint`, `pnpm type-check`, and `pnpm test` are all green.

## Risk Assessment

- **`skipGlobalError` meta not honored:** If the global error boundary doesn't check `meta.skipGlobalError`, a confirm-failure would show a duplicate toast. Mitigation: the OTP hooks already use `meta: { skipGlobalError: true }` (seen in `useAuth.ts`), so the pattern is established and honored.
- **Lazy import path typo:** `lazy(() => import("./pages/ForgotPassword"))` must match the file's default export. A typo → blank page on navigation. Mitigation: `pnpm type-check` catches default-export mismatches; manual nav test in step 6.
- **Test setup differs from OTPLogin.test.tsx:** Read that file first and mirror its imports (msw handlers, render-with-router helper, etc.) to avoid reinventing the harness.
