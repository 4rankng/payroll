---
title: "Cash-readiness forecast for the next timesheet payment (/admin/timesheet)"
description: "Gives the admin a purpose-built 'how much cash to prepare for the next weekly bulk transfer' metric on /admin/timesheet. Bottom-up: confirmed-payable (already-approved+unpaid timesheets, the deterministic ~70-90% of the total) plus a short-horizon projected-accrual band, minus wallet balance = cash gap to prepare. Reuses the existing newsvendor Monte-Carlo engine on a NEW timesheet-accrual cohort (the wallet forecast's cohort is advance-payment-only and is the wrong data source for this problem). Advisory-only; never feeds SyncBalance. Statistical baseline chosen over ML (Prophet/LightGBM) — see Decision."
status: in-progress
priority: P2
branch: "main"
tags: [forecasting, timesheet, admin, payroll, cash-readiness, advisory]
blockedBy: []
blocks: []
created: "2026-07-10T14:34:32.270Z"
createdBy: "ck:plan"
source: skill
---

# Cash-readiness forecast for the next timesheet payment (/admin/timesheet)

## Overview

**The pain (verbatim):** *"today is 10 July and admin need to pay 462 million but because there is no forecast so admin has a hard time to prepare the cash."*

The admin cannot pre-arrange wallet liquidity because nothing tells them, **N days before the pay date**, how much cash the next weekly bulk transfer will require. They find out the number only when they sit down to export the MBank file.

**The key finding from research:** the system is *not* missing a forecasting engine. It already has two:
- Damped-Holt ETS (`backend/internal/app/services/dashboard/forecasting.go`), fed by `getWeeklyPayHistory`, served via `GET /historical` — a 24-step trend-projection dashboard.
- Newsvendor Monte-Carlo (`backend/internal/app/services/wallet_demand_forecast_math.go`) — advisory wallet top-up for **advance payments**.

And the **confirmed-payable** half of the answer already exists: `GetSummaryStats` returns `PendingPaymentAmount` (approved + unpaid timesheets), served at `GET /api/v1/timesheets/summary` and shown on `/admin/timesheet` as the **"Chờ thanh toán"** card.

What is actually missing is a **purpose-built operational metric** that the existing pieces don't compose:
1. The confirmed-payable number is shown **flat** — no projection of accrual between today and the pay date.
2. No **wallet-balance / gap** context ("you have W, you'll need X, prepare X−W").
3. No **confidence band** (p50 / p95) on the projected part.
4. No **lead-time framing** ("prepare by date D").

This plan builds exactly that: a **CashReadinessForecastService** that takes the existing confirmed-payable figure, adds a short-horizon projected-accrual band (reusing the newsvendor-MC engine on a new **timesheet-accrual cohort**), subtracts the live wallet balance, and surfaces a single "cash to prepare" card on `/admin/timesheet` (desktop + mobile).

## Decision: statistical baseline, not ML (locked)

User asked whether an ML method (Prophet / LightGBM) would be more reliable. **Answer: no, for this data shape.** Locked approach = statistical baseline, behind a swappable `ForecastProvider` interface so ML can be added later without touching handler/UI.

Why ML loses here:
- The 462M is **mostly deterministic** (approved+unpaid timesheets). No model beats "sum the known values." ML contributes zero to the dominant term and obscures it.
- The projection term has a **short horizon** (days, the weekly bulk-transfer cadence) and **near-constant per-employee daily amounts** (rate × hours). For that shape a transparent ETS/seasonal-naive average + Monte-Carlo band beats a black box.
- **Makridakis M-competition evidence:** simple statistical methods and ensembles match or beat complex ML on aggregate, low-frequency business series; ML wins concentrate in high-frequency, high-dimensional data — the opposite of weekly payroll.
- ML needs **≥18 months clean history** + a retraining/drift pipeline; on a short horizon it overfits and you cannot explain a 40M miss, which is exactly what matters when cash is short.

Escalation path (documented, not built now): keep the `ForecastProvider` interface; if the instrumented p50-vs-actual error (Phase 5) stays >~12% over several cycles AND Tết/seasonality is shown to swing payout, add a Prophet/LightGBM provider behind the same interface.

## Architecture

```
 confirmed-payable (DETERMINISTIC)        projected accrual (STOCHASTIC, short horizon)
 ─────────────────────────────────        ───────────────────────────────────────────
 GetSummaryStats.PendingPaymentAmount  +  timesheet-accrual cohort  ──►  newsvendor MC
 (approved+unpaid timesheets,             (NEW: daily cumulative approved             (reuse
  already computed)                        pay per cycle, N historical cycles)        forecastDemandDistributionBetween)
                                                │
                                                ▼
                                   p50 projection + [p50, p95] band
                                                │
              cash_to_prepare  =  confirmed_payable + projected_p50
              gap              =  cash_to_prepare − wallet.Available
              prepare_by_date  =  next_pay_date − CASH_FORECAST_LEAD_DAYS
                                                │
                                                ▼
                        GET /api/v1/timesheets/cash-readiness  (advisory, short-TTL cache)
                                                │
                                                ▼
                        "Chuẩn bị tiền trả" card on /admin/timesheet (+ mobile)
```

**Reuse, do not rebuild:**
- Math: `forecastDemandDistributionBetween`, `newsvendorRecommendation`, `confidenceLabel`, `forecastSeed` (`wallet_demand_forecast_math.go`). These are I/O-free and period/seed-parameterised — call them directly on the new timesheet cohort.
- Confirmed-payable: `GetSummaryStats` already returns it; do not recompute.
- Config pattern: mirror `WalletForecastConfig` (`config.go:198`, env `WALLET_FORECAST_*`) as a new `CashForecastConfig` (env `CASH_FORECAST_*`).
- Wallet balance: `wallet.WalletService.GetBalance(ctx).Available` (same call the wallet forecast already makes).
- Frontend card: reuse `WalletDemandCard` styling/CI-display patterns (no gold tokens — see Constraints).
- **Discipline:** advisory-only, MUST NOT feed `SyncBalance` / `CreateTopup` / any auto top-up (same invariant as the wallet demand forecast — re-introducing it would resurrect the wallet-balance-inflation bug).

**Do NOT reuse (wrong data source):** `AdvancePaymentRequestRepository.GetCohortByMonths` and the whole `WalletDemandForecastService` input pipeline. That cohort is **advance-payment** net cash-out (Nhận lương sớm 24/7 disbursements). The 462M is **timesheet payroll** (weekly bulk transfer, MBank/VFIC export). Different outflow stream — the new service needs a timesheet-accrual cohort built in Phase 1.

## Data sources & wiring points (verified)

| Concern | Location |
|---|---|
| Confirmed-payable figure | `timesheetService.GetSummaryStats(ctx, filters).PendingPaymentAmount` → `dto.TimesheetSummaryResponse` |
| Summary endpoint (admin) | `backend/internal/transport/http/handlers/timesheet/timesheet_summary_handler.go::GetSummary` → `GET /api/v1/timesheets/summary` |
| MC math to reuse | `backend/internal/app/services/wallet_demand_forecast_math.go` |
| ETS cross-check / fallback | `backend/internal/app/services/dashboard/forecasting.go::HoltWintersForecaster` + `getWeeklyPayHistory` |
| Wallet balance | `wallet.WalletService.GetBalance` |
| Config to mirror | `backend/internal/config/config.go:198` (`WalletForecastConfig`), populated at `config.go:419` |
| DI: dashboard Service | `backend/internal/app/services/dashboard/service.go::Service` struct |
| DI wiring | `backend/internal/app/bootstrap/services/init.go`, `container.go` |
| Frontend page (desktop) | `frontend/src/pages/admin/TimesheetPage/index.tsx` |
| Frontend page (mobile) | `frontend/src/pages/mobile/admin/TimesheetPage/index.tsx` |
| Frontend stats hook | `frontend/src/hooks/useTimesheetStatsConfig.ts`, `frontend/src/hooks/api/useTimesheets.ts::useTimesheetSummary` |
| Card styling reference | `frontend/src/components/wallet/WalletDemandCard.tsx` |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Timesheet cash-flow cohort + confirmed-payable query](./phase-01-timesheet-cash-flow-cohort-confirmed-payable-query.md) | Done |
| 2 | [CashReadinessForecastService + ForecastProvider interface](./phase-02-cashreadinessforecastservice-forecastprovider-interface.md) | Done |
| 3 | [HTTP endpoint + DI wiring + caching](./phase-03-http-endpoint-di-wiring-caching.md) | Done (no backend cache — see notes) |
| 4 | [Admin timesheet cash-readiness card (desktop + mobile sync)](./phase-04-admin-timesheet-cash-readiness-card-desktop-mobile-sync.md) | Done |
| 5 | [Tests + timezone safety + accuracy instrumentation](./phase-05-tests-timezone-safety-accuracy-instrumentation.md) | Partial — tests + backtest done; always-on instrumentation + standalone CLI deferred |

Linear dependency chain: 1 → 2 → 3 → 4; 5 weaves through all (test-first where it reduces risk).

## Constraints (from project memory + user preferences)

- **Advisory only** — never feeds `SyncBalance`/`CreateTopup`/disbursement (mirrors wallet-forecast invariant; prevents wallet-balance-inflation regression).
- **No schema migration unless required** — code-level queries only. If `EXPLAIN` shows a table scan on the new cohort query, add one index (flag it; do not assume). [[feedback_prefer_no_migration]]
- **Timezone safety** — build day bounds in `time.Local` (DSN is `loc=Local`); parse dates with `time.ParseInLocation`. The summary handler already does this correctly (lines 52–57) — follow it verbatim. The UTC-day-bounds bug dropped 65.7M once; do not reintroduce. [[lesson_bulk_create_utc_day_bounds]] [[lesson_timesheet_utc_daybounds_false_24h_reject]]
- **Aggregation = SQL GROUP BY**, never LIMIT-N an aggregation. [[feedback_dashboard_sql_aggregation]]
- **No gold / accent-gold design tokens** in the card; avoid template-default branding. [[feedback_no_gold_design_tokens]]
- **Mobile + desktop sync** — same component/data/title on both surfaces. [[feedback_mobile_desktop_sync]]
- **VND, no decimals** in display formatting.
- **All business time via `clock.Now()`** (Asia/Ho_Chi_Minh).
- **gofmt + golangci-lint clean; `npm run build` clean** before done.

## Acceptance criteria (whole plan)

- [ ] Admin opens `/admin/timesheet` and sees a "Chuẩn bị tiền trả" card showing: confirmed-payable now, projected by next pay date (p50 with [p50–p95] band), wallet balance, and the **gap to prepare**, with a "prepare by <date>" line.
- [ ] For the documented 10-July case, confirmed-payable matches the existing "Chờ thanh toán" figure exactly (same `GetSummaryStats` source) — no divergence.
- [ ] Projected p50 is within the band; band widens when historical basis is thin (`confidence: low`).
- [ ] Advisory invariant holds: removing/commenting the card changes zero balance/disbursement behavior. Verified by a test asserting the service never calls `SyncBalance`/`CreateTopup`.
- [ ] Desktop and mobile cards show identical numbers and title.
- [ ] `go test` (new + existing timesheet/wallet packages) green; `golangci-lint run ./...` clean; frontend `tsc --noEmit` + `npm run build` clean.
- [ ] Accuracy instrumentation logs p50-vs-actual after each pay cycle (feeds the future ML-escalation decision).

## Pay-cycle model (resolved — Session 1)

The timesheet pay cycle is a **global, fixed 4-cycle monthly schedule** (same for all employees/projects):

| Kỳ | Work days | Pay date | Prepare-by (lead 2) |
|----|-----------|----------|---------------------|
| 1 | 1–7 | day **10** | day 8 |
| 2 | 8–14 | day **17** | day 15 |
| 3 | 15–21 | day **24** | day 22 |
| 4 | 22–28 (this month) | day **1** (next month) | day 29 (this month) |

Cohort pools across all employees. The per-project `SalaryPeriodFrom/To` (1–28) is the overall salary window; the 4 sub-cycles apply uniformly on top. This is the advance-payment cycle's sibling — but **distinct** (advance cycle = day 20→9; timesheet cycle = pay days 10/17/24/1). The 462M = July 10 = Kỳ 1 (work Jul 1–7) ✓.

## Validation Log

### Session 1 — 2026-07-10 (critical-questions interview)

**Resolved decisions (user-confirmed):**
1. Pay cycle = **global 4-cycle schedule** (10/17/24/1; work 1–7 / 8–14 / 15–21 / 22–28). Cohort pools across everyone; per-project `SalaryPeriodFrom/To` is the salary window, not a separate cycle.
2. **Lead time = 2 days** → `CASH_FORECAST_LEAD_DAYS` default 2 (prepare-by = pay date − 2).
3. Forecast scope = **timesheet-only** (the 462M case). Advance payments keep their separate wallet-demand forecast; combined total-cash deferred.
4. Card view = **next payment only** (next pay date + gap + prepare-by). All-4-Kỳ month view deferred.

**Verification results (codebase):** Claims checked: 7 · Verified: 6 · Failed: 0 · New findings: 1
- ✅ `cohortSeries` + MC math reusable — package `services`, `wallet_demand_forecast_math.go`.
- ✅ `GetSummaryStats.PendingPaymentAmount` = confirmed-payable; `Timesheet` domain has `Status`/`PaymentStatus`/`ApprovedAt`/`Amount`.
- ✅ `wallet.WalletService.GetBalance`, `WalletForecastConfig` pattern, summary handler `loc=Local` date parse all present.
- 🆕 **NEW finding:** NO existing timesheet pay-cycle helper for 10/17/24/1 — `clock` only has the advance-payment cycle (day 20→9). The 4-cycle anchor is **new code** → Phase 1 adds a small `clock` pay-cycle helper. ⚠️ Phase 1 must first check `listTimesheetsForCycle` for any pre-existing cycle definition before building it.

### Whole-Plan Consistency Sweep
- Replaced the "weekly cycle / confirm the anchor with the user" framing with the known 4-cycle model across `plan.md` + `phase-01` + `phase-02`.
- Cohort basis restated: ~4 cycles/month → a 6-month lookback ≈ 24 cycle observations (strong basis → high confidence likely once ≥3 months of clean history exist).
- Config field reconciled to `HistoryMonths` (default 6), not `HistoryCycles`.
- Zero unresolved contradictions. **Failed: 0 → plan is eligible for implementation.**

## Implementation log (2026-07-11)

Phases 1–4 implemented and verified; Phase 5 tests + backtest delivered. Files: `clock/pay_cycle.go`, `services/cash_readiness_forecast.go`, `domain/cash_readiness.go`, `dto/cash_readiness.go`, `handlers/timesheet/cash_readiness_handler.go`, repo + mock + config + DI wiring + route + casbin, frontend `CashReadinessCard` + hook + service + types mounted on desktop + mobile.

**Verified gates:** `go build ./...` green; `go test` green for pkg/clock, app/services (incl. backtest: 7.3% + 0.0% error forecasting 2 cycles from priors — validates statistical baseline), infra/persistence/repositories, domain; `golangci-lint` clean on new code (3 pre-existing errcheck in `auth/auth_service.go` unrelated; `bootstrap/repositories` TestInitialize fails on nil-DB, pre-existing/environmental); frontend `tsc --noEmit` + `vite build` green.

**Documented deviations (validated against acceptance criteria):**
- **No separate backend 15s cache.** `GetSummaryStats` already caches + invalidates the confirmed-payable (the fast-changing number); the cohort query is a scoped indexed `GROUP BY`. A redundant cache adds stale-cache risk (the exact `stale-cache` lessons) without value. A projection cache is a trivial later add behind `ForecastProvider`. The frontend `useCashReadiness` key sits under `QueryKeys.timesheets.all`, so bulk-approve/transfer already refresh the card.
- **Cohort counts ALL approved timesheets regardless of `payment_status`.** Historical cycles are fully paid — excluding paid would zero the basis. Paid-exclusion belongs only on the confirmed-payable floor, which `GetSummaryStats` owns.
- **`buildTimesheetCohort` lives in `services`; the repo returns daily rows** `{WorkDate, ApprovedDate, Amount}`; cycle-day math is in `clock/pay_cycle.go`. Simpler than the `TimesheetAccrualCycleRow` intermediate originally sketched.
- **`ForecastProvider.ProjectAccrual` takes an explicit `seed int64`** (the plan's signature omitted it) for UI determinism.
- **`CashToPrepare = confirmed + p50`; band `[confirmed+p50, confirmed+p95]`** (matches the validated acceptance "p50 with [p50–p95] band").

**Deferred (need a product decision, flagged for follow-up):**
- **Always-on accuracy instrumentation** (forecast-vs-actual per cycle). Honest accuracy needs a persisted forecast snapshot taken at the prepare-by date, which needs a storage decision (new table vs rotated log) — not done to avoid coupling the advisory forecast to the bulk-transfer money path. The offline backtest (above) provides the same accuracy signal in the meantime.
- **Standalone `qa/scripts/cash-readiness-backtest` CLI.** The backtest *methodology* is implemented + tested; wrapping it as a DB-wired CLI is the remaining plumbing.
- **ML escalation decision hook:** if median abs % error stays > ~12% over ≥4 cycles AND Tết/seasonality swings payout, add a Prophet/LightGBM provider behind `ForecastProvider` (no handler/UI change).

## Risks

- **Coherent cohort is thin.** <3 historical cycles → MC method degrades to `gamma-fit`/`no-history`; band collapses to confirmed-payable only. *Mitigation:* `confidenceLabel` already surfaces this; card shows "low confidence" and the deterministic floor. Honest, not hidden.
- **Double-counting.** Confirmed-payable must exclude already-transferred (paid) timesheets; projection must exclude the confirmed set. *Mitigation:* single source (`GetSummaryStats` status filters) for confirmed; cohort query keys on approval/transfer timestamps.
- **Stale cache hides a freshly-approved batch.** *Mitigation:* short TTL (mirror the 15s timesheet-summary cache) + invalidate on bulk-approve/transfer events (recall the stale-summary-cache investigation, obs 44466–44469).
- **Over-claiming precision.** A precise-looking p50 invites false trust. *Mitigation:* always show the band + confidence label; never show p50 alone.

## Dependencies

None blocking. Touches timesheet, wallet-forecast-math (read-only reuse), config, bootstrap, and frontend timesheet pages — all owned by this repo. No cross-plan overlap found in `plans/`.
