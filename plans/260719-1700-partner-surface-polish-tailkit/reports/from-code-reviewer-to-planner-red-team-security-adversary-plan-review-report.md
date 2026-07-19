# Red Team Plan Review — Security Adversary

**Plan:** Partner Surface Polish via Tailkit Reference (`260719-1700-partner-surface-polish-tailkit`)
**Reviewer role:** Security Adversary (attacker mindset: auth bypass, CSS scope leakage, privilege escalation, XSS / injection, OWASP top 10)
**Method:** Fact-check plan claims against `/Users/dev/Documents/projects/payroll/frontend` with grep/read, then attack each load-bearing assumption.
**Verification tier:** Standard.

---

## Finding 1: Plan's "every partner route" scope omits 1,506 lines of separately-routed mobile pages

- **Severity:** Critical
- **Location:** `plan.md` "Scope → In" (lines 41–47); `phase-03-partner-route-migration.md` "Requirements → Functional" (lines 18–22) and "Modify (route pages)" (lines 44–48)
- **Flaw:** The plan enumerates desktop pages only (`pages/partner/*`) and lists mobile components under `components/partner-*/mobile/*`. It never mentions `pages/mobile/partner/*`, which is a completely separate, parallel code tree rendered via `ResponsivePage`. The plan claims "desktop + mobile in the same step" but treats mobile as a sub-component of the desktop pages — it is not.
- **Failure scenario:** Router wires each partner route as `<ResponsivePage desktopComponent={PartnerXPage} mobileComponent={PartnerXPageMobile} />` (`App.tsx:278–289`). On a 390px viewport the user sees `pages/mobile/partner/{Dashboard,Projects,Employees,Timesheets}Page/index.tsx` (1506 LOC total — verified by `wc -l`). These pages use `MobilePageShell`/`MobilePageHeader` from `@/components/shared/*`, not any `partner-*` component. After Phase 3, a phone visitor sees zero of the new design system, no `[data-partner-ui]` styling, no token translation, no PartnerStatsCard / PartnerFilterBar / PartnerMobileListRow. The acceptance criterion "Every declared `/partner/*` route ... inherits the new partner design system" silently fails for ~50% of traffic.
- **Evidence:**
  - Router wiring: `frontend/src/App.tsx:61–64,278,278–289` (lazy imports + `<ResponsivePage desktopComponent=... mobileComponent=...>`).
  - `frontend/src/components/ResponsivePage.tsx:12–20` (mobile branch returns `<Mobile />` directly, no shared wrapper).
  - `frontend/src/pages/mobile/partner/{DashboardPage,EmployeesPage,ProjectsPage,TimesheetsPage}/index.tsx` exist (1,506 LOC).
  - Plan's "In" scope (plan.md:41–47) lists only `pages/partner/*` and `components/partner-*/*`.
  - Plan's "Out of scope" (plan.md:51–59) does not declare these mobile pages out of scope — they are simply absent.
- **Suggested fix:** Either (a) add `pages/mobile/partner/*` explicitly to Phase 3 scope and migrate those four pages through `partner-ui/` primitives + the `[data-partner-ui]` attribute must be set by `PartnerLayout` (the parent — verified it wraps both branches), or (b) declare the mobile page tree out of scope as a separate plan, with explicit acceptance that mobile partner users will not receive the redesign.

---

## Finding 2: `[data-partner-ui]` scoping silently fails to reach Radix-portal surfaces (dialogs/sheets/dropdowns)

- **Severity:** Critical
- **Location:** `phase-01-partner-design-foundation.md` "Implementation Steps" step 6 (lines 76–78) and "Success Criteria" (lines 96–104); `plan.md` "Risk Controls" (lines 124)
- **Flaw:** The plan's isolation model assumes the `[data-partner-ui]` div is an ancestor of every styled element. It is not. Radix portals (Dialog, Sheet, DropdownMenu) render to `document.body`, OUTSIDE the layout div. The admin surface already solved this with an `html.admin-route-active` class escape hatch that styles `[role="dialog"]`, `[role="menu"]`, `[role="listbox"]` portals. The partner plan has no equivalent escape hatch and never references the admin pattern.
- **Failure scenario:** Partner users open `UserProfileSheet` (`PartnerSidebar.tsx:256`), the change-password modal (`PartnerSidebar.tsx:232` → `MODAL_IDS.CHANGE_PASSWORD`), or any dropdown (`PartnerSidebar.tsx:236` → `MODAL_IDS.NOTIFICATION_SHEET`). These render at `document.body`, outside `<div data-partner-ui>`. Their `--partner-*` token references resolve to nothing, and any `partner-ui/` primitive rendered inside a Sheet/Dialog inherits the global shadcn theme, not the partner design. Result: partner portal surfaces look unstyled or admin-styled, and the Phase 4 isolation check ("`document.querySelector('[data-partner-ui]')` returns null on /admin") passes while the partner's own portal surfaces are broken — a false-negative.
- **Evidence:**
  - Partner portals: `frontend/src/components/PartnerSidebar.tsx:232,236,256` (change-password, notification, profile sheet all wired).
  - Shared sheets hardcode admin theme tokens: `frontend/src/components/sheets/UserProfileSheet.tsx:198` (`data-theme="congtruong"`), `frontend/src/components/modals/ChangePasswordModal.tsx:84` (`data-theme="congtruong"`), `frontend/src/components/notifications/NotificationSheet.tsx:135` (defaults to `congtruong` when not `employee`).
  - Admin's escape-hatch pattern: `frontend/src/styles/admin-daisy.css:393,399,405` (`html.admin-route-active [role="dialog"] ...`, `:is([role="menu"], [role="listbox"])`).
  - Admin sets the html class: `frontend/src/layouts/AdminLayout.tsx:99–103` (`document.documentElement.classList.add("admin-route-active")`).
  - Plan never mentions portal escape hatch, `html.*-route-active`, or `role="dialog"` in any phase.
- **Suggested fix:** Phase 1 step 6 must add `document.documentElement.classList.add("partner-route-active")` to `PartnerLayoutInner` (with cleanup), AND `partner.css` must mirror the `html.partner-route-active [role="dialog"|"menu"|"listbox"]` pattern from `admin-daisy.css:393–409`. Phase 4 isolation check must add a portal-coverage assertion (open `UserProfileSheet` on `/partner/dashboard`, assert `[data-partner-ui]`-derived tokens resolve on at least one element inside the portal).

---

## Finding 3: Existing `--partner-accent` token is `:root`-global; plan moves it under `[data-partner-ui]`, silently breaking the in-flight sidebar

- **Severity:** Critical
- **Location:** `phase-01-partner-design-foundation.md` "Implementation Steps" step 2 (line 59) and "Architecture" diagram (lines 30–40)
- **Flaw:** Plan step 2 instructs deriving a `--partner-*` token set "mirroring the admin pattern" — but admin tokens (`--admin-*`) live under `[data-admin-ui], html.admin-route-active` (`variables.css:313–314`), while the existing `--partner-accent` lives in `:root` (`variables.css:212`). The plan does not call out that moving partner tokens from `:root` into `[data-partner-ui]` scope is a breaking change for any code that consumes `--partner-accent` outside the layout div (e.g., portal sheets, global toasts, lazy-loaded content rendered before layout mounts).
- **Failure scenario:** Today `PartnerSidebar.tsx:79,84,90` consumes `hsl(var(--partner-accent)...)`. After Phase 1 moves it under `[data-partner-ui]`, any consumer outside that scope (toasts via sonner at `document.body`, modal content rendered via `data-theme="congtruong"` portals) silently falls back to `hsl()` with no argument → invalid color → `transparent`/`canvas` background. The active-state accent on the partner sidebar disappears the moment it is wrapped by a portal or rendered during a route transition before `PartnerLayoutInner` mounts.
- **Evidence:**
  - Token at `:root`: `frontend/src/styles/variables.css:212` (`--partner-accent: 148 60% 58%;` — inside `:root` block, not `[data-...]`).
  - Existing consumers: `frontend/src/components/PartnerSidebar.tsx:79,84,90` (3 sites).
  - Plan's instruction: `phase-01...md:59` ("Derive `--partner-*` token set ... mirroring the admin pattern").
  - Admin's scoping pattern that plan would mirror: `frontend/src/styles/variables.css:313–314` (`[data-admin-ui], html.admin-route-active { --background: var(--admin-background); ... }`).
- **Suggested fix:** Phase 1 step 2 must (a) audit every consumer of `--partner-accent` and confirm none render outside `[data-partner-ui]`, AND (b) decide explicitly: keep new `--partner-*` tokens at `:root` (lower isolation risk, accept they exist on every surface) OR move them under `[data-partner-ui]` + add `html.partner-route-active` to reach portals. State the trade-off in the plan; do not silently inherit one strategy from admin.

---

## Finding 4: `PremiumStatStrip` "API parity" claim is factually wrong — plan invents `valueFormat`, `icon`, `sparkline` not present in source

- **Severity:** High
- **Location:** `plan.md` "Validation Log — Confirmed Decisions" (lines 184–188); `phase-02-shared-partner-primitives.md` "Implementation Steps" step 3 (line 62) and "Success Criteria" (line 79); `phase-04-visual-qa-and-verification.md` "Success Criteria" (line 91)
- **Flaw:** The plan repeatedly claims `PartnerStatsCard` "clones" `PremiumStatStrip`'s API "1:1" and that the API is "proven, not speculative." Verification of `PremiumStatStrip.tsx` (113 lines, full read) shows the actual `PremiumStatItem` API is `{ label, value, unit?, highlight?, onClick?, trend? }` — period. No `valueFormat`, no `icon`, no `sparkline`. The plan invents three extra props and calls them "proven."
- **Failure scenario:** Phase 4 ships a "parity check" gate (`phase-04...md:91`) that cannot pass — the parity target doesn't have the props the gate requires. Implementer either (a) widens the gate to make it pass (defeating the purpose of the check), or (b) ships Phase 2 with extra props and records a "parity failure" against admin even though admin is correct. The validation log's claim of 17/18 verified claims is also inaccurate if this one was counted as verified.
- **Evidence:**
  - Actual `PremiumStatItem` interface: `frontend/src/components/admin-dashboard/PremiumStatStrip.tsx:5–14` — fields are `label`, `value`, `unit?`, `highlight?`, `onClick?`, `trend?: { value: string; positive: boolean }`. No `icon`, no `sparkline`, no `valueFormat`.
  - `PremiumStatStripProps` adds `items`, `isLoading`, `className`, `wrap`, `icon` — `icon` here is for the whole strip, not per-card (`PremiumStatStrip.tsx:23–28`).
  - Plan's claim of "valueFormat?: 'number' | 'currency'" as part of the cloned API: `phase-02...md:62`.
  - Plan's "API parity" success criterion: `phase-04...md:91` and `phase-02...md:79`.
- **Suggested fix:** Either (a) define `PartnerStatsCard`'s API as "inspired by" `PremiumStatStrip` plus three NEW partner-only props (and remove the "parity" gate from Phase 4), or (b) actually clone the API exactly and defer `icon`/`sparkline`/`valueFormat` to a separate ticket. Stop calling speculative props "proven."

---

## Finding 5: Phase 3 "migration" target list is dead code — the pages already use `@/components/shared/*`, not the listed `partner-*` files

- **Severity:** High
- **Location:** `phase-03-partner-route-migration.md` "Related Code Files → Modify (component groups)" (lines 49–57) and "Implementation Steps" step 9 (lines 92–96)
- **Flaw:** Phase 3 lists `partner-employees/PartnerEmployees{Header,Filters,Stats,SummaryStats,List,Table}.tsx` and equivalent for projects/timesheet as migration targets ("consolidate into primitives, then delete"). Grep shows ZERO external importers of any `Stats`, `Header`, or `Filters` file in `partner-employees/`, `partner-projects/`, or `partner-timesheet/`. The actual pages import from `@/components/shared/PageHeader`, `@/components/shared/InlineStatStrip`, `@/components/shared/SearchBar`, `@/components/shared/FilterPill`, `@/components/timesheet/*`, `@/components/payroll/*` — not the per-page `partner-*` components at all.
- **Failure scenario:** Phase 3 implementer spends hours refactoring `PartnerEmployeesStats.tsx` into `PartnerStatsCard` and then deletes it, only to discover (a) nothing imported it, so no visible UI changed, and (b) the actual employee stats come from `InlineStatStrip` in `pages/partner/EmployeesPage/index.tsx`, which the plan never mentions. The acceptance criterion "All five partner routes consume PartnerStatsGrid" is unverifiable because the routes don't currently consume a `*Stats*` partner component to migrate from. Net effect: most of Phase 3's listed work is no-op churn, and the real migration surface (`pages/partner/*/index.tsx` direct markup + `@/components/shared/*` + `@/components/timesheet/*`) is undocumented.
- **Evidence:**
  - Zero external consumers of `PartnerEmployeesStats`: `grep -rn "PartnerEmployeesStats" src` returns only `partner-employees/PartnerEmployeesStats.tsx:14,25` (its own definition). Same for `PartnerEmployeesSummaryStats`, `PartnerEmployeesHeader`, `PartnerEmployeesFilters`, `PartnerProjectsStats`, `PartnerTimesheetStats`.
  - Real imports: `frontend/src/pages/partner/EmployeesPage/index.tsx:5–8` (`PageHeader`, `InlineStatStrip`, `SearchBar`, `FilterPill` from `@/components/shared/*`).
  - `frontend/src/pages/partner/TimesheetsPage/index.tsx:4–8` (`TimesheetFilters`, `TimesheetListTable`, `TimesheetMobileList`, `TimesheetProvider` from `@/components/timesheet/*`).
  - Plan's deletion target list: `phase-03...md:52–57`.
- **Suggested fix:** Phase 3's first step must be a "current-state audit": grep each page's actual imports, list the REAL styling sources (shared components, inline markup), and write the migration plan from that — not from the `partner-*` folder contents. Most of the listed `partner-*` files should be deleted as dead code in Phase 3 step 9 BEFORE migration work begins, not after.

---

## Finding 6: Plan does not address `data-theme="congtruong"` hardcoded on shared sheets/modals used by partner

- **Severity:** High
- **Location:** `plan.md` "Preserve" (lines 62–68); `phase-01...md` step 7 sidebar refactor (lines 79–90)
- **Flaw:** Partner shares `UserProfileSheet`, `ChangePasswordModal`, and `NotificationSheet` with admin and login. All three hardcode `data-theme="congtruong"` (the admin daisyUI theme). If Phase 1's `[data-partner-ui]` token system tries to restyle these shared surfaces, partner and admin themes will collide inside the same DOM subtree.
- **Failure scenario:** A partner user opens the profile sheet. The sheet renders with `data-theme="congtruong"` and the daisyUI themeRoot selector `:where([data-admin-ui], [data-employee-ui])` — note `[data-partner-ui]` is NOT in `themeRoot`. The sheet content uses daisyUI `ct-btn`, `ct-input` classes that resolve against the admin theme's `--btn-focus-scale`, `--rounded-box`, etc. If Phase 1 Path B extends `themeRoot` to include `[data-partner-ui]`, then admin's `--primary: #08783e` becomes the active daisyUI primary inside partner's portal sheet — breaking any partner-specific accent the plan introduces. Worse, the sheet's CSS variables bleed back into the partner surface if the partner layout is a DOM ancestor.
- **Evidence:**
  - Shared sheet uses admin theme: `frontend/src/components/sheets/UserProfileSheet.tsx:198` (`data-theme="congtruong"`).
  - Partner sidebar wires that sheet: `frontend/src/components/PartnerSidebar.tsx:41,256`.
  - daisyUI themeRoot: `frontend/tailwind.config.ts:88` (`":where([data-admin-ui], [data-employee-ui])"`).
  - Login also uses `congtruong`: `frontend/src/pages/Login.tsx:209,226` (so the theme is shared across surfaces, not partner-owned).
- **Suggested fix:** Either (a) accept that partner-shared sheets remain admin-themed (and add this to "Preserve" + Phase 4 isolation check asserts these surfaces are NOT touched by `[data-partner-ui]`), or (b) parameterize the shared sheets with a `theme` prop and thread partner's theme through — but that is a much larger change than the plan's "no contract changes" scope, so option (a) is the only safe path. Either way, the plan must explicitly call this out.

---

## Finding 7: Path B (daisyUI `themeRoot` expansion) has undocumented privilege-style risk — partner tokens would override admin daisyUI inside shared sheets

- **Severity:** High
- **Location:** `phase-01-partner-design-foundation.md` step 7 "Path B" (line 81) and "Risk Assessment" (lines 116–117); `plan.md` Key Decision #1 (line 72)
- **Flaw:** Plan's Path B proposes extending `themeRoot` to `:where([data-admin-ui], [data-employee-ui], [data-partner-ui])`. CSS cascade: a `[data-theme="congtruong"]` element inside a `[data-partner-ui]` ancestor would receive BOTH admin daisyUI variables (via `data-theme`) AND partner theme-root variables (via `data-partner-ui` ancestor). Order-of-declaration in `tailwind.config.ts` decides which wins; daisyUI's compiled output is not order-deterministic across surfaces.
- **Failure scenario:** Partner opens the shared `ChangePasswordModal` (rendered with `data-theme="congtruong"` but DOM-descendant of `<div data-partner-ui>`). Some daisyUI `ct-*` classes now resolve `--primary` from the partner theme (whichever wins the cascade), producing admin buttons that look like partner buttons. Conversely, admin sheets rendered inside admin layout but pulled into a partner-routed modal (deep-link `?modal=...`) get the wrong theme. This is not "visual regression" — it is a trust-boundary-style leak where one surface's design system overrides another's, with no test catching it (Phase 4's screenshot diff is non-deterministic for cascade-order bugs).
- **Evidence:**
  - Plan proposes the expansion: `phase-01...md:81`.
  - `themeRoot` currently isolates admin/employee only: `frontend/tailwind.config.ts:88`.
  - Shared modal in partner DOM tree: `frontend/src/components/PartnerSidebar.tsx:232,256` + `frontend/src/components/modals/ChangePasswordModal.tsx:84`.
- **Suggested fix:** Mark Path B as forbidden, not "fallback." If daisyUI parity is required for the sidebar, render the partner sidebar in its own `<div data-theme="partner">` (a new sibling theme entry in `tailwind.config.ts:13–77`) and apply `data-partner-ui` only outside the sidebar — never join `themeRoot`. Document this constraint in the plan's Risk Controls.

---

## Finding 8: "Translate, don't transplant" mitigates XSS for Tailwind classes but not for Tailkit snippet pastes containing arbitrary HTML/JSX

- **Severity:** Medium
- **Location:** `plan.md` Key Decision #5 (line 76); `phase-02-shared-partner-primitives.md` step 1 (line 60) and step 13 "Story-free visual smoke" (line 72)
- **Flaw:** The plan says implementers will read Tailkit code via `mcp__tailkit__get_component_code` and "translate" it. The mapping table only covers color classes. Nothing in the plan addresses what to do if a Tailkit snippet contains inline event handlers (`onClick={...}`), inline `style={{ ... }}`, `dangerouslySetInnerHTML`, an `<a href={userInput}>`, or an `<iframe>`/`<svg>` with attacker-controlled attributes. "Translate, don't transplant" is a stylistic guideline, not a security control.
- **Failure scenario:** An implementer translating `a-c-empty-states-05` (which the plan explicitly selects, `plan.md:106`) copies a Tailkit `<svg>` with embedded text or an icon library call that accepts a `src`/`href` from component props. Partner route passes user-controllable data (project name, employee name) into that prop. Result: stored XSS inside the partner surface — and because the partner layout shares the same origin/cookies as admin (`ProtectedRoute` uses one `useAuth` context), an XSS in `/partner/projects` can pivot to fetch `/admin/users` with the user's session cookie.
- **Evidence:**
  - Plan selects Tailkit patterns but provides no XSS checklist: `plan.md:101–107`.
  - Plan's mapping table is color-only: `phase-01...md:60–75`.
  - Same-origin session: `frontend/src/components/ProtectedRoute.tsx:7–10` (one role enum covers admin/partner/employee/adv_partner — single auth context, single cookie).
  - No `dangerouslySetInnerHTML` in current partner code (verified clean), but the plan introduces no rule to prevent one being added during translation.
- **Suggested fix:** Phase 2 step 1 must add a "Tailkit translation security checklist": (1) no `dangerouslySetInnerHTML`, (2) no inline `style` with interpolated user input, (3) all `href`/`src` from props must be validated against an allowlist, (4) SVG content from Tailkit must be inlined as static JSX only, (5) Phase 4 grep for `dangerouslySetInnerHTML|innerHTML|eval(` in `partner-ui/` and partner pages must return zero matches.

---

## Finding 9: Phase 3 deletion policy has no audit gate for route-guard / role-check side effects

- **Severity:** Medium
- **Location:** `phase-03-partner-route-migration.md` "Delete" (line 58) and step 9 (lines 92–96); `phase-04-visual-qa-and-verification.md` Success Criteria (line 92)
- **Flaw:** Deletion safety check is "grep for external importers." That catches direct imports but NOT: (a) dynamic `lazy(() => import(...))` paths, (b) `modal-registry-auto.ts` entries that reference a deleted component by string ID, (c) test files that import by type only, (d) `route.element` references in `App.tsx` that wire a deleted component as a route target.
- **Failure scenario:** Phase 3 deletes `PartnerEmployeesSummaryStats.tsx` (currently zero static importers — verified). What the static grep misses: a future test in `*.test.tsx`, a `modal-registry-auto.ts` entry that references the component by string name for deep-linking, or a `route.element` lazy import added later. After deletion, `tsc --noEmit` may pass (if the references were string-based) but runtime throws `Failed to fetch dynamically imported module` on a partner deep-link, breaking auth-protected navigation for partner users.
- **Evidence:**
  - Plan's deletion check: `phase-03...md:95` (`grep -rln "<OldComponentName>" src`).
  - Modal registry exists and is consumed by partner: `frontend/src/components/PartnerSidebar.tsx:43–44` (`useModalNavigation`, `MODAL_IDS`).
  - Auto-registry file: `frontend/src/lib/modal-registry-auto.ts` exists (string-keyed).
  - Lazy routes use string dynamic imports: `frontend/src/App.tsx:54–64` (`lazy(() => import("./pages/..."))`).
- **Suggested fix:** Phase 3 step 9 must run four checks before any delete: (1) static `grep -rln`, (2) `grep -rln "ComponentName" src/lib/modal-registry-auto.ts src/constants/modalRegistry*`, (3) `grep -rln "import(.*ComponentName" src` (dynamic imports), (4) `tsc --noEmit` after a temporary rename to catch type-only consumers. Phase 4 must grep `modal-registry-auto.ts` for any reference to deleted component names.

---

## Finding 10: `MobileBottomNav` shared-file edit risk is asserted safe but the partner branch identifier is not specified — easy to edit the wrong branch

- **Severity:** Medium
- **Location:** `phase-01-partner-design-foundation.md` step 8 (line 90); "Risk Assessment" (lines 110–111)
- **Flaw:** Plan says "find the partner-specific conditional (likely keyed off `groups === PARTNER_NAV_GROUPS` or a `role` prop) and swap its tokens." The implementer must locate the partner branch in a shared file that also renders admin (with `ADMIN_NAV_GROUPS`, `ADMIN_MORE_ITEMS`) and adv-partner (with `ADV_PARTNER_NAV_GROUPS`). The plan does not specify the exact branching key, does not require a diff-gate proving admin/employee branches are byte-unchanged, and Phase 1's only safety net is "Diff after edit."
- **Failure scenario:** Implementer keys off `groups.length === 4` (partner has 4 items) instead of identity with `PARTNER_NAV_GROUPS`. Admin's `ADMIN_NAV_GROUPS` also has 3 items, but `moreItems` adds 10 more — an implementer who misreads the branch swaps tokens on the admin mobile nav too. Phase 4's "screenshot /admin unchanged" catches it only if someone remembers to look; nothing in the build prevents the regression. If admin's mobile nav loses its daisyUI styling, admin users on phones lose navigation affordances.
- **Evidence:**
  - Shared file with three role branches: `frontend/src/layouts/AdminLayout.tsx:28–50` (three `NavGroup` arrays) + `AdminLayout.tsx:116–118` (conditional groups/moreItems).
  - Partner layout passes its own constant: `frontend/src/layouts/PartnerLayout.tsx:12–17,45`.
  - Plan's vague branch-finding instruction: `phase-01...md:90`.
- **Suggested fix:** Phase 1 step 8 must (a) name the exact branching key (read `MobileBottomNav.tsx` first and cite the line), (b) require a pre-edit `git diff` baseline of `MobileBottomNav.tsx` and a post-edit diff that ONLY touches the partner branch, (c) add a Vitest snapshot of the admin and adv-partner `MobileBottomNav` output before and after the change as a regression gate.

---

Total findings: 10 (3 Critical, 5 High, 2 Medium)
