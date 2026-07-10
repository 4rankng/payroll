# Cash-readiness forecast for the next timesheet bulk transfer

**Date:** 2026-07-11 · **Commit:** `9a24a05` · **Status:** shipped (Phases 1–5)

## Problem
The admin had no way to know, ahead of the pay date, how much cash the next weekly
bulk transfer would require. Pain (verbatim): *"today is 10 July and admin need to
pay 462 million but because there is no forecast so admin has a hard time to prepare
the cash."* They discovered the number only when sitting down to export the MBank file.

## What shipped
A single advisory **"Chuẩn bị tiền trả"** card on `/admin/timesheet` (desktop + mobile)
showing the gap to prepare: `cash_to_prepare = confirmed_payable + projected_p50`,
`band = [confirmed+p50, confirmed+p95]`, `gap = max(0, cash_to_prepare − wallet.Available)`,
plus a "prepare by \<pay date − 2 days\>" line and a confidence badge.

- `clock/pay_cycle.go` — global 4-cycle pay model (pay days 10/17/24/1; work 1-7/8-14/15-21/22-28). Distinct from the advance-payment day-20→9 cycle.
- `CashReadinessForecastService` behind a swappable `ForecastProvider` (v1 statistical; ML is a drop-in later). Reuses the wallet-forecast newsvendor Monte-Carlo engine — no math reimplemented.
- `GET /api/v1/timesheets/cash-readiness` — partner-scoped, `loc=Local` date parse, casbin rule.
- `CashReadinessCard` + `useCashReadiness` — shared component on both surfaces; no gold tokens.

## Key decisions
- **Statistical baseline over ML.** The total is mostly deterministic (approved + unpaid); short horizon, near-constant daily amounts; Makridakis favors simple methods on low-frequency series. **Backtest validated it: 7.3% + 0.0% abs error** forecasting 2 cycles from priors. ML stays a documented escalation path behind `ForecastProvider` if error stays >~12% over ≥4 cycles.
- **Advisory invariant enforced structurally.** A `WalletBalanceReader` port narrows to `GetBalance` only — the service has no type-level reach to `SyncBalance`/`CreateTopup`, making the wallet-balance-inflation regression structurally impossible (not just commented).
- **Confirmed-payable routed through the cached `TimesheetService.GetSummaryStats`** (not the raw repo) so the card's "Đã chốt" line can never diverge from the sibling "Chờ thanh toán" tile.
- **No separate backend cache** — `GetSummaryStats` already caches/invalidates the hot number; the cohort is a scoped indexed `GROUP BY`. Avoids stale-cache risk.

## Verification
`go build` green; `go test` green (clock, services incl. backtest, repositories, domain); `golangci-lint` clean on new code; frontend `tsc` + `vite build` green. Code-review subagent: no critical/high; the one medium (cache divergence) + three cosmetic findings all fixed.

## Deferred (need a product decision)
- Always-on accuracy instrumentation (needs a forecast-snapshot storage decision — the offline backtest covers the same signal meanwhile).
- Standalone `qa/scripts/cash-readiness-backtest` CLI (methodology built + tested; DB-wiring is the remaining plumbing).
- **Deploy check:** `EXPLAIN` the cohort query on first deploy (it reuses the summary's indexed path).
