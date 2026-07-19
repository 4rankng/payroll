---
phase: 1
title: Foundation
status: completed
priority: P2
dependencies: []
---

# Phase 1: Foundation

## Overview

Establish the partner scope hook by mirroring AdminLayout's **two-part** pattern: (1) a `html.partner-route-active` classList effect on `document.documentElement` for base/global rules, AND (2) a `data-partner-ui` attribute on the layout's wrapper div for the actual portal-reach selector `[data-partner-ui] [data-radix-popper-content-wrapper]` (admin's exact pattern at `admin-daisy.css:35`). Create `partner.css` with selectors scoped under BOTH layers. No visual changes yet — this phase only establishes the scope that Phase 3's rules attach to.

## Requirements

- **Functional:**
  - `PartnerLayoutInner` adds `html.partner-route-active` class to `document.documentElement` on mount and removes it on unmount.
  - The wrapper div around `<Outlet />` (or the outer layout div) gets a `data-partner-ui=""` attribute, matching admin's `data-admin-ui` pattern at `AdminLayout.tsx:109`.
  - The classList effect is **guarded by the partner role check**, not `[]` deps — see H1 mitigation below.
  - `src/styles/partner.css` exists with selectors scoped under `html.partner-route-active` (for base/global) and `[data-partner-ui]` (for portal-reaching presentation rules).
  - `src/index.css` imports `partner.css` exactly once.
- **Non-functional:**
  - The classList effect is idempotent under React StrictMode double-invocation.
  - The effect's cleanup removes the class on every route-away path — including the ~100ms window between auth failure and ProtectedRoute's `setTimeout` redirect (`ProtectedRoute.tsx:27`).
  - Zero CSS selectors in `partner.css` can match when both `html.partner-route-active` is absent AND no `[data-partner-ui]` node exists.

## Architecture

Mirror admin's **two-part** pattern exactly — this is the v2 Critical fix:

```tsx
// PartnerLayout.tsx — PartnerLayoutInner
const { user } = useAuth();  // already imported per PartnerSidebar usage
const isPartner = user?.role === 'partner';

useEffect(() => {
  if (!isPartner) return;  // guard — mirror AdminLayout.tsx:100
  document.documentElement.classList.add("partner-route-active");
  return () => document.documentElement.classList.remove("partner-route-active");
}, [isPartner]);

// JSX wrapper around Outlet:
<div data-partner-ui={isPartner ? "" : undefined}>
  {/* existing layout content */}
</div>
```

```css
/* partner.css — TWO selector layers, matching admin-daisy.css exactly */

/* Layer 1: base/global rules under the route class */
html.partner-route-active,
html.partner-route-active body {
  /* token overrides, base typography, selection color, autofill rules */
}

/* Layer 2: presentation rules that must reach Radix portals.
   This is the EXACT selector admin uses at admin-daisy.css:35 —
   [data-admin-ui] [data-radix-popper-content-wrapper]. */
[data-partner-ui] [data-radix-popper-content-wrapper] {
  /* z-index fixes, etc. — whatever partner needs */
}

/* Layer 2 also covers in-scope presentation rules */
[data-partner-ui] {
  /* typography, tabular-nums, etc. */
}

/* Reduced motion, scoped */
@media (prefers-reduced-motion: reduce) {
  [data-partner-ui] *,
  [data-partner-ui] *::before,
  [data-partner-ui] *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

**Why both layers (v2 Critical fix C1):** admin's actual portal-reach selector at `admin-daisy.css:35` is `[data-admin-ui] [data-radix-popper-content-wrapper]`, NOT `html.admin-route-active ...`. The `html.admin-route-active` class only handles base/global rules. v2's original draft copied only the classList half — meaning Radix portals (Dialog/Sheet/DropdownMenu rendered to `document.body` outside the wrapper div) would NOT be reachable. This fix restores the full pattern.

Do NOT relocate `--partner-accent` (it stays at `variables.css:212` alongside `--admin-accent` and `--manager-accent`). Add new `--partner-*` tokens under the `html.partner-route-active` block only if Phase 2 audit identifies a real need.

## Related Code Files

- **Create:** `frontend/src/styles/partner.css`
- **Modify:**
  - `frontend/src/layouts/PartnerLayout.tsx` — add the classList `useEffect`. Import `useEffect` from React.
  - `frontend/src/index.css` — append `@import './styles/partner.css';` (after `admin-daisy.css` to preserve cascade order).
- **Read-only references:**
  - `frontend/src/layouts/AdminLayout.tsx:101-102` (classList effect pattern).
  - `frontend/src/styles/admin-daisy.css:393-409` (admin's `html.admin-route-active` reference rules).
  - `frontend/src/styles/variables.css:212` (`--partner-accent` location — do not move).

## Implementation Steps

1. **Read `AdminLayout.tsx:97-115` in full** to confirm BOTH halves of the pattern: the classList effect at lines 99-103 AND the wrapper `data-admin-ui` attribute at line 109. Match both.
2. **Read `admin-daisy.css:1-50`** to confirm the two-layer selector strategy: `[data-admin-ui]` rules (incl. `[data-admin-ui] [data-radix-popper-content-wrapper]` at line 35) for presentation + portals, and `html.admin-route-active` rules for base/global.
3. **Add the classList `useEffect` to `PartnerLayoutInner`** with the role guard:
   ```tsx
   const { user } = useAuth();
   const isPartner = user?.role === 'partner';
   useEffect(() => {
     if (!isPartner) return;
     document.documentElement.classList.add("partner-route-active");
     return () => document.documentElement.classList.remove("partner-route-active");
   }, [isPartner]);
   ```
   The role guard is H1 mitigation — without it, ProtectedRoute's ~100ms `setTimeout` redirect on auth failure (`ProtectedRoute.tsx:27`) would leak the class to `/login` or another surface.
4. **Add the `data-partner-ui` wrapper attribute** on the layout's outer div (matching `AdminLayout.tsx:109`):
   ```tsx
   <div
     data-partner-ui={isPartner ? "" : undefined}
     className="relative flex h-dvh w-full group/layout"
   >
     {/* existing PartnerLayoutInner content */}
   </div>
   ```
5. **Create `partner.css`** with the two-layer structure from the Architecture section above. Phase 1 ships only the skeleton — actual presentation rules are added in Phase 3.
6. **Import `partner.css` in `src/index.css`** after `@import './styles/admin-daisy.css';`.
7. **Smoke check the classList half:** Boot `pnpm dev`, navigate to `/partner/dashboard`. `document.documentElement.classList.contains('partner-route-active')` returns `true`. Navigate to `/admin/dashboard`: `false`. Navigate to `/login`: `false`. Navigate to `/partner/projects`: `true`.
8. **Smoke check the wrapper-attribute half:** On `/partner/dashboard`, `document.querySelector('[data-partner-ui]')` returns the layout div. On `/admin/dashboard`, returns null.
9. **Portal-reach verification:** On `/partner/dashboard`, open a Radix portal that renders to `document.body` (e.g., the partner sidebar's UserAvatarDropdown menu, or any partner Sheet). Confirm the portal's `[data-radix-popper-content-wrapper]` node is a DOM sibling of `[data-partner-ui]`, not a child — then verify the `[data-partner-ui] [data-radix-popper-content-wrapper]` selector WOULD reach it (this is the CSS descendant combinator — works across siblings only if one is an ancestor of the other; if the portal renders to `document.body` outside the wrapper, this selector does NOT match — confirm by testing the actual rule with a temporary `z-index: 9999` rule and observing whether the portal's z-index changes).
   - **If the selector does NOT reach the portal** (likely — Radix portals render to `document.body`): the correct selector is `html.partner-route-active [data-radix-popper-content-wrapper]`, NOT `[data-partner-ui] [data-radix-popper-content-wrapper]`. Re-audit `admin-daisy.css:35` and confirm whether admin's `[data-admin-ui] [data-radix-popper-content-wrapper]` actually reaches admin portals today, or whether admin portals are simply unscoped and rely on default z-index. **This is an open verification item — do not assume.**
10. **StrictMode double-mount check:** React 18 StrictMode mounts → unmounts → remounts. Confirm class is present after remount.
11. **Auth-failure leak check (H1 verification):** Force an auth failure on a partner route (e.g., temporarily revoke role in dev). Within the 100ms before ProtectedRoute redirects, check `document.documentElement.classList`. With the `[isPartner]` guard, class should never stamp when role is wrong.
12. **Build sanity:** `cd frontend && pnpm lint && tsc --noEmit && pnpm build` all pass.

## Success Criteria

- [ ] `html.partner-route-active` class is present on `documentElement` ONLY when a partner role user is on a `/partner/*` route; absent on `/admin/*`, `/employee/*`, `/login`, AND during ProtectedRoute's auth-failure redirect window.
- [ ] `[data-partner-ui]` attribute is present on the PartnerLayout wrapper div when `isPartner`, absent otherwise.
- [ ] **Portal-reach selector verified:** Step 9's open verification item resolved — confirm whether the correct selector is `[data-partner-ui] [data-radix-popper-content-wrapper]` (matches admin's literal pattern) OR `html.partner-route-active [data-radix-popper-content-wrapper]` (works for portals rendered to `document.body`). Document the finding in `partner.css` as a comment.
- [ ] `partner.css` exists, is imported exactly once via `src/index.css`, and contains zero selectors that can match outside the partner scope.
- [ ] The classList effect survives React StrictMode double-invocation in dev.
- [ ] `--partner-accent` is untouched in `variables.css:212`.
- [ ] No visual changes to any route (Phase 1 is foundation-only).
- [ ] `pnpm lint`, `tsc --noEmit`, `pnpm build` pass.

## Risk Assessment

- **Risk:** Effect cleanup fails on certain route transitions (e.g., partner → partner via modal navigation), leaving the class stuck on `<html>`.
  **Mitigation:** Step 5 verifies class state on every partner route transition. If a stuck-class bug appears, mirror admin's exact cleanup shape — admin already handles this correctly.
- **Risk:** Cascade-order choice (placing partner.css after admin-daisy.css) creates unexpected overrides.
  **Mitigation:** Phase 1 has no real rules; the placeholder cannot override anything. Phase 3's actual rules will use `html.partner-route-active` scoping, which is absent on admin routes, so no overlap is possible.
- **Risk:** Future editor adds a `:root` rule in `partner.css` that leaks globally.
  **Mitigation:** File header comment explicitly forbids `:root` selectors; Phase 4 grep-verifies `grep -E '^[^/]*:root' frontend/src/styles/partner.css` returns empty.

## Phase 1 Verification Log (Session 2026-07-19)

**Implementation:**
- `frontend/src/layouts/PartnerLayout.tsx` — added `useEffect` adding `html.partner-route-active` class with `[isPartner]` guard + early return; added `data-partner-ui` attribute on wrapper div (matching `AdminLayout.tsx:108-112`); threaded `isPartner` from outer `PartnerLayout` via `useAuth()` into `PartnerLayoutInner` prop.
- `frontend/src/styles/partner.css` — created with dual-scope skeleton: Layer 1 `[data-partner-ui]` rules (matches admin-daisy.css:1-48), Layer 2 `html.partner-route-active` rules (matches admin-daisy.css:393-409 for portal reach), reduced-motion block.
- `frontend/src/index.css` — appended `@import './styles/partner.css';` after `admin-daisy.css`.

**Step 9 open item RESOLVED:** Inspection of `admin-daisy.css:35` vs `:393-409` revealed admin uses BOTH selectors for different purposes:
- `[data-admin-ui] [data-radix-popper-content-wrapper]` (line 35) — inline popovers inside the wrapper
- `html.admin-route-active [role="dialog"]` etc. (lines 399-409) — portals rendered to `document.body`

`partner.css` implements the same dual pattern. Documented in the file's header comment.

**Gates passed:**
- `pnpm lint` — 0 errors (3 pre-existing coverage warnings only)
- `tsc -p tsconfig.json --noEmit` (invoked via `pnpm lint`) — exit 0
- `pnpm build` — exit 0 (built in 7.87s)

**Code review (code-reviewer subagent): APPROVED.** All 8 acceptance criteria verified. Confirmed:
- `html.partner-route-active` and `html.admin-route-active` are mutually exclusive (one role per user, both effects have cleanup).
- `PartnerLayout` default export signature unchanged; `isPartner` prop is internal.
- Effect shape mirrors AdminLayout:99-103 byte-identically modulo role name.
- `--partner-accent` untouched in `variables.css:212`.
- `partner.css` strict awk-scan confirms zero out-of-scope selectors.

**Non-blocking observations carried forward:**
- **EC-1 → Phase 3 TODO:** Add in-scope focus-visible rule (`[data-partner-ui] :is(button, a, ...):focus-visible`) to match admin-daisy.css:392 parity. Phase 1 ships only the portal-half.
- **EC-2 → Fixed:** Comment line drift `210-213` → `211-213` corrected in `partner.css:19`.
- **EC-3 (pre-existing, out of scope):** Double-nested `ProtectedRoute` (App.tsx:276 + PartnerLayout.tsx:56). Pre-existing pattern also present in AdminLayout. Not a Phase 1 concern.
- **EC-4 → Phase 4 verification note:** `main.tsx` has no `<StrictMode>` wrapper, so the StrictMode double-invoke smoke test cannot run at runtime. The effect is idempotent so the criterion is satisfied in principle; Phase 4 will document this.
- **Small improvement:** `partner.css:90` adds the portal-reach clause to the reduced-motion block (`html.partner-route-active :is([role="dialog"],...) *`) which admin-daisy.css:39-48 does NOT cover — admin's reduced-motion misses body-portaled dialogs. Worth noting for Phase 4 cross-surface audit.
