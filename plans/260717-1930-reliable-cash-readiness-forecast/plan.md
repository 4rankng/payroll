---
title: "Reliable cash-readiness forecast"
status: completed
created: 2026-07-17
---

# Reliable cash-readiness forecast

## Outcome

Upgrade `/api/v1/timesheets/cash-readiness` and the shared admin timesheet card
from an unmeasured same-Ky estimate to an advisory forecast that:

- separates deterministic approved pay, pending-approval exposure, and future
  or not-yet-created timesheets;
- recommends a reserve quantile separately from the expected payout;
- persists company-wide point-in-time snapshots and resolves them against the
  weekly bulk-transfer export outcome;
- reports confidence from real rolling accuracy when enough resolved snapshots
  exist, falling back to an explicit uncalibrated state otherwise.

Existing response fields remain available. New response fields are additive.

## Scope

### Included

1. Migration and repository for idempotent forecast snapshots/outcomes.
2. Rich target-Ky forecast rows including creation, approval, employee, project,
   status, and amount data.
3. A transparent statistical provider using pooled recent cycles, pending
   conversion, headcount normalization, non-parametric simulation, and resolved
   residual calibration.
4. Company-wide snapshot writes on unfiltered forecast reads and outcome
   resolution from weekly bulk-transfer export events.
5. Additive API fields and shared desktop/mobile UI for expected payout,
   recommended reserve, drivers, and measured/uncalibrated reliability.
6. Backend unit, repository/event-handler, API mapping, frontend lint/type, and
   integration-flow coverage.

### Excluded

- Prophet, LightGBM, neural-network, or external forecasting services.
- Automated wallet top-up, balance synchronization, or payment execution.
- Production deployment or retroactive mutation of payroll data.
- Accuracy scoring for partner/project/employee-filtered forecasts; those remain
  available but only the unfiltered company-wide forecast is reconciled to the
  company-wide transfer outcome.

## Constraints

- Go/GORM/MySQL backend; React/TypeScript frontend.
- All business time uses `clock.Now()` and Asia/Ho_Chi_Minh cycle helpers.
- Monetary values stay integer VND.
- Forecast remains advisory-only through read-only ports.
- Migration is limited to durable forecast measurement, which cannot be made
  reliable with mutable operational rows or process logs alone.
- Work directly on `main`; no deployment or commit in this plan.

## Phases

1. [Measurement foundation](./phase-01-measurement-foundation.md)
2. [Forecast model and API](./phase-02-forecast-model-and-api.md)
3. [UI, verification, and rollout gate](./phase-03-ui-verification-rollout.md)

## Acceptance criteria

- Repeated reads in the same company-wide cycle day upsert one snapshot rather
  than creating duplicates.
- Repeated weekly export events cannot double-count the actual; the outcome
  update is idempotent and scoped to the exact work-date range.
- Expected payout is never below observed approved value.
- Recommended reserve is never below expected payout or observed approved value.
- Pending target-Ky rows affect the forecast without importing the global
  outstanding-payment backlog.
- Historical basis includes comparable observations across all four Ky while
  retaining Ky/project/headcount controls; it is not a blind absolute-amount
  pool.
- Confidence is `uncalibrated` until sufficient resolved company snapshots
  exist, then derives from horizon-specific WAPE, bias, reserve shortfall, and
  interval coverage.
- Existing JSON fields and desktop/mobile consumers remain compatible.
- No service path can call `SyncBalance`, `CreateTopup`, or a disbursement port.
- Focused backend tests, backend package tests, frontend type-check/lint, and the
  repository-required regression suite pass.

## Rollback

The API additions and UI can be reverted independently. Snapshot writes are
advisory and can be disabled without affecting payroll. The new table can remain
unused safely or be dropped with its down migration.

## Completion

Completed 2026-07-17. Migrations 090 and 091 were applied and schema-verified on
local, production, and demo databases. Focused backend tests, race checks,
frontend display tests, lint/type checking, and the production frontend build
pass. The repository-wide live API flow still requires an authenticated local
session and was not weakened or bypassed.
