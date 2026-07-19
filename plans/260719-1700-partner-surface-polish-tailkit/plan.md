---
title: Partner Surface Polish (Rewrite)
description: >-
  Consistency-and-parity polish across the /partner/* route tree (desktop pages,
  mobile pages, shared sidebar, shared Radix-portaled sheets) using the existing
  components/shared/ and components/ui/ libraries as the foundation. Tailkit MCP
  is the visual reference catalog only. Built on a corrected codebase survey
  after v1 of this plan was rejected by red-team for targeting dead-code folders
  and missing the real mobile routes.
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
  - consistency
blockedBy: []
blocks: []
created: '2026-07-19T14:01:24.581Z'
createdBy: 'ck:plan'
source: skill
supersedes: plans/260719-1700-partner-surface-polish-tailkit-v1-superseded
---

# Partner Surface Polish (Rewrite)

## Overview

Bring the five partner routes and their mobile variants to a single, coherent visual standard by **reusing what already exists** — the large `components/shared/*` library (31 `.tsx` files plus `index.ts`: `PageHeader`, `MobilePageHeader`, `MobilePageShell`, `InlineStatStrip`, `GroupedStatCard`, `StatsCards`, `FilterBar`, `FilterPill`, `EmptyState`, `MobileCard`, `MobileOperationsPanel`, …) and `components/ui/responsive-table`. The actual partner surfaces already import from these libraries; the work is consistency, gap-filling, and removal of dead code — not a new primitives layer.

The Tailkit MCP catalog (`mcp__tailkit__browse_catalog`, `mcp__tailkit__search_components`, `mcp__tailkit__get_component_code`) is a **visual reference** for stat layouts, empty states, and table styling. Tailkit ships Tailwind v3.4-compatible utilities (`bg-secondary-100/75`, `text-secondary-500`, `dark:` variants are all v3.4-valid — v1's "v4 translation tax" claim was unsubstantiated and is dropped). Adopted patterns are translated to project tokens (`bg-card`, `text-muted-foreground`, `hsl(var(--success))`), never copy-pasted.

## Why a Rewrite (v1 rejection summary)

v1 of this plan (now archived at `plans/260719-1700-partner-surface-polish-tailkit-v1-superseded/`) was rejected by red-team review for ten verified Critical/High findings, all confirmed by the planner:

1. v1 targeted `components/partner-employees/*`, `partner-projects/*`, `partner-timesheet/*` — these are **orphaned dead code** with zero external importers (`grep -rln "from \"@/components/partner-" --include="*.tsx" --include="*.ts" | grep -v partner-` returns empty).
2. v1 omitted `pages/mobile/partner/*/index.tsx` — the actual mobile routes wired via `ResponsivePage` (`App.tsx:280-284`).
3. v1 claimed "admin sidebar uses daisyUI `ct-menu`" — `grep -E '\bct-[a-z]' AdminSidebar.tsx` returns **0 matches**. AdminSidebar is shadcn-only.
4. v1 gated Phase 4 on `make api-test` — the target only exists at `backend/Makefile:242`, not the root.
5. v1 proposed scoping `--partner-*` tokens under `[data-partner-ui]` — but `--partner-accent` already lives at `:root` in `variables.css:212` and is consumed by `PartnerSidebar.tsx`.
6. v1's `[data-partner-ui]` attribute scoping cannot reach Radix portals (Dialog/Sheet/DropdownMenu render to `document.body`). Admin solves this via `document.documentElement.classList.add('admin-route-active')` (`AdminLayout.tsx:101-102`).
7. v1's `PartnerDataTable` primitive is redundant — `ui/data-table.tsx`, `ui/mobile-table.tsx`, `ui/responsive-table.tsx` already exist.
8. v1's `PartnerStatsCard`/`PremiumStatStrip` "API parity" claim was overstated — PremiumStatStrip takes `items: PremiumStatItem[]` (strip), partner routes use `InlineStatStrip` and `GroupedStatCard`.
9. v1's "TanStack Table preserved" assumption holds for only 1 of 5 partner routes (Employees); others use plain table markup or `ResponsiveTable`.
10. v1's `pages/partner/PaymentHistoryPage/index.tsx` migration target is a 5-line re-export of `BankTransferHistoryPageContent`, shared with admin — restyling it as partner-only would regress admin.

The v2 plan is rebuilt on the corrected survey below.

## Corrected Codebase Survey (authoritative for this plan)

### Partner desktop routes (`pages/partner/*`)

| Route | Page file | Real imports (styling source) |
|---|---|---|
| `/partner/dashboard` | `DashboardPage/index.tsx` | `components/shared/PageHeader`, `components/partner-dashboard/{PartnerEmployeeListSheet, PartnerWorkforceOverviewCard}` (these two ARE live), `ui/skeleton`, `ui/user-avatar` |
| `/partner/projects` | `ProjectsPage/index.tsx` | `components/shared/PageHeader`, `ui/skeleton`, hooks |
| `/partner/employees` | `EmployeesPage/index.tsx` | `components/shared/{PageHeader, InlineStatStrip, SearchBar, FilterPill}`, `ui/responsive-table`, `components/employees/{MissingBankDetailsSection}`, `components/modals/ExportEmployeesModal` |
| `/partner/timesheet` | `TimesheetsPage/index.tsx` | `components/timesheet/{TimesheetFilters, TimesheetListTable, TimesheetMobileList, TimesheetProvider, EditRequestTable}`, `components/transaction/ExportSaoKeDialog`, `components/payroll/PaymentHistorySheet` |
| `/partner/timesheet/payment-history` | `PaymentHistoryPage/index.tsx` | **5-line re-export** of `components/payroll/BankTransferHistoryPageContent` (SHARED WITH ADMIN — out of scope) |

### Partner mobile routes (`pages/mobile/partner/*`)

All wired via `ResponsivePage` (`App.tsx:280-284`):

| Route | Page file | Real imports (styling source) |
|---|---|---|
| `/partner/dashboard` (mobile) | `pages/mobile/partner/DashboardPage/index.tsx` | `components/shared/{MobilePageHeader, MobilePageShell, MobileSurface, MobileOperationsPanel, MobileTaskList}` |
| `/partner/projects` (mobile) | `pages/mobile/partner/ProjectsPage/index.tsx` | `components/shared/{MobileSearchInput, MobilePageHeader, MobilePageShell, MobileSurface}`, `components/projects/ProjectMobileList` |
| `/partner/employees` (mobile) | `pages/mobile/partner/EmployeesPage/index.tsx` | `components/shared/{MobileSearchInput, MobilePageHeader, MobilePageShell, MobileSurface}`, `components/employees/{EmployeeMobileCard, EmployeeEmptyStates}` |
| `/partner/timesheet` (mobile) | `pages/mobile/partner/TimesheetsPage/index.tsx` | `components/timesheet/mobile/TimesheetPageHeaderMobile`, `components/timesheet/TimesheetDisplaySection`, `components/shared/{GroupedStatCard, MobilePageShell, MobileSurface}` |
| `/partner/timesheet/payment-history` (mobile) | **NOT WIRED** — file exists at `pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx` but `App.tsx` renders desktop `PartnerPaymentHistoryPage` on both viewports |

### Shell

- `layouts/PartnerLayout.tsx:1-46` — `ProtectedRoute requiredRole="partner"` → `SidebarProvider` → `PartnerLayoutInner` (with `SidebarToggle`, `<main>` + `SectionErrorBoundary sectionName="trang đối tác"` → `Outlet`) + `MobileBottomNav groups={PARTNER_NAV_GROUPS}` + `NotificationFAB`. No data-attribute scope today.
- `components/PartnerSidebar.tsx` — shadcn-only (`ui/sidebar`, `ui/dropdown-menu`, `ui/tooltip`, lucide icons). Uses `--partner-accent` from `:root`/`variables.css:212`. NavLink paths: dashboard, projects, employees, timesheet, timesheet/payment-history.
- `components/MobileBottomNav.tsx` — shared file with role-keyed branches.

### Existing primitives (do NOT recreate)

- `components/shared/*` — 31 `.tsx` files plus barrel `index.ts`. Includes: `PageHeader`, `MobilePageHeader`, `MobilePageShell`, `MobileSurface`, `InlineStatStrip`, `GroupedStatCard`, `StatsCards`, `SummaryStatsCards`, `FilterBar`, `FilterPill`, `MobileFilterPill`, `SearchBar`, `MobileSearchInput`, `EmptyState`, `MobileCard`, `MobileOperationsPanel`, `MobileSectionHeader`, `MobileSheetHeader`, `MobileSubPageHeader`, `SheetFooter`, `QuickActionBar`, `AccentStripCard`, `ContextStrip`, `EarningsPill`, `InfoPanel`, `InlineAlert`, `MobilePagination`, `MobileProgressBar`, `ShortcutHint`, `TabBarWithBadges`, `UserAvatarDropdown`.
- `components/ui/responsive-table.tsx`, `components/ui/data-table.tsx`, `components/ui/mobile-table.tsx` — TanStack-powered, already support search + mobile.
- `components/ui/sidebar.tsx`, `dialog.tsx`, `sheet.tsx`, `dropdown-menu.tsx`, `badge.tsx`, `card.tsx`, `skeleton.tsx`, `button.tsx` — shadcn primitives, untouched.

### Dead code candidates (Phase 2 will confirm before any deletion)

Verified to have **zero external importers** via `grep -rln "from \"@/components/partner-" --include="*.tsx" --include="*.ts" | grep -v partner-`:

- `components/partner-employees/*` (entire folder — 8 files incl. mobile)
- `components/partner-projects/*` (entire folder — 5 files incl. mobile)
- `components/partner-timesheet/*` (entire folder — 7 files incl. mobile)

These are NOT touched in v1's planned way (migrate-then-delete). v2 deletes them outright in Phase 3 if Phase 2 audit confirms orphans — but the deletion is a **cleanup side-effect**, not a migration step.

### Live `partner-dashboard/` components (NOT orphaned)

- `components/partner-dashboard/PartnerEmployeeListSheet.tsx` — imported by `pages/partner/DashboardPage/index.tsx`.
- `components/partner-dashboard/PartnerWorkforceOverviewCard.tsx` — imported by `pages/partner/DashboardPage/index.tsx`. Uses recharts `PieChart` + `Skeleton`.

## Scope

### In

- Polish for visual consistency and parity across the **live** partner surfaces:
  - 4 desktop pages: Dashboard, Projects, Employees, Timesheets (PaymentHistoryPage is fully OUT — see H2/H3 fix).
  - 4 mobile pages: same four routes via `ResponsivePage`.
  - Partner shell: `PartnerLayout`, `PartnerSidebar`, partner-branch of `MobileBottomNav`.
- **Adopt `components/shared/*` primitives** where any partner page has drifted (e.g. a page that hand-rolls a header instead of using `PageHeader`).
- **Tailkit as visual reference** for: stat-card layouts (`a-c-statistics-09`, `a-c-statistics-10`), empty states (`a-c-empty-states-03`, `a-c-empty-states-05`), table styling (`a-c-tables-13`).
- **Delete confirmed dead code** in `components/partner-{employees,projects,timesheet}/` after Phase 2 audit.
- Portal-scope strategy for partner-shared sheets (mirror admin's two-part pattern — classList effect + `[data-partner-ui]` wrapper attribute).

### Out

- `pages/partner/PaymentHistoryPage/index.tsx` restyle AND mobile-route wiring — fully out (v2.1 H2/H3). The desktop file is a 5-line re-export of `BankTransferHistoryPageContent`, which is **variant-aware** (`variant?: 'admin' | 'partner'`) and renders different content per variant. The optional mobile-wiring proposal is removed: the mobile file uses different service methods (`useInfinitePaymentHistories` + `useExportPaymentHistories`) than desktop (`useBankTransferHistories`), so wiring it changes the data a partner sees on mobile vs desktop. That's a product decision, not polish.
- Backend/API/schema/business-rule changes, new product features, route renames.
- Modal contract changes (registry IDs, deep-link `returnTo`, focus trap behavior).
- Tailwind 4 migration, daisyUI version bump, adding Tailkit as an npm dependency.
- Replacing Radix primitives.
- Replacing `recharts`.
- Redesigning admin or employee surfaces — non-regression boundaries.
- New global primitives in `components/ui/*` (composition only).

### Preserve

- `ProtectedRoute requiredRole="partner"` guard.
- URL query state (`?month=&project=&status=&search=&page=&sort=`) and `useSearchParams` semantics on Timesheets/Employees/Projects.
- All `useQuery` / `useMutation` hooks and cache invalidation rules.
- Deep-linked sheets (`modal-registry-auto.ts` / `modalRegistry/*`).
- Vietnamese copy and `vi-VN` number/date formatting with tabular numerals for money.
- `ResponsivePage` desktop/mobile switching contract (`useIsMobile`).
- `SectionErrorBoundary sectionName="trang đối tác"` wrap in `PartnerLayout`.

## Key Decisions

1. **Reuse over recreate.** `components/shared/*` already has every primitive v1 tried to invent (`PartnerPageHeader` → `PageHeader`, `PartnerStatsCard` → `InlineStatStrip` + `GroupedStatCard`, `PartnerEmptyState` → `EmptyState`, `PartnerFilterBar` → `FilterBar`, `PartnerMobileListRow` → `MobileCard`). v2 extends these only when a real partner gap exists.
2. **Tailkit is visual reference only.** Use `mcp__tailkit__get_component_code` to read patterns during Phase 3; translate to project tokens. Never install. v3.4-compatible utilities mean the translation is lighter than v1 claimed — most Tailkit classes are directly valid Tailwind, the only changes are color tokens.
3. **Portal scoping mirrors admin's TWO-PART pattern (v2 Critical fix C1).** Admin uses BOTH a `html.admin-route-active` classList effect (for base/global rules) AND a `data-admin-ui` wrapper attribute (the actual portal-reach selector is `[data-admin-ui] [data-radix-popper-content-wrapper]` at `admin-daisy.css:35`). v2 originally copied only the classList half — v2.1 restores both. The classList effect is guarded by `[isPartner]` deps + early return (H1 mitigation for ProtectedRoute's ~100ms auth-failure redirect window at `ProtectedRoute.tsx:27`). Phase 1 Step 9 explicitly verifies which selector actually reaches Radix portals rendered to `document.body`.
4. **Dead-code deletion is a Phase 3 side-effect, not a migration.** `components/partner-{employees,projects,timesheet}/` are confirmed orphaned; Phase 3 deletes them after a final grep safety check. The "consolidate then delete" framing of v1 was wrong — there's nothing to consolidate, the routes don't import them.
5. **Payment History is OUT of scope entirely (v2.1 fix H2, H3).** `BankTransferHistoryPageContent` is **variant-aware** (`variant?: 'admin' | 'partner'`, line 280), not byte-shared — admin and partner already render differently. Worse, the optional mobile-Payment-History wiring proposed in v2 is **a data divergence, not a presentation swap**: desktop partner uses `useBankTransferHistories` (`payroll.service.ts:17`, paginated, `cycle` filter) while the mobile file uses `useInfinitePaymentHistories` + `useExportPaymentHistories` (`payroll.service.ts:30,41`, infinite scroll, `projectId/employeeId/position/fromDate/toDate` filters). Wiring the mobile file would change what data a partner sees on mobile vs desktop. v2.1 removes the mobile-wiring option entirely; Payment History is touched in Phase 4 only as a regression check (admin variant must remain admin-styled).
6. **No new sidebar structural work.** `PartnerSidebar` is shadcn-only and uses `--partner-accent` from `:root`. v2 only adjusts visual tokens (colors, spacing, hover states) if Phase 2 audit finds drift from admin's visual standard — but admin sidebar is ALSO shadcn-only (no `ct-*`), so they're already structurally aligned.
7. **`make api-test` lives at `backend/Makefile:242`.** Phase 4 invokes it via `cd backend && make api-test`, not `make api-test` from repo root.

## Phases

| Phase | Name | Status | Purpose |
|-------|------|--------|---------|
| 1 | [Foundation](./phase-01-foundation.md) | Pending | Completed |
| 2 | [Audit and Primitives Parity](./phase-02-audit-and-primitives-parity.md) | Pending | Per-route audit: which `shared/*` primitives are used, which patterns drift, dead-code confirmation |
| 3 | [Route Polish](./phase-03-route-polish.md) | Pending | Apply consistency fixes per route (desktop + mobile together), delete confirmed dead code per-file (mobile Payment History wiring is OUT — v2.1 H2) |
| 4 | [Verification](./phase-04-verification.md) | Pending | Lint, type-check, build, `cd backend && make api-test`, partner Playwright, cross-surface isolation, dead-code-removal verification, `graphify update .` |

## Dependencies

- **No blocking cross-plan dependency.** Admin (`260718-2242-admin-daisyui-redesign`) and Employee (`260719-1310-employee-mobile-polish`) are both complete; non-regression boundaries only.
- **Internal:** Phase 2 depends on Phase 1. Phase 3 depends on Phase 2 audit results. Phase 4 depends on Phase 3.
- **Wallet bulk-transfer plan** (`260718-2130-wallet-bulk-transfer-pipeline`) touches `/admin/wallet` only; no overlap.

## Acceptance Criteria

- All four in-scope partner routes (Dashboard, Projects, Employees, Timesheets — desktop + mobile) render with consistent header treatment, stat-card style, filter bar pattern, empty/loading/error states, drawn from `components/shared/*`.
- `html.partner-route-active` classList effect AND `[data-partner-ui]` wrapper attribute both work as the two-part portal-scope mirror of admin. Class is added on partner route mount, removed on unmount, and absent on `/admin/*`, `/employee/*`, `/login` AND during ProtectedRoute's auth-failure redirect window.
- Portal-reach selector verified working (Phase 1 Step 9 resolution): Radix portals opened from partner routes are reachable by the chosen selector.
- Confirmed dead-code folders (`components/partner-{employees,projects,timesheet}/`) are deleted per-file (Phase 2 audit may keep individual files if greps find importers); `grep -rln "from \"@/components/partner-"` returns zero hits across `src/` only after each file's three independent greps confirmed zero importers.
- `/partner/timesheet/payment-history` AND admin's payment-history view both render unchanged — `BankTransferHistoryPageContent` variant logic untouched (v2.1 H3 fix).
- No horizontal scroll at 320 / 390 / 768 / 1440 on any in-scope partner route.
- All interactive elements ≥44px on mobile; focus-visible rings preserved; tabular numerals on every money figure; status badges pair color with icon + text.
- **Tailkit literal-class ban is DELTA-ONLY (v2.1 C2 fix):** the grep checks that no NEW `secondary-*`/`emerald-*`/`sky-*`/`violet-*`/`orange-*`/`slate-*`/`dark:` classes were INTRODUCED by this plan — NOT that zero exist (37 pre-existing matches in partner routes today are out of scope).
- URL query state, deep-linked sheets, mutations, and Vietnamese copy preserved.
- `pnpm lint`, `tsc --noEmit`, `pnpm build` pass; `cd backend && make api-test` passes; partner Playwright specs pass (or documented as vacuous — see Phase 4 H5 fix); `graphify update .` exits 0.
- Cross-surface isolation: `/admin/*`, `/employee/*`, `/login` visually unchanged (Phase 4 captures fresh baselines if no prior baseline exists).

## Risk Controls

- **Token drift between surfaces.** Mitigation: Phase 2 publishes a "shared primitive adoption map" so every Phase 3 route change references it.
- **Accidental deletion of live code.** Mitigation: Phase 3 deletion requires two independent greps confirming zero importers (one for type imports, one for value imports), plus a `tsc --noEmit` pass after deletion.
- **Admin regression on shared components.** Mitigation: any edit to a `components/shared/*` or `components/payroll/*` file triggers a baseline-vs-after screenshot on the admin route that consumes it.
- **Portal scoping leak.** Mitigation: Phase 4 grep for `html.partner-route-active` selectors must match only partner rules; verify class is removed on route change.
- **Mobile Payment History wiring breaks admin.** Mitigation: the desktop re-export stays untouched; only `App.tsx` route definition may change to wrap it in `ResponsivePage` with the existing mobile file.

## Touchpoints (Indicative)

- **Create:** `frontend/src/styles/partner.css` (new, scoped under `html.partner-route-active`).
- **Modify:**
  - `frontend/src/layouts/PartnerLayout.tsx` — add the `html.partner-route-active` classList effect.
  - `frontend/src/index.css` — single `@import "./styles/partner.css";`.
  - Per-route polish targets identified by Phase 2 audit (likely: `pages/partner/{Dashboard,Projects,Employees,Timesheets}Page/index.tsx` and `pages/mobile/partner/*/index.tsx`).
  - `frontend/src/App.tsx` — **NOT MODIFIED (v2.1 H2 fix).** Mobile Payment History wiring removed from scope; the mobile file uses different service methods than desktop, so wiring it changes the data a partner sees on mobile vs desktop.
- **Delete (Phase 3, post-audit):** `frontend/src/components/partner-employees/`, `partner-projects/`, `partner-timesheet/` (confirmed orphaned).
- **Read-only references:** Tailkit MCP snippets; `components/shared/*`; `AdminLayout.tsx:101-102` (classList effect pattern); `AdminSidebar.tsx` (shadcn-only sidebar reference); `variables.css:212` (`--partner-accent` location).

## Open Questions

1. ~~Is wiring the missing mobile Payment History route in scope?~~ **RESOLVED v2.1 (H2):** Out of scope entirely. Mobile file uses different service methods than desktop — wiring changes data, not presentation.
2. Are there baseline screenshots of admin/employee/login for the Phase 4 cross-surface check, or does Phase 4 capture fresh baselines? (Default: capture fresh — the existing admin/employee polish plans don't reference a shared baseline dir. v2 red-team Failure Mode #8 confirmed no baseline infrastructure exists.)
3. **NEW v2.1:** Should the pre-existing 37 Tailkit literal-class matches in partner routes today (DashboardPage ×16, EmployeesPage ×6, ProjectsPage ×15) be cleaned up as part of this plan, or deferred? Default: **defer** — they predate this plan and folding them in expands scope significantly.

## Red Team Review

### Session 1 — 2026-07-19 (against v1, archived)

**Trigger:** User-requested adversarial review of v1 after Validation Session 1.
**Reviewers:** 3 (Security Adversary, Failure Mode Analyst, Assumption Destroyer) — Standard tier.
**Findings:** 31 raw → 10 deduped Critical/High after evidence filter.
**Outcome:** v1 rejected; full rewrite authorized by user.

Reviewer reports (preserved for traceability in `reports/`):
- `from-code-reviewer-to-planner-red-team-security-adversary-plan-review-report.md`
- `from-code-reviewer-to-planner-red-team-failure-mode-analyst-plan-review-report.md`
- `from-code-reviewer-to-planner-red-team-assumption-destroyer-plan-review-report.md`

| # | Finding | Severity | Disposition | Resolved in v2 by |
|---|---------|----------|-------------|-------------------|
| 1 | v1 targeted `components/partner-{employees,projects,timesheet}/` which are orphaned dead code | Critical | Accept | Completed |
| 2 | v1 omitted `pages/mobile/partner/*` (real mobile routes wired via ResponsivePage) | Critical | Accept | v2 Corrected Codebase Survey lists all 4 mobile routes explicitly; Phase 3 mandates desktop + mobile in the same step. |
| 3 | v1 claimed admin sidebar uses daisyUI `ct-menu` — false, 0 matches | Critical | Accept | v2 Key Decision #6: PartnerSidebar is already shadcn-only; no structural rewrite. |
| 4 | v1 gated Phase 4 on root `make api-test` — doesn't exist there | Critical | Accept | v2 Key Decision #7 and Phase 4 Step 1: invoke via `cd backend && make api-test`. |
| 5 | v1 proposed scoping `--partner-*` under `[data-partner-ui]` — token is at `:root` | Critical | Accept | v2 Key Decision #3: portal-scope via `html.partner-route-active` classList; `--partner-accent` stays at `variables.css:212`. |
| 6 | v1's `[data-partner-ui]` attribute scoping can't reach Radix portals | Critical | Accept | v2 Key Decision #3 and Phase 1 Architecture: mirror AdminLayout's `document.documentElement.classList` pattern. |
| 7 | v1's `PartnerDataTable` is redundant — `ui/{responsive,data,mobile}-table` exist | High → Critical | Accept | v2 Key Decision #1: reuse over recreate; Phase 2 maps existing primitives. |
| 8 | v1's PremiumStatStrip "API parity" overstated; partner uses InlineStatStrip | High | Accept | v2 Key Decision #1: adopt existing `InlineStatStrip` / `GroupedStatCard` / `StatsCards`. |
| 9 | v1's TanStack assumption held for only 1 of 5 partner routes | High | Accept | v2 Survey records actual table primitive per route; Phase 3 preserves what's there. |
| 10 | v1's PaymentHistoryPage target is a 5-line re-export shared with admin | High | Accept | v2 Scope explicitly excludes PaymentHistoryPage restyle; only optional mobile-route wiring is in scope. |

Plus Medium-severity findings folded into v2: `partner.css` import location (use `src/index.css`, not the non-existent `src/styles/index.css`), Tailwind v4 "translation tax" claim dropped (Tailkit classes are v3.4-valid), Vitest partner-spec gate documented as potentially vacuous.

### Whole-Plan Consistency Sweep (v2 post-rewrite)

- **Files reread:** `plan.md` (v2), all four `phase-*.md` (v2).
- **Decision deltas checked:** 10 (from v1 rejection) + 5 (v2 Key Decisions).
- **Reconciled stale references:**
  - Corrected `shared/` count from "32 files" to "31 `.tsx` files plus barrel" (verified via `ls frontend/src/components/shared/*.tsx | wc -l` → 31).
  - Removed all v1-era claims that admin sidebar uses daisyUI.
  - Removed all v1-era claims that `[data-partner-ui]` is the scope mechanism.
  - Removed all v1-era references to migrating `components/partner-*` folders.
  - Confirmed v2 Phase 4 invokes `make api-test` only via `cd backend &&`.
- **Unresolved contradictions:** 0.
- **v2 Fact-Checker pass:** 8 claims verified against codebase (`shared/` count, ResponsivePage contract, AdminLayout classList line 99-103, `backend/Makefile:242`, `variables.css:212`, `App.tsx:278-289`, `index.css` aggregator, PaymentHistoryPage 5-line re-export). All VERIFIED.
- **Recommendation:** v2 is internally consistent and grounded in the actual codebase shape. Eligible for `/ck:cook` or another validation pass per user preference.

### Session 2 — 2026-07-19 (v2 red-team, leading to v2.1)

**Trigger:** User-requested second adversarial pass after v2 rewrite.
**Reviewers:** 3 (Security Adversary, Failure Mode Analyst, Assumption Destroyer) — Standard tier.
**Findings:** 30 raw → 10 deduped Critical/High after evidence filter.
**Verdict on v1's 10 findings:** RESOLVED at plan-text level (all three reviewers agreed).
**Verdict on v2's NEW findings:** 1 Critical + 4 High required a v2.1 patch.

Reviewer reports (preserved for traceability in `reports/`):
- `from-code-reviewer-to-planner-red-team-v2-security-adversary-plan-review-report.md`
- `from-code-reviewer-to-planner-red-team-v2-failure-mode-analyst-plan-review-report.md`
- `from-code-reviewer-to-planner-red-team-v2-assumption-destroyer-plan-review-report.md`

| # | Finding | Severity | Disposition | v2.1 Resolution |
|---|---------|----------|-------------|-----------------|
| C1 | v2 copied only the classList half of admin's portal-scope pattern — the actual portal-reach selector is `[data-admin-ui] [data-radix-popper-content-wrapper]` at `admin-daisy.css:35`, not `html.admin-route-active ...`. Radix portals (rendered to `document.body`) wouldn't be reachable. | Critical | Accept | Phase 1 rewritten: TWO-PART pattern — classList effect for base/global + `[data-partner-ui]` wrapper attribute for portal reach. Phase 1 Step 9 explicitly verifies which selector actually reaches Radix portals. |
| C2 | Tailkit literal-class ban grep fails today: 37 pre-existing matches in partner routes (DashboardPage ×16, EmployeesPage ×6, ProjectsPage ×15). Phase 4 gate was unachievable. | Critical | Accept | Phase 3 + Phase 4 success criteria changed to DELTA-ONLY grep: `git diff main ... \| grep "^\+.*literal"` returns zero NEW additions. Pre-existing 37 are out of scope (Open Question 3). |
| H1 | ProtectedRoute has ~100ms `setTimeout` redirect on auth failure; class could leak during that window. AdminLayout guards with `[isAdmin]` deps + early return; v2's `[]` deps had no guard. | High | Accept | Phase 1 effect now uses `[isPartner]` deps + `if (!isPartner) return;` early return. Phase 1 Step 11 verifies the auth-failure window. |
| H2 | Optional mobile Payment History wiring is a DATA divergence, not a presentation swap. Desktop uses `useBankTransferHistories`; mobile uses `useInfinitePaymentHistories` + `useExportPaymentHistories`. Different endpoints, different filter schemas. | High | Accept | Mobile Payment History wiring REMOVED from scope entirely. Phase 3 Step 8 struck through; plan.md Touchpoints + Scope + Open Question 1 updated. |
| H3 | `BankTransferHistoryPageContent` is variant-aware (`variant?: 'admin' \| 'partner'`, line 280), not byte-shared. Phase 4's "visually identical" check was unachievable. | High | Accept | Phase 4 Step 6 rewritten: check that both variants render correctly AND that the file is byte-unchanged from pre-plan state. |
| H4 | Phase 2 dynamic-import grep `partner-employees/` false-positives on LIVE `hooks/partner-employees/usePartnerEmployeesData`. | High | Accept | Phase 2 Step 4 greps now anchor on `components/partner-{folder}` (not just `partner-{folder}`). |
| H5 | Phase 4 Playwright partner-specs gate is vacuous — zero partner specs exist. | High | Accept | Phase 4 Step 1 + Success Criteria updated: Vitest AND Playwright partner-spec gates documented as potentially vacuous, with spec files that SHOULD exist listed as follow-up. |
| M1 | Modal-registry grep targeted wrong dirs (`constants/`, `lib/`); `lib/modal-registry-auto.ts` uses `import.meta.glob`. | Medium | Accept | Phase 2 Step 4 + Phase 4 Step 3 modal-registry grep now scans all source with `--include` filters. |
| M2 | Phase 3 deletion leaves stale AGENTS.md references (4+ files reference deleted paths). | Medium | Accept | Phase 3 Step 7 + Modify list updated: AGENTS.md cleanup is an explicit sub-step. |
| M3 | "32 files" still appeared in `phase-02:43` despite `plan.md` sweep correcting it. | Medium | Accept | `phase-02` corrected to "31 `.tsx` files plus `index.ts` barrel". |

### Whole-Plan Consistency Sweep (v2.1 post-red-team)

- **Files reread:** `plan.md`, all four `phase-*.md`.
- **Decision deltas checked:** 10 (v2.1 red-team accepted findings).
- **Reconciled stale references:** 4
  - `plan.md:156` phase table row said "wire missing mobile Payment History if in scope" — rewritten.
  - `plan.md:194` Touchpoints said App.tsx IF-in-scope swap — rewritten to NOT MODIFIED.
  - `phase-03:13` Overview said "Optionally wire the missing mobile Payment History route" — rewritten.
  - `phase-02:43` said "32 files" for shared/ — corrected to 31 + barrel.
- **Unresolved contradictions:** 0.
- **Recommendation:** v2.1 is internally consistent and addresses every verified Critical/High finding from both red-team sessions. Eligible for `/ck:cook` or another validation pass per user preference.
