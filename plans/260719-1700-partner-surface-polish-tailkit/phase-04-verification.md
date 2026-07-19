---
phase: 4
title: "Verification"
status: completed
priority: P2
dependencies: [1, 2, 3]
---

# Phase 4: Verification

## Overview

Close the loop. Automated gates, visual QA across desktop + mobile + narrow viewports, cross-surface isolation screenshots, dead-code removal verification, and the mandatory Whole-Plan Consistency Sweep. The `make api-test` invocation uses `cd backend && make api-test` (not root — the root Makefile does not define this target).

## Requirements

- **Functional:** every acceptance criterion from `plan.md` is demonstrably met with captured evidence.
- **Non-functional:** zero unresolved contradictions across plan files; zero literal Tailkit classes; zero orphaned imports after Phase 3 deletion; cross-surface screenshots confirm no admin/employee/login regression.

## Architecture

Four verification passes in order: automated, visual, isolation, consistency. Evidence is appended to this file's Success Criteria section as it's captured.

## Related Code Files

- **Read-only verification targets:** every file touched in Phases 1–3.
- **Tooling:** `pnpm lint`, `tsc --noEmit`, `pnpm build`, `pnpm test:run` (if any partner specs exist), Playwright partner specs, `cd backend && make api-test`, `graphify update .`.

## Implementation Steps

1. **Automated pass.**
   - `cd frontend && pnpm lint`
   - `cd frontend && tsc -p tsconfig.json --noEmit`
   - `cd frontend && pnpm build`
   - `cd frontend && pnpm test:run -- partner` (note: v1 red-team found zero partner `*.test.tsx` specs — if still zero, document that this gate passes vacuously and rely on Playwright + visual QA instead)
   - `cd backend && make api-test` (NOT `make api-test` from repo root — that target does not exist there)
   - Capture pass/fail + output tail for each.
2. **Tailkit literal-class grep — DELTA-ONLY (v2.1 C2 fix).**
   - Pre-existing baseline (verified by v2 red-team): 37 Tailkit-literal matches across `pages/partner/{Dashboard,Employees,Projects}Page/index.tsx` today. These are OUT OF SCOPE.
   - Delta grep: `git diff main -- frontend/src/pages/partner frontend/src/pages/mobile/partner frontend/src/styles/partner.css frontend/src/components/PartnerSidebar.tsx frontend/src/components/MobileBottomNav.tsx frontend/src/layouts/PartnerLayout.tsx | grep -E "^\+.*((secondary|emerald|sky|violet|orange|slate)-[0-9]|dark:)"`
   - Expected: zero NEW additions introduced by this plan.
3. **Dead-code removal verification (v2.1 per-file).**
   - `find frontend/src/components/partner-employees frontend/src/components/partner-projects frontend/src/components/partner-timesheet -type f 2>/dev/null` — expected: either no such directories (all files deleted) OR only files explicitly kept per Phase 3 log.
   - `grep -rln "from \"@/components/partner-" frontend/src --include="*.tsx" --include="*.ts"` — expected: zero hits for deleted files; non-zero only for explicitly-kept live files (logged in `reports/phase-03-deletion-log.md`).
   - **Modal-registry grep (v2.1 M1 fix: scan all source, not just `constants/`+`lib/`):** `grep -rln "components/partner-employees\|components/partner-projects\|components/partner-timesheet" frontend/src --include="*.tsx" --include="*.ts"` — expected: zero hits except in the kept live files' own folders.
4. **Portal scope verification.**
   - Visit `/partner/dashboard` (desktop): `document.documentElement.classList.contains('partner-route-active')` returns `true`.
   - Visit `/admin/dashboard`, `/employee`, `/login`: returns `false` on each.
   - Open a partner sheet (e.g., `UserProfileSheet`): the Radix portal content should be reachable by partner CSS rules via the `html.partner-route-active` scope.
5. **Visual QA — in-scope partner routes.**
   - For each of `/partner/{dashboard,projects,employees,timesheet}`:
     - Desktop 1440px screenshot.
     - Tablet 768px screenshot.
     - Mobile 390px screenshot.
     - Mobile 320px screenshot.
     - Verify: no horizontal scroll, no clipped controls, no nested-scroll hazards, no tap targets <44px on mobile.
     - Verify: status pills have icon + text (color is not the sole signal).
     - Verify: tabular numerals on every financial figure.
6. **Visual QA — Payment History (v2.1 H3 fix: variant-aware, NOT byte-shared).**
   - `BankTransferHistoryPageContent` is variant-aware (`variant?: 'admin' | 'partner'`, line 280). Partner and admin variants already differ by design.
   - Screenshot `/partner/timesheet/payment-history` (renders `variant='partner'` by default) and the admin payment-history route (renders `variant='admin'`). Confirm BOTH variants still render correctly AND that the `variant=` prop wiring is byte-unchanged from pre-plan state.
   - Grep `git diff main -- frontend/src/components/payroll/BankTransferHistoryPageContent.tsx` — expected: zero changes (this file is out of scope).
7. **Cross-surface isolation check.**
   - Capture fresh baselines of `/admin/dashboard`, `/admin/employees`, `/employee` (one route), `/login`.
   - Compare to pre-plan state (if no prior baseline exists, this capture becomes the baseline; document that the comparison is "first baseline" not "regression check").
   - DOM check: `document.querySelector('[data-partner-ui]')` and `document.documentElement.classList.contains('partner-route-active')` both return null/false on each non-partner route.
8. **Accessibility spot check.**
   - Keyboard tab-through one representative polished partner route.
   - VoiceOver pass on the same route.
   - 200% zoom on the same route.
   - `prefers-reduced-motion` toggle; confirm no jarring animation.
9. **Cross-surface consumer regression check (if any `shared/*` was edited).**
   - For each `components/shared/*` file touched in Phase 3: visit an admin or employee route that consumes it. Screenshot. Confirm no visual regression.
10. **Whole-Plan Consistency Sweep (mandatory).**
    - Re-read `plan.md` and every `phase-*.md`.
    - Confirm: no stale references to v1's deleted folders as migration targets; no claims that admin sidebar uses daisyUI; no `make api-test` (root) references; no `[data-partner-ui]`-scoped token claims that contradict `--partner-accent` living at `:root`.
    - Append `### Whole-Plan Consistency Sweep` to `plan.md`'s Red Team Review section.
11. **Graph refresh.**
    - `graphify update .` from repo root. Confirm exit 0.
12. **Verification report.**
    - Append captured evidence to this file's Success Criteria.
    - Note deviations, files deleted, follow-up issues, and the answer to Open Question 1 (mobile Payment History wiring).

## Success Criteria

- [ ] `pnpm lint` passes (tail captured).
- [ ] `tsc --noEmit` passes.
- [ ] `pnpm build` passes.
- [ ] `cd backend && make api-test` passes (note: invoked from `backend/`, not repo root).
- [ ] **Vitest partner specs: either pass, OR documented as vacuous (v2.1 H5 fix).** Pre-verification: `find frontend/src -name "*.test.tsx" -path "*partner*"` — if zero, document that this gate is vacuous and rely on Playwright + visual QA.
- [ ] **Playwright partner specs: either pass, OR documented as vacuous (v2.1 H5 fix).** Pre-verification: `find frontend/tests -name "*.spec.*" -path "*partner*"` — v2 red-team found zero partner specs. If still zero, document the gap and the spec files that SHOULD exist as a follow-up.
- [ ] **Tailkit literal-class delta grep returns zero (v2.1 C2 fix):** `git diff main -- <scope> | grep -E "^\+.*(literal-classes)"` returns zero NEW additions. Pre-existing 37 matches are out of scope.
- [ ] Dead-code folders `partner-{employees,projects,timesheet}/` either fully removed OR reduced to only explicitly-kept live files per `reports/phase-03-deletion-log.md`.
- [ ] `html.partner-route-active` class is present on `/partner/*` ONLY when role is partner; absent on `/admin/*`, `/employee/*`, `/login`, AND during ProtectedRoute's auth-failure redirect window.
- [ ] `[data-partner-ui]` wrapper attribute present on PartnerLayout div when `isPartner`.
- [ ] Portal-reach selector verified working (Phase 1 Step 9 resolution documented in `partner.css`).
- [ ] All four in-scope partner routes pass desktop + tablet + mobile visual QA at 320 / 390 / 768 / 1440 with no horizontal scroll, no <44px tap targets, tabular numerals on money, icon + text status badges.
- [ ] Filter / sort / paginate / deep-link / mutation spot-checks pass on every polished route.
- [ ] **`BankTransferHistoryPageContent` byte-unchanged (v2.1 H3 fix):** `git diff main -- frontend/src/components/payroll/BankTransferHistoryPageContent.tsx` returns zero; both admin and partner variants render correctly.
- [ ] **`App.tsx` partner payment-history route byte-unchanged (v2.1 H2 fix):** no `ResponsivePage` swap; desktop re-export untouched.
- [ ] Cross-surface isolation check: `/admin/*`, `/employee/*`, `/login` visually unchanged (Phase 4 captures fresh baselines if no prior baseline exists — v2 red-team Failure Mode #8 confirmed no baseline infrastructure).
- [ ] If any `components/shared/*` was edited, the admin/employee consumer screenshot check shows no regression.
- [ ] Keyboard, screen-reader, 200% zoom, and reduced-motion spot checks pass on one representative polished partner route.
- [ ] Whole-Plan Consistency Sweep reports zero unresolved contradictions.
- [ ] `graphify update .` exits 0.
- [ ] Verification report appended with evidence, deviations, and Open Question resolutions.

## Risk Assessment

- **Risk:** Visual QA finds a regression introduced during Phase 3 polish, forcing a reopen.
  **Mitigation:** Phase 3 already required per-route behavioral + accessibility spot-checks; Phase 4 is confirmation. Regressions reopen the specific Phase 3 step, not the whole phase.
- **Risk:** Mobile Payment History wiring (Open Question 1) changes admin's view of `BankTransferHistoryPageContent`.
  **Mitigation:** Step 6 explicitly screenshots both surfaces and compares. Any divergence blocks Phase 4 completion.
- **Risk:** A `shared/*` edit regresses an admin/employee consumer that Phase 2 didn't map.
  **Mitigation:** Step 9 runs the consumer screenshot check for every touched `shared/*` file. Any unmapped consumer regression halts Phase 4 and requires a Phase 2 audit update + Phase 3 mitigation.
- **Risk:** Dead-code deletion surprises: a dynamic import surfaces at runtime that static greps missed.
  **Mitigation:** Step 3 includes the modal-registry string grep. Phase 3 Step 7 also re-ran greps immediately before deletion. If a runtime error appears in Playwright, restore the file and document the dynamic-import pattern that was missed.
