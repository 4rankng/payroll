# Visual baseline — frozen-tree runs — 2026-10-04 ~21:00

Follow-up on visual-baseline-261004-1958.md. Window: UU primitive waves
committed (036c2e39..42e045a9), code tree frozen. Mandate: regenerate
baselines, compare twice, expect 24/24 twice.

## What was added to the spec first

Compare 1 on the frozen tree (before any spec change) still showed 5 flips;
forensics showed late-arriving aggregate widgets (bank-usage, payout
forecast, ops metrics — the >10s slow endpoints) rendering skeleton/zero
intermediate states that resolve at varying times. Added to stabilize():

- `waitForNetworkQuiet(page, 2500, 45000)`: a PerformanceObserver tracks
  every resource fetch; capture proceeds only after NO request has fired for
  2.5s continuously (45s budget, catch-through). All slow endpoints drain
  before the shot; skeleton re-check runs after it.
- SETTLE_MS 1000 -> 400 (shorter window for a poll to fire mid-capture);
  per-test timeout 90s -> 180s to absorb the quiet budget on slow pages.

tsc --strict clean, 24 tests listed.

## Run results on the frozen tree

| Run | Result |
|-----|--------|
| Regenerate (--update-snapshots) | 24/24 wrote |
| Compare 1 | 17/24 — 7 failed |
| (compare 2 skipped — see below) | — |

## Why the remaining 7 flip: live DATABASE drift, not the harness

Every remaining diff band is data-derived content. Proof, from
partner-dashboard@390 (band y=207..218), actual vs baseline:

    Tổng chi trả 0 ₫  -54% so tháng trước     <- compare run
    Tổng chi trả 0 ₫   +0% so tháng trước     <- baseline, 16 min earlier

The "percent vs last month" aggregate changed because QA payment data
changed between the two runs. Same class for the rest:

- partner-timesheets@1280: whole content area shifted — the timesheet
  summary/list heights changed (rows moved) between runs.
- partner-timesheets@320, partner-dashboard@390: live stat values and row
  text.
- admin-dashboard@1280: bank-allocation chart region (y=577..899) — chart
  data changed; tiny 2-4px slivers are axis tick labels.
- admin-timesheet@390/320: +8px height — a data-driven text wraps after a
  value changed.

Important: the earlier skeleton/loading incident class is GONE from the
failure set — the quiet-window wait works. The code tree is frozen; the QA
DATABASE is not (payments/approvals/test-suite runs continue). Exact-match
baselines cannot survive run-to-run data changes, whatever the wait
strategy. I did not run compare 2: with data moving on a ~15-min cadence it
would only re-measure the drift, and I did not regenerate again — each
regeneration re-bakes the current data and re-opens the treadmill.

## State of the harness (frontend/tests/e2e/visual-baseline.spec.ts)

- Determinism machinery is complete and demonstrably effective: reduced
  motion, animation kill, notifications pin, network-quiet window,
  skeleton/"Đang tải" waits, dashboard masks, login retry. Loading-state
  flakes: eliminated. Login stalls: eliminated (no recurrence in 4 runs).
- Baselines currently on disk were captured in this window (frozen code +
  then-current data): 17/24 verified byte-stable across a 16-minute gap;
  the other 7 differ only on live-data content.

## The decision this needs (owner ruling)

Two viable paths to a stable 24/24 — both outside the spec:

1. **Data-quiet window (no code change).** Nobody and nothing touches the QA
   database for ~30 minutes (including `make api-test`, which mutates QA
   data), then: regenerate once, compare twice. With the current spec this
   should give 24/24 twice; any later data change breaks exact-match again
   until baselines are regenerated — acceptable if the team accepts
   "regenerate baselines when QA data moved" as part of the wave workflow.
2. **Pin data endpoints (spec change, robust forever).** Fulfill the ~8-10
   aggregate/list endpoints (dashboard summary, bank usage, forecast, ops
   metrics, timesheet lists, partner summary) with fixed payloads so
   screenshots show real chrome over frozen data. Deterministic across days,
   machine-independent, CI-safe. Trade-off: baselines stop reflecting live
   data (fine for a styling-migration reference) and it formally crosses the
   "no business-data mocks" line the spec documents — needs an explicit OK.

Recommendation: option 2 for the migration (the reference is about styling,
not data), option 1 as the zero-change fallback. A tolerance-based diff is
the only other lever and was excluded up front (masks real regressions).

## Files

- Spec (modified, not committed):
  frontend/tests/e2e/visual-baseline.spec.ts
- Snapshots (regenerated this window, not committed):
  frontend/tests/e2e/visual-baseline.spec.ts-snapshots/ (24 files)
- Report: plans/reports/visual-baseline-261004-2100.md

Status: DONE_WITH_CONCERNS
Summary: quiet-window stabilization added and proven (loading flakes gone,
24/24 regeneration, 17/24 byte-stable over 16 minutes); the 7 residual flips
are live-QA-database drift (proof: "-54% so tháng trước" vs "+0% so tháng
trước" on the same card) — needs either a data-quiet run window or an owner
ruling to pin data endpoints.
Concerns/Blockers: exact-match baselines cannot survive data drift; pick
option 1 (data-quiet window, zero changes) or option 2 (pin endpoints, needs
ruling); do not keep regenerating — that re-bakes data and re-opens the
treadmill.
