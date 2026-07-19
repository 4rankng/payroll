# Red Team — Assumption Destroyer Plan Review

**Reviewer:** Fact Checker (Standard tier)
**Plan:** 260719-1700-partner-surface-polish-tailkit
**Verification method:** grep/glob against `/Users/dev/Documents/projects/payroll/frontend/src` + root tooling files
**Stance:** Hostile — assumptions must be backed by codebase evidence or killed.

Method note: every finding cites at least one `path:line` from the actual codebase. Findings without codebase evidence were rejected before writing.

---

## Finding 1: `make api-test` is not a valid target from the repo root

- **Severity:** Critical
- **Location:** `plan.md` line 118 ("Acceptance Criteria"); `phase-04-visual-qa-and-verification.md` line 39 (Step 1) and line 85 (Success Criteria).
- **Flaw:** The plan gates shipping on `make api-test`, but the root `Makefile` has no such target.
- **Failure scenario:** Phase 4 runs `make api-test` from `/Users/dev/Documents/projects/payroll`. `make` aborts with `*** No rule to make target 'api-test'. Stop.` (verified — `make -n api-test` from repo root produced exactly that error). The gate fails before any actual test runs. Either the implementer silently skips the check (defeating the gate) or halts on a phantom target.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/Makefile` `.PHONY` line declares only `deploy demo dev backup restore demo-db sandbox` — no `api-test`. The full target list (`grep -E "^[a-z][a-z-]*:" Makefile`) returns: `deploy, dev, sandbox, backup, restore, demo, demo-db, adminer`. No `api-test`.
  - `/Users/dev/Documents/projects/payroll/backend/Makefile:1` declares `api-test` and `:242` defines it (`go run ./tests/integration/`).
  - `/Users/dev/Documents/projects/payroll/AGENTS.md` tells agents to run `make api-test` from the repo root, which is itself broken — but the plan copied that instruction verbatim without verifying it.
  - `make -n api-test` from `/Users/dev/Documents/projects/payroll` (executed during this review) returns `make: *** No rule to make target 'api-test'. Stop.` Exit code is reported as 0 by shell wrapping but `make` itself errors.
- **Suggested fix:** Either (a) change every Phase 4 reference to `cd backend && make api-test`, or (b) add a delegating `api-test:` target to the root Makefile (`cd backend && make api-test`). Plan should specify which.

---

## Finding 2: AdminSidebar does NOT use `ct-menu` — the entire daisyUI-vs-shadcn sidebar decision is built on a false premise

- **Severity:** Critical
- **Location:** `plan.md` line 72 (Key Decision 1), line 80 (Key Decision 9); `phase-01-partner-design-foundation.md` lines 79–81 (Step 7, "Path A vs Path B"), line 100 (Success Criteria).
- **Flaw:** The plan repeatedly asserts "admin sidebar uses daisyUI `ct-menu` for its shell" and frames the entire sidebar decision as "do we add a partner daisyUI theme OR re-implement daisyUI's menu in shadcn." The premise is false: `AdminSidebar.tsx` uses zero `ct-*` classes. It is already built from `@/components/ui/sidebar` shadcn primitives.
- **Failure scenario:** Phase 1 implementer reads "partner sidebar must match admin, which uses daisyUI `ct-menu`" and either (a) joins the daisyUI `themeRoot` (Path B) — expanding blast radius for nothing — or (b) spends the Phase 1 design session choosing between Path A and Path B when there is no decision to make. The "validated Session 1" claim (Decision 9) rests on a factoid that does not exist in the code.
- **Evidence:**
  - `grep -n "ct-" /Users/dev/Documents/projects/payroll/frontend/src/components/AdminSidebar.tsx` returns 0 matches for `ct-menu`, `ct-sidebar`, `ct-btn`, `ct-card`, `ct-badge` — all zero.
  - `grep -rn "ct-menu" /Users/dev/Documents/payroll/.../src` matches ONLY `/Users/dev/Documents/projects/payroll/frontend/src/styles/admin-daisy.css:25,29,30` — i.e. CSS rules that select `.ct-menu` exist, but no AdminSidebar markup emits that class.
  - `AdminSidebar.tsx` lines 37–45 import `Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu, SidebarMenuItem, SidebarSeparator` from `@/components/ui/sidebar` — pure shadcn.
  - The 8 `ct-` occurrences the validation log attributes to `AdminSidebar.tsx` (plan.md line 80) cannot be reproduced; the actual file emits none.
- **Suggested fix:** Delete Path A vs Path B from Phase 1 Step 7. Replace with: "Refactor `PartnerSidebar.tsx` to mirror `AdminSidebar.tsx`'s shadcn-sidebar pattern using `--partner-*` tokens." Drop all references to a partner daisyUI theme and to extending `themeRoot`. The plan's "daisyUI is permitted only for the partner sidebar" clause (plan.md line 72) is also void.

---

## Finding 3: Most "to-be-migrated-then-deleted" partner components are already orphaned dead code

- **Severity:** Critical
- **Location:** `phase-03-partner-route-migration.md` lines 49–58 (Related Code Files), lines 92–96 (Step 9 deletion), line 104 (Success Criteria).
- **Flaw:** Phase 3 lists ~19 per-page partner components (`PartnerEmployeesStats`, `PartnerEmployeesSummaryStats`, `PartnerEmployeesTable`, `PartnerEmployeesList`, `PartnerProjectsStats`, `PartnerProjectsList`, `PartnerProjectsHeader`, `PartnerProjectsFilters`, `PartnerTimesheetList`, etc.) as currently consumed by partner routes and slated for "consolidate → delete." Grep shows none of them are imported by any partner route page.
- **Failure scenario:** The Phase 3 migration template (Step 1–7) says "Replace the per-page `*Stats.tsx` with `<PartnerStatsGrid>`", "Replace the table wrapper markup with `<PartnerDataTable>`", etc. But the partner routes already use `shared/PageHeader`, `shared/InlineStatStrip`, `shared/SearchBar`, `shared/FilterPill`, and `ui/responsive-table` directly. There is nothing to replace. Implementer either edits files that aren't rendered (wasted work + confusing Phase 4 report) or assumes the plan is wrong about Phase 3 scope and stalls.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/frontend/src/pages/partner/EmployeesPage/index.tsx` imports `ResponsiveTable`, `PageHeader`, `InlineStatStrip`, `SearchBar`, `FilterPill` from `@/components/{ui,shared}/*`. It imports ZERO files from `@/components/partner-employees/*`.
  - `grep -rln "PartnerEmployeesStats\|PartnerEmployeesSummaryStats\|PartnerEmployeesTable" /Users/dev/Documents/projects/payroll/frontend/src` returns ONLY the definition files themselves — no consumer.
  - `grep -rln "PartnerProjectsList\|PartnerTimesheetList" /Users/dev/.../src` returns only the definition files and their mobile sibling — no page imports them.
  - Only `partner-dashboard/PartnerWorkforceOverviewCard.tsx`, `PartnerEmployeeListSheet.tsx`, `PartnerProjectCard.tsx`, `PartnerProjectsList.tsx` are actually imported — and only by `pages/partner/DashboardPage/index.tsx` and `pages/mobile/partner/DashboardPage/index.tsx` (the dashboard surface).
- **Suggested fix:** Re-scope Phase 3 to (a) what the partner pages actually render (Dashboard only — the only place using bespoke `partner-*` components) and (b) migration of `shared/PageHeader`/`InlineStatStrip`/`FilterPill` to the new `partner-ui/` primitives, which is the real work. Add a separate "delete orphans" sub-task with the actual orphan list (not a deferral to "if it becomes a no-op wrapper"). The current plan will produce a misleading Phase 4 deletion log.

---

## Finding 4: `PartnerDataTable` is redundant — `ui/data-table.tsx`, `ui/mobile-table.tsx`, `ui/responsive-table.tsx` already exist and partner uses them

- **Severity:** High
- **Location:** `phase-02-shared-partner-primitives.md` line 21 (PartnerDataTable spec), line 64 (Step 5); `phase-03-partner-route-migration.md` line 36 (template step 4), line 101 (Success Criteria).
- **Flaw:** The plan proposes a new `PartnerDataTable` that wraps `ui/table.tsx` primitives "at the shell level only" and accepts "TanStack Table instance or children." Three existing primitives already do this. The plan claims primitives are composed from `ui/card.tsx` + `ui/table.tsx` only — overlooking that `ui/responsive-table.tsx` already composes desktop DataTable + mobile MobileTable with TanStack column defs, sorting, pagination, search.
- **Failure scenario:** Phase 3 says "Move the column definitions and TanStack Table instance unchanged" into `PartnerDataTable`. But `EmployeesPage/index.tsx` already uses `ResponsiveTable` with column defs defined inline in the route. Moving them into a thinner `PartnerDataTable` wrapper around `ResponsiveTable` adds an abstraction layer with zero capability gain, then forces every route to swap imports. The "behavior-preserving" claim becomes hard to verify because the wrapper hides the table instance.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/frontend/src/components/ui/data-table.tsx:10` imports `Column, ColumnDef, flexRender, getCoreRowModel, useReactTable, SortingState, OnChangeFn, Row` from `@tanstack/react-table`. Full TanStack table with sorting + pagination + column sizing.
  - `/Users/dev/Documents/projects/payroll/frontend/src/components/ui/mobile-table.tsx:1` (Accordion-based mobile variant with RowAction, MobileField generics).
  - `/Users/dev/Documents/projects/payroll/frontend/src/components/ui/responsive-table.tsx:5` composes both via `useMediaQuery`, accepting `columns`, `mobileFields`, `rowTitle`, `rowActions`, `onRowClick`, `searchKey`.
  - `pages/partner/EmployeesPage/index.tsx:4` already imports `ResponsiveTable` and renders it.
  - The plan's phase-02 Related Code Files (lines 54–55) lists `card.tsx, table.tsx, badge.tsx, skeleton.tsx, button.tsx` as references but omits `data-table.tsx`, `mobile-table.tsx`, `responsive-table.tsx` — the actual table abstractions.
- **Suggested fix:** Either drop `PartnerDataTable` from Phase 2 (use `ResponsiveTable` directly with a partner-token class wrapper), or change the spec to "PartnerDataTable is a thin presentation card wrapper that accepts `<ResponsiveTable>` as children — does NOT accept a TanStack instance." The current "TanStack Table instance or children" API is incoherent with the existing `ResponsiveTable` API.

---

## Finding 5: `PartnerStatsCard` API parity claim with `PremiumStatStrip` is overstated — `PremiumStatStrip` is a multi-item strip, not a single card

- **Severity:** High
- **Location:** `plan.md` lines 74, 186 (Key Decision 3, Confirmed Decisions); `phase-02-shared-partner-primitives.md` lines 54, 62 (Step 3), line 79 (Success Criteria).
- **Flaw:** Plan repeatedly asserts the proposed `PartnerStatsCard` API is "cloned from `PremiumStatStrip`" and that parity is checkable. But `PremiumStatStrip` is a strip component that takes `items: PremiumStatItem[]` — it is not a single stat card. The "clone the API" framing produces a false sense of proven-ness; the actual mapping is "extract a single item from the strip into its own component," which is new design.
- **Failure scenario:** Phase 4 Success Criteria demands "PartnerStatsCard API parity with PremiumStatStrip — same label/value/unit/highlight/onClick/trend contract." But `PremiumStatStrip` does not export a single-card component or a `PartnerStatsCardProps` — the parity check is undefined. The validation log's "not speculative, proven" claim (plan.md line 170) overstates: proven-ness covers field names, not the single-vs-strip component shape.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/frontend/src/components/admin-dashboard/PremiumStatStrip.tsx:16–25` — `interface PremiumStatStripProps { items: PremiumStatItem[]; isLoading?: boolean; className?: string; wrap?: boolean; icon?: React.ReactNode; }`. There is no single-card prop interface.
  - `PremiumStatItem` (lines 5–14) has the fields the plan lists, but the *component* API is plural.
  - Plan's proposed `PartnerStatsCard` is singular — different component shape.
- **Suggested fix:** Reword the parity claim to: "`PartnerStatsCard` reuses the `PremiumStatItem` field shape (label/value/unit/highlight/onClick/trend); the component wrapper is new." Drop the "API parity" success criterion or restate it as "field-level parity on the item interface."

---

## Finding 6: `make api-test` requires a live backend — Phase 4 will fail in any environment without one running

- **Severity:** High
- **Location:** `phase-04-visual-qa-and-verification.md` line 39 (Step 1), line 85 (Success Criteria); `plan.md` line 118.
- **Flaw:** Even when invoked correctly as `cd backend && make api-test`, the target runs `go run ./tests/integration/` against a live backend. The plan presents this as a deterministic gate, but it depends on backend availability, DB state, and seed data. There is no instruction to stand up the backend first.
- **Failure scenario:** Implementer runs Phase 4 Step 1 in a frontend-only worktree. `make api-test` either hangs (waiting for backend) or fails on connection refused. They either skip the step (gate defeated) or spend hours debugging an environment issue that has nothing to do with a UI-only plan.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/backend/Makefile:242` — `api-test:` target body is `echo "..."` + `go run ./tests/integration/` (verified via `make -n api-test` in `backend/`).
  - `/Users/dev/Documents/projects/payroll/AGENTS.md` line 27 (root) — "Run `make api-test` after every feature change to catch regressions" — same assumption.
  - Plan claims "UI-only" scope (`plan.md` lines 5, 9, 53) but gates on a backend-integration check.
- **Suggested fix:** Either (a) drop `make api-test` from a UI-only plan's gate (it's a non-sequitur), or (b) explicitly require "ensure backend is running via `make dev` first" and document that flaky backend failures do not block this plan's sign-off (because nothing in this plan touches backend).

---

## Finding 7: Plan-wide assumption that TanStack Table is used on every partner table route is false

- **Severity:** High
- **Location:** `phase-02-shared-partner-primitives.md` line 64 (PartnerDataTable Step 5, "passed-in columns/rows via TanStack Table"); `phase-03-partner-route-migration.md` line 36 (template step 4, "TanStack Table instance unchanged"), line 101 (Success Criteria, "TanStack Table instances are unchanged"), line 115 (Risk: "TanStack Table column defs drift").
- **Flaw:** The plan assumes every partner table route runs TanStack Table. Only `EmployeesPage` does. Projects, Timesheets, and Payment History do not.
- **Failure scenario:** Phase 3 Step 4 is templated across all five routes ("Replace the table wrapper markup with `<PartnerDataTable>`. Move the column definitions and TanStack Table instance unchanged."). For Timesheets/Projects, there is no TanStack instance to move. Either the implementer writes nothing (template mismatch) or invents a TanStack layer to satisfy the template (scope creep + behavior risk on routes that currently use plain markup).
- **Evidence:**
  - `grep -rln "useReactTable\|ColumnDef" /Users/dev/Documents/projects/payroll/frontend/src/pages/partner /Users/dev/.../src/pages/mobile/partner` returns only `pages/partner/EmployeesPage/index.tsx`.
  - `pages/partner/DashboardPage/index.tsx`, `pages/partner/ProjectsPage/index.tsx`, `pages/partner/TimesheetsPage/index.tsx`, `pages/partner/PaymentHistoryPage/index.tsx` — none import TanStack.
  - `components/partner-timesheet/PartnerTimesheetTable.tsx:6` imports only `type { Row }` from `@tanstack/react-table` (a type-only import, no `useReactTable`).
- **Suggested fix:** Per-route Phase 3 steps should describe the actual table technology on that route. Drop "TanStack instance unchanged" from the global template and Success Criteria; restrict it to the Employees step.

---

## Finding 8: `partner.css` import location is mis-specified — plan says `src/styles/index.css` or `src/index.css`; `src/styles/index.css` does not exist

- **Severity:** Medium
- **Location:** `plan.md` line 218 (Touchpoints); `phase-01-partner-design-foundation.md` lines 51, 77 (Step 5); `phase-04-visual-qa-and-verification.md` line 89 (Success Criteria, "imported exactly once via `src/index.css`").
- **Flaw:** Multiple references disagree on whether `partner.css` is imported via `src/styles/index.css` or `src/index.css`. Only `src/index.css` exists. The Validation Log (plan.md line 150) quotes `index.css:1-5` as the aggregator and claims it imports `./styles/{variables,base,utilities,premium,admin-daisy}.css` — which is correct — but the plan body keeps offering `src/styles/index.css` as an alternative.
- **Failure scenario:** Implementer follows `phase-01...md` line 51 ("`src/styles/index.css` (or `main.tsx` stylesheet entry)"), creates `src/styles/index.css`, and adds an import that never runs because the aggregator is `src/index.css`. The new tokens silently don't load; Phase 1 Step 10 isolation check still passes (because the attribute is added at the JSX level) but visual styling never applies. Phase 4 catches it only if someone notices the styles are missing.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/frontend/src/index.css:1–5` contains the actual imports.
  - `/Users/dev/Documents/projects/payroll/frontend/src/styles/index.css` does NOT exist (`ls src/styles/` returns only `admin-daisy.css, base.css, premium.css, react-datepicker.css, utilities.css, variables.css`).
  - Plan's Phase 1 line 51 lists `src/styles/index.css (or main.tsx stylesheet entry)` as a valid target.
- **Suggested fix:** Pick one canonical location: `src/index.css` (the actual aggregator). Delete every mention of `src/styles/index.css` and `main.tsx` as alternative import sites.

---

## Finding 9: Tailwind v4 "translation tax" claim is asserted but never substantiated — and may be a non-issue

- **Severity:** Medium
- **Location:** `plan.md` line 33 (Overview), line 76 (Key Decision 5); `phase-01-partner-design-foundation.md` lines 60–75 (Step 3 mapping table).
- **Flaw:** The plan asserts "Tailkit ships Tailwind v4 idioms (`secondary-*`, `emerald-*`, `dark:` variants) that do not match this project's shadcn token system." But `bg-white`, `text-emerald-600`, `border-secondary-200`, and `dark:` variants are all legal Tailwind v3.4 syntax — `secondary-*` is a default Tailwind palette that ships with v3.4. The "v4 idiom" framing implies a v3/v4 incompatibility that may not exist for the listed classes. The plan never verifies what Tailkit actually emits (`@theme` directives? CSS-first config? `@custom-variant`?) versus what's just Tailwind v3 palette naming.
- **Failure scenario:** Two failure modes. (a) If Tailkit's actual v4-isms are `@theme`/CSS-first config, the mapping table is wildly underspecified — translating `bg-secondary-100` to a project token is trivial, but translating `@theme { --color-secondary-100: ... }` requires understanding Tailkit's full theme layer. (b) If Tailkit snippets are plain Tailwind utility classes, the "must translate" rule forces implementers to rewrite classes that would work as-is, slowing the project for no benefit and producing an inaccurate Phase 4 grep gate.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/frontend/package.json` confirms `"tailwindcss": "^3.4.17"` — not v4.
  - `/Users/dev/Documents/projects/payroll/frontend/tailwind.config.ts` uses v3-style `require("tailwindcss-animate")`, `require("@tailwindcss/typography")`, `require("daisyui")` — JS config, not v4 CSS-first.
  - `/Users/dev/Documents/projects/payroll/frontend/vite.config.ts` does not import `@tailwindcss/vite` (the v4 plugin).
  - `bg-emerald-600`, `text-secondary-500`, `dark:` are all valid v3.4 utilities. No codebase evidence shows Tailkit actually uses v4-only constructs.
  - The plan never cites a single concrete Tailkit snippet that fails on v3.4.
- **Suggested fix:** Phase 1 Step 1 (or earlier) should fetch ONE concrete Tailkit snippet via `mcp__tailkit__get_component_code` and verify what idioms it actually uses. Update the mapping table's framing based on that evidence. If Tailkit uses only v3-compatible utilities, the rule becomes "translate literal color utilities to project tokens for visual consistency," not "translate v4 idioms." If Tailkit does use `@theme`, expand the mapping-table task accordingly.

---

## Finding 10: `make api-test` and other root-Makefile references in AGENTS.md are broken, but the plan inherits them as gospel

- **Severity:** Medium
- **Location:** `plan.md` line 118; `phase-04-visual-qa-and-verification.md` line 39; both reference `AGENTS.md`'s `make api-test` rule as "non-negotiable."
- **Flaw:** The plan cites `AGENTS.md`'s rule (`make api-test` after every change) as authoritative ("the rule from `AGENTS.md` is non-negotiable", plan.md line 118). But that rule is itself broken at the repo root (Finding 1). The plan propagates an upstream doc bug instead of catching it.
- **Failure scenario:** Every consumer of this plan — implementer, future plans referencing the same convention — inherits the same broken instruction. The plan becomes a vector for the bug.
- **Evidence:**
  - `/Users/dev/Documents/projects/payroll/AGENTS.md` lines 27, 37 — instruct `make api-test` from the repo root.
  - `/Users/dev/Documents/projects/payroll/Makefile` — no such target.
- **Suggested fix:** Either (a) flag this upstream bug in the plan's Open Questions and require it be fixed before cook, or (b) explicitly scope the plan to `cd backend && make api-test` and note the root-Makefile gap.

---

## Finding 11: "Existing `*.test.tsx` under partner-* or pages/partner" — none exist

- **Severity:** Medium
- **Location:** `phase-03-partner-route-migration.md` line 108 (Success Criteria: "Focused Vitest specs (any existing `*.test.tsx` under `partner-*` or `pages/partner`) pass"); `phase-04-visual-qa-and-verification.md` line 38 (`pnpm test:run -- partner`), line 85 ("Focused Vitest partner specs pass").
- **Flaw:** The plan gates on "focused Vitest partner specs pass" but no such specs exist. `find` for `*.test.tsx` under `partner-*` or `pages/partner` returns zero files. Vitest is configured (`vitest.config.ts`) but every existing test file is for utils or non-partner components.
- **Failure scenario:** Phase 4 Step 1 runs `pnpm test:run -- partner`. Vitest matches zero files (or matches only files with "partner" in their path — also zero). Test command exits 0 trivially. Implementer records "tests pass" in the verification report — technically true, but the gate provides zero regression protection.
- **Evidence:**
  - `find /Users/dev/Documents/projects/payroll/frontend -name "*.test.tsx" -path "*partner*"` — empty result.
  - `find /Users/dev/Documents/projects/payroll/frontend/src -name "*.test.tsx"` returns 5 files: `ResponsivePage.test.tsx`, `AdvancePaymentHistoryCard.test.tsx`, `AdvancePaymentRequestForm.test.tsx`, `MobilePageHeader.test.tsx`, `TimesheetFilters.test.tsx`, `BankTransferHistoryPageContent.test.tsx`. None touch partner.
  - Playwright E2E: `find tests -name "*.spec.ts"` returned `auth.spec.ts`, `timesheet.spec.ts` — neither partner-scoped.
- **Suggested fix:** Either (a) add a Phase 2/3 task that creates at least one Vitest spec for `PartnerStatsCard` / `PartnerStatusBadge` (the only primitives with non-trivial behavior), or (b) drop the "Vitest partner specs pass" gate and explicitly note "no partner unit specs exist; E2E coverage is also absent — visual QA is the only regression gate."

---

Total findings: 11

