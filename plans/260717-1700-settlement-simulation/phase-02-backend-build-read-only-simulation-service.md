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

### Cycle date ranges (the Kỳ 1–4 model)

```go
// payrollCycle.go (new, tiny helper)
//
// A payroll month is divided into 4 cycles. Each cycle covers a 7-day window
// and is paid ~3 days after the window closes:
//   Kỳ 1: days  1–7  → paid day 10
//   Kỳ 2: days  8–14 → paid day 17
//   Kỳ 3: days 15–21 → paid day 24
//   Kỳ 4: days 22–28 → paid day  1 of the NEXT month

type PayrollCycle struct {
    Index    int       // 1..4
    MonthRef time.Time // first-of-month this cycle belongs to
    FromDate time.Time
    ToDate   time.Time
    PayDate  time.Time
}

// CycleContaining returns the Kỳ 1..4 that contains t (Ho Chi Minh time).
func CycleContaining(t time.Time) PayrollCycle { ... }

// Next returns the next cycle in monthly sequence, wrapping Kỳ 4 → Kỳ 1 of next month.
func (c PayrollCycle) Next() PayrollCycle { ... }
```

The simulation starts at `CycleContaining(clock.Now())` and walks `.Next()` up to `req.ProjectedCycleCount` times (default 4, max 6).

### Per-cycle planning

```go
for i := 0; i < req.ProjectedCycleCount; i++ {
    cycleReq := cloneRequestForCycle(req, cycle) // sets FromDate/ToDate, clears ForMonth (weekly mode)
    plan, err := planner.Plan(ctx, cycleReq)
    if err != nil { return nil, fmt.Errorf("cycle %d: %w", cycle.Index, err) }

    // Subtract timesheets already included in earlier cycles (sim-only: in production
    // this happens automatically because status flips to paid between exports).
    alreadyIncluded := unionOfTimesheetIDs(prevCycles)
    included, excluded, remaining := partition(plan, alreadyIncluded)

    // Run sim-only validators (Phase 2 core — see "Extra validators" below)
    findings := simValidators.Run(included, cycle)

    cycles = append(cycles, CycleProjection{...})
    cycle = cycle.Next()
}
```

**Important modeling note:** because `Plan()` selects `payment_status IN (pending, failed)` and we don't actually flip statuses between simulated cycles, every cycle would otherwise return the **same** set. The simulation's job is therefore: (a) show what production **would** include in this cycle's date window, and (b) explicitly model "items already settled by a prior simulated cycle" as the subtraction. This is honest: it answers *"if exports 1..k each succeed at the bank, what's left for cycle k+1?"*

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

### Verdict computation (full-pool)

The verdict is computed against the **entire outstanding pool**, not the union of the N windows:

```go
// 1. Compute the FULL outstanding pool: all approved timesheets with
//    payment_status IN (pending, failed), NO date filter, project/employee filters only.
fullPool := planner.Plan(ctx, reqWithNoDateFilter)  // reuse same code path
fullPoolIDs := setOf(fullPool.ValidatedData timesheet IDs)

// 2. Compute what the N projected windows WILL cover
coveredIDs := unionOf(cycle.Included timesheet IDs for cycle in projectedCycles)

// 3. Remainder = full pool minus covered
remainderIDs := fullPoolIDs - coveredIDs

// 4. Classify each remainder item by money-flow (see classification table above)
remainders := classifyByMoneyFlow(ctx, remainderIDs)
//   For each timesheet:
//     - query linked wallet_payment(s) by entity_id chain (timesheet → transaction → settlement → wallet_payment,
//       OR timesheet → advance_payment_request → wallet_payment)
//     - if any wallet_payment.Status == completed       → OP_LOSS (money left, not recovered)
//     - elif any wallet_payment.Status in (pending, verified, authorised) → STUCK_IN_FLIGHT
//     - else                                              → UNPAID_WAGES
//   Classify at the timesheet level; aggregate to employee×project for display.

// 5. Verdict
hasOpLoss        := any(remainders, class == OP_LOSS)
hasUnpaidWages   := any(remainders, class == UNPAID_WAGES)
hasBlocking      := any(findings, severity == blocking)
deltaNonZero     := reconciliation.Delta != 0

switch {
case hasOpLoss || (hasBlocking && hasUnpaidWages):
    Verdict = "KHONG_THE_TAT_TOAN"   // Không thể tất toán — op-loss or blocking unpaid items
case hasUnpaidWages || hasBlocking || deltaNonZero || len(warnings) > 0:
    Verdict = "CAN_KIEM_TRA"          // Cần kiểm tra — gaps exist but recoverable
default:
    Verdict = "AN_TOAN_DE_XUAT"       // An toàn để xuất — full pool covered, reconciled, clean
}
```

**Verdict semantics (locked):**
- `AN_TOAN_DE_XUAT`: every outstanding approved timesheet is covered by one of the N windows, no blocking findings, reconciliation delta = 0.
- `CAN_KIEM_TRA`: some outstanding items are NOT covered, OR validation/reconciliation problems exist — admin must investigate but the situation is recoverable with manual action.
- `KHONG_THE_TAT_TOAN`: at least one `OP_LOSS` remainder exists (money already disbursed, receivable unrecovered) OR blocking findings on unpaid wages. This is the "stop and escalate" verdict.

### Past-cycle straggler handling (locked decision)

Per user decision: **just report them**. Stragglers (items in `fullPoolIDs` but outside all N projected windows) appear in the `remainders` list with their op-loss class. The simulation does **not** auto-add a catch-up batch and does **not** widen windows backward. The admin sees exactly: "K VND unpaid wages + L VND op-loss will remain after these N exports" and decides manually.

This is surfaced at the top of the dialog:
> "Các giao dịch thuộc kỳ trước vẫn chưa thanh toán sẽ KHÔNG được tự động bao gồm. Xem danh sách 'Còn lại' để xử lý thủ công."
> ("Items from prior cycles that remain unpaid will NOT be auto-included. See the 'Remaining' list to handle manually.")

### Read-only enforcement

```go
func (s *SimulationService) Simulate(ctx context.Context, req *dto.SimulateSettlementRequest) (*dto.SimulationResult, error) {
    // Wrap in a read-only tx so a programming error can't mutate.
    return s.db.Transaction(func(tx *gorm.DB) error {
        roCtx := context.WithValue(ctx, ctxKeyReadOnlyTx{}, tx)
        result, err := s.simulate(roCtx, req)
        // always return err == nil at the end so the tx rolls back cleanly;
        // carry result/err out via closure.
        ...
    }, &gorm.Session{&sql.TxOptions{ReadOnly: true}})
}
```

If the underlying MySQL driver or GORM version makes `ReadOnly: true` a no-op, the secondary guarantee is that `simulate()` only ever calls `ExportPlanner.Plan()` (which Phase 1 made pure) and the read-only ledger query. Add a `t.Helper`-style assertion test in Phase 5 that no write was issued (assert row counts of `bulk_transfer_files`, `transaction_codes`, `timesheets.payment_status` unchanged before/after).

## Related Code Files

- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_service.go` — `SimulationService`, `Simulate`, cycle loop, verdict, full-pool scan.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/payroll_cycle.go` — `PayrollCycle`, `CycleContaining`, `Next` (uses `clock.Now()` in `Asia/Ho_Chi_Minh`).
- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_validators.go` — `simValidators.Run`, `SimFinding`.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/remainder_classifier.go` — `classifyByMoneyFlow(ctx, remainderIDs) → []ClassifiedRemainder`. Queries wallet_payment linkage per item.
- **Read-only deps:** `backend/internal/app/services/payroll/bulktransfer/planner.go` (Phase 1), `backend/internal/app/services/payroll/excel/service.go`, `backend/internal/infra/persistence/ledger_repository_queries.go`, `backend/internal/infra/persistence/wallet_payment_repository.go` (for money-flow classification).
- **DTO (added in Phase 3 but referenced here):** `dto.SimulateSettlementRequest`, `dto.SimulationResult`, `dto.ClassifiedRemainder` — see Phase 3 for full shape.

## Implementation Steps

1. **Write `payroll_cycle.go` first** with unit tests (`payroll_cycle_test.go`): assert `CycleContaining` for days 1, 7, 8, 14, 15, 21, 22, 28, month-wrap. Uses `clock.Now()` per CLAUDE.md.
2. **Define `SimulationService` struct** holding: `planner *ExportPlanner`, `ledgerRepo`, `clock`, `db *gorm.DB`, `logger`.
3. **Implement `Simulate`:**
   - Resolve starting cycle from `clock.Now()`.
   - Loop `ProjectedCycleCount` times (default 4, clamp 1–6).
   - For each: build a cycle-scoped `ExportBulkTransferRequest` (weekly mode, `FromDate`/`ToDate` = cycle range; drop `ForMonth`), call `planner.Plan()`, partition against already-included IDs, run sim-only validators, accumulate.
   - Compute reconciliation via one ledger `SUM`.
   - Compute verdict.
   - Track `snapshotEpoch = max(cycle.Plan.SnapshotEpoch)` across cycles.
4. **Implement `simulation_validators.go`** as a slice of small functions `func(*ValidationContext) []SimFinding`. Each is pure. Mark `InProduction: bool` per validator for UI messaging.
5. **Wrap in read-only transaction.** Verify `sql.TxOptions{ReadOnly: true}` is honored; if not, document and rely on the "Plan is pure" guarantee + Phase 5 no-mutation test.
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

**Risk: MySQL silently ignores `ReadOnly: true`.**
Mitigation: belt-and-suspenders — Phase 1 guarantees `Plan()` is pure. Phase 5 includes an explicit no-mutation integration test that asserts row counts and statuses are unchanged.

**Risk: `int64` overflow on very large sums.**
Not realistic for VND payroll (would require ~9.2 × 10^18 VND). Document as accepted.
