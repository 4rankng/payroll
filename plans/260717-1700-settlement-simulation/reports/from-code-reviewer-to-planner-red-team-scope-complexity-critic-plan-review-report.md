# Red-Team Review: Settlement Simulation Plan

**Reviewer role:** Scope & Complexity Critic (YAGNI enforcer) + Contract Verifier
**Method:** All claims verified against `/Users/dev/Documents/projects/payroll/backend` and `frontend` via grep/read. Codebase citations are real, not assumed from plan text.
**Verdict:** This plan is over-scoped for an MVP dry-run preview. It is also built on an unverifiable money-flow data model that does not exist in the schema. At least three of its central design decisions will not survive contact with the actual code.

---

## Finding 1: Phase 1 Plan/Persist refactor is unnecessary — a `Preview*` dry-run pattern already exists in the codebase

- **Severity:** Critical
- **Location:** Phase 1 (`phase-01-...md`, entire); `plan.md` lines 56, 97, 110–121; decision #1
- **Flaw:** Over-engineering / premature abstraction. The plan claims "Without this refactor, any simulation would necessarily duplicate production logic and drift" (phase-01:21) and frames the refactor as "the linchpin" (plan.md:97). This is false. The codebase already has a `PreviewTimesheets` method that does dry-run validation without splitting its host service into Plan/Persist.
- **Failure scenario:** Phase 1 is sized as "M" effort and is the single biggest blast-radius change in the plan — it rewrites a 513-line production file (`export_service.go` is 513 lines, **not** the "62–160" claimed in plan.md:55 — the citation is wrong), moves `listTimesheetsForCycle`/`filterTimesheetsByRequest`/`aggregateTimesheetData` across files, and adds a new `ExportPlanner` type with its own constructor wired into `NewExportService`. The plan itself lists four behavior-drift risks (phase-01:134–145) and admits "Risk: behavior drift during cut-paste." All of this risk is incurred to enable a simulation that could instead pass a `DryRun bool` into `Export()` and early-return after `ValidateAndFilterBulkTransferData` (export_service.go:162) — exactly the boundary the planner would carve out anyway. The refactor delays the actual feature (Phases 2–4) by an entire phase for zero user-visible value.
- **Evidence:**
  - Existing precedent: `backend/internal/domain/services/timesheet_domain_service_bulk_preview.go:21` — `PreviewTimesheetResult { Success bool; Errors []PreviewError }`, `func (s *TimesheetDomainService) PreviewTimesheets(...) (*PreviewTimesheetResult, error)` — a dry-run validation method on the same domain service that owns the write path. No Plan/Persist split was needed there.
  - Natural insertion point already exists: `backend/internal/app/services/payroll/bulktransfer/export_service.go:162` — `validationResult := es.excelService.ValidateAndFilterBulkTransferData(aggregatedData)` is the exact line where Plan logic ends and Persist (`saveBulkTransferFile:165`, `GenerateBulkTransferExcel:171`, `publishExportAudit:184`) begins. A `if req.DryRun { return buildPreview(validationResult, aggregatedData), nil }` between lines 162 and 164 achieves the same isolation without moving a single function.
  - Wrong citation in plan: `plan.md:55` claims refactor spans `export_service.go:62-160`. Actual method body is `62–187` (return at 187), and the file is 513 lines total (`wc -l`).
- **Suggested fix:** Cut Phase 1 entirely. Add `DryRun bool` (or a `Mode: "preview" | "export"` enum) to `dto.ExportBulkTransferRequest`. Insert one early-return in `Export()` after line 162. Ship Phase 2 directly against that. If a clean `Plan`/`Persist` split is genuinely wanted later, do it as a follow-up refactor PR with its own justification — not as a prerequisite masquerading as a feature.

---

## Finding 2: The op-loss money-flow classifier has no schema to walk — `wallet_payments` does not reference timesheets

- **Severity:** Critical
- **Location:** Phase 2 (`phase-02-...md` lines 26–28, 145–155, 207, 230); Phase 3 DTO `money_flow_evidence` (phase-03:170–177, 223); decision #5 (plan.md:116)
- **Flaw:** The plan's central business output — `OP_LOSS` / `STUCK_IN_FLIGHT` classification of every remainder timesheet — is built on a fictional entity chain: "timesheet → transaction → settlement → wallet_payment" (phase-02:150) and "timesheet → advance_payment_request → wallet_payment" (phase-02:152). Neither chain exists in the schema.
- **Failure scenario:** The classifier is "the single most important business output of the feature" (phase-02:28). Implementation will discover, mid-Phase-2, that there is no joinable path from a `timesheet.id` to a `wallet_payment`. Every `OP_LOSS`/`STUCK_IN_FLIGHT` row will return `nil` evidence, the verdict will collapse to `UNPAID_WAGES` for everything, and decision #5 ("any OP_LOSS → KHONG_THE_TAT_TOAN") becomes unreachable. The Phase 5 op-loss integration test (phase-05:74–80) cannot be seeded without first adding a schema migration the plan does not propose.
- **Evidence:**
  - `backend/internal/domain/wallet/wallet_payment.go:9-28` — `WalletPayment` has only `EntityID *uint64` as a nullable FK. No `TimesheetID`, no `EntityType` discriminator, no settlement FK.
  - `backend/internal/app/services/disbursement/wallet_payment_service.go:106` — `EntityID: &p.AdvanceRequestID` confirms `EntityID` points at **advance_payment_requests.id**, not timesheets.
  - `backend/internal/app/services/disbursement/wallet_payment_service.go:535` — `s.advancePaymentReqs.GetByID(ctx, uint64(*row.EntityID))` confirms the only resolution path is advance-request → employee, not timesheet.
  - `backend/internal/domain/advance_payment.go:22-39` — `AdvancePayment` has `ProjectID`, `EmployeeID`, `MaxAdvAmount`, `Salary`. **No `TimesheetID` field.** No way to map an advance to a specific timesheet.
  - The "timesheet → transaction → settlement" leg is also ungrounded: `grep` for `TimesheetID` in `domain/settlement*.go` returns nothing tying settlements to individual timesheets.
- **Suggested fix:** Defer the entire op-loss/stuck-in-flight classifier to v2. For v1, ship a single binary verdict: covered vs. uncovered. Call the uncovered bucket `UNPAID_WAGES` and stop. Add a single TODO citing the missing schema link (advance_payments has no timesheet FK; wallet_payments has only entity_id→advance_request). If op-loss detection is genuinely required for v1, the plan must first propose the migration that adds the missing FK — that is a separate, larger piece of work.

---

## Finding 3: The `IfMatchSnapshot` 409 guard is scope creep on a production endpoint the user never asked to touch

- **Severity:** High
- **Location:** Phase 3 (`phase-03-...md` lines 14, 56–77, 189–203, 225, 231, 236); `plan.md` decision #8 (line 119); acceptance criterion #6
- **Flaw:** Scope creep / gold-plating. The user asked for a read-only preview. The plan adds an optional `if_match_snapshot` field to the **production** `ExportBulkTransferRequest`, a new `domain.ErrStaleSimulation` sentinel, a new 409 response shape, and a new handler branch — none of which the user requested. Worse, the plan's own frontend (Phase 4) declines to wire it: "v1 does not auto-wire `if_match_snapshot` into the existing export button (out of scope)" (phase-04:205). So the guard ships with zero callers.
- **Failure scenario:** The guard modifies a production endpoint's contract for a feature no client uses. It introduces a new `SnapshotEpoch` computation that must walk `employee`/`project`/`assignment` `updated_at` (phase-01:94, 99–102) — extra queries on every export, plus a correctness burden (the plan itself flags "Risk: SnapshotEpoch correctness" at phase-01:141). All of this to enforce a property ("real export cannot silently proceed using stale simulation results", plan.md:143) that cannot fire in v1 because nothing sends the header. It is a stored procedure with no caller.
- **Evidence:**
  - Production endpoint touched: `backend/internal/app/bootstrap/routes_disbursement.go:103` — `payrolls.POST("/export-bulk-transfer", container.Handlers.Payroll.ExportBulkTransfer)`. This is live production code.
  - No 409 mapping precedent in actual handler code: `grep -rn "StatusConflict" backend/internal/transport/http/handlers/ backend/internal/pkg/response/` returns only Swagger `// @Failure 409` doc comments (settings.go:49, user.go:35/89/281, bank.go:54/144, employee_profile.go:73). No runtime `response.Error(c, http.StatusConflict, ...)` call exists. The plan's claim "mirror how other domain errors are mapped in `response/`" (phase-03:265) has no mirror to copy.
  - Self-admitted dead code: phase-04:205 explicitly states the frontend will not pass the field in v1.
- **Suggested fix:** Cut `IfMatchSnapshot`, `ErrStaleSimulation`, the 409 branch, and the `SnapshotEpoch` computation from this plan entirely. Acceptance criterion #6 ("real export cannot silently proceed using stale simulation results") is a v2 concern and should be re-scoped to a follow-up that lands the frontend wiring in the same PR as the backend guard. If you keep the snapshot concept at all in v1, return it in the sim response as display-only ("data frozen at T") — do not let it touch production export.

---

## Finding 4: Configurable cycle count (1–6) is YAGNI — the user said "current + next 3" = 4

- **Severity:** Medium
- **Location:** Phase 2 (`phase-02-...md` lines 32, 64); Phase 3 request DTO (phase-03:91, 98); Phase 4 dialog control (phase-04:129); Phase 5 cycle-variants test (phase-05:65–67, success criteria line 178)
- **Flaw:** Unnecessary configurability / gold-plating. The user's stated requirement is "current + next 3 exports" (plan.md:19, 40). The plan adds a `projected_cycle_count` parameter, clamping logic (1–6), a UI `<Select>` with six options, four integration-test variants (0, 1, 3, 4, 6, 7→clamp), and a 400-error path for zero.
- **Failure scenario:** Every configurable parameter doubles the test matrix and invites support questions ("why did I get 6 cycles when I asked for 4 last time?"). For an admin preview that exists to answer one question — "will my standard 4-cycle month settle everything?" — the knob adds UI surface and validation logic for a use case nobody requested. The plan even admits the default is 4 everywhere; the range exists purely as speculation.
- **Evidence:**
  - User wording in plan itself: plan.md:19 — "**current** payroll bulk-transfer export plus the **next 3** projected exports"; plan.md:40 — "The admin can override the projected-batch count (1–6) in the dialog" is presented as an addition, not a requirement.
  - Test surface inflated by the knob: phase-05:65 — `projected_cycle_count ∈ {0, 1, 3, 4, 6, 7}` is six integration cases solely to defend a parameter nobody asked for.
  - No existing precedent for cycle configuration: `grep "PayrollCycle\|payCycle\|Kỳ" backend/internal/app/services/payroll/` returns zero hits — the entire 4-cycle model is invented by this plan, so "1–6" is inventing flexibility on top of invention.
- **Suggested fix:** Hardcode `ProjectedCycleCount = 4` in the service. Drop the DTO field, the `<Select>`, the clamp logic, and the cycle-variants integration test. Keep one test that asserts exactly 4 cycles are returned. If a future user genuinely asks for "next 5," ship it then.

---

## Finding 5: Five frontend sub-components for one dialog has zero precedent in this codebase

- **Severity:** Medium
- **Location:** Phase 4 (`phase-04-...md` lines 28–35, 124–144, 163, 175); file layout
- **Flaw:** Over-decomposition. The plan splits `SettlementSimulationDialog` into `SimulationVerdictBanner`, `SimulationCycleTable`, `SimulationRowDrilldown`, `SimulationRemainders`, `SimulationFindings` — five new files in a `settlement-simulation/` subfolder. The two dialogs the plan explicitly cites as patterns do not decompose this way at all.
- **Failure scenario:** Five tiny prop-drilled components for what is a single results page creates import churn, five files to navigate for any change, and no reuse (these components are not used anywhere else). The plan's own implementation step 4 admits the build order is "bottom-up ... each is pure, fed by props" (phase-04:175) — meaning the parent dialog becomes a prop-forwarding shell. This is the textbook "extracted too early" smell.
- **Evidence:**
  - `frontend/src/components/ledger/OnePayFeeReportDialog.tsx` — **112 lines total, zero local sub-components**. Imports only shared shadcn primitives (`Dialog`, `Alert`, `Button`) plus `FileDropZone` (a cross-feature shared component, not a private child). Verified by `grep "^import"`.
  - `frontend/src/components/ledger/DoubleEntryModal.tsx` — **489 lines, zero local sub-components**. `grep "from './" ` returns nothing — no private children.
  - Plan's own reference list (phase-04:168) names exactly these two dialogs as "Reference patterns." Neither validates the proposed decomposition.
- **Suggested fix:** Ship `SettlementSimulationDialog.tsx` as a single file. Inline the five sections as JSX blocks within one component, exactly as `OnePayFeeReportDialog` and `DoubleEntryModal` do. Extract a sub-component only if a section is reused or exceeds ~150 lines on its own. Target: one file, ~300–400 lines.

---

## Finding 6: The full-pool verdict requires a second `Plan()` call with no date filter — a hidden second selection algorithm

- **Severity:** High
- **Location:** Phase 2 (`phase-02-...md` lines 17–18, 130–148, 226–229 "Full-pool scan runs"); decision #4 (plan.md:115); Phase 5 test 5d (phase-05:88–93)
- **Flaw:** Self-contradiction with the plan's primary invariant. Decision #1 (plan.md:112) is "Reuse, don't reimplement ... No second selection/aggregation algorithm exists." Decision #4 then requires `planner.Plan(ctx, reqWithNoDateFilter)` (phase-02:137) — a second invocation that selects a **different population** (no date filter, status-only) than any of the N cycle-windowed calls. This is a second selection algorithm in everything but name.
- **Failure scenario:** The full-pool call double-counts query load (N+1 `Plan` invocations: 4 for cycles + 1 for the pool), and its result diverges from the union of cycle results precisely in the case the feature is built to detect (stragglers). The plan then re-subtracts (`fullPoolIDs - coveredIDs`, phase-02:144) to recover what should have been the input. This is logically equivalent to "select everything, then carve out windows" — a different algorithm than "select windows, then check what's missing." The two can disagree on items that are eligible in the full pool but filtered out of every window by the `ForcePayroll` union logic (export_service.go:120–137). The parity test (phase-05:101) only proves sim==export **per cycle**; it does not prove the full-pool call agrees with the union of cycle calls.
- **Evidence:**
  - Contradicting decisions in the same plan: `plan.md:112` ("No second selection/aggregation algorithm exists") vs `plan.md:115` ("the verdict measures coverage of the **entire outstanding pool** ... no date filter") vs `phase-02:137` (`fullPool := planner.Plan(ctx, reqWithNoDateFilter)`).
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:120-137` — the forced-payroll union is applied inside `Export()`, not inside the repo filter. A `reqWithNoDateFilter` invocation will apply the same union, but over a different base set, so the subtraction at phase-02:144 is not guaranteed to be a clean set difference.
- **Suggested fix:** Drop the full-pool call for v1. Define the verdict **window-internally**: "do these N projected exports cover all items eligible within their own combined date range?" This is a single algorithm (the cycle loop) and is provably drift-free by the parity test. The "past-cycle straggler" case is real but is already what the cycle loop surfaces as `remaining` in cycle 1; report that. If full-pool verdict is genuinely required, accept that decision #1 must be weakened and add an explicit parity test for `union(cycles) == fullPool` on the same data.

---

## Finding 7: `STUCK_IN_FLIGHT` is a third classification the user never mentioned

- **Severity:** Medium
- **Location:** Phase 2 (`phase-02-...md` lines 26, 152–153, 173–175); Phase 3 DTO (phase-03:155, 175–177, 223); Phase 5 test (phase-05:74–80, success criteria 173); decision #5 (plan.md:116)
- **Flaw:** Scope creep. The user's framing (plan.md:25, 116) is binary: "money paid = op loss" vs. "money owed = unpaid wages." The plan adds a third bucket, `STUCK_IN_FLIGHT`, for wallet_payments in non-terminal states. This third bucket depends on the same fictional schema link killed in Finding 2.
- **Failure scenario:** Even if the wallet_payment link existed (it does not — see Finding 2), `STUCK_IN_FLIGHT` requires querying per-item wallet_payment status, expanding the N+1 surface of the classifier, adding a third stat card to the UI (phase-04:139), a third field to the DTO summary, and a third integration test case. All for a distinction the user did not request.
- **Evidence:**
  - User's stated framing: `plan.md:25` — "(4) not reconcile against the ledger"; `plan.md:116` — "`OP_LOSS` (money already disbursed via advance/wallet but receivable not settled)". The user's mental model is two states. `STUCK_IN_FLIGHT` is introduced by the plan, not the brief.
  - Inherits Finding 2's schema problem: even the simplified two-bucket classifier is not implementable today; a third bucket compounds the unimplementable surface.
- **Suggested fix:** Cut `STUCK_IN_FLIGHT` entirely. Combined with Finding 2's recommendation, v1 ships with a single uncovered bucket. If/when the wallet_payment link is added, the natural v2 split is `OP_LOSS` vs `UNPAID_WAGES` (two buckets), still not three.

---

## Finding 8: 10+ new DTO types where existing `dto/payroll.go` types would do

- **Severity:** Medium
- **Location:** Phase 3 (`phase-03-...md` lines 79–187, 232); Phase 4 types file (phase-04:80–101); `plan.md:159`
- **Flaw:** Parallel reimplementation of existing DTOs. The plan adds `SimulateSettlementRequest`, `SimulationResult`, `CycleProjection`, `SimFinding`, `Reconciliation`, `VerdictSummary`, `RemaindersPayload`, `ClassifiedRemainder`, plus their TS mirrors. Many fields duplicate existing shapes 1:1.
- **Failure scenario:** `dto/payroll.go` already has 28 types (verified). The new `SimulationRow` shape (phase-04:88 `included: SimulationRow[]`) duplicates the existing `BulkTransferSuccessfulPayment` / `SkippedEmployeeInfo` (`backend/internal/app/dto/payroll.go:34, 42, 184`) field-for-field in employee/project/amount/bank-account. Maintaining two parallel hierarchies guarantees drift the next time someone adds a column to the export.
- **Evidence:**
  - `backend/internal/app/dto/payroll.go` — `wc -l` = 314 lines, 28 `^type` definitions. Existing types already model employee×project×amount×bank-info (lines 34, 42, 184–196).
  - Phase 3 DTO (`phase-03:137`) defines an `included` row as `{employee_id, employee_name, project_id, project_name, amount, timesheet_ids, bank_account_number}` — a strict subset of fields already in `BulkTransferSuccessfulPayment`.
- **Suggested fix:** Reuse existing DTOs for row shapes. Define only the genuinely new types: `SimulationResult` (verdict + cycles + reconciliation + snapshot_epoch), `SimFinding`. Reuse `BulkTransferSuccessfulPayment` for included/remaining rows; reuse `SkippedEmployeeInfo` for excluded rows. Net new types: ~3, not ~10.

---

## Finding 9: `project_ids` / `employee_ids` filters in the sim request are unused by the user's actual workflow

- **Severity:** Medium
- **Location:** Phase 3 (`phase-03-...md` lines 85–86, 93–94); Phase 4 dialog control (phase-04:129 "Optional project/employee filters ... otherwise omit for v1")
- **Flaw:** Speculative generality. The filters exist in the DTO, but Phase 4 itself hedges ("if practical; otherwise omit for v1"). The user's question — "will my next 4 payroll exports settle everything?" — is unconditional on project or employee.
- **Failure scenario:** Adding optional filter parameters that no UI control feeds produces dead branches in the service (the `if len(req.ProjectIDs) > 0` paths in `Export()` already exist at export_service.go:106 and 129 — the sim inherits them for free, but they are exercised by zero callers). They also expand the parity test matrix: sim-vs-export parity must now be proven for `{projects: [], employees: []}`, `{projects: [X], employees: []}`, `{projects: [], employees: [Y]}`, `{projects: [X], employees: [Y]}`.
- **Evidence:**
  - Self-hedge: `phase-04:129` — "Optional project/employee filters (reuse pattern from existing ledger filters if practical; **otherwise omit for v1**)."
  - User's question is unconditional: `plan.md:19, 25` — "will my next 4 payroll exports settle everything?" No project/employee scoping in the brief.
- **Suggested fix:** Drop `project_ids` and `employee_ids` from `SimulateSettlementRequest` for v1. The service simulates all eligible outstanding timesheets, full stop. Re-add when a user asks for per-project drill-down.

---

## Finding 10: The plan mis-cites the file it is refactoring and several code locations — fact-checking failures that propagate

- **Severity:** Medium (process signal, not a single bug)
- **Location:** `plan.md:50` (R1), `plan.md:55` (R6), `plan.md:52` (R3); phase-01:118, phase-01:8; phase-02:128, phase-02:208
- **Flaw:** Verification role failure. Multiple file:line citations in the plan are wrong, and at least one referenced file does not contain the function the plan proposes to reuse.
- **Failure scenario:** An implementer trusting these citations will look in the wrong place, conclude the codebase "doesn't have" what the plan says it has, and either re-invent it or stall. Specific errors:
  - `plan.md:55` says refactor spans `export_service.go:62-160`. Actual: method body is `62–187`, file is 513 lines (`wc -l`).
  - `phase-02:128` instructs the implementer to "Confirm the exact method with `backend/internal/infra/persistence/ledger_repository_queries.go`" and proposes `GetAccountBalanceByDateRange` as a candidate. **No such method exists.** `grep "^func" ledger_repository_queries.go` returns `GetByID, GetByAssetID, GetByTransactionID, GetBySettlementID, GetByAccount, GetByDateRange, GetAccountEntriesByDateRange, GetCumulativeTotalsBeforeDate, SearchLedgerEntries` — none returns a balance SUM. The reconciliation step is hand-waved over a query that must be written from scratch.
  - `phase-02:208` cites `backend/internal/infra/persistence/wallet_payment_repository.go` as the dependency for money-flow classification. As shown in Finding 2, that repo has no timesheet-indexed lookup — the cited dependency cannot answer the question the classifier asks.
  - `phase-03:31` claims the admin gate "mirror `settle_from_notification.go:16`" — that file is not in the verified file list and was not located; the citation is unverified.
- **Evidence:** All citations above were verified by `grep`/`wc -l`/`Read` against the actual tree in this review.
- **Suggested fix:** Re-run the planning reconnaissance pass. Treat every `file:line` citation as a claim to be grep-verified before the plan is approved. Specifically: correct the export_service.go span, add an explicit "new query required" line item for the ledger SUM (do not pretend it is a confirmation step), and remove or re-ground the wallet_payment_repository citation per Finding 2.

---

## Summary of recommended cuts (in priority order)

1. **Cut Phase 1 entirely** — use `DryRun` flag in `Export()` instead. Saves ~M effort, removes the biggest behavior-drift risk, and the codebase already has a `PreviewTimesheets` precedent.
2. **Cut the op-loss/stuck-in-flight classifier** — `wallet_payments` has no timesheet FK. The feature's "most important business output" is unimplementable without a schema migration this plan does not propose. Ship binary covered/uncovered for v1.
3. **Cut `IfMatchSnapshot` / 409 / `ErrStaleSimulation`** — modifies production export contract for zero callers (the frontend admits it won't wire it).
4. **Cut configurable cycle count** — hardcode 4. Removes a DTO field, a UI control, clamp logic, and 6 test variants.
5. **Cut 4 of 5 frontend sub-components** — one `SettlementSimulationDialog.tsx` file, matching the precedent set by `OnePayFeeReportDialog` (0 sub-components) and `DoubleEntryModal` (0 sub-components).
6. **Cut the full-pool verdict call** — window-internal verdict is one algorithm and is provably parity-test-safe; the full-pool call contradicts decision #1.
7. **Cut `project_ids` / `employee_ids` from the request DTO** — no UI feeds them; the user's question is unconditional.
8. **Re-verify every file:line citation** — at least three are materially wrong.

After these cuts the plan is roughly 3 phases (backend service + endpoint + frontend dialog) instead of 6, with the simulation answering the user's actual question: "of the items in my next 4 standard cycles, what's covered and what's left."

