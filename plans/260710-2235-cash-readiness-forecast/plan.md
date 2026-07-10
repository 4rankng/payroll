---
title: "Cash-readiness forecast for the next timesheet payment (/admin/timesheet)"
description: "Gives the admin a purpose-built 'how much cash to prepare for the next weekly bulk transfer' metric on /admin/timesheet. Bottom-up: confirmed-payable (already-approved+unpaid timesheets, the deterministic ~70-90% of the total) plus a short-horizon projected-accrual band, minus wallet balance = cash gap to prepare. Reuses the existing newsvendor Monte-Carlo engine on a NEW timesheet-accrual cohort (the wallet forecast's cohort is advance-payment-only and is the wrong data source for this problem). Advisory-only; never feeds SyncBalance. Statistical baseline chosen over ML (Prophet/LightGBM) — see Decision."
status: pending
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
| 1 | [Timesheet cash-flow cohort + confirmed-payable query](./phase-01-timesheet-cash-flow-cohort-confirmed-payable-query.md) | Pending |
| 2 | [CashReadinessForecastService + ForecastProvider interface](./phase-02-cashreadinessforecastservice-forecastprovider-interface.md) | Pending |
| 3 | [HTTP endpoint + DI wiring + caching](./phase-03-http-endpoint-di-wiring-caching.md) | Pending |
| 4 | [Admin timesheet cash-readiness card (desktop + mobile sync)](./phase-04-admin-timesheet-cash-readiness-card-desktop-mobile-sync.md) | Pending |
| 5 | [Tests + timezone safety + accuracy instrumentation](./phase-05-tests-timezone-safety-accuracy-instrumentation.md) | Pending |

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

## Open questions (resolve before/during Phase 1)

1. **Next-pay-date anchor.** Is there a fixed weekly bulk-transfer day (e.g. every Friday / Monday), or is it ad-hoc? This sets the projection horizon. If ad-hoc, default horizon = `CASH_FORECAST_LEAD_DAYS` (default 2) and let the card show a rolling horizon.
2. **Lead-time default.** How many days does the admin actually need to move cash into the wallet? Default 2 (mirrors wallet forecast); confirm with user.
3. **Scope: timesheet-only vs total-cash.** v1 = timesheet weekly bulk transfer (the 462M case). Should the card also fold in pending advance-payment disbursements for one combined number? Recommendation: **v1 timesheet-only** (advance payments already have their own wallet-demand forecast); total-cash = optional Phase 6.

## Risks

- **Coherent cohort is thin.** <3 historical cycles → MC method degrades to `gamma-fit`/`no-history`; band collapses to confirmed-payable only. *Mitigation:* `confidenceLabel` already surfaces this; card shows "low confidence" and the deterministic floor. Honest, not hidden.
- **Double-counting.** Confirmed-payable must exclude already-transferred (paid) timesheets; projection must exclude the confirmed set. *Mitigation:* single source (`GetSummaryStats` status filters) for confirmed; cohort query keys on approval/transfer timestamps.
- **Stale cache hides a freshly-approved batch.** *Mitigation:* short TTL (mirror the 15s timesheet-summary cache) + invalidate on bulk-approve/transfer events (recall the stale-summary-cache investigation, obs 44466–44469).
- **Over-claiming precision.** A precise-looking p50 invites false trust. *Mitigation:* always show the band + confidence label; never show p50 alone.

## Dependencies

None blocking. Touches timesheet, wallet-forecast-math (read-only reuse), config, bootstrap, and frontend timesheet pages — all owned by this repo. No cross-plan overlap found in `plans/`.
