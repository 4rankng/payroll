---
phase: 5
title: "Tests: Unit + Integration + No-Mutation"
status: pending
priority: P1
effort: "L"
dependencies: [1, 2, 3, 4]
---

# Phase 5: Tests: Unit + Integration + No-Mutation

## Overview

Prove the five safety promises of this feature: (a) simulation is **read-only**, (b) simulation and real export pick **the same transaction IDs/totals** on unchanged data, (c) a **stale simulation** is detected, (d) cycle math handles 0, <4, =4, and >4 cycle counts, and (e) the refactor (Phase 1) didn't regress production behavior. Plus unit coverage for the cycle helper, validators, and verdict logic.

## Requirements

- **Functional:** Every acceptance criterion #2–#8 backed by an automated test.
- **Non-functional:** Tests run in `make api-test` (integration) and `go test ./... -race -cover` (unit). No flakiness — deterministic data.

## Architecture

### Unit tests (Go, fast)

| File | Covers |
|------|--------|
| `backend/internal/app/services/payroll/bulktransfer/payroll_cycle_test.go` | `CycleContaining` for days 1/7/8/14/15/21/22/28; `Next()` wrap Kỳ 4 → Kỳ 1 next month; uses fake clock |
| `backend/internal/app/services/payroll/bulktransfer/simulation_validators_test.go` | Each validator: zero/negative, dup-across-batches, broken refs, missing bank code, already-settled, batch>5000 — table-driven |
| `backend/internal/app/services/payroll/bulktransfer/simulation_service_test.go` | Verdict computation: all-`AN_TOAN` case, missing-bank → `CAN_KIEM_TRA`, blocking → `KHONG_THE_TAT_TOAN`; reconciliation delta logic; cycle subtraction; **full-pool verdict** (window-internal clean but full-pool has remainder → `CAN_KIEM_TRA`); uses planner fake |
| `backend/internal/app/services/payroll/bulktransfer/remainder_classifier_test.go` | Money-flow classification: `UNPAID_WAGES` (no wallet_payment), `OP_LOSS` (wallet_payment=completed, receivable unsettled), `STUCK_IN_FLIGHT` (wallet_payment=authorised); mixed-case aggregation; cites wallet_payment_id in evidence |
| `backend/internal/app/services/payroll/bulktransfer/planner_test.go` | Phase 1 regression: `Plan()` returns same selection as a snapshot of the pre-refactor `Export()` would have. Uses seeded repo fakes. |

### Integration tests (Go, `make api-test`)

New file: `backend/tests/integration/flow_settlement_simulation.go`. Registered in `main.go` alongside `flowSaoKe`, `flowManualBulkTransfer`, etc.

```go
const flowSettlementSim = "SettlementSimulation"

func runSettlementSimulationTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
    reporter.PrintSection("FLOW: Settlement Simulation (Mô phỏng đối soát)")
    admin := client.WithToken(data.AdminToken)

    // ── 1. NO-MUTATION PROOF ────────────────────────────────────────
    //    Snapshot row counts + statuses before, run sim, assert after.
    reporter.RunTest(flowSettlementSim, "Simulation makes zero DB mutations", func() error {
        before := snapshotMutationSurface(client) // counts: bulk_transfer_files, transaction_codes, timesheets by payment_status
        _, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", gin.H{"projected_cycle_count": 4})
        if err != nil { return err }
        after := snapshotMutationSurface(client)
        return assertEqualSnapshot(before, after) // fails on any drift
    })

    // ── 2. SIM/PROD PARITY (acceptance criterion #4 + brief) ─────────
    //    Run sim for cycle K. Capture included timesheet IDs + total.
    //    Run real export for same cycle. Parse exported file → same IDs + total.
    reporter.RunTest(flowSettlementSim, "Sim and real export select same IDs/totals on unchanged data", func() error { ... })

    // ── 3. STALE SNAPSHOT (acceptance criterion #6) ──────────────────
    //    Sim → get snapshot_epoch. Touch a timesheet (admin edit). Real export
    //    with if_match_snapshot=old → 409.
    reporter.RunTest(flowSettlementSim, "Stale snapshot is rejected with 409", func() error { ... })

    // ── 4. CYCLE COUNT VARIANTS (brief requirement) ──────────────────
    //    projected_cycle_count ∈ {0, 1, 3, 4, 6, 7}. Assert: 0→400, 7→clamped to 6,
    //    1..6 → that many cycle projections returned.
    reporter.RunTest(flowSettlementSim, "Cycle count variants (0,1,3,4,6,7)", func() error { ... })

    // ── 5. EMPTY POOL ────────────────────────────────────────────────
    //    Find/configure a project+cycle with zero eligible timesheets.
    //    Assert verdict=AN_TOAN_DE_XUAT, all counts 0, reconciled=true.
    reporter.RunTest(flowSettlementSim, "Zero eligible timesheets → clean empty result", func() error { ... })

    // ── 5b. OP-LOSS DETECTION (acceptance criterion #2, the hard case) ─
    //    Seed: an approved timesheet from Kỳ 1 (day 3) with payment_status=failed,
    //    AND a linked wallet_payment in 'completed' state (advance already paid).
    //    Simulate from Kỳ 2 onward (Kỳ 1 window is past).
    //    Assert: remainder contains that item, class=OP_LOSS,
    //            verdict=KHONG_THE_TAT_TOAN, money_flow_evidence cites wallet_payment_id.
    reporter.RunTest(flowSettlementSim, "Op-loss detection: paid-out advance not recovered → KHONG_THE_TAT_TOAN", func() error { ... })

    // ── 5c. STRAGGLER REPORTING (locked decision) ─────────────────────
    //    Seed: Kỳ 1 unpaid timesheet (no wallet_payment) + sim from Kỳ 2.
    //    Assert: remainder class=UNPAID_WAGES, verdict=CAN_KIEM_TRA,
    //            prior-cycle-straggler warning present, NO catch-up batch added.
    reporter.RunTest(flowSettlementSim, "Past-cycle straggler reported as UNPAID_WAGES, not auto-caught", func() error { ... })

    // ── 5d. FULL-POOL vs WINDOW-INTERNAL VERDICT ──────────────────────
    //    Construct: 4-cycle window coverage is internally complete (windows
    //    cover all their own items) BUT a Kỳ-1 item remains in the full pool.
    //    Assert: verdict is CAN_KIEM_TRA (NOT AN_TOAN_DE_XUAT), proving
    //    the verdict uses full-pool logic, not window-internal.
    reporter.RunTest(flowSettlementSim, "Full-pool verdict overrides clean window-internal state", func() error { ... })

    // ── 6. PERMISSIONS ───────────────────────────────────────────────
    //    Employee/partner token → 403.
    reporter.RunTest(flowSettlementSim, "Non-admin token → 403", func() error { ... })
}
```

### The parity test (most important — design carefully)

This is the canonical "simulation doesn't drift" proof. Pseudocode:

```go
// 1. Pick a deterministic test project with seeded timesheets in cycle K.
cycle := CycleContaining(clock.Now()) // or a fixed test cycle
simReq := { project_ids: [testProject], projected_cycle_count: 1, for_cycle: cycle }
simResp, _ := admin.Post("/api/v1/payrolls/simulate-settlement", simReq)
simIDs := collectTimesheetIDs(simResp.cycles[0].included)
simTotal := simResp.cycles[0].included_amount

// 2. Run the REAL export for the same cycle (this DOES mutate — set up data fresh
//    for this test or run it last, or better: capture sim first, then export, then
//    parse the file).
exportReq := { project_ids: [testProject], fromDate: cycle.from, toDate: cycle.to }
exportResp, _ := admin.DownloadPost("/api/v1/payrolls/export-bulk-transfer", exportReq)
exportedIDs, exportedTotal := parseBulkTransferExcel(exportResp.body) // reuse helpers from flow_manual_bulk_transfer.go

// 3. Assert
assert(simIDs == exportedIDs)         // set equality
assert(simTotal == exportedTotal)     // int64 equality
```

Order matters: capture the sim first (read-only), then export (mutates by marking items), then parse the file. If the test flips ordering the sim will see post-export statuses. Run this test with a **freshly seeded project** each time — do not depend on global test state.

### The no-mutation snapshot helper

```go
type mutationSurface struct {
    BulkTransferFiles    int64
    TransactionCodes     int64
    TimesheetsByStatus   map[string]int64
    LedgerEntries        int64
    SettlementUploads    int64
}

func snapshotMutationSurface(client *APIClient) mutationSurface {
    // Use direct DB handle (tests/integration has one) — NOT the API, since
    // there's no API to count rows. Mirror the pattern in existing flow_*.go
    // files that read the test DB directly.
}
```

This test is the **real** enforcement of "strictly read-only" — stronger than the GORM `ReadOnly: true` flag, which some MySQL configs ignore.

## Related Code Files

- **Create:** `backend/internal/app/services/payroll/bulktransfer/payroll_cycle_test.go`.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_validators_test.go`.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/simulation_service_test.go`.
- **Create:** `backend/internal/app/services/payroll/bulktransfer/planner_test.go` (Phase 1 regression).
- **Create:** `backend/tests/integration/flow_settlement_simulation.go`.
- **Modify:** `backend/tests/integration/main.go` — register `runSettlementSimulationTests` in the suite.
- **Reference:** `backend/tests/integration/flow_manual_bulk_transfer.go` (DownloadPost + Excel parse helpers), `backend/tests/integration/models.go` (shared types).

## Implementation Steps

1. **Write `payroll_cycle_test.go` first** (no DB needed) to lock the cycle model before integration tests rely on it.
2. **Write `simulation_validators_test.go`** — table-driven, pure inputs, no DB.
3. **Write `simulation_service_test.go`** with a fake `ExportPlanner` (interface extracted if not already) and a fake ledger repo. Lock the verdict table.
4. **Write `planner_test.go`** — this is Phase 1's safety net. Seed a fake repo with a known timesheet set; assert `Plan()` returns the expected `ExportPlan` (same as a golden snapshot of pre-refactor output).
5. **Write the integration flow.** Add the `mutationSurface` helper (direct DB read). Write the 6 subtests in order: no-mutation → parity → stale → cycle-count → empty → permissions.
6. **Register in `main.go`.**
7. **Run the full suite:** `make api-test`. All green.
8. **Run unit suite with race + coverage:** `cd backend && go test ./internal/app/services/payroll/bulktransfer/... -v -race -cover`. Coverage for the new package should be ≥ 80%.

## Success Criteria

- [ ] `payroll_cycle_test.go`: 8+ cases covering all Kỳ boundaries and wrap.
- [ ] `simulation_validators_test.go`: every validator code has ≥1 pass + ≥1 fail case.
- [ ] `simulation_service_test.go`: 3 verdict paths + reconciliation + cycle subtraction + **full-pool verdict override**.
- [ ] `remainder_classifier_test.go`: all 3 classes + mixed aggregation; every `OP_LOSS` cites a wallet_payment_id.
- [ ] `planner_test.go`: Phase 1 regression — selection matches golden snapshot.
- [ ] Integration: no-mutation test passes (row counts + status map identical before/after).
- [ ] Integration: parity test passes (sim IDs == exported IDs as sets; totals equal as `int64`).
- [ ] Integration: stale snapshot → 409.
- [ ] Integration: cycle count 0→400, 7→6-clamp, 1/3/4/6 → exact count.
- [ ] Integration: empty pool → clean `AN_TOAN_DE_XUAT` with zero counts.
- [ ] Integration: op-loss detection (paid-out advance + Kỳ-1 remainder from Kỳ 2) → `KHONG_THE_TAT_TOAN`.
- [ ] Integration: straggler reported as `UNPAID_WAGES` + prior-cycle warning, no catch-up batch.
- [ ] Integration: full-pool verdict overrides clean window-internal state.
- [ ] Integration: non-admin → 403.
- [ ] `make api-test` fully green.
- [ ] Backend unit coverage ≥ 80% for `bulktransfer` package.

## Risk Assessment

**Risk: parity test is flaky because test data isn't isolated.**
Mitigation: dedicate a project ID to this test (seed in `TestConfig` or create+teardown). Never share mutable fixtures with other flows. Document the dependency.

**Risk: no-mutation test misses a write because it runs against a stale connection.**
Mitigation: the snapshot helper reads via a fresh `BEGIN RO BACKEND` or `SELECT ... FOR ...` — actually a plain `SELECT COUNT(*)` is fine; just ensure it runs after the API call fully returns. Use the same DB handle the rest of the suite uses.

**Risk: golden snapshot in `planner_test.go` rots when seed data changes.**
Mitigation: build the golden expectation from the same seed fixtures programmatically (i.e. derive expected IDs from the seed array), don't hardcode. Then any seed change updates both sides together.

**Risk: test order dependency — parity test mutates, breaking later tests.**
Mitigation: order the suite so mutations come last, OR scope each mutating test to its own project/timesheet set and clean up. The no-mutation test must run first.
