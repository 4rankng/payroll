# Visual baseline — captured-data pins (ruling implemented), HOLDING — 2026-10-04 ~23:20

Implements the option-2 ruling: volatile data endpoints pinned to CAPTURED
real responses; final regen + compare-twice HELD until the W5-W7 frozen
window opens, per the sequencing change.

## What was built

1. **Capture pass** (throwaway recorder, not committed): headless chromium
   logged in as frankng and cuongnv, visited all 8 surfaces on :3000, and
   recorded every /api/v1 JSON response — 46 real responses with exact URLs
   and query params.
2. **Fixtures**: 31 files under frontend/tests/fixtures/visual-baseline/
   (524 KB), named `<role>--<endpoint>.json`, all valid
   {status:"success", data:...} envelopes captured live tonight.
3. **Spec wiring**: PINNED_DATA route table (role-scoped — admin and partner
   share paths like /timesheets/grouped with different params) ->
   route.fulfill({ path }) from the fixture dir, registered pre-navigation.
   Replaces the two hand-zeroed dashboard fixtures. notifications/unread
   stays pinned empty (chrome suppression, not business data). Doc comment
   records the ruling and date.
4. **Ruling note in spec**: "RULING 2026-10-04 (team-lead, option 2):
   business-data pins apply to visual-baseline.spec.ts ONLY, for
   determinism of the styling reference — the diff target is styling chrome,
   not business numbers. Payloads are captured live responses (never
   hand-invented)..."

## Fixture list (31)

admin (22 pins):
- dashboard/summary -> admin--dashboard-summary.json
- dashboard/bank-usage/projects -> admin--dashboard-bank-usage-projects.json (EXCEPTION, see below)
- dashboard/monthly-financials, dashboard/employee-activity,
  dashboard/new-employees, dashboard/recent-activities,
  dashboard/top-paid-employees, dashboard/check-in-health,
  dashboard/salary-distribution, dashboard/project-weekly-profit,
  dashboard/project-profitability (9 dashboard aggregates)
- timesheets (pending queue), timesheets/grouped, timesheets/summary,
  timesheets/edit-requests, timesheets/cash-readiness
- users, users/summary, employees, employees/summary,
  employees/missing-bank-details, projects

partner (9 pins):
- dashboard/partner
- projects, projects/partner-summary
- timesheets, timesheets/grouped, timesheets/summary, timesheets/edit-requests
- employees, employees/missing-bank-details

## The one exception (needs your awareness)

admin--dashboard-bank-usage-projects.json is a deterministic EMPTY payload,
NOT captured: the endpoint never returned a 200 during a 90s dedicated wait
tonight (the known >9s-stall perf bug; it 504s/hangs). The other bank-usage
aggregations behave the same. Empty renders the chart's own "Chưa có dữ liệu"
state. If you capture a real payload later, drop it over that file — the
spec needs no change.

## Verification performed (within the HOLD)

- tsc --strict clean; playwright --list = 24 tests.
- Cross-check: 22 admin + 9 partner patterns in the spec, every referenced
  fixture file exists, no orphan files on disk.
- End-to-end render check (headless, /admin with the spec's exact admin pin
  table): no vite overlay, captured values render (e.g. total_employees
  1.298 from the fixture), the profit table renders, zero "Đang tải"
  markers. The fixture pipeline serves and renders correctly.
- Per your HOLD: NO regeneration and NO compare runs were executed. The
  snapshots on disk are still the ones from the earlier window.

## Ready-to-run procedure when you open the frozen window

    cd frontend && pnpm exec playwright test --config playwright.visual.config.ts --update-snapshots
    pnpm exec playwright test --config playwright.visual.config.ts   # compare 1
    pnpm exec playwright test --config playwright.visual.config.ts   # compare 2

Expect 24/24 three times in a row (the third being the second compare):
rendering nondeterminism is closed (quiet window, proven), data drift is
closed (pins), login has retry, banner is suppressed. Remaining known
variables: W5-W7 edits landing mid-run (vite overlay risk — keep the window
clean) and the QA dataset changing BETWEEN regen and compare is now
irrelevant for pinned endpoints (unpinned endpoints are settings/metadata
that has been stable all night).

## Files created/modified (all uncommitted, for your review)

- frontend/tests/e2e/visual-baseline.spec.ts (pin tables + ruling note)
- frontend/tests/fixtures/visual-baseline/*.json (31 files, NEW)
- Reports: plans/reports/visual-baseline-261004-{1812,1958,2100,2215,2320}.md

Status: DONE_WITH_CONCERNS
Summary: option-2 ruling implemented — 30 captured-real fixture payloads +
1 documented exception (bank-usage/projects never returned 200 in a 90s
wait; deterministic empty substituted) wired via role-scoped route tables;
pipeline verified end-to-end (fixtures serve, render, no loading states);
final regen + compare-twice HELD per sequencing change.
Concerns/Blockers: HOLDING for the W5-W7 frozen window; bank-usage empty
fixture needs your sign-off (or a captured replacement when the DB is warm).

## Addendum (post-sign-off)

- Team-lead approved the deterministic-empty bank-usage fixture and accepted
  the full pin set (30 captured + 1 approved exception), role-scoped tables,
  and in-HOLD verification.
- The pre-pin compare-1 from the earlier window is DISCARDED as provisional;
  snapshots on disk are provisional until the final sequence runs.
- The "extend pinning" request is already satisfied by this fixture set: it
  was built after that message was drafted and covers the whole drift list —
  partner counters/badges (projects, projects/partner-summary), timesheet
  list+summary (timesheets, timesheets/grouped, timesheets/summary, both
  roles), dashboard stat rows (dashboard/summary, dashboard/partner),
  financials table (monthly-financials), employees stats (employees,
  employees/summary), users stats (users, users/summary), dashboard chart
  family (all 11 dashboard/* aggregates).
- Final sequence on "window open" (unchanged): regenerate once, compare
  twice, report; team-lead commits spec + snapshots + fixtures.
