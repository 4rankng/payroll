---
title: "Partner Surface Polish via Tailkit Reference"
description: >-
  Polish the /partner/* route tree with a scoped design system that mirrors the
  shipping admin and employee systems, using the MCP Tailkit catalog as the
  visual reference library. Covers Dashboard, Projects, Employees, Timesheets,
  and Payment History. No backend, route, modal contract, or business-rule
  changes.
status: pending
priority: P2
branch: main
tags:
  - frontend
  - partner
  - ui
  - tailkit
  - shadcn
  - responsive
  - accessibility
blockedBy: []
blocks: []
created: '2026-07-19T13:30:38.978Z'
createdBy: 'ck:plan'
source: skill
---

# Partner Surface Polish via Tailkit Reference

## Overview

Apply one production-grade partner visual system across every reachable `/partner/*` route. The direction closes the documented gap left by the completed Admin daisyUI Redesign (`260718-2242-admin-daisyui-redesign`, complete) and Employee Mobile Polish (`260719-1310-employee-mobile-polish`, completed) plans — both of which explicitly excluded the partner surface from scope.

The **Tailkit MCP** catalog is the component and pattern reference, not a drop-in dependency. Tailkit ships Tailwind v4 idioms (`secondary-*`, `emerald-*`, `dark:` variants) that do not match this project's shadcn token system (`bg-card`, `hsl(var(--primary))`, `data-admin-ui` / `data-employee-ui` scoped roots). Each adopted pattern must be **translated** into the project's token vocabulary, not copy-pasted.

Implementation strategy mirrors the admin approach: scope a new `[data-partner-ui]` root, add partner-only CSS tokens (`--partner-*`), and compose existing Radix/shadcn primitives with presentation wrappers — never rewrite hooks, services, mutations, or route state.

## Scope

### In

- Every reachable `/partner` page, its desktop + mobile pair, and the routed mobile-only subpage variants under:
  - `/partner/dashboard` — `pages/partner/DashboardPage/`
  - `/partner/projects` — `pages/partner/ProjectsPage/` + `components/partner-projects/` (+ `mobile/`)
  - `/partner/employees` — `pages/partner/EmployeesPage/` + `components/partner-employees/` (+ `mobile/`)
  - `/partner/timesheet` — `pages/partner/TimesheetsPage/` + `components/partner-timesheet/` (+ `mobile/`)
  - `/partner/timesheet/payment-history` — `pages/partner/PaymentHistoryPage/`
- Partner shell: `PartnerLayout.tsx`, `PartnerSidebar.tsx`, partner-mode `MobileBottomNav`.
- Partner-scoped theme tokens, page headers, summary/stat cards, filters, tables + mobile list variants, pagination, tabs, status treatments, loading/empty/error states.
- One new shared primitives folder: `components/partner-ui/` (partner-scoped, not global `ui/`).

### Out

- Backend/API/schema changes, business-rule changes, new product features, route renames.
- Modal contract changes (registry IDs, deep-link `returnTo`, focus trap behavior).
- Redesign of admin or employee experiences — those are non-regression boundaries.
- Tailwind 4 migration, daisyUI version bump, or adding Tailkit as an npm dependency.
- Replacing Radix primitives — Radix stays authoritative for focus, dismissal, selects, dialogs, sheets, dropdowns.
- New charts library or replacing `recharts`.
- Dark mode for partner (admin/employee systems also ship light-only).

### Preserve

- `ProtectedRoute requiredRole="partner"` guard, deep-linked modal/`returnTo` behavior.
- URL query params (month, project, status, search, page), sorting, filtering, pagination semantics.
- All `useQuery` / `useMutation` hooks and cache invalidation rules in `src/hooks/api/`.
- Vietnamese copy and number/date formats (`vi-VN`, tabular numerals for money).
- Mobile-first responsive behavior and the `useIsMobile` hook contract.
- `SectionErrorBoundary sectionName="trang đối tác"` wrap in `PartnerLayout`.

## Key Decisions

1. **shadcn-first hybrid (validated Session 1).** shadcn primitives (`components/ui/*`) are the workhorse — partner composes the same primitives admin already ships. Tailkit MCP is the **visual reference catalog only** (read patterns via `mcp__tailkit__get_component_code`, never install it; every adopted snippet is rewritten against project tokens). daisyUI is permitted **only for the partner sidebar**, mirroring admin's `ct-menu` pattern — partner does **not** join daisyUI's `themeRoot` to avoid 3-way blast radius.
2. **Scoped token root.** Add `[data-partner-ui]` attribute on `PartnerLayoutInner`'s root div. New CSS lives in `src/styles/partner.css`, imported once via `src/index.css`. No global selectors leak outside `[data-partner-ui]`.
3. **Reuse the `PremiumStatStrip` precedent (validated Session 1).** Admin's `src/components/admin-dashboard/PremiumStatStrip.tsx:1-113` already implements the exact stat API partner needs (`label`, `value`, `unit?`, `highlight?`, `onClick?`, `trend?: { value, positive }`, `vi-VN` formatting, `font-financial`, tabular-nums, accessible button role). Phase 2's `PartnerStatsCard` clones this API and extends with `icon?` and `sparkline?` slots — not speculative, proven.
4. **Primitives folder isolation.** New shared components live under `components/partner-ui/` with barrel `index.ts`. They may *compose* `components/ui/*` but must not *edit* them — global primitive edits risk admin/employee regressions.
5. **Translate, don't transplant.** Tailkit color classes map: `bg-white` → `bg-card`; `text-secondary-500` → `text-muted-foreground`; `text-emerald-600` → `hsl(var(--success))` or status tokens; `ring-secondary-200/50` → `ring-border/60`. Document the mapping table in Phase 1 and reference it in every phase.
6. **Keep `recharts`.** `PartnerWorkforceOverviewCard` and any dashboard charts stay on recharts; Tailkit chart visuals are an aesthetic reference only.
7. **Behavior-preserving wrappers.** Use composition (`<PartnerStatsCard>` wrapping `<Card>` + `<dl>`) instead of editing data hooks or mutation handlers.
8. **Delete replaced components (validated Session 1).** Phase 3 explicitly deletes per-page partner components that become no-op wrappers after migration, per `frontend/AGENTS.md` DON'T #6 ("DON'T maintain multiple versions of the same component"). Deletions are tracked in the Phase 4 verification report.
9. **Partner sidebar follows admin sidebar design (validated Session 1).** Phase 1 expands to include rendering the partner sidebar using the admin sidebar pattern. Because admin sidebar uses daisyUI `ct-menu`, partner either gets a small daisyUI partner theme OR re-renders the admin pattern using shadcn primitives. Decision between these two approaches is made in Phase 1 based on which produces the closer visual match with least risk.

## Phases

| Phase | Name | Status | Purpose |
|-------|------|--------|---------|
| 1 | [Partner Design Foundation](./phase-01-partner-design-foundation.md) | Pending | Tokens, `[data-partner-ui]` root, `partner.css`, Tailkit→shadcn mapping table, sidebar + shell polish |
| 2 | [Shared Partner Primitives](./phase-02-shared-partner-primitives.md) | Pending | Build the `components/partner-ui/` kit: page header, stats card, filter bar, data table wrapper, mobile list row, empty/error/loading states |
| 3 | [Partner Route Migration](./phase-03-partner-route-migration.md) | Pending | Apply Phase-2 primitives across Dashboard, Projects, Employees, Timesheets, Payment History — desktop + mobile |
| 4 | [Visual QA and Verification](./phase-04-visual-qa-and-verification.md) | Pending | Lint, type-check, build, focused tests, route-by-route visual QA, cross-surface isolation check, `graphify update .` |

## Dependencies

- **No blocking cross-plan dependency.** Admin (`260718-2242-admin-daisyui-redesign`) and Employee (`260719-1310-employee-mobile-polish`) are both complete; they are non-regression boundaries, not blockers.
- **Internal:** Phase 2 depends on Phase 1. Phase 3 depends on Phases 1–2. Phase 4 depends on all implementation phases.
- **Wallet bulk-transfer plan** (`260718-2130-wallet-bulk-transfer-pipeline`) touches `/admin/wallet` only; no overlap with `/partner/*`.

## Tailkit Reference Catalog (Initial Selection)

These identifiers are starting points — implementers should `mcp__tailkit__browse_catalog` and `mcp__tailkit__search_components` for additional variants during their phase.

| Partner surface gap | Tailkit reference | Why |
|---|---|---|
| Stat / KPI cards | `a-c-statistics-10` (Simple with Info), `a-c-statistics-09` (Bordered with Heading + Chart) | Replaces ad-hoc `PartnerEmployeesStats` watermark pattern with consistent grid |
| Data tables | `a-c-tables-15` (In Card Alt with Search + Actions), `a-c-tables-12` (Column Sorting) | Partner tables currently have inconsistent card wrappers and search placement |
| Empty states | `a-c-empty-states-03` (With Actions), `a-c-empty-states-05` (With Placeholders) | Partner currently rolls its own empty states per page |
| Page headings | `a-c-page-headings-03` (With Actions), `a-c-page-headings-02` (With Icon) | Standardize the per-page `*Header.tsx` trio (Employees, Projects, Timesheet) |
| Layout / sidebar | `a-l-light-sidebar-06` (With Mini Sidebar) | Reference only — partner already uses `ui/sidebar.tsx` shell |

## Acceptance Criteria

- Every declared `/partner/*` route, mobile subpage, sheet, and dialog inherits the new partner design system without broken navigation, missing actions, or focus-trap regressions.
- Desktop (1440px), tablet (768px), and mobile (390px, 320px) layouts have no horizontal overflow, clipped controls, nested-scroll hazards, or touch targets below 44px on mobile.
- Stat cards, tables, page headers, filters, empty/loading/error states are visually consistent across all five partner pages and match the adopted Tailkit reference within token-translation tolerance.
- Partner styles are scoped under `[data-partner-ui]` and **do not** visually alter admin, employee, login, or partner-unrelated routes. Verified by DOM-token diff on representative routes.
- Financial values use `tabular-nums` (`font-feature-settings: "tnum"`); status never relies on color alone; keyboard, 200% zoom, reduced-motion, and screen-reader flows remain usable.
- Existing URL query state (month, project, status, search, page, sort) and deep-linked sheets preserve their current behavior across the redesign.
- `pnpm lint`, `tsc --noEmit`, `pnpm build`, focused Vitest specs, and representative partner Playwright E2E pass.
- `make api-test` still green (sanity — frontend changes should not affect backend, but the rule from `AGENTS.md` is non-negotiable).
- `graphify update .` completes after implementation.

## Risk Controls

- **Token translation drift.** Mitigation: Phase 1 publishes the Tailkit→shadcn mapping table; every phase references it. Whole-Plan Consistency Sweep verifies it at the end.
- **Cross-surface leakage.** Mitigation: `[data-partner-ui]` scoping; Phase 4 includes a DOM-token isolation check on `/admin/*`, `/employee/*`, and `/login`.
- **Mobile regression.** Mitigation: every Phase 3 route change ships desktop + mobile in the same step; no "mobile later" deferments.
- **Global primitive edits.** Mitigation: `components/partner-ui/*` is the only allowed new folder; `components/ui/*` is read-only for this plan.
- **Radix behavior drift.** Mitigation: primitives compose existing Radix wrappers; no new dialog/sheet/select implementations.

## Open Questions

None after Validation Session 1. All four decision points resolved (see Validation Log below).

## Validation Log

### Session 1 — 2026-07-19
**Trigger:** User-requested critical-questions interview before implementation.
**Questions asked:** 4

#### Verification Results (Standard tier — Fact Checker + Contract Verifier)

- **Claims checked:** 18
- **Verified:** 17 | **Failed:** 0 | **Unverified:** 1
- **Tier:** Standard (4-phase plan)

Notable verified claims:
- All 20 cited file paths exist (`PartnerLayout.tsx`, `PartnerSidebar.tsx`, all 5 partner page `index.tsx`, all `partner-*` component files, `admin-daisy.css`, `tailwind.config.ts`).
- `tailwind.config.ts:81-90` confirmed: `prefix: "ct-"`, `themeRoot: ":where([data-admin-ui], [data-employee-ui])"`, `daisyui: 4.12.24`, `tailwindcss: 3.4.17`.
- `PartnerLayout.tsx:1-46` structure matches plan: `ProtectedRoute requiredRole="partner"` → `SidebarProvider` → `PartnerLayoutInner` (with `SidebarToggle`, `<main>` + `SectionErrorBoundary sectionName="trang đối tác"` → `Outlet`) + `MobileBottomNav groups={PARTNER_NAV_GROUPS}` + `NotificationFAB`.
- No collision: `src/styles/partner.css` and `src/components/partner-ui/` do not exist yet.
- `index.css:1-5` is the aggregator (`@import './styles/{variables,base,utilities,premium,admin-daisy}.css'`) — `partner.css` should be added there.

Unverified findings that shaped the interview:
- `data-employee-ui` is applied per-component (`EmployeeMobileShell.tsx:108`, `EmployeePortalHeader.tsx:57`), NOT at a layout level. `data-admin-ui` is applied at `AdminLayout.tsx:109` (layout level). This asymmetry drove Q1 below.

Bonus discovery during verification (decisive for Q1 and Q2):
- `src/components/admin-dashboard/PremiumStatStrip.tsx` (113 LOC) **already implements** the exact `PartnerStatsCard` API proposed in Phase 2: `label`, `value: string|number`, `unit?`, `highlight?`, `onClick?`, `trend?: { value, positive }`, `vi-VN` formatting, `font-financial` + tabular-nums, accessible button role. Admin uses shadcn `Skeleton` for loading, NOT daisyUI `ct-stats`.

#### Questions & Answers

1. **[Architecture]** Which scoping strategy should partner adopt? (Plan proposed pure semantic-CSS layer matching employee approach; verification showed admin uses layout-level `data-admin-ui` + daisyUI, employee uses per-component `data-employee-ui` + semantic CSS.)
   - Options: Pure semantic CSS layer | Join daisyUI themeRoot | Hybrid
   - **Answer:** Research first — user asked "which provides the best UI experience: tailkit mcp, daisyui or shadcn mcp"
   - **Custom input (verbatim):** "you need to see which provide best UI experience tailkit mcp, daisyui or shadcn"
   - **Research conclusion (presented back to user, accepted):** **shadcn-first hybrid.** No shadcn MCP and no daisyUI MCP are configured in this repo — Tailkit is the only component-library MCP. Admin already ships `PremiumStatStrip` as a shadcn component with the exact API partner needs, using daisyUI only for the sidebar (`ct-menu`). Recommended: shadcn primitives as the workhorse, Tailkit as visual reference, daisyUI strictly for the sidebar (following admin's pattern). Confirms the plan's original "pure semantic CSS" recommendation — strengthened by the `PremiumStatStrip` discovery.
   - **Rationale:** Maximizes reuse of proven admin patterns; zero blast radius to admin/employee; keeps partner on the employee-style scoping model (no themeRoot expansion).

2. **[Scope]** Should the Phase 2 `PartnerStatsCard` primitive ship with `trend` and `sparkline` optional props? Current partner stats show plain values + watermark icons, no trends.
   - Options: Trim to what exists | Keep full proposed API | Minimal now + audit
   - **Answer:** **Keep full proposed API**
   - **Rationale:** Verification confirmed `PremiumStatStrip` already ships `trend: { value, positive }` — the API surface is not speculative, it's proven. Keeping the full API lets Phase 3 route migration adopt trend indicators without re-editing the primitive. Sparkline slot is reserved but optional; recharts stays external.

3. **[Scope]** What happens to the existing ~19 per-page partner components after Phase 3 migration?
   - Options: Delete replaced ones | Keep as thin wrappers | Decide per-file
   - **Answer:** **Delete the replaced ones**
   - **Rationale:** Aligns with `frontend/AGENTS.md` DON'T #6 — "DON'T maintain multiple versions of the same component." Phase 3 implementer deletes files that become no-op wrappers; deletions are listed in the Phase 4 verification report for traceability.

4. **[Scope]** Is Phase 1 PartnerSidebar polish in scope? Current sidebar already has a coherent green theme.
   - Options: Keep sidebar polish | Defer to later | Tokens only no structural change
   - **Answer:** **Sidebar follow admin sidebar design**
   - **Custom input (verbatim):** "sidebar follow admin sidebar design"
   - **Rationale:** Partner sidebar must visually match admin's. Since admin sidebar uses daisyUI `ct-menu`, partner needs the same shell. Phase 1 expands to include daisyUI sidebar rendering for partner (either via a small partner theme OR by adopting the admin sidebar pattern with shadcn primitives). This is the only place daisyUI is used for partner.

#### Confirmed Decisions

- **Token strategy: shadcn-first hybrid.** shadcn primitives are the workhorse; Tailkit MCP is the visual reference catalog (read-only, not installed); daisyUI is permitted for the partner sidebar only, mirroring admin.
- **`PartnerStatsCard` API:** full proposed API (label, value, valueFormat, unit, trend, icon, onClick, loading, sparkline slot). Cloned from `PremiumStatStrip` proven shape, extended with `icon` and `sparkline`.
- **Old partner components:** deleted in Phase 3 once replaced; deletions tracked in Phase 4 report.
- **PartnerSidebar:** must follow admin sidebar design (daisyUI `ct-menu` pattern). Phase 1 expanded to cover this.

#### Action Items

- [ ] Update Phase 1 to include partner-sidebar daisyUI rendering following admin pattern.
- [ ] Update Phase 2 `PartnerStatsCard` spec to clone `PremiumStatStrip` API (label, value, unit, highlight, onClick, trend) and extend with icon + sparkline slot. Reference `src/components/admin-dashboard/PremiumStatStrip.tsx` as the canonical example.
- [ ] Update Phase 2 to note `PremiumStatStrip` is the canonical admin precedent — partner primitive should be API-compatible where feasible.
- [ ] Update Phase 3 to explicitly delete replaced per-page components; deletions listed in Phase 4 report.
- [ ] Update Phase 4 success criteria to include: partner sidebar visually matches admin sidebar; `PremiumStatStrip` API parity check passes.
- [ ] Update plan.md "Key Decisions" to reflect the shadcn-first hybrid and `PremiumStatStrip` precedent.

#### Impact on Phases

- **Phase 1:** Sidebar scope expanded to include daisyUI rendering (admin pattern). New step: add partner daisyUI theme OR re-render admin sidebar with shadcn primitives. Risk note updated.
- **Phase 2:** `PartnerStatsCard` implementation now has a proven admin reference (`PremiumStatStrip`). Reduces YAGNI risk.
- **Phase 3:** Explicit deletion step added per route migration.
- **Phase 4:** New acceptance criterion for sidebar visual parity with admin.

### Whole-Plan Consistency Sweep

- **Files reread:** `plan.md`, `phase-01-partner-design-foundation.md`, `phase-02-shared-partner-primitives.md`, `phase-03-partner-route-migration.md`, `phase-04-visual-qa-and-verification.md`.
- **Decision deltas checked:** 4 (token strategy, stats API, deletion policy, sidebar scope).
- **Reconciled stale references:** 1
  - `phase-03...md` Related Code Files → Delete rule said "only files that become true no-op wrappers... Otherwise keep as thin composition roots" — contradicted Q3's "delete replaced ones" decision. Rewritten to align with Q3 + `frontend/AGENTS.md` DON'T #6.
- **Unresolved contradictions:** 0
- **Stale-term grep results:** All remaining mentions of "pure semantic-CSS layer", "YAGNI", "speculative", "No daisyUI" appear only inside the Validation Log's question/answer context where they belong (describing the rejected/modified proposals). No stale assertions in normative plan body.
- **Recommendation:** Plan is internally consistent. Eligible for `/ck:cook` or `/ck:plan red-team` per user preference.

## Touchpoints (Indicative)

- **Tokens / build:** `frontend/tailwind.config.ts` (extend `themeRoot` only if absolutely necessary — preferred: leave alone), `frontend/src/styles/partner.css` (new), `frontend/src/styles/index.css` or main entry (import partner.css once).
- **Shell:** `frontend/src/layouts/PartnerLayout.tsx`, `frontend/src/components/PartnerSidebar.tsx`, `frontend/src/components/MobileBottomNav.tsx` (partner branch only).
- **New primitives:** `frontend/src/components/partner-ui/` (barrel + ~6 components).
- **Routes:** everything under `frontend/src/pages/partner/*` and `frontend/src/components/partner-*/*`.
