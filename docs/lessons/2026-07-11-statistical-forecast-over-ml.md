# Statistical Forecast Over ML

**Date:** 2026-07-11
**Source:** Commit `9a24a05`, journal `2026-07-11-cash-readiness-forecast.md`
**Tags:** [forecasting, ml, statistics, payroll, decision-making]

## Context

The admin needed a forecast of how much cash to prepare for the next weekly bulk transfer. The question arose: should we use ML (Prophet / LightGBM) or a simpler statistical approach?

The total amount (462M VND) was mostly deterministic — approved + unpaid timesheets. Only the projected accrual between today and the pay date was uncertain.

## Decision / Outcome

Chose a **statistical baseline** (ETS + newsvendor Monte-Carlo) over ML. Backtest validated: **7.3% + 0.0% absolute error** forecasting 2 cycles from priors.

ML was kept as a documented escalation path behind a swappable `ForecastProvider` interface — if error stays >~12% over ≥4 cycles, a Prophet/LightGBM provider can be added without touching handlers or UI.

## Lesson

**ML is not the default answer for forecasting.** For series that are:
- Short horizon (days, not months)
- Near-constant per-period amounts (rate × hours)
- Low frequency (weekly, not hourly)
- Mostly deterministic (the dominant term is a known sum)

...simple statistical methods (ETS, seasonal-naive, Monte-Carlo bands) match or beat ML, are transparent and explainable, and require no training pipeline or drift monitoring.

**Generalized rule:** Before reaching for ML, ask:
1. Is the dominant term deterministic? (If yes, ML adds nothing to it.)
2. Is the horizon short and the variance low? (If yes, simple methods win.)
3. Can you explain a miss to a non-technical stakeholder? (If not, the black box is a liability.)
4. Do you have ≥18 months of clean history for training? (If not, ML will overfit.)

ML wins when the data is high-frequency, high-dimensional, and the signal is hard to model with simple statistics. Payroll is the opposite.

## References

- [Journal: Cash-readiness forecast](../journals/2026-07-11-cash-readiness-forecast.md)
- [ADR-010: Statistical forecast over ML](../decisions/ADR-010-statistical-forecast-over-ml.md)
- Plan: `plans/260710-2235-cash-readiness-forecast/`
- Code: `backend/internal/app/services/` (CashReadinessForecastService), `backend/internal/domain/` (ForecastProvider interface)
