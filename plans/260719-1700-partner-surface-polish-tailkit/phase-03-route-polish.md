---
phase: 3
title: "Route Polish"
status: pending
priority: P2
dependencies: [1, 2]
---

# Phase 3: Route Polish

## Overview

Execute the Phase 2 audit work list. For each in-scope partner route (4 desktop + 4 mobile = 8 files): apply the recommended `components/shared/*` primitive adoptions, fix visual drift using Tailkit references translated to project tokens, and ship desktop + mobile in the same change. After all route polish is done, delete the confirmed dead-code files per-file (folders removed only when every file passes the three anchored greps). Mobile Payment History wiring is OUT (v2.1 H2 — see Key Decisions).

## Requirements

- **Functional:**
  - Every drift identified in `reports/phase-02-audit.md` is resolved.
  - Desktop and mobile variants of each route are updated in the same commit/step.
  - Dead-code folders (`components/partner-{employees,projects,timesheet}/`) are deleted only for files where all three Phase 2 greps returned zero.
  - After deletion, `tsc --noEmit` passes (no orphaned imports).
- **Non-functional:**
  - No edits to `components/shared/*` or `components/payroll/*` without a documented cross-surface consumer check.
  - All adopted Tailkit patterns are translated to project tokens (no `secondary-*`, `emerald-*`, `dark:` literals in committed code).
  - URL state, deep-linked sheets, mutations, Vietnamese copy, and `SectionErrorBoundary` wrap are preserved.

## Architecture

Phase 3 works file-by-file, not primitive-by-primitive. Each route change is a self-contained diff that can be reviewed independently. Dead-code deletion happens in a final sweep after all routes are polished, so reviewers see "polish first, cleanup second."

## Related Code Files

- **Modify (polish targets, per Phase 2 audit):**
  - `frontend/src/pages/partner/{Dashboard,Projects,Employees,Timesheets}Page/index.tsx`
  - `frontend/src/pages/mobile/partner/{Dashboard,Projects,Employees,Timesheets}Page/index.tsx`
  - `frontend/src/styles/partner.css` — populate with actual visual rules under `html.partner-route-active` selectors.
- **Modify (shell, only if Phase 2 audit finds drift):**
  - `frontend/src/components/PartnerSidebar.tsx` — visual token tweaks only (NO structural rewrite; it's already shadcn-only).
  - `frontend/src/components/MobileBottomNav.tsx` — partner-branch token tweaks only.
- **Modify (optional, Open Question 1):** ~~mobile Payment History wiring~~ — **REMOVED v2.1 (H2)**: the mobile file uses different service methods than desktop (data divergence, not presentation swap). Out of scope entirely.
- **Delete (final sweep, post-audit confirmation, v2.1 per-file not per-folder):**
  - Each individual file in `components/partner-employees/`, `partner-projects/`, `partner-timesheet/` that Phase 2 confirmed orphaned via three anchored greps. Folders are removed only when every file in them passes; if even one file has an importer, that file stays and the folder remains.
  - **AGENTS.md cleanup (v2.1 M2 fix):** after deletion, grep `find frontend/src -name AGENTS.md` and update any file that references the deleted paths (per v2 red-team Finding 7: `pages/partner/AGENTS.md:50-52`, `components/AGENTS.md:50-52`, `hooks/AGENTS.md:59-60`, `config/AGENTS.md:25-26,38-39`).
- **Read-only references:**
  - `reports/phase-02-audit.md` — the work list.
  - Tailkit MCP snippets cited in the audit.

## Implementation Steps

1. **Polish `/partner/dashboard` (desktop + mobile together).**
   - Apply audit recommendations.
   - Preserve `PartnerEmployeeListSheet` and `PartnerWorkforceOverviewCard` (these are LIVE, imported by the desktop page).
   - Preserve recharts usage in `PartnerWorkforceOverviewCard`.
2. **Polish `/partner/projects` (desktop + mobile together).**
   - Apply audit recommendations. Both pages already use `shared/PageHeader` / `shared/MobilePageHeader` — drift expected to be minimal.
3. **Polish `/partner/employees` (desktop + mobile together).**
   - Desktop already uses `ResponsiveTable`, `InlineStatStrip`, `SearchBar`, `FilterPill` — verify audit's drift findings before changing anything.
   - Mobile uses `EmployeeMobileCard`, `EmployeeEmptyStates` from `components/employees/` — these are shared with admin; edit only if audit confirms partner-specific drift, and only via props/className, not structural rewrite.
4. **Polish `/partner/timesheet` (desktop + mobile together).**
   - Desktop imports heavily from `components/timesheet/*` (shared with admin). Audit must distinguish partner-only drift from shared-component issues.
   - If a `components/timesheet/*` edit is required, document the admin consumer and run the cross-surface screenshot check in Phase 4.
5. **Populate `partner.css`.**
   - Add actual visual rules under `html.partner-route-active` selectors based on what the polish work needed.
   - If no rules were needed (because `components/shared/*` already covers everything), `partner.css` may stay near-empty — that is a valid outcome. Document it.
6. **Shell visual tweaks (only if Phase 2 audit found drift).**
   - `PartnerSidebar`: token-level changes only (e.g., hover color, active indicator). No structural rewrite. Verify against admin sidebar visually.
   - `MobileBottomNav` partner branch: same constraint.
7. **Dead-code deletion sweep (v2.1 per-file, not per-folder).**
   - Re-run Phase 2's three anchored greps for each candidate file immediately before deletion (code may have changed during polish).
   - Delete only files with zero hits on all three greps. If any file has an importer, KEEP that file and document the discrepancy — do not delete the folder just because most files passed.
   - Record each deletion (and each keep-with-reason) in `reports/phase-03-deletion-log.md` with file path + grep evidence.
   - After deletion: update AGENTS.md files that reference deleted paths (see Modify list above).
8. ~~**Optional: wire mobile Payment History.**~~ **REMOVED v2.1 (H2).** Mobile file uses different service methods than desktop (`useInfinitePaymentHistories` + `useExportPaymentHistories` vs `useBankTransferHistories`). Wiring it changes what data a partner sees on mobile vs desktop — that's a product decision, not polish. Out of scope.
9. **Behavioral spot-checks per polished route.**
   - URL query state survives a page reload.
   - Deep-linked sheet (e.g., employee detail via modal registry) opens and returns correctly.
   - Mutation feedback (toast / sonner) still fires.
   - Mobile ↔ desktop switching preserves scroll where current behavior does.
10. **Accessibility spot-check per route.**
    - Keyboard tab-through; visible focus rings; logical order.
    - VoiceOver on one representative route.
    - 200% browser zoom; no clipped content.
    - `prefers-reduced-motion`; no jarring animation.

## Success Criteria

- [ ] Every drift item in `reports/phase-02-audit.md` is resolved; the audit report is annotated with resolution references.
- [ ] Desktop and mobile variants of each polished route are in the same change (verified by `git log --stat`).
- [ ] `reports/phase-03-deletion-log.md` lists every deleted file with the three anchored grep results that justified deletion; any kept file is listed with its importer.
- [ ] After deletion: `grep -rln "from \"@/components/partner-" frontend/src --include="*.tsx" --include="*.ts"` returns zero hits ONLY for files actually deleted (live files in kept folders are expected to remain).
- [ ] After deletion: `tsc --noEmit` passes.
- [ ] AGENTS.md files updated to remove references to deleted paths.
- [ ] `partner.css` contains no `:root` selectors and no selectors that match outside the partner scope (`html.partner-route-active` for base/global, `[data-partner-ui]` for presentation/portals).
- [ ] **Tailkit literal-class ban is DELTA-ONLY (v2.1 C2 fix):** `git diff main -- frontend/src/pages/partner frontend/src/pages/mobile/partner frontend/src/styles/partner.css frontend/src/components/PartnerSidebar.tsx frontend/src/layouts/PartnerLayout.tsx | grep -E "^\+.*(secondary-[0-9]|emerald-[0-9]|orange-[0-9]|sky-[0-9]|violet-[0-9]|slate-[0-9]|dark:)"` returns zero (no NEW literals introduced). Pre-existing matches (37 today) are out of scope.
- [ ] URL state, deep-linked sheets, mutations, Vietnamese copy, and `SectionErrorBoundary` wrap are preserved across all polished routes.
- [ ] `PartnerEmployeeListSheet` and `PartnerWorkforceOverviewCard` (live `partner-dashboard/` components) are untouched in structure; recharts still renders.
- [ ] **Payment History untouched (v2.1 H2/H3):** `App.tsx` partner route for `/partner/timesheet/payment-history` is byte-unchanged; `BankTransferHistoryPageContent` variant logic is byte-unchanged.

## Risk Assessment

- **Risk:** Phase 2 audit was incomplete; Phase 3 deletion removes a file that IS imported somewhere.
  **Mitigation:** Step 7 re-runs the three anchored greps immediately before deletion. If a new importer appeared during polish, the file is kept and the discrepancy is documented.
- **Risk:** A `components/shared/*` edit during polish regresses an admin or employee consumer.
  **Mitigation:** Phase 2 Step 6 maps every consumer (named + barrel); Phase 3 touches a shared primitive only when partner is the sole consumer, OR the change is additive (new optional prop). Phase 4 runs the cross-surface screenshot check. **Circuit breaker (v2.1 Security #3 fix):** if a `shared/*` edit is required and non-partner consumers exist, the change MUST be additive-only (new optional prop with partner-scoped default). Non-additive edits to shared primitives require a separate plan.
- **Risk:** Polish introduces a NEW Tailkit literal class that slips through review.
  **Mitigation:** Success criterion uses `git diff` to check only NEW additions, not pre-existing baseline (v2.1 C2 fix — the baseline-zero grep was unachievable).
- **Risk:** Per-file deletion policy creates a confusing half-empty folder structure.
  **Mitigation:** Phase 3 deletion log explicitly documents which files were kept and why; reviewers can see the structure decision.
