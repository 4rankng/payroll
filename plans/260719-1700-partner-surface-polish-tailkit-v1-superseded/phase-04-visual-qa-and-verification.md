---
phase: 4
title: "Visual QA and Verification"
status: pending
priority: P2
dependencies: [1, 2, 3]
---

# Phase 4: Visual QA and Verification

## Overview

Close the loop: lint, type-check, build, focused tests, route-by-route visual QA across desktop and mobile, cross-surface isolation check (partner styles must not leak into admin/employee/login), Tailkit→shadcn mapping consistency sweep, and `graphify update .`. This is the gate before recommending the plan for cook or handoff.

## Requirements

- **Functional:** every acceptance criterion from `plan.md` is demonstrably met with evidence (command output, screenshots, DOM checks).
- **Non-functional:** zero unresolved contradictions across plan files; zero literal Tailkit classes left in code; zero cross-surface style leakage.

## Architecture

Verification is organized in four passes — automated, visual, isolation, and consistency — run in that order. Each pass produces evidence captured in the plan's verification report (this file, Success Criteria section).

## Related Code Files

- **Read-only verification targets:**
  - Every file touched in Phases 1–3.
  - `frontend/src/styles/partner.css` (mapping-table reconciliation).
  - `frontend/tailwind.config.ts` (confirm `themeRoot` was not changed, or was changed deliberately with documented reason).
- **Tooling:** `pnpm lint`, `tsc --noEmit`, `pnpm build`, `pnpm test:run`, Playwright partner specs, `make api-test`, `graphify update .`.

## Implementation Steps

1. **Automated pass.**
   - `cd frontend && pnpm lint`
   - `cd frontend && pnpm type-check` (or `tsc -p tsconfig.json --noEmit`)
   - `cd frontend && pnpm build`
   - `cd frontend && pnpm test:run -- partner` (or run the focused Vitest filter that covers `partner-*` and `pages/partner`)
   - `make api-test` (sanity — required by `AGENTS.md` after every change)
   - Capture each command's pass/fail and tail of output.
2. **Tailkit literal-class grep.**
   - `grep -rn -E "(secondary-[0-9]|emerald-[0-9]|orange-[0-9]|sky-[0-9]|violet-[0-9]|slate-[0-9]|dark:)" frontend/src/components/partner-ui frontend/src/components/partner-* frontend/src/pages/partner frontend/src/layouts/PartnerLayout.tsx frontend/src/components/PartnerSidebar.tsx`
   - Expected: zero matches. Any match is a Phase 1–3 regression to fix before proceeding.
3. **Scope leakage grep.**
   - `grep -rn "data-partner-ui" frontend/src` — confirm the attribute is set only in `PartnerLayout.tsx` (or partner-owned primitives), nowhere else.
   - `grep -rn "partner.css" frontend/src` — confirm exactly one import.
4. **Visual QA — partner routes (desktop + mobile).**
   - For each of: `/partner/dashboard`, `/partner/projects`, `/partner/employees`, `/partner/timesheet`, `/partner/timesheet/payment-history`:
     - Desktop 1440px screenshot.
     - Tablet 768px screenshot.
     - Mobile 390px screenshot.
     - Mobile 320px screenshot (narrowest support).
     - Verify: no horizontal scroll, no clipped controls, no nested-scroll hazards, no tap targets <44px on mobile.
     - Verify: status pills have icon + text (color is not the sole signal).
     - Verify: tabular numerals on every financial figure.
5. **Visual QA — partner interactive flows.**
   - Filter + sort + paginate on each list route; confirm URL state updates and survives reload.
   - Open and dismiss one deep-linked sheet per route (where present).
   - Trigger one mutation per route (create / update / delete) against the dev backend; confirm toast feedback and cache invalidation.
6. **Cross-surface isolation check.**
   - Visit `/admin/dashboard`, `/admin/employees`, `/employee` (one employee route), `/login`.
   - Screenshot each; confirm visually unchanged from pre-plan baseline (or grab a fresh baseline if none exists).
   - DOM check on each: `document.querySelector('[data-partner-ui]')` returns null.
7. **Accessibility spot check.**
   - Keyboard tab-through one representative partner route; confirm visible focus rings and logical order.
   - VoiceOver pass on the same route.
   - 200% browser zoom on the same route; confirm no clipped content.
   - Toggle `prefers-reduced-motion`; confirm no jarring animation on hover/transition.
8. **Whole-Plan Consistency Sweep (mandatory per `ck:plan`).**
   - Re-read `plan.md` and every `phase-*.md`.
   - Search all plan files for: stale Tailkit identifiers that were rejected, renamed tokens, superseded decisions, duplicate embedded drafts.
   - Reconcile the Tailkit→shadcn mapping table in `partner.css` against what Phase 2/3 actually implemented.
   - If unresolved contradictions remain, report them and ask the user; do not recommend cook until zero remain.
9. **Graph refresh.**
   - `graphify update .` from repo root. Confirm exit code 0.
10. **Verification report.**
    - Update this phase file's Success Criteria with checkmarks and link/inline the captured evidence.
    - Note any deviations, deletions of legacy files, and follow-up issues.

## Success Criteria

- [ ] `pnpm lint` passes (tail of output captured).
- [ ] `tsc --noEmit` passes.
- [ ] `pnpm build` passes.
- [ ] Focused Vitest partner specs pass.
- [ ] `make api-test` passes.
- [ ] Tailkit literal-class grep returns zero matches across partner surface.
- [ ] `data-partner-ui` scope-leakage grep confirms attribute lives only in `PartnerLayout.tsx` / partner-owned primitives.
- [ ] `partner.css` is imported exactly once.
- [ ] **Partner sidebar visual parity with admin sidebar** (validated Session 1, Q4) — side-by-side screenshot comparison shows matching active state, hover, focus-ring, collapsed behavior.
- [ ] **`PartnerStatsCard` API parity with `PremiumStatStrip`** (validated Session 1, Q2) — same label/value/unit/highlight/onClick/trend contract; partner extensions (icon, sparkline, valueFormat, loading) verified working.
- [ ] **Phase 3 deletion log** (validated Session 1, Q3) — every removed per-page partner component listed with deletion reason; grep confirms zero orphaned imports.
- [ ] All five partner routes pass desktop + tablet + mobile visual QA at 320 / 390 / 768 / 1440 with no horizontal scroll, no <44px tap targets, tabular numerals on money, and icon + text status badges.
- [ ] Filter / sort / paginate / deep-link / mutation spot-checks pass on every partner route.
- [ ] Cross-surface isolation check confirms `/admin/*`, `/employee/*`, `/login` are visually unchanged and `data-partner-ui` is absent.
- [ ] Keyboard, screen-reader, 200% zoom, and reduced-motion spot checks pass on one representative partner route.
- [ ] Whole-Plan Consistency Sweep reports zero unresolved contradictions; Tailkit→shadcn mapping table in `partner.css` matches Phase 2/3 implementation.
- [ ] `graphify update .` exits 0.
- [ ] Verification report appended to this phase file with evidence and any deviations.

<!-- Updated: Validation Session 1 - added success criteria for sidebar visual parity (Q4), PremiumStatStrip API parity (Q2), Phase 3 deletion log (Q3) -->

## Risk Assessment

- **Risk:** Visual QA finds regressions late, forcing a Phase 3 reopen.
  **Mitigation:** Phase 3 already required per-route behavior spot-checks; Phase 4 is confirmation, not discovery. If Phase 4 surfaces significant issues, reopen the specific route's Phase 3 step rather than expanding scope.
- **Risk:** Mapping table in `partner.css` has drifted from Phase 2/3 reality.
  **Mitigation:** Step 8 Whole-Plan Consistency Sweep explicitly reconciles this; any update is a one-line edit, not a redesign.
- **Risk:** Cross-surface screenshots reveal a real leak despite the `[data-partner-ui]` scope.
  **Mitigation:** This is the highest-severity finding possible — if found, halt and patch the `partner.css` selector before any other Phase 4 step continues.
