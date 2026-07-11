---
phase: 1
title: "Commit foundation + growth-adjusted quantiles"
status: pending
priority: P2
dependencies: []
---

# Phase 1: Commit foundation + growth-adjusted quantiles

## Overview

Single-phase MVP. Commits the uncommitted `trendRatio`/`trendRatioCutoff` foundation, adds a `growthEWMA` helper (~15 lines), multiplies the gamma quantiles by the growth factor when `trendRatio >= cutoff`, plumbs `growth_rate` through the full DTO chain, adds a real-fixture backtest test, and sorts `buildTimesheetCohort` chronologically.

No new providers, no new structs beyond a field, no new endpoints, no new config knobs beyond one (`GrowthEWMAlpha`).

## Requirements

- **Functional:**
  - Commit the uncommitted `trendRatio` field, `trendRatioCutoff` constant, and `confidenceLabel` trend-aware downgrade (currently working-tree state).
  - Add `growthEWMA(grandTotals []int64, alpha float64) float64` — EWMA of the last 3 MoM growth rates, seeded to 1.0 (no growth).
  - In `forecastDemandDistributionBetween`, when `trendRatio >= trendRatioCutoff`: multiply all MC samples (and p50/p95/p90/p99) by `growthFactor`; set `method = "growth-adjusted"`.
  - `buildTimesheetCohort` returns chronologically sorted series (sort by `forMonth`).
  - `growth_rate` surfaces: `demandDistribution.growthFactor` → `AccrualProjection.GrowthRate` → `domain.CashReadiness.GrowthRate` → DTO → frontend type.
  - Walk-forward backtest test using the real Ky-2 production fixture validates the accuracy claim.

- **Non-functional:**
  - `growthEWMA` is O(n) closed-form — negligible cost.
  - Deterministic: same cohort + seed → same output.
  - When `trendRatio < cutoff`, output is byte-identical to pre-change.

## Architecture

### The growthEWMA helper

```go
// growthEWMA computes the exponentially-weighted moving average of the last 3
// month-over-month growth rates from grandTotals (chronological). Returns a
// multiplier (e.g., 1.5 = 50% expected growth). Seeded to 1.0 (no growth) when
// fewer than 2 data points exist.
func growthEWMA(grandTotals []int64, alpha float64) float64 {
    if len(grandTotals) < 2 || alpha <= 0 {
        return 1.0
    }
    // Collect MoM growth rates, take last 3.
    var rates []float64
    for i := 1; i < len(grandTotals); i++ {
        if grandTotals[i-1] > 0 {
            rates = append(rates, float64(grandTotals[i])/float64(grandTotals[i-1]))
        }
    }
    if len(rates) == 0 {
        return 1.0
    }
    // Take last 3 (or fewer if not enough).
    start := len(rates) - min(3, len(rates))
    rates = rates[start:]
    // EWMA seeded to the first rate in the window.
    ewma := rates[0]
    for _, r := range rates[1:] {
        ewma = alpha*r + (1-alpha)*ewma
    }
    return ewma
}
```

### Where it's called

In `forecastDemandDistributionBetween`, after the existing MC sampling + quantile computation, before returning:

```go
// Growth adjustment: when the basis shows directional growth (trendRatio >=
// cutoff), the gamma fit treats the growth as variance. Multiply the
// distribution by the EWMA growth factor to re-center on the trend.
growthFactor := 1.0
method := "monte-carlo"
if len(rem) < 3 {
    method = "gamma-fit"
}
if dist.trendRatio >= trendRatioCutoff {
    grandTotals := chronologicalGrandTotals(historical)
    growthFactor = growthEWMA(grandTotals, growthEWMAlpha)
    for i := range samples {
        samples[i] *= growthFactor
    }
    p50 *= growthFactor
    p90 *= growthFactor
    p95 *= growthFactor
    p99 *= growthFactor
    empiricalMaxCash *= growthFactor
    method = "growth-adjusted"
}
```

**Why multiply the entire sample array:** this preserves the distribution shape (skew, tail ratio) while shifting the level. The `divergent` flag + `empiricalMax` floor in `newsvendorRecommendation` remain self-consistent because `empiricalMaxCash` is also scaled (red-team Finding: wallet collateral).

**Why `grandTotals` for growth but `rem` for gamma:** `rem` is the remaining-demand delta (what the gamma fits). `grandTotals` is the full-cycle total (what growth rate measures). The growth rate is a property of the cycle, not the window. Using `grandTotals` for the EWMA is correct even though gamma operates on `rem` — they measure different things.

### Chronological sort

```go
// In buildTimesheetCohort, before returning:
sort.Slice(series, func(i, j int) bool {
    return series[i].forMonth < series[j].forMonth
})
```

## Related Code Files

- **Modify:** `backend/internal/app/services/wallet_demand_forecast_math.go`
  - Add `growthEWMA` helper.
  - Add `growthFactor float64` to `demandDistribution`.
  - Add the growth-adjustment branch in `forecastDemandDistributionBetween`.
  - Add `chronologicalGrandTotals([]cohortSeries) []int64` helper.
- **Modify:** `backend/internal/app/services/cash_readiness_forecast.go`
  - Add `GrowthRate float64` to `AccrualProjection`.
  - Map `dist.growthFactor` → `proj.GrowthRate` in `TimesheetAccrualProvider.ProjectAccrual`.
  - Sort `buildTimesheetCohort` output by `forMonth`.
- **Modify:** `backend/internal/domain/cash_readiness.go` — add `GrowthRate float64` to `CashReadiness`.
- **Modify:** `backend/internal/app/dto/cash_readiness.go` — add `GrowthRate` to the DTO + response mapping.
- **Modify:** `backend/internal/config/config.go` — add `GrowthEWMAlpha float64` to `CashForecastConfig` (default 0.5, validation (0, 1]).
- **Modify:** `frontend/src/types/api/cash-readiness.types.ts` — add `growth_rate: number`.
- **Modify:** `backend/internal/app/services/wallet_demand_forecast_math_test.go` — add growthEWMA + growth-adjustment tests.
- **Modify:** `backend/internal/app/services/cash_readiness_backtest_test.go` — add real Ky-2 fixture backtest.

## Implementation Steps

1. **Commit the foundation** — stage and commit the uncommitted `trendRatio`, `trendRatioCutoff`, `trendRatio` field on `demandDistribution`, the `confidenceLabel` trend-aware downgrade, and the `CashReadinessCard` confidence UI. This is a prerequisite commit (red-team Finding 3).

2. **Add `growthEWMAlpha` to config** — `CashForecastConfig.GrowthEWMAlpha float64`, default 0.5. Add validation in the existing `validate()` function: `GrowthEWMAlpha` in (0, 1], default 0.5 when 0. No new env var required (uses the existing `CASH_FORECAST_*` prefix pattern if exposed).

3. **Add `growthEWMA` helper** — the function above. Pure, I/O-free, tested.

4. **Add `chronologicalGrandTotals` helper** — extracts `grandTotal` from each `cohortSeries`, sorted by `forMonth`. Returns `[]int64`.

5. **Add the growth-adjustment branch** — in `forecastDemandDistributionBetween`, after the existing quantile computation. Multiply samples + quantiles + empiricalMaxCash by `growthFactor` when `trendRatio >= trendRatioCutoff`. Set `method = "growth-adjusted"`.

6. **Add `growthFactor` to `demandDistribution`** — so the caller can surface it.

7. **Sort `buildTimesheetCohort`** — `sort.Slice` by `forMonth` before returning. Add a test asserting chronological order (red-team Finding 4).

8. **Plumb `growth_rate` through the chain:**
   - `TimesheetAccrualProvider.ProjectAccrual`: map `dist.growthFactor` → `proj.GrowthRate` (red-team Finding 6).
   - `CashReadinessForecastService.GetCashReadiness`: map `proj.GrowthRate` → `result.GrowthRate`.
   - `domain.CashReadiness`: add `GrowthRate float64`.
   - DTO + handler: add `growth_rate` to the JSON response.
   - Frontend type: add `growth_rate: number`.

9. **Write the real-fixture backtest** — in `cash_readiness_backtest_test.go`, add a test with the real Ky-2 grand totals (80M, 130M, 132M, 158M, 240M, 457M). Walk-forward: forecast each cycle from priors, log abs % error. This validates the 41.4% baseline claim and the improvement (red-team Finding 2).

10. **Write tests:**
    - `TestGrowthEWMA_KnownRates` — verify EWMA computation on known growth rates.
    - `TestGrowthEWMA_EdgeCases` — n=0, n=1, n=2, n=6; zero grandTotals; negative (clamp).
    - `TestGrowthAdjustment_TrendingData` — trendRatio ≥ cutoff → samples scaled; method = "growth-adjusted".
    - `TestGrowthAdjustment_StationaryUnchanged` — trendRatio < cutoff → identical output.
    - `TestBuildTimesheetCohort_ChronologicallySorted` — verify sort order.
    - `TestBacktest_RealKy2Data` — the real-fixture walk-forward test.

## Success Criteria

- [ ] `trendRatio`, `trendRatioCutoff`, and confidence downgrade are committed (not working-tree state).
- [ ] `growthEWMA` helper exists and passes edge-case tests.
- [ ] Growth-adjustment branch fires when `trendRatio >= trendRatioCutoff`; `method = "growth-adjusted"`.
- [ ] Stationary data produces byte-identical output to pre-change.
- [ ] `buildTimesheetCohort` returns chronologically sorted series (test verifies).
- [ ] `growth_rate` surfaces through the full chain: math → provider → domain → DTO → frontend type.
- [ ] Real Ky-2 backtest validates the accuracy claim (baseline error documented; growth-adjusted error lower).
- [ ] July Ky-2 forecast ≥300M (vs current 198M).
- [ ] `go test ./internal/app/services/... -race` green.
- [ ] `golangci-lint run ./internal/app/services/...` clean.
- [ ] `pnpm lint` + `tsc --noEmit` clean.

## Risk Assessment

- **EWMA overshoots on mean-reversion.** If July drops to 250M, forecast (~314M) overshoots ~25%. *Mitigation:* band + confidence label communicate uncertainty; overshooting high is safer for cash prep.
- **α is a guess on 6 data points.** *Mitigation:* one config field, env-tunable, no claim of optimality. The follow-up plan validates with real data.
- **Growth adjustment changes the `divergent` semantics.** *Mitigation:* `empiricalMaxCash` is also scaled, so the `divergent` comparison (p95 vs empiricalMax) remains self-consistent — both grow by the same factor.
