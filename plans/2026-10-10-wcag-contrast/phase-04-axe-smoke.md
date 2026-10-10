# Phase 4 — Playwright + axe smoke sweep

## Context

The static scanner cannot see runtime-computed classes, hover/focus/disabled states, chart colors, or shadcn composed components. axe-core on a real rendered DOM catches these.

## Requirements

- Sweep every route in `App.tsx` across roles, themes, and viewports.
- Fail on any `serious` or `critical` `color-contrast` violation.
- Store per-route axe JSON + a screenshot as evidence for Phase 5.
- Fit within the existing `playwright.config.ts` setup (already has chromium / firefox / webkit projects).

## Files

- Modify: `frontend/package.json` (add `@axe-core/playwright` devDep)
- Modify: `frontend/playwright.config.ts` (add a `wcag` project)
- Create: `frontend/tests/e2e/wcag-contrast.spec.ts`

## Steps

1. `pnpm add -D @axe-core/playwright`.
2. Build a route table from `App.tsx` (path × requiredRole × template).
3. For each route × role × theme (light / dark via `[data-theme]`) × viewport (1280 / 390 / 320):
   - Navigate (authenticate via the existing fixture).
   - Wait for network idle.
   - Run `new AxeBuilder({ page }).withRules(['color-contrast', 'color-contrast-enhanced']).analyze()`.
   - Write JSON to `plans/2026-10-10-wcag-contrast/reports/axe/<route>-<vp>-<theme>.json`.
   - Screenshot to `plans/2026-10-10-wcag-contrast/reports/screenshots/`.
4. Fail the test if any `serious`/`critical` violation is present.

## Tests / validation

- `pnpm test:e2e --grep @wcag` runs the sweep.
- Every JSON report is written; the axe summary shows zero `serious`/`critical` color-contrast violations.

## Risks

- Full matrix = 50 routes × 2 themes × 3 viewports = 300 runs. Mitigation: smoke first against one representative route per template (Dashboard, List, Detail, Form, Login), then expand to the full matrix only where failures are reported.
- Authentication is required for protected routes. Use the existing e2e auth fixture; if a role cannot be authenticated in CI, mark those routes as `test.skip` and record the gap in `VERIFICATION.md` under "Not covered".
