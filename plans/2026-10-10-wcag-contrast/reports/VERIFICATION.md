# WCAG 2.2 AA — Verification coverage (updated 2026-10-10)

**Target**: 4.5:1 normal text, 3:1 large text / non-text UI
**Method**: pixel-level measurement of the real rendered DOM (Playwright + pngjs), not computed-style guessing.

## Root cause of the original "washed out" labels

The `daisyui` plugin generated a `\.!dark { ... !important }` block mirroring the `.dark`
token block in `variables.css`. Its `--muted-foreground: 145 15% 70%` resolves to
`rgb(167,190,177)` = **1.97:1 on white** — exactly the washed-out look in the screenshot.
daisyUI had **zero** remaining component usages (`ct-*` hits were all comments/regex noise);
`@untitledui/icons` (119 files) plus the shadcn/UUI component set were already the real
design system. Removing daisyUI removed the leak at its source.

## Pixel-verified results (11 routes)

| Route | Active text failing | Exempt (inactive UI) | Worst active ratio |
|---|---|---|---|
| `/admin/users` | 0 | 0 | 8.89:1 |
| `/admin` | 0 | 0 | — |
| `/admin/timesheet` | 0 | 0 | — |
| `/admin/ledger` | 0 | 0 | — |
| `/admin/settings` | 0 | 5 | — |
| `/admin/wallet` | 0 | 0 | — |
| `/admin/employees` | 0 | 1 | — |
| `/partner/dashboard` | 0 | 0 | — |
| `/partner/projects` | 0 | 0 | — |
| `/partner/employees` | 0 | 0 | — |
| `/partner/timesheet` | 0 | 0 | — |

Key measured pairs:

| Element | fg on bg | Ratio | AA 4.5:1 |
|---|---|---|---|
| KPI labels ("Quản lý"/"Nhân viên") | `rgb(60,78,67)` on white | **8.89:1** | PASS |
| "Thêm người dùng" button | white on `rgb(8,120,62)` | **5.58:1** | PASS |
| Sidebar nav / section headers | white/75-85% on dark green | **6.7–9.9:1** | PASS |
| Table headers | `rgb(60,78,67)` on white | **8.89:1** | PASS |

## Verification coverage

| Claim / bug | Rung | Evidence | Not covered |
|---|---|---|---|
| daisyUI removal kills `!important` dark-token leak | UI DRIVEN | live CSS `grep -c "!dark"` = **0**; `--muted-foreground: 220 18% 34%` in prod bundle `index.ZUAqdLNX.css` | other browsers |
| KPI label contrast (the screenshot) | UI DRIVEN | `tests/e2e/measure-contrast.mjs` pixel sample: 8.89:1; `reports/screenshots/VERIFY-local-render.png` | dark theme, employee role |
| 11 routes, active text | UI DRIVEN | pixel sweep: 0 failing across admin + partner | employee/accountant roles, dark theme, firefox/webkit |
| `fg.disabled` #d0d5dd → #8794a3 (1.47→3.09:1) | UI DRIVEN | prod CSS `rgb(135 148 163)` present in `fg-disabled:disabled` | inactive-UI is WCAG 1.4.3 exempt; not gated at 4.5 |
| Token contract (37 vitest tests) | DB/API VERIFIED | `npx vitest run wcag-contrast.test.ts` 37/37 | tests do not drive a browser |

## Exempt (WCAG 1.4.3 — inactive UI components, no contrast requirement)

- `/admin/settings`: 5 disabled fee-preset chips at 3.09:1 (was 1.47:1)
- `/admin/employees`: 1 disabled control

## Remaining gaps

| Gap | Reason |
|---|---|
| Dark theme | No `.dark` class is ever added by the app; the dead token block was removed. Not exercised. |
| Employee / Accountant / Adv-partner roles | Not in the pixel sweep; add to `measure-contrast.mjs` routes |
| Firefox / WebKit | Sweep ran on chromium only |
| Hover / focus / disabled transitions | Sweep samples rest state |
| Chart canvas colours (Recharts) | Canvas pixels are not text nodes |