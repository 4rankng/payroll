# ADR-010: Statistical Baseline Over ML for Cash-Readiness Forecasting

**Date:** 2026-07-11
**Status:** Accepted

## Context

The admin had no way to know, ahead of the pay date, how much cash the next weekly bulk transfer would require. The pain (verbatim): *"today is 10 July and admin need to pay 462 million but because there is no forecast so admin has a hard time to prepare the cash."* They discovered the number only when sitting down to export the MBank file.

The question was whether to use ML (Prophet / LightGBM) or a statistical baseline for the forecast.

## Decision

Use a **statistical baseline** (ETS + newsvendor Monte-Carlo) rather than ML, behind a swappable `ForecastProvider` interface so ML can be added later without touching handlers/UI.

### Rationale

1. **The total is mostly deterministic.** The confirmed-payable half (approved + unpaid timesheets) is a known sum — no model beats "sum the known values." ML contributes zero to the dominant term and obscures it.
2. **Short horizon, near-constant daily amounts.** The projection term covers days (the weekly bulk-transfer cadence) with near-constant per-employee daily amounts (rate × hours). For that shape, a transparent ETS/seasonal-naive average + Monte-Carlo band beats a black box.
3. **Makridakis M-competition evidence.** Simple statistical methods and ensembles match or beat complex ML on aggregate, low-frequency business series. ML wins concentrate in high-frequency, high-dimensional data — the opposite of weekly payroll.
4. **ML needs ≥18 months clean history + a retraining/drift pipeline.** On a short horizon it overfits, and you cannot explain a 40M VND miss — which is exactly what matters when cash is short.

### Validation

**Backtest results: 7.3% + 0.0% absolute error** forecasting 2 cycles from priors.

### Escalation Path (Documented, Not Built)

Keep the `ForecastProvider` interface. If the instrumented p50-vs-actual error stays >~12% over ≥4 cycles AND Tết/seasonality is shown to swing payout, add a Prophet/LightGBM provider behind the same interface. No handler or UI changes would be needed.

### Advisory Invariant

The forecast is advisory-only and never feeds `SyncBalance`. A `WalletBalanceReader` port narrows to `GetBalance` only — the service has no type-level reach to `SyncBalance`/`CreateTopup`, making the wallet-balance-inflation regression **structurally impossible** (not just commented).

## Consequences

**Positive:**
- Transparent and explainable — the admin can see exactly how the number is computed.
- No ML infrastructure (training pipeline, model registry, drift monitoring) needed.
- The swappable interface leaves the ML door open without paying its cost now.
- Backtest validated the approach with low error.

**Negative:**
- May underperform ML if Tết/seasonality causes large payout swings in the future.
- The escalation path requires instrumented accuracy tracking (deferred — needs a forecast-snapshot storage decision).

## Alternatives Considered

1. **Prophet (Facebook)** — Rejected for now. Needs ≥18 months clean history and a retraining pipeline. Overfits on short horizons. Available as a drop-in `ForecastProvider` if needed.
2. **LightGBM** — Rejected for now. Same reasons as Prophet. Black-box nature makes it hard to explain a 40M VND miss to the admin.
3. **Pure deterministic (no forecast)** — Rejected. The confirmed-payable alone doesn't account for accrual between today and the pay date.
4. **Neural network (LSTM)** — Rejected. Over-provisioned for weekly payroll data. Would require significant data engineering.

## References

- [Journal: Cash-readiness forecast](../journals/2026-07-11-cash-readiness-forecast.md)
- [Lesson: Statistical forecast over ML](../lessons/2026-07-11-statistical-forecast-over-ml.md)
- Plan: `plans/260710-2235-cash-readiness-forecast/`
