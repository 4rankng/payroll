# ADR-010: Statistical Baseline Over ML for Cash-Readiness Forecasting

**Date:** 2026-07-11
**Status:** Accepted

## Context

The admin had no way to know, ahead of the pay date, how much cash the next weekly bulk transfer would require. The pain (verbatim): *"today is 10 July and admin need to pay 462 million but because there is no forecast so admin has a hard time to prepare the cash."* They discovered the number only when sitting down to export the MBank file.

The question was whether to use ML (Prophet / LightGBM) or a statistical baseline for the forecast.

## Decision

Use a **statistical baseline** (ETS + newsvendor Monte-Carlo) rather than ML, behind a swappable `ForecastProvider` interface so ML can be added later without touching handlers/UI.

### Target-cycle scope clarification (2026-07-17)

The forecast represents only the next Kỳ shown on the card. It must not include
`GetSummaryStats.PendingPaymentAmount` or any outstanding-payment backlog from
other cycles. Its point estimate is:

```text
approved value observed in the target Kỳ
+ projected target-Kỳ approvals through its pay date
```

The interval applies to that same target-Kỳ total. The separate “Chờ thanh toán”
KPI remains an operational backlog metric and is never a forecast input.

### Rationale

1. **The observed target-Kỳ amount is deterministic.** Approved value already recorded inside the target Kỳ is summed directly; only approvals still expected before that Kỳ's pay date are projected. Outstanding payments from other cycles are outside the forecast scope.
2. **Short horizon, near-constant daily amounts.** The projection term covers days (the weekly bulk-transfer cadence) with near-constant per-employee daily amounts (rate × hours). For that shape, a transparent ETS/seasonal-naive average + Monte-Carlo band beats a black box.
3. **Makridakis M-competition evidence.** Simple statistical methods and ensembles match or beat complex ML on aggregate, low-frequency business series. ML wins concentrate in high-frequency, high-dimensional data — the opposite of weekly payroll.
4. **ML needs ≥18 months clean history + a retraining/drift pipeline.** On a short horizon it overfits, and you cannot explain a 40M VND miss — which is exactly what matters when cash is short.

### Validation

The existing test demonstrates rolling-origin methodology with synthetic
fixtures; it is not a production accuracy measurement. Real confidence must be
based on persisted forecast snapshots compared with actual target-Kỳ payouts.

### Escalation Path (Documented, Not Built)

Keep the `ForecastProvider` interface. If the instrumented p50-vs-actual error stays >~12% over ≥4 cycles AND Tết/seasonality is shown to swing payout, add a Prophet/LightGBM provider behind the same interface. No handler or UI changes would be needed.

### Advisory Invariant

The forecast is advisory-only and never feeds `SyncBalance`. A `WalletBalanceReader` port narrows to `GetBalance` only — the service has no type-level reach to `SyncBalance`/`CreateTopup`, making the wallet-balance-inflation regression **structurally impossible** (not just commented).

## Consequences

**Positive:**
- Transparent and explainable — the admin can see exactly how the number is computed.
- No ML infrastructure (training pipeline, model registry, drift monitoring) needed.
- The swappable interface leaves the ML door open without paying its cost now.
- The approach remains simple enough to backtest and explain once real snapshots are available.

**Negative:**
- May underperform ML if Tết/seasonality causes large payout swings in the future.
- The escalation path requires instrumented accuracy tracking (deferred — needs a forecast-snapshot storage decision).

## Alternatives Considered

1. **Prophet (Facebook)** — Rejected for now. Needs ≥18 months clean history and a retraining pipeline. Overfits on short horizons. Available as a drop-in `ForecastProvider` if needed.
2. **LightGBM** — Rejected for now. Same reasons as Prophet. Black-box nature makes it hard to explain a 40M VND miss to the admin.
3. **Pure deterministic (no forecast)** — Rejected. Target-Kỳ approvals observed so far do not account for approvals expected between today and the pay date.
4. **Neural network (LSTM)** — Rejected. Over-provisioned for weekly payroll data. Would require significant data engineering.

## References

- [Journal: Cash-readiness forecast](../journals/2026-07-11-cash-readiness-forecast.md)
- [Lesson: Statistical forecast over ML](../lessons/2026-07-11-statistical-forecast-over-ml.md)
- Plan: `plans/260710-2235-cash-readiness-forecast/`
