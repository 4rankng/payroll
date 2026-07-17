# Reliable cash-readiness forecast stopped collapsing to zero

**Date**: 2026-07-17 20:39
**Severity**: High
**Component**: Cash readiness forecast / bulk-transfer measurement
**Status**: Resolved

## What Happened

The cash-readiness forecast could still collapse to zero when the target Ky had no rows yet or only had partially observed data. The old logic effectively trusted in-progress target-cycle rows and had no honest fallback for a fully completed historical cycle, so `forecastCashReadinessV2` could end up with an empty observation set and fall back to `no-history` even though comparable completed Kys existed. That was a bad answer, not a conservative one.

I also had to tighten the measurement path at the same time: the weekly export outcome needed to be reconciled against distinct timesheet IDs, forecast snapshots needed to become immutable once an actual payout was attached, and the weekly payment percentage had to apply to the full forecast basis consistently instead of only to one slice of the result.

## The Brutal Truth

This was maddening because the system was not “slightly off” - it was confidently returning nothing useful in the exact window where admins need a number they can trust. A zero forecast is worse than an approximate one because it looks intentional. The bug was not in the math alone; it was in the assumption that every cycle would already have observable target-row data. That assumption was wrong.

## Technical Details

- Root cause: `buildCashCycleObservations` filtered out target cycles with no observed rows, so the forecast could degenerate into `no-history` and a zero band.
- Fix: add a `completed-cycle-bootstrap` fallback that uses the final approved total from completed Ky when the target cycle has no rows yet.
- The weekly payment percentage now scales the whole forecast basis through `scaleCashReadinessProjection`, so approved, pending, expected future, reserve, and the legacy band all use the same payable basis.
- Forecast snapshots are persisted through `cash_forecast_snapshots` with a unique company-cycle identity, and `Upsert` now refuses to rewrite resolved history.
- Weekly export reconciliation now deduplicates by exact work-date range plus `TimesheetID`, so split exports and retries do not double-count the actual payout.

## What We Tried

- I first traced the zero output back through `cash_readiness_forecast.go` and the new V2 math path.
- Then I codified the failure modes in `cash_readiness_forecast_v2_test.go`, `cash_forecast_snapshot_repository_test.go`, and `cash_forecast_accuracy_handler_test.go`.
- I rejected any “just widen the cohort” fix because that would have reintroduced cross-cycle contamination and the old backlog leakage.

## Root Cause Analysis

The fundamental mistake was treating incomplete target-cycle data as if it were the only valid signal. That made the forecast brittle and silently wrong. The measurement layer had a second mistake: it treated snapshots like mutable cache entries instead of historical records, which would have let later reads rewrite an already resolved outcome.

## Lessons Learned

- If a forecast can be zero because the current cycle is still empty, you do not have a forecast.
- Completed historical cycles are not optional fallback data; they are the only honest basis when the current Ky has not started filling in yet.
- Measurement data must be immutable after resolution. Otherwise calibration turns into moving-target nonsense.
- Reconciliation should key on distinct timesheet outcomes, not just “whatever the latest export file said.”

## Next Steps

- Keep the focused backend regression set running in CI for the forecast, snapshot repository, and export handler paths.
- Watch the repo-wide backend suite separately; the active bulk-transfer split plan still notes unrelated baseline failures in `bootstrap/config`, and those are not caused by this work.
- Owner: backend payroll. Verify the next production export resolves against the same exact work-date range that generated the snapshot.
