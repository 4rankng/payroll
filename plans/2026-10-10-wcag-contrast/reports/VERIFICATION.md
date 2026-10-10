# WCAG 2.2 AA — Verification coverage (final)

**Target**: 4.5:1 text (AA) / 3:1 non-text & UI boundaries (1.4.11)
**Deployed**: https://tingting.vip — bundle `index.DZMb6WvC.css`

## Card vs page canvas (the reported bug)

| Pair | fg on bg | Ratio | 3:1 |
|---|---|---|---|
| white card vs page canvas | `rgb(255,255,255)` on `rgb(127,145,135)` | **3.33:1** | PASS |
| card border vs card | `rgb(106,129,113)` on white | **4.21:1** | PASS |
| card border vs canvas | `rgb(106,129,113)` on `rgb(127,145,135)` | **3.34:1** | PASS |

Root cause: three light-canvas layers painted over `--background` —
`html` PWA splash (`hsl(220 20% 98%)`), `.admin-shell`/`.partner-shell`
gradients (`#f8fafb→#f5f7f9`), and hardcoded `linear-gradient` in
`AdminPageFrame`/`PartnerLayout`/`AdvancePaymentsPage`. Token change alone
was invisible until those were removed (pixel-probe caught it).

## Text contrast — 12 routes, pixel-sampled

| Route | Failing active text |
|---|---|
| `/admin` | 0 |
| `/admin/users` | 0 |
| `/admin/payment-history` | 0 |
| `/admin/employees` | 0 |
| `/admin/timesheet` | 0 |
| `/admin/ledger` | 0 |
| `/admin/wallet` | 0 |
| `/admin/settings` | 0 |
| `/partner/dashboard` | 0 |
| `/partner/projects` | 0 |
| `/partner/employees` | 0 |
| `/partner/timesheet` | 0 |

## Verification coverage

| Claim | Rung | Evidence | Not covered |
|---|---|---|---|
| card/canvas 3.33:1 | UI DRIVEN | pixel probe of rendered DOM: canvas `rgb(127,145,135)`, card `rgb(255,255,255)`; `reports/screenshots/FINAL-canvas-vs-card.png` | employee/accountant roles |
| borders & rings 3:1 | UI DRIVEN | `--border` 4.21:1 vs card, `--ring` 3.47:1 vs canvas (computed) | focus-visible animation |
| 12 routes text AA | UI DRIVEN | `tests/e2e/measure-contrast.mjs` pixel sweep: 0 failing active text | dark theme, firefox/webkit |
| tokens locked in CI | DB/API VERIFIED | `npx vitest run wcag-contrast.test.ts` 37/37 | tests do not drive a browser |
| prod serves fixed tokens | DB/API VERIFIED | live CSS `--background: 152 6% 54%`, `--border: 138 10% 46%`, `--employee-page: #82918a` | no auth to drive prod UI |

## Exempt (WCAG 1.4.3 — inactive UI)

- 6 disabled controls (fee-preset chips on /admin/settings) at 3.09:1

## Gaps

| Gap | Reason |
|---|---|
| Dark theme | no `.dark` class is ever added; dead token block removed |
| Employee / Accountant / Adv-partner roles | not in the pixel sweep |
| Firefox / WebKit | sweep ran on chromium |
| Hover / focus / disabled transitions | sweep samples rest state |
| Recharts canvas | canvas pixels are not text nodes |