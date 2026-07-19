---
phase: 1
title: "Phase A: Eliminate timesheet-date fetch at root"
status: pending
priority: P1
effort: "M"
dependencies: []
---

# Phase A: Eliminate timesheet-date fetch at root

## Overview

The 5,334-row `SELECT id, date FROM timesheets WHERE id IN (...)` fetch exists only because `domain.CyclePayData` does not persist the cycle's date range. We extend the struct, populate the new fields at every transaction-code creation site, propagate them onto the `BulkTransferFile` at result-processing time, and convert the history view to prefer the persisted range with a lazy per-file fallback for legacy rows.

## Background / why

`PayrollService.GetBankTransferHistories` (`app/services/payroll/service.go:278`) resolves each entry's weekly cycle via `resolveWeeklyHistoryCycle` (`service.go:542`). That function prefers `file.FromDate`/`file.ToDate` but falls through to a timesheet-date lookup when those are nil. Files created by `result_processor.go:storeResultMetadata` (the result-upload path) never set `FromDate`/`ToDate`, so every history request builds the **month-wide union** of all referenced timesheet IDs and fetches their dates — currently 5,334 IDs.

The date range is known at the **export** step (`plan.FromDate`, `plan.ToDate`, `plan.Cycle` are in scope in `export_service.go:172`), but it is discarded when constructing `CyclePayData`. Persisting it there eliminates the downstream fetch entirely.

## Architecture

```
export_service.go / ninepay_service.go / onepay_exporter.go
  plan.FromDate, plan.ToDate, plan.Cycle ──► CyclePayData{FromDate, ToDate, CycleNum}
                                              (stored in TransactionCode.Data JSON)

result_processor.go:storeResultMetadata
  matchedEntry.CyclePayData.FromDate ──► BulkTransferFile{FromDate, ToDate, ForMonth}

service.go:GetBankTransferHistories
  resolveWeeklyHistoryCycle(file, weeklyData, lazyResolver, month)
    ├─ file.FromDate present?  ──► return cycle from file dates  (NEW: 0 queries)
    └─ weeklyData.FromDate present? ──► return cycle from CyclePayData (NEW: 0 queries)
    └─ else ──► lazyResolver(id) per needed ID  (LEGACY fallback: small per-file batch)
```

## Requirements

- **Functional:** every new bulk-transfer transaction code carries its cycle's `FromDate`, `ToDate`, and `CycleNum`. The history view resolves the cycle without a timesheet query when either the file or the transaction code carries the range. Legacy rows (range absent) still resolve correctly via a lazy fetch.
- **Non-functional:** zero timesheet queries on the hot path for new data; bounded per-file fetches for legacy data; no JSON migration required (new fields are `omitempty`).

## Related code files

- Modify: `backend/internal/domain/transaction_code.go` — add 3 fields to `CyclePayData`.
- Modify: `backend/internal/app/services/payroll/bulktransfer/export_service.go` — populate new fields at transaction-code creation.
- Modify: `backend/internal/app/services/payroll/bulktransfer/ninepay_service.go` — populate in `buildTransactionCodes`.
- Modify: `backend/internal/app/services/payroll/bulktransfer/onepay_exporter.go` — populate; thread date range if not in scope.
- Modify: `backend/internal/app/services/payroll/bulktransfer/result_processor.go` — `storeResultMetadata` reads matched entries' range → sets `BulkTransferFile.FromDate/ToDate/ForMonth`.
- Modify: `backend/internal/app/services/payroll/service.go` — `resolveWeeklyHistoryCycle` signature changes to a lazy resolver; `GetBankTransferHistories` builds the resolver only for legacy files.
- Modify: `backend/internal/app/services/payroll/bank_transfer_history_test.go` — exercise new persisted-range path + lazy fallback.

## Implementation steps

1. **Extend `CyclePayData`** (`domain/transaction_code.go`):
   ```go
   // Populated at creation time so the history view can resolve the cycle
   // without fetching timesheet dates. Old rows have zero values; the lazy
   // fallback in resolveWeeklyHistoryCycle handles them.
   FromDate *time.Time `json:"from_date,omitempty"`
   ToDate   *time.Time `json:"to_date,omitempty"`
   CycleNum int        `json:"cycle_num,omitempty"` // 1-4 weekly, 1 monthly
   ```
   Ensure `time` is imported.

2. **Populate at creation sites:**
   - `export_service.go:172-191`: compute pointers from `plan.FromDate`/`plan.ToDate`; `CycleNum` = `clock.KyFromWorkDay(plan.FromDate.Day())` for weekly, `1` for monthly. Set on both `MonthlyPay` and `WeeklyPay` branches.
   - `ninepay_service.go:buildTransactionCodes` (line ~498): same, using the bulk-transfer plan's date range.
   - `onepay_exporter.go:321-347`: if the plan/cycle is not in scope there, thread it through the exporter's caller. Verify before editing.

3. **Propagate to `BulkTransferFile` at result time** (`result_processor.go:610-636`):
   - `findDataByTransactionCodes` already returns matched entries with parsed `CyclePayData`. Pick the first non-nil `FromDate`/`ToDate`/`CycleNum` and set them on the `BulkTransferFile{FromDate, ToDate, ForMonth, Cycle}` (convert `CycleNum` to the `*string` cycle pointer the struct expects, or leave the existing `cyclePtr` logic intact and add the date fields).
   - `ForMonth` = `fromDate.Format("2006-01")`.

4. **Lazy resolver in `service.go`:**
   - Change `resolveWeeklyHistoryCycle` signature from `timesheetDates map[uint]time.Time` to `resolveDate func(uint) (time.Time, bool)`.
   - First check `weeklyData.FromDate`/`ToDate`/`CycleNum` → if present, compute cycle from those without any lookup.
   - Else iterate `weeklyData.TimesheetIDs` calling `resolveDate(id)` until first valid match.
   - In `GetBankTransferHistories`, replace the eager `GetTimesheetDatesByIDs(unionOfAllIDs)` with a per-file lazy path:
     - For each file, if **any** of its entries' `CyclePayData` has `FromDate`, skip the fetch (the resolver will hit the persisted path).
     - Only for legacy files: collect the file's own completed+ref entries' timesheet IDs and build a resolver backed by a single `GetTimesheetDatesByIDs(smallIDSet)`.
   - Keep `GetTimesheetDatesByIDs` in the repo for the legacy path.

5. **Tests** (`bank_transfer_history_test.go`):
   - Update existing test's mock `CyclePayData` to include `FromDate`/`ToDate`/`CycleNum`; assert the resolver does not call `GetTimesheetDatesByIDs` when those are present.
   - Add a second test case with a legacy file (no `FromDate`) that exercises the lazy fallback and asserts the cycle resolves correctly.

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] New unit tests pass (persisted-range path + lazy fallback)
- [ ] Manual log check: `/payrolls/bank-transfer-histories?month=<recent-month-with-new-data>` emits **zero** `SELECT id, date FROM timesheets` queries
- [ ] Legacy month still returns correct cycles (lazy path)
- [ ] No JSON migration required; existing rows deserialize with zero-valued new fields

## Risk assessment

- **Correctness:** the cycle-resolution result must be identical between the file-dates path, the `CyclePayData` path, and the timesheet-fallback path. All three already exist; we are adding a middle path and making the first path hit more often. Risk is low because `resolveWeeklyHistoryCycle` already had the file-dates branch.
- **Data migration:** none required. New JSON fields are `omitempty`; old rows decode fine.
- **`onepay_exporter.go` date availability:** if the exporter lacks date context, threading it could ripple. Mitigation: verify first; if missing, leave that one site unpopulated and rely on the lazy fallback for onepay-sourced files.
- **Mixed cycles in one file:** `findDataByTransactionCodes` already rejects mixed cycles (`result_processor.go` ~line 600), so picking the first matched entry's range is safe.
