---
title: "Trend-aware cash-readiness forecast (growth-adjusted MVP)"
description: "Minimal fix for the 5.7× growth problem in the cash-readiness forecast. The deployed flat-gamma fit treats directional growth as random variance (198M forecast for July Ky-2 when June alone was 457M). This plan adds a single growthEWMA helper that adjusts the gamma quantiles when trendRatio ≥ cutoff — ~15 lines in the existing branch, no new structs/providers/endpoints. Cut from 4 phases to 1 after red-team review found the original plan over-engineered (2.4pp accuracy delta for 4-6× the code) and built on a false premise (gamma fits on rem, not grand totals)."
status: pending
priority: P2
branch: "main"
tags: [forecasting, timesheet, cash-readiness, advisory, trend, mvp]
blockedBy: []
blocks: []
created: "2026-07-11T02:54:45.836Z"
createdBy: "ck:plan"
source: skill
---

# Trend-aware cash-readiness forecast (growth-adjusted MVP)

## Overview

### The problem

The deployed cash-readiness forecast fits a gamma distribution to the **remaining-demand** series (`rem`) of historical per-Ky timesheet cohorts. On 2026-07-11 this produced a headline of **198.578.533 ₫** for Ky-2 (pay day 17/07), with a band of **169.393.914–458.294.954 ₫**.

The cohort has a **strong monotonic growth trend**:

| Month | Ky-2 grand total | MoM growth |
|-------|-----------------:|-----------:|
| Jan 2026 | 80,350,500 ₫ | — |
| Feb 2026 | 130,036,050 ₫ | +62% |
| Mar 2026 | 132,502,900 ₫ | +2% |
| Apr 2026 | 158,614,931 ₫ | +20% |
| May 2026 | 240,188,932 ₫ | +51% |
| **Jun 2026** | **457,082,437 ₫** | **+90%** |

The gamma fit sees this 5.7× span as *random variance*, not *directional growth*. The P50 (169M) is dragged down by stale early months. **July Ky-2 is almost certainly higher than June's 457M given the trend** — the forecast undershoots badly.

### The backtest evidence

Walk-forward backtest against real Ky-2 production data (forecast each cycle from priors):

| Method | Mean abs % error | July Ky-2 forecast |
|--------|-----------------:|-------------------:|
| **A. Flat gamma (deployed)** | **41.4%*** | 141M vs 457M actual (−69%) |
| B. Linear trend (OLS) | 29.3% | 252M (−45%) |
| D. Blended (50% trend + 50% recent) | 28.6% | 213M (−53%) |
| **F. EWMA growth-rate × last** | **31.0%** | **314M (−31%)** |

*The 41.4% figure was computed in a local Python backtest on 2026-07-11. It is **not yet reproducible from the repo** — Phase 0 commits the fixture + test that validates it (red-team Finding 2).

### Why MVP, not 4 phases (red-team decision)

The original 4-phase plan (detrend engine + blend provider + accuracy harness + trend UI) was red-teamed by 3 reviewers who independently found:

1. **Critical: false premise.** The plan said "gamma fits on grand totals" — it actually fits on `rem` (windowed remaining-demand delta). The detrend math was designed for the wrong series.
2. **Critical: unfalsifiable blend.** Methods B/D/E/F are statistically indistinguishable (28.6-31.4%, Δ < 3pp on n=5). Three tuning knobs (w₁, w₂, α) on 6 data points cannot be validated.
3. **The 2.4pp accuracy delta between the best blend (D, 28.6%) and the simplest method (F, 31.0%) is inside the noise.** Method F needs ~15 lines; the blend needs 4 phases.

**Decision: ship method F as a single-phase MVP.** Defer detrend/blend/accuracy/UI to a follow-up plan gated on ≥6 months of real accuracy data proving the MVP insufficient.

## Architecture (MVP)

```
 forecastDemandDistributionBetween(historical, fromCycleDay, throughCycleDay, ...)
        │
        ├── [existing] build rem[] (remaining demand per cycle)
        ├── [existing] compute trendRatio = max(rem)/min(rem)
        ├── [existing] fitGammaMoM(rem) → shape, scale
        ├── [existing] MC sample 5000 → sorted samples
        ├── [existing] quantileOfSorted → p50, p95
        │
        ├── [NEW] IF trendRatio >= trendRatioCutoff:
        │     ├── grandTotals = extractChronologicalGrandTotals(historical)
        │     │   (sorted by forMonth — red-team Finding 4)
        │     ├── growthFactor = growthEWMA(grandTotals, alpha)
        │     ├── p50 *= growthFactor
        │     ├── p95 *= growthFactor
        │     ├── for each sample: sample *= growthFactor  (preserve distribution shape)
        │     └── method = "growth-adjusted"
        │
        └── [existing] return demandDistribution{...}
```

**Why multiplicative, not additive:** the gamma captures the *shape* of per-cycle variation. Growth adjusts the *level*. Multiplying the entire sample array by `growthFactor` shifts the level while preserving the skew and tail ratio — no double-counting, no scaled/unscaled mixing (red-team Finding 9).

**Why this doesn't touch the wallet forecast:** the growth adjustment is gated on `trendRatio >= cutoff`, which is computed on `rem`. The wallet forecast passes `paidFrac < 1.0`, but the growth factor is applied uniformly to all samples (including the paidFrac-scaled ones), so the scaling is self-consistent (red-team Finding 9 / wallet collateral). If wallet cohorts are stationary (low trendRatio), the branch never fires.

## Phase

| Phase | Name | Status | Effort |
|-------|------|--------|--------|
| 0 | [Commit foundation + growth-adjusted quantiles](./phase-01-backtest-harness-with-real-production-data.md) | Pending | S |

Single phase. No dependency chain.

## Constraints

- **Advisory only** — never feeds `SyncBalance`/`CreateTopup`/disbursement.
- **All business time via `clock.Now()`** (Asia/Ho_Chi_Minh).
- **Deterministic output** — growthEWMA is closed-form; MC sampling stays seeded.
- **VND, no decimals** in display formatting.
- **gofmt + golangci-lint clean; `pnpm lint` + `tsc --noEmit` clean** before done.
- **The `trendRatio` field + `trendRatioCutoff` constant are currently uncommitted working-tree state** (red-team Finding 3). Phase 0 commits them as a prerequisite.

## Acceptance criteria

- [ ] `trendRatio`, `trendRatioCutoff`, and the `confidenceLabel` trend-aware downgrade are **committed** (not just working-tree state).
- [ ] `growthEWMA(grandTotals, alpha)` helper exists, tested for n=1, n=2, n=6.
- [ ] When `trendRatio >= trendRatioCutoff`, the gamma quantiles are multiplied by `growthFactor`; `method = "growth-adjusted"`.
- [ ] When `trendRatio < cutoff`, output is **identical** to pre-change (no regression on stationary data).
- [ ] `buildTimesheetCohort` returns chronologically sorted series (red-team Finding 4).
- [ ] `growth_rate` field surfaces through `demandDistribution` → `AccrualProjection` → `domain.CashReadiness` → DTO → frontend type (red-team Finding 6: field plumbing).
- [ ] Walk-forward backtest test with real Ky-2 fixture validates the accuracy claim (red-team Finding 2: reproducible baseline).
- [ ] July Ky-2 forecast lands ≥300M (vs current 198M).
- [ ] `go test ./internal/app/services/... -race` green; `golangci-lint run ./internal/app/services/...` clean; `pnpm lint` + `tsc --noEmit` clean.

## Red Team Review

### Session — 2026-07-11
**Findings:** 12 unique (deduplicated from 30 raw across 3 reviewers)
**Accepted:** 12 · **Rejected:** 0
**Severity breakdown:** 3 Critical, 7 High, 2 Medium

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | False premise: gamma fits on `rem`, not grand totals | Critical | Accept | Rewrite: MVP operates on `rem` samples directly |
| 2 | 41.4% baseline unreproducible | Critical | Accept | Phase 0 commits fixture + validation test |
| 3 | trendRatio/trendRatioCutoff uncommitted | Critical | Accept | Phase 0 commits as prerequisite |
| 4 | buildTimesheetCohort unsorted | High | Accept | Phase 0 adds sort + test |
| 5 | DI wiring at services/init.go:514, not container.go | High | Accept | N/A — MVP doesn't add a provider |
| 6 | demandDistribution fields can't reach provider | High | Accept | Phase 0 plumbs growth_rate through the full chain |
| 7 | φ contradicts itself (0.9 vs 0.98) | High | Accept | N/A — MVP has no φ (no OLS) |
| 8 | 3 composition formulas | High | Accept | MVP picks one: multiplicative |
| 9 | Wallet forecast collateral (paidFrac scaling) | High | Accept | MVP applies growth uniformly to samples |
| 10 | prepareByCycleDay doesn't exist | High | Accept | N/A — MVP defers accuracy harness |
| 11 | Damped Holt already exists in dashboard | High | Accept | N/A — MVP doesn't add OLS |
| 12 | 3 tuning knobs unfalsifiable | High | Accept | MVP drops blend; single α knob |

### Whole-Plan Consistency Sweep
- Replaced 4-phase detrend+blend+harness+UI structure with single-phase MVP.
- Removed all references to: detrend engine, OLS, TrendAwareAccrualProvider, accuracy endpoint, Redis snapshots, asynq jobs, JSON-lines logger, trend UI badge.
- "grand totals" language corrected to "remaining-demand series (rem)" throughout.
- Composition formula unified to multiplicative (growthFactor × samples).
- Zero unresolved contradictions.

## Deferred to follow-up plan (gated on ≥6 months accuracy data)

- Detrended gamma engine (OLS on residuals)
- Blended ForecastProvider (trend + growth)
- Always-on accuracy instrumentation (Redis/asynq)
- Trend UI (growth badge, accuracy footer)
- ML escalation (Prophet/LightGBM)

## Risks

- **EWMA overshoots on mean-reversion.** If July drops to 250M, the growth-adjusted forecast (~314M) overshoots by ~25%. *Mitigation:* the band (p50–p95) communicates the uncertainty; the confidence label already downgrades to "medium" on high trendRatio. This is honest — overshooting high is safer for cash preparation than undershooting.
- **6 data points can't validate α.** The EWMA smoothing factor (default 0.5) is a guess. *Mitigation:* it's one config field, env-tunable, with no claim of optimality. Real validation needs ≥8 cycles.
- **Method F is not the best method.** The blend (D) is 2.4pp better. *Mitigation:* the delta is inside the noise for n=5. When more data arrives, the follow-up plan can revisit.

## Dependencies

- Builds on the `trendRatio` field and `trendRatioCutoff` constant (currently uncommitted — Phase 0 commits them).
- Follows `plans/260710-2235-cash-readiness-forecast` (initial implementation, done).
- No cross-plan overlap.
