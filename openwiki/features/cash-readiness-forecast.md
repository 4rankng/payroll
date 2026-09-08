---
type: feature
title: Cash Readiness Forecast
description: Statistical forecast pipeline that tells the admin how much cash to prepare for the next weekly bulk transfer, behind a swappable ForecastProvider.
tags: [forecast, cash-readiness, statistical, monte-carlo, newsvendor, advisory]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-1f9c70272650e1d90f1d73c0
    resource: repo://backend/internal/domain/cash_forecast_snapshot.go
  - id: openwiki-source-a867680aa327d0b1e98d21cb
    resource: repo://docs/decisions/ADR-010-statistical-forecast-over-ml.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Cash Readiness Forecast

The Cash Readiness Forecast tells the admin, days before the pay date, how much cash to prepare for the next weekly bulk transfer. Before it existed the admin discovered the number only when exporting the MBank file — sometimes 462 million VND with hours of notice. The pipeline combines the target Kỳ's already-approved value with a short-horizon projection of approvals still expected before the pay date; the result is displayed on the dashboard alongside the live wallet balance so the gap is visible immediately.

The forecast is **advisory only**. It never feeds `SyncBalance`, `CreateTopup`, or any disbursement. A narrowed `WalletBalanceReader` port enforces this at the type level — see [Advisory invariant](#advisory-invariant).

## Why statistical, not ML

<!-- openwiki: broken internal link [../decisions/ADR-010-statistical-forecast-over-ml.md] file "../decisions/ADR-010-statistical-forecast-over-ml.md" does not exist. Fix the href or restore the target, then delete this comment. -->
The choice is documented in [ADR-010](../decisions/ADR-010-statistical-forecast-over-ml.md). Three reasons drove the decision:

1. **The target-Kỳ observed value is deterministic.** Approved value already inside the target cycle is summed directly; only approvals still expected before the pay date are projected. Outstanding payments from other cycles are out of scope — they belong to the operational "Chờ thanh toán" KPI, never the forecast input.
2. **Short horizon, near-constant daily amounts.** Weekly bulk transfer with rate × hours per day is the textbook case where a transparent ETS baseline plus a Monte-Carlo band beats a black-box model, both in interpretability and in out-of-sample accuracy on aggregate business series (Makrid­akis M-competition evidence).
3. **ML cost.** Prophet and LightGBM need ≥18 months of clean history and a retraining/drift pipeline. On a short horizon they overfit, and there is no way to explain a 40M VND miss to an admin making a cash call.

The implementation sits behind a `ForecastProvider` interface, so a Prophet or LightGBM provider can be dropped in later without changing handlers or UI — but only after instrumented p50-vs-actual error stays above ~12% for four or more cycles AND Tết/seasonality is shown to swing payouts.

## Forecast shape

`CashReadiness` (`repo://backend/internal/domain/cash_readiness.go#L11-L77`) is the in-memory advisory payload returned to handlers. Key fields:

- `ObservedApproved` — deterministic sum of approved value already recorded inside the target Kỳ.
- `ProjectedP50`, `ProjectedExpected`, `ProjectedP95` — median, mean, and tail projection of additional approvals before the pay date.
- `ExpectedTotal` — point estimate for the target-Kỳ payout, observed + projected p50.
- `RecommendedReserve` — operational service-level quantile (newsvendor style). `IntervalLower` / `IntervalUpper` are the p50/p95 band.
- `Method` — `monte-carlo` | `gamma-fit` | `no-history` | `growth-adjusted`.
- `Confidence` — `high` | `medium` | `low`, derived from sample count and MC/gamma agreement.
- `CalibrationSamples`, `ReliabilityState`, `AccuracyWAPE`, `AccuracyBias`, `IntervalCoverage`, `ReserveShortfallRate` — measured against same-horizon resolved outcomes.
- `CashToPrepare` and `Gap` — wire the forecast to the live wallet. `WalletAvailableOK` flags reads that failed so the UI can show "wallet read failed" instead of a silently-zero balance.

## Snapshot persistence

Every company-wide point-in-time forecast is persisted as a `CashForecastSnapshot` (`repo://backend/internal/domain/cash_forecast_snapshot.go#L26-L51`) keyed by `(scope_key, cycle_key, cycle_day, model_version)`. Resolved snapshots carry the actual payout amount, outcome source, and resolution timestamp; they are immutable so later dashboard reads cannot rewrite history.

Scope is restricted to `CashForecastCompanyScope = "company"` — filtered forecasts must never be reconciled against a company-wide transfer export.

The `CashForecastOutcomeItem` records validated timesheets included in a generated weekly bank file so retries and split exports deduplicate by timesheet ID.

## v4 model semantics

Model version `cash-readiness-v4` tightens the completed-cycle fallback so sparse target rows do not collapse the forecast scale. When the usable historical shapes are all completed-cycle fallbacks:

- The target headcount keeps the recent final workforce scale instead of dropping to the first observed row count.
- The projected future amount is sampled as `completed_cycle_total − current_approved − full_pending_exposure`, so the forecast does not double-count amounts already visible in the target cycle.

The `ModelVersion` column keeps v4 accuracy isolated from v3. Older resolved snapshots stay out of v4's calibration metrics.

## Validation gates

Reliability is reported only after enough resolved samples:

- `<12` resolved observations → `uncalibrated` (UI shows no accuracy figures).
- `12–23` → `learning`.
- `≥24` → reliability reported via WAPE, bias, interval coverage, reserve-shortfall rate.

Comparisons use the configured weekly payable percentage so accuracy is measured in actual cash-transfer units rather than gross timesheet value, and only at the same days-to-pay horizon as the live forecast.

When analogous in-progress historical cycles are unavailable (for example legacy data entered after the original cycle day), the estimator uses completed comparable cycles and labels the method `completed-cycle-bootstrap`. This prevents a misleading all-zero projection while preserving an explicit low-confidence state.

For an entirely empty target Kỳ, the bootstrap uses full completed-cycle totals normalized per final participating employee. Workforce scale is an EWMA of recent completed-cycle participation; median-headcount across the entire six-month window materially underforecasts during rapid workforce growth, so recency is deliberate.

## Advisory invariant

The forecast is structurally prevented from driving wallet actions. The service consumes a narrowed `WalletBalanceReader` port that exposes only `GetBalance` — there is no type-level path from the forecast service to `SyncBalance` or `CreateTopup`. The wallet-balance-inflation regression cannot happen by construction, not by comment.

## Related pages

- [Transaction Manager and Outbox](../architecture/transaction-manager-and-outbox.md) — how the forecast read participates in the request lifecycle.
- [Salary Disbursement and Payment Providers](./salary-disbursement-and-payment-providers.md) — the bulk-transfer pipeline the forecast prepares cash for.
- [Timesheet Engine](./timesheet-engine.md) — approvals feed the observed-approved portion of the forecast.
