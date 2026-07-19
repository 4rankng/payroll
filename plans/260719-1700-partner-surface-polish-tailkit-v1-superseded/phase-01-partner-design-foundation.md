---
phase: 1
title: "Partner Design Foundation"
status: pending
priority: P2
dependencies: []
---

# Phase 1: Partner Design Foundation

## Overview

Establish the scoped design-token layer for the partner surface: a `[data-partner-ui]` root attribute, a new `partner.css` stylesheet, the `--partner-*` CSS variable set, and the Tailkit→shadcn class mapping table that every later phase references. Polish the partner shell (`PartnerLayout`, `PartnerSidebar`, partner-mode `MobileBottomNav`) on this foundation. No route content changes yet.

## Requirements

- **Functional:**
  - `PartnerLayoutInner` renders a `data-partner-ui` attribute on its root container.
  - `src/styles/partner.css` is created, imported once in the global stylesheet entry, and all rules are scoped under `[data-partner-ui]`.
  - Partner sidebar gets the same active-state, hover, focus-ring, collapsed, and keyboard treatment as the (already shipped) admin sidebar — using partner tokens, not `ct-*` daisyUI classes.
  - Partner-mode `MobileBottomNav` items meet 44px minimum tap targets and inherit partner accent tokens.
- **Non-functional:**
  - Zero CSS rules in `partner.css` may match outside `[data-partner-ui]`.
  - No new runtime dependency; no Tailwind 4 / daisyUI version changes.
  - `prefers-reduced-motion` respected for every animation introduced.

## Architecture

```
[PartnerLayout]
├── <div data-partner-ui>                      ← new scope root
│   ├── <PartnerSidebar>                       ← polished with partner tokens
│   ├── <SidebarToggle>
│   ├── <main>
│   │   └── <SectionErrorBoundary>             ← unchanged
│   │       └── <Outlet/>                       ← Phase 3 target
│   └── (existing radial-gradient bg moves into [data-partner-ui] rule)
├── <MobileBottomNav groups={PARTNER_NAV_GROUPS}/>   ← partner branch only
└── <NotificationFAB/>
```

Token mapping is documented as code comments at the top of `partner.css` so Phase 2/3 implementers can reference it without opening the plan.

## Related Code Files

- **Create:** `frontend/src/styles/partner.css`
- **Modify:**
  - `frontend/src/layouts/PartnerLayout.tsx` — add `data-partner-ui` attribute, import partner.css via global entry (not direct import in component).
  - `frontend/src/components/PartnerSidebar.tsx` — replace ad-hoc `bg-card/[0.08]` / `text-white/50` classes with `--partner-*` tokens; preserve `NavLink` contract and `useSidebar` collapsed state.
  - `frontend/src/components/MobileBottomNav.tsx` — partner-branch token swap only (do not touch admin/employee branches).
  - `frontend/src/styles/index.css` (or `main.tsx` stylesheet entry) — single `@import "./partner.css";`.
- **Read-only references:**
  - `frontend/src/styles/admin-daisy.css` (scoping pattern reference)
  - `frontend/tailwind.config.ts` lines 81–90 (`ct-` prefix, `themeRoot`)

## Implementation Steps

1. **Inventory partner tokens.** Read `PartnerSidebar.tsx`, `PartnerLayout.tsx`, `PartnerWorkforceOverviewCard.tsx`, and any `partner-*` component that hardcodes colors. Collect every literal color/spacing/radius used.
2. **Derive `--partner-*` token set.** Map inventories to a coherent token family mirroring the admin pattern: `--partner-bg`, `--partner-surface`, `--partner-surface-muted`, `--partner-border`, `--partner-foreground`, `--partner-muted-foreground`, `--partner-accent`, `--partner-accent-foreground`, `--partner-sidebar-bg`, `--partner-sidebar-active`, `--partner-sidebar-hover`, `--partner-radius`, `--partner-shadow`. Reuse existing `--success`, `--warning`, `--danger`, `--primary` HSL variables where the design system already exposes them.
3. **Publish the Tailkit→shadcn mapping table.** Add it as a comment block at the top of `partner.css`:
   ```text
   Tailkit (Tailwind v4)  →  Partner (project tokens)
   --------------------------------------------------
   bg-white                  →  bg-[hsl(var(--partner-surface))]
   bg-secondary-100/75       →  bg-[hsl(var(--partner-surface-muted))]
   text-secondary-900        →  text-[hsl(var(--partner-foreground))]
   text-secondary-500        →  text-[hsl(var(--partner-muted-foreground))]
   border-secondary-200/50   →  border-[hsl(var(--partner-border))/0.6]
   ring-secondary-200/50     →  ring-[hsl(var(--partner-border))/0.6]
   text-emerald-600          →  text-[hsl(var(--success))]
   bg-emerald-100            →  bg-[hsl(var(--success))/0.12]
   shadow-xs                 →  shadow-[0_1px_2px_rgba(0,0,0,0.04)]
   rounded-lg / rounded-xl   →  rounded-[hsl(var(--partner-radius))]  (or use existing radius tokens)
   dark:*                    →  omitted (partner ships light-only)
   ```
4. **Create `partner.css`** with `[data-partner-ui]` block, `--partner-*` `:root`-style declaration block (under the data attribute, not `:root`), base typography (tabular-nums on `.num` or globally for money), selection color, autofill protection (mirror `[data-login-ui]` pattern), and reduced-motion rules.
5. **Import `partner.css` once** in `main.tsx` (or `styles/index.css` if that's the existing aggregator). Confirm with `grep -rn "partner.css" frontend/src` that it appears exactly once.
6. **Add `data-partner-ui` attribute** to `PartnerLayoutInner`'s outer `<div className="relative flex h-dvh...">`. Remove the inline `bg-[radial-gradient(...)]` on `<main>` and move that style into a `[data-partner-ui] main` rule in `partner.css`.
7. **Refactor `PartnerSidebar.tsx` to follow the admin sidebar design (validated Session 1, Q4).** Admin sidebar (`AdminSidebar.tsx`, 529 LOC, 8 `ct-` occurrences) uses daisyUI `ct-menu` for its shell. Two implementation paths — pick the one that produces the closer visual match with least blast radius:
   - **Path A (preferred if visually equivalent):** Re-render the admin sidebar pattern using shadcn primitives (`ui/sidebar.tsx`, `ui/dropdown-menu.tsx`, lucide icons, `--partner-*` tokens). Keeps daisyUI confined to admin/employee.
   - **Path B:** Add a small partner daisyUI theme in `tailwind.config.ts` (a sibling to `congtruong`) and extend `themeRoot` to `:where([data-admin-ui], [data-employee-ui], [data-partner-ui])`. Higher blast radius — any daisyUI misconfig affects all three surfaces. Only choose if Path A cannot reproduce admin's sidebar visual fidelity.

   Either path must:
   - Replace partner's literal `bg-card/[0.08]`, `text-white/50`, `ring-white/[0.12]` with `--partner-*` tokens
   - Preserve `NavLink` `to=` paths and `end` flags exactly
   - Preserve `useSidebar().state` collapsed/expanded behavior
   - Preserve `UserProfileSheet` trigger, `useUnreadNotifications` badge, `useAuth().user` avatar
   - Preserve keyboard order and focus-visible rings
   - Visually match admin sidebar (active state, hover, focus-ring, collapsed, keyboard treatment)
8. **Update `MobileBottomNav` partner branch only.** Find the partner-specific conditional (likely keyed off `groups === PARTNER_NAV_GROUPS` or a `role` prop) and swap its tokens to `--partner-*`. Do not touch admin or employee branches.
9. **Visual smoke check.** Boot `pnpm dev`, navigate to `/partner/dashboard`. Confirm sidebar/shell look coherent and `/admin/dashboard` + `/employee` + `/login` are visually unchanged.
10. **Isolation check.** In a browser dev console on `/admin/dashboard`: `document.querySelector('[data-partner-ui]')` returns null. On `/partner/dashboard`: returns the layout div. Repeat for `/employee`.

## Success Criteria

- [ ] `[data-partner-ui]` attribute is present on `/partner/*` routes and absent on every other route.
- [ ] `partner.css` exists, is imported exactly once via `src/index.css`, and contains zero selectors that can match outside `[data-partner-ui]`.
- [ ] Tailkit→shadcn mapping table is present as a code comment in `partner.css` and referenced in Phase 2/3 task descriptions.
- [ ] `PartnerSidebar` visually matches admin sidebar design (active state, hover, focus-ring, collapsed, keyboard treatment) — validated by side-by-side screenshot in Phase 4.
- [ ] `PartnerSidebar` uses `--partner-*` tokens; no literal color classes remain in the file (unless Path B daisyUI approach was chosen, in which case `ct-` classes are permitted only on sidebar shell elements).
- [ ] `MobileBottomNav` partner branch uses partner tokens; admin and employee branches are byte-unchanged.
- [ ] `/partner/dashboard` desktop and mobile render without regressions; no horizontal scroll at 320/390/768/1440.
- [ ] `pnpm lint`, `tsc --noEmit` pass.
- [ ] DOM-token isolation check passes on `/admin/*`, `/employee/*`, `/login`.

<!-- Updated: Validation Session 1 - Q4 sidebar must follow admin design; success criterion for visual parity added -->

## Risk Assessment

- **Risk:** Editing `MobileBottomNav.tsx` accidentally affects admin/employee branches.
  **Mitigation:** Diff after edit; the file is shared but branches must stay isolated. Phase 4 will re-verify.
- **Risk:** Adding `[data-partner-ui]` to the wrong node causes token leakage or gaps.
  **Mitigation:** Put it on the outermost layout div; verify with the Step 10 console check before moving to Phase 2.
- **Risk:** Mapping table drifts from actual code decisions made in Phase 2/3.
  **Mitigation:** Whole-Plan Consistency Sweep at end of Phase 4 re-reads `partner.css` and reconciles.
- **Risk:** Path B (daisyUI themeRoot expansion) introduces a regression on admin/employee surfaces.
  **Mitigation:** Path A (shadcn re-render) is preferred. Path B is only used if Path A cannot achieve visual parity, and Phase 4's cross-surface isolation check explicitly covers daisyUI leakage.
