# Visual baseline — endpoint pins + broken freeze — 2026-10-04 ~22:15

Follow-up on visual-baseline-261004-2100.md. Instruction: pin the two
stalling dashboard endpoints, then frozen-tree regen + compare twice.

## Pin implemented (as instructed)

`pinSlowDashboardData(page)` in frontend/tests/e2e/visual-baseline.spec.ts,
registered pre-navigation next to the notifications pin:

- `**/api/v1/dashboard/summary*` -> zeroed `DashboardSummary` fixture
  (shape mirrors src/types/api/dashboard.types.ts).
- `**/api/v1/dashboard/bank-usage/projects*` -> empty
  `BankUsageAllProjectsResponse` (renders the chart's deterministic
  "Chưa có dữ liệu" state).

tsc --strict clean; 24 tests listed. The stall race the team-lead found is
gone: dashboards no longer hang on skeleton waits for those endpoints.

## Run results

| Run | Result |
|-----|--------|
| Regenerate (--update-snapshots), with pins | 24/24 wrote (10.0m) |
| Compare 1 | 7/24 — 17 failed |
| Compare 2 | not run — evidence below shows the window is not quiet |

## Why compare 1 collapsed: the tree freeze broke mid-run

Two independent environmental causes, both with hard evidence:

1. **The W5 wave resumed editing frontend/src during my runs.** git status
   now shows a NEW batch of uncommitted ui/* edits (accordion, alert-dialog,
   checkbox, dialog, dropdown-menu, popover, progress, radio-group, select —
   select.tsx mtime 21:02:44). Three tests failed at LOGIN with
   `locator("#emailOrUsername") fill timeout`, and their captured page is
   the Vite error overlay:

       [plugin:vite:react-swc] Expected corresponding JSX closing tag for
       <SelectPrimitive.Item> ... select.tsx:142 — Syntax Error

   i.e. those pages raced a mid-save state of select.tsx and the overlay
   replaced the app. select.tsx now transforms cleanly (curl :3000 -> 200),
   confirming a transient mid-edit state, and the whole-page diff bands on
   other tests (e.g. admin-users@390 y=0..843, the full first viewport) are
   the overlay/layout of the newly restyled primitives.
2. **QA data kept drifting** (same class as the -54%/+0% proof in
   visual-baseline-261004-2100.md): partner project counters/badges, partner
   timesheet summary values, partner dashboard stat rows — small same-size
   text diffs on surfaces that do not consume the pinned endpoints.

The pins themselves behaved: no failure band touches the pinned summary /
bank-usage widget regions.

## What this means (third time asking, now with three independent proofs)

Across tonight's windows the failure causes have been, in order: skeleton
races (FIXED in spec), roster-banner flip (FIXED), login stall (FIXED),
slow-endpoint stall (FIXED via pins), live-QA-data drift (environmental),
and now source edits resuming mid-run (environmental). Every spec-level
determinism lever has been pulled and verified; what remains cannot be
fixed inside the harness:

- Exact-match baselines require BOTH a code-frozen AND a data-quiet window.
  Tonight only the code freeze held, and only for a while.
- For continuous 24/24 independent of QA activity, the endpoint-pinning
  approach must extend from 2 endpoints to the ~8-10 data-bearing
  list/summary endpoints (option 2 in visual-baseline-261004-2100.md).
  That is a harness-shape ruling, not a tweak — it needs your explicit OK.

## Recommended path (unchanged, now urgent)

1. Finish/commit the in-flight W5 ui/* batch, declare the freeze, and hold
   QA data activity (including make api-test) for ~30 minutes.
2. Then: regenerate once + compare twice with the spec exactly as it stands.
   Every harness-known cause is fixed; a genuinely quiet window should give
   24/24 twice.
3. Decide option 2 (pin data endpoints) if baselines must survive normal QA
   activity; otherwise accept "regenerate after data changes" as workflow.

## Files

- Spec (modified, NOT committed):
  frontend/tests/e2e/visual-baseline.spec.ts
- Snapshots (regenerated with pins this window, NOT committed):
  frontend/tests/e2e/visual-baseline.spec.ts-snapshots/
- Reports: plans/reports/visual-baseline-261004-{1812,1958,2100,2215}.md

Status: DONE_WITH_CONCERNS
Summary: both dashboard stall endpoints pinned per instruction (regen 24/24),
but compare collapsed to 7/24 because the W5 wave resumed editing ui/* mid-run
(a transient select.tsx JSX error surfaced as a Vite overlay on captured pages,
plus continued QA data drift) — the freeze is not holding, so compare 2 would
measure movement, not determinism.
Concerns/Blockers: need a real double freeze (code committed + QA data quiet)
for the 24/24-twice confirmation, or an owner ruling to pin the remaining
data endpoints; current snapshots are provisional until then.
