# Red Team Plan Review — Assumption Destroyer / Scope Auditor

**Plan:** Statement Settlement Simulation for /admin/ledger
**Reviewer posture:** Hostile — every "will work" claim is treated as false until proven against the codebase.
**Method:** All claims verified via `grep`/read against the actual codebase. No assumption accepted on plan text alone.

The plan is riddled with factually wrong codebase claims. The refactor at its core (Phase 1) is mis-described, the money-flow classifier (Phase 2's headline output) follows a domain chain that does not exist, the cycle helper duplicates an existing module with different semantics, the authz claim is provably wrong, and the frontend wiring targets the wrong files. Several findings below are individually plan-blocking.

---

## Finding 1: `saveBulkTransferFile` does NOT save a `bulk_transfer_files` row — the entire Phase 1 split narrative is wrong

- **Severity:** Critical
- **Location:** Phase 1, "Architecture → Current (monolithic)" and "Implementation Steps #4"; `plan.md` "Architecture" diagram; Root-cause R6.
- **Flaw:** The plan repeatedly asserts `saveBulkTransferFile` is the WRITE step that "writes `bulk_transfer_files` row, creates `transaction_codes` batch" (phase-01 lines 40, 65-66, 116, 119, 144-145). Reading the actual function: it only generates a filename string and calls `transactionCodeRepo.CreateBatch`. It never calls `fileRepo.Create` and never persists a `BulkTransferFile`. The plan's `ExportPlan` / `Persist` split is built around a write that does not happen here.
- **Failure scenario:** The implementer moves `saveBulkTransferFile` into `Persist()`, assumes the file row is now saved there, and ships. Production silently stops persisting `bulk_transfer_files` rows that were being created elsewhere (audit_service.go `SaveProcessingHistory` at line 109, called from the result-import flow, not export). The "byte-for-byte unchanged" promise of Phase 1 is unverifiable because the plan models the wrong side effect.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:422-513` — `saveBulkTransferFile` body returns only `(filename, error)`, calls `transactionCodeRepo.CreateBatch` (line 507), no `fileRepo` reference at all in this file:
    ```
    $ grep -rn "fileRepo\." backend/internal/app/services/payroll/bulktransfer/export_service.go
    (no matches)
    ```
  - The only `fileRepo.Create` in the package is in `audit_service.go:109` inside `SaveProcessingHistory`, which is invoked by the result-upload flow, not by `Export`.
  - `payroll/service.go:146` — `PayrollService.ExportBulkTransfer` delegates straight to `bulkTransferService.Export` → `ExportService.Export`; nothing in that chain writes a file row.
- **Suggested fix:** Re-scope Phase 1 around what `Export()` actually does: read+aggregate+validate, `transactionCodeRepo.CreateBatch` (the only write), Excel generate, `publishExportAudit` (event only). Drop the false "writes bulk_transfer_files row" claim. The refactor's correctness argument depends on knowing the true write surface.

---

## Finding 2: The money-flow chain `timesheet → transaction → settlement → wallet_payment` does not exist; `WalletPayment.EntityID` points to `advance_payment_requests.id`, not timesheets

- **Severity:** Critical
- **Location:** Phase 2, "Verdict computation → classifyByMoneyFlow"; "Architecture → Related Code Files" (remainder_classifier.go); Phase 3 DTO `ClassifiedRemainder.money_flow_evidence`.
- **Flaw:** Phase 2's headline business output (OP_LOSS / UNPAID_WAGES / STUCK_IN_FLIGHT classification) is specified as: *"query linked wallet_payment(s) by entity_id chain (timesheet → transaction → settlement → wallet_payment, OR timesheet → advance_payment_request → wallet_payment)"* and the `money_flow_evidence` JSON cites `wallet_payment_id`. Neither chain exists in the codebase:
  - `wallet.WalletPayment` has no `TransactionID`, `SettlementID`, or `TimesheetID` field — only `EntityID *uint64`.
  - The repository comment and JOIN prove `entity_id` references `advance_payment_requests.id`, exclusively:
    `internal/infra/persistence/wallet_payment_repository.go:349-351` — `INNER JOIN advance_payment_requests apr ON wp.entity_id = apr.id`.
  - `tx_wallet_payment_repository.go:384-399` `HasNonTerminalByEntityID` doc: "linked to the given advance_payment_requests.id (entity_id)".
  - There is no `advance_payment_request.timesheet_id` foreign key either — `advance_payment_request.go` grep returns no timesheet linkage. Advance payments are a separate pool from timesheet payroll entirely (confirmed by the plan's own Non-goals: "Out of scope: FlexPay / advance-payment").
- **Failure scenario:** `remainder_classifier.go` is implemented per spec. For a payroll timesheet in `remainderIDs`, every wallet-payment lookup returns nothing (no row has `entity_id = timesheet.id`), so every remainder is mis-classified as `UNPAID_WAGES` and `OP_LOSS` is never produced. The `KHONG_THE_TAT_TOAN` verdict — the feature's most important output (locked decision #5) — can never fire. Phase 5's op-loss integration test (5b) cannot pass against this schema without fabricating a relationship the DB does not have.
- **Evidence:**
  - `backend/internal/domain/wallet/wallet_payment.go:8-29` — struct fields, no transaction/timesheet FK.
  - `backend/internal/infra/persistence/wallet_payment_repository.go:349-351` (JOIN), `:269-271` (`EntityID` filter), `:55` (insert writes `entity_id`).
  - `backend/internal/domain/advance_payment_request.go:32-33` — links to `TransactionID` for settlement tracing, no timesheet linkage.
  - `backend/internal/domain/timesheet.go:60` — `TransactionID *uint` links timesheet → revenue transaction only; there is no path from `Transaction` to `WalletPayment` (`Transaction` has `Settlements`, `LedgerEntries`, no wallet field — `transaction.go:88-92`).
- **Suggested fix:** Either (a) restrict the classifier to the only chain that exists — timesheet → `TransactionID` → `Settlements` / ledger receivable — and drop `OP_LOSS` based on wallet_payment entirely, or (b) defer the OP_LOSS classification to a future phase that first establishes the data model linking payroll disbursements to receivables. The current spec is unimplementable as written.

---

## Finding 3: A Kỳ 1–4 cycle helper already exists at `clock/pay_cycle.go` with DIFFERENT semantics; Phase 2 plans to re-invent it incorrectly

- **Severity:** Critical
- **Location:** Phase 2, "Architecture → Cycle date ranges (the Kỳ 1–4 model)"; Phase 2 Implementation Step #1; Phase 5 unit test plan.
- **Flaw:** The plan creates a brand-new `payroll_cycle.go` (`PayrollCycle`, `CycleContaining`, `Next`) and treats it as the canonical Kỳ 1–4 model. A canonical model already exists: `backend/internal/pkg/clock/pay_cycle.go`. The existing model and the plan disagree on a load-bearing point:
  - Existing `NextTimesheetPayCycle` (`pay_cycle.go:156-187`) explicitly returns the next cycle STRICTLY AFTER today ("days 1–9 → Ky 1, days 10–16 → Ky 2, days 17–23 → Ky 3, days 24–end → Ky 4").
  - Plan's `CycleContaining` returns the cycle CONTAINING today ("For example, running the simulation on day 12 (Kỳ 2) projects Kỳ 2, 3, 4...").
  - For day 12, the existing helper says "Kỳ 2 is currently being paid, project Kỳ 3 onward"; the plan says "Kỳ 2 is upcoming, include Kỳ 2." These are off-by-one and produce different eligibility windows.
- **Failure scenario:** Two divergent Kỳ models ship in the same codebase. Cash-readiness forecasting (uses `clock.NextTimesheetPayCycle`, `cash_readiness_forecast.go:147`) and settlement simulation disagree on what "current Kỳ" means. Admins see one cycle number in the forecast card and a different cycle number in the simulation dialog for the same day. Phase 5's cycle-boundary unit tests will lock in semantics that contradict production forecasting.
- **Evidence:**
  - `backend/internal/pkg/clock/pay_cycle.go:23-33` — `TimesheetKyCount = 4`, `kyWorkStartDay`, `kyPayDayInMonth` constants.
  - `backend/internal/pkg/clock/pay_cycle.go:148-156` — comment: "days 1–9 → Ky 1 (pay 10); days 10–16 → Ky 2..." — i.e. cycle is selected by APPROACHING pay date, not by work-window containment.
  - `backend/internal/app/services/cash_readiness_forecast.go:147` — production code already consumes `clock.NextTimesheetPayCycle`.
  - Plan phase-02 lines 39-64 describe the opposite (containment) model.
- **Suggested fix:** Do not create a new helper. Extend `clock/pay_cycle.go` with a `CycleContaining(t)` (or expose `KyFromWorkDay` plus `PayDate` to compose the cycle), and make Phase 2's projection explicitly call into `clock`. Delete the planned `payroll_cycle.go`. Reconcile the off-by-one with the product owner before writing tests.

---

## Finding 4: The `/payrolls` group is NOT admin-only; `partner` has explicit Casbin allow rules and the new route needs its own policy entry

- **Severity:** High
- **Location:** Phase 3, "Architecture → Route"; plan.md decision #2 ("admin-only endpoint"); Phase 3 Success Criteria "Non-admin token → 403".
- **Flaw:** The plan asserts: *"The `/payrolls` group already sits behind `Auth.Authenticate()` + `Authorization.Authorize()`, so admin gating is inherited. Confirm `Authorize()` defaults to admin for this group during implementation; if it's broader, add an explicit admin check."* Two false assumptions:
  1. `Authorize()` does not "default to admin." When `authorizationService` is present (production path), it calls Casbin `CanAccess(userRole, resource, action)` (`authorization.go:85`). Authorization is granted iff a policy row matches — there is no admin default inside the middleware.
  2. The Casbin policy file already grants `partner` access to specific `/api/v1/payrolls/*` routes (e.g. `/histories`, `/bulk-transfer-template`) and uses `keyMatch2` glob matching against `admin → /api/*, *, allow` and `partner → /api/v1/payrolls/<x>, <verb>, allow`. The NEW route `/api/v1/payrolls/simulate-settlement` has no policy entry for `partner` → denied by default — which is the desired outcome, but for a different reason than the plan states.
  - Real risk: `keyMatch2` is glob-pattern matching. If anyone later adds `p, partner, /api/v1/payrolls/*, *, allow` (mirroring the projects/employees patterns at policy lines 10, 16), the simulate endpoint silently becomes partner-accessible. The plan does not add a defensive explicit `deny` row.
- **Failure scenario:** Partner token today → correctly 403 (no policy row). But the plan ships believing admin-gating is "inherited from the group." A future contributor adds a `/api/v1/payrolls/*` partner allow (consistent with how `/projects/*`, `/employees/*` are written, policy lines 10/16) and the simulation endpoint leaks to partners without anyone reviewing this PR. The "non-admin → 403" test passes today and stops passing tomorrow.
- **Evidence:**
  - `backend/configs/casbin_policy.csv:1` — `p, admin, /api/*, *, allow` (admin via wildcard).
  - `backend/configs/casbin_policy.csv:84-86` — explicit `partner` allows for `/payrolls/histories`, `/payrolls/histories/export`, `/payrolls/bulk-transfer-template`.
  - `backend/configs/casbin_model.conf` matcher: `keyMatch2(r.obj, p.obj)` (glob).
  - `backend/internal/transport/http/middleware/authorization.go:62-70` — only the `authorizationService == nil` fallback is admin-only; the live path delegates to Casbin.
  - `backend/internal/app/services/auth/authorization_service.go:37-72` — `CanAccess` enforces via Casbin; no admin default.
  - `backend/internal/app/bootstrap/routes_disbursement.go:99-117` — `setupPayrollRoutes` mounts on `protected.Group("/payrolls")` with `Authorize()` but no role restriction at the group level.
- **Suggested fix:** Add an explicit `p, partner, /api/v1/payrolls/simulate-settlement, *, deny` row to `casbin_policy.csv` (and the same for `adv_partner`/`employee`). Mirror the explicit deny pattern already used for `partner → /timesheets/bulk-approve` (policy lines 59-61). Do not rely on route-group inheritance that does not exist. Phase 3 success criterion "Non-admin → 403" must be backed by a policy assertion, not an absence-of-policy accident.

---

## Finding 5: Frontend wiring targets the wrong service, wrong hook, and wrong endpoints group

- **Severity:** High
- **Location:** Phase 4, "Endpoint + service + hook"; "Files touched (summary)" in plan.md; Phase 4 file layout.
- **Flaw:** Phase 4 mandates:
  - Add endpoint into `frontend/src/config/api.config.ts` under the `payrolls` group.
  - Add `simulateSettlement()` method to `frontend/src/services/api/ledger.service.ts`.
  - Add a mutation to `frontend/src/hooks/ledger/useLedgerManagement.ts`.
  All three are inconsistent with how the existing `exportBulkTransfer` (the directly analogous payroll endpoint) is wired today:
  - `exportBulkTransfer` is implemented in `frontend/src/services/api/bulk-transfer.service.ts:162-183`, exposed via `frontend/src/hooks/api/usePayrolls.ts:15` (`useExportBulkTransfer`), and consumed from `TimesheetPage` — NOT the ledger page, NOT `ledger.service.ts`, NOT `useLedgerManagement`.
  - The plan even hedges ("note: payroll endpoints live on the PayrollService client if one exists; if not, follow how `exportBulkTransfer` is called today") but then prescribes the wrong files anyway. A separate `payroll.service.ts` does exist (`frontend/src/services/api/payroll.service.ts`), which the plan never mentions.
- **Failure scenario:** Implementer adds a `simulateSettlement` method to `ledger.service.ts` and a mutation to `useLedgerManagement`. The dialog calls into the ledger hook chain, which has different error handling, different auth headers, and different base-path assumptions than the payroll hook chain. Type errors at build, or — worse — the method silently POSTs to a URL the ledger client mangles. Phase 6 `pnpm type-check` fails, or the feature 404s in QA.
- **Evidence:**
  - `frontend/src/config/api.config.ts:155` — `exportBulkTransfer: '/payrolls/export-bulk-transfer'` (in the `payrolls` group).
  - `frontend/src/services/api/bulk-transfer.service.ts:162-183` — `exportBulkTransfer()` defined here, not in `ledger.service.ts`.
  - `frontend/src/hooks/api/usePayrolls.ts:15-17` — `useExportBulkTransfer` hook, with the comment "exportBulkTransfer returns void and downloads file directly."
  - `frontend/src/pages/mobile/admin/TimesheetPage/index.tsx:164,248,396` — actual consumer of `useExportBulkTransfer`.
  - `frontend/src/services/api/payroll.service.ts` exists (separate from both ledger and bulk-transfer services).
- **Suggested fix:** Put the endpoint in `api.config.ts → payrolls` (correct), put the service method in `bulk-transfer.service.ts` or `payroll.service.ts` (matching `exportBulkTransfer`), expose via a new mutation in `hooks/api/usePayrolls.ts`, and have the dialog import that hook. Drop all references to `ledger.service.ts` and `useLedgerManagement.ts` for this feature.

---

## Finding 6: `ExportPlanner.Plan()` cannot return the per-timesheet IDs the simulation needs; aggregation collapses them, and the "subtract already-included" step is built on data Plan() does not surface

- **Severity:** High
- **Location:** Phase 1 `ExportPlan` struct; Phase 2 "Per-cycle planning" (the partition step); Phase 2 full-pool scan.
- **Flaw:** Phase 2's algorithm requires, per cycle: `included`, `excluded`, `remaining` as **per-timesheet** sets, plus `partition(plan, alreadyIncluded)` operating on timesheet IDs across cycles. But the planner's aggregated output `excel.BulkTransferData` keys everything by `EmployeeProjectKey{EmployeeID, ProjectID}` and only preserves timesheet IDs as `map[EmployeeProjectKey][]uint` (`excel/service.go:31`). Consequences:
  - When an employee has timesheets T1 (in cycle K) and T2 (in cycle K+1) under the same `(employee, project)`, both cycles' `Plan()` aggregate to the same key. The simulation cannot tell which timesheet IDs were "already included" in cycle K versus which are new in cycle K+1 at the per-row level — `aggregateTimesheetData` SUMS amounts across all matched timesheets per key.
  - Phase 1's `ExportPlan` exposes `ValidatedData` and `RawAggregated` (both aggregated), with no field carrying the underlying timesheet set.
- **Failure scenario:** Cycle K includes T1+T2 (same employee×project, both approved, both in date range). Cycle K+1's date window covers only T2 — but since statuses don't actually flip in sim, `Plan()` for K+1 also returns T1+T2 aggregated under the same key. The sim's "subtract already-included" operates on a key set, sees the same key in K and K+1, and either drops both (double-excluding T2) or keeps both (double-counting T1). The parity test (Phase 5 #2) is the only thing that catches this — by failing — at which point the implementer discovers the abstraction is wrong.
- **Evidence:**
  - `backend/internal/app/services/payroll/excel/service.go:30-34` — `EmployeeProjectAmounts map[EmployeeProjectKey]int64`, `EmployeeProjectTimesheets map[EmployeeProjectKey][]uint` — no per-row timesheet preservation in `ValidatedData`.
  - `backend/internal/app/services/payroll/excel/service.go:56-59` — `EmployeeProjectKey` = `{EmployeeID, ProjectID}` only.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:357-358` — `employeeProjectAmounts[key] += timesheet.Amount` (aggregation collapses the timesheet axis).
  - Plan phase-01 lines 50-58 — `ExportPlan` carries `ValidatedData` + `RawAggregated` only, no `[]*domain.Timesheet` field.
- **Suggested fix:** Extend `ExportPlan` with `FilteredTimesheets []*domain.Timesheet` (the post-filter, pre-aggregation set) and `IncludedTimesheetIDs []uint` (post-validation). Rewrite Phase 2's partition to operate on timesheet IDs from that field, not on aggregated key sets. Update Phase 5's parity test to compare timesheet-ID sets, which is the only meaningful unit.

---

## Finding 7: The ledger "receivable SUM by date range" query does not exist; only a global `GetBalanceByAccount` (no date filter) and a multi-account cumulative (wrong account set)

- **Severity:** High
- **Location:** Phase 2, "Reconciliation" pseudo-code (`ledgerRepo.GetAccountBalanceByDateRange(ctx, "receivable", fromDate, toDate)`); Phase 2 Risk Assessment ("use the lightest existing query in `ledger_repository_queries.go`").
- **Flaw:** Phase 2 cites a method that does not exist and recommends "Confirm the exact method with `ledger_repository_queries.go` during implementation; if none fits, add one read-only `SUM` query." Reading the file:
  - `ledger_repository_queries.go` has no method returning a SUM of a single account filtered by date range. The closest are: `GetCumulativeTotalsBeforeDate` (SUMs Revenue/Expense/Equity/Loan — NOT Receivable, and only "before date", not a range) and `GetAccountEntriesByDateRange` (returns full rows, not a SUM).
  - `ledger_repository_balance.go:44-55` has `GetBalanceByAccount(ctx, account)` — single account, no date filter.
  - Worse, `GetCumulativeTotalsBeforeDate` returns `float64` (`ledger_repository_queries.go:116-120, 145`), which collides with the plan's "int64 VND everywhere, exact integer equality" promise (decision #9).
- **Failure scenario:** Implementer finds no fitting query, adds a new one (scope creep — Phase 2 said "if none fits, add one"). Or worse, they reuse `GetCumulativeTotalsBeforeDate` thinking it covers receivable, and reconciliation silently compares export total against an `Expense+Revenue+Equity+Loan` sum that excludes receivable entirely. Delta is always nonzero; verdict is always `CAN_KIEM_TRA` for every real dataset.
- **Evidence:**
  - `backend/internal/infra/persistence/ledger_repository_queries.go` (entire file, 181 lines) — no `receivable`-by-date SUM; cumulative query at lines 125-158 covers only `AccountRevenue`, `AccountExpense`, `AccountEquity`, `AccountLoan` (line 136) and returns `float64`.
  - `backend/internal/infra/persistence/ledger_repository_balance.go:44-55` — `GetBalanceByAccount` returns `int64` but takes no date range.
  - `backend/internal/domain/ledger.go:330` — `AccountReceivable = "receivable"`.
- **Suggested fix:** Either scope a new method explicitly into Phase 2's deliverables (`GetAccountBalanceByDateRange(ctx, account, from, to) int64`), or drop the reconciliation to "compare against `GetBalanceByAccount("receivable")` global" and document the loss of date-filtering. Do not leave it as "confirm during implementation."

---

## Finding 8: "Reuse the same `Plan()` for the full-pool scan (no date filter)" hits a validation error path the plan never accounted for

- **Severity:** Medium
- **Location:** Phase 2, "Verdict computation" step 1 (`fullPool := planner.Plan(ctx, reqWithNoDateFilter)`); plan.md decision #4.
- **Flaw:** The full-pool verdict requires calling `planner.Plan()` with no date scope. But `ExportBulkTransferRequest` requires either `ForMonth` (monthly) OR `FromDate`+`ToDate` (weekly); both empty makes `isMonthly := false` and then `ResolveWeeklyRange` returns `domain.NewValidationError(MsgFromDateRequiredForWeeklyVN)` (`period_calculator.go:32-34`). `Plan()` aborts before returning any plan. There is no "no date filter" branch in the existing code.
- **Failure scenario:** Implementer calls `Plan()` with an empty request per the spec. Validation fails immediately. The full-pool scan — the foundation of the locked verdict semantics (decision #4) — never runs. Every verdict collapses to window-internal logic that the plan explicitly rejected.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/period_calculator.go:31-52` — `ResolveWeeklyRange` returns `MsgFromDateRequiredForWeeklyVN` when `FromDate`/`ToDate` empty.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:62-88` — `isMonthly := strings.TrimSpace(req.ForMonth) != ""` is false when both `ForMonth` and date strings are empty; control enters the `else` branch and calls `ResolveWeeklyRange`, which errors.
- **Suggested fix:** Either (a) add an explicit "no-date" mode to `Plan()` that skips both `ResolveWeeklyRange` and `ResolveMonthlyRange` (and the `filters.FromDate/ToDate` assignment at lines 102-105), with a new request discriminant, or (b) call the underlying `listTimesheetsForCycle` directly with `isMonthly=false`, `FromDate=nil`, `ToDate=nil`. Whichever path, the plan must state that the planner requires a non-trivial extension to support the no-filter case; it is not a free "reuse."

---

## Finding 9: "int64 VND everywhere — no decimal drift" is false at the aggregation boundary

- **Severity:** Medium
- **Location:** plan.md decision #9 and R4; Phase 2 "Reconciliation" (`Reconciled: delta == 0 // int64 — exact equality`); Phase 5 parity test asserts `simTotal == exportedTotal // int64 equality`.
- **Flaw:** Money IS stored as int64 in the leaf models (`timesheet.Amount`, `BulkTransferFile.TransferAmount`, `BulkTransferData.EmployeeProjectAmounts`). But the aggregation pipeline through which both sim and export go multiplies by a percentage using float arithmetic and truncates back to int64:
  - `excel/service.go:277` — `paymentAmount := int64(float64(totalAmount) * bulkTransferPaymentPercentage)`.
  - `bulktransfer/payments.go:21` — `paidAmount := int64(float64(ts.Amount) * bulkTransferPaymentPercentage)`.
  - `timesheet.go:339-340, 399-400` — `CalculateAmount`, `CalculateRevenueReceivable` both use `int64(float64(...) * ...)`.
  - Ledger `GetCumulativeTotalsBeforeDate` returns `float64` (`ledger_repository_queries.go:117-145`).
- **Failure scenario:** Two export runs over the same input produce the same `int64` total, so the parity test passes — and the plan uses that to "prove" no drift. But reconciliation against the ledger can show `delta != 0` purely due to float→int truncation across many transactions, not because anything is actually wrong. The locked verdict semantics treat that as `CAN_KIEM_TRA`, surfacing a false warning to the admin on every clean run.
- **Evidence:**
  - `backend/internal/app/services/payroll/excel/service.go:277` (float multiplication in the validated-data path).
  - `backend/internal/app/services/payroll/bulktransfer/payments.go:21`, `payment_service.go:155-156` (float→int truncation in paid-amount computation).
  - `backend/internal/domain/timesheet.go:340, 400` (float math in amount/receivable calculation).
  - `backend/internal/infra/persistence/ledger_repository_queries.go:117-120, 145` (`float64` treasury totals).
- **Suggested fix:** Either narrow decision #9 to "money is stored int64 but computed via float percentage truncation; reconciliation must tolerate truncation drift ≤ N × max-percentage-rounding" and add a tolerance to the `delta == 0` check, or document the rounding as an accepted false-positive source in Phase 6's "edge cases not automatically verifiable." Do not claim "exact integer equality."

---

## Finding 10: `parseBulkTransferExcel` helper does not exist in `flow_manual_bulk_transfer.go`; the parity test's keystone helper is missing

- **Severity:** Medium
- **Location:** Phase 5, "The parity test" pseudocode line `exportedIDs, exportedTotal := parseBulkTransferExcel(exportResp.body) // reuse helpers from flow_manual_bulk_transfer.go`.
- **Flaw:** The plan asserts a reusable Excel-parsing helper already exists in `flow_manual_bulk_transfer.go`. It does not. The entire file is 39 lines and contains only one test that calls `adminClient.DownloadPost(...)` and checks the status code — it discards the body, never parses it, has no helper. There is no `parseBulkTransferExcel` symbol anywhere in the integration tests.
- **Failure scenario:** Phase 5 implementation arrives at the parity test, grep's for the helper, finds nothing, and either (a) writes an Excel parser from scratch (scope creep into Phase 5, unestimated) or (b) skips the parity assertion (the most important acceptance criterion — "simulation doesn't drift" — goes unproven).
- **Evidence:**
  - `backend/tests/integration/flow_manual_bulk_transfer.go` — 39 lines total; only `DownloadPost` and `statusCode` check; no body parsing, no helper exported.
  - `grep -rn "parseBulkTransferExcel" backend/tests/integration/` — zero matches.
- **Suggested fix:** Move the helper creation into Phase 5 as an explicit deliverable (e.g. `bulk_transfer_parse_helpers_test.go`), or pull the parsing logic out of `excel/service.go`'s writer into a shared reader/writer pair the test can import. Update Phase 5 effort estimate accordingly.

---

## Cross-cutting takeaway

The plan reads as if written from plausible-sounding domain assumptions without grep-verification. Six of the seven "locked" decisions depend on at least one false codebase claim. Phases 1, 2, and 4 each contain at least one Critical/High finding that blocks implementation as specified. Recommend the planner re-run the assumption list in the "Focus on these assumption questions" brief against the evidence above and revise before any phase is opened for implementation.
