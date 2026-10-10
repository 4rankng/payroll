# Phase 5 — Rung-3 UI verification

## Context

Per the project's anti-lying contract (`AGENTS.md` — "UI verification contract"), any claim of "verified" for a user-visible fix requires **rung 3 = UI DRIVEN**: a real browser session, a post-click DOM/screenshot, and a DB/API side-effect when applicable. Rung 2 ("DB/API verified, UI not driven") must be reported as such, never dressed up as rung 3.

## Requirements

- For each fixed pair a user can actually see, drive the real browser via Playwright.
- Capture: post-render screenshot, post-render DOM text, and (where the fix affects data) the DB/API side effect.
- Produce `reports/VERIFICATION.md` with a coverage table whose "Not covered" column is never empty.

## Files

- Create: `plans/2026-10-10-wcag-contrast/reports/VERIFICATION.md`
- Capture: `plans/2026-10-10-wcag-contrast/reports/screenshots/*.png`

## Steps

1. Select the highest-impact fixed pairs (button labels, badge text, table cells, chart labels, sidebar nav).
2. For each, run a Playwright driver that:
   - Navigates to the containing page at 1280 / 390 / 320.
   - Renders the state that shows the fixed text.
   - Captures a screenshot after render (or after click, when the fix is in a hover/disabled/error state).
   - Reads the rendered text and the computed `color` / `backgroundColor` from the DOM.
   - Where applicable, queries the API to confirm the side effect.
3. Write the coverage table.

## Coverage table format

```
## Verification coverage
| Claim / bug | Rung | Evidence | Not covered |
|---|---|---|---|
| text-slate-400 → text-muted-foreground | UI DRIVEN | screenshots/users-390.png, axe/users-390-light.json, DOM text="..." | hover state, partner role |
| base.css #0284c7 → var(--info) | UI DRIVEN | screenshots/login-1280.png, computed color=rgb(23,92,211) | dark theme variant |
```

## Rules (from the project's claim ladder)

- Rung 3 requires **all four** artefacts: (1) screenshot after the interaction, (2) post-click DOM/text assertion quoted verbatim, (3) DB side-effect proof when applicable, (4) driver log with exit status.
- If the driver fails, report `NOT TESTED — driver failed: <reason>`. Never fall back to rung-2 wording that sounds like rung-3.
- If the UI cannot be driven, say so in the first sentence of the report and ask the user whether to invest in the driver.

## Tests / validation

- Every row in the coverage table has a rung label, an evidence path that exists on disk, and a non-empty "Not covered" cell.

## Risks

- Mobile viewport (390 / 320) can hide table content behind `ResponsivePage` swaps; the driver must navigate to the mobile variant explicitly, not rely on CSS media queries.
