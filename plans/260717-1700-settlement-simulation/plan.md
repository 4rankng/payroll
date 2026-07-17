---
title: "Statement Settlement Simulation (Mô phỏng đối soát) for /admin/ledger"
description: "Read-only dry-run that projects the current payroll bulk-transfer export plus the next 3 monthly payout cycles (Kỳ 1–4 cadence) using the EXACT same selection/validation logic as production. Surfaces included, excluded, remaining, and reconciliation status before any download or upload."
status: pending
priority: P1
branch: "main"
tags: [payroll, bulk-transfer, settlement, simulation, ledger, dry-run, admin]
blockedBy: []
blocks: []
created: "2026-07-17T08:43:04.397Z"
createdBy: "ck:plan"
source: skill
---

# Statement Settlement Simulation (Mô phỏng đối soát) for /admin/ledger

## Overview

Add a read-only "Mô phỏng đối soát" (settlement simulation) feature to `/admin/ledger` that lets an admin preview the **current** payroll bulk-transfer export plus the **next 3** projected exports (the monthly Kỳ 1–4 cadence: days 1–7, 8–14, 15–21, 22–28) — **before** downloading or uploading anything — and get an answer-first verdict on whether those exports will settle every eligible outstanding timesheet.

The simulation **must not drift** from production: it reuses the exact same `ExportService.Export()` selection/aggregation/validation pipeline as `POST /api/v1/payrolls/export-bulk-transfer`. The only difference is that every side effect (saving `bulk_transfer_files`, creating `transaction_codes`, audit events, Excel generation) is **suppressed** and the resulting batch projections are returned as structured JSON instead.

### The problem this solves

The admin's stated worry is that exporting a bank payment file ("xuất sao kê") may (1) contain wrong amounts/transactions, (2) miss/duplicate/mis-assign transactions, (3) fail to settle all eligible items even across the current + next 3 cycles, and (4) not reconcile against the ledger. This feature answers all four questions **without mutating the database**.

### How "current + next 3 exports" maps to this codebase

The production export (`ExportService.Export`) takes **100% of currently-eligible timesheets** in one shot — there is no batch splitting, no row limit, no sequential sequencing. So "the next 3 exports" cannot mean "the next 3 chunks of this pool."

Per the user's confirmed cadence, **the payroll is paid in 4 weekly cycles each month**:

| Kỳ  | Pay day (next month for Kỳ 4) | Covers days |
|-----|-------------------------------|-------------|
| 1   | Day 10                        | 1–7         |
| 2   | Day 17                        | 8–14        |
| 3   | Day 24                        | 15–21       |
| 4   | Day 1 (next month)            | 22–28 (prev month) |

The simulation therefore projects **the current cycle (whichever of Kỳ 1–4 "today" falls in) plus the remaining cycles of the month, up to 4 batches total**. For example, running the simulation on day 12 (Kỳ 2) projects Kỳ 2, 3, 4, and Kỳ 1 of next month. The admin can override the projected-batch count (1–6) in the dialog.

Each projected cycle reruns `ExportService.Export` selection logic against that cycle's day-range, on the assumption that prior cycles' items were marked `paid` (the normal production flow). This is a **read-only model**: no timesheet status is actually changed; the planner simply subtracts already-included timesheet IDs from later cycles' eligibility pools.

## Root-cause findings from reconnaissance (context for implementers)

These pre-existing risks in the export path are **flagged, not fixed** by this plan (per user decision: sim-only validators, gap warnings). Each is surfaced in the simulation UI as a "production does not check this" warning where relevant.

| # | Finding | File:Line | Impact on simulation |
|---|---------|-----------|----------------------|
| R1 | `ExportService.Export` has **no row/batch size limit** | `export_service.go:62` | Projection can be large; pagination is admin-set |
| R2 | `GROUP_CONCAT` ordering is non-deterministic in some repos | (advance_payment repo) | Not in payroll path; N/A here |
| R3 | Validation only checks bank account number/name presence — **no bank code, currency, precision, zero/negative, or duplicate checks** | `excel/service.go:89-145` | Sim adds these as **new sim-only validators** with warnings |
| R4 | Money is `int64` VND everywhere in this path — **no decimal/rounding drift risk** | `bulk_transfer_file.go`, `timesheet.go` | Good — reconciliation is exact integer math |
| R5 | No concurrency/optimistic-lock check on timesheet between sim and export | `export_service.go` | Sim stamps a `snapshot_epoch` (max `updated_at`); real export rejects stale sim if `updated_at > snapshot_epoch` |
| R6 | `ExportService` mixes read planning with side effects (file/code creation, audit) in one method | `export_service.go:62-160` | **Phase 1 refactors this** so sim and prod share one planner |

## Architecture (high level)

```
┌─────────────────────────────────────────────────────────────────────┐
│  /admin/ledger  (React)                                             │
│    TransactionPageHeader / LedgerPageHeader                         │
│      └─ [Mô phỏng đối soát] button                                  │
│           └─ SettlementSimulationDialog                             │
│                ├─ cycle count selector (default 4)                  │
│                ├─ POST /api/v1/payrolls/simulate-settlement         │
│                └─ renders: verdict banner, per-cycle table,         │
│                            included/excluded/remaining drill-downs, │
│                            reconciliation, warnings                  │
└────────────────────────────────────┬────────────────────────────────┘
                                     │  (read-only HTTP)
┌────────────────────────────────────▼────────────────────────────────┐
│  Backend (Go)                                                       │
│  POST /api/v1/payrolls/simulate-settlement   (admin-only)           │
│       └─ SimulationHandler.SimulateSettlement                       │
│            └─ SimulationService.Simulate(ctx, req)                  │
│                 │                                                   │
│                 │  for each projected cycle k = 1..N:               │
│                 │    1. compute cycle date range (Kỳ 1–4 rules)     │
│                 │    2. call ExportPlanner.Plan(ctx, reqForCycle)   │
│                 │       ↳ **same code path as production Export**   │
│                 │    3. subtract IDs already included in cycles<k   │
│                 │    4. run extra sim-only validators               │
│                 │    5. record included/excluded/remaining          │
│                 │                                                   │
│                 │  reconcile: sum(included) vs ledger receivable    │
│                 │  compute verdict: AN_TOAN / CAN_KIEM_TRA /        │
│                 │                          KHONG_THE_TAT_TOAN       │
│                 │  stamp snapshot_epoch (max updated_at seen)       │
│                 │                                                   │
│                 │  NO writes. NO status changes. NO file/code rows. │
│                 ▼                                                   │
│            SimulationResult (JSON)                                  │
└─────────────────────────────────────────────────────────────────────┘
```

The refactor in Phase 1 is the linchpin: it splits `ExportService.Export()` into **`Plan()` (pure read+validate)** and **`Persist()` (write file, create codes, emit audit, gen Excel)**. Production `Export()` becomes `Plan() → Persist()`. The simulation calls only `Plan()`.

## Phases

| Phase | Name | Status | Effort | Summary |
|-------|------|--------|--------|---------|
| 1 | [Refactor: Extract Reusable Export Planner](./phase-01-refactor-extract-reusable-export-planner.md) | Pending | M | Split `ExportService.Export` into `Plan`/`Persist` without behavior change |
| 2 | [Backend: Build Read-Only Simulation Service](./phase-02-backend-build-read-only-simulation-service.md) | Pending | L | Cycle projection + extra validators + reconciliation + verdict |
| 3 | [Backend: Add Endpoint & Wire Routes](./phase-03-backend-add-endpoint-wire-routes.md) | Pending | S | Handler, DTO, route, container wiring |
| 4 | [Frontend: Simulation Dialog & Button](./phase-04-frontend-simulation-dialog-button.md) | Pending | M | Button in both headers + dialog mirroring `OnePayFeeReportDialog` |
| 5 | [Tests: Unit + Integration + No-Mutation](./phase-05-tests-unit-integration-no-mutation.md) | Pending | L | Sim/prod parity, no-mutation, stale-snapshot, 0/<4/=4/>4 cycles |
| 6 | [Backend & Frontend Quality Gates](./phase-06-backend-frontend-quality-gates.md) | Pending | S | `make api-test`, lint, type-check, race, coverage |

## Key design decisions (locked)

1. **Reuse, don't reimplement.** The simulation calls the production `ExportPlanner.Plan()` for each cycle. No second selection/aggregation algorithm exists. This is the single most important acceptance criterion — see Phase 5 parity test.
2. **Strictly read-only.** Simulation runs inside a readonly transaction (`db.BeginTx` with `Mode: ReadOnly` enforced at the GORM session level) so even a programming mistake cannot mutate. Belt-and-suspenders on top of "just don't call write methods."
3. **Cycle model = Kỳ 1–4.** "Current + next 3" means: the cycle containing today, then the next cycles in the monthly sequence, wrapping into the next month after Kỳ 4, capped at the admin-selected count (default 4, range 1–6).
4. **Verdict is full-pool, not window-internal.** The verdict measures coverage of the **entire outstanding pool** (all approved timesheets with `payment_status ∈ {pending, failed}`, no date filter) minus what the N projected windows will cover. Past-cycle stragglers that fall outside the N windows are **failures of coverage**, not edge cases.
5. **Remainder classification by money-flow.** Every item not covered by the N exports is classified as `UNPAID_WAGES` (employee still owed), `OP_LOSS` (money already disbursed via advance/wallet but receivable not settled — operational loss), or `STUCK_IN_FLIGHT` (wallet_payment non-terminal — would double-pay). Any `OP_LOSS` remainder forces the `KHONG_THE_TAT_TOAN` verdict. This is the feature's most important business output.
6. **Past-cycle stragglers: report only.** Per user decision, stragglers are surfaced in the `remainders` list; the simulation does NOT auto-add a catch-up batch and does NOT widen windows backward. Admin handles manually.
7. **Sim-only extra validators, gap warnings.** Per user decision, validators that production doesn't run (zero/negative amount, currency, precision, dedup-across-batches, broken refs, concurrent-modification) are added **in the simulation service only**, and for each one not in production the UI shows `"SẢN XUẤT KHÔNG KIỂM TRA ĐIỀU NÀY — cảnh báo mô phỏng"`.
8. **Stale-snapshot guard on real export.** The simulation returns a `snapshot_epoch` (max `timesheets.updated_at` it observed). The real `Export` endpoint optionally accepts `IfMatchSnapshot` and rejects with `409 StaleSimulation` if any relevant row's `updated_at > snapshot_epoch`. This enforces acceptance criterion #6 ("real export cannot silently proceed using stale simulation results") without coupling export to simulation.
9. **`int64` VND everywhere.** No decimal type introduced. Reconciliation is integer equality.
10. **Backward-compatible.** Phases 1–3 must not change any existing endpoint's request/response shape or behavior. Phase 1 is a pure refactor verified by the existing integration suite.

## Dependencies

- **None.** No unfinished plan in `plans/` touches the bulk-transfer export path, settlement, or ledger simulation. Standalone work.

## Non-goals (explicit)

- Out of scope: FlexPay / advance-payment `sao kê` (`/advance-payments/reconciliation/export`). User confirmed payroll-only.
- Out of scope: fixing the production gaps R1–R6. Sim surfaces them; fixes are separate work.
- Out of scope: introducing real batch-size limits in production.
- Out of scope: HTML plan artifact, GitHub issue, AgentWiki publish (no `--html`/`--github`/`--wiki` flags passed).

## Acceptance criteria (from the brief, mapped to phases)

| # | Criterion | Phase |
|---|-----------|-------|
| 1 | Admin can simulate current + next 3 exports from `/admin/ledger` | 3, 4 |
| 2 | Result clearly answers whether exports will settle every eligible outstanding transaction | 2, 4 |
| 3 | Included/excluded/duplicated/invalid/remaining rows are explainable | 2, 4 |
| 4 | Export totals reconcile against ledger using same rules as production | 2 |
| 5 | Simulation makes no persistent changes | 2, 5 |
| 6 | Real export cannot silently proceed using stale simulation results | 1, 3, 5 |
| 7 | Existing ledger/export behavior stays backward-compatible | 1, 5 |
| 8 | All relevant backend and frontend tests pass | 5, 6 |

## Files touched (summary — details per phase)

**Backend — create:**
- `backend/internal/app/services/payroll/bulktransfer/planner.go` — extracted pure `Plan()` (Phase 1)
- `backend/internal/app/services/payroll/bulktransfer/planner_test.go` — extracted-planner regression (Phase 1/5)
- `backend/internal/app/services/payroll/bulktransfer/simulation_service.go` — projection + validators + verdict (Phase 2)
- `backend/internal/app/services/payroll/bulktransfer/simulation_service_test.go` — unit tests (Phase 5)
- `backend/internal/transport/http/handlers/payroll_simulation_handler.go` — HTTP handler (Phase 3)

**Backend — modify:**
- `backend/internal/app/services/payroll/bulktransfer/export_service.go` — split into `Plan`/`Persist` (Phase 1); accept optional `IfMatchSnapshot` (Phase 3)
- `backend/internal/app/services/payroll/bulktransfer/service.go` — register `simulationService` (Phase 3)
- `backend/internal/app/dto/payroll.go` — add simulation DTOs (Phase 3)
- `backend/internal/app/bootstrap/routes_disbursement.go` — register route (Phase 3)
- `backend/internal/app/bootstrap/container*.go` — wire deps (Phase 3)

**Frontend — create:**
- `frontend/src/components/ledger/SettlementSimulationDialog.tsx` (Phase 4)
- `frontend/src/types/api/settlement-simulation.types.ts` (Phase 4)

**Frontend — modify:**
- `frontend/src/config/api.config.ts` — add endpoint (Phase 4)
- `frontend/src/services/api/ledger.service.ts` — add method (Phase 4)
- `frontend/src/hooks/ledger/useLedgerManagement.ts` — add mutation (Phase 4)
- `frontend/src/components/transaction/TransactionPageHeader.tsx` — desktop button (Phase 4)
- `frontend/src/components/ledger/LedgerPageHeader.tsx` — mobile button (Phase 4)
- `frontend/src/pages/admin/TransactionsPage/index.tsx` — wire dialog (Phase 4)
- `frontend/src/pages/mobile/admin/LedgerEntriesPage/index.tsx` — wire dialog (Phase 4)

**Tests — create:**
- `backend/tests/integration/flow_settlement_simulation.go` — no-mutation + parity + stale-snapshot (Phase 5)
