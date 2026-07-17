---
phase: 2
title: "Backend: Build Read-Only Simulation Service"
status: pending
priority: P1
effort: "L"
dependencies: [1]
---

# Phase 2: Backend: Build Read-Only Simulation Service

## Overview

`SimulationService.Simulate(ctx, req)` projects the current payroll cycle + next N−1 cycles (default N=4, the Kỳ 1–4 monthly cadence). For each cycle it calls **the same `ExportPlanner.Plan()` refactored in Phase 1**, accumulates results, subtracts already-included timesheet IDs from later cycles, runs **sim-only extra validators** that production doesn't have (per user decision), reconciles totals against the ledger, computes a verdict, and returns a structured `SimulationResult`. **Zero writes.** Runs inside a read-only DB transaction as belt-and-suspenders.

### The verdict is full-pool, not window-internal (locked decision)

The verdict answers: **"Of ALL outstanding approved timesheets (full pool, NO date filter), how many will these N projected exports cover, and how many will remain unsettled?"** Not "are these N windows internally complete." Past-cycle stragglers (a day-3 timesheet still `failed` when projecting from Kỳ 2 onward) are therefore **failures of coverage**, not edge cases.

Each remaining item after all N cycles is classified by **money-flow direction** because the business consequence differs:

| Remainder class | Definition | Business meaning |
|-----------------|------------|------------------|
| `UNPAID_WAGES` | Timesheet not yet disbursed to employee (no linked `wallet_payment` in terminal state, no linked advance payout) | **Employee is owed money** — must be re-exported |
| `OP_LOSS` | Timesheet where money already left our account (advance paid out, or wallet_payment reached `completed`) but receivable not settled against the client | **Operational loss / unrecovered cost** — client billing gap |
| `STUCK_IN_FLIGHT` | Has a non-terminal `wallet_payment` (pending/verified/authorised) | **Pending bank outcome** — re-export would double-pay; wait |

This classification is the single most important business output of the feature: it tells the admin not just "X items remain" but "X items = Y VND unpaid wages + Z VND op-loss + W VND in-flight."

## Requirements

- **Functional:** Project 1–6 cycles. Per cycle: included count/amount, excluded-with-reasons, remaining-after. Across all cycles: **full-pool** verdict, op-loss classification of every remaining item, reconciliation, warnings, snapshot epoch.
- **Non-functional:** Strictly read-only (no DB writes, no status flips, no transaction-code rows, no `bulk_transfer_files` rows, no events). Deterministic ordering. `int64` VND throughout — no decimal drift.

## Architecture

### Cycle date ranges — REUSE `clock.PayCycle` (red-team Finding 3)

**Do not create `payroll_cycle.go`.** The canonical model already exists at `backend/internal/pkg/clock/pay_cycle.go`:
- `clock.KyFromWorkDay(day int) int` → returns 1..4
- `clock.WorkStartDay(ky int) int` → 1, 8, 15, 22
- `clock.PayDate(ky, year, month) time.Time` → handles Kỳ 4 wrap to next-month day 1
- `clock.NextTimesheetPayCycle(t) TimesheetPayCycle` → the next upcoming cycle relative to `t`

The simulation starts at `clock.NextTimesheetPayCycle(clock.Now())` and walks forward via a **small new helper** `clock.NextPayCycleAfter(current TimesheetPayCycle) TimesheetPayCycle` (added to `pay_cycle.go`, ~15 lines, wraps Kỳ 4 → Kỳ 1 next month). Each cycle's date window:
- `FromDate` = `time.Date(year, month, clock.WorkStartDay(ky), ..., DefaultLocation)`
- `ToDate`   = `FromDate.AddDate(0, 0, 6)` (7-day window)
- `PayDate`  = `clock.PayDate(ky, year, month)`

If `clock.NextPayCycleAfter` already exists, use it; otherwise add it. This is the only new code in the cycle layer — everything else is reuse.

### Per-cycle planning

```go
startCycle := clock.NextTimesheetPayCycle(s.clock.Now())
cycle := startCycle
var coveredTimesheetIDs map[uint]struct{}
var cycles []CycleProjection

for i := 0; i < req.ProjectedCycleCount; i++ {
    cycleReq := cloneRequestForCycle(req, cycle) // sets FromDate/ToDate, clears ForMonth (weekly mode)
    plan, err := planner.Plan(ctx, cycleReq)
    if err != nil { return nil, fmt.Errorf("cycle Kỳ %d: %w", cycle.Ky, err) }

    // Surface the timesheet IDs that Plan() aggregated. Plan's RawAggregated
    // is *excel.BulkTransferData, which DOES carry EmployeeProjectTimesheets
    // (map[EmployeeProjectKey][]uint). Flatten it to get the included set.
    cycleIncludedIDs := flattenTimesheetIDs(plan.RawAggregated.EmployeeProjectTimesheets)

    // Track cumulative coverage for the full-pool verdict (computed after the loop)
    for id := range cycleIncludedIDs { coveredTimesheetIDs[id] = struct{}{} }

    // Run sim-only validators on this cycle's included data
    findings := simValidators.Run(plan, cycle)

    cycles = append(cycles, CycleProjection{
        Sequence: i + 1,
        Label:    fmt.Sprintf("Kỳ %d%s", cycle.Ky, i == 0 && " (hiện tại)"),
        ...,
        Included:    buildRowsFromAggregated(plan.ValidatedData.ValidData, masker),
        Excluded:    buildExcludedRows(plan.ValidatedData.SkippedEmployees),
        Findings:    findings,
    })

    cycle = clock.NextPayCycleAfter(cycle) // walks Kỳ 1→2→3→4→(next month)1
}
```

**Red-team correction (Finding 6):** the original plan said "subtract already-included timesheet IDs from later cycles". That step is now **removed** — it was based on a misreading that `Plan()` returns per-timesheet IDs at the top level. In reality each cycle's `Plan()` runs against that cycle's date window, and an item in Kỳ 1's window will NOT appear in Kỳ 2's `Plan()` output (different date range). So the cycles are naturally disjoint by date. The cross-cycle dedup problem only arises at the **full-pool verdict** step (next subsection), where we compute `fullPool − coveredAcrossAllCycles`.

### Extra sim-only validators (not in production — per user decision)

Implemented in `simulation_validators.go`. Each emits a `SimFinding{Severity, Code, Msg, Refs}`. **Crucially, each one also emits a companion warning when production does NOT run it**, surfaced in the UI as "SẢN XUẤT KHÔNG KIỂM TRA ĐIỀU NÀY".

| Code | Check | In production? | Severity on fail |
|------|-------|----------------|------------------|
| `ZERO_NEGATIVE_AMOUNT` | `amount <= 0` | ❌ No | blocking |
| `DUPLICATE_WITHIN_BATCH` | same employee×project appears twice in one cycle | (cannot happen — aggregation dedups) | N/A — info only |
| `DUPLICATE_ACROSS_BATCHES` | timesheet ID appears in two cycles | ❌ No (status flip prevents in prod) | blocking in sim |
| `BROKEN_EMPLOYEE_REF` | `employeeData[id]` missing/incomplete | partial (export errors out) | blocking |
| `BROKEN_PROJECT_REF` | `projectData[id]` missing | partial | blocking |
| `MISSING_BANK_CODE` | `employee.Bank == nil` or `BankCode == ""` | ❌ No (only checks account number/name) | warning |
| `CURRENCY_PRECISION` | non-integer cents / currency != VND | ❌ No (always int64 VND) | info — "always passes, logged for completeness" |
| `BATCH_SIZE_OVER_RECOMMENDED` | included rows > 5000 (advisory, no prod limit) | ❌ No | warning |
| `CONCURRENT_MODIFICATION` | `plan.SnapshotEpoch` differs across cycles in the same sim run | ❌ No | warning |
| `ALREADY_SETTLED_INCLUDED` | a timesheet with `payment_status=paid` or `revenue_paid=true` slipped into the eligible set | partial (status filter) | blocking |

Validators that production **does** run (`ValidateAndFilterBulkTransferData` — missing bank account number/name) are not re-run; their results come straight from `plan.ValidatedData.SkippedEmployees`.

### Reconciliation

```go
// Total actually includable across all cycles
totalIncluded := sum(cycle.IncludedAmount for cycle in cycles)

// Ledger receivable for the same period (double-entry: receivable account, party=employee)
// Reuse the existing ledger query path — do NOT roll a new one.
fromDate := cycles[0].FromDate
toDate   := lastCycle.ToDate
receivable, err := ledgerRepo.GetAccountBalanceByDateRange(ctx, "receivable", fromDate, toDate)

reconciliation := Reconciliation{
    ExportedTotal:    totalIncluded,
    LedgerReceivable: receivable,
    Delta:            receivable - totalIncluded,
    Reconciled:       delta == 0, // int64 — exact equality
}
```

Use the lightest existing ledger query (e.g. a `SUM` over `ledger_entries` filtered by account + date). Avoid loading full ledger rows. Confirm the exact method with `backend/internal/infra/persistence/ledger_repository_queries.go` during implementation; if none fits, add one read-only `SUM` query — no new mutation.

### Verdict computation (full-pool, corrected classifier chain)

The verdict is computed against the **entire outstanding pool**, not the union of the N windows:

```go
// 1. Compute the FULL outstanding pool: all approved timesheets with
//    payment_status IN (pending, failed), NO date filter, project/employee filters only.
//    (Phase 1 added req.noDateFilter=true to make Plan() support this — red-team Finding 8)
fullPoolReq := cloneRequest(req)
fullPoolReq.noDateFilter = true
fullPoolPlan, err := planner.Plan(ctx, fullPoolReq)
fullPoolIDs := flattenTimesheetIDs(fullPoolPlan.RawAggregated.EmployeeProjectTimesheets)

// 2. Compute what the N projected windows WILL cover (accumulated in the loop above)
// coveredTimesheetIDs is already populated.

// 3. Remainder = full pool minus covered
remainderIDs := setDifference(fullPoolIDs, coveredTimesheetIDs)

// 4. Classify each remainder item by money-flow via the REAL chain
//    (red-team Finding 2: original chain was wrong; this is the corrected path)
//
//    timesheet.ID
//      → transaction_codes (where JSON_CONTAINS(data->'$.weekly_pay.timesheet_ids', id)
//                            OR JSON_CONTAINS(data->'$.monthly_pay.timesheet_ids', id))
//      → transaction_codes.code
//      → wallet_payments (where txn_id = code)
//      → wallet_payments.status
remainders := classifyByMoneyFlow(ctx, remainderIDs)
//   Classification rule per timesheet ID:
//     - matchedCodes := transactionCodeRepo.FindByTimesheetID(ctx, id)  // new indexed query, see below
//     - wpRows := walletPaymentRepo.GetByTxnIDs(ctx, codes-of-matchedCodes)  // batch
//     - if any wp.Status == completed                     → OP_LOSS
//     - elif any wp.Status in (pending,verified,authorised) → STUCK_IN_FLIGHT
//     - else (no codes, no wallet rows, or all failed)    → UNPAID_WAGES

// 5. Verdict
hasOpLoss        := any(remainders, class == OP_LOSS)
hasUnpaidWages   := any(remainders, class == UNPAID_WAGES)
hasBlocking      := any(cycle.Findings for cycle in cycles, severity == blocking)
deltaNonZero     := reconciliation.Delta != 0

switch {
case hasOpLoss || (hasBlocking && hasUnpaidWages):
    Verdict = "KHONG_THE_TAT_TOAN"
case hasUnpaidWages || hasBlocking || deltaNonZero || len(warnings) > 0:
    Verdict = "CAN_KIEM_TRA"
default:
    Verdict = "AN_TOAN_DE_XUAT"
}
```

**Two new repository queries required** (red-team Findings 3 & 7 — neither exists today):
1. `transactionCodeRepo.FindByTimesheetIDs(ctx, ids []uint) (map[uint][]*TransactionCode, error)` — batch query: `SELECT * FROM transaction_codes WHERE JSON_CONTAINS(data->'$.weekly_pay.timesheet_ids', ?) OR JSON_CONTAINS(data->'$.monthly_pay.timesheet_ids', ?)`. Build the reverse map in Go. MySQL 8 supports `JSON_CONTAINS` and the `data` column is `JSON` type.
2. `walletPaymentRepo.GetByTxnIDs(ctx, txnIDs []string) ([]*WalletPayment, error)` — batch lookup (avoids the N+1 red-team Finding 3 flagged).

Both are read-only SELECTs against indexed columns.

### Reconciliation (red-team Finding 7 — new ledger method)

The existing `ledger_repository_queries.go` has NO method that returns a SUM of an account over a date range. `GetCumulativeTotalsBeforeDate` returns the wrong shape (and uses `float64`). Phase 2 adds **one** new read-only method:

```go
// LedgerEntryRepository (interface addition)
// GetAccountTotalInRange returns SUM(credit) - SUM(debit) for the given account
// over [from, to]. int64 VND — no float.
GetAccountTotalInRange(ctx context.Context, account domain.LedgerAccount, from, to time.Time) (int64, error)
```

SQL: `SELECT COALESCE(SUM(credit), 0) - COALESCE(SUM(debit), 0) FROM ledger_entries WHERE account = ? AND deleted_at IS NULL AND entry_date BETWEEN ? AND ?`. Single round-trip, indexed.

Reconciliation then:
```go
receivable, err := ledgerRepo.GetAccountTotalInRange(ctx, domain.AccountReceivable, firstCycle.FromDate, lastCycle.ToDate)
reconciliation := Reconciliation{
    ExportedTotal:    totalIncluded, // int64
    LedgerReceivable: receivable,    // int64
    Delta:            receivable - totalIncluded,
    Reconciled:       receivable == totalIncluded, // exact int64 equality
}
```

### Read-only enforcement (red-team Finding 9 — corrected)

**Do NOT rely on `sql.TxOptions{ReadOnly: true}`** — red-team verified the configured `go-sql-driver/mysql` silently ignores it. The read-only guarantee rests on two real pillars:

1. **Code-level:** `SimulationService.Simulate()` only ever calls: `ExportPlanner.Plan()` (Phase 1 makes this provably write-free — grep-verified zero `fileRepo.*`/`transactionCodeRepo.Create*`/`eventBus.Publish`/`*Update*` calls), the new read-only `GetAccountTotalInRange`, `FindByTimesheetIDs`, and `GetByTxnIDs`. A grep assertion test in Phase 5 enforces no write calls appear in the simulation source.
2. **Test-level:** Phase 5's canonical no-mutation integration test snapshots row counts of `bulk_transfer_files`, `transaction_codes`, `timesheets` (by `payment_status`), `wallet_payments`, and `ledger_entries` before and after a simulation call, and fails on any drift.

A plain `db.WithContext(ctx)` (no transaction wrapper) is sufficient. The simulation may OPTIONALLY wrap in a transaction for **snapshot consistency** (so all N cycles see the same data view) — but the safety claim does not depend on the tx being read-only.

## Related Code Files

- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_service.go` — `SimulationService`, `Simulate`, cycle loop, verdict, full-pool scan.
- **Modify:** `backend/internal/pkg/clock/pay_cycle.go` — add `NextPayCycleAfter(current TimesheetPayCycle) TimesheetPayCycle` (~15 lines, wraps Kỳ 4 → Kỳ 1 next month). Do NOT create a new cycle file.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_validators.go` — `simValidators.Run`, `SimFinding`.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/remainder_classifier.go` — `classifyByMoneyFlow(ctx, remainderIDs) → []ClassifiedRemainder`. Traverses timesheet → transaction_codes → wallet_payments (the corrected chain).
- **Modify (interface additions):** `backend/internal/infra/persistence/ledger_repository_queries.go` + `ledger.go` repo interface — add `GetAccountTotalInRange`.
- **Modify (interface additions):** `backend/internal/domain/transaction_code.go` (repo interface) — add `FindByTimesheetIDs`.
- **Modify (interface additions):** `backend/internal/domain/transactions/repository.go` (wallet_payment repo interface) — add `GetByTxnIDs`.
- **Read-only deps:** `backend/internal/app/services/payroll/bulktransfer/planner.go` (Phase 1), `backend/internal/app/services/payroll/excel/service.go`.
- **DTO (added in Phase 3 but referenced here):** `dto.SimulateSettlementRequest`, `dto.SimulationResult`, `dto.ClassifiedRemainder` — see Phase 3 for full shape.

## Implementation Steps

1. **Add `clock.NextPayCycleAfter`** to `backend/internal/pkg/clock/pay_cycle.go` (~15 lines) + unit test in `pay_cycle_test.go`: assert wrap Kỳ 4 → Kỳ 1 next month. Reuses the existing canonical model — do NOT create a parallel one.
2. **Define `SimulationService` struct** holding: `planner *ExportPlanner`, `ledgerRepo`, `transactionCodeRepo`, `walletPaymentRepo`, `clock`, `db *gorm.DB`, `logger`.
3. **Implement `Simulate`:**
   - Resolve starting cycle from `clock.Now()`.
   - Loop `ProjectedCycleCount` times (default 4, clamp 1–6).
   - For each: build a cycle-scoped `ExportBulkTransferRequest` (weekly mode, `FromDate`/`ToDate` = cycle range; drop `ForMonth`), call `planner.Plan()`, partition against already-included IDs, run sim-only validators, accumulate.
   - Compute reconciliation via one ledger `SUM`.
   - Compute verdict.
   - Track `snapshotEpoch = max(cycle.Plan.SnapshotEpoch)` across cycles.
4. **Implement `simulation_validators.go`** as a slice of small functions `func(*ValidationContext) []SimFinding`. Each is pure. Mark `InProduction: bool` per validator for UI messaging.
5. **Read-only enforcement:** do NOT wrap in a read-only transaction (red-team Finding 9 — `ReadOnly:true` is a no-op in the configured MySQL driver). The safety guarantee is structural: `Simulate()` only calls the pure `Plan()` + read-only repo queries. A grep-assertion test in Phase 5 enforces no write calls appear in the simulation source.
6. **Determinism:** sort `Included`, `Excluded`, `Remaining` slices by `(employee_id, project_id, timesheet_id)` before returning.

## Success Criteria

- [ ] `SimulationService.Simulate` returns a populated `SimulationResult` with 1–6 cycle projections.
- [ ] **Full-pool scan** runs (one extra `Plan()` call with no date filter) and `remainders` is computed as `fullPool − covered`.
- [ ] Every remainder item is classified `UNPAID_WAGES` / `OP_LOSS` / `STUCK_IN_FLIGHT` with a non-empty evidence chain (linked wallet_payment status cited).
- [ ] Verdict is one of `AN_TOAN_DE_XUAT` / `CAN_KIEM_TRA` / `KHONG_THE_TAT_TOAN` per the locked semantics (any `OP_LOSS` → `KHONG_THE_TAT_TOAN`).
- [ ] Zero calls to `fileRepo.*Create*`, `transactionCodeRepo.Create*`, `eventBus.Publish`, `timesheetRepo.BulkUpdate*`, `settlement*` writes.
- [ ] Reconciliation delta is exact integer (`int64`) equality.
- [ ] Each sim-only validator that fails also surfaces a "production does not check this" warning.
- [ ] `SnapshotEpoch` returned equals `max(updated_at)` of any timesheet the planner observed (full-pool scan included).
- [ ] Ordering is deterministic across repeated calls with the same data.

## Risk Assessment

**Risk: cycle subtraction model doesn't match real production behavior.**
In production, an item paid in Kỳ 1 won't appear in Kỳ 2's `Plan()` because its status is `paid`. In the sim we don't flip statuses, so we must subtract manually. **Mitigation:** the model is explicitly documented as "assuming each prior cycle succeeds at the bank." Surfaced as a top-of-dialog caveat: *"Mô phỏng giả định mỗi kỳ trước đó đã được ngân hàng thanh toán thành công."*

**Risk: ledger `SUM` query doesn't exist or is expensive.**
Mitigation: use the lightest existing query in `ledger_repository_queries.go`; if a new one is needed it's a single indexed `SUM(credit) - SUM(debit)` on `ledger_entries` for `account='receivable'`. Bounded by date range. Not N+1.

**Risk: a write call leaks into the simulation path during future maintenance.**
Mitigation: Phase 5's grep-assertion test (`TestSimulationSourceHasNoWriteCalls`) fails the build if any forbidden write-call substring appears in the simulation source files. Plus the row-count snapshot integration test catches runtime mutations.

**Risk: `int64` overflow on very large sums.**
Not realistic for VND payroll (would require ~9.2 × 10^18 VND). Document as accepted.
