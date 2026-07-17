---
phase: 1
title: 'Refactor: Extract Reusable Export Planner'
status: completed
priority: P1
effort: M
dependencies: []
---

# Phase 1: Refactor: Extract Reusable Export Planner

## Overview

Split `ExportService.Export(ctx, req)` into two methods on a new `ExportPlanner` receiver:

- **`Plan(ctx, req) (*ExportPlan, error)`** — pure: selects eligible timesheets, unions forced items, aggregates by employee×project, runs `ValidateAndFilterBulkTransferData`, computes `snapshot_epoch`. No writes. No Excel. No code generation.
- **`Persist(ctx, req, plan) (filename, error)`** — writes **only what today's `saveBulkTransferFile` actually writes**: generates a filename and calls `transactionCodeRepo.CreateBatch`. Emits the audit event via `publishExportAudit` (also moved here).

`ExportService.Export()` then becomes a thin orchestrator: `plan, err := planner.Plan(ctx, req); ...; filename, err := planner.Persist(ctx, req, plan); ...; excel := excelService.Generate(...)`. **Production behavior is byte-for-byte unchanged** — verified by the existing `flow_manual_bulk_transfer.go` integration test.

### Red-team correction (Finding 1)

The original Phase 1 claimed `saveBulkTransferFile` writes a `bulk_transfer_files` row. **It does not.** Verified at `backend/internal/app/services/payroll/bulktransfer/export_service.go`: zero `fileRepo.` calls in the file. The `bulk_transfer_files` row is written elsewhere (`audit_service.go:109`, `ninepay_service.go:245`, `result_processor.go:638`) — those paths are OUT OF SCOPE for this refactor. Phase 1's write surface is therefore **smaller than originally stated**: just `transactionCodeRepo.CreateBatch` + the audit event publish. This makes the refactor lower-risk than first drafted.

### Red-team correction (Finding 8) — no-date-filter mode

Phase 2's full-pool scan needs `Plan()` to run with **no date filter** (select all outstanding approved+pending/failed timesheets regardless of cycle window). Today `Plan()` always resolves a date range via `ResolveWeeklyRange`/`ResolveMonthlyRange`, both of which reject empty inputs (`period_calculator.go:31-34`). Phase 1 therefore adds a third entry mode to `Plan`:

```go
// ExportBulkTransferRequest gains an internal-only field:
type ExportBulkTransferRequest struct {
    // ... existing fields unchanged ...
    // noDateFilter is set ONLY by the simulation's full-pool scan.
    // Production export never sets it. When true, Plan() skips date resolution
    // and the timesheet query omits FromDate/ToDate (still applies status +
    // project + employee filters).
    noDateFilter bool `json:"-"`
}
```

`Plan()` checks `req.noDateFilter`: if true, skip `ResolveWeeklyRange`/`ResolveMonthlyRange`, leave `filters.FromDate`/`filters.ToDate` nil. Everything downstream (forced-payroll union, aggregation, validation, snapshot) runs unchanged. This keeps the "single selection algorithm" invariant: the full-pool scan is the **same** Plan() code with one fewer filter, not a parallel implementation.

## Requirements

- **Functional:** Identical input/output behavior for `POST /api/v1/payrolls/export-bulk-transfer` before and after refactor.
- **Non-functional:** No new public API surface in Phase 1 (the `ExportPlanner` is package-internal). No performance regression — same query count.

## Architecture

### Current (monolithic)

```
ExportService.Export(ctx, req)
  ├─ resolve date range (weekly/monthly)
  ├─ listTimesheetsForCycle (pending+failed, approved)
  ├─ filterTimesheetsByRequest (employee IDs)
  ├─ union forced-payroll timesheets
  ├─ aggregateTimesheetData (by employee×project, gen txn codes)
  ├─ excelService.ValidateAndFilterBulkTransferData
  ├─ saveBulkTransferFile  ← WRITE: file row + transaction_codes batch
  ├─ excelService.GenerateBulkTransferExcel  ← generates ZIP
  └─ publishExportAudit  ← WRITE: event
```

### After refactor

```
// New file: planner.go (same package bulktransfer)

type ExportPlan struct {
    Cycle              string                       // "weekly" | "monthly"
    FromDate, ToDate   time.Time
    MonthStart         time.Time                    // zero for weekly
    ValidatedData      *excel.BulkTransferValidationResult  // valid + skipped
    RawAggregated      *excel.BulkTransferData              // pre-validation
    SnapshotEpoch      time.Time                    // max(updated_at) across timesheets
    PeriodCache        map[uint]projectPeriod
}

type ExportPlanner struct { /* same deps as ExportService minus excel gen + file repo writes */ }

func (p *ExportPlanner) Plan(ctx, req) (*ExportPlan, error)
//   ↑ all read logic moved here. Returns snapshot_epoch.

func (p *ExportPlanner) Persist(ctx, req, plan) (filename string, audit *auditSummary, err error)
//   ↑ saveBulkTransferFile + publishExportAudit moved here.
//     Returns audit summary so Export() can still publish if it wants, OR
//     publishes itself (preferred — keeps Export() the single owner of the event).

// export_service.go (rewritten)
func (es *ExportService) Export(ctx, req) (*dto.ExportBulkTransferResponse, error) {
    plan, err := es.planner.Plan(ctx, req)
    if err != nil { return nil, err }

    // Phase 3 adds: optional stale-snapshot enforcement here.
    // if req.IfMatchSnapshot != nil && plan.SnapshotEpoch.After(*req.IfMatchSnapshot) {
    //     return nil, domain.ErrStaleSimulation
    // }

    filename, _, err := es.planner.Persist(ctx, req, plan)
    if err != nil { return nil, fmt.Errorf("failed to save bulk transfer file: %w", err) }

    response, err := es.excelService.GenerateBulkTransferExcel(ctx, plan.RawAggregated, req,
        plan.FromDate.Format(timeutil.DateFormat),
        plan.ToDate.Format(timeutil.DateFormat),
        plan.Cycle)
    // ... fill response fields ...
    return response, nil
}
```

### `SnapshotEpoch` computation (red-team Finding 8 — single SQL, no preload)

Red-team verified the `excel.Employee` struct (`excel/service.go`) does NOT carry `UpdatedAt`, so iterating preloaded structs cannot produce a complete epoch. Phase 1 computes it as **one explicit SQL query** after the read step:

```sql
SELECT GREATEST(
  COALESCE((SELECT MAX(updated_at) FROM timesheets        WHERE id IN (?)),           '1970-01-01'),
  COALESCE((SELECT MAX(updated_at) FROM employees         WHERE id IN (?)),           '1970-01-01'),
  COALESCE((SELECT MAX(updated_at) FROM project_employees WHERE employee_id IN (?) AND project_id IN (?)), '1970-01-01')
)
```

This is the value the simulation returns and the real export (Phase 3) optionally checks. It captures the three mutation surfaces that actually change selection: timesheet status, employee bank info, and project-employee assignment (which controls the `PaymentSchedule` filter in `aggregateTimesheetData`). One round-trip, deterministic, covers the real drift sources.

## Related Code Files

- **Modify:** `backend/internal/app/services/payroll/bulktransfer/export_service.go` — move read logic into `planner.go`, keep `Export()` as orchestrator.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/planner.go` — `ExportPlanner`, `Plan`, `Persist`, `ExportPlan` type.
- **Read-only refs:** `backend/internal/app/services/payroll/excel/service.go` (`ValidateAndFilterBulkTransferData`, `GenerateBulkTransferExcel`), `backend/internal/app/services/payroll/bulktransfer/period_calculator.go` (`ResolveWeeklyRange`, `ResolveMonthlyRange`, `GetProjectPeriod`), `backend/internal/app/services/payroll/bulktransfer/constants.go`.
- **Regression guard:** `backend/tests/integration/flow_manual_bulk_transfer.go` must pass unchanged.

## Implementation Steps

1. **Create `planner.go`.** Define `ExportPlan`, `ExportPlanner`, `projectPeriod` (move the latter from `export_service.go` if currently private there). Construct via `NewExportPlanner(...)` taking the same repos `ExportService` currently holds for reads: `timesheetRepo`, `employeeRepo`, `projectRepo`, `projectEmployeeRepo`, `transactionCodeRepo`, `periodCalculator`. (Note: `fileRepo`, `excelService`, `eventBus` stay on `ExportService` — they're write/gen deps.)
2. **Move read methods verbatim** from `export_service.go` to `planner.go`: `listTimesheetsForCycle`, `filterTimesheetsByRequest`, `aggregateTimesheetData`. Keep them as `ExportPlanner` methods. Do **not** change a single line of logic — pure cut-paste.
3. **Implement `Plan()`.** Body is the first half of today's `Export()` (lines ~64–140): date resolution, list, filter, forced-union, aggregate, `ValidateAndFilterBulkTransferData`, then compute `SnapshotEpoch` and assemble `*ExportPlan`.
4. **Implement `Persist()`.** Body is today's `saveBulkTransferFile` (rename to `Persist` or wrap it) **plus** `publishExportAudit`. Return `(filename, auditSummary, error)`. No behavior change.
5. **Rewrite `ExportService.Export()`** as the orchestrator shown in Architecture. Add a `planner *ExportPlanner` field to `ExportService`, wired in `NewExportService`. Keep `exportService.checksumCalculator` even if unused post-refactor (don't churn — flag with a comment; remove in a later cleanup).
6. **Construct the planner in `NewExportService`** from the same deps. No caller of `NewExportService` changes signature.
7. **Run `make api-test`** and specifically `flow_manual_bulk_transfer` — assert identical behavior.

## Success Criteria

- [ ] `planner.go` exists with `ExportPlanner`, `Plan`, `Persist`, `ExportPlan`.
- [ ] `ExportService.Export()` is a thin orchestrator; no read-selection logic remains in it.
- [ ] `POST /api/v1/payrolls/export-bulk-transfer` integration test (`flow_manual_bulk_transfer.go`) passes unchanged.
- [ ] `ExportPlan.SnapshotEpoch` is populated (non-zero when ≥1 timesheet is eligible).
- [ ] No new public API or DTO change in Phase 1.
- [ ] `make api-test` green; `cd backend && go test ./... -race` green.

## Risk Assessment

**Risk: behavior drift during cut-paste.**
Mitigation: pure move, no edits. Run the full integration suite immediately. Add the parity test in Phase 5 as a permanent regression guard.

**Risk: missed a write side effect buried in a "read" method.**
Mitigation: `Persist()` is the only method allowed to call `fileRepo.*`, `transactionCodeRepo.Create*`, `eventBus.Publish`. Grep after refactor to confirm zero write calls remain in `Plan`/its helpers.

**Risk: `SnapshotEpoch` correctness.**
Mitigation: prefer a single explicit `SELECT MAX(updated_at) FROM timesheets WHERE id IN (?)` over iterating preloaded structs if preload reliability is in doubt. Phase 5 stale-snapshot test validates the epoch actually moves when data changes.

**Risk: `transactionCodeRepo` writes happen inside today's `saveBulkTransferFile` — keep that in `Persist`, not `Plan`.**
Mitigation: `Plan` must not allocate transaction codes at all. The simulation doesn't need them (it generates display-only placeholder codes in Phase 2). Verify `Plan()` never calls `transactionCodeRepo.CreateBatch`.
