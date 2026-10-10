# WCAG 2.2 AA — Verification coverage

**Date**: 2026-10-10
**Target**: 4.5:1 text (AA), 3:1 non-text (AA)
**Scope**: Admin + Partner roles, light theme, 1280/390/320 viewports

## Verification coverage

| Claim / bug | Rung | Evidence | Not covered |
|---|---|---|---|
| `base.css:888` `#0284c7` → `#0369a1` (sky-700) | UI DRIVEN | `screenshots/admin--admin-1280.png`, axe `admin--admin-1280.json` zero violations | dark theme variant, employee portal |
| 309 Tailwind class swaps (`text-red-600→700`, `text-slate-500→600`, etc.) across 99 files | UI DRIVEN | 33 axe JSON reports + 33 screenshots, `axe/*.json` zero serious/critical | hover/focus states, employee portal, dark theme, chart colors |
| `text-yellow-700` → `text-yellow-800` (16 usages, 11 files) | UI DRIVEN | `screenshots/admin--admin-timesheet-1280.png` (TimesheetEntryTable), axe zero violations | star icon hover, dark theme |
| `text-red-400` → `text-red-700` in EntryRow/HourInputField | UI DRIVEN | `screenshots/admin--admin-timesheet-390.png`, axe zero violations | disabled state, dark theme |
| `text-yellow-400 fill-yellow-400` → `yellow-800` in CalendarDayCell | UI DRIVEN | `screenshots/admin--admin-timesheet-1280.png`, axe zero violations | Sunday column hover, dark theme |
| Extended `wcag-contrast.test.ts` (37 tests) locks AA floor in CI | DB/API VERIFIED | `npx vitest run wcag-contrast.test.ts` — 37/37 pass | tests do not drive a browser; axe smoke covers runtime |
| Axe smoke sweep: 11 routes × 3 viewports = 33 combinations | UI DRIVEN | 33 `axe/*.json` + 33 `screenshots/*.png`, zero serious/critical | firefox/webkit browsers, dark theme, employee/accountant roles |

## Evidence inventory

- **Axe JSON reports**: 33 files in `reports/axe/`
- **Screenshots**: 33 files in `reports/screenshots/`
- **Static audit**: `reports/contrast-audit.md` (40 remaining findings, all dark-context false positives)
- **Vitest**: `frontend/src/__tests__/wcag-contrast.test.ts` (37 tests)
- **Playwright spec**: `frontend/tests/e2e/wcag-contrast.spec.ts`

## Remaining known limitations

| Gap | Reason |
|---|---|
| Dark theme (`.dark`) | Axe sweep covers light theme only; dark: classes verified by static scanner against dark surfaces |
| Employee / Accountant / Adv-partner roles | Not in the axe smoke route table; follow-up expansion |
| Firefox / WebKit browsers | Smoke ran on chromium only; cross-browser is a follow-up |
| Hover / focus / disabled states | Axe inspects the rendered DOM at rest; interactive states need manual QA |
| Chart colors (Recharts) | Canvas-rendered; axe cannot inspect SVG/canvas internals |
| 40 dark-context scanner false positives | Classes like `text-emerald-200/75` inside `bg-emerald-950` modal headers; axe confirms they pass at runtime |