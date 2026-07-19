# Red Team v2 Review — Assumption Destroyer (Fact-Checker Standard)

Scope: `/Users/dev/Documents/projects/payroll/plans/260719-1700-partner-surface-polish-tailkit/` (plan.md + 4 phase docs).
Method: every load-bearing claim sampled against the codebase via grep/read. Citations are `path:line`.

---

## Finding 1: Plan's own Success Criterion grep is FALSE-POSITIVE on the live codebase — Phase 3/4 ban will fail before any code is written

- **Severity:** Critical
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (the grep was added by v2, v1 didn't have it)
- **Location:** Phase 3 Success Criteria bullet (`phase-03-route-polish.md:95`) and Phase 4 Step 2 (`phase-04-verification.md:39`)
- **Flaw:** The mandated grep `grep -rn -E "(secondary-[0-9]|emerald-[0-9]|orange-[0-9]|sky-[0-9]|violet-[0-9]|slate-[0-9]|dark:)" frontend/src/pages/partner frontend/src/pages/mobile/partner ...` is asserted to "return zero matches" as a gate. The CURRENT baseline already produces dozens of matches.
- **Failure scenario:** Phase 4 Step 2 runs the grep. It returns ~30 hits in `pages/partner/DashboardPage`, `pages/partner/ProjectsPage`, `pages/partner/EmployeesPage`. The gate reports failure. Either the engineer removes legitimate, working Tailwind classes (regressing visuals) or the gate is waived (gate becomes non-functional — same defect class as v1's vacuous Vitest gate).
- **Evidence:** `frontend/src/pages/partner/DashboardPage/index.tsx:90,104,106,120,199,200,258,337,338,358,360,361,366,367,372,396`; `frontend/src/pages/partner/EmployeesPage/index.tsx:45,46,47,403,404,540`; `frontend/src/pages/partner/ProjectsPage/index.tsx:39,40,45,46,73,75,76,135`. Confirmed via running the exact grep from the plan: matches against `emerald-*`, `slate-*`, `sky-*`, `violet-*`, `orange-*` in 3 of the 4 in-scope desktop files. Plan claims these tokens are "v3.4-valid" (`plan.md:36`) — they ARE valid Tailwind defaults, but the success criterion still bans them, contradicting the "valid Tailwind" framing.
- **Suggested fix:** Either (a) scope the grep to NEW code only (diff-aware: `git diff` against pre-plan baseline), or (b) drop the per-color ban and replace with: "no committed code introduces NEW Tailkit-literal classes that the existing partner pages don't already use; track additions via diff inspection." Today's literal ban cannot pass against today's baseline.

---

## Finding 2: Plan's "Dead-code three-grep" verification is structurally insufficient — modal-registry uses filesystem globbing, not string lookup

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (Phase 2 Step 4 was added as a v2 safety control in response to v1's "delete live code" risk)
- **Location:** Phase 2 Step 4 (`phase-02-audit-and-primitives-parity.md:62-66`), Risk Assessment (`phase-02-...md:83-84`), Phase 3 Step 7 (`phase-03-route-polish.md:68-71`), Phase 4 Step 3 (`phase-04-verification.md:43-44`)
- **Flaw:** The third "modal-registry string grep" is `grep -rln "partner-{folder}" frontend/src/constants frontend/src/lib`. The actual modal registry (`modal-registry-auto.ts`) uses `import.meta.glob('/src/components/**/*{Sheet,Modal}.tsx', ...)` to discover modals by **filesystem globbing**, then loads each module's `modalConfig` export at runtime. There are zero string references in `constants/` or `lib/` for ANY modal folder — that grep returns empty for *every* folder, not just the dead ones.
- **Failure scenario:** The dead-code deletion safety check is structurally vacuous: it always returns zero for the modal-registry dimension regardless of whether the folder is live. If a future engineer deletes a folder that DOES contain a `*Sheet.tsx` with a `modalConfig` export (or a future contributor adds one to a partner-* folder), the plan's gate will still pass and the runtime will fail with a broken deep-link.
- **Evidence:** `frontend/src/lib/modal-registry-auto.ts:10-12`: `const modalModules = import.meta.glob('/src/components/**/*{Sheet,Modal}.tsx', { eager: false })`. Plan's grep (`phase-02-...md:65`): `grep -rln "partner-{folder}" frontend/src/constants frontend/src/lib`. Re-running that grep for the live `partner-dashboard/` folder also returns zero, proving the grep can't distinguish live from dead.
- **Mitigating fact (downgrade from Critical):** A direct filesystem check of the three target folders shows no `*Sheet.tsx`/`*Modal.tsx` files and no `modalConfig` exports today (`find ... \( -name "*Sheet.tsx" -o -name "*Modal.tsx" \)` and `grep -l modalConfig` both empty). So the specific deletion is safe *by accident*, not by verification design.
- **Suggested fix:** Replace the modal-registry grep with a filesystem assertion: `find frontend/src/components/partner-{folder} -name "*Sheet.tsx" -o -name "*Modal.tsx" | xargs grep -l "modalConfig"` must return empty. Document that the current registry is glob-based, so static string greps in `constants/`/`lib/` are insufficient as a general safety check.

---

## Finding 3: Plan survey misreports the desktop `/partner/timesheet` imports — `TimesheetsExportDialog` and `BCCUploadModal`/`UploadHistorySheet` are silently dropped

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v1 rejection cause #1 was a survey error; v2 repeats the pattern in a different file)
- **Location:** Plan `plan.md:64` "Corrected Codebase Survey" row for `/partner/timesheet`
- **Flaw:** Plan states the page imports "`components/timesheet/{TimesheetFilters, TimesheetListTable, TimesheetMobileList, TimesheetProvider, EditRequestTable}`, `components/transaction/ExportSaoKeDialog`, `components/payroll/PaymentHistorySheet`". The actual imports also include `TimesheetsExportDialog` (different file from `ExportSaoKeDialog`), `BCCUploadModal`, and `UploadHistorySheet` from `components/timesheet/`. Two distinct export dialogs coexist in this file.
- **Failure scenario:** Phase 2 audit row is wrong; Phase 3 polish may attempt to "consolidate" two dialogs that have distinct behavior, or miss that `BCCUploadModal` and `UploadHistorySheet` (which carry their own deep-linked state) exist. Phase 3 Step 9's "deep-linked sheet opens and returns correctly" check never enumerates these three modals, so a regression in `BCCUploadModal` wiring is not specifically tested.
- **Evidence:** `frontend/src/pages/partner/TimesheetsPage/index.tsx:10-13` (`TimesheetsExportDialog`), `:14` (`ExportSaoKeDialog` — distinct from `:10`), `:25` (`BCCUploadModal`), `:26` (`UploadHistorySheet`). Plan survey (`plan.md:64`) lists only the first set + `ExportSaoKeDialog`, omitting `TimesheetsExportDialog`, `BCCUploadModal`, `UploadHistorySheet`.
- **Suggested fix:** Correct the survey row in `plan.md:64` to list all six `components/timesheet/*` imports + both export dialogs. Phase 3 Step 4 must enumerate `BCCUploadModal` and `UploadHistorySheet` as preservation targets alongside `PaymentHistorySheet`.

---

## Finding 4: Plan mis-classifies `BankTransferHistoryPageContent` as "shared re-export that restyling would regress admin" — the component is variant-aware, not byte-shared

- **Severity:** High
- **v1 resolution check or NEW v2 finding:** Reframes v1 rejection finding #10 — v2's resolution is overstated
- **Location:** `plan.md:51` (v1 rejection #10 resolution), `plan.md:65` (survey), `plan.md:122` (Out of scope), `plan.md:147` (Key Decision #5), `phase-03-route-polish.md:75` ("byte-unchanged"), `phase-04-verification.md:93` ("visually identical")
- **Flaw:** The plan repeatedly asserts the desktop PaymentHistoryPage is a "5-line re-export" of `BankTransferHistoryPageContent` that is "SHARED WITH ADMIN" and would "regress admin" if restyled. The component is in fact a variant-aware component: `BankTransferHistoryPageContent({ variant = 'partner' })`, with `isAdmin = variant === 'admin'` branching internally. Admin and partner are NOT rendering byte-identical markup — they're rendering the same component with different variant logic.
- **Failure scenario:** Two scenarios:
  1. **Phase 4 Step 6 mismatch:** Phase 4 asserts `/partner/timesheet/payment-history` must be "visually identical to admin payment-history view". The current code already produces non-identical output because of the `isAdmin` branch. The check is unachievable as written — it will either be silently waived or block Phase 4 indefinitely.
  2. **Future regression risk:** Because the plan treats this file as "untouchable shared re-export," no Phase 2 audit row exists for it. If a partner-specific drift is found in the partner variant branch, the plan provides no path to fix it.
- **Evidence:** `frontend/src/components/payroll/BankTransferHistoryPageContent.tsx:279-289`: `interface BankTransferHistoryPageContentProps { variant?: 'admin' | 'partner'; }` and `{ variant = 'partner' }: ...` with `const isAdmin = variant === 'admin';`. Admin consumer (`pages/admin/PaymentHistoryPage/index.tsx:4`) passes `variant="admin"`; partner consumer (`pages/partner/PaymentHistoryPage/index.tsx:4`) passes nothing (defaults to `'partner'`). The "byte-unchanged" claim in `phase-03-route-polish.md:75` is tautologically true (Phase 3 isn't allowed to touch the file) but the "visually identical" check in `phase-04-verification.md:93` is achievable only by accident.
- **Suggested fix:** Rewrite Key Decision #5 and Phase 4 Step 6 to acknowledge the variant split. Replace "visually identical" with "admin variant unchanged vs. pre-plan baseline (screenshot-diff)". Add a Phase 2 audit row for the partner-variant branch of `BankTransferHistoryPageContent` so future partner-side drift has a remediation path.

---

## Finding 5: Phase 1's "cascade order — import after admin-daisy.css" is technically sound but the stated justification ("partner rules can override admin defaults") is unfalsifiable theater

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 added the cascade-order claim; v1 had a different bug)
- **Location:** `phase-01-foundation.md:58` ("Related Code Files"), `phase-01-foundation.md:72` (Step 4), `phase-01-foundation.md:90-91` (Risk Assessment mitigation)
- **Flaw:** The plan justifies placing `partner.css` after `admin-daisy.css` so "partner rules can override admin defaults if needed (they shouldn't — surfaces are isolated — but cascade order is defensible)." But `admin-daisy.css` selectors are scoped under `[data-admin-ui]` (verified: `frontend/src/styles/admin-daisy.css:1,8,25,29,30,35,40,50,...`) and `html.admin-route-active` (verified at `:393-409`). Partner's `html.partner-route-active` selectors can NEVER match the same DOM as admin's `[data-admin-ui]` selectors because `[data-admin-ui]` is only rendered when `isAdmin` (AdminLayout.tsx:109) and `admin-route-active` only exists on `/admin/*` and `/adv-partner/*` routes. There is zero CSS rule overlap by construction.
- **Failure scenario:** No functional bug. The cost is cognitive: a future contributor reads the cascade-order rationale, assumes there IS a real overlap risk, and either (a) invents partner rules they think they need to "win" the cascade against admin, or (b) adds an `!important` defensively. The justification primes the wrong mental model.
- **Evidence:** `frontend/src/styles/admin-daisy.css:1` (`[data-admin-ui] {`) — every admin selector is namespaced. `frontend/src/layouts/AdminLayout.tsx:101-102,109` confirms `admin-route-active` class and `data-admin-ui` attribute are admin-only. `index.css:5` (`@import './styles/admin-daisy.css';`) is the only admin import — no global rules leak. Partner's selectors (`phase-01-foundation.md:38-49`) use `html.partner-route-active` which is mutually exclusive with admin.
- **Suggested fix:** Delete the "override admin defaults" justification. State plainly: "Import order is irrelevant because partner selectors cannot match admin DOM. Place `partner.css` last for readability; cascade order has no functional role here."

---

## Finding 6: Open Question 1 (mobile Payment History wiring) deferred status makes Phase 4's 320/390px QA of `/partner/timesheet/payment-history` a content-free check

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 added the deferral; v1 didn't address mobile payment-history at all)
- **Location:** `plan.md:199` (Open Question 1, default "defer"), `plan.md:77` (survey: "NOT WIRED"), `phase-04-verification.md:58-59` (Step 6 visual QA), `phase-04-verification.md:91` (Success Criterion)
- **Flaw:** Phase 4 Step 6 + Success Criterion require visual QA of `/partner/timesheet/payment-history` at 320/390/768/1440 (the partner-route QA list at `:91` includes all four in-scope routes; Step 6 separately calls for screenshots). If Open Question 1 is deferred (the plan's default), the route renders the desktop `BankTransferHistoryPageContent` re-export on mobile viewports too — there is no mobile variant. Running mobile-viewport QA on a desktop component is either trivially "passes because there's nothing partner-mobile to test" or "fails because the desktop component overflows at 320px and the plan provides no remediation path."
- **Failure scenario:** Engineer captures 320px screenshot of `/partner/timesheet/payment-history`. Either: (a) desktop layout overflows horizontally at 320px → the Success Criterion "no horizontal scroll at 320" fails → no in-plan remediation because the restyle is explicitly Out of Scope (`plan.md:122`); or (b) the check is silently skipped because the route is "out of visual restyle scope," making Phase 4's "all four in-scope routes" claim misleading since the route is half-in/half-out.
- **Evidence:** `plan.md:77` documents the route is "NOT WIRED" on mobile and `App.tsx:282` (actual `App.tsx:282`) confirms `element={<PartnerPaymentHistoryPage />}` with no `ResponsivePage` wrapper. `plan.md:122` excludes the page from visual restyle. `phase-04-verification.md:91` nonetheless lists "all four in-scope partner routes" — payment-history is the fifth partner route and is implicitly swept in.
- **Suggested fix:** Make the scope contradiction explicit. Either (a) resolve Open Question 1 in the plan (wire mobile or formally exclude the route from Phase 4's mobile-viewport QA list with a documented reason), or (b) add a Phase 4 carve-out: "Payment History at 320/390px is not in scope if Open Question 1 is deferred; the route is documented as desktop-only and the mobile-viewport check is skipped with a recorded reason."

---

## Finding 7: Phase 3 Step 6 ("PartnerSidebar token-level changes only") is speculative — no token changes are enumerated, and the prerequisite ("only if Phase 2 audit finds drift") makes the step a phantom

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 introduced this step as part of "shell visual tweaks")
- **Location:** `phase-03-route-polish.md:65-67` (Step 6), `phase-03-route-polish.md:38-39` (Related Code Files condition)
- **Flaw:** Step 6 says "PartnerSidebar: token-level changes only (e.g., hover color, active indicator). No structural rewrite. Verify against admin sidebar visually." But: (a) the step is gated on "only if Phase 2 audit found drift" (Related Code Files line 37-38); (b) `plan.md:148` Key Decision #6 already concluded "PartnerSidebar is shadcn-only and uses `--partner-accent` ... admin sidebar is ALSO shadcn-only (no `ct-*`), so they're already structurally aligned"; (c) no specific token drift is enumerated anywhere in the plan. The step is conditional on a finding the plan's own Key Decision pre-judges as unlikely.
- **Failure scenario:** Phase 2 audit finds no sidebar drift (consistent with Key Decision #6). Phase 3 Step 6 becomes a no-op. The plan still lists `PartnerSidebar.tsx` and `MobileBottomNav.tsx` as Modify targets, leaving the Touchpoints and Related Code Files ambiguous about whether they're touched or not. Reviewers and engineers cannot determine scope from the document alone.
- **Evidence:** `phase-03-route-polish.md:37-39`: "Modify (shell, only if Phase 2 audit finds drift): PartnerSidebar.tsx, MobileBottomNav.tsx". `plan.md:148`: "they're already structurally aligned". No token drift enumerated in any phase doc. `plan.md:190` Touchpoints lists `PartnerLayout.tsx` but not PartnerSidebar — internally inconsistent with `phase-03-route-polish.md:38`.
- **Suggested fix:** Either delete Step 6 (the prerequisite makes it unreachable per Key Decision #6), or enumerate the specific drift hypothesis to test in Phase 2 (e.g., "Phase 2 must compare PartnerSidebar active-state color vs. AdminSidebar active-state color; if they diverge, Phase 3 changes the partner value to match"). Today the step has no actionable content.

---

## Finding 8: Plan's central premise ("reuse `components/shared/*` covers everything") is partly self-falsifying — Phase 3 Step 5 admits `partner.css` "may stay near-empty — that is a valid outcome," implicitly conceding the plan is largely a no-op

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 added this admission)
- **Location:** `phase-03-route-polish.md:64` (Step 5), `plan.md:34` (Overview), `phase-02-audit-and-primitives-parity.md:67` (Step 5 "Gap analysis ... default expectation: none")
- **Flaw:** The plan's hypothesis is: (1) `components/shared/*` already has every primitive partner needs, (2) the live partner pages already import from it, (3) Phase 2 will find "few or no" gaps. If all three are true, then Phase 3 produces: an empty `partner.css`, a classList effect, deletion of three folders, and nothing else. The plan acknowledges this (`phase-03-route-polish.md:64`) but doesn't reconcile it with the existence of a 4-phase plan, Phase 2's adoption-map ritual, and the Tailkit reference pass.
- **Failure scenario:** Phase 2 runs and confirms the expected no-drift baseline (consistent with Key Decision #1 and the survey showing all four routes already use `shared/PageHeader` etc.). Phase 3 becomes "delete three folders + add a classList effect." The 4-phase structure is then over-engineered: a 1-phase "delete dead code + add scope hook" plan would suffice. Worse, Phase 4's elaborate visual-QA grid (32 screenshots: 4 routes × 4 viewports) becomes verification of unchanged pages — pure ceremony that risks waiving real future regressions because the engineer trains on "passes that don't reflect actual change."
- **Evidence:** Survey table `plan.md:60-65` shows all 4 desktop routes and all 4 mobile routes already import from `components/shared/*`. `phase-02-audit-and-primitives-parity.md:67`: "Expected answer: none, because the inventory is large — but verify honestly." `phase-03-route-polish.md:64`: "If no rules were needed ... `partner.css` may stay near-empty — that is a valid outcome."
- **Mitigating fact:** The dead-code deletion and classList effect are real, valuable work; this finding is about plan *proportionality*, not incorrectness.
- **Suggested fix:** State the no-op scenario explicitly at the top of `plan.md` ("If Phase 2 confirms no drift, the deliverable is: `partner.css` skeleton + PartnerLayout effect + dead-code deletion"). Compress Phases 2+3+4 into a single execution phase that branches on Phase 2's outcome. Or: enumerate the *specific* drift hypotheses Phase 2 must falsify (e.g., "ProjectsPage header uses X, Dashboard uses Y — confirm match") so the audit has a concrete falsification target rather than a generic "find drift" mandate.

---

## Finding 9: Plan v2 "fact-check pass" claim (`plan.md:242`) cites "8 claims verified" but does not include the four high-risk claims this reviewer found false or imprecise on first pass

- **Severity:** Medium
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v2 added the consistency-sweep self-attestation)
- **Location:** `plan.md:242` ("v2 Fact-Checker pass: 8 claims verified ... All VERIFIED"), `plan.md:232-243` (Whole-Plan Consistency Sweep)
- **Flaw:** The self-attested "8 claims verified" list does not include: (a) the in-scope Tailwind literal-class ban (`plan.md:242` doesn't mention that the grep fails against current code — Finding 1 here), (b) the desktop `/partner/timesheet` import list (Finding 3 here), (c) `BankTransferHistoryPageContent` variant semantics (Finding 4 here), (d) the modal-registry grep sufficiency (Finding 2 here). All four are load-bearing for plan correctness and all four are wrong/imprecise. The "All VERIFIED" claim provides false confidence to downstream consumers (`/ck:cook`, exec review).
- **Failure scenario:** A downstream approver reads "All VERIFIED" and skips independent verification. Plan enters execution with three Critical/High incorrect claims (this report's Findings 1, 3, 4) baked into Success Criteria.
- **Evidence:** `plan.md:242`: "v2 Fact-Checker pass: 8 claims verified against codebase ... All VERIFIED." Compare to this report's Findings 1, 2, 3, 4 — all sampled and falsified on first pass with file:line citations.
- **Suggested fix:** Strike "All VERIFIED." Replace with: "8 claims verified, 4 claims [list] not yet verified — pending red-team v2 review." Do not merge self-attested verification with adversarial verification.

---

## Finding 10: Phase 1 Step 1 instruction "Read `AdminLayout.tsx:99-103`" cites lines that contain only the `useEffect` opening — the `ProtectedRoute` wrapper and `SidebarProvider` framing the plan asks the engineer to mirror are on different lines

- **Severity:** Low
- **v1 resolution check or NEW v2 finding:** NEW v2 finding (v1's line numbers were wrong; v2 corrected to a narrower but still imprecise range)
- **Location:** `phase-01-foundation.md:60,66` (Read-only reference + Step 1)
- **Flaw:** Plan tells the engineer to mirror `AdminLayout.tsx:99-103` and "match the pattern." Actual layout: lines 99-103 contain only the `useEffect`'s conditional + classList call (`AdminLayout.tsx:99-102` verified). The plan also instructs (Step 2) "Verify it's inside the component body, not inside the outer `PartnerLayout`" — but AdminLayout's effect is inside `AdminLayoutInner`-equivalent and is gated by `if (!isAdmin) return;` (AdminLayout.tsx:100). Partner has no equivalent role check (`PartnerLayout`'s `ProtectedRoute requiredRole="partner"` is the gate). The mirror is asymmetric and the line citation doesn't surface this.
- **Failure scenario:** Engineer copy-pastes the admin pattern including the `if (!isAdmin) return;` guard, but PartnerLayout has no `isAdmin` variable — TypeScript error, or engineer invents a guard that doesn't apply.
- **Evidence:** `frontend/src/layouts/AdminLayout.tsx:99-102`: `const { user } = useAuth(); const isAdvPartner = user?.role === 'adv_partner'; const isAdmin = user?.role === 'admin'; useEffect(() => { if (!isAdmin) return; ...`. `frontend/src/layouts/PartnerLayout.tsx` (entire file): no `useAuth`, no role check; the role gate is in the wrapping `<ProtectedRoute requiredRole="partner">`.
- **Suggested fix:** Add to Step 2: "AdminLayout gates the effect with `if (!isAdmin) return;` because AdminLayout hosts both admin and adv-partner roles. PartnerLayout is single-role (always `partner` via ProtectedRoute), so the guard is unnecessary — add the classList unconditionally."

---

## Total findings: 10

## v1 finding resolution verdict: PARTIALLY RESOLVED — v1's 10 findings are individually addressed (dead-code folder reframing, mobile route inclusion, shadcn-only sidebar, `cd backend` invocation, `:root` token location, classList portal pattern, redundant primitive deletion, stat-strip API correction, per-route table audit, PaymentHistoryPage exclusion). However, v2 introduces NEW defects: a self-falsifying Success Criterion (Finding 1), a structurally vacuous safety grep (Finding 2), and a survey misreport (Finding 3) that repeat the *class* of v1's root cause (survey-driven plans that under-verify against the codebase). Net: v1 resolved, v2 not yet ready.
