---
phase: 2
title: Audit and Primitives Parity
status: completed
generated: '2026-07-19'
auditor: ck:cook (Phase 2)
consumed_by: phase-03-route-polish
---

# Phase 2 Audit Report

Read-only audit of the 8 in-scope partner surface files (4 desktop + 4 mobile) plus dead-code confirmation. Output is consumed by Phase 3 as its work list.

## 0. Scope recap

In-scope (8 page files):

| # | File |
|---|------|
| D1 | `frontend/src/pages/partner/DashboardPage/index.tsx` |
| D2 | `frontend/src/pages/partner/ProjectsPage/index.tsx` |
| D3 | `frontend/src/pages/partner/EmployeesPage/index.tsx` |
| D4 | `frontend/src/pages/partner/TimesheetsPage/index.tsx` |
| M1 | `frontend/src/pages/mobile/partner/DashboardPage/index.tsx` |
| M2 | `frontend/src/pages/mobile/partner/ProjectsPage/index.tsx` |
| M3 | `frontend/src/pages/mobile/partner/EmployeesPage/index.tsx` |
| M4 | `frontend/src/pages/mobile/partner/TimesheetsPage/index.tsx` |

Out-of-scope (re-affirmed, do not touch): `frontend/src/pages/partner/PaymentHistoryPage/index.tsx` (5-line re-export of `BankTransferHistoryPageContent`, shared + variant-aware with admin — v2.1 H2/H3).

Shared inventory confirmed: `ls frontend/src/components/shared/*.tsx | wc -l` → **31** (matches plan v2.1 M3).

---

## 1. Per-route audit table

Each row cites `file:line`. Treatments marked **✅ Adopted** / **⚠️ Hand-rolled** / **➖ N/A**.

### D1 — Desktop Dashboard (`pages/partner/DashboardPage/index.tsx`, 517 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell bg | Hand-rolled radial gradient | `index.tsx:335` `bg-[radial-gradient(...rgba(8,120,62,0.12)...)]` |
| Header | ✅ `PageHeader` (shared) | `index.tsx:19,340` |
| Header card wrapper | Hand-rolled `rounded-3xl border border-emerald-100` + corner decor | `index.tsx:337-339` |
| Hero stat | Hand-rolled gradient hero card with dark-on-emerald style | `index.tsx:358-401` |
| Stat tiles | Hand-rolled `StatTile` local component (watermark-style) | `index.tsx:125-194` |
| Loading | Hand-rolled `HeroSkeleton` / `LeaderboardSkeleton` | `index.tsx:273-305` |
| Empty | Hand-rolled inline block in leaderboard card | `index.tsx:466-474` |
| Trend chip | Hand-rolled `TrendChip` with literal `emerald-*` / `rose-*` / `slate-*` classes | `index.tsx:70-113` |
| Status badge | Hand-rolled color pill `bg-rose-50 text-rose-700` w/ icon (good: icon+text+color) | `index.tsx:249-253` |
| Child components | `PartnerWorkforceOverviewCard`, `PartnerEmployeeListSheet` from `components/partner-dashboard/` | `index.tsx:21-22` |

**Drift vs other routes:** unique hero-card / leaderboard pattern; only desktop route that hand-rolls its own stat tiles.

### D2 — Desktop Projects (`pages/partner/ProjectsPage/index.tsx`, 501 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell bg | Hand-rolled radial gradient (mirrors D1) | `index.tsx:273` |
| Header | ✅ `PageHeader` (shared) | `index.tsx:23,278` |
| Header card wrapper | Hand-rolled (same pattern as D1) | `index.tsx:275-277` |
| Stat row (3 stats) | Hand-rolled inline array, no shared primitive | `index.tsx:290-307` |
| Filter bar | Hand-rolled — status pills + raw `<input>` search, NOT `SearchBar`/`FilterPill` | `index.tsx:310-364` |
| List | Hand-rolled `ProjectCard` grid (no shared primitive, but appropriate — card grid) | `index.tsx:144-228,403-412` |
| Loading | Hand-rolled 4× `Skeleton` card list | `index.tsx:367-372` |
| Empty | Hand-rolled dashed-border empty block | `index.tsx:373-396` |
| Status badge | Hand-rolled `STATUS_META` map; good: icon+label+color | `index.tsx:33-67` |
| Pagination | Hand-rolled numeric pager with chevrons | `index.tsx:417-494` |

**Drift vs other routes:** Only desktop route that hand-rolls the search input (D3 uses `SearchBar`); only route with a numeric pager.

### D3 — Desktop Employees (`pages/partner/EmployeesPage/index.tsx`, 597 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell bg | Hand-rolled radial gradient (mirrors D1/D2/D4) | `index.tsx:400` |
| Header | ✅ `PageHeader` (shared) | `index.tsx:5,406` |
| Header card wrapper | Hand-rolled (same pattern) | `index.tsx:403-405` |
| Stat strip | ✅ `InlineStatStrip` (shared) — BEST-IN-CLASS | `index.tsx:6,430` |
| Filter bar | ✅ `SearchBar` + `FilterPill` (shared) — BEST-IN-CLASS | `index.tsx:7-8,488,498,506,516` |
| List/Table | ✅ `ResponsiveTable` (shared ui/) — BEST-IN-CLASS | `index.tsx:4,548` |
| Loading | Hand-rolled `Skeleton` block (initial-load only) | `index.tsx:367-397` |
| Empty | Inline `emptyState` prop on `ResponsiveTable` (well-formed, icon+title+sub) | `index.tsx:561-578` |
| Status badge | `SCHEDULE_STYLES` map with `bg-sky-50/violet-50/emerald-50` ring pills (icon N/A — info badge, not status) | `index.tsx:44-54` |
| Side widget | `MissingBankDetailsSection` shared component | `index.tsx:535` |
| Initial-load guard | `isInitialLoad` early-return skeleton (smart UX) | `index.tsx:365-397` |

**Drift vs other routes:** **Reference route** — uses the shared primitive stack the other desktops should converge toward. No `bg-[#...]` literal hex (unlike M2/M3/M4).

### D4 — Desktop Timesheets (`pages/partner/TimesheetsPage/index.tsx`, 597 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell bg | Hand-rolled radial gradient (mirrors others) | `index.tsx:393` |
| Header | ⚠️ Hand-rolled — custom `<h1>` + icon chip + badge (NOT `PageHeader`) | `index.tsx:396-433` |
| Stat row | Hand-rolled local `TimesheetStatsRow` component (watermark style, mirrors D1's `StatTile`) | `index.tsx:46-175,438` |
| Filter bar | `TimesheetFilters` from `components/timesheet/` (shared with admin — DO NOT EDIT) | `index.tsx:4,538` |
| List/Table | `TimesheetListTable` / `TimesheetMobileList` (shared with admin — DO NOT EDIT) | `index.tsx:5-6,550` |
| Loading | Hand-rolled `Skeleton` page | `index.tsx:357-389` |
| Empty | (delegated to shared `TimesheetListTable`) | — |
| Status badge | (delegated to shared table) | — |
| Action bar | Hand-rolled: 5 outline buttons + 1 primary, in a card with bg-muted/25 label | `index.tsx:452-510` |

**Drift vs other routes:** Only desktop route NOT using `PageHeader`; the only one whose data table + filter live in a shared-with-admin `components/timesheet/` package (Phase 3 cannot restyle without regressing admin). The local `TimesheetStatsRow` watermark style duplicates D1's `StatTile` pattern — both are hand-rolled.

### M1 — Mobile Dashboard (`pages/mobile/partner/DashboardPage/index.tsx`, 368 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell | ✅ `MobilePageShell` + `MobileSurface` | `index.tsx:20,280,289,324` |
| Header | ✅ `MobilePageHeader` (shared) | `index.tsx:19,281` |
| Stat block | ✅ `MobileOperationsPanel` + `MobileTaskList` (shared, BEST-IN-CLASS mobile pattern) | `index.tsx:21-27,306,318` |
| Loading | ✅ `Skeleton` (inline in `MobileSurface`) | `index.tsx:293-304,333-345` |
| Empty | Hand-rolled inline block | `index.tsx:346-353` |
| Top-employees list | Hand-rolled local `TopEmployeeRow` (clean, uses `bg-gradient-to-r from-primary/40`) | `index.tsx:102-149,355-357` |
| Status badge | ✅ `Badge` from `ui/badge` paired with `UserCheck`/`UserX` icon | `index.tsx:126-133` |

**Drift:** **Reference mobile route** — full `shared/*` adoption. Uses semantic tokens (`primary`, `success`, `destructive`, `warning`) — no literal hex.

### M2 — Mobile Projects (`pages/mobile/partner/ProjectsPage/index.tsx`, 376 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell | ✅ `MobilePageShell` + `MobileSurface` | `index.tsx:4,141` |
| Header | ✅ `MobilePageHeader` | `index.tsx:3,143` |
| Search | ✅ `MobileSearchInput` | `index.tsx:2,199` |
| Stat strip | ⚠️ Hand-rolled with **literal hex `bg-[#E7EEF6]` / `bg-[#DDF7EC]` / `bg-[#EEE7FF]` / `bg-[#FFF0C2]`** and `border-[#D8E2EE]` | `index.tsx:164-187` |
| Filter | Hand-rolled `Sheet` bottom drawer with `Select`s (NOT `MobileFilterPill`) | `index.tsx:288-370` |
| List | `ProjectMobileList` (from `components/projects/`) | `index.tsx:21,255` |
| Loading | Hand-rolled skeleton | `index.tsx:118-138` |
| Empty | Hand-rolled inline block with `bg-[#E7EEF6]` literal hex | `index.tsx:258-272` |
| Active-filter chips | ✅ `Badge` from `ui/badge` | `index.tsx:225-243` |

**Drift:** only mobile route with literal hex `bg-[#…]` colors. The 4 stat-strip literals are 4 of the 56 pre-existing C2 baseline matches.

### M3 — Mobile Employees (`pages/mobile/partner/EmployeesPage/index.tsx`, 484 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell | ✅ `MobilePageShell` + `MobileSurface` | `index.tsx:4,184` |
| Header | ✅ `MobilePageHeader` | `index.tsx:3,186` |
| Search | ✅ `MobileSearchInput` | `index.tsx:2,269` |
| Stat strip | ⚠️ Hand-rolled with literal hex (mirrors M2: `bg-[#E7EEF6]`, `bg-[#DDF7EC]`, `bg-[#FFF0C2]`, `bg-[#EEE7FF]`, `border-[#D8E2EE]`) | `index.tsx:113-153,216-258` |
| Filter | Hand-rolled `Sheet` (mirrors M2) | `index.tsx:362-470` |
| List | ✅ `EmployeeMobileCard` + `EmployeeEmptyStates` + `InfiniteScrollContainer` | `index.tsx:22-23,337-358` |
| Loading | Hand-rolled skeleton | `index.tsx:155-181` |
| Active chips | ✅ `Badge` | `index.tsx:295-323` |
| Side widget | `MissingBankDetailsSection` | `index.tsx:335` |

**Drift:** duplicates M2's stat-strip pattern (same literal hexes). One of 2 routes that pass `onEmployeeClick` to `MissingBankDetailsSection` (the other is D3).

### M4 — Mobile Timesheets (`pages/mobile/partner/TimesheetsPage/index.tsx`, 282 lines)

| Concern | Treatment | Evidence |
|---|---|---|
| Shell | ✅ `MobilePageShell` + `MobileSurface` | `index.tsx:13,175,198,248,252` |
| Header | `TimesheetPageHeaderMobile` from `components/timesheet/mobile/` (shared with admin — DO NOT EDIT) | `index.tsx:4,199` |
| Stat | ✅ `GroupedStatCard` (shared, BEST-IN-CLASS) | `index.tsx:12,212,222` |
| List / Filters | `TimesheetDisplaySection` (shared with admin — DO NOT EDIT) | `index.tsx:5,253` |
| Loading | Hand-rolled `Skeleton` page | `index.tsx:174-194` |
| Entry mode | Full-page `MobileTimesheetEntry` when `?modal=timesheet_entry` | `index.tsx:21,156-168` |

**Drift:** **Reference mobile route for stats** — uses `GroupedStatCard`. Zero literal hexes.

---

## 2. Shared primitive adoption map

Phase 3's primary directive. Cells: **K** = route already uses it (Keep) · **M** = route must migrate to it · **N** = N/A for this route.

| Primitive | D1 | D2 | D3 | D4 | M1 | M2 | M3 | M4 |
|---|---|---|---|---|---|---|---|---|
| `PageHeader` | K | K | K | **M** (header is hand-rolled `<h1>`) | N | N | N | N |
| `MobilePageHeader` | N | N | N | N | K | K | K | N (uses `TimesheetPageHeaderMobile` — shared, OOS) |
| `InlineStatStrip` | N | **M** (hand-rolled 3-stat row) | K | **M** (`TimesheetStatsRow` is local) | N | N | N | N |
| `GroupedStatCard` | N | N | N | N | N | N | N | K |
| `MobileOperationsPanel` | N | N | N | N | K | N | N | N |
| `MobileStatStrip` *(see §5 gap analysis)* | N | N | N | N | N | **M** | **M** | N |
| `SearchBar` | N | **M** (raw `<input>`) | K | N (uses `TimesheetFilters`) | N | N | N | N |
| `FilterPill` | N | **M** (raw status pills) | K | N | N | N | N | N |
| `MobileSearchInput` | N | N | N | N | N | K | K | N |
| `MobileFilterPill` | N | N | N | N | N | **M** (`Sheet`+`Select`) | **M** (`Sheet`+`Select`) | N |
| `EmptyState` | **M** | **M** | K-ish (uses `ResponsiveTable.emptyState` — fine) | N (delegated to shared table) | **M** | **M** | **M** (uses `EmployeeEmptyStates` — Keep) | N |
| `MobilePageShell` + `MobileSurface` | N | N | N | N | K | K | K | K |
| `MobilePagination` | N | **M** (hand-rolled pager on desktop would move to mobile; desktop pagination stays) | N (uses `ResponsiveTable` pagination) | N | N | N (infinite scroll) | N (infinite scroll) | N |

**Summary of M (migrate) cells — Phase 3 work list:**
- D4 → adopt `PageHeader` for the timesheet header card (currently `<h1>` at `index.tsx:405`).
- D2 → swap raw search `<input>` → `SearchBar`; swap raw status pills → `FilterPill`.
- D1 + D4 → consider extracting the shared watermark `StatTile`/`TimesheetStatsRow` pattern into a single shared component (see §5).
- D1, D2, M1, M2, M3 → swap hand-rolled empties → `EmptyState` (where shape permits).
- M2 + M3 → swap hand-rolled stat strip → a shared mobile stat-strip primitive (see §5); swap `Sheet`+`Select` filter drawer → `MobileFilterPill`-based or keep as-is (see §5 trade-off).

---

## 3. Dead-code manifest

Three greps per file (v2.1 H4 anchored on `components/partner-{folder}`, M1 broad scan, Any-ref).

### `components/partner-employees/` — 7 files, ALL orphans ✅ safe to delete

| File | Static | Dynamic | Any-ref | Verdict |
|---|---|---|---|---|
| `PartnerEmployeesFilters.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerEmployeesHeader.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerEmployeesList.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerEmployeesStats.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerEmployeesSummaryStats.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerEmployeesTable.tsx` | 0 | 0 | 0 | DELETE |
| `mobile/PartnerEmployeesListMobile.tsx` | 0 | 0 | 0 | DELETE |

> **H4 note:** the LIVE `hooks/partner-employees/usePartnerEmployeesData.ts` is a different directory (`hooks/`, not `components/`) and is consumed by D3+M3. NOT affected by this deletion.

### `components/partner-projects/` — 5 files, ALL orphans ✅ safe to delete

| File | Static | Dynamic | Any-ref | Verdict |
|---|---|---|---|---|
| `PartnerProjectsFilters.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerProjectsHeader.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerProjectsList.tsx` | 0 | 0 | text-only false positive (see below) | DELETE |
| `PartnerProjectsStats.tsx` | 0 | 0 | 0 | DELETE |
| `mobile/PartnerProjectsListMobile.tsx` | 0 | 0 | text-only false positive | DELETE |

> **False-positive audit:** the Any-ref matches on `PartnerProjectsList` were the literal string `PartnerProjectsList` appearing in source text of the sibling file `components/partner-dashboard/PartnerProjectsList.tsx` (and vice versa). `grep -n "partner-projects" src/components/partner-dashboard/PartnerProjectsList.tsx` returns **0 matches** — i.e. no import relationship. Both are dead.

### `components/partner-timesheet/` — 6 files, ALL orphans ✅ safe to delete

| File | Static | Dynamic | Any-ref | Verdict |
|---|---|---|---|---|
| `PartnerTimesheetFilters.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerTimesheetHeader.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerTimesheetList.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerTimesheetStats.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerTimesheetTable.tsx` | 0 | 0 | 0 | DELETE |
| `mobile/PartnerTimesheetListMobile.tsx` | 0 | 0 | 0 | DELETE |

### `components/partner-dashboard/` — 5 files, **3 orphans + 2 LIVE** ⚠️ partial delete

| File | Static | Dynamic | Any-ref | Verdict |
|---|---|---|---|---|
| `PartnerEmployeeListSheet.tsx` | 0 | 2 (D1+M1) | 2 | **KEEP** — live |
| `PartnerWorkforceOverviewCard.tsx` | 0 | 1 (D1) | 1 | **KEEP** — live |
| `PartnerDashboardHeader.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerProjectCard.tsx` | 0 | 0 | 0 | DELETE |
| `PartnerProjectsList.tsx` | 0 | 0 | text-only false positive | DELETE |

> **Plan delta:** the plan body lists only `PartnerEmployeeListSheet` and `PartnerWorkforceOverviewCard` as live (Corrected Codebase Survey §"Live `partner-dashboard/` components"). Audit confirms exactly that. The other 3 in `partner-dashboard/` are also orphans and **should be deleted in Phase 3 alongside the 3 main folders** — this is a Phase 3 scope addendum (the plan didn't explicitly enumerate them but the deletion criterion is the same: zero importers).

**String-based / modal-registry scan (v2.1 M1):** 4 files contain the literal strings `partner-employees`/`partner-projects`/`partner-timesheet`/`partner-dashboard` outside `components/partner-*`:
- `contexts/CommandPaletteContext.tsx` → uses as nav-id strings (`nav-partner-projects`), not imports.
- `config/actions.ts` → action-id strings.
- `config/partner-dashboard/partner-dashboard-utils.ts` → type-import from sibling `./partner-dashboard-types` (different dir, not components/).
- `lib/queryKeys/index.ts` → query-key string.

**All 4 are unrelated to the dead component folders.** Safe to delete the 21 dead component files listed above without touching these.

**Total deletion manifest for Phase 3:** 7 + 5 + 6 + 3 = **21 files across 4 folders**. Plus optional `AGENTS.md` reference cleanup (Phase 3 Step 7 per M2).

---

## 4. Cross-surface preservation map (blast radius)

For every shared primitive partner consumes, list non-partner consumers so Phase 3 knows what could regress.

| Primitive | Non-partner consumers (verified by grep) | Phase 3 risk |
|---|---|---|
| `PageHeader` | `users/UserPageHeader.tsx`, `admin/SystemHealthPage`, `admin/AdvancePaymentsPage` | LOW — name-only import; prop additions are additive. Visual edits to the primitive itself are NOT in scope (Phase 3 only changes what *partner routes* pass in). |
| `MobilePageHeader` | 14 mobile admin files + `transaction/mobile/TransactionPageHeaderMobile` + `timesheet/mobile/TimesheetPageHeaderMobile` | LOW — same rationale. |
| `InlineStatStrip` | `transaction/TransactionSummaryCard`, `ledger/LedgerSummaryCard`, `advance-payment/TransactionSummaryStrip`, `admin-dashboard/CheckInHealthStrip`, `admin/DashboardPage`, `admin/EmployeesPage`, `admin/LoansPage` | **MEDIUM** — D2 and D4 will START using it. No primitive change needed. |
| `GroupedStatCard` | `sheets/ProjectDetailsSheet`, `advance-payment/AdvancePaymentStatsStrip`, `projects/ProjectStats`, `employees/details/.../EmployeeStatistics`, `hooks/admin-dashboard/useDashboardStats`, `mobile/admin/DashboardPage` | LOW. |
| `SearchBar` | `project-employees/ProjectEmployeesList`, `admin/AdvancePaymentsPage` (×2) | LOW. |
| `FilterPill` | `project-employees/ProjectEmployeesList`, `wallet/WalletTransactionsList` (+ indirect via `transaction/TransactionFilters`, `timesheet/TimesheetFilters`, etc.) | LOW. |
| `MobileSearchInput` | 8 mobile admin files + `transaction/mobile/TransactionFiltersMobile` | LOW. |
| `MissingBankDetailsSection` | `admin/EmployeesPage`, `admin/TimesheetPage`, `mobile/admin/EmployeesPage`, `mobile/admin/TimesheetPage` + the 4 partner pages | **MEDIUM** — D3 and M3 pass `onEmployeeClick`; admin does NOT. Prop is optional. Confirm in Phase 4 that admin rendering is unchanged. |
| `MobileOperationsPanel` | `mobile/admin/DashboardPage` | LOW. |
| `MobilePageShell` / `MobileSurface` | 6 mobile admin pages | LOW. |
| `ResponsiveTable` | `admin/EmployeesPage` (likely) | Verify in Phase 3; D3 is a heavy user. |

**Conclusion:** Phase 3 should NOT modify any shared primitive's exported API. All Phase 3 changes are (a) swap hand-rolled markup → primitive in the 4 desktop files, (b) delete dead code, (c) optionally create new shared primitives identified in §5 below.

---

## 5. Gap analysis

**Question:** does Phase 3 need any NEW `components/shared/*` primitive that doesn't exist today?

**Answer:** Two candidate gaps, both **OPTIONAL** (Phase 3 may ship without them; the alternative is to leave the hand-rolled equivalents in place).

### Gap A — Shared desktop watermark stat tile (`StatTile`)

**Pattern appears in:** D1's `StatTile` (`pages/partner/DashboardPage/index.tsx:125-194`) and D4's `TimesheetStatsRow` (`pages/partner/TimesheetsPage/index.tsx:46-175`). Both implement the same shape: large faint icon watermark on the right, label row top-left with small icon prefix, big number, optional trend/sublabel. D4 adds an `active` state for filter-toggle.

**Decision for Phase 3:** extract to `components/shared/StatTile.tsx` with props `{ label, value, icon, color, sublabel?, change?, active?, onClick? }`, OR leave as-is if extraction proves to be over-DRY (the two callers diverge in details: D1 is a button, D4 is sometimes a button sometimes static).

**Tailkit reference informing the design:** `a-c-statistics-02` (icon-left, number-right, three-card grid) and `a-c-statistics-03` (number-left, trend-chip + action-link bottom). The watermark style is unique to this project — Tailkit doesn't have an exact match, so the extraction should preserve the existing watermark art.

### Gap B — Shared mobile stat strip (`MobileStatStrip`)

**Pattern appears in:** M2 (`pages/mobile/partner/ProjectsPage/index.tsx:164-187`) and M3 (`pages/mobile/partner/EmployeesPage/index.tsx:113-153`). Both render a horizontal-scroll row of small stat tiles with **literal hex colors** (`bg-[#E7EEF6]`, `bg-[#DDF7EC]`, `bg-[#EEE7FF]`, `bg-[#FFF0C2]`, `border-[#D8E2EE]`). M3 adds an `active`-state for filter-toggle.

**Decision for Phase 3:** extract to `components/shared/MobileStatStrip.tsx` AND replace the 5 literal hexes with semantic tokens (`bg-primary/10`, `bg-success/10`, `bg-warning/10`, etc.). This **simultaneously** resolves 8 of the 56 pre-existing literal-class matches (C2 baseline) without changing the visual identity meaningfully.

> **Tailkit literal-class ban interaction (plan C2):** the ban is **DELTA-ONLY** — the 56 pre-existing matches are out of scope. Extracting M2/M3's stat strips to a shared primitive that uses semantic tokens is a *removal* of literals, not an *addition*. This is a safe improvement; Phase 4's `git diff main ... | grep "^\+.*literal"` gate stays green.

**Other gaps considered and rejected:**
- `MobileFilterPill` exists but doesn't fit the bottom-drawer filter pattern M2/M3 use (Select-based Sheet). The drawer pattern is closer to a `MobileFilterDrawer` primitive that doesn't exist. **Recommendation: leave M2/M3's drawer as-is** — extracting it would create a partner-specific primitive used by only 2 callers with divergent contents (project filter sheet vs. employee filter sheet). YAGNI.

**Conclusion:** Phase 3's only new-primitive candidates are StatTile (Gap A) and MobileStatStrip (Gap B). Both are optional; if Phase 3 time-boxes, ship without them and revisit.

---

## 6. Tailkit reference selections

Documented for Phase 3 visual-fix justification. **Note:** Tailkit is visual reference only; classes are translated to project tokens (`bg-card`, `text-muted-foreground`, `hsl(var(--success))`).

| Drift category | Tailkit ref | Identifier | Where applied (Phase 3) |
|---|---|---|---|
| Desktop stat card with icon | Simple with Icons | `a-c-statistics-02` | If Gap A `StatTile` extraction ships — informs icon-left / number-right layout (NOT adopted verbatim; the watermark pattern stays). |
| Desktop stat card with trend + action | Simple with Action | `a-c-statistics-03` | Informs D2's hand-rolled 3-stat row IF migration to `InlineStatStrip` proves incompatible (InlineStatStrip doesn't have trend chips — Phase 3 should not invent them). |
| Empty state with action | With Actions | `a-c-empty-states-03` | Informs D2 + D1 empty blocks. Pattern: dashed border, centered icon, headline, message, primary action button. Translates to existing `EmptyState` primitive props. |
| Empty state with placeholders | With Placeholders | `a-c-empty-states-05` | NOT adopted — partner pages don't have card-grid layouts needing placeholder cards. Documented as considered-and-rejected. |
| Table in card | In Card | `a-c-tables-09` | Reference for the wrapper around `ResponsiveTable` in D3 (`index.tsx:537`) — already implements the "card with header strip" pattern. No Phase 3 change needed. |

**Translation rules for Phase 3:**
- `bg-white` → `bg-card`
- `text-secondary-500` / `text-secondary-600` → `text-muted-foreground`
- `bg-emerald-50 text-emerald-600` → `bg-success/10 text-success` (or keep literal — pre-existing, out of scope)
- `dark:` variants → drop (project uses CSS variables that flip via `.dark` class automatically)
- `rounded-lg shadow-xs` → `rounded-xl shadow-soft` (project's idiom)

---

## 7. Phase 3 work list (consolidated)

Ordered by blast-radius (lowest-risk first).

### 7.1 Dead-code deletion (zero risk — already confirmed orphaned)

Delete 21 files across 4 folders:
- `frontend/src/components/partner-employees/` (7 files, full folder)
- `frontend/src/components/partner-projects/` (5 files, full folder)
- `frontend/src/components/partner-timesheet/` (6 files, full folder)
- `frontend/src/components/partner-dashboard/` — delete ONLY `{PartnerDashboardHeader, PartnerProjectCard, PartnerProjectsList}.tsx` (3 files); KEEP `{PartnerEmployeeListSheet, PartnerWorkforceOverviewCard}.tsx`.

Post-deletion gate: `cd frontend && tsc --noEmit && pnpm build` must pass. Also re-run `grep -rln "from \"@/components/partner-"` to confirm zero remaining imports.

### 7.2 Desktop route convergence (medium effort, low risk)

| Route | Change | Why |
|---|---|---|
| D4 Timesheets | Wrap existing `<h1>` header content in `PageHeader` | Header consistency with D1/D2/D3 |
| D2 Projects | Replace raw `<input>` search with `SearchBar`; replace raw status pills with `FilterPill` | Filter consistency with D3 |
| D2 Projects | Optional: replace hand-rolled 3-stat row with `InlineStatStrip` | Stat consistency with D3 |
| D4 Timesheets | Optional: replace local `TimesheetStatsRow` with `InlineStatStrip` if shape permits (5 cells vs 4 — `InlineStatStrip` may need a 5th item; verify API) | Stat consistency with D3 |
| D1, D2 | Replace hand-rolled empty blocks with `EmptyState` (where shape permits) | Empty-state consistency |

### 7.3 Mobile route convergence (low effort, low risk)

| Route | Change | Why |
|---|---|---|
| M2 Projects, M3 Employees | If Gap B `MobileStatStrip` ships: extract + swap in both. Otherwise: replace 4 literal hexes with semantic tokens in-place (`bg-[#E7EEF6]` → `bg-primary/10`, etc.) | Visual token consistency |
| M1, M2, M3 | Replace hand-rolled empties with `EmptyState` (where shape permits) | Empty-state consistency |

### 7.4 New shared primitives (optional, time-boxed)

- **Gap A:** `components/shared/StatTile.tsx` — extract from D1+D4.
- **Gap B:** `components/shared/MobileStatStrip.tsx` — extract from M2+M3 with token-translation.

### 7.5 Cross-surface preservation (Phase 4 check, not Phase 3 work)

Confirm unchanged in Phase 4:
- `/admin/dashboard`, `/admin/employees`, `/admin/loans` (InlineStatStrip consumers)
- `/admin/system-health`, `/admin/advance-payments` (PageHeader consumers)
- `/admin/employees`, `/admin/timesheet`, mobile equivalents (MissingBankDetailsSection consumers)
- `/partner/timesheet/payment-history` AND `/admin/...payment-history` (BankTransferHistoryPageContent variants — OOS entirely)

---

## 8. Open verifications carried to Phase 3

1. **InlineStatStrip 5-cell fit.** D4's `TimesheetStatsRow` has 5 stat cells; D3's `InlineStatStrip` is shown with 4 items. Phase 3 Step (7.2 D4) must verify `InlineStatStrip` accepts a 5-item array before migrating; if not, KEEP D4's local component (it's well-built).
2. **D2 filter-pill status multi-select.** D2's current status filter is single-select (`'all' \| ProjectStatus`). M2's is multi-select (array). `FilterPill` is single-select. Phase 3 must not regress D2 to multi or M2 to single — verify before migrating D2.
3. **D1 watermark StatTile uniqueness.** D1's watermark pattern is distinctive and on-brand. If Gap A extraction homogenizes it, the dashboard may lose visual identity. Phase 3 should visually A/B before merging the extraction.
4. **`main.tsx` has no `<StrictMode>` wrapper** (per Phase 1 EC-4). The Phase 1 classList effect is idempotent so StrictMode-double-invoke isn't a runtime concern, but Phase 4 verification of double-mount behavior can't run at dev time — verify by code-read only.

---

## 9. Audit self-check

- [x] One row per in-scope file (8 rows in §1).
- [x] Every claim cites `file:line` evidence.
- [x] Dead-code manifest lists every file in the 4 folders with three independent greps per file (§3).
- [x] Adoption map complete: every shared primitive in use is mapped to current and target consumers (§2).
- [x] Cross-surface consumer map complete: for each shared primitive partner uses, non-partner consumers are listed (§4).
- [x] Tailkit reference selections documented with identifier + drift addressed (§6).
- [x] Gap analysis explicitly states whether Phase 3 needs any new primitive (§5 answer: two optional candidates, otherwise none).
