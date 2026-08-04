---
phase: 5
title: "Frontend: Admin Zalo Settings Tab"
status: pending
priority: P1
dependencies: ["4"]
effort: "M"
---

# Phase 5: Frontend: Admin Zalo Settings Tab

## Overview

Add a **"Zalo ZNS" tab** to the existing `/admin/settings` page where an admin can: enter
the OA keys, run the OAuth connect flow, see live connection status, toggle the feature
on/off, and force a token refresh. This tab is the single control surface for the Zalo
connection — no env edits, no redeploy.

Matches the existing `SettingsPage` tab architecture (`TAB_GENERAL`, `TAB_FEE_CONFIG`,
`TAB_EMAIL`, `TAB_NOTIFICATIONS`) — adds `TAB_ZALO`. Reuses the `SettingCard` component
pattern and the `useSettingsForm` hook conventions. **Desktop + mobile (390px) parity
mandatory** per `AGENTS.md`.

## Requirements

- **Functional**
  - A "Zalo ZNS" tab at `/admin/settings?tab=zalo` (and the mobile equivalent).
  - **Connection status card:** shows `connected`/`expired`/`not configured` badge, `template_id`, `expires_at` (countdown), and `last_error` (if any). Masked — never shows tokens.
  - **Credentials form:** `App ID`, `Secret Key` (password input), `Template ID` (default `617976`). Save button → `PUT /admin/zalo/credentials`. After saving keys, the OAuth button becomes enabled.
  - **"Kết nối Zalo" button:** → `POST /admin/zalo/oauth/start` → `window.location = data.redirect_url`. On return to `?zalo_connected=1`, show a success toast + refetch status.
  - **Enable/disable toggle:** switch → `PUT /admin/zalo/enabled`. Disabled when not `connected` (can't enable a broken connection). Confirmation dialog on disable ("Employees won't be able to reset via Zalo until re-enabled").
  - **"Làm mới token" button:** → `POST /admin/zalo/refresh`. Disabled if not connected.
  - **Copy callback URL:** a read-only field showing the OAuth callback URL with a copy button, + a hint "Đăng ký URL này trong Zalo OA Console".
- **Non-functional**
  - Secret Key field is `type="password"`, never echoed back from the server (server returns masked/empty; the field is a write-only input).
  - All mutations show loading state + optimistic status update; errors render inline.
  - Desktop (≥1280px) and mobile (390px; 320px spot-check) verified.
  - Vietnamese copy throughout.

## Architecture

### File layout

```
frontend/src/
  pages/admin/SettingsPage/index.tsx              # MODIFY — add TAB_ZALO + tab trigger
  pages/mobile/admin/SettingsPage/index.tsx        # MODIFY — add TAB_ZALO (mobile parity)
  components/settings/
    ZaloConnectionSection.tsx                      # NEW — the whole tab content (desktop)
    ZaloConnectionSectionMobile.tsx                # NEW — mobile layout (or one responsive component)
  hooks/api/
    useZaloConnection.ts                           # NEW — status query + 5 mutations
  services/api/
    admin.service.ts (or settings.service.ts)      # MODIFY — add 5 admin methods
  config/api.config.ts                             # MODIFY — add 5 endpoint keys
```

### API config + service

```ts
// api.config.ts
adminZaloStatus:     '/admin/zalo',
adminZaloCreds:      '/admin/zalo/credentials',
adminZaloOAuthStart: '/admin/zalo/oauth/start',
adminZaloOAuthCb:    '/admin/zalo/oauth/callback', // browser-navigated, not called directly
adminZaloEnabled:    '/admin/zalo/enabled',
adminZaloRefresh:    '/admin/zalo/refresh',
```

### Hooks (`hooks/api/useZaloConnection.ts`)

```ts
export const useZaloStatus = () => useQuery({ queryKey: ['zalo','status'], queryFn: admin.getZaloStatus });
export const useSaveZaloCredentials = () => useMutation({ mutationFn: admin.saveZaloCredentials, onSuccess: invalidate(['zalo','status']) });
export const useStartZaloOAuth = () => useMutation({ mutationFn: admin.startZaloOAuth }); // returns redirect_url
export const useSetZaloEnabled = () => useMutation({ mutationFn: admin.setZaloEnabled, onSuccess: invalidate(['zalo','status']) });
export const useRefreshZaloToken = () => useMutation({ mutationFn: admin.refreshZaloToken, onSuccess: invalidate(['zalo','status']) });
```

### Component UX — `ZaloConnectionSection.tsx`

```
┌─ Zalo ZNS (OTP đặt lại mật khẩu) ──────────────────────────────┐
│                                                                │
│  Trạng thái: [● Đã kết nối]   Hết hạn: còn 23h 12m            │
│  Template ID: 617976 (OTP-ZNS-v1)                              │
│  Lỗi gần nhất: (không)                                         │
│                                                                │
│  ── Thông tin kết nối ──                                       │
│  App ID        [______________]                                │
│  Secret Key    [•••••••••••••]   (write-only)                 │
│  Template ID   [617976]                                        │
│                                  [ Lưu thông tin ]             │
│                                                                │
│  Callback URL  https://api.../admin/zalo/oauth/callback  [📋]  │
│  (Đăng ký URL này trong Zalo OA Console)                       │
│                                                                │
│  [ Kết nối Zalo ]   [ Làm mới token ]                          │
│                                                                │
│  ── Bật tính năng ──                                           │
│  Cho phép nhân viên đặt lại mật khẩu qua Zalo ZNS              │
│  [○────── Bật]    (disabled when !connected)                   │
└────────────────────────────────────────────────────────────────┘
```

- The status badge color: green (`connected` + `enabled`), yellow (`connected` + `disabled`), red (`error`), gray (`not configured`).
- "Kết nối Zalo" button: disabled until `configured=true` (app_id + secret saved). On click → mutate → `window.location.href = data.redirect_url`.
- On mount + on `?zalo_connected=1` query param → refetch status; show success toast; strip the param from the URL (so a refresh doesn't re-toast).
- Enable toggle: disabled + tooltip "Kết nối Zalo trước khi bật" when `!connected`. On disable, show a confirm dialog.

### SettingsPage integration (`pages/admin/SettingsPage/index.tsx`)

```tsx
const TAB_ZALO = 'zalo';
const VALID_TABS = new Set([TAB_GENERAL, TAB_FEE_CONFIG, TAB_EMAIL, TAB_NOTIFICATIONS, TAB_ZALO]);
// In the Tabs:
<TabsTrigger value={TAB_ZALO}><MessageCircle className="h-4 w-4" /> Zalo ZNS</TabsTrigger>
// In the content:
<TabsContent value={TAB_ZALO}><ZaloConnectionSection /></TabsContent>
```

Repeat for the mobile SettingsPage (parity).

## Related Code Files

- **Create:**
  - `frontend/src/components/settings/ZaloConnectionSection.tsx` (+ mobile variant or one responsive component)
  - `frontend/src/hooks/api/useZaloConnection.ts`
- **Modify:**
  - `frontend/src/pages/admin/SettingsPage/index.tsx` — add `TAB_ZALO` + trigger + content.
  - `frontend/src/pages/mobile/admin/SettingsPage/index.tsx` — same, for mobile parity.
  - `frontend/src/services/api/admin.service.ts` (or settings service) — 5 methods.
  - `frontend/src/config/api.config.ts` — 5 endpoint keys.
- **Reference (read-only):**
  - `frontend/src/pages/admin/SettingsPage/index.tsx` — existing tab pattern (`SettingCard`, `useSettingsForm`).
  - `frontend/src/components/settings/SettingCard.tsx` — card styling convention.
  - `frontend/src/components/email/AdminEmailComposer.tsx` — example of a complex settings tab.

## Implementation Steps

1. **API config + service methods** — add the 5 endpoint keys + admin service methods. Type the status response (`ZaloStatusDTO`).
2. **Hooks** — `useZaloStatus` (query) + 4 mutations; invalidate `['zalo','status']` on success.
3. **`ZaloConnectionSection.tsx`** — status card + credentials form + OAuth button + toggle + refresh + callback-URL copy. Wire to hooks.
4. **SettingsPage (desktop + mobile)** — add the tab trigger + content; import the section. Verify the mobile layout at 390px.
5. **`?zalo_connected=1` handling** — on mount, detect the param, refetch, toast, strip it.
6. **Tests** — render `ZaloConnectionSection` with a mocked status (connected / not configured / error); assert the badge, the disabled-states of the OAuth + toggle buttons, and that the credentials form calls the right mutation. Follow the `settings-page-parity.test.tsx` pattern.
7. **Visual QA (manual)** — 1280px + 390px + 320px; verify the status badge, form, and toggle render correctly in all connection states; verify 44px touch targets on mobile.

## Success Criteria

- [ ] `/admin/settings?tab=zalo` renders the Zalo ZNS section on desktop and mobile (390px), with no horizontal overflow at 320px.
- [ ] The status badge accurately reflects `connected` / `expired` / `not configured` / `error` states.
- [ ] Saving credentials calls `PUT /admin/zalo/credentials`; the OAuth button enables only after `configured=true`.
- [ ] "Kết nối Zalo" navigates to the Zalo permission URL; on return with `?zalo_connected=1`, the status refetches and a success toast shows (once — param stripped).
- [ ] The enable toggle is disabled with a tooltip when not connected; disabling shows a confirmation dialog.
- [ ] Secret Key is `type="password"` and is **never** populated from the server (write-only).
- [ ] The OAuth callback URL field is read-only with a working copy-to-clipboard button.
- [ ] `pnpm lint && pnpm type-check` green; new test(s) pass.

## Risk Assessment

- **R-Z22 (mobile tab overflow):** Adding a 5th tab to the existing `TabsList` may overflow at 390px. **Mitigation:** check the existing `TabsList` responsiveness; if needed, switch to a horizontal-scroll tab bar or a dropdown on mobile (match whatever the existing tabs already do at 390px — don't invent a new pattern).
- **R-Z23 (OAuth popup vs redirect):** Using `window.location` (full redirect) loses the admin's in-page context but is simplest and works on mobile. A popup (`window.open`) preserves context but is blocked by some mobile browsers. **Default:** full redirect (the `?zalo_connected=1` param restores context on return). Revisit if admins complain.
- **R-Z24 (stale status after server-side token refresh):** The Provider refreshes tokens internally during a `/auth/zalo-reset/request` (Phase 2/3), so `expires_at` can change without an admin action. The status badge may show stale "expired" until refetch. **Mitigation:** `useZaloStatus` uses TanStack Query with a 60s `refetchInterval` when the tab is visible (`refetchIntervalInBackground: false`); a manual "Làm mới" re-checks immediately.
