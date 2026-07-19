---
phase: 1
title: "Phase A: Eliminate timesheet-date fetch at root"
status: pending
priority: P1
effort: "L"
dependencies: []
---

# Phase A: Eliminate timesheet-date fetch at root

## Overview

The 5,334-row `SELECT id, date FROM timesheets WHERE id IN (...)` fetch exists only because `domain.CyclePayData` does not persist the cycle's date range. We extend the struct, populate the new fields at every transaction-code creation site, and convert `resolveWeeklyHistoryCycle` to resolve **per-entry** from `weeklyData.FromDate` (skipping the buggy `fixedWeeklyCycle` validation), with a **month-level union** fallback for legacy rows.

> **Red-team revision (findings F1, F4, F5, F9):** the original plan tried to denormalize dates onto `BulkTransferFile` at result-processing time. That path does not exist — `storeResultMetadata` receives `[]dto.BulkTransferFileData` (no date fields), and file-level dates are wrong when a result file mixes Ky1+Ky2 entries (F4). Resolved cycles **per-entry from `weeklyData`** instead. Legacy fallback reverted to **month-level union** (today's behavior) rather than N+1 per-file fetches (F5).

## Background / why

`PayrollService.GetBankTransferHistories` (`app/services/payroll/service.go:278`) resolves each entry's weekly cycle via `resolveWeeklyHistoryCycle` (`service.go:542`). That function prefers `file.FromDate`/`file.ToDate` but falls through to a timesheet-date lookup when those are nil. Files created by `result_processor.go:storeResultMetadata` never set `FromDate`/`ToDate`, so every history request builds the **month-wide union** of all referenced timesheet IDs and fetches their dates — currently 5,334 IDs.

The date range is known at the **export** step (`plan.FromDate`, `plan.ToDate`, `plan.Cycle` are in scope in `export_service.go:172`), but discarded when constructing `CyclePayData`. Persisting it there lets the resolver short-circuit per-entry, **without** any file-level denormalization.

### Why per-entry, not per-file (red-team F1, F4)

- `storeResultMetadata` (`result_processor.go:612`) receives `[]dto.BulkTransferFileData`, which has no date fields. Propagating dates to the file level would require extending 3 DTOs (`CyclePayData` → `BulkTransferFileDataEntry` → `dto.BulkTransferFileData`) across the repo boundary — ~4× the surface area, for no benefit.
- A single result-upload file can legitimately contain Ky1 and Ky2 codes (both `GetCycle()=="weekly"`; `findDataByTransactionCodes` only rejects mixed *types*, not *numbers*). File-level `FromDate` would mislabel the whole file as one cycle. Per-entry resolution from `weeklyData.FromDate` is correct because each entry's `CyclePayData` carries its own range.

### Why month-level union for legacy (red-team F5)

- The original "per-file lazy fetch" plan was N+1 by file count: a legacy month with F files fires F queries. For a partner with 24 re-imported files, that's a 24× regression. Reverted to today's behavior: one batched union call for all legacy files.

## Architecture

```
export_service.go / ninepay_service.go / onepay_exporter.go
  plan.FromDate, plan.ToDate, plan.Cycle ──► CyclePayData{FromDate, ToDate, CycleNum}
                                              (stored in TransactionCode.Data JSON)

service.go:GetBankTransferHistories
  for each file:
    parse codeData map (CyclePayData per transaction code)   ← already happens today
  if ALL entries across ALL files have weeklyData.FromDate:
      skip the timesheet fetch entirely  (NEW: 0 queries)
  else:
      union of ALL timesheet IDs across the month's files   ← today's behavior
      GetTimesheetDatesByIDs(unionIDs)                      ← batched, bounded by month

  resolveWeeklyHistoryCycle(file, weeklyData, resolveDate, month)
    ├─ weeklyData.FromDate present?
    │     cycle := KyFromWorkDay(weeklyData.FromDate.Day())   ← skip fixedWeeklyCycle (F2)
    │     fromDate := WorkStartDay(cycle) / +6d               ← canonical reconstruction
    │     return                                                (NEW: 0 queries)
    └─ else: resolveDate(id) per needed ID                     (LEGACY fallback)
```

## Requirements

- **Functional:** every new bulk-transfer transaction code carries its cycle's `FromDate`, `ToDate`, and `CycleNum`. The history view resolves the cycle without a timesheet query when every entry in the month carries the range. Legacy rows still resolve correctly via the month-level union fetch (today's behavior, unchanged).
- **Non-functional:** zero timesheet queries on the hot path for new data; identical-to-today single batched union query for legacy data; no JSON migration required.

## Related code files

- Modify: `backend/internal/domain/transaction_code.go` — add 3 fields to `CyclePayData`.
- Modify: `backend/internal/app/services/payroll/bulktransfer/export_service.go` — populate new fields at transaction-code creation (`Persist`, line 172).
- Modify: `backend/internal/app/services/payroll/bulktransfer/ninepay_service.go` — change `buildTransactionCodes` signature (line 498) to accept `fromDate, toDate time.Time, cycleNum int`; update caller (line 250); also set `FromDate/ToDate/ForMonth` on the `btf` literal (line 234).
- Modify: `backend/internal/app/services/payroll/bulktransfer/onepay_exporter.go` — thread `plan.FromDate, plan.ToDate` from `Export` (line 131) → `buildRowsWithSwift` (line 202) → `buildTransactionCode` (line 321).
- **NOT modified:** `result_processor.go:storeResultMetadata` — per-entry resolution means no file-level date propagation needed (red-team F1, F9).
- Modify: `backend/internal/app/services/payroll/service.go` — `resolveWeeklyHistoryCycle` adds a `weeklyData.FromDate` branch using `KyFromWorkDay` directly (skips `fixedWeeklyCycle`); `GetBankTransferHistories` skips the union fetch when all entries have `weeklyData.FromDate`.
- Modify: `backend/internal/app/services/payroll/bank_transfer_history_test.go` — exercise new persisted-range path (incl. off-boundary dates), legacy fallback, and per-entry resolution.

## Implementation steps

1. **Extend `CyclePayData`** (`domain/transaction_code.go`):
   ```go
   // Populated at creation time so the history view can resolve the cycle
   // without fetching timesheet dates. Old rows have zero values; the
   // month-level union fallback in GetBankTransferHistories handles them.
   FromDate *time.Time `json:"from_date,omitempty"`
   ToDate   *time.Time `json:"to_date,omitempty"`
   CycleNum int        `json:"cycle_num,omitempty"` // 1-4 weekly, 1 monthly
   ```
   Ensure `time` is imported. Fields MUST be pointers with `omitempty` so legacy rows decode to nil (red-team F-MEDIUM-1).

2. **Populate at creation sites:**
   - `export_service.go:172-191` (`Persist`): `plan.FromDate`/`plan.ToDate` in scope. `CycleNum` = `clock.KyFromWorkDay(plan.FromDate.Day())` for weekly, `1` for monthly. Set pointers (`&plan.FromDate`, `&plan.ToDate`) on both `MonthlyPay` and `WeeklyPay` branches.
   - `ninepay_service.go`: change `buildTransactionCodes(data, cycle string, fileID uint)` → `buildTransactionCodes(data, cycle string, fileID uint, fromDate, toDate time.Time, cycleNum int)`. Update caller (line 250). **Also** add `FromDate/ToDate/ForMonth` to the `btf` literal at line 234 (red-team H2 — without this, ninepay files still hit the lazy path).
   - `onepay_exporter.go`: thread `plan.FromDate, plan.ToDate` from `Export` (line 131, where `plan` is in scope) into `buildRowsWithSwift` (line 202) and onward to `buildTransactionCode` (line 321). Concrete 2-level thread, not a hedge (red-team M1).

3. **Per-entry resolution in `resolveWeeklyHistoryCycle`** (`service.go:542`):
   - Add a new branch BEFORE the existing `file.FromDate` check:
     ```go
     if weeklyData != nil && weeklyData.FromDate != nil {
         cycle := clock.KyFromWorkDay(weeklyData.FromDate.Day())
         if cycle >= 1 && cycle <= 4 &&
            weeklyData.FromDate.Year() == workMonth.Year() &&
            weeklyData.FromDate.Month() == workMonth.Month() {
             fromDate := time.Date(workMonth.Year(), workMonth.Month(),
                 clock.WorkStartDay(cycle), 0, 0, 0, 0, clock.DefaultLocation)
             return cycle, fromDate, fromDate.AddDate(0, 0, 6), true
         }
     }
     ```
   - **Do NOT call `fixedWeeklyCycle` on the persisted range** (red-team F2: admins export arbitrary date ranges like `2026-07-03 → 2026-07-09`; `fixedWeeklyCycle` rejects day-3 starts). Derive the cycle via `KyFromWorkDay` directly and reconstruct canonical `fromDate`/`toDate` from `WorkStartDay(cycle)`.
   - Keep the existing `file.FromDate`/`fixedWeeklyCycle` branch and the lazy fallback for legacy rows — unchanged.
   - Keep the existing signature `timesheetDates map[uint]time.Time` (or the lazy resolver variant from the prior session). No need to change it again.

4. **Month-level union for legacy** (`GetBankTransferHistories`, `service.go:344-368`):
   - After parsing `codeData`, scan: does **every** `codeData[code].WeeklyPay` across all files have `FromDate != nil`?
   - **If yes:** skip `GetTimesheetDatesByIDs` entirely. Build an empty `timesheetDates` map (the resolver will never consult it).
   - **If no (any legacy entry):** build the month-wide union of timesheet IDs (today's behavior) and call `GetTimesheetDatesByIDs(unionIDs)` once. This preserves today's query count for legacy data — no N+1 (red-team F5).
   - Keep `GetTimesheetDatesByIDs` in the repo.

5. **Tests** (`bank_transfer_history_test.go`):
   - Update existing test: set `FromDate`/`ToDate`/`CycleNum` on the mock `CyclePayData`; assert `GetTimesheetDatesByIDs` is **not** called when all entries have `FromDate`.
   - **New test — off-boundary dates (red-team F2):** `CyclePayData{FromDate: 2026-07-03, ToDate: 2026-07-09}` (day 3, not a canonical boundary). Assert cycle resolves to the correct Ky via `KyFromWorkDay(3)=1`, NOT via `fixedWeeklyCycle` (which would reject).
   - **New test — mixed Ky in one file (red-team F4):** two entries in the same file, one with `FromDate=2026-07-01` (Ky1), one with `FromDate=2026-07-15` (Ky3). Assert each resolves to its own cycle (not the file's first entry's cycle).
   - **New test — legacy fallback:** entries without `FromDate` → union fetch fires once, cycle resolves via timesheet dates.
   - **New test — zero-value `FromDate` (red-team M1):** `FromDate: ptr(time.Time{})` (Go zero). Assert falls through to legacy path (the `Year()==workMonth.Year()` guard rejects year-1).

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] New unit tests pass (persisted-range path, off-boundary dates, mixed-Ky, legacy fallback, zero-value guard)
- [ ] Manual log check: `/payrolls/bank-transfer-histories?month=<recent-month-with-ONLY-new-data>` emits **zero** `SELECT id, date FROM timesheets` queries
- [ ] Manual log check: legacy month emits exactly **one** batched `SELECT id, date` query (today's behavior — no N+1 regression)
- [ ] No JSON migration required; existing rows deserialize with nil new fields

## Risk assessment

- **Correctness (red-team F2 — accepted):** deriving cycle via `KyFromWorkDay(weeklyData.FromDate.Day())` rather than `fixedWeeklyCycle` is a deliberate semantics choice. `KyFromWorkDay` returns cycle 4 for any day ≥ 22 (including 29-31, which `fixedWeeklyCycle` rejects). This is **correct** for the history view — an admin who exported `2026-07-23 → 2026-07-29` intends Ky4. The reconstructed `fromDate=WorkStartDay(4)=22, toDate=28` shows canonical cycle dates in the UI, which matches what `clock.PayDate(cycle, ...)` expects.
- **Mixed Ky in one file (red-team F4 — accepted):** per-entry resolution handles this correctly. Each entry's `weeklyData.FromDate` is independent. No file-level denormalization.
- **Legacy N+1 (red-team F5 — accepted):** month-level union preserves today's single-query behavior. No regression.
- **Rollback (red-team F6, F9 — accepted):** Phase A does NOT modify `BulkTransferFile.Cycle` (`*string`) or any file-level field. New `CyclePayData` JSON fields are `omitempty`; rolled-back code decodes them fine and ignores them. No data is written that the old binary cannot read.
- **`onepay_exporter.go` thread depth (red-team M1 — accepted):** 2-level signature thread is committed in the plan, not hedged.
- **`ninepay_service.go` (red-team H2 — accepted):** both `buildTransactionCodes` signature AND `btf` literal update are listed explicitly.
