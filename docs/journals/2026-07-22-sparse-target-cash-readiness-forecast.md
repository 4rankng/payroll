# Sparse target stopped collapsing the cash-readiness forecast

**Date**: 2026-07-22 21:24
**Severity**: High
**Component**: Cash readiness forecast v2 math
**Status**: Resolved

## What Happened

The sparse-target forecast was collapsing to the wrong scale as soon as the current Ky had only a first batch of rows. In the bad case, the model produced `85,814,063` while nearby historical payable cycles were still landing in the `363M-418M` range. That was not a conservative estimate; it was a broken one that looked plausible enough to hide.

## The Brutal Truth

This was a stupidly expensive assumption bug. The code treated "some target rows exist" as proof that the current cycle should be scaled off that tiny sample, even when the only honest basis was the recent completed-cycle history. The result was a forecast that undercut reality by multiples and still printed like a normal answer.

## Technical Details

- Root cause: the v3 path only sampled completed totals when the target was completely empty. A sparse target could still be labelled `completed-cycle-bootstrap` while taking the target-present scaling branch, so `targetHeadcount` effectively became `1` instead of the recent completed-cycle workforce scale.
- v4 fixes that by introducing `bootstrapCompletedTotals` for both empty targets and targets whose usable history is only completed-cycle fallback data.
- When that flag is set, the sampler now uses `completedTotal - approved - pending` instead of re-scaling the whole completed amount, which avoids double-counting the visible target-cycle rows.
- `cashReadinessModelVersion` was bumped to `cash-readiness-v4`, so the corrected calibration/snapshot bucket stays partitioned from the old v3 residuals.
- Regression proof: the new synthetic test `TestCashReadinessV2_SparseTargetKeepsCompletedCycleScale` locks the sparse case at `expectedFuture=900`, `expectedPayout=950`, and `recommendedReserve=1000` on a ten-person completed-cycle scale.

## What We Tried

- I reproduced the failure with a synthetic sparse target that had one current-cycle row and only completed-cycle historical shapes.
- I kept the fix narrowly inside `cash_readiness_forecast_v2_math.go` and backed it with a focused service test instead of widening the cohort or loosening the thresholds.

## Root Cause Analysis

The model confused "current cycle has started" with "current cycle has enough signal to anchor itself." That is the whole failure. A single observed row should not override completed-cycle scale when every usable historical shape is still a completed-cycle fallback.

## Lessons Learned

- Sparse target data is not a valid scale anchor.
- Forecast math must use completed-total semantics whenever completed-cycle fallbacks are the only usable historical shape.
- Model-version partitioning matters; otherwise the fix and the broken history contaminate each other.

## Next Steps

- Keep the focused backend regression in CI.
- I ran `go test ./internal/app/services -run 'Test(BuildCashCycleObservations|CashReadinessV2|GetCashReadinessV2)' -count=1`, and it passed.
- I did not claim production impact here; deployment still needs to happen before anyone treats this as a live fix.
- The wider backend baseline still has unrelated noise, so do not read this as a full repo-green signal.
