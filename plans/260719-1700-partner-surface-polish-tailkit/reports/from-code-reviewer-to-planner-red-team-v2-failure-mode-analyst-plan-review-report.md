# Red-Team Plan Review v2 — Failure Mode Analyst (Contract Verifier)

**Target plan:** `260719-1700-partner-surface-polish-tailkit` (v2 rewrite)
**Reviewer lens:** Murphy's Law — race conditions, data loss, cascading failures, recovery gaps, deployment/rollback risk, mobile↔desktop behavior drift, regression on shipped admin/employee surfaces.
**Verification tier:** Contract Verifier (Standard). Every claim grep-verified against `frontend/src/` and `frontend/tests/` with `path:line` citations.
**v1 findings:** 10 Critical/High. Resolution status assessed per finding at end.

Method note on the suggested-scope probes: the v2 Phase 1 classList effect with `[]` deps and no role check is sound. `ProtectedRoute.tsx:104-106` returns `<AuthLoadingScreen />` (not children) on role mismatch, so `PartnerLayout` unmounts and its cleanup fires before any non-partner route mounts. PartnerLayout is single-use (only `App.tsx:276`), unlike `AdminLayout` which is shared with `/adv-partner` and therefore needs the `if (!isAdmin) return` gate (`AdminLayout.tsx:100`). The upstream-role-gate reasoning holds; not a finding.

---

## Finding 1: Phase 4 Playwright "partner specs" gate is unverifiable — zero partner specs exist anywhere

- **Severity:** Critical
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Phase 4 introduces this gate; v1 had no equivalent)
- **Location:** Phase 4, "Implementation Steps → 1. Automated pass" (`phase-04-verification.md:35`) and "Success Criteria" (`phase-04-verification.md:91`)
- **Flaw:** Phase 4 Step 1 says "Playwright partner specs" pass, and Acceptance Criteria (`plan.md:175`, `phase-04-verification.md:91`) repeat "partner Playwright specs pass." Phase 4 acknowledges that `pnpm test:run -- partner` may be vacuous for Vitest, but makes the parallel claim about Playwright without verifying any partner spec exists. Verified: there are none.
- **Failure scenario:** Implementer reaches Phase 4, runs `pnpm test:e2e`, sees all 5 existing specs pass (none of them touch `/partner/*`), marks the partner-Playwright criterion green, and ships. A visual regression introduced during Phase 3 polish on `/partner/employees` is never caught by E2E because no spec loads the route. The "partner Playwright specs pass" line is a phantom gate — it passes vacuously by 0/0, identical to the Vitest case the plan does flag.
- **Evidence:**
  - Plan: `phase-04-verification.md:35` "`cd frontend && pnpm test:run -- partner` (note: v1 red-team found zero partner `*.test.tsx` specs ...)" — flags Vitest vacuousness only.
  - Plan: `phase-04-verification.md:91` acceptance criterion "[...] partner Playwright specs pass [...]" — no vacuousness caveat.
  - Actual: `find frontend/src -name "*.test.tsx" -path "*partner*"` → 0 results.
  - Actual: `ls frontend/tests/e2e` → `auth.spec.ts employee-portal.spec.ts employees.spec.ts projects.spec.ts timesheet.spec.ts` — 5 specs, none partner-scoped.
  - Actual: `grep -rn "/partner/" frontend/tests` → 0 route visits in any spec. `timesheet.spec.ts:18-19` mocks admin login and `page.goto('/timesheet')` (admin route, not `/partner/timesheet`).
  - `playwright.config.ts:7` `testDir: './tests/e2e'` confirms no other spec location.
- **Suggested fix:** Either (a) add a pre-Phase-4 task to write at least one partner smoke spec (`/partner/dashboard`, `/partner/employees` load + assert visible) so the gate is non-vacuous, or (b) explicitly document that BOTH Vitest and Playwright partner gates pass vacuously and that Phase 4 relies entirely on manual visual QA. Today the plan misleads by caveat-ing one and not the other.

---

## Finding 2: Phase 3 + Phase 4 Tailkit-literal-class grep is pre-doomed — 58 existing violations in scope today

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v1 had no equivalent gate; v2 introduced it as a quality guard)
- **Location:** Phase 3, "Success Criteria" (`phase-03-route-polish.md:95`); Phase 4, "Implementation Steps → 2" (`phase-04-verification.md:38-40`)
- **Flaw:** Both phases gate completion on `grep -rn -E "(secondary-[0-9]|emerald-[0-9]|orange-[0-9]|sky-[0-9]|violet-[0-9]|slate-[0-9]|dark:)" frontend/src/pages/partner frontend/src/pages/mobile/partner frontend/src/styles/partner.css frontend/src/components/PartnerSidebar.tsx frontend/src/components/MobileBottomNav.tsx frontend/src/layouts/PartnerLayout.tsx` returning zero matches. Verified today: 58 matches already exist in the in-scope files — most concentrated in `pages/partner/DashboardPage/index.tsx` and `pages/partner/EmployeesPage/index.tsx`. These are not introduced by Phase 3 polish; they are pre-existing.
- **Failure scenario:** Two failure modes. (a) Phase 3 implementer polishes one route, runs the gate, sees 50+ matches they did not create, and either halts (false-failure) or hand-waves past it (false-pass). (b) To make the gate actually return zero, the implementer must refactor ~58 pre-existing class literals across Dashboard/Employees — a scope expansion that contradicts `plan.md:113-117` ("Tailkit as visual reference ... never copy-pasted") and the "polish-only" framing. Either path invalidates the acceptance criterion.
- **Evidence:**
  - Plan: `phase-03-route-polish.md:95` and `phase-04-verification.md:38-40` define the zero-match gate.
  - Actual: `grep -rn -E "(secondary-[0-9]|emerald-[0-9]|sky-[0-9]|violet-[0-9]|orange-[0-9]|slate-[0-9]|dark:)" frontend/src/pages/partner frontend/src/pages/mobile/partner frontend/src/components/PartnerSidebar.tsx frontend/src/components/MobileBottomNav.tsx frontend/src/layouts/PartnerLayout.tsx` → 58 matches.
  - Sample: `pages/partner/EmployeesPage/index.tsx:45-47` `bg-sky-50 text-sky-700 border-sky-200/80`, `bg-violet-50 text-violet-700`, `bg-emerald-50 text-emerald-700`; `pages/partner/DashboardPage/index.tsx:120` `text-emerald-600`, `:199-200` `text-slate-400 ring-slate-200 bg-slate-100 text-slate-700`, `bg-orange-100 text-orange-700`, `:258` `from-violet-400 to-violet-500`.
- **Suggested fix:** Either (a) narrow the grep to ONLY files Phase 3 actually creates/modifies (`git diff --name-only` against the phase-3 start commit, then grep just those), or (b) add a baseline-exception clause: "Pre-existing Tailkit literals in `DashboardPage` and `EmployeesPage` are out of scope unless Phase 2 audit flags that specific class for migration; the gate applies only to NEW classes introduced by Phase 3." Without this the gate cannot pass without undeclared scope creep.

---

## Finding 3: Phase 2 dead-code grep recipe false-positives on the live `hooks/partner-employees/` folder — manifest will mark `components/partner-employees/` files as "still imported"

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Phase 2 introduces the three-grep deletion manifest)
- **Location:** Phase 2, "Implementation Steps → 4. Dead-code audit" (`phase-02-audit-and-primitives-parity.md:62-66`)
- **Flaw:** Phase 2 Step 4's dynamic-import grep is `grep -rln "partner-{folder}/" --include="*.tsx" --include="*.ts" frontend/src | grep -v partner-{folder}/`. For `partner-employees`, this returns 2 hits that are NOT importers of `components/partner-employees/` — they are importers of the live, in-use `hooks/partner-employees/usePartnerEmployeesData`. The substring `partner-employees/` matches both `components/partner-employees/` and `hooks/partner-employees/`. The recipe's exclusion `grep -v partner-{folder}/` only excludes self-references inside the same folder, not the unrelated `hooks/` nameshare.
- **Failure scenario:** Phase 2 auditor runs the dynamic grep for `partner-employees/`, sees 2 matches (`pages/partner/EmployeesPage/index.tsx`, `pages/mobile/partner/EmployeesPage/index.tsx`), and concludes "components/partner-employees/ is still referenced — block deletion." Phase 3 then skips the deletion, leaving 7 orphaned `.tsx` files in the tree. The plan's `plan.md:170` acceptance criterion "`grep -rln "from \"@/components/partner-"` returns zero hits across `src/`" then cannot be met, blocking Phase 4. Auditor has no documented path to disambiguate "importer of components/ vs importer of hooks/."
- **Evidence:**
  - Plan: `phase-02-audit-and-primitives-parity.md:64` `grep -rln "partner-{folder}/" --include="*.tsx" --include="*.ts" frontend/src | grep -v partner-{folder}/` (no anchor on `components/`).
  - Actual: `grep -rln "partner-employees/\|partner-projects/\|partner-timesheet/" frontend/src | grep -v AGENTS.md` → exactly 2 hits, both `hooks/` importers:
    - `pages/partner/EmployeesPage/index.tsx:9` `import { usePartnerEmployeesData } from "@/hooks/partner-employees/usePartnerEmployeesData";`
    - `pages/mobile/partner/EmployeesPage/index.tsx:24` same import.
  - Actual: `grep -rln "from \"@/components/partner-employees\|from \"@/components/partner-projects\|from \"@/components/partner-timesheet"` → 0 hits (confirmed dead).
  - Actual: `ls frontend/src/hooks/partner-employees/` → `usePartnerEmployeeInfiniteScroll.ts usePartnerEmployeesData.ts` (live, used by 2 pages).
- **Suggested fix:** Anchor every grep on the literal `components/partner-{folder}` (or `@/components/partner-{folder}`). Recipe: `grep -rln "@/components/partner-{folder}\|components/partner-{folder}/" frontend/src --include="*.tsx" --include="*.ts" | grep -v "components/partner-{folder}/"`. Explicitly note that `hooks/partner-{folder}/` is a separate live tree and out of scope for deletion.

---

## Finding 4: Phase 4 portal-scope verification is unverifiable when Phase 3 legitimately produces an empty partner.css

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (consequence of v2 Key Decision #1 "reuse over recreate" combined with Phase 3 Step 5's "partner.css may stay near-empty — that is a valid outcome")
- **Location:** Phase 4, "Implementation Steps → 4. Portal scope verification" (`phase-04-verification.md:45-48`); contradiction with Phase 1 Step 3 (`phase-01-foundation.md:68-72`) and Phase 3 Step 5 (`phase-03-route-polish.md:62-64`)
- **Flaw:** Phase 1 ships `partner.css` with only a placeholder comment (`/* Phase 3 will populate */`). Phase 3 Step 5 explicitly says "If no rules were needed (because `components/shared/*` already covers everything), `partner.css` may stay near-empty — that is a valid outcome." Phase 4 Step 4 then asks the verifier to "Open a partner sheet (e.g., `UserProfileSheet`): the Radix portal content should be reachable by partner CSS rules via the `html.partner-route-active` scope." If partner.css has no rules, "reachable by partner CSS rules" is vacuously true for any rule set (empty set ⊆ anything), but the verification provides no signal that the scoping actually works in practice.
- **Failure scenario:** Phase 3 ships an empty partner.css (legitimate per Step 5). Phase 4 Step 4 verifier reports "portal content reachable — pass" without any rule to test against. Later, Phase 5/6 (some future plan) populates `partner.css` with `[data-partner-ui]` presentation rules assuming the `html.partner-route-active` scope reaches Radix portals — but the scope was never stress-tested with a real rule. If the selector is wrong (e.g., admin uses `[role="dialog"]` per `admin-daisy.css:393`, not `[data-radix-popper-content-wrapper]` as Phase 1's CSS template suggests at `phase-01-foundation.md:46`), the bug ships silently because Phase 4 had no rule to fail on.
- **Evidence:**
  - Plan: `phase-01-foundation.md:46` template uses `html.partner-route-active [data-radix-popper-content-wrapper]`.
  - Plan: `phase-01-foundation.md:70` placeholder rule `/* Phase 3 will populate */`.
  - Plan: `phase-03-route-polish.md:64` "partner.css may stay near-empty — that is a valid outcome."
  - Plan: `phase-04-verification.md:48` "Open a partner sheet ... should be reachable by partner CSS rules."
  - Actual: `frontend/src/styles/admin-daisy.css:393` admin's proven pattern is `html.admin-route-active :is([role="dialog"], [role="menu"], [role="listbox"])` — NOT `[data-radix-popper-content-wrapper]`. The v2 template's portal selector may not match the project's actual Radix portal DOM.
- **Suggested fix:** Phase 1 should ship at least ONE functional portal-reach rule in partner.css (e.g., a harmless `html.partner-route-active [role="dialog"] { outline: 2px solid transparent; }`) so Phase 4 Step 4 has a concrete selector to verify. Alternatively, Phase 4 Step 4 should be reworded to "if partner.css contains any `[role="dialog"]`-style rule, exercise it; otherwise document that portal scope is untested and add a follow-up issue."

---

## Finding 5: Phase 3 Step 8 mobile Payment History wiring ignores divergent data contracts between desktop and mobile variants

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 made mobile Payment History wiring an optional Phase 3 Step 8; v1 had it implicit)
- **Location:** Phase 3, "Implementation Steps → 8. Optional: wire mobile Payment History" (`phase-03-route-polish.md:72-75`)
- **Flaw:** Phase 3 Step 8 says "Confirm `pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx` is a working mobile component (read its exports first)" and treats the swap as a pure routing change. Verified: the two components are NOT interchangeable presentations of the same data. Desktop `BankTransferHistoryPageContent` uses `useBankTransferHistories` (paginated) with filter fields `{ month, cycle, search, page, pageSize }`. Mobile `PaymentHistoryPage` uses `useInfinitePaymentHistories` (infinite scroll) with filter fields `{ sortBy, sortOrder, fromDate, toDate, projectId, employeeId, position, search }`. They consume different hooks, expose different filter dimensions (desktop has `cycle`; mobile has `projectId`/`employeeId`/`position`/date-range), and have different pagination models. Wiring mobile via `ResponsivePage` gives partners a fundamentally different feature surface on mobile vs desktop — not a parity polish.
- **Failure scenario:** Partner user on desktop filters Payment History by `cycle` (pay cycle), then opens the same page on mobile — the `cycle` filter is gone, replaced by `fromDate/toDate/projectId/employeeId/position`. URL query state (which `plan.md:174` Preserve list says must be maintained) cannot round-trip because the two filter schemas are disjoint. Phase 4 Step 6 "Confirm `/partner/timesheet/payment-history` is visually identical to admin" passes (desktop unchanged), but Phase 3 Step 9 "URL query state survives a page reload" fails opaquely on cross-device handoff. The plan flags this risk only as "wiring breaks admin" (`plan.md:184`) — the actual risk is partner-mobile UX divergence.
- **Evidence:**
  - Plan: `phase-03-route-polish.md:73-75` treats the wiring as a route swap with no filter-schema diff noted.
  - Actual: `frontend/src/components/payroll/BankTransferHistoryPageContent.tsx:24,283-297` — `useBankTransferHistories`, filters `{ month, cycle, debouncedSearch, page, pageSize }`.
  - Actual: `frontend/src/pages/mobile/partner/TimesheetsPage/PaymentHistoryPage.tsx:12,28-39,92` — `useInfinitePaymentHistories`, filters `{ sortBy, sortOrder, fromDate, toDate }` plus `PaymentHistoryFilters` props `{ projectId, employeeId, position }` (line 79-82).
- **Suggested fix:** Either (a) make Step 8 explicitly out of scope (defer to a dedicated plan that reconciles the two filter schemas first), or (b) add a Step 8a requiring the implementer to document the filter-schema diff and either reconcile them or mark mobile-only filters as intentional divergence. Do not present the swap as "verify the desktop re-export is byte-unchanged" — that is the trivial part; the data-contract gap is the real risk.

---

## Finding 6: Phase 4 modal-registry string grep targets the wrong directories — actual `partner-*` substrings live in `contexts/` and `config/`

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Phase 4 Step 3 introduces this grep)
- **Location:** Phase 4, "Implementation Steps → 3. Dead-code removal verification" (`phase-04-verification.md:44`)
- **Flaw:** Phase 4 Step 3 runs `grep -rln "partner-employees\|partner-projects\|partner-timesheet" frontend/src/constants frontend/src/lib 2>/dev/null` and expects zero hits as proof that "modal-registry strings cleaned up." Verified: `constants/` and `lib/` have zero such substrings (grep returns empty), so the gate always passes — but the actual `partner-employees`/`partner-timesheet` substrings in the codebase live in `contexts/CommandPaletteContext.tsx` and `config/actions.ts` as command-palette action IDs. The verification gives a false "clean" signal.
- **Failure scenario:** Phase 4 verifier runs the gate, sees zero hits, marks "modal-registry strings cleaned up" complete. In reality, `contexts/CommandPaletteContext.tsx:164,174,184` and `config/actions.ts:79,97` still emit `nav-partner-projects`, `nav-partner-employees`, `nav-partner-timesheet` command IDs. These aren't broken references (they're string IDs, not file imports), so nothing crashes — but the plan's claim of cleanliness is false, and a future developer greps for "partner-employees" expecting zero hits per the plan's verification log and is misled.
- **Evidence:**
  - Plan: `phase-04-verification.md:44` `grep -rln "partner-employees\|partner-projects\|partner-timesheet" frontend/src/constants frontend/src/lib` — wrong dirs.
  - Actual: `grep -rn "partner-employees\|partner-projects\|partner-timesheet" frontend/src/contexts frontend/src/config`:
    - `contexts/CommandPaletteContext.tsx:164` `id: 'nav-partner-projects'`
    - `contexts/CommandPaletteContext.tsx:174` `id: 'nav-partner-employees'`
    - `contexts/CommandPaletteContext.tsx:184` `id: 'nav-partner-timesheet'`
    - `config/actions.ts:79` `id: 'nav-partner-employees'`
    - `config/actions.ts:97` `id: 'nav-partner-timesheet'`
  - Actual: `grep -rln "partner-employees\|partner-projects\|partner-timesheet" frontend/src/constants frontend/src/lib` → 0 (gate passes vacuously).
- **Suggested fix:** Either (a) broaden the gate to `grep -rln "partner-employees\|partner-projects\|partner-timesheet" frontend/src --include="*.ts" --include="*.tsx" | grep -v AGENTS.md | grep -v "components/partner-"` and document that command-palette ID strings (`nav-partner-*`) are EXPECTED to remain (they are not file refs), or (b) drop the misleading "modal-registry strings cleaned up" framing — there are no modal-registry strings for these folders to begin with (verified: `grep partner-employees frontend/src/constants` empty).

---

## Finding 7: Phase 3 deletion leaves at least 4 AGENTS.md files with stale references to deleted folders

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Phase 3 deletion sweep has no doc-sync step)
- **Location:** Phase 3, "Implementation Steps → 7. Dead-code deletion sweep" (`phase-03-route-polish.md:68-71`)
- **Flaw:** Phase 3 Step 7 records each deletion in `reports/phase-03-deletion-log.md` but does not update AGENTS.md files that reference the deleted folders as live components. Verified: at least 4 AGENTS.md files describe `partner-employees/`, `partner-projects/`, `partner-timesheet/` as current component locations.
- **Failure scenario:** After Phase 3 deletion, a future contributor (or AI agent) reads `pages/partner/AGENTS.md:50-52` directing them to `../../components/partner-employees/` for the employee list, finds no such directory, and either wastes time searching or assumes the documented architecture is wrong and re-creates the folder. Same for `components/AGENTS.md:50-52`, `hooks/AGENTS.md:59-60`, `config/AGENTS.md:38-39`. Documentation rot propagates from a "cleanup" task.
- **Evidence:**
  - Plan: `phase-03-route-polish.md:71` only requires logging the deletion, not syncing docs.
  - Actual stale refs (verified):
    - `frontend/src/pages/partner/AGENTS.md:50` `../../components/partner-employees/` for employee list
    - `frontend/src/pages/partner/AGENTS.md:51-52` same for projects, timesheet
    - `frontend/src/components/AGENTS.md:50-52` lists all three folders as live
    - `frontend/src/hooks/AGENTS.md:59-60` lists `partner-employees/`, `partner-timesheet/`
    - `frontend/src/config/AGENTS.md:25-26,38-39` lists `partner-timesheet-columns.tsx`, `partner-timesheet-mobile.tsx`, `partner-employees/`, `partner-timesheet/`.
- **Suggested fix:** Add Step 7a: "For each deleted folder, grep all `AGENTS.md` files under `frontend/src/` for references and either delete the line or update it to point at the actual replacement (e.g., `components/shared/PageHeader`, `ResponsiveTable`). Verify with `grep -rln "partner-employees/\|partner-projects/\|partner-timesheet/" frontend/src --include="AGENTS.md"` returns zero."

---

## Finding 8: Phase 4 cross-surface isolation "visually unchanged" criterion is unverifiable — no baseline infrastructure exists

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 acknowledges this in Open Q2 but the Acceptance Criteria text still overclaims)
- **Location:** `plan.md:176` Acceptance Criteria; Phase 4 "Implementation Steps → 7" (`phase-04-verification.md:61-63`) and "Success Criteria" (`phase-04-verification.md:94`)
- **Flaw:** Open Question 2 (`plan.md:200`) correctly defaults to "capture fresh — the existing admin/employee polish plans don't reference a shared baseline dir," and Phase 4 Step 7 acknowledges "if no prior baseline exists, this capture becomes the baseline; document that the comparison is 'first baseline' not 'regression check'." But the Acceptance Criteria line still says "Cross-surface isolation: `/admin/*`, `/employee/*`, `/login` visually unchanged (baseline screenshots captured in Phase 4 if not already present)." With no prior baseline, "visually unchanged" is a meaningless phrase — there is nothing to compare against. The Phase 4 Success Criteria `[...] Cross-surface isolation check confirms /admin/*, /employee/*, /login are visually unchanged (or, if first baseline, documented as such)` hedges this with the parenthetical, but the plan-level Acceptance Criterion does not.
- **Failure scenario:** Phase 4 implementer captures fresh screenshots of `/admin/dashboard`, marks "visually unchanged" pass. A real admin regression introduced by accident (e.g., a CSS cascade leak from `partner.css` placed after `admin-daisy.css` per Phase 1 Step 4 `phase-01-foundation.md:72`) is invisible because there is no prior screenshot to diff against. The `plan.md:181` Risk Controls entry for "Admin regression on shared components" promises baseline-vs-after diffing, but the infrastructure for it does not exist and the plan does not require creating it before Phase 3 changes can land.
- **Evidence:**
  - Plan: `plan.md:176` "Cross-surface isolation: `/admin/*`, `/employee/*`, `/login` visually unchanged (baseline screenshots captured in Phase 4 if not already present)."
  - Plan: `plan.md:200` Open Q2 acknowledges "Default: capture fresh — the existing admin/employee polish plans don't reference a shared baseline dir."
  - Plan: `phase-01-foundation.md:72` places `partner.css` after `admin-daisy.css` in cascade — actual potential leak vector.
  - Actual: `find . -path "*baseline*" -not -path "*/node_modules/*"` → 0 results (no baseline infrastructure). `find . -name "*.png" -not -path "*/node_modules/*" -not -path "*/dist/*"` → only static brand assets under `dist/` and `public/`, no visual-regression baseline dir.
- **Suggested fix:** Reword `plan.md:176` to: "Cross-surface isolation: fresh baselines of `/admin/dashboard`, `/admin/employees`, `/employee`, `/login` are captured in Phase 4; a follow-up issue is filed to integrate Playwright screenshot diffing so future plans have a real baseline. DOM-level isolation (no `partner-route-active` class, no `[data-partner-ui]` node) is asserted on each non-partner route as the only enforceable check this plan." Also consider making Phase 1 Step 4 place `partner.css` BEFORE `admin-daisy.css` to eliminate the cascade-leak vector at the source.

---

## Finding 9: Phase 2 "shared primitive adoption map" Step 6 grep recipe misses barrel and dynamic consumers of `shared/*`

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Phase 2 Step 6 introduces this cross-surface consumer map as the blast-radius source for Phase 3)
- **Location:** Phase 2, "Implementation Steps → 6. Cross-surface preservation check" (`phase-02-audit-and-primitives-parity.md:68`)
- **Flaw:** Step 6 recipe is `grep -rln "from \"@/components/shared/{Name}\""` per primitive. This misses three real consumer patterns verified in the codebase: (a) barrel imports via `components/shared/index.ts` (verified to exist) — consumers may import as `from "@/components/shared"` and the grep returns nothing; (b) re-exports via `components/shared/index.ts` that rename or aggregate; (c) imports without the `@/` alias (relative paths). The "blast radius" map that Phase 3 Step 5 / Phase 4 Step 9 rely on (to decide whether a `shared/*` edit is safe) can silently undercount consumers.
- **Failure scenario:** Phase 3 polishes `pages/partner/EmployeesPage` by tweaking `InlineStatStrip` (e.g., adding a new optional prop with a default, OR fixing what looks like a bug). Phase 2 Step 6 reported "InlineStatStrip consumers: 2" (only the two files using the literal `from "@/components/shared/InlineStatStrip"` path). Actual consumers via barrel `from "@/components/shared"` push the count higher. Phase 4 Step 9 screenshots only the 2 mapped consumers and ships; an unmapped admin consumer regresses silently.
- **Evidence:**
  - Plan: `phase-02-audit-and-primitives-parity.md:68` `grep -rln "from \"@/components/shared/{Name}}"` — anchored only on the deep path.
  - Actual: `frontend/src/components/shared/index.ts` exists (verified via `ls`).
  - Actual consumers of partner-used primitives verified via `grep "from \"@/components/shared/"`: 4 partner pages use the deep-import path (EmployeesPage, mobile EmployeesPage, mobile ProjectsPage, mobile TimesheetsPage), but the barrel is also exported and could be used by any other surface. Phase 2 has no recipe to detect barrel consumers.
- **Suggested fix:** Replace Step 6 recipe with two greps: (1) the existing deep-path grep; (2) `grep -rln "from \"@/components/shared\"" frontend/src --include="*.tsx" --include="*.ts"` to find barrel importers, then `grep -l "{Name}"` on those files to filter. Also instruct the auditor to consult `components/shared/index.ts` to confirm which primitives are barrel-exported before declaring the consumer map complete.

---

## Finding 10: Plan-level claim "shared/ count: 31 `.tsx` files plus barrel" is correct, but Phase 2 doc says "32 files" — internal inconsistency

- **Severity:** Low
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 Whole-Plan Consistency Sweep at `plan.md:236` claims to have reconciled this exact count, but did not surface the Phase 2 wording)
- **Location:** `plan.md:236` ("Corrected `shared/` count from '32 files' to '31 `.tsx` files plus barrel'"); `phase-02-audit-and-primitives-parity.md:43` ("`frontend/src/components/shared/*` (32 files — the primitive inventory)")
- **Flaw:** `plan.md`'s Whole-Plan Consistency Sweep explicitly claims to have corrected the shared/ count discrepancy and verified it via `ls frontend/src/components/shared/*.tsx | wc -l → 31`. Phase 2's "Related Code Files" still says "32 files." Both can be defended (31 `.tsx` + 1 `index.ts` = 32 total entries), but the inconsistency survives the sweep that claimed to catch it — suggesting the sweep was not actually run against Phase 2.
- **Failure scenario:** Low impact, but signals that the v2 "Whole-Plan Consistency Sweep" step (`plan.md:231-243`) is not a reliable gate. A more material inconsistency (e.g., a stale v1 claim) could survive the same sweep.
- **Evidence:**
  - Plan: `plan.md:236` "Corrected `shared/` count from '32 files' to '31 `.tsx` files plus barrel'."
  - Plan: `phase-02-audit-and-primitives-parity.md:43` "`frontend/src/components/shared/*` (32 files — the primitive inventory)".
  - Actual: `find frontend/src/components/shared -name "*.tsx" | wc -l` → 31. `ls frontend/src/components/shared/index.ts` → exists. So "32 files" is defensible if counting the barrel, but Phase 2's bare "32" without qualification conflicts with `plan.md`'s "31 `.tsx` plus barrel."
- **Suggested fix:** Align Phase 2's wording to `plan.md`'s: "31 `.tsx` files plus `index.ts` barrel." Re-run the consistency sweep against ALL phase docs, not just plan.md.

---

## v1 Findings Resolution Status

| # | v1 Finding | v2 Status | Evidence |
|---|---|---|---|
| 1 | v1 targeted dead code | RESOLVED | v2 `plan.md:91-99` repurposes folders as deletion targets; `grep -rln "from \"@/components/partner-"` confirmed 0 hits. (See Finding 3 for a NEW v2 recipe bug in the deletion verification, but the underlying resolution is sound.) |
| 2 | v1 omitted mobile routes | RESOLVED | v2 enumerates all 4 mobile routes in `plan.md:71-77`; verified `App.tsx:278-281` uses `ResponsivePage` for all four. (See Finding 5 for the optional 5th mobile route divergence.) |
| 3 | False "admin sidebar uses `ct-menu`" | RESOLVED | v2 Key Decision #6 drops the claim; `AdminSidebar.tsx` confirmed shadcn-only. |
| 4 | `make api-test` location | RESOLVED | v2 `phase-04-verification.md:36` invokes via `cd backend &&`; `backend/Makefile:242` confirmed; root `Makefile` confirmed no such target. |
| 5 | `--partner-*` token at `:root` | RESOLVED | v2 Key Decision #3 keeps `--partner-accent` at `variables.css:212`; verified at line 212. |
| 6 | `[data-partner-ui]` can't reach portals | RESOLVED | v2 mirrors `AdminLayout.tsx:99-103` classList pattern; admin-daisy.css:393 uses `[role="dialog"]` selector that does reach portals. (See Finding 4 for v2's incomplete verification when partner.css is empty.) |
| 7 | Redundant `PartnerDataTable` | RESOLVED | v2 Key Decision #1 reuses `ui/{responsive,data,mobile}-table`; verified all three exist. |
| 8 | PremiumStatStrip parity overstatement | RESOLVED | v2 `plan.md:87` lists `InlineStatStrip`/`GroupedStatCard` as the adopted primitives; verified in use at `pages/partner/EmployeesPage/index.tsx:6`. |
| 9 | TanStack assumption held for 1 of 5 | RESOLVED | v2 Survey records actual table primitive per route; only Employees uses `ResponsiveTable` (TanStack), others use plain markup or shared components. |
| 10 | PaymentHistoryPage is shared re-export | RESOLVED | v2 Scope excludes restyle; `pages/partner/PaymentHistoryPage/index.tsx` confirmed 5-line re-export of `BankTransferHistoryPageContent`. (See Finding 5 for risk in the optional mobile-wiring step.) |

**Verdict:** 10/10 v1 findings RESOLVED at the plan-text level. v2 introduces 10 NEW failure modes (this report) — 1 Critical (Finding 1: phantom Playwright gate), 4 High (Findings 2, 3, 4, 5), 4 Medium (Findings 6, 7, 8, 9), 1 Low (Finding 10). Net: v1's defects are fixed, but v2's verification architecture is over-claiming — three of its four "zero-match" gates (Vitest partner specs, Playwright partner specs, Tailkit literal classes) either pass vacuously or fail pre-existingly, and a fourth (modal-registry strings) checks the wrong directories.

**Total findings: 10**
**v1 resolution verdict: RESOLVED** (10/10 at plan-text level; v2 introduces 10 new defects that must be addressed before execution).
