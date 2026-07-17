# Red-Team Plan Review — Settlement Simulation (Failure-Mode Analyst / Flow Tracer)

Reviewer lens: Murphy's Law. Trace every proposed code path against the actual codebase; find dead ends, partial-failure windows, broken chains, and recovery gaps. Codebase evidence required for every finding.

Scope: `plan.md` + `phase-01..06-*.md`. Codebase verified at `backend/`.

---

## Finding 1: The "Phase 1 = pure cut-paste, byte-for-byte unchanged" premise is FALSE — the current `Export()` does NOT call `fileRepo.Create` at all

- **Severity:** Critical
- **Location:** Phase 1, section "Architecture / After refactor" + "Risk: missed a write side effect buried in a 'read' method"; also plan.md decision #1 ("Reuse, don't reimplement") and R6 ("ExportService mixes read planning with side effects ... saveBulkTransferFile, publishExportAudit").
- **Flaw:** The entire premise of Phase 1 — that today's `ExportService.Export()` writes a `bulk_transfer_files` row via `saveBulkTransferFile` that must move into `Persist()` — is wrong. `saveBulkTransferFile` in the current `export_service.go` only builds a filename and calls `transactionCodeRepo.CreateBatch` (lines 421–513). It does NOT create any `bulk_transfer_files` row. `fileRepo` is held as a field (lines 27, 41, 52) but is NEVER called anywhere in `export_service.go` — grep `fileRepo` in that file returns only the field declaration and constructor.
- **Failure scenario:** The refactor is sold to reviewers as "byte-for-byte unchanged" with the only write being `transaction_codes` creation plus an audit publish. Phase 5's parity test (`flow_manual_bulk_transfer.go`) only checks an HTTP 200 + that a download body exists — it does NOT assert that a `bulk_transfer_files` row was written, because production never writes one from this path. The `bulk_transfer_files` rows in production are actually created by `NinePayBulkTransferService` (`ninepay_service.go:245`) and `AuditTrailService.SaveProcessingHistory` (`audit_service.go:109`) — i.e. on the *auto-bulk-transfer* and *result-upload* paths, NOT on the manual `export-bulk-transfer` path that the simulation mirrors. So the no-mutation test will "pass" trivially for `bulk_transfer_files` (nothing was ever going to be written), giving false confidence that the refactor preserves behavior. Worse, when implementers follow the Phase 1 instruction ("move `saveBulkTransferFile` and `publishExportAudit` into `Persist`"), they will be tempted to also start writing a `bulk_transfer_files` row there "for parity with the audit_service pattern" — silently changing the manual export's footprint under cover of a "pure refactor."
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:421-513` — `saveBulkTransferFile` returns `(filename, error)`, calls only `transactionCodeRepo.CreateBatch`, no `fileRepo.Create`.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:27,41,52` — `fileRepo` is injected but unused; grep "fileRepo" in this file returns only those three lines.
  - `backend/internal/app/services/payroll/bulktransfer/audit_service.go:109` — the *real* `fileRepo.Create` lives in a different service, on a different code path.
  - `backend/internal/app/services/payroll/bulktransfer/ninepay_service.go:245` — another `fileRepo.Create` site, also not on the manual export path.
  - `backend/tests/integration/flow_manual_bulk_transfer.go:21-35` — the regression test the plan leans on only asserts HTTP 200; no DB row assertion.
- **Suggested fix:** Re-scope Phase 1 to what the code actually does: the only side effects in `Export()` today are (a) `transactionCodeRepo.CreateBatch` and (b) `publishExportAudit` (event bus). `bulk_transfer_files` is NOT written. Rewrite Phase 1 to extract a `Plan()` that ends after `ValidateAndFilterBulkTransferData` and a `Persist()` that wraps only `saveBulkTransferFile` (code-generation only) + `publishExportAudit`. Add an explicit integration assertion that the post-refactor manual export still issues the SAME write footprint (number of `transaction_codes` rows, zero `bulk_transfer_files` rows). Drop the false claim of "byte-for-byte unchanged" — the SnapshotEpoch and the optional `IfMatchSnapshot` check (Phase 3) are already behavior changes.

---

## Finding 2: The "money-flow" classifier chain (timesheet → transaction → settlement → wallet_payment) does NOT exist in this codebase — `OP_LOSS` and `STUCK_IN_FLIGHT` will always misclassify as `UNPAID_WAGES`

- **Severity:** Critical
- **Location:** Phase 2, sections "Verdict computation (full-pool)" step 4 and "Remainder classification by money-flow"; plan.md decision #5 ("the most important business output"); Phase 3 DTO `ClassifiedRemainder.money_flow_evidence`.
- **Flaw:** The plan asserts the classifier traverses `timesheet → transaction → settlement → wallet_payment` "OR timesheet → advance_payment_request → wallet_payment." Neither chain is wired in this codebase:
  1. `timesheet.TransactionID` exists (`timesheet.go:60`) and links to a *revenue* transaction, but `wallet_payment.EntityID` links **exclusively** to `advance_payment_requests.id` per the repository contract comment (`internal/domain/transactions/repository.go:54-60`: "wallet_payment row linked to the given advance_payment_requests.id (column entity_id)"). There is NO FK or column path from `timesheets` → `advance_payment_requests`. `AdvancePaymentRequest` has `EmployeeID/ProjectID` but no `TimesheetID` (verified `advance_payment_request.go:20-42` and grep across `internal/domain/advance_payment*.go`).
  2. `Settlement.TransactionID` (`settlement.go:15`) links a settlement to a transaction, but `Transaction` has no link to `wallet_payment` either — `wallet_payment` is in a separate `transactions` subpackage (`domaintx.WalletPayment`) and joins to the human-facing `Transaction` only via `TxnID`/`RequestID` strings, not a FK the classifier can follow.
  3. There is no `entity_id` index or column pointing at timesheet IDs — only at `advance_payment_requests.id`.
- **Failure scenario:** The classifier's most important output — "this remaining timesheet is `OP_LOSS` because money already left via a `wallet_payment` in `completed` state" — cannot be computed. With no joinable path, the classifier will silently fall through to the `else → UNPAID_WAGES` branch for EVERY remainder. The verdict logic (`hasOpLoss = any(...)` will always be false) means `KHONG_THE_TAT_TOAN` can NEVER fire from this path, defeating the entire feature's stated purpose. The admin sees "all unpaid wages, recoverable" when in reality advance payments have already drained the wallet.
- **Evidence:**
  - `backend/internal/domain/transactions/repository.go:54-60` — `HasNonTerminalByEntityID` is documented as scoped to `advance_payment_requests.id`.
  - `backend/internal/domain/transactions/wallet_payment.go:34` — `EntityID *uint64 gorm:"column:entity_id"`; comment in `wallet_payment_service.go:96` confirms "EntityID is optional and links back to advance_payment_requests.id".
  - `backend/internal/domain/advance_payment_request.go:20-42` — no `TimesheetID` field.
  - `backend/internal/domain/timesheet.go:60` — `TransactionID *uint` points at revenue transaction, not wallet_payment.
  - `backend/internal/domain/settlement.go:15` — `TransactionID uint` joins to `Transaction`, which itself has no FK to `wallet_payment` (`internal/domain/transaction.go:68-93`).
  - `backend/internal/app/services/payroll/bulktransfer/` — grep for `wallet_payment`/`WalletPayment` returns only `ninepay_service.go` and `batch_completion_checker.go`, both of which use `ListByBatchID` keyed by the disbursement batch_id, NOT by timesheet. The simulation has no batch_id.
- **Suggested fix:** Before Phase 2 ships, the planner must produce a *concrete SQL/ORM plan* that traverses from a timesheet ID to a `wallet_payment.status`. Options that must be evaluated and picked:
  (a) The link is `timesheet → transaction (revenue_receivable side) → ledger_entries → settlements → ...` — but settlements settle *revenue* receivables, not wallet disbursements, so this does not answer "did money leave the wallet."
  (b) The real link is `employee + date range → advance_payment_request → wallet_payment`. The classifier must therefore be defined at employee+period granularity, NOT at timesheet granularity, and "covered" vs "OP_LOSS" cannot be decided per-timesheet at all.
  (c) Accept the limitation and remove `OP_LOSS`/`STUCK_IN_FLIGHT` from v1, shipping only `UNPAID_WAGES` with a documented "we cannot yet classify money already disbursed" gap — this is the honest scope.
  The current Phase 2 design promises a classification that the data model cannot deliver.

---

## Finding 3: N+1 in `remainder_classifier` plus no batch query exists on `WalletPaymentRepository` for "by entity IDs"

- **Severity:** High
- **Location:** Phase 2, "Verdict computation step 4" + "Related Code Files: `wallet_payment_repository.go`".
- **Flaw:** The plan classifies "each remainder item" by querying "linked wallet_payment(s)" per item. Even ignoring Finding 2 (the chain is broken), `WalletPaymentRepository` exposes no batch lookup. The only entity-keyed method is `HasNonTerminalByEntityID(ctx, entityID uint64)` — singular, boolean, non-terminal only — see `internal/domain/transactions/repository.go:60`. There is no `ListByEntityIDs`, no `GetByEntityID`, no `ListByTimesheetIDs`. The other list methods are keyed by `status+date`, `batch_id`, `provider+date`, or `request_id` — none usable for a remainder list of N timesheets.
- **Failure scenario:** If `remainders` contains e.g. 200 timesheets, the classifier will issue 200 individual lookups (`HasNonTerminalByEntityID` × 200). On a large month-end pool this is hundreds of round-trips inside an admin endpoint that already runs 4–6 `Plan()` calls + a full-pool `Plan()`. Combined with Finding 2 (each lookup returns `false` because the entity_id is not a timesheet id), the result is hundreds of useless queries producing an all-`UNPAID_WAGES` verdict — slow AND wrong.
- **Evidence:**
  - `backend/internal/domain/transactions/repository.go:9-61` — full interface; only singular entity lookup is `HasNonTerminalByEntityID`.
  - `backend/internal/infra/persistence/tx_wallet_payment_repository.go:384-400` — implementation of `HasNonTerminalByEntityID`, single `WHERE entity_id = ?`.
- **Suggested fix:** Add a real batch query method (`ListByEntityIDs(ctx, ids []uint64)`) OR — better — once Finding 2's actual linkage is established, write one set-based query that joins all remainder employee+period pairs to `wallet_payment` in a single SQL statement. Until either lands, the classifier cannot be O(1).

---

## Finding 4: The "full-pool Plan() with no date filter" call cannot use the same code path — `ResolveWeeklyRange` hard-rejects empty dates

- **Severity:** High
- **Location:** Phase 2, "Verdict computation (full-pool)", step 1: `fullPool := planner.Plan(ctx, reqWithNoDateFilter)  // reuse same code path`. Also plan.md decision #4 and Phase 2 success criterion "Full-pool scan runs (one extra Plan() call with no date filter)".
- **Flaw:** "Reuse same code path" is false. `ExportService.Export()` (which becomes `Plan()`) at line 84 unconditionally calls `es.periodCalculator.ResolveWeeklyRange(req)` for the non-monthly branch, and `ResolveWeeklyRange` (`period_calculator.go:31-34`) returns a hard validation error when `FromDate` or `ToDate` is empty. There is no `weeklyWithoutDate` mode, no "skip date resolution" branch, no nullable handling. So `reqWithNoDateFilter` cannot flow through `Plan()` as-is.
- **Failure scenario:** Implementer either (a) hits a 400 on the full-pool call and the verdict computation dies before producing a result, or (b) "fixes" it by widening the date range to a huge window (e.g. `0001-01-01` to `9999-12-31`), which silently coerces the full-pool call into a date-filtered call with bizarre semantics — and the timesheet query `timesheets.date >= ? AND timesheets.date < ?` will include or exclude rows based on `paid_at`/`date` semantics that don't match "all outstanding," or (c) introduces a brand-new code path inside `Plan()` (e.g. `if req.FullPool { ... }`) — which is precisely the "drift risk" the plan claims to avoid.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:83-88` — non-monthly branch unconditionally calls `ResolveWeeklyRange`.
  - `backend/internal/app/services/payroll/bulktransfer/period_calculator.go:31-34` — empty `FromDate`/`ToDate` returns validation error.
  - `backend/internal/infra/persistence/repositories/timesheet_query_repository.go:632-639` — confirms the underlying query treats nil pointers as "no date filter" — but you can never get to it via the current `Plan()` entry path.
- **Suggested fix:** Explicitly define how the full-pool call enters `Plan()`. Two viable options, pick one in the plan:
  - Add a new request mode `Mode: "full_pool"` (or sentinel `ForMonth: "__FULL__"`) and a corresponding branch in `Plan()` that skips `ResolveWeeklyRange`. Document this as a *necessary* new code path, not a reuse.
  - Compute the full pool directly in `SimulationService` via `timesheetRepo.List(ctx, filters)` with the same status filters but nil dates, without going through `Plan()`. But then it bypasses aggregation/validation/forced-union — so the "full pool" set will NOT be comparable to the cycle results, breaking the subtraction logic.

---

## Finding 5: `ReadOnly: true` is a no-op promise — `gorm.io/driver/mysql` does not enforce it, and the plan's "belt-and-suspenders" is unverified

- **Severity:** High
- **Location:** Phase 2, "Read-only enforcement" code block + Risk "MySQL silently ignores ReadOnly: true"; plan.md decision #2.
- **Flaw:** The plan wraps the simulation in `&gorm.Session{&sql.TxOptions{ReadOnly: true}}`. The go-sql-driver/mysql used by this repo (`go.mod:66`: `github.com/go-sql-driver/mysql v1.8.1`) does **not** send `START TRANSACTION READ ONLY` to the server. `sql.TxOptions.ReadOnly` is a hint the driver is free to ignore, and the pure-Go mysql driver ignores it (only ` TRANSACTION READ ONLY` in the postgres lib / certain drivers honors it). Grep across the entire backend returns ZERO existing uses of `ReadOnly` or `sql.TxOptions` — meaning this codebase has never exercised this code path, so the plan is introducing an unverified "safety" mechanism.
- **Failure scenario:** A future contributor adds a write call inside `Plan()` (e.g. a `gorm.AutoMigrate` shim, a `Clauses(clause.Locking{Strength: "UPDATE"})` that escalates, or a GORM callback that writes audit rows on `Find`). The "read-only transaction" silently allows the write because the driver never enforced it. The Phase 5 row-count snapshot test catches it only AFTER the damage is done in some other environment, not at the call site.
- **Evidence:**
  - `backend/go.mod:66` — `go-sql-driver/mysql v1.8.1` (pure-Go driver, no `START TRANSACTION READ ONLY`).
  - Grep `ReadOnly|TxOptions` across `backend/` returns no matches — no prior use of this feature anywhere in the codebase.
  - `backend/internal/infra/persistence/` — every existing transaction uses `db.Transaction(func(tx *gorm.DB) error {...})` without `ReadOnly`.
- **Suggested fix:** Drop the `ReadOnly: true` claim from the plan — it is misleading. The real guarantees are: (a) Phase 1 ensures `Plan()` makes no write calls (verified by grep), and (b) Phase 5's pre/post row-count snapshot is the load-bearing assertion. State this honestly. Optionally, add a unit test that monkey-patches the DB to assert that any write attempted inside the simulation returns an error — but do NOT claim the driver enforces it.

---

## Finding 6: Stale-snapshot `SnapshotEpoch` does NOT cover the columns that change selection (employee.bank_id, project_employee.PaymentSchedule, project_employee assignment itself)

- **Severity:** High
- **Location:** Phase 1, "SnapshotEpoch computation" + Phase 3 "Stale-snapshot guard on real export" + plan.md decision #8 + R5.
- **Flaw:** The plan computes `SnapshotEpoch = max(timesheets.updated_at)` with a hand-waved "also walk employee/project/assignment rows" — but: (a) the existing query path (`aggregateTimesheetData` at `export_service.go:320-419`) fetches `employeeRepo.GetByID`, `projectRepo.GetByID`, and `projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee` AFTER the timesheet list, and **none** of the preloaded timesheet structs (`timesheet_query_repository.go` preloads `Project`, `Employee`, `CreatedUser`, `ApprovedUser` only — no `Bank`, no `ProjectEmployee`) carry the bank or assignment `UpdatedAt`. The plan even admits this: "If GORM doesn't preload `UpdatedAt` reliably on all those models, the fallback is a single `SELECT MAX(updated_at) FROM timesheets ...` plus equivalent for employees — one round-trip, deterministic" — but then never commits to actually doing that fallback. As written, SnapshotEpoch only reflects `timesheets.updated_at`.
- **Failure scenario:** Admin runs the simulation at T0. Between T0 and the real export, an admin edits an employee's `bank_id` (changing whether they're included in the file — `ValidateAndFilterBulkTransferData` excludes on missing bank account), or unassigns the employee from the project (changing `project_employee.PaymentSchedule`, which `aggregateTimesheetData:352` filters on), or marks the timesheet `force_payroll=1`. None of these flip `timesheets.updated_at` reliably (bank changes update `employees.updated_at`; assignment changes update `project_employees.updated_at`). The real export proceeds with `IfMatchSnapshot` from T0 and the guard passes — but the simulation's verdict is now wrong: a previously-`AN_TOAN` verdict silently masks an excluded employee who now has no bank. This is exactly the bug class decision #8 claims to prevent.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:320-419` — `aggregateTimesheetData` reads employee/project/assignment separately from the timesheet row, no `UpdatedAt` propagation.
  - `backend/internal/infra/persistence/timesheet_repository.go:197-204` and `repositories/timesheet_query_repository.go` Preload list — only `Project/Employee/CreatedUser/ApprovedUser`; no `Employee.Bank`, no `ProjectEmployee`.
  - `backend/internal/app/services/payroll/excel/service.go` (referenced by plan R3) — bank presence is validated AFTER read, so a bank_id change between sim and export changes the result without moving the snapshot.
  - Phase 1 SnapshotEpoch section's own "fallback" paragraph — never committed to as a hard requirement, only suggested.
- **Suggested fix:** Specify a single deterministic query that returns `GREATEST(MAX(timesheets.updated_at), MAX(employees.updated_at), MAX(project_employees.updated_at))` for the exact set of rows the planner touched. Make this the contract for `SnapshotEpoch` (not "iterating preloaded structs"). Add a Phase 5 test that mutates `employee.bank_id` (not timesheet) and asserts the real export still returns 409. Without this, the guard is security theater for the most likely real-world drift.

---

## Finding 7: The reconciliation query method `ledgerRepo.GetAccountBalanceByDateRange("receivable", fromDate, toDate)` does not exist; the only date-scoped methods return raw rows, not a SUM

- **Severity:** High
- **Location:** Phase 2, "Reconciliation" code block: `receivable, err := ledgerRepo.GetAccountBalanceByDateRange(ctx, "receivable", fromDate, toDate)`. Also Phase 2 Risk "ledger SUM query doesn't exist or is expensive" admits this but waves it off.
- **Flaw:** The actual `LedgerEntryRepository` interface (`backend/internal/domain/ledger.go:40-64`) has `GetBalanceByAccount(ctx, account)` (NO date range — returns the all-time net for an account) and `GetByDateRange(ctx, start, end)` (returns rows, not a SUM). There is no `GetAccountBalanceByDateRange`. The plan's "Confirm the exact method ... during implementation; if none fits, add one read-only `SUM` query" punts the actual design to implementation time, but it's load-bearing for the verdict (`deltaNonZero := reconciliation.Delta != 0` drives `CAN_KIEM_TRA`).
- **Failure scenario:** Implementer calls `GetBalanceByAccount("receivable")` because it's the only matching name — this returns the ALL-TIME receivable balance, not the cycle's. The reconciliation `Delta` is then `allTimeReceivable - cycleIncluded`, which is non-zero for essentially every real dataset, forcing `CAN_KIEM_TRA` on virtually every simulation. The verdict loses meaning. Alternatively, the implementer loads all rows via `GetByDateRange` and sums in Go — O(N) memory and a much larger query than the "lightest existing" promise.
- **Evidence:**
  - `backend/internal/domain/ledger.go:40-64` — interface; methods are `GetBalanceByAccount(account)`, `GetByDateRange(start, end)`, `GetTotalByAccountType(accountType, startDate, endDate)`. None matches the plan's signature.
  - `backend/internal/infra/persistence/ledger_repository_balance.go:44-55` — `GetBalanceByAccount` confirmed: `WHERE account = ?` only, no date filter.
- **Suggested fix:** Decide the reconciliation contract in Phase 2, not at implementation time. Either (a) commit to adding `GetAccountBalanceByDateRange(ctx, account, from, to)` to the interface with a single indexed `SUM`, gated by the existing `(account, date)` index, and list this as a new repo method, or (b) use `GetTotalByAccountType` if its semantics match (verify in `ledger_repository_analytics.go` before committing). Make the verifier choose before Phase 5, because the parity test will be comparing apples to oranges otherwise.

---

## Finding 8: Cycle boundary dedup is performed by subtracting `unionOfTimesheetIDs(prevCycles)`, but a timesheet whose `date` straddles a cycle boundary will appear in BOTH cycle windows — and `Plan()` makes no attempt to dedup within itself

- **Severity:** Medium
- **Location:** Phase 2, "Per-cycle planning" loop + plan.md decision #3 (cycle model Kỳ 1–4) + Risk "cycle subtraction model doesn't match real production behavior."
- **Flaw:** The timesheet query filters on `timesheets.date >= ? AND timesheets.date < ?` (`timesheet_query_repository.go:633-638`) using `< nextDay` semantics. The Kỳ 1–4 windows are defined as days 1–7, 8–14, 15–21, 22–28 (plan.md table). These are non-overlapping at day granularity IF the date column is a DATE — but production export ranges come from `req.FromDate`/`req.ToDate` which are admin-provided strings, and the simulation's `cloneRequestForCycle` constructs them from `PayrollCycle.FromDate`/`ToDate`. The risk is the day-28→day-1 wrap (Kỳ 4 to next-month Kỳ 1): a timesheet on day 28 of month M is in Kỳ 4 of M, and the cycle math must NOT also include it in Kỳ 1 of M+1. The plan's `Next()` is sketched as pseudocode (`PayrollCycle.Next()`) with no unit test for the wrap. The Kỳ 4 → next-month Kỳ 1 transition is exactly where off-by-one or timezone bugs (the plan uses `Asia/Ho_Chi_Minh` but the timesheet query uses `time.LoadLocation("Local")` per `period_calculator.go:36`) collide.
- **Failure scenario:** A timesheet on 2026-07-28 (last day of Kỳ 4) gets included in Kỳ 4 of July AND in Kỳ 1 of August because the cycle `Next()` wrap pushes `FromDate` to 2026-08-01 but the timesheet query's `< nextDay` with `ToDate = 2026-08-01` resolves to `date < 2026-08-02`, accidentally sweeping in 2026-08-01 timesheets. The `DUPLICATE_ACROSS_BATCHES` validator (Phase 2 table) is supposed to catch this and emit a blocking finding — but the verdict logic says "blocking + unpaid wages → KHONG_THE_TAT_TOAN", so a date-boundary bug silently escalates the verdict to the worst bucket even when production is fine. The Phase 5 cycle test covers `CycleContaining` boundaries but does NOT cover the `Next()` wrap combined with the timesheet query's actual `< nextDay` semantics.
- **Evidence:**
  - `backend/internal/infra/persistence/repositories/timesheet_query_repository.go:636-638` — `nextDay := filters.ToDate.Add(24 * time.Hour); ... timesheets.date < nextDay` — exclusive of nextDay, but if `ToDate` IS the start of the next cycle this still includes that day.
  - `backend/internal/app/services/payroll/bulktransfer/period_calculator.go:36` — `loc, _ := time.LoadLocation("Local")` — different TZ from the planned `Asia/Ho_Chi_Minh`.
  - Phase 5 `payroll_cycle_test.go` row — covers `CycleContaining`, not `Next()` wrap × timesheet query boundary.
- **Suggested fix:** Add a Phase 5 integration test that seeds timesheets on day 28 of month M and day 1 of month M+1, runs a 4-cycle projection starting from Kỳ 4 of M, and asserts (a) day-28 appears only in Kỳ 4, (b) day-1 appears only in Kỳ 1 of M+1, (c) no `DUPLICATE_ACROSS_BATCHES` finding fires. Lock the TZ to `Asia/Ho_Chi_Minh` in BOTH `payroll_cycle.go` and the request `time.Parse` paths — currently they would diverge.

---

## Finding 9: The parity test in Phase 5 will be flaky / self-poisoning — it mutates the seed by exporting, but the suite's other flows depend on the same seed

- **Severity:** Medium
- **Location:** Phase 5, "The parity test (most important — design carefully)" + Risk "test order dependency — parity test mutates, breaking later tests."
- **Flaw:** The parity test pseudocode runs sim (read) → real export (mutates timesheet `payment_status` to `paid` via the disbursement worker pipeline; `MarkExternallyPaid` calls `BulkUpdatePaymentStatus` at `payment_service.go:231`) → parse the Excel. The plan acknowledges the mutation and says "Run this test with a freshly seeded project each time." But the actual integration suite (per `flow_manual_bulk_transfer.go`) shares `data.WeeklyProject`, `data.MonthlyProject`, `data.AdminToken` across flows via a single `TestData`. There is no per-test project teardown today; the existing `flow_manual_bulk_transfer.go` simply downloads the file and doesn't even check whether the export mutated statuses. Once the parity test marks the seeded weekly project's timesheets `paid`, ANY later flow that re-lists eligible timesheets for that project will see an empty pool.
- **Failure scenario:** Test suite ordering is `flowManualBulk → flowSettlementSim` (registration order in `main.go` per Phase 5 step 6). The parity test inside `flowSettlementSim` runs a real export that flips the weekly project's timesheets to `paid`. Then the no-mutation test, the empty-pool test, or any flow registered after `SettlementSimulation` that depends on the weekly project having `pending` timesheets silently breaks — and the failure looks like "the refactor broke production," when it's actually the parity test poisoning the shared fixture.
- **Evidence:**
  - `backend/tests/integration/flow_manual_bulk_transfer.go:11-17` — uses shared `data.WeeklyProject`, `data.WeeklyEmployee`, `data.AdminToken`.
  - `backend/internal/app/services/payroll/bulktransfer/payment_service.go:231` — `BulkUpdatePaymentStatus` is the mutation triggered by `MarkExternallyPaid`.
  - Phase 5 pseudocode line: "// 2. Run the REAL export for the same cycle (this DOES mutate ...)".
  - Phase 5 Risk section admits "scope each mutating test to its own project/timesheet set" but does not say how, given the suite has no per-test fixture isolation today.
- **Suggested fix:** Either (a) introduce a dedicated test project + timesheet cohort created inside `runSettlementSimulationTests` (with teardown at end of flow), OR (b) register `runSettlementSimulationTests` LAST in `main.go` so its mutation cannot poison other flows, OR (c) drop the real-export step from the parity test and instead assert parity by calling `ExportPlanner.Plan` directly vs. `ExportPlanner.Plan` through the simulation service in-process — i.e. remove the "parse the Excel" step entirely. Option (c) is cleanest because the Excel is generated by a service the simulation never invokes.

---

## Finding 10: Route authorization is "inherited from `/payrolls` group" — but the group's `Authorize()` is Casbin-policy based and the plan never verifies the new path is in the policy; the fallback is "admin-only if authorizationService == nil", which is not the production config

- **Severity:** Medium
- **Location:** Phase 3, "Route" + "The `/payrolls` group already sits behind ... `Authorization.Authorize()`, so admin gating is inherited. Confirm `Authorize()` defaults to admin for this group during implementation; if it's broader, add an explicit admin check inside the handler."
- **Flaw:** `Authorize()` (`backend/internal/transport/http/middleware/authorization.go:45-90`) does NOT default to admin. It checks `m.authorizationService.CanAccess(userRole, resource, action)` where `resource = c.Request.URL.Path` (the literal path `/api/v1/payrolls/simulate-settlement`). This is a Casbin policy lookup keyed by URL path. The new path is NOT in any Casbin policy file today (the plan does not list any policy file under "Modify"). The only way an admin can reach the endpoint is if the policy has a wildcard like `/payrolls/*` for `admin` — which must be verified, not assumed. The fallback "if authorizationService == nil → admin-only" (line 62-69) only fires when the service is nil, which is not the production configuration.
- **Failure scenario:** After Phase 3 lands, the new endpoint is mounted but no Casbin policy grants `admin` access to `/api/v1/payrolls/simulate-settlement`. Every admin call returns 403 ("Insufficient permissions"). The feature appears broken in production. Or worse: if an existing broad policy like `partner, /payrolls/*, GET` is in place but the new route is POST, the partner role might or might not be allowed depending on policy verb granularity — Phase 5's "non-admin → 403" test passes either way (it asserts the negative) without proving admin → 200.
- **Evidence:**
  - `backend/internal/transport/http/middleware/authorization.go:62-89` — actual logic: Casbin `CanAccess` with the literal path; no admin default.
  - Phase 3 "Modify" file list — does NOT include any Casbin policy file (`policy.csv`, `rbac_model.conf`, etc.).
  - Phase 5 success criteria — asserts "non-admin → 403" but not "admin → 200 in production-typical config".
- **Suggested fix:** Add to Phase 3's "Modify" list: the Casbin policy file that grants `/api/v1/payrolls/*` to `admin` (find it via `grep -rn "payrolls" --include="*.csv" --include="*.conf" .`). Add a Phase 5 integration assertion: `admin → 200` AND `partner → 403` for the new path specifically, in the production-typical `authorizationService != nil` configuration. Do not rely on the `nil` fallback.

---

## Summary of unverified / dead-end paths

1. `Plan()` "no date filter" entry — dead end (Finding 4).
2. `timesheet → wallet_payment` classification chain — does not exist (Finding 2).
3. `WalletPaymentRepository` batch-by-entity query — does not exist (Finding 3).
4. `LedgerEntryRepository.GetAccountBalanceByDateRange` — does not exist (Finding 7).
5. `fileRepo.Create` in `Export()` — does not exist; Phase 1's premise is wrong (Finding 1).
6. `ReadOnly: true` enforcement — does not work in the configured driver (Finding 5).
7. `SnapshotEpoch` coverage of bank/assignment changes — not implemented (Finding 6).

Each of these is a code path the plan asserts exists or will work but which the codebase contradicts. Resolving them requires concrete plan revisions before implementation, not implementation-time discovery.
