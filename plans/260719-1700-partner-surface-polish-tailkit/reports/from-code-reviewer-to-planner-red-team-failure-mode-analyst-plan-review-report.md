# Red-Team Plan Review — Failure Mode Analyst

**Target plan:** `260719-1700-partner-surface-polish-tailkit`
**Reviewer lens:** Murphy's Law — race conditions, data loss, cascading failures, recovery gaps, deployment risk, rollback holes, mobile ↔ desktop behavior drift, regression on shipped admin/employee surfaces.
**Verification tier:** Contract Verifier (Standard). Every claim below is grep-verified against `frontend/src/`.
**Verdict:** Plan is built on a fictional codebase. Its core migration template, file inventory, sidebar rationale, and mobile handling do not match what actually ships. Executing this plan as written will not polish the partner surface; it will leave it untouched and may regress admin/login.

---

## Finding 1: Mobile partner routes live in `pages/mobile/partner/*`, NOT in `partner-*/mobile/*` — plan migrates the wrong files

- **Severity:** Critical
- **Location:** Phase 3, "Related Code Files" and "Implementation Steps" 1–5; plan.md "Scope → In" (lines 41–46); plan.md "Risk Controls → Mobile regression" (line 125)
- **Flaw:** The plan asserts that mobile variants of partner pages live in `components/partner-employees/mobile/PartnerEmployeesListMobile.tsx`, `partner-projects/mobile/PartnerProjectsListMobile.tsx`, `partner-timesheet/mobile/PartnerTimesheetListMobile.tsx` and that migrating those files plus the desktop `index.tsx` covers "desktop + mobile in the same step." That is wrong. The actual mobile route variants live in `frontend/src/pages/mobile/partner/{DashboardPage,ProjectsPage,EmployeesPage,TimesheetsPage}/index.tsx` and are wired via `<ResponsivePage desktopComponent={...} mobileComponent={...}>` in `frontend/src/App.tsx:278–281`. Those mobile pages import an entirely separate component tree (`MobilePageHeader`, `MobilePageShell`, `MobileSurface` from `@/components/shared/`) — see `frontend/src/pages/mobile/partner/DashboardPage/index.tsx:19–20`, `frontend/src/pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx:5–7`.
- **Failure scenario:** Phase 3 implementer follows the migration template literally, edits `pages/partner/EmployeesPage/index.tsx` + `components/partner-employees/mobile/PartnerEmployeesListMobile.tsx`, ships Phase 4, and declares done. A partner user on a phone (`useIsMobile` true, `frontend/src/components/ResponsivePage.tsx:16`) hits the `pages/mobile/partner/EmployeesPage` route — which the plan never touched. Result: untouched UI ships to prod, and the Phase 4 acceptance criterion "All five partner routes pass desktop + tablet + mobile visual QA at 320 / 390 / 768 / 1440" passes by reviewer eyeball on desktop only, while mobile prod remains visually broken.
- **Evidence:**
  - Plan: `phase-03-partner-route-migration.md:53–57` lists only `components/partner-*/mobile/Partner*ListMobile.tsx` as mobile surfaces.
  - Plan: `phase-03-partner-route-migration.md:67` asserts "Header, filters, stats, list, and mobile list" — conflating the unused `partner-*/mobile/*` files with the route-mobile pages.
  - Plan: `plan.md:67` only mentions "the `useIsMobile` hook contract" — never `ResponsivePage`, never `pages/mobile/partner/*`.
  - Actual router: `frontend/src/App.tsx:278–281` uses `<ResponsivePage desktopComponent={PartnerDashboardPage} mobileComponent={PartnerDashboardMobile} />` and similarly for projects/employees/timesheet.
  - Actual mobile routes: `frontend/src/pages/mobile/partner/{DashboardPage,ProjectsPage,EmployeesPage,TimesheetsPage,TimesheetsPage/PaymentHistoryPage.tsx}` (full file listing confirmed).
- **Suggested fix:** Rewrite Phase 3 to operate on two parallel file trees per surface: `pages/partner/<Page>/index.tsx` (desktop) AND `pages/mobile/partner/<Page>/index.tsx` (mobile). Add an explicit Phase 3 sub-step per page to migrate the mobile route variant against the partner primitives, and update Phase 4 QA to enumerate both trees.

---

## Finding 2: Phase 3 deletes `PartnerEmployeesStats/Header/Filters/List/Table` etc., but those files are already dead code — the plan invents a migration that has nothing to migrate

- **Severity:** Critical
- **Location:** Phase 3, "Related Code Files → Delete" (line 49–58) and Step 9 (lines 92–96); plan.md Key Decision #8 (line 79) and Q3 answer (lines 172–175)
- **Flaw:** The plan's migration template assumes each route page imports per-page `Partner*Header/Filters/Stats/List/Table.tsx` components, replaces them with `partner-ui/` primitives, then deletes the originals. Grep shows NONE of these per-page components are imported anywhere outside their own file:
  - `PartnerEmployeesHeader` importers besides self: **0** (`grep -rln PartnerEmployeesHeader frontend/src` → only the file itself).
  - `PartnerEmployeesFilters`: **0 external importers**.
  - `PartnerEmployeesStats` / `PartnerEmployeesSummaryStats`: **0 external importers**.
  - `PartnerEmployeesList` / `PartnerEmployeesTable` / `PartnerEmployeesListMobile`: **0 external importers**.
  - Same for `PartnerProjects{Header,Filters,Stats,List}` and `PartnerTimesheet{Header,Filters,Stats,List,Table}` and all three `partner-*/mobile/*` files.
  - The actual `pages/partner/EmployeesPage/index.tsx` imports `PageHeader` from `@/components/shared/PageHeader:5`, `InlineStatStrip` from `@/components/shared/InlineStatStrip:6`, `SearchBar` from `@/components/shared/SearchBar:7`, `FilterPill` from `@/components/shared/FilterPill:8`, and `ResponsiveTable` from `@/components/ui/responsive-table:4` — none of the `partner-employees/*` files.
- **Failure scenario:** Two failure modes. (a) Implementer runs the migration, finds nothing to replace (the imports don't exist), and ships Phase 4 with a false " migrated" claim — prod UI is byte-identical to pre-plan, plan delivers zero visible value. (b) Implementer "restyles" the orphaned `partner-employees/PartnerEmployeesStats.tsx` files anyway, deletes them, and ships — also no visible effect, but the deletion log claims work product that did not happen. Either way the Phase 4 success criteria "[a]ll five partner routes consume `PartnerPageHeader`, `PartnerStatsGrid`, `PartnerStatsCard`" is unattainable against the current code without first rewiring `pages/partner/*/index.tsx` to actually use them.
- **Evidence:**
  - Plan: `phase-03-partner-route-migration.md:92–96` Step 9 lists `PartnerEmployeesStats.tsx, PartnerEmployeesSummaryStats.tsx, PartnerEmployeesHeader.tsx, *Filters.tsx` as targets for deletion-after-replacement.
  - Plan: `plan.md:172–175` Q3 answer "Delete the replaced ones."
  - Actual: `frontend/src/pages/partner/EmployeesPage/index.tsx:1–42` (imports enumerated above) — no `partner-employees/*` import.
  - Actual: `grep -rn PartnerEmployeesHeader frontend/src` returns only `frontend/src/components/partner-employees/PartnerEmployeesHeader.tsx` (self-declaration). Confirmed identical for all 6 employees, 4 projects, 5 timesheet per-page components.
- **Suggested fix:** Before any implementation, redo Phase 3 with a real audit table: for each `partner-*/*.tsx` file, list (file, importers count, currently-in-route yes/no). The plan will discover ~14 of the cited files are already dead code; either delete them as a separate pre-plan cleanup task, or pivot the migration target to `components/shared/{PageHeader,InlineStatStrip,SearchBar,FilterPill}` and `components/ui/responsive-table`.

---

## Finding 3: The "admin sidebar uses daisyUI `ct-menu`" rationale is false; Path B risk analysis is fabricated

- **Severity:** Critical
- **Location:** Phase 1, Step 7 (lines 79–89) and Risk Assessment (line 116); plan.md Key Decision #1 (line 72) and Key Decision #9 (line 80), Validation Log Q1/Q4 (lines 164, 181)
- **Flaw:** The plan repeatedly asserts "Admin sidebar uses daisyUI `ct-menu`" as the reason partner sidebar needs Path A/B analysis. Verified claim is false: `frontend/src/components/AdminSidebar.tsx` (529 LOC) contains ZERO `ct-menu` class usages. The 8 `ct-` substring matches the plan author counted (plan.md:148 "8 ct- occurrences") are coincidental substrings inside classnames like `select-none`, `cursor-pointer`, `object-contain`, etc. (`frontend/src/components/AdminSidebar.tsx:108, 212, 398, 401, 402, 517` — verified). The actual AdminSidebar uses shadcn `ui/sidebar` (`Sidebar, SidebarContent, SidebarHeader, SidebarMenu, SidebarMenuItem`, lines 26–37). `ct-menu` is defined in CSS at `frontend/src/styles/admin-daisy.css:25–30` but grep shows it is referenced in zero `.tsx` files anywhere in `frontend/src`.
- **Failure scenario:** Phase 1 implementer picks Path B (extend `themeRoot` to include `[data-partner-ui]` in `tailwind.config.ts:88`), believing admin needs this for `ct-menu` parity. Now daisyUI's `themeRoot: :where([data-admin-ui], [data-employee-ui], [data-partner-ui])` is expanded — every daisyUI class compiled into the bundle now also tries to resolve on partner nodes. Result: any future daisyUI misconfiguration in admin-daisy.css applies its style to partner surfaces, and any partner token overrides cascading through `[data-partner-ui]` can fight daisyUI specificity. The risk-benefit tradeoff that justified Path B (need `ct-menu` parity) does not exist; the implementer accepted real blast radius for an imaginary benefit. Worse: since admin sidebar is just shadcn, the "Path A: render admin pattern with shadcn" option is the ONLY honest option — there is no Path B use case.
- **Evidence:**
  - Plan: `phase-01-partner-design-foundation.md:79` "Admin sidebar (`AdminSidebar.tsx`, 529 LOC, 8 `ct-` occurrences) uses daisyUI `ct-menu` for its shell."
  - Plan: `plan.md:148` validation log "tailwind.config.ts:81-90 confirmed: `prefix: 'ct-'`..." — implies `ct-` is in active use.
  - Actual: `grep -rn "ct-menu" frontend/src --include="*.tsx" --include="*.ts"` returns 0 hits in component code; only `admin-daisy.css:25,29,30`.
  - Actual: `frontend/src/components/AdminSidebar.tsx:26–37` imports `useSidebar, Sidebar, SidebarContent, SidebarHeader, SidebarMenu, SidebarMenuItem, ...` from `@/components/ui/sidebar` — pure shadcn.
- **Suggested fix:** Strike Path B entirely. The partner sidebar should be polished directly against shadcn `ui/sidebar` matching AdminSidebar's actual pattern (which is what it already does). Update Phase 1 to remove the daisyUI themeRoot discussion from the partner sidebar scope. Leave `tailwind.config.ts:88` untouched. The validation log's Q4 ("sidebar follow admin sidebar design") becomes trivially "continue using shadcn `ui/sidebar` like AdminSidebar" — no new theme required.

---

## Finding 4: `data-partner-ui` root placement is wrong; `PartnerLayout` structure does not match plan's Architecture diagram

- **Severity:** High
- **Location:** Phase 1, Architecture diagram (lines 29–40) and Step 6 (line 78); Phase 1 Success Criteria (line 96)
- **Flaw:** Plan's Architecture diagram shows the layout as: `<div data-partner-ui>` → `<PartnerSidebar>`, `<SidebarToggle>`, `<main>` → `<SectionErrorBoundary>` → `<Outlet>`. Verified actual structure (`frontend/src/layouts/PartnerLayout.tsx:21–47`):
  - `<div className="relative flex h-dvh w-full group/layout">` (root, PartnerSidebar is sibling here)
  - `<PartnerSidebar />`, `<SidebarToggle />` (siblings at top level)
  - `<div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">` (extra wrapping div the plan ignores)
  - `<main ...>` containing ANOTHER wrapping `<div className="min-h-full mobile-main-content animate-page-enter max-w-[1320px] mx-auto">` before `<SectionErrorBoundary>`.
  - And `MobileBottomNav` and `NotificationFAB` are INSIDE `<SidebarProvider>` as siblings of `PartnerLayoutInner`, not outside it as the plan's diagram implies.
- **Failure scenario:** Phase 1 Step 6 says "Add `data-partner-ui` attribute to `PartnerLayoutInner`'s outer `<div className=\"relative flex h-dvh...\">`" and "Remove the inline `bg-[radial-gradient(...)]` on `<main>` and move that style into a `[data-partner-ui] main` rule." Two cascading problems: (a) If implementer scopes tokens on the outer `relative flex h-dvh` div, then `PartnerSidebar` (which is a child of `SidebarProvider`'s portal target) may NOT inherit because shadcn `<Sidebar>` uses portals for mobile drawer rendering — partner tokens silently fail on the mobile sidebar drawer. (b) The radial gradient lives on `<main>`, but plan says to move it into `[data-partner-ui] main`. If implementer follows Step 6, the `.mobile-main-content` inner div still wraps `<SectionErrorBoundary>`, so `[data-partner-ui] main` rules now compete with the pre-existing `.mobile-main-content` styling. (c) Step 10 isolation check `document.querySelector('[data-partner-ui]')` on `/partner/dashboard` only verifies presence, not that tokens actually reach all rendered subtrees.
- **Evidence:**
  - Plan: `phase-01-partner-design-foundation.md:30–40` diagram omits the wrapping `<div className="flex min-h-0 min-w-0 flex-1 ...">` and the `.mobile-main-content` inner wrapper.
  - Plan: `phase-01-partner-design-foundation.md:78` Step 6 says "Remove the inline `bg-[radial-gradient(...)]` on `<main>`" — but the radial-gradient is `<main>`'s OWN className (`PartnerLayout.tsx:27`), not an "inline" style; moving it to `[data-partner-ui] main` is fine but the plan does not address the inner `.mobile-main-content` div.
  - Actual: `frontend/src/layouts/PartnerLayout.tsx:21–47` (structure enumerated above).
- **Suggested fix:** Rewrite Phase 1 Step 6 with the real DOM tree. Specify whether `data-partner-ui` should be placed on the outer `relative flex h-dvh` div OR on a new wrapper that also encloses `<SidebarProvider>`'s portal output. Add an isolation check that exercises the mobile sidebar drawer open/close, not just the desktop sidebar.

---

## Finding 5: `PartnerStatsCard` clones the WRONG precedent — `InlineStatStrip` is already in use on partner pages and has a divergent API

- **Severity:** High
- **Location:** Phase 2, Step 3 (lines 62) and "Reference" section (line 54); plan.md Key Decision #3 (line 74) and Validation Log Q2 (lines 167–170)
- **Flaw:** The plan declares `src/components/admin-dashboard/PremiumStatStrip.tsx` as the canonical precedent and clones its API (`label, value, unit?, highlight?, onClick?, trend?`). It ignores that the partner's EmployeesPage already uses `@/components/shared/InlineStatStrip` (`frontend/src/pages/partner/EmployeesPage/index.tsx:6, 430`), whose API is similar but divergent:
  - `InlineStatStrip` items (`frontend/src/components/shared/InlineStatStrip.tsx:6–12`): `label, value, unit?, highlight?, valueClassName?, onClick?` — NO `trend`, but HAS `valueClassName`.
  - `PremiumStatStrip` items (`frontend/src/components/admin-dashboard/PremiumStatStrip.tsx:7–14`): `label, value, unit?, highlight?, onClick?, trend?` — HAS `trend`, NO `valueClassName`.
  - `InlineStatStrip` is shared across admin (`pages/admin/EmployeesPage`) and partner (`pages/partner/EmployeesPage`). PremiumStatStrip is admin-dashboard only.
- **Failure scenario:** Implementer builds `PartnerStatsCard` cloned from PremiumStatStrip (gains `trend`, loses `valueClassName`). Phase 3 re-points `pages/partner/EmployeesPage/index.tsx:430–477` from `<InlineStatStrip>` to `<PartnerStatsGrid>`/`<PartnerStatsCard>`. But the existing call sites use `valueClassName` (or rely on InlineStatStrip's specific layout: vertical/horizontal via prop at line 35) — the migration silently drops that styling. Partner sees stat-strip visual regression; Phase 4 catches it as a "tail of output captured" screenshot mismatch but Phase 3 implementer has no signal because the plan told them PremiumStatStrip was the canonical reference.
- **Evidence:**
  - Plan: `phase-02-shared-partner-primitives.md:62` declares `PremiumStatStrip` canonical.
  - Plan: `plan.md:74` Key Decision #3 — "Reuse the PremiumStatStrip precedent."
  - Actual: `frontend/src/pages/partner/EmployeesPage/index.tsx:430` uses `<InlineStatStrip>` with `items=[{label, value, onClick, highlight?}]` — confirmed API uses onClick + highlight, no trend.
  - Actual: `frontend/src/components/shared/InlineStatStrip.tsx:6–12, 35` — different prop shape.
  - Both files exist: `find frontend/src/components -name "InlineStatStrip.tsx"` returns `shared/InlineStatStrip.tsx`; `find frontend/src/components -name "PremiumStatStrip.tsx"` returns `admin-dashboard/PremiumStatStrip.tsx` AND `premium/PremiumStatStrip.tsx` (a duplicate, itself a DON'T #6 violation per `frontend/AGENTS.md:72`).
- **Suggested fix:** Phase 2 should explicitly name `InlineStatStrip` as the migration source for partner (it is already the in-use component on the surface being polished), and specify how `PartnerStatsCard` handles the `valueClassName` and `direction` props that InlineStatStrip exposes. If `PartnerStatsCard` is supposed to be a fresh primitive replacing BOTH, the migration step must call out the InlineStatStrip → PartnerStatsCard re-point as the actual work.

---

## Finding 6: `/partner/timesheet/payment-history` has NO mobile route variant — Phase 4 mobile QA criterion cannot pass for it

- **Severity:** High
- **Location:** Phase 4, Success Criteria (line 93) and Step 4 (lines 47–55); plan.md Acceptance Criteria (line 112)
- **Flaw:** Phase 4 promises "All five partner routes pass desktop + tablet + mobile visual QA at 320 / 390 / 768 / 1440." But `/partner/timesheet/payment-history` does not have a ResponsivePage wrapper in the router — it renders `<PartnerPaymentHistoryPage />` directly (`frontend/src/App.tsx:282`: `<Route path="timesheet/payment-history" element={<PartnerPaymentHistoryPage />} />`). There IS a mobile PaymentHistoryPage at `frontend/src/pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx` but it is not wired into the router. On mobile, partners see the desktop `BankTransferHistoryPageContent` (`frontend/src/pages/partner/PaymentHistoryPage/index.tsx:1–5`), with no `MobilePageShell` wrapper.
- **Failure scenario:** Phase 4 implementer loads `/partner/timesheet/payment-history` at 390px and either (a) declares pass because the desktop content reflows "fine" (false — no mobile shell, no FAB coordination, likely horizontal scroll), or (b) declares fail and tries to wire the mobile variant, expanding Phase 4 scope into router changes that were explicitly OUT of scope (`plan.md:53` "Out: route renames" and `:62` Preserve list). The success criterion is unverifiable as written.
- **Evidence:**
  - Plan: `phase-04-visual-qa-and-verification.md:93` "All five partner routes pass desktop + tablet + mobile visual QA at 320 / 390 / 768 / 1440."
  - Plan: `plan.md:46` Scope In lists `/partner/timesheet/payment-history` as a fifth surface.
  - Actual: `frontend/src/App.tsx:282` no `ResponsivePage` wrap.
  - Actual: `frontend/src/pages/partner/PaymentHistoryPage/index.tsx` is 5 lines, renders `<BankTransferHistoryPageContent />` directly.
  - Actual: `frontend/src/pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx` exists but is unused (grep for that path under router returns nothing).
- **Suggested fix:** Either (a) make Phase 4 explicitly say "Payment History: desktop only QA at 1440/768; mobile QA is out of scope" with rationale, or (b) add an explicit pre-Phase-1 step to wire `<ResponsivePage>` for the payment-history route, accepting the scope expansion. Don't leave the criterion ambiguous.

---

## Finding 7: Plan claims "19 partner components"; actual count is 28 — inventory math is wrong and deletion sweep will miss files

- **Severity:** Medium
- **Location:** plan.md Validation Log Q3 (line 172): "the existing ~19 per-page partner components after Phase 3 migration"; Phase 3 Step 9 (line 93) lists a partial subset
- **Flaw:** `find frontend/src/components/partner-* frontend/src/components/partner-dashboard -name "*.tsx" | wc -l` returns **28** `.tsx` files, not 19. Phase 3 Step 9's enumerated deletion list (`phase-03-partner-route-migration.md:93`) names only `PartnerEmployeesStats.tsx, PartnerEmployeesSummaryStats.tsx, PartnerEmployeesHeader.tsx, *Filters.tsx` — vague enough that an implementer could delete 6 or 16 depending on interpretation, with no authoritative manifest.
- **Failure scenario:** Phase 4's success criterion "every removed per-page partner component listed with deletion reason; grep confirms zero orphaned imports" cannot be satisfied because there is no authoritative pre-deletion list to diff against. Implementer either under-deletes (leaves dead `partner-projects/PartnerProjectsList.tsx`, `partner-dashboard/PartnerProjectsList.tsx` — both confirmed dead-by-import), or over-deletes (removes `PartnerWorkforceOverviewCard.tsx` which IS still imported by `pages/partner/DashboardPage/index.tsx:22` — verified) and breaks the dashboard build mid-Phase-3.
- **Evidence:**
  - Plan: `plan.md:172` "~19 per-page partner components."
  - Actual: `find frontend/src/components/partner-* frontend/src/components/partner-dashboard -name "*.tsx"` returns 28 files (full list enumerated in review).
  - Importer status verified: 27 of the 28 have ZERO external importers; only `PartnerWorkforceOverviewCard.tsx` and `PartnerEmployeeListSheet.tsx` are imported (both by `pages/partner/DashboardPage/index.tsx`).
- **Suggested fix:** Replace the ~19 estimate with an explicit table in Phase 3 listing all 28 files, their current importer count (0 for 26 of them, 1 for `PartnerWorkforceOverviewCard`, 1 for `PartnerEmployeeListSheet`), and the per-file deletion / preservation decision. Phase 4 verification = diff this table against actual `git rm` log.

---

## Finding 8: Payment History page is `BankTransferHistoryPageContent` — shared with admin; restyling it as a "partner" page silently regresses admin

- **Severity:** High
- **Location:** Phase 3 Step 5 (line 77); plan.md Acceptance Criteria (line 114) "[d]o not visually alter admin..."
- **Flaw:** `frontend/src/pages/partner/PaymentHistoryPage/index.tsx` is a 5-line re-export of `@/components/payroll/BankTransferHistoryPageContent` (`grep -rln BankTransferHistoryPageContent frontend/src` confirms `admin/PaymentHistoryPage/index.tsx` AND `partner/PaymentHistoryPage/index.tsx` both consume the SAME component). The plan's Phase 3 Step 5 says "Bring it to parity with the other four routes" — implying restyle with partner primitives — but there is no partner-only surface to restyle. The component is the admin payment history content.
- **Failure scenario:** Implementer wraps `<PartnerPageHeader>` or applies `--partner-*` tokens to `BankTransferHistoryPageContent` to "bring it to parity." Now `/admin/payment-history` inherits the same change because it's the same React subtree. Cross-surface isolation check (`phase-04-visual-qa-and-verification.md:61`) catches the leak — but only after the regression already happened. Rollback requires un-styling the shared component, which un-does partner polish too. The plan's Risk Controls section does not list this shared-component trap.
- **Evidence:**
  - Plan: `phase-03-partner-route-migration.md:77–78` Step 5 "Migrate `/partner/timesheet/payment-history` ... Bring it to parity."
  - Actual: `frontend/src/pages/partner/PaymentHistoryPage/index.tsx` 5 lines, imports `BankTransferHistoryPageContent`.
  - Actual: `grep -rln "BankTransferHistoryPageContent" frontend/src` returns the component file, its test, AND both `admin/PaymentHistoryPage` and `partner/PaymentHistoryPage`.
  - Test: `frontend/src/components/payroll/BankTransferHistoryPageContent.test.tsx:1–4` exists — `pnpm test:run -- partner` will run it; deletion or restyle of partner re-export that touches the shared content will trip this test.
- **Suggested fix:** Either (a) carve out a partner-specific wrapper that injects `--partner-*` overrides INSIDE `[data-partner-ui]` scoping WITHOUT editing `BankTransferHistoryPageContent`, or (b) explicitly mark Payment History as "token-only polish via `[data-partner-ui]` CSS overrides; no React-level migration." Add this trap to Phase 4 Risk Assessment.

---

## Finding 9: Phase 4 "visually unchanged from pre-plan baseline" criterion has no captured baseline — unverifiable acceptance gate

- **Severity:** Medium
- **Location:** Phase 4, Step 6 (line 62) and Success Criteria (line 95); plan.md Acceptance Criteria (line 114)
- **Flaw:** Phase 4 Step 6 says "Screenshot each; confirm visually unchanged from pre-plan baseline (or grab a fresh baseline if none exists)." But (a) no baseline is captured in Phase 1, (b) "grab a fresh baseline if none exists" is logically equivalent to "the criterion is auto-satisfied," (c) there is no tool/URL list/screenshot artifact path for what counts as the baseline. The cross-surface isolation check is the highest-severity gate (Phase 4 Risk Assessment line 110) and it is structurally a no-op.
- **Failure scenario:** Implementer runs Phase 4, sees `/admin/dashboard` looks "fine," checks the box. A subtle admin regression (e.g., `PartnerStatsCard`'s `bg-card` token resolving differently because `[data-partner-ui]` rule was mis-scoped and now cascades globally) is invisible because there is no side-by-side. The plan's whole risk model (`[data-partner-ui]` isolation) rests on this check, and the check is unfalsifiable.
- **Evidence:**
  - Plan: `phase-04-visual-qa-and-verification.md:62` "confirm visually unchanged from pre-plan baseline (or grab a fresh baseline if none exists)."
  - Plan: `phase-04-visual-qa-and-verification.md:95` Success Criterion "Cross-surface isolation check confirms `/admin/*`, `/employee/*`, `/login` are visually unchanged."
  - Plan: `phase-04-visual-qa-and-verification.md:110` Risk: "This is the highest-severity finding possible."
  - No baseline artifact, Phase 1 baseline-capture step, or screenshot path exists in any phase file.
- **Suggested fix:** Add a Phase 1 (or pre-Phase-1) step that captures baseline screenshots at fixed widths for `/admin/dashboard`, `/admin/employees`, `/employee`, `/login` using Playwright MCP, commits them under `plans/260719-1700-partner-surface-polish-tailkit/baselines/`. Phase 4 Step 6 becomes a pixel-diff against those, not a vibes check.

---

## Finding 10: Phase 3 claim "Move the column definitions and TanStack Table instance unchanged" is false — no route calls `useReactTable`

- **Severity:** Medium
- **Location:** Phase 3, Architecture step 4 (line 36) and Implementation Step 3 (line 72); Phase 3 Success Criteria (line 101)
- **Flaw:** Phase 3 Architecture step 4: "Replace the table wrapper markup with `<PartnerDataTable>`. Move the column definitions and TanStack Table instance unchanged." Verified: NO partner route calls `useReactTable` or `getCoreRowModel`. Grep for `useReactTable|useTable|getCoreRowModel` under `frontend/src/pages/partner/` AND `frontend/src/pages/mobile/partner/` returns ZERO hits. The only TanStack import on partner routes is `import type { ColumnDef } from "@tanstack/react-table"` (`pages/partner/EmployeesPage/index.tsx:42`) — a TYPE only, not an instance. The actual table instantiation lives INSIDE `<ResponsiveTable>` (`frontend/src/components/ui/responsive-table.tsx:56`), which partner routes consume as a black box.
- **Failure scenario:** Phase 3 implementer hunts for the "TanStack Table instance" to "preserve" per the plan, finds none, and either (a) assumes the migration is complete when nothing happened, or (b) inlines `useReactTable` directly into `pages/partner/EmployeesPage/index.tsx` to match the plan's expectation — a real behavior change (re-implementing sorting/pagination outside `ResponsiveTable`) that breaks URL-state preservation and the Phase 3 Success Criterion "[n]o route changes any hook, mutation, or service contract."
- **Evidence:**
  - Plan: `phase-03-partner-route-migration.md:36` Architecture step 4 "Move the column definitions and TanStack Table instance unchanged."
  - Plan: `phase-03-partner-route-migration.md:101` Success Criterion "column definitions and TanStack Table instances are unchanged."
  - Actual: `grep -rn "useReactTable|useTable|getCoreRowModel" frontend/src/pages/partner/ frontend/src/pages/mobile/partner/` → 0 matches.
  - Actual: `frontend/src/pages/partner/EmployeesPage/index.tsx:548–580` passes `columns`, `sorting`, `onSortingChange`, `pagination`, `onPageChange` as PROPS to `<ResponsiveTable>` — no instance to "move."
- **Suggested fix:** Rewrite Phase 3 table migration step to say "preserve the `columns`, `sorting`, `pagination`, and callback props passed to `<ResponsiveTable>`; the TanStack instance is internal to `ResponsiveTable` and is not modified by this plan." Drop the "TanStack Table instance unchanged" language.

---

**Total findings:** 10
