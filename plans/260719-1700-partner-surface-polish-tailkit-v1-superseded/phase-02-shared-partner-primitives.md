---
phase: 2
title: "Shared Partner Primitives"
status: pending
priority: P2
dependencies: [1]
---

# Phase 2: Shared Partner Primitives

## Overview

Build the reusable partner primitives kit under `frontend/src/components/partner-ui/`. These components translate selected Tailkit patterns into the project's token vocabulary (per the Phase 1 mapping table) and compose existing Radix/shadcn primitives. They are the only new folder created by this plan; Phase 3 consumes them across all five partner routes.

## Requirements

- **Functional:** deliver a barrel-exported kit covering the recurring partner UI needs:
  - `PartnerPageHeader` — title, optional subtitle, optional icon, optional actions slot.
  - `PartnerStatsCard` + `PartnerStatsGrid` — stat tiles with optional trend, optional icon, optional sparkline slot (recharts stays external).
  - `PartnerDataTable` — card-wrapped table with optional header (title + search + actions slots), passed-in columns/rows via TanStack Table or children.
  - `PartnerMobileListRow` — accessible list row used by the `partner-*/mobile/*` variants.
  - `PartnerFilterBar` — consistent slot for filter chips + active-filter summary.
  - `PartnerEmptyState`, `PartnerLoadingState`, `PartnerErrorState` — status trio aligned to `a-c-empty-states-03` reference.
  - `PartnerStatusBadge` — semantic status pill (success / warning / danger / info / neutral) that never relies on color alone.
- **Non-functional:**
  - Every primitive composes existing `components/ui/*` primitives; no direct Radix re-imports.
  - No primitive accepts a Tailkit literal class — all styling goes through project tokens.
  - All interactive primitives meet 44px tap targets on mobile and expose focus-visible rings.

## Architecture

```
components/partner-ui/
├── index.ts                      ← barrel
├── partner-page-header.tsx
├── partner-stats-card.tsx        ← PartnerStatsCard + PartnerStatsGrid
├── partner-data-table.tsx
├── partner-mobile-list-row.tsx
├── partner-filter-bar.tsx
├── partner-empty-state.tsx       ← re-exports Loading + Error variants
├── partner-loading-state.tsx
├── partner-error-state.tsx
└── partner-status-badge.tsx
```

Each primitive is `.tsx`, UI-only (per `frontend/AGENTS.md` "File Responsibility Rule" — no API calls, no business logic). Any data shaping stays in the consuming route or in `.ts` utils.

## Related Code Files

- **Create:** everything under `frontend/src/components/partner-ui/`.
- **Modify:** none (composition only). If a Phase-2 need reveals a missing `components/ui/*` capability, stop and surface it rather than editing the global primitive.
- **Reference (read-only):**
  - Phase 1 `partner.css` mapping table
  - **`src/components/admin-dashboard/PremiumStatStrip.tsx` (CANONICAL — `PartnerStatsCard` API cloned from this)**
  - `components/ui/card.tsx`, `components/ui/table.tsx`, `components/ui/badge.tsx`, `components/ui/skeleton.tsx`, `components/ui/button.tsx`
  - Existing one-off implementations to consolidate: `partner-employees/PartnerEmployeesStats.tsx`, `partner-projects/PartnerProjectsStats.tsx`, `partner-timesheet/PartnerTimesheetStats.tsx` (becomes the migration source for `PartnerStatsCard`).

## Implementation Steps

1. **Re-confirm the mapping table.** Open `partner.css` and copy the Tailkit→shadcn mapping comment into each primitive file header so future editors see the contract inline.
2. **`PartnerPageHeader`** — translate `a-c-page-headings-03` (With Actions). Slots: `title`, `subtitle?`, `icon?`, `actions?`, `breadcrumb?`. Use `<header>` semantic with `aria-labelledby`. Mobile collapses actions below the title.
3. **`PartnerStatsCard`** — translate `a-c-statistics-10` (Simple with Info). **API cloned from `src/components/admin-dashboard/PremiumStatStrip.tsx` (validated Session 1, Q2)** — that file is the canonical admin precedent, already shipping with `label`, `value: string|number`, `unit?`, `highlight?`, `onClick?` (button role + Enter/Space handler), `trend?: { value: string; positive: boolean }`, `vi-VN` number formatting, `font-financial` + tabular-nums, accessible `role="button"` + `tabIndex`. Partner extends with: `icon?: ReactNode`, `sparkline?: ReactNode` (recharts stays external), `valueFormat?: 'number' | 'currency'`, `loading?`. Keep `PremiumStatStrip`'s `group-hover:scale-105` hover affordance. Full proposed API ships — not speculative, proven by admin.
4. **`PartnerStatsGrid`** — responsive grid wrapper (`grid-cols-2 sm:grid-cols-3 lg:grid-cols-4`) that takes `children` or `items: PartnerStatsCardProps[]`.
5. **`PartnerDataTable`** — translate `a-c-tables-15` (In Card Alt with Search + Actions) at the *shell* level only. Props: `title?`, `description?`, `search?` (controlled input slot), `actions?` (ReactNode slot for buttons/selects), `table?` (TanStack Table instance) or `children`. Renders the existing `ui/table.tsx` primitives inside; column definitions stay in the consuming route. Wrap in `ui/card.tsx`. Honor `aria-busy` during loading.
6. **`PartnerMobileListRow`** — accessible row used by `partner-*/mobile/`. Slots: `avatar?`, `title`, `subtitle?`, `meta?` (right-aligned), `status?` (uses `PartnerStatusBadge`), `action?` (chevron / menu trigger). Min height 44px, role `listitem` inside `role="list"` parent.
7. **`PartnerFilterBar`** — sticky-top filter strip. Props: `children` (filter chips/selects), `activeCount?`, `onClearAll?`. Translates the per-page `*Filters.tsx` ad-hoc markup into one shape.
8. **`PartnerEmptyState`** — translate `a-c-empty-states-03`. Props: `icon?`, `title`, `description?`, `actions?` (ReactNode, typically buttons). `<section role="status">` semantics.
9. **`PartnerLoadingState`** — `skeleton`-based placeholder shaped like the consuming surface (rows × cols param, or `variant: 'grid' | 'list' | 'table' | 'card'`).
10. **`PartnerErrorState`** — props: `title`, `description?`, `onRetry?`. Uses `PartnerStatusBadge` variant `danger`. `<section role="alert">`.
11. **`PartnerStatusBadge`** — variants: `success | warning | danger | info | neutral`. Each variant ships **both** a color and an icon (lucide-react `Check`, `AlertTriangle`, `X`, `Info`, `Minus`) so color is never the sole signal. Accepts optional `children` label.
12. **Barrel `index.ts`** — re-export all primitives. Run `pnpm lint` to catch circular imports.
13. **Story-free visual smoke.** Mount each primitive in a temporary route (or use the existing `Index.tsx` dev page) — confirm tokens resolve, dark-mode classes are absent, reduced-motion respected. Remove the temporary mount before commit.

## Success Criteria

- [ ] `components/partner-ui/` exists with all primitives listed and a working `index.ts` barrel.
- [ ] No primitive file contains a literal Tailkit color class (`secondary-*`, `emerald-*`, `dark:`).
- [ ] Each primitive file references the Tailkit→shadcn mapping table in a header comment.
- [ ] **`PartnerStatsCard` API parity with `PremiumStatStrip`**: same `label`, `value`, `unit?`, `highlight?`, `onClick?`, `trend?` semantics, plus partner extensions (`icon?`, `sparkline?`, `valueFormat?`, `loading?`). Validated by side-by-side prop comparison in Phase 4.
- [ ] `PartnerStatusBadge` pairs every color variant with a non-color signal (icon + text).
- [ ] All primitives pass `pnpm lint` and `tsc --noEmit`.
- [ ] A temporary smoke mount demonstrates each primitive renders against `--partner-*` tokens under `[data-partner-ui]`.
- [ ] No edits to `components/ui/*` (verified by `git diff frontend/src/components/ui`).

<!-- Updated: Validation Session 1 - Q2 PartnerStatsCard API confirmed full; PremiumStatStrip declared canonical reference -->

## Risk Assessment

- **Risk:** Primitives accidentally introduce `dark:` variants that conflict with the project's HSL-variable dark-mode strategy.
  **Mitigation:** Mapping table explicitly omits `dark:*`; Phase 4 grep-verifies `grep -rn "dark:" frontend/src/components/partner-ui` is empty.
- **Risk:** `PartnerDataTable` abstraction leaks too much table logic and forces Phase 3 to rewrite TanStack Table usage.
  **Mitigation:** Keep the shell slim — column defs, sorting, and pagination stay in the route. Primitive only owns card + header + search/actions slots.
- **Risk:** Status badge variants diverge from existing admin/employee status semantics and confuse users across surfaces.
  **Mitigation:** Use the existing `--success`, `--warning`, `--danger`, `--info` HSL variables that already back admin/employee badges; only the *shape* changes, not the semantic color.
