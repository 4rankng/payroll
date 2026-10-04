# Final migration certification — re-baseline + font gate — 2026-10-05 ~01:15

Window opened after W13b/W13c (9b1a340e, ee27d588 — admin table+financial
pages) and the WCAG accent fix (635eaa7a — darkened info/admin/manager
badges+buttons to clear 4.5:1). Tree clean throughout.

## Enumeration run (compare mode)

16 failed / 26 passed — SIX surfaces moved (18 tests), 2 unmoved:

Moved: admin/users, admin/projects, admin/employees, admin/timesheet (all 3
viewports) and partner/projects, partner/timesheets. Partial-viewport
movement within a surface observed (e.g. partner/timesheets@1280 passed
while @390/@320 failed): the WCAG accent fix only darkens chips that some
views render.

Unmoved: admin/dashboard, partner/dashboard (all viewports) — their badge
sets are untouched by the info/admin/manager accent change.

## Per-surface re-baseline with attribution

| Surface | Moved by |
|---------|----------|
| admin/users, admin/projects, admin/employees, admin/timesheet | 9b1a340e + ee27d588 (W13b/W13c admin pages) and 635eaa7a (WCAG accent) |
| partner/projects, partner/timesheets | 635eaa7a (WCAG accent chips visible in partner lists) |

6 x --update-snapshots -g "<surface>", 3 tests each, all green (18 PNGs).

Pre-rebaseline integrity check: mask anchors survived the daisyUI
retirement (8c4bf525) — .admin-dashboard-page-mobile, MobileOperationsPanel
badge classes, and .animate-pulse skeletons all still present (the retired
.ct-loading selector simply never matches now; harmless).

## New volatility class found and closed: font-swap race

First certification compare flipped 41/42: partner/projects@320 diffed with
content-identical text — 257 antialiasing pixels (per-channel deltas of
~1-10) in the search placeholder and per-card rows. Root cause: the app
applies Google Fonts stylesheets via an async media="print" -> "all" swap
(index.html data-font-stylesheet links: Manrope, Inter, JetBrains Mono); a
capture landing before a stylesheet applies / woff2 finishes renders
fallback font metrics.

Fix in stabilize(): wait until every link[data-font-stylesheet] has
media="all" (15s bound, offline-safe catch), then await document.fonts.ready
plus an explicit load() of all registered faces. Because this changes the
capture state for ALL surfaces, a full regeneration followed (per the same
convention as the quiet-window change).

## Final sequence results

| Run | Result | Duration |
|-----|--------|----------|
| Regeneration (full, with font gate) | 42/42 passed | 3.2m |
| Compare 1 | 42/42 passed | 3.2m |
| Compare 2 | 42/42 passed | 3.2m |

Certification target met: 42/42 x2 back-to-back (24 visual + 18 auth
riders). The font race is the seventh and last volatility class closed;
baseline determinism is now total (code, data, timing, and fonts).

## Handoff for commit (team-lead)

- frontend/tests/e2e/visual-baseline.spec.ts (modified: font gate in
  stabilize())
- frontend/tests/e2e/visual-baseline.spec.ts-snapshots/ (24 PNGs, final)
- This report; fixtures unchanged.

Status: DONE
Summary: enumerated 6 moved surfaces (16 tests), re-baselined per-surface
with W13b/W13c + WCAG attribution, closed a newly found font-swap race with
a stylesheet+fonts.ready gate, full regen 42/42 and certified 42/42 x2.
Concerns/Blockers: none.
