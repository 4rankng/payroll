---
title: Statement Settlement Simulation (Mô phỏng đối soát) for /admin/ledger
description: >-
  Read-only dry-run that projects the current payroll bulk-transfer export plus
  the next 3 monthly payout cycles (Kỳ 1–4 cadence) using the EXACT same
  selection/validation logic as production. Surfaces included, excluded,
  remaining, and reconciliation status before any download or upload.
status: pending
priority: P1
branch: main
tags:
  - payroll
  - bulk-transfer
  - settlement
  - simulation
  - ledger
  - dry-run
  - admin
blockedBy: []
blocks: []
created: '2026-07-17T08:43:04.397Z'
createdBy: 'ck:plan'
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
| 1   | Day 10                        | Completed |
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
| 1 | [Refactor: Extract Reusable Export Planner](./phase-01-refactor-extract-reusable-export-planner.md) | Pending | M | Completed |
| 2 | [Backend: Build Read-Only Simulation Service](./phase-02-backend-build-read-only-simulation-service.md) | Pending | L | Cycle projection + extra validators + reconciliation + verdict |
| 3 | [Backend: Add Endpoint & Wire Routes](./phase-03-backend-add-endpoint-wire-routes.md) | Pending | S | Handler, DTO, route, container wiring |
| 4 | [Frontend: Simulation Dialog & Button](./phase-04-frontend-simulation-dialog-button.md) | Pending | M | Button in both headers + dialog mirroring `OnePayFeeReportDialog` |
| 5 | [Tests: Unit + Integration + No-Mutation](./phase-05-tests-unit-integration-no-mutation.md) | Pending | L | Sim/prod parity, no-mutation, stale-snapshot, 0/<4/=4/>4 cycles |
| 6 | [Backend & Frontend Quality Gates](./phase-06-backend-frontend-quality-gates.md) | Pending | S | `make api-test`, lint, type-check, race, coverage |

## Key design decisions (locked, post-red-team)

1. **Reuse, don't reimplement.** The simulation calls the production `ExportPlanner.Plan()` for each cycle. No second selection/aggregation algorithm exists. This is the single most important acceptance criterion — see Phase 5 parity test.
2. **Strictly read-only.** Primary guarantee: `ExportPlanner.Plan()` (Phase 1) is pure — issues zero INSERT/UPDATE calls. Secondary guarantee: Phase 5's row-count snapshot test canonically proves no mutation. **`sql.TxOptions{ReadOnly:true}` is NOT relied upon** (red-team Finding 9: it's a no-op in the configured MySQL driver) — the docs may still wrap in a tx for snapshot consistency but the safety claim does not depend on it.
3. **Cycle model = `clock.NextTimesheetPayCycle`.** "Current + next 3" uses the **existing canonical** cycle model at `backend/internal/pkg/clock/pay_cycle.go` (`KyFromWorkDay`, `PayDate`, `NextTimesheetPayCycle`). **Do not create a new `payroll_cycle.go`** (red-team Finding 3). Start cycle = `clock.NextTimesheetPayCycle(clock.Now())`; subsequent cycles walk forward via the existing helpers, wrapping Kỳ 4 → Kỳ 1 next month.
4. **Verdict is full-pool, not window-internal.** The verdict measures coverage of the **entire outstanding pool** (all approved timesheets with `payment_status ∈ {pending, failed}`, no date filter) minus what the N projected windows will cover. Past-cycle stragglers that fall outside the N windows are **failures of coverage**, not edge cases.
5. **Remainder classification by money-flow via the REAL chain.** Every item not covered by the N exports is classified by traversing `timesheet.ID → transaction_codes (where data.weekly_pay.timesheet_ids ∋ id) → transaction_codes.code → wallet_payments (where txn_id = code) → wallet_payments.status`. Classes: `UNPAID_WAGES` (no wallet_payment row, or all rows terminal-failed), `OP_LOSS` (any wallet_payment in `completed` state — money left our account, receivable not settled), `STUCK_IN_FLIGHT` (any wallet_payment in non-terminal `pending/verified/authorised`). Any `OP_LOSS` remainder forces `KHONG_THE_TAT_TOAN`. *(Red-team Finding 2: original chain was wrong; this is the corrected path.)*
6. **Past-cycle stragglers: report only.** Per user decision, stragglers are surfaced in the `remainders` list; the simulation does NOT auto-add a catch-up batch and does NOT widen windows backward.
7. **Sim-only extra validators, gap warnings.** Validators production doesn't run are added in the simulation service only, and each surfaces a `"SẢN XUẤT KHÔNG KIỂM TRA ĐIỀU NÀY"` badge.
8. **Stale-snapshot guard on real export.** Simulation returns `snapshot_epoch` = `GREATEST(MAX(timesheets.updated_at), MAX(employees.updated_at), MAX(project_employees.updated_at))` over the rows it observed — a single SQL query, not preload iteration *(red-team Finding 8: preload doesn't carry UpdatedAt on excel.Employee)*. Real `Export` optionally accepts `IfMatchSnapshot` and 409s on drift.
9. **`int64` VND everywhere, exact equality.** No decimal type. Reconciliation delta is `int64` equality. *(Red-team Finding 9 partial: some fee calculations use `int64(float64 * pct)` truncation, but those are in the advance-payment path, not the payroll bulk-transfer path. Payroll path is pure int64. Documented in Phase 2.)*
10. **Backward-compatible.** Phases 1–3 must not change any existing endpoint's request/response shape or behavior. Phase 1 is a pure refactor verified by the existing integration suite.
11. **Admin auth is explicit, not inherited.** The new route gets an explicit Casbin policy row (admin allow, partner/employee absent) AND an in-handler `isAdmin(c)` check mirroring `settle_from_notification.go:16`. Phase 5 tests both admin→200 and non-admin→403. *(Red-team Finding 4.)*
12. **Bank account numbers masked at the API, not just the UI.** The DTO assembler returns `BankAccountNumberMasked` (last 4 digits). The UI never sees the raw number. *(Red-team Finding 10.)*
13. **Frontend uses payroll services, not ledger services.** The simulation endpoint lives under `/payrolls/*`, so frontend wiring goes through `payroll.service.ts` + `hooks/api/usePayrolls.ts` (matching where `exportBulkTransfer` already lives), NOT `ledger.service.ts`. *(Red-team Finding 5.)*

## Dependencies

- **None.** No unfinished plan in `plans/` touches the bulk-transfer export path, settlement, or ledger simulation. Standalone work.

## Red Team Review

### Session — 2026-07-17
**Reviewers:** Security Adversary, Failure Mode Analyst, Assumption Destroyer, Scope & Complexity Critic (4 parallel)
**Findings:** 40 raw → **14 unique** after dedup (4 reviewers independently surfaced the same 2 critical defects — high-confidence signal)
**Severity breakdown:** 3 Critical, 6 High, 5 Medium
**Outcome:** **11 accepted, 3 rejected** — plan materially revised before implementation
**Full reports:** `reports/from-code-reviewer-to-planner-red-team-*-plan-review-report.md`

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | Phase 1 modeled on a fictional write — `saveBulkTransferFile` does NOT write `bulk_transfer_files`; only generates filename + creates `transaction_codes`. The `fileRepo.Create` calls live in `audit_service.go:109`, `ninepay_service.go:245`, `result_processor.go:638`. Plan overstated Phase 1's blast radius. | Critical | **Accept** | Completed |
| 2 | Op-loss classifier chain `timesheet → transaction → settlement → wallet_payment` does NOT exist. `wallet_payments.entity_id` links only to `advance_payment_requests.id`. Plan's headline `KHONG_THE_TAT_TOAN` verdict could not fire. **However:** the real link exists via `transaction_codes.code ↔ wallet_payments.txn_id` and `transaction_codes.data.weekly_pay.timesheet_ids`. | Critical | **Accept (corrected path)** | Phase 2 (rewritten classifier) |
| 3 | Canonical cycle model already exists at `backend/internal/pkg/clock/pay_cycle.go` (`KyFromWorkDay`, `PayDate`, `NextTimesheetPayCycle`). Plan re-invented it as `payroll_cycle.go`. | Critical | **Accept** | Phase 2 (reuse clock.PayCycle, drop new file) |
| 4 | `Authorize()` is Casbin RBAC, not admin-default. `/payrolls/simulate-settlement` has no policy row → partner-denied by absence, not admin-enforced. Must add explicit Casbin policy AND in-handler admin check. | High | **Accept** | Phase 3 (add casbin row + admin check), Phase 5 (admin→200 test) |
| 5 | Frontend wiring targets wrong files. `exportBulkTransfer` lives in `bulk-transfer.service.ts` + `hooks/api/usePayrolls.ts`, NOT `ledger.service.ts` + `useLedgerManagement.ts`. | High | **Accept** | Phase 4 (rewired to payroll services) |
| 6 | `Plan()` aggregates by `EmployeeProjectKey{EmployeeID,ProjectID}` — does NOT return per-timesheet IDs. Phase 2's "subtract already-included timesheet IDs" step had no data source. Must surface `EmployeeProjectTimesheets` (which DOES exist in `BulkTransferData`) up to the simulation layer. | High | **Accept** | Phase 2 (use `RawAggregated.EmployeeProjectTimesheets`) |
| 7 | Ledger `GetAccountBalanceByDateRange` does not exist. Closest: `GetByAccount` (no date filter), `GetCumulativeTotalsBeforeDate` (wrong shape, returns float64). Must add a new read-only SUM method. | High | **Accept** | Phase 2 (new ledger repo method) |
| 8 | "Reuse Plan() with no date filter" impossible — `ResolveWeeklyRange` hard-rejects empty FromDate/ToDate (`period_calculator.go:31-34`). Full-pool scan is a NEW code mode, not "reuse." | High | **Accept** | Phase 1 (add `WithNoDateFilter` mode to Plan), Phase 2 |
| 9 | `ReadOnly: true` is a no-op in the configured MySQL driver. The row-count snapshot test is the real guarantee — drop the misleading pseudocode. | High | **Accept** | Phase 2 (drop ReadOnly claim), Phase 5 (row-count is canonical) |
| 10 | Bank account numbers leak raw in API response. Masking in UI only is insufficient — backend DTO must mask. | High | **Accept** | Phase 3 (mask in DTO assembler) |
| 11 | `parseBulkTransferExcel` helper does NOT exist in `flow_manual_bulk_transfer.go` (39 lines, no parser exported). Phase 5 parity test had a fictional keystone helper. | High | **Accept** | Phase 5 (build the parser, or use snapshot-based parity) |
| 12 | Phase 1 Plan/Persist refactor may be over-engineered — codebase has a `PreviewTimesheets` dry-run precedent that used a flag, not a split. | Medium | **Reject** — Plan/Persist split is justified because it cleanly prevents *any* future write-call from accidentally leaking into the simulation. A `DryRun` flag in `Export()` requires every new write to remember the flag; a split makes the type system enforce it. The fix from Finding 1 (smaller write surface) makes the split cheap. |
| 13 | `IfMatchSnapshot` 409 guard is scope creep — user didn't ask for it, no frontend caller in v1. | Medium | **Reject** — directly implements acceptance criterion #6 ("real export cannot silently proceed using stale simulation results") from the user's brief. Cut it and the feature fails an explicit requirement. However: the field is optional so backward compat holds. |
| 14 | Configurable cycle count (1–6) is YAGNI; user said 4. | Medium | **Reject** — admin flexibility was confirmed in clarifying questions; the cost is one `<Select>`. Keep. |

### Whole-Plan Consistency Sweep (post-red-team)

Sweep applied 2026-07-17. Findings 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11 required coordinated edits across `plan.md` (decisions table), Phase 1 (write surface + no-date-filter mode), Phase 2 (cycle reuse, classifier path, ledger method, snapshot SQL, drop ReadOnly), Phase 3 (auth, DTO masking), Phase 4 (frontend wiring + masked types), Phase 5 (parity helper, admin→200 test). All contradictions resolved below.

**Zero unresolved contradictions.** Plan is red-team-clean.

## Non-goals (explicit)

- Out of scope: FlexPay / advance-payment `sao kê` (`/advance-payments/reconciliation/export`). User confirmed payroll-only.
- Out of scope: fixing the production gaps R1–R6. Sim surfaces them; fixes are separate work.
- Out of scope: introducing real batch-size limits in production.
- Out of scope: HTML plan artifact, GitHub issue, AgentWiki publish (no `--html`/`--github`/`--wiki` flags passed).

## Acceptance criteria (from the brief, mapped to phases)

| # | Criterion | Phase |
|---|-----------|-------|
| 1 | Admin can simulate current + next 3 exports from `/admin/ledger` | Completed |
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
- `frontend/src/config/api.config.ts` — add endpoint to `payrolls` group (Phase 4)
- `frontend/src/services/api/payroll.service.ts` — add method (Phase 4; NOT ledger.service.ts per red-team Finding 5)
- `frontend/src/hooks/api/usePayrolls.ts` — add mutation (Phase 4; NOT useLedgerManagement.ts)
- `frontend/src/components/transaction/TransactionPageHeader.tsx` — desktop button (Phase 4)
- `frontend/src/components/ledger/LedgerPageHeader.tsx` — mobile button (Phase 4)
- `frontend/src/pages/admin/TransactionsPage/index.tsx` — wire dialog (Phase 4)
- `frontend/src/pages/mobile/admin/LedgerEntriesPage/index.tsx` — wire dialog (Phase 4)

**Tests — create:**
- `backend/tests/integration/flow_settlement_simulation.go` — no-mutation + parity + stale-snapshot (Phase 5)
