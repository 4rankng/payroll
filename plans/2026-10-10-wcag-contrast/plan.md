# WCAG 2.2 AA Contrast Audit & Remediation

**Status**: complete
**Target**: 4.5:1 for normal text, 3:1 for large text and non-text UI (WCAG 2.2 AA)
**Scope**: All roles (Admin, Partner, Employee, Accountant, Adv-partner) × both themes × 1280 / 390 / 320 viewports
**Gate**: Static scanner + extended vitest + Playwright/axe smoke + rung-3 UI verification with artifacts

## Phases

- [x] `phase-01-static-scanner.md` — build & run the audit script, capture full failure list
- [x] `phase-02-token-fixes.md` — token-first palette swaps, `base.css` hex fix
- [x] `phase-03-vitest-contract.md` — extend `wcag-contrast.test.ts` to lock in the floor
- [x] `phase-04-axe-smoke.md` — `@axe-core/playwright` sweep over routes × themes × viewports
- [x] `phase-05-ui-verification.md` — rung-3 claims with screenshots + coverage table

## Dependencies

- Phase 2 depends on Phase 1's failure list (do not edit UI before the audit report exists).
- Phase 3 and 4 are independent of each other but both depend on Phase 2.
- Phase 5 depends on all prior phases.

## Acceptance criteria

1. `reports/contrast-audit.md` lists every failing pair with `file:line`, current ratio, required ratio, suggested fix.
2. Zero failing pairs remain for text ≥4.5:1 and non-text ≥3:1 in both themes.
3. `pnpm test:run` passes, including the extended `wcag-contrast.test.ts`.
4. `pnpm test:e2e` passes, including the new axe smoke spec (zero `serious`/`critical` color-contrast violations).
5. `reports/VERIFICATION.md` contains a coverage table with rung labels and a non-empty "Not covered" column for every claim.

## Reports

- `reports/contrast-audit.md` — Phase 1 output
- `reports/axe/*.json` — Phase 4 per-route axe output
- `reports/screenshots/*` — Phase 5 rung-3 evidence
- `reports/VERIFICATION.md` — Phase 5 coverage table
