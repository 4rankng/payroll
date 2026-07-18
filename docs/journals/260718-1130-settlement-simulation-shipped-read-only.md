# Settlement simulation shipped read-only — and a fictional money-flow almost shipped with it

**Date**: 2026-07-18 11:30
**Severity**: Medium (design defect caught pre-ship; feature shipped green)
**Component**: `/admin/ledger` settlement simulation / payroll bulk-transfer export path
**Status**: Resolved (v1 shipped); follow-ups tracked below

## What Happened

Shipped the `Mô phỏng đối soát` (settlement simulation) feature: a read-only `POST /api/v1/payrolls/simulate-settlement` (admin-only) that projects the current + next N−1 monthly Kỳ 1–4 payroll cycles through the **exact same `ExportPlanner.Plan()` code path as production** `export-bulk-transfer`. Returns a full-pool coverage verdict — `AN_TOAN_DE_XUAT` / `CAN_KIEM_TRA` — with remainders and ledger reconciliation. Also bolted an opt-in `IfMatchSnapshot` 409 guard onto the real export so a live run can't silently proceed on stale simulation results.

Phase 1 split the monolithic `ExportService.Export()` into `Plan()` (pure read+aggregate+validate) and `Persist()` (write `transaction_codes` + emit audit) — behavior-identical, verified by the existing integration test. 12 new unit tests, 1 new integration flow, all green. Frontend `pnpm lint` + `pnpm build` clean. tester subagent: READY-TO-SHIP; code-reviewer subagent: APPROVE.

## The Brutal Truth

The most valuable moment of the whole build was the user catching a design that **did not exist in the schema**. An earlier draft — post-red-team, supposedly clean — invented a 3-class op-loss classifier traversing `timesheet → transaction_codes → wallet_payments`. The pushback was instant: *"why are we touching wallet route? this is weekly payment timesheet only."* They were right. `wallet_payments` is in the advance-payment / FlexPay money flow (the `entity_id` FK points at `advance_payment_requests.id`, not timesheets). It has nothing to do with the payroll bulk-transfer path. We were one confident-looking diagram away from shipping code that joined tables that have no business join.

What makes this particularly painful is that the red team had already gone over this. They caught the same chain as Critical Finding 2 — but the fix path they pointed at (`transaction_codes.code ↔ wallet_payments.txn_id`) was itself a different money-flow link. The user's domain knowledge was the only thing that killed the classifier outright. The eligible pool is `payment_status IN (pending, failed)` = "not yet paid", so every remainder is unpaid wages **by construction**. An `OP_LOSS` class is structurally impossible here. The verdict collapsed cleanly to two states and a fictional chain fell out of the design.

## Technical Details

- **Parity guarantee**: simulation calls `ExportPlanner.Plan()` per cycle, the same function production `Export()` delegates to after Phase 1. No second selection algorithm exists.
- **Read-only proof**: canonical guarantee is `TestSimulationSourceHasNoWriteCalls` — a grep-assert over the simulation call graph. `sql.TxOptions{ReadOnly:true}` is **not** relied on because the configured MySQL driver silently ignores it. Structural enforcement > flag-based enforcement > runtime-only enforcement.
- **Red team**: 4 parallel reviewers independently surfaced the same 2 critical defects (fictional `bulk_transfer_files` write + non-existent `timesheet → wallet_payment` chain). 11 of 14 findings accepted.
- **Snapshot guard**: `SnapshotEpoch` = `MAX(timesheets.updated_at)` over observed rows; real export 409s if drift.
- **Auth**: explicit Casbin admin policy + in-handler `isAdmin(c)` check (red-team Finding 4).

## What We Tried

- First design: 3-class op-loss classifier. Reverted — schema doesn't support the join.
- Rejected a `DryRun: true` flag in `Export()` (red-team Finding 12) in favor of the `Plan()`/`Persist()` split, because a flag requires every future write call to remember the flag; the split makes the type system enforce read-only.

## Root Cause Analysis

The classifier wasn't a typo or a missing test — it was a **domain-boundary failure**. I theorized an abstract "what if there's an op-loss class" instead of verifying which money flow each table belongs to. The red team fixed the *path* of the fictional chain; only the user's domain call killed the *concept*. Money-flow boundaries matter more than abstract taxonomy.

## Lessons Learned

- **Verify schema joins exist before designing around them.** A money-flow diagram that "looks reasonable" is not evidence the FK exists.
- **Red-team convergence is the signal.** Four reviewers independently hitting the same two criticals is high-confidence. Without it, both ship as "reasonable design."
- **Structural > flag-based enforcement for invariants.** Grep-assert beats `ReadOnly:true`; `Plan()`/`Persist()` split beats a `DryRun` flag.
- **Mid-implementation corrections from domain owners are the highest-value events in a build.** Don't defend the diagram — go re-read the schema.

## Next Steps

- **Wire the mobile `/admin/ledger` button** (LedgerEntriesPage). ~10-line follow-up; desktop already works. Owner: frontend. This cycle.
- **SnapshotEpoch coverage gap**: covers `timesheets` only. Bank-info edits or schedule edits to `employees` / `project_employees` between sim and export will NOT trip the 409. Documented, deferred — widen the snapshot SQL before trusting the guard against those drift modes. Owner: backend.
- **Sim-only validators descoped**: zero/negative amounts, duplicate-across-batch, bank-code checks were promised in the plan but cut for v1. Per-cycle `Findings[]` is currently always empty; verdict still works via remainders + reconciliation alone. Revisit if admins start trusting the empty list.
- **Integration parity test**: live-HTTP "sim IDs == real-export IDs" was descoped to unit-level `Plan()` parity. The live version needs deterministic seed isolation the shared fixture doesn't support. Track separately.
