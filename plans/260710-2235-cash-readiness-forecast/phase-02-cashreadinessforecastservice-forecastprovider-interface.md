---
phase: 2
title: "CashReadinessForecastService + ForecastProvider interface"
status: pending
priority: P2
dependencies: [1]
---

# Phase 2: CashReadinessForecastService + ForecastProvider interface

## Overview

The forecast brain. Lives in package `services` (sibling to `wallet_demand_forecast.go`) so it reuses the unexported MC primitives directly. Defines a `ForecastProvider` interface so the statistical baseline can be swapped for Prophet/LightGBM later without touching the handler or UI. Composes: confirmed-payable (Phase 1 / `GetSummaryStats`) + projected accrual (provider) − wallet balance = cash gap.

## Requirements

- Functional: return a `CashReadiness` DTO — `{ConfirmedPayable, ProjectedP50, BandLower, BandUpper, WalletAvailable, Gap, PrepareByDate, NextPayDate, LeadDays, Method, Confidence, BasisCycles}`.
- Functional: provider is pluggable via `ForecastProvider`; v1 implementation = `TimesheetAccrualProvider` (statistical).
- Non-functional: advisory-only invariant is *machine-enforced* (the service has no handle to `SyncBalance`/`CreateTopup`/disbursement — only read-only `WalletService.GetBalance`). Deterministic output for a fixed clock+seed (no UI flicker).

## Architecture

```
CashReadinessForecastService (package services)
 ├─ timesheetAnalyticsRepo.GetAccrualCohort(...)   → []TimesheetAccrualCycleRow   (Phase 1)
 ├─ buildTimesheetCohort(rows)                     → []cohortSeries               (new; same shape as pivotCohort)
 ├─ provider.ProjectAccrual(cohortSeries, horizon) → Projection{P50,P95,...}      (interface)
 │     v1 TimesheetAccrualProvider wraps:
 │       forecastDemandDistributionBetween(...) + newsvendorRecommendation(...) + confidenceLabel(...)
 │       (reuse, do NOT reimplement)
 ├─ timesheetService.GetSummaryStats(filters).PendingPaymentAmount  → ConfirmedPayable (deterministic)
 ├─ walletSvc.GetBalance(ctx).Available              → WalletAvailable (read-only)
 └─ clock.Now() + CashForecastConfig.LeadDays        → NextPayDate / PrepareByDate
```

`ForecastProvider` interface (so ML is a drop-in later):
```go
type ForecastProvider interface {
    ProjectAccrual(ctx context.Context, hist []cohortSeries, fromCycleDay, throughCycleDay int, cfg CashForecastConfig) (AccrualProjection, error)
}
type AccrualProjection struct { P50, P95 int64; Method, Confidence string; BasisCycles int }
```
v1 `TimesheetAccrualProvider` delegates to the existing `forecastDemandDistributionBetween` + `newsvendorRecommendation`. A future `ProphetProvider`/`LightGBMProvider` implements the same interface — handler/UI unchanged.

`CashReadiness` math:
- `cash_to_prepare = confirmed_payable + projected_p50`
- `band = [confirmed_payable + 0, confirmed_payable + projected_p95]` (confirmed part is deterministic; band only widens on the accrual term)
- `gap = max(0, cash_to_prepare − wallet_available)`
- `prepare_by_date = next_pay_date − lead_days`

## Related Code Files

- Create: `backend/internal/app/services/cash_readiness_forecast.go` — service + provider interface + `TimesheetAccrualProvider` + `buildTimesheetCohort`.
- Create: `backend/internal/app/services/cash_readiness_forecast_test.go`.
- Create: `backend/internal/domain/timesheet/cash_readiness.go` (or extend an existing timesheet domain file) — `CashReadiness` + `AccrualProjection` DTOs.
- Read-only reuse: `backend/internal/app/services/wallet_demand_forecast_math.go` (`forecastDemandDistributionBetween`, `newsvendorRecommendation`, `confidenceLabel`, `forecastSeed`, `cohortSeries`), `backend/internal/app/services/wallet_demand_forecast.go` (`cohortSeries` shape, `cumulativeAt`).
- Modify: `backend/internal/config/config.go` — add `CashForecastConfig` (env `CASH_FORECAST_*`) mirroring `WalletForecastConfig` (lines 187–206), populate at ~line 419.

<!-- Updated: Validation Session 1 - HistoryMonths not HistoryCycles; uses new pay_cycle clock helper -->

`CashForecastConfig` fields (all optional, mirror wallet forecast):
`ServiceLevel` (default 0.95), `NSim` (5000), `HistoryMonths` (default 6 → ≈24 cycle observations across Kỳ 1–4), `LeadDays` (default 2, resolved Session 1), `UncertaintyFactor` (0). The service uses the Phase-1 `clock/pay_cycle.go` helper (`CurrentKy`, `NextPayDate`, `CycleDay`) — not the advance-payment `clock` functions, which model a different cycle.

## Implementation Steps

1. Add `CashForecastConfig` to `config.go` with env loading + defaults; log resolved values at boot.
2. Define `ForecastProvider` interface + `AccrualProjection` in the new service file.
3. Implement `buildTimesheetCohort(rows) []cohortSeries` — same internal shape as `pivotCohort` output, so the existing math consumes it unchanged. Pay attention to `cumulativeAt`, `grandTotal`, `maxCycleDay`, `completedTotal` fields.
4. Implement `TimesheetAccrualProvider.ProjectAccrual`: call `forecastDemandDistributionBetween(hist, fromCycleDay, throughCycleDay, cfg.NSim, seed, paidFrac=1)` then `newsvendorRecommendation` + `confidenceLabel`. Seed via a `forecastSeed`-equivalent for the timesheet cycle.
5. Implement `CashReadinessForecastService.GetCashReadiness(ctx, filters, horizon)`: assemble `CashReadiness` per the math above. Read wallet balance defensively (on error, `WalletAvailable=0` + `Confidence` note; never block the card on a wallet read failure).
6. Enforce advisory invariant structurally: constructor takes only read-only ports (`TimesheetAnalyticsReader`, `TimesheetSummaryReader`, `wallet.WalletService` for `GetBalance`, `clock.Clock`, `CashForecastConfig`). No disbursement/top-up/sync handle.
7. Tests: confirmed-sum exactness; band monotonic (`Lower ≤ ProjectedP50 ≤ Upper`); no-history fallback (`method=no-history` → band collapses to confirmed-only, `confidence=low`); gap math incl. `wallet ≥ cash_to_prepare` → gap 0; deterministic for fixed clock+seed; **invariant spy test** asserting `SyncBalance`/`CreateTopup` are never invoked.

## Success Criteria

- [ ] `GetCashReadiness` returns a fully populated `CashReadiness` for the admin scope.
- [ ] Confirmed-payable equals `GetSummaryStats(...).PendingPaymentAmount` exactly (asserted in test).
- [ ] Band monotonic; no-history → confirmed-only floor with `confidence=low`.
- [ ] Same `(clock, seed)` → byte-identical projection (no UI flicker).
- [ ] Invariant test proves no path to `SyncBalance`/`CreateTopup` (spy asserts zero calls).
- [ ] `ForecastProvider` is an interface with `TimesheetAccrualProvider` as v1; swapping providers needs no service/handler/UI change.
- [ ] `go test ./internal/app/services/...` green; `golangci-lint` clean.

## Risk Assessment

- **`cohortSeries` shape mismatch** → math misreads the cohort. *Mitigation:* unit test `buildTimesheetCohort` against a hand-computed curve before wiring the provider.
- **Seed/time determinism broken** → flicker / non-reproducible bands. *Mitigation:* derive seed from cycle+cycle-day exactly like `forecastSeed`; clock injected, never `time.Now()` directly.
- **Wallet read failure degrades UX** → card blanks out. *Mitigation:* defensive read; show confirmed-payable + projection regardless, flag wallet figure as unavailable.
