# Phase 2: Forecast model and additive API

**Status:** Completed

## Files

- `backend/internal/domain/timesheet_types.go`
- `backend/internal/infra/persistence/repositories/timesheet_analytics_repository.go`
- `backend/internal/app/services/cash_readiness_forecast.go`
- forecast math/provider tests
- cash-readiness domain, DTO, and handler mapping
- config loading/validation

## Implementation

1. Read point-in-time forecast rows with work date, created time, approved time,
   final status, amount, employee, and project. Keep role/project scoping in the
   existing query builder.
2. Decompose the target Ky into:
   - approved by the forecast timestamp (deterministic floor),
   - created but not yet approved rows (pending exposure),
   - rows expected to be created after the forecast timestamp (future/missing).
3. Build 24-52 recent cycle observations across all Ky. Normalize future-created
   amounts by observed employee count and use project-level estimates only when
   their basis is sufficient; otherwise shrink to the company estimate.
4. Estimate pending conversion with a beta-smoothed historical approval rate.
   Simulate pending conversion plus bootstrapped future-created amounts without
   assuming a gamma distribution.
5. When enough resolved snapshots exist for the same days-to-pay bucket, add
   empirical residual calibration and compute measured WAPE, bias, interval
   coverage, and reserve shortfall rate.
6. Return additive fields: pending amount, expected pending, expected future,
   expected payout, recommended reserve, central interval, model version,
   calibration sample count, accuracy metrics, and reliability state.
7. Preserve old fields by mapping them consistently to the upgraded result.
8. Persist only unfiltered company-wide snapshots. Snapshot write failures are
   logged and never blank the advisory response.

## Reliability gate

Use `uncalibrated` below 12 resolved same-horizon snapshots. At 12-23 samples,
report `learning`. At 24+ samples, report measured reliability from configurable
thresholds, initially WAPE <= 10%, absolute bias <= 3%, reserve shortfall <= 10%,
and central interval coverage between 85% and 95%.

## Risk

Historical mutable statuses cannot perfectly reconstruct every old as-of state.
Use `created_at` and `approved_at` for the best leakage-free backfill, while
treating persisted forward snapshots as the authoritative calibration source.
