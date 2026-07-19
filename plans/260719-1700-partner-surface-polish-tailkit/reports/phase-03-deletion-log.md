---
phase: 3
title: Deletion Log + Audit Resolution
status: completed
generated: '2026-07-19'
---

# Phase 3 Deletion Log + Audit Resolution

## 1. Dead-code deletion (audit §3, 21 files)

Re-confirmed orphaned via 3 anchored greps immediately before deletion (Phase 3 Step 7 mandate). All 21 files showed `static=0`, `dynamic=0`, `any-ref=0` (or text-only false positive, see below).

### Deleted — `components/partner-employees/` (full folder, 7 files)
- `PartnerEmployeesFilters.tsx`
- `PartnerEmployeesHeader.tsx`
- `PartnerEmployeesList.tsx`
- `PartnerEmployeesStats.tsx`
- `PartnerEmployeesSummaryStats.tsx`
- `PartnerEmployeesTable.tsx`
- `mobile/PartnerEmployeesListMobile.tsx`

> Note: the LIVE `hooks/partner-employees/` directory (usePartnerEmployeesData.ts et al.) is unaffected — separate directory.

### Deleted — `components/partner-projects/` (full folder, 5 files)
- `PartnerProjectsFilters.tsx`
- `PartnerProjectsHeader.tsx`
- `PartnerProjectsList.tsx`  *(any-ref=1 was text-only false positive: the literal string "PartnerProjectsList" appearing in source text of `components/partner-dashboard/PartnerProjectsList.tsx`. `grep -n "partner-projects" components/partner-dashboard/PartnerProjectsList.tsx` returns 0 — no import relationship.)*
- `PartnerProjectsStats.tsx`
- `mobile/PartnerProjectsListMobile.tsx`

### Deleted — `components/partner-timesheet/` (full folder, 6 files)
- `PartnerTimesheetFilters.tsx`
- `PartnerTimesheetHeader.tsx`
- `PartnerTimesheetList.tsx`
- `PartnerTimesheetStats.tsx`
- `PartnerTimesheetTable.tsx`
- `mobile/PartnerTimesheetListMobile.tsx`

### Deleted — `components/partner-dashboard/` (3 of 5 files; 2 KEPT)
Deleted:
- `PartnerDashboardHeader.tsx`  *(zero importers)*
- `PartnerProjectCard.tsx`  *(zero importers)*
- `PartnerProjectsList.tsx`  *(zero importers)*

Kept (LIVE — imported by D1/M1):
- `PartnerEmployeeListSheet.tsx` — imported by `pages/partner/DashboardPage/index.tsx:21`, `pages/mobile/partner/DashboardPage/index.tsx:29`
- `PartnerWorkforceOverviewCard.tsx` — imported by `pages/partner/DashboardPage/index.tsx:22`

### Post-deletion verification
```
$ grep -rln 'from "@/components/partner-' frontend/src --include="*.tsx" --include="*.ts"
(empty — zero orphaned imports)
```

```
$ ls frontend/src/components/partner-dashboard/
PartnerEmployeeListSheet.tsx
PartnerWorkforceOverviewCard.tsx
```

## 2. AGENTS.md cleanup (v2.1 M2 fix)

Two AGENTS.md files referenced the deleted folders. Updated:

- **`frontend/src/components/AGENTS.md`** — removed 3 rows for `partner-employees/`, `partner-projects/`, `partner-timesheet/`; updated `partner-dashboard/` description from "header and project list" to accurate "employee list sheet, workforce overview card".
- **`frontend/src/pages/partner/AGENTS.md`** — removed 3 dependency lines for the dead folders; updated remaining deps to reflect actual imports (`components/shared/`, `components/ui/responsive-table`).

**NOT touched (out of scope — different directories):**
- `frontend/src/config/AGENTS.md` — references `config/partner-*` dirs which are LIVE (column configs, not the deleted components).
- `frontend/src/hooks/AGENTS.md` — references `hooks/partner-employees/` and `hooks/partner-timesheet/` which are LIVE.

## 3. Audit §7 work-list resolution

| Audit item | Status | Resolution |
|---|---|---|
| §7.1 Delete 21 dead-code files | ✅ Done | See §1 above |
| §7.1 AGENTS.md cleanup | ✅ Done | See §2 above |
| §7.2 D2 → SearchBar | ✅ Done | `pages/partner/ProjectsPage/index.tsx` raw `<input>` replaced with `<SearchBar>` primitive; matches D3 Employees pattern |
| §7.2 D2 → FilterPill | ✅ Done | `pages/partner/ProjectsPage/index.tsx` raw status-pill buttons replaced with single `<FilterPill>` (matches D3 pattern; preserves single-select semantics) |
| §7.2 D4 → PageHeader | ❌ **Skipped (justified)** | D4's existing header has 3 elements that don't compose in `PageHeader`: eyebrow badge ("Không gian chấm công"), icon-chip + h1 + description, AND a month pager (`handlePrevMonth`/`handleNextMonth`). `PageHeader` only takes `title/description/actions/children` — forcing it would lose the eyebrow + month-pager or require awkward `children` composition. D4's header is already well-built and on-brand. **Skipping is the conservative choice; convergence not worth the structural risk.** |
| §7.2 D4 → InlineStatStrip | ❌ **Skipped (justified)** | `InlineStatStrip` items have no per-item `icon` or color override (only `valueClassName`). D4's `TimesheetStatsRow` uses 5 cells with watermark icons + `active` state for filter-toggle + per-state iconText/watermark colors. Migration would lose all of this. **Skipping; `TimesheetStatsRow` is well-built.** |
| §7.2 D1/D2 empties → EmptyState | ❌ **Skipped (justified)** | D2's empty has multi-state copy (filtered vs unfiltered) + 2-button footer. `EmptyState` primitive is simpler (single title + description + single action). Migration would lose conditional messaging. **Skipping.** |
| §7.3 M2/M3 stat strip → semantic tokens | ✅ Done | 4 literal hexes per file (`bg-[#E7EEF6]` → `bg-primary/10`, `bg-[#DDF7EC]` → `bg-success/10`, `bg-[#EEE7FF]` → `bg-info/10`, `bg-[#FFF0C2]` → `bg-warning/10`); plus `border-[#D8E2EE]` → `border-border`, `bg-white` → `bg-card`, `bg-slate-300` → `bg-muted-foreground/30`, `border-slate-300` → `border-border`. Removes 16+ pre-existing literal-class matches from C2 baseline. |
| §7.4 Gap A: extract `StatTile` | ❌ **Skipped (YAGNI)** | Audit §5 already flagged this optional. D1's StatTile and D4's TimesheetStatsRow diverge in details (button vs static+active). Extraction would over-DRY. Revisit if a 3rd caller appears. |
| §7.4 Gap B: extract `MobileStatStrip` | ❌ **Skipped (YAGNI)** | M2 + M3 still diverge (M3 has filter-toggle active state, M2 doesn't). The semantic-token migration in §7.3 resolved the literal-hex drift without needing extraction. |
| Step 5 partner.css additions | ✅ Decided | **No additions needed.** Phase 1 skeleton suffices: convergence used shared primitives + tokens, no partner-specific CSS rules required. Per Phase 3 Step 5: "If no rules were needed … `partner.css` may stay near-empty — that is a valid outcome." |

## 4. Files modified in Phase 3

- **Deleted:** 21 files across `components/partner-{employees,projects,timesheet}/` (full folders) + 3 files from `components/partner-dashboard/`
- **Modified:**
  - `frontend/src/pages/partner/ProjectsPage/index.tsx` — SearchBar + FilterPill adoption; removed unused `Search` import
  - `frontend/src/pages/mobile/partner/ProjectsPage/index.tsx` — stat strip + filter button + sheet border hex → semantic tokens
  - `frontend/src/pages/mobile/partner/EmployeesPage/index.tsx` — stat strip + active state + filter button + sheet border + drag-handle + Export button hex/slate → semantic tokens
  - `frontend/src/components/AGENTS.md` — removed dead-folder references
  - `frontend/src/pages/partner/AGENTS.md` — updated dependency list
- **Untouched (confirmed):**
  - `frontend/src/components/payroll/BankTransferHistoryPageContent.tsx` — byte-unchanged (v2.1 H3)
  - `frontend/src/App.tsx` — byte-unchanged (v2.1 H2 — no mobile Payment History wiring)
  - `frontend/src/components/shared/*` — no API changes
  - `frontend/src/components/PartnerSidebar.tsx`, `MobileBottomNav.tsx` — no drift found by audit, no edits
  - `frontend/src/styles/partner.css` — Phase 1 skeleton unchanged (no new rules needed)

## 5. Carry-forward to Phase 4

- Verify Tailkit literal delta grep returns zero NEW additions (§2 of phase-04).
- Verify `tsc --noEmit` and `pnpm build` pass after the deletions (§1 of phase-04).
- Verify `BankTransferHistoryPageContent.tsx` byte-unchanged via `git diff main` (§6 of phase-04).
- Visual QA at 320/390/768/1440 on the 4 in-scope routes (manual; outside this codebase session's tooling).
