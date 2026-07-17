# Cash-readiness forecast for the next timesheet bulk transfer

**Date:** 2026-07-11 · **Commit:** `9a24a05` · **Status:** shipped (Phases 1–5)

> **Scope correction shipped 2026-07-17:** the forecast is target-Kỳ only. The
> current “Chờ thanh toán” amount is a separate backlog KPI and is not added to
> the forecast. The corrected point estimate is target-Kỳ approved value
> observed so far plus projected remaining target-Kỳ approvals.

## Problem
The admin had no way to know, ahead of the pay date, how much cash the next weekly
bulk transfer would require. Pain (verbatim): *"today is 10 July and admin need to
pay 462 million but because there is no forecast so admin has a hard time to prepare
the cash."* They discovered the number only when sitting down to export the MBank file.

## What shipped
A single advisory **"Dự báo tiền trả"** card on `/admin/timesheet` (desktop + mobile)
showing the mathematical point estimate:
`expected_total = observed_target_ky_approved + projected_remaining_mean`, with
`band = [observed_target_ky_approved+p50, observed_target_ky_approved+p95]`
shown beneath it. Outstanding payments, wallet balance, funding shortfall, and advance-payment information
are intentionally kept out of the timesheet page; those belong to their respective
financial workflows.

- `clock/pay_cycle.go` — global 4-cycle pay model (pay days 10/17/24/1; work 1-7/8-14/15-21/22-28). Distinct from the advance-payment day-20→9 cycle.
- `CashReadinessForecastService` behind a swappable `ForecastProvider` (v1 statistical; ML is a drop-in later). Reuses the wallet-forecast newsvendor Monte-Carlo engine — no math reimplemented.
- `GET /api/v1/timesheets/cash-readiness` — partner-scoped, `loc=Local` date parse, casbin rule.
- `CashReadinessCard` + `useCashReadiness` — shared component on both surfaces; no gold tokens.

## Key decisions
- **Statistical baseline over ML.** The target-Kỳ horizon is short and daily amounts are comparatively stable, so a transparent baseline remains preferable until real rolling accuracy data justifies more complexity. The existing backtest uses synthetic fixtures and must not be presented as production accuracy.
- **Advisory invariant enforced structurally.** A `WalletBalanceReader` port narrows to `GetBalance` only — the service has no type-level reach to `SyncBalance`/`CreateTopup`, making the wallet-balance-inflation regression structurally impossible (not just commented).
- **Target-Kỳ isolation.** The service reads approved accrual for the target Kỳ directly from the cohort source. It has no summary-reader dependency, so the sibling “Chờ thanh toán” backlog cannot affect the forecast.
- **No separate backend cache** — the cohort is a scoped indexed `GROUP BY`; avoiding another cache removes stale-forecast risk.

## Verification
Original implementation checks were green for the focused forecast packages and frontend build. The backtest is a synthetic methodology regression, not a claim about production error.

## Deferred (need a product decision)
- Always-on accuracy instrumentation (needs a forecast-snapshot storage decision — the offline backtest covers the same signal meanwhile).
- Standalone `qa/scripts/cash-readiness-backtest` CLI (methodology built + tested; DB-wiring is the remaining plumbing).
- **Deploy check:** `EXPLAIN` the cohort query on first deploy (it reuses the summary's indexed path).
