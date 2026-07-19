---
phase: 3
title: "Partner Route Migration"
status: pending
priority: P2
dependencies: [1, 2]
---

# Phase 3: Partner Route Migration

## Overview

Apply the Phase 2 primitives across every reachable `/partner/*` route. Each route migrates desktop and mobile in the same step — no "mobile later" deferments. Behavior is preserved: URL state, query params, mutations, deep-linked sheets, and Vietnamese copy all stay intact.

## Requirements

- **Functional:** all five partner routes and their mobile variants consume the `partner-ui/` primitives:
  - `/partner/dashboard`
  - `/partner/projects` (+ mobile list)
  - `/partner/employees` (+ mobile list)
  - `/partner/timesheet` (+ mobile list)
  - `/partner/timesheet/payment-history`
- **Non-functional:**
  - No route changes any hook, mutation, or service contract.
  - URL query state (`?month=&project=&status=&search=&page=&sort=`) is preserved.
  - Vietnamese copy is preserved verbatim.
  - Deep-linked sheets (`returnTo`, modal IDs in `modal-registry-auto.ts`) keep working.

## Architecture

Each route follows the same migration template:

1. Replace the per-page `*Header.tsx` with `<PartnerPageHeader>`.
2. Replace the per-page `*Stats.tsx` with `<PartnerStatsGrid>` + `<PartnerStatsCard>` (move any recharts sparkline into the card's sparkline slot).
3. Replace the per-page `*Filters.tsx` body with `<PartnerFilterBar>` wrapping existing chip/select controls.
4. Replace the table wrapper markup with `<PartnerDataTable>`. Move the column definitions and TanStack Table instance unchanged.
5. Replace mobile list rows with `<PartnerMobileListRow>` inside a `role="list"` parent.
6. Replace ad-hoc empty/loading/error markup with the `partner-ui/` status trio.
7. Replace inline status pills with `<PartnerStatusBadge>`.

## Related Code Files

- **Modify (route pages):**
  - `frontend/src/pages/partner/DashboardPage/index.tsx`
  - `frontend/src/pages/partner/ProjectsPage/index.tsx`
  - `frontend/src/pages/partner/EmployeesPage/index.tsx`
  - `frontend/src/pages/partner/TimesheetsPage/index.tsx`
  - `frontend/src/pages/partner/PaymentHistoryPage/index.tsx`
- **Modify (component groups — consolidate into primitives, then delete the old file if it becomes a thin wrapper):**
  - `components/partner-dashboard/PartnerWorkforceOverviewCard.tsx` — keep the recharts donut; wrap in `PartnerStatsCard`-style shell.
  - `components/partner-dashboard/PartnerProjectCard.tsx`, `PartnerProjectsList.tsx`, `PartnerEmployeeListSheet.tsx` — adopt primitives; preserve sheet behavior.
  - `components/partner-employees/PartnerEmployees{Header,Filters,Stats,SummaryStats,List,Table}.tsx`
  - `components/partner-employees/mobile/PartnerEmployeesListMobile.tsx`
  - `components/partner-projects/PartnerProjects{Header,Filters,Stats,List}.tsx`
  - `components/partner-projects/mobile/PartnerProjectsListMobile.tsx`
  - `components/partner-timesheet/PartnerTimesheet{Header,Filters,Stats,List,Table}.tsx`
  - `components/partner-timesheet/mobile/PartnerTimesheetListMobile.tsx`
- **Delete:** any per-page file that becomes a no-op wrapper around a `partner-ui/` primitive after migration, per Q3 (Validation Session 1) and `frontend/AGENTS.md` DON'T #6. Before deleting, grep for external importers; re-point imports or keep as a thin re-export only when a real external consumer exists. Every deletion is recorded in the Phase 4 verification report with file path + reason.

## Implementation Steps

1. **Migrate `/partner/dashboard`.**
   - Header → `PartnerPageHeader`.
   - Workforce card → keep recharts donut, restyle shell.
   - Project list → `PartnerStatsGrid` summary above existing `PartnerProjectsList`.
   - Mobile: confirm the dashboard's mobile branch (if any) consumes the same primitives.
2. **Migrate `/partner/projects`.**
   - Header, filters, stats, list, and mobile list.
   - Preserve any existing deep-link to project detail (modal or nested route).
3. **Migrate `/partner/employees`.**
   - Header, filters, stats (both `PartnerEmployeesStats` and `PartnerEmployeesSummaryStats` → consolidate into `PartnerStatsGrid`).
   - Table → `PartnerDataTable`. Preserve TanStack Table column defs and row click → detail sheet.
   - Mobile list → `PartnerMobileListRow`.
4. **Migrate `/partner/timesheet`.**
   - Header, filters, stats, table, mobile list.
   - Preserve month picker URL state and any approval action surfaces.
5. **Migrate `/partner/timesheet/payment-history`.**
   - The newest of the partner pages — confirm it currently has the thinnest styling. Bring it to parity with the other four routes.
6. **Cross-route sweep.**
   - Confirm every route uses `PartnerPageHeader`, `PartnerFilterBar`, and the status trio.
   - Confirm no route re-introduces a literal Tailkit color class.
   - Confirm every status pill is now `PartnerStatusBadge` (color + icon).
7. **Behavioral spot-checks (per route).**
   - URL query state survives a page reload.
   - Deep-linked sheet (e.g., employee detail via `?modal=...&returnTo=...`) opens and returns correctly.
   - Mobile ↔ desktop navigation preserves scroll position where the current behavior does.
   - Mutation feedback (toast / sonner) still fires after create/update/delete.
8. **Reduced-motion + a11y pass.**
   - Toggle OS reduced-motion; confirm no jarring animation.
   - Keyboard-tab through each route; confirm visible focus rings and logical order.
   - Run a screen-reader spot check (VoiceOver) on one representative route.
9. **Delete replaced per-page components (validated Session 1, Q3).**
   - After each route migration, identify per-page files that have become no-op wrappers around `partner-ui/` primitives (e.g., `PartnerEmployeesStats.tsx`, `PartnerEmployeesSummaryStats.tsx`, `PartnerEmployeesHeader.tsx`, `*Filters.tsx` if fully replaced).
   - Delete them per `frontend/AGENTS.md` DON'T #6 ("DON'T maintain multiple versions of the same component").
   - Before deleting, grep for external importers (`grep -rln "<OldComponentName>" src`); if any test, story, or sibling component still imports the old name, either re-point the import or keep the file as a thin re-export and note it as an exception.
   - Record every deletion (file path + reason) for the Phase 4 verification report.

## Success Criteria

- [ ] All five partner routes consume `PartnerPageHeader`, `PartnerStatsGrid`/`PartnerStatsCard` (where stats exist), `PartnerFilterBar` (where filters exist), and the `partner-ui/` status trio.
- [ ] Every table on partner routes is wrapped in `PartnerDataTable`; column definitions and TanStack Table instances are unchanged.
- [ ] Every mobile list on partner routes uses `PartnerMobileListRow`.
- [ ] No `partner-*/*.tsx` file contains a literal Tailkit color class (`bg-white`, `text-emerald-600`, `secondary-*`, `dark:`).
- [ ] **Replaced per-page components deleted** (validated Session 1, Q3) — files that became no-op wrappers after migration are removed; deletion log captured for Phase 4 report.
- [ ] URL query state, deep-linked sheets, mutations, and Vietnamese copy are preserved on every route.
- [ ] No horizontal scroll at 320 / 390 / 768 / 1440 widths on any partner route.
- [ ] `pnpm lint`, `tsc --noEmit`, `pnpm build` pass.
- [ ] Focused Vitest specs (any existing `*.test.tsx` under `partner-*` or `pages/partner`) pass.

<!-- Updated: Validation Session 1 - Q3 deletion of replaced components made explicit success criterion -->

## Risk Assessment

- **Risk:** TanStack Table column defs drift during the `PartnerDataTable` migration.
  **Mitigation:** The primitive only owns the *shell*; columns stay in the route. Verify by diffing each route's column-def file before/after.
- **Risk:** Deep-linked sheets break when the trigger element is re-styled.
  **Mitigation:** Preserve the `<ModalLink>` / `useModalNavigation` trigger element; only wrap it visually. Spot-check in Step 7.
- **Risk:** Mobile and desktop variants drift in behavior because they're migrated separately.
  **Mitigation:** Step template requires desktop + mobile in the same route step. Phase 4 will run both viewports per route.
- **Risk:** Payment History page is underbuilt and migration reveals missing states.
  **Mitigation:** If the route lacks empty/error/loading variants, this phase adds them using the `partner-ui/` trio — that is in scope, not a scope expansion.
