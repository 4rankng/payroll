---
phase: 2
title: "Audit and Primitives Parity"
status: pending
priority: P2
dependencies: [1]
---

# Phase 2: Audit and Primitives Parity

## Overview

Per-route audit of the four in-scope partner surfaces (desktop + mobile = 8 files). For each route: which `components/shared/*` primitives are used, which patterns drift from the rest of the partner surface, which Tailkit reference patterns would improve consistency, and whether the dead-code `components/partner-{employees,projects,timesheet}/` folders are truly safe to delete. Output: an adoption map and a deletion manifest that Phase 3 executes.

## Requirements

- **Functional:**
  - A route-by-route audit table covering all 8 in-scope files.
  - A "shared primitive adoption map" listing which primitives each route should use after Phase 3.
  - A dead-code deletion manifest with two independent grep results per folder.
  - A list of any genuine gaps in `components/shared/*` that Phase 3 must fill (expected: few or none).
- **Non-functional:**
  - Every audit claim cites `file:line` evidence.
  - The deletion manifest is cross-checked against dynamic imports and string-based modal-registry references, not just static `from "..."` imports.

## Architecture

Audit is read-only — no code changes in this phase. The output is a Markdown report at `reports/phase-02-audit.md` that Phase 3 consumes as its work list.

## Related Code Files

- **Read-only audit targets (desktop):**
  - `frontend/src/pages/partner/DashboardPage/index.tsx`
  - `frontend/src/pages/partner/ProjectsPage/index.tsx`
  - `frontend/src/pages/partner/EmployeesPage/index.tsx`
  - `frontend/src/pages/partner/TimesheetsPage/index.tsx`
- **Read-only audit targets (mobile):**
  - `frontend/src/pages/mobile/partner/DashboardPage/index.tsx`
  - `frontend/src/pages/mobile/partner/ProjectsPage/index.tsx`
  - `frontend/src/pages/mobile/partner/EmployeesPage/index.tsx`
  - `frontend/src/pages/mobile/partner/TimesheetsPage/index.tsx`
- **Read-only references:**
  - `frontend/src/components/shared/*` (31 `.tsx` files plus `index.ts` barrel — the primitive inventory; v2.1 M3 fix: previous draft said "32 files")
  - `frontend/src/components/ui/responsive-table.tsx`, `data-table.tsx`, `mobile-table.tsx`
  - `frontend/src/components/timesheet/*` (shared with admin — out of scope to edit, but in scope to audit consistency)
  - Tailkit MCP snippets retrieved via `mcp__tailkit__get_component_code` (visual reference only)
- **Create:** `reports/phase-02-audit.md` (in this plan's `reports/` dir).

## Implementation Steps

1. **For each of the 8 in-scope files**, produce an audit row with:
   - **Styling source:** which `shared/*`, `ui/*`, or sibling components it imports (cite `file:line`).
   - **Header treatment:** uses `PageHeader` (desktop) / `MobilePageHeader` (mobile)? Or hand-rolled?
   - **Stat treatment:** uses `InlineStatStrip` / `GroupedStatCard` / `StatsCards`? Or hand-rolled?
   - **Filter treatment:** uses `FilterBar` / `FilterPill` / `MobileFilterPill` / `SearchBar` / `MobileSearchInput`? Or hand-rolled?
   - **List/Table treatment:** uses `ResponsiveTable` / `MobileCard` / sibling list component?
   - **Empty/Loading/Error treatment:** uses `EmptyState` / `Skeleton`? Or hand-rolled?
   - **Status badge treatment:** consistent icon+text+color, or color-only?
   - **Drift:** where does this route diverge from the other three?
2. **Build the "shared primitive adoption map."** A table: primitive → which routes currently use it → which routes should adopt it after Phase 3. Mark each adoption as `Migrate` (route must switch to the primitive), `Keep` (route already uses it), `N/A` (primitive doesn't apply).
3. **Pull 2-3 Tailkit reference snippets per drift category** via `mcp__tailkit__get_component_code`. Document which snippet informs each Phase 3 visual fix. Limit to: stat cards (`a-c-statistics-*`), empty states (`a-c-empty-states-*`), table card wrappers (`a-c-tables-*`).
4. **Dead-code audit (v2.1 H4 fix: greps must anchor on `components/`).** For each of `components/partner-employees/`, `partner-projects/`, `partner-timesheet/`:
   - Static import grep (anchored): `grep -rln "from \"@/components/partner-{folder}" --include="*.tsx" --include="*.ts" frontend/src | grep -v "components/partner-{folder}/"`
   - Dynamic import grep (anchored): `grep -rln "@/components/partner-{folder}/\|components/partner-{folder}/" --include="*.tsx" --include="*.ts" frontend/src | grep -v "components/partner-{folder}/"`
   - **Modal-registry grep (v2.1 M1 fix: scan ALL source, not just `constants/`+`lib/`).** Modal registry uses `import.meta.glob` (not string lookup), so the original grep was structurally vacuous. Instead scan `contexts/`, `config/`, `hooks/`, `lib/`, `constants/`, and any other dir that might hold string-based component references:
     ```
     grep -rln "partner-{folder}" frontend/src --include="*.tsx" --include="*.ts" | grep -v "components/partner-{folder}/"
     ```
     This will catch `contexts/CommandPaletteContext.tsx`, `config/actions.ts`, etc. (per v2 red-team Finding 6).
   - Record results PER FILE (not per folder — v2 Security #5 fix). A file is safe to delete only if all three greps return zero hits for its name. Folders are deleted only when every file in them passes; otherwise individual live files are kept and the discrepancy is documented.
5. **Gap analysis.** List any cases where Phase 3 needs a primitive that doesn't exist in `shared/*`. Expected answer: none, because the inventory is large — but verify honestly.
6. **Cross-surface preservation check (v2.1 M-ish fix: include barrel importers).** For each `shared/*` primitive that partner uses, grep its other consumers via BOTH the named-import path AND the barrel path:
   - Named: `grep -rln "from \"@/components/shared/{Name}\"" frontend/src`
   - Barrel: `grep -rln "from \"@/components/shared\".*\b{Name}\b\|from \"@/components/shared\"\s*$" frontend/src` (catches `import { Name } from "@/components/shared"` and `import * from "@/components/shared"` patterns).
   Record which non-partner routes also consume each primitive — Phase 3 visual changes would affect them.
7. **Write `reports/phase-02-audit.md`** with all of the above. This is Phase 3's work list.

## Success Criteria

- [ ] `reports/phase-02-audit.md` exists with one row per in-scope file (8 rows) and per-primitive adoption recommendations.
- [ ] Every audit claim cites `file:line` evidence.
- [ ] Dead-code manifest lists every file in `partner-{employees,projects,timesheet}/` with three independent grep results per file (static, dynamic, modal-registry).
- [ ] "Shared primitive adoption map" is complete: every primitive in `components/shared/*` mapped to its current and target partner-route consumers.
- [ ] Cross-surface consumer map is complete: for each shared primitive partner uses, non-partner consumers are listed so Phase 3 knows the blast radius of any edit.
- [ ] Tailkit reference selections are documented with identifier + which drift they address.
- [ ] Gap analysis explicitly states whether Phase 3 needs any new primitive (default expectation: none).

## Risk Assessment

- **Risk:** Audit misses a dynamic import or modal-registry string reference, leading to a Phase 3 deletion that breaks a deep-linked sheet.
  **Mitigation:** Step 4 mandates three independent greps per file. Any non-zero result blocks deletion of that file.
- **Risk:** Adoption map over-prescribes migration — proposes switching a route to a primitive when the current implementation is actually fine.
  **Mitigation:** Adoption recommendations are limited to genuine drift cases (inconsistent header/stat/filter/empty treatment across the four routes). If all four routes already agree, the map's answer is "Keep" everywhere and Phase 3 has little to do.
- **Risk:** Tailkit reference selection drifts into scope creep — pulling snippets for things partner doesn't need.
  **Mitigation:** Step 3 limits selections to three drift categories. Any addition requires explicit justification in the audit report.
