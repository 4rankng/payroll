# Red-Team Plan Review — Settlement Simulation
**Reviewer role:** Security Adversary (auth bypass, injection, data exposure, privilege escalation, OWASP top 10)
**Target:** `plans/260717-1700-settlement-simulation/` (all 7 files)
**Verification posture:** Every claim below is grep-verified against the actual codebase. No speculative findings.

---

## Finding 1: The plan's central auth claim is wrong — `Authorize()` does NOT default to admin
- **Severity:** High (mitigating control absent; but current default-deny happens to protect the route)
- **Location:** Phase 3, section "Route"; also plan.md locked decision #2 implies admin inheritance
- **Flaw:** The plan states: *"The `/payrolls` group already sits behind `Auth.Authenticate()` + `Authorization.Authorize()`, so admin gating is inherited. Confirm `Authorize()` defaults to admin for this group during implementation; if it's broader, add an explicit admin check inside the handler (mirror `settle_from_notification.go:16`)."* This frames admin-gating as the default and an in-handler `isAdmin(c)` check as a fallback. The reality is the opposite: `Authorize()` performs Casbin RBAC with `keyMatch2` matching, and the only reason `/payrolls/simulate-settlement` will be denied to partners is **default-deny** (no allow rule matches), not "default admin."
- **Failure scenario:** A future maintainer reads this phase, trusts the claim that `/payrolls` is admin-inherited, and later adds a partner rule that *accidentally* matches `simulate-settlement` (e.g. `p, partner, /api/v1/payrolls/*, *, allow`, or broadening `/api/v1/payrolls/histories` to a glob). Because there is no in-handler admin check (the plan defers to "if it's broader" rather than mandating it), partner/adv_partner suddenly gain the ability to enumerate cross-project employee PII via the simulation. The plan's "confirm during implementation" punt is exactly the kind of conditional that gets skipped under time pressure.
- **Evidence:**
  - `backend/internal/app/bootstrap/routes.go:48-83` — `protected := v1.Group("/"); protected.Use(Auth.Authenticate()); protected.Use(Authorization.Authorize())`. Generic, role-agnostic.
  - `backend/internal/transport/http/middleware/authorization.go:45-90` — `Authorize()` calls `m.authorizationService.CanAccess(userRole, resource, action)` for all non-timesheet routes; there is **no admin default**. The only admin-fallback branch (lines 62-69) fires when `authorizationService == nil`, which is not the production path.
  - `backend/configs/casbin_policy.csv:83-86` — the *only* partner rules for `/payrolls` are `histories GET` and `bulk-transfer-template GET`. Neither matches `/payrolls/simulate-settlement POST` under `keyMatch2`. Default-deny is the sole protection.
  - `backend/configs/casbin_model.conf:14` — `m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")`. No wildcard admin fallback for non-admin roles.
  - `backend/internal/transport/http/handlers/settlement/settle_from_notification.go:15-19` — the cited "admin-only pattern" is an **explicit in-handler** `if !isAdmin(c)` check, NOT middleware inheritance. The plan misreads the reference.
  - `backend/internal/transport/http/handlers/payroll.go:93-123` — the sibling `ExportBulkTransfer` handler has **no** in-handler admin check; it relies entirely on Casbin. So the plan's "inherit admin from group" claim is also historically inaccurate for this exact group.
- **Suggested fix:** Mandate, do not "consider": every endpoint in the plan that returns employee/project/timesheet aggregates MUST call `isAdmin(c)` at the top of the handler (mirroring `settle_from_notification.go:16`), regardless of middleware. Add a permanent Phase 5 unit test that asserts a forged partner-role token yields 403 *at the handler*, not at middleware, so the test still passes if Casbin policy drifts.

---

## Finding 2: Bank account numbers are exposed raw in the API response — masking exists only in the UI
- **Severity:** High
- **Location:** Phase 3, section "API Contract (final)" — `cycles[].included[].bank_account_number: "....1234"`; Phase 4 step 8 ("Mask bank account numbers in the UI")
- **Flaw:** The DTO example shows a masked value `"....1234"`, but nowhere in Phases 2 or 3 does the plan specify server-side masking. Phase 4 step 8 explicitly relegates masking to the *frontend*: "Mask bank account numbers in the UI (show last 4 digits)." This means the production `SimulationService.Simulate` will serialize full, unmasked `employee.BankAccountNumber` into the JSON response, traverse the network, hit browser memory, and only then be visually masked. The example DTO value is misleading — it implies server masking that is not actually planned.
- **Failure scenario:**
  1. A partner or compromised admin session captures the raw response via devtools, browser extension, or a proxy — full account numbers for every employee across every project (the simulation intentionally aggregates the *full outstanding pool*, no project filter by default) are exposed in transit and in client memory.
  2. Frontend console logs (the plan itself warns in Phase 4 risk section: "Never log full account numbers to the console" — confirming the data *is* present in client memory).
  3. Server access logs / APM that capture response bodies (some tracing setups do) record full PAN-like data.
  4. Any future non-admin client (see Finding 1) reaps unmasked data wholesale.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:386-387` — production aggregation already loads `BankAccountNumber: employee.BankAccountNumber` raw into `excel.Employee`. The new `planner.Plan()` (Phase 1) is a pure cut-paste of this logic, so it preserves raw account numbers in the returned `ExportPlan`.
  - `backend/internal/app/dto/employee.go:15,28,54,178` — every existing employee DTO uses raw `BankAccountNumber string json:"bank_account_number"`. There is **no masking helper anywhere** in the codebase (`grep MaskAccountNumber|maskedAccount` returns zero results).
  - `phase-04-...md:179` — "Mask bank account numbers in the UI (show last 4 digits) — these are PII; full numbers don't belong on a preview screen." UI-only.
  - `phase-04-...md:201-202` Risk section: "PII (bank account) leakage in preview. Mitigation: always mask to last 4. Never log full account numbers to the console." — confirms the data crosses the wire in plaintext and the only protection is JS discipline.
- **Suggested fix:** Mask server-side in the DTO assembler (Phase 3), not the UI. Either: (a) emit only `bank_account_number_last4` and drop the full field entirely from `SimulationRow`, or (b) write a `maskAccount(s string) string` helper invoked at serialization time and add a unit test asserting no full account number appears in any marshalled `SimulationResult`. Then the UI masking in Phase 4 becomes defense-in-depth, not the primary control.

---

## Finding 3: `OP_LOSS` / `STUCK_IN_FLIGHT` classification for *timesheet* remainders is unimplementable — no timesheet→wallet_payment link exists
- **Severity:** Critical (the plan's headline business output cannot be produced as specified)
- **Location:** Phase 2, section "Verdict computation (full-pool)" step 4 `classifyByMoneyFlow`; remainder classification table; Phase 5 test 5b "Op-loss detection"
- **Flaw:** The plan specifies classifying each unpaid *timesheet* by walking `timesheet → transaction → settlement → wallet_payment, OR timesheet → advance_payment_request → wallet_payment`, then reading `wallet_payment.Status`. In the actual schema, `wallet_payments.entity_id` is **only** an index of `advance_payment_requests.id`. There is no `timesheets.id → wallet_payments.entity_id` join, no `timesheets.transaction_id → wallet_payments` chain that reaches a wallet payment row, and the `WalletPaymentRepository` interface exposes no `GetByTimesheetID` method.
- **Failure scenario:** When the implementer reaches Phase 2 step 4, they will discover the join does not exist. Every unpaid payroll *timesheet* remainder will fall into the `else` branch → classified `UNPAID_WAGES` regardless of whether money has actually left the account via the advance-payment pipeline for that employee. The headline `KHONG_THE_TAT_TOAN` verdict (any `OP_LOSS` → block) will then never fire for payroll-path remainders — defeating acceptance criterion #2. Worse: Phase 5 test 5b ("paid-out advance not recovered → KHONG_THE_TAT_TOAN") cannot be seeded with the schema as-is.
- **Evidence:**
  - `backend/internal/domain/transactions/wallet_payment.go:34` — `EntityID *uint64 gorm:"column:entity_id;type:bigint unsigned;index"`. Comment-free; no FK to timesheets.
  - `backend/internal/domain/transactions/repository.go:54-60` — `HasNonTerminalByEntityID` doc explicitly says *"linked to the given **advance_payment_requests.id** (column entity_id)"*. No timesheet variant.
  - `backend/internal/infra/persistence/advance_payment_request_repository.go:661,663` — the only existing join is `wallet_payments.entity_id = advance_payment_requests.id`. Grepping `internal/domain/` and `internal/infra/persistence/` for `timesheet.*wallet|wallet.*timesheet|timesheets.*wallet_payment` returns **zero** results.
  - `backend/internal/domain/timesheet.go:60` — `TransactionID *uint` links to a *revenue* transaction (per the comment: "Linked revenue transaction for this timesheet"), not to `wallet_payments`. There is no documented chain from there to a wallet_payment row.
  - `phase-02-...md:149-153` — plan literally invents two chains, neither of which has a backing schema path.
- **Suggested fix:** Either (a) restrict the remainder classifier to operate only on `advance_payment_requests`-linked items (and explicitly document that payroll timesheet remainders are always classified `UNPAID_WAGES` because the disbursement pipeline runs through bulk-transfer files, not `wallet_payments`), or (b) add a Phase 0 task to map the actual employee→wallet_payment join (likely via the employee's `BankAccountNumber` and `RecipientAccountNo`, which is a weak identity match and itself a data-exposure concern). Pick one and rewrite the classification table and Phase 5 test 5b accordingly. As written, the plan promises an output the data model cannot deliver.

---

## Finding 4: `IfMatchSnapshot` 409 is a clean data-state oracle — abusable as a change-detection side-channel
- **Severity:** Medium
- **Location:** Phase 3, section "Stale-snapshot guard on real export"
- **Flaw:** The opt-in `IfMatchSnapshot` check returns 409 *iff* any relevant row's `updated_at > snapshot_epoch`, else 200. The branch is unconditional and distinguishes only on data state. An authenticated admin (or any future non-admin per Finding 1) can submit `POST /export-bulk-transfer` with `if_match_snapshot` set to arbitrary epoch values and observe 200 vs 409 to binary-search the exact `updated_at` of any timesheet in scope — without ever completing the export (because the 409 path returns before `Persist()`). This is a classic blind oracle.
- **Failure scenario:** Attacker with a leaked admin token (or a partner if Finding 1's mitigation is skipped and Casbin drifts) iterates `if_match_snapshot` values for a target `project_ids=[X]` and observes the 200/409 boundary. They learn the precise moment any timesheet in project X was last edited — useful for insider-trading-style inference (e.g. "this project's payroll was edited at 03:14 last night, just before the disbursement cut"). Repeated calls also serve as a high-resolution timing oracle because the `Plan()` half runs before the guard.
- **Evidence:**
  - `phase-01-...md:73-79` — Phase 1's `Export()` rewrite places the snapshot check between `Plan()` and `Persist()`. `Plan()` performs the full read selection (multi-table join over timesheets, employees, projects, project_employees) before the guard fires, so timing differences reflect real workload, amplifying the side-channel.
  - `phase-03-...md:60-64` — the guard is `if req.IfMatchSnapshot != nil && !req.IfMatchSnapshot.IsZero() { if plan.SnapshotEpoch.After(*req.IfMatchSnapshot) { return ErrStaleSimulation } }`. No rate-limit beyond the global `APIRateLimit` middleware; no audit log on the 409 path.
  - `backend/internal/app/bootstrap/routes.go:30` — only `container.Middleware.APIRateLimit` and `APIMetrics` sit on `v1`; there is no per-endpoint rate limit.
- **Suggested fix:** Either (a) require `if_match_snapshot` to be a value previously issued by a *successful* simulation tied to the caller's session (track issued epochs server-side with a TTL), rejecting any caller-supplied epoch that was not issued to them, or (b) rate-limit the `export-bulk-transfer` endpoint's 409 path aggressively and emit an audit event on every stale-snapshot rejection so the oracle is at least visible to forensics. As written, the guard is an unauthenticated-state-detection primitive.

---

## Finding 5: ReadOnly transaction pseudocode is type-incorrect and would not compile
- **Severity:** High (the plan's "belt-and-suspenders" read-only guarantee is broken at the API layer; falls back entirely to the "Plan is pure" claim)
- **Location:** Phase 2, section "Read-only enforcement"
- **Flaw:** The pseudocode is:
  ```go
  s.db.Transaction(func(tx *gorm.DB) error { ... }, &gorm.Session{&sql.TxOptions{ReadOnly: true}})
  ```
  The actual GORM signature is `func (db *DB) Transaction(fc func(*tx *DB) error, opts ...*sql.TxOptions)`. The second argument must be `*sql.TxOptions`, not `*gorm.Session`. `&gorm.Session{...}` is a different type entirely. The plan also misuses `gorm.Session` (which takes fields, not a pointer to `*sql.TxOptions`).
- **Failure scenario:** The implementer copies the pseudocode verbatim, the build fails, and they "fix" it by dropping the options argument entirely (the path of least resistance when a deadline looms). The result: the simulation runs in a default read-write transaction, and the only remaining guard against accidental mutation is the Phase 1 claim that `Plan()` is pure — exactly the single point of failure the read-only tx was supposed to backstop.
- **Evidence:**
  - GORM v1.30.1 source (the pinned version per `go.mod`): `gorm.io/gorm@v1.30.1/finisher_api.go` — `func (db *DB) Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error)`.
  - `backend/go.mod` — `gorm.io/gorm v1.30.1`.
  - Existing codebase pattern: `backend/internal/infra/transaction/gorm_transaction_manager.go:25,45` and `backend/internal/infra/persistence/timesheet_repository.go:217` all call `db.Transaction(func(tx *gorm.DB) error {...})` with **no** options argument — confirming the option is unused anywhere in the project today, so there's no precedent for the implementer to follow.
  - `phase-02-...md:188-198` — pseudocode as quoted.
- **Suggested fix:** Replace the pseudocode with `s.db.Transaction(func(tx *gorm.DB) error { ... }, &sql.TxOptions{ReadOnly: true})` and add a Phase 5 unit test asserting the `*sql.TxOptions` is actually passed (e.g. by injecting a fake `ConnPool` that records the `BeginTx` opts). Note that even with this fix, MySQL `START TRANSACTION READ ONLY` is honored (verified in `go-sql-driver/mysql@v1.8.1/connection.go:115-125`), so the guarantee is real once the call compiles.

---

## Finding 6: No audit trail — the simulation is the single largest unlogged mass-PII read in the system
- **Severity:** High
- **Location:** plan.md locked decision #2 ("Strictly read-only... NO writes. NO status changes. NO file/code rows. NO events."); Phase 6 step 6 ("confirm the simulation does NOT publish any event")
- **Flaw:** The plan treats "read-only = no audit event" as a virtue. But the codebase explicitly audits *less sensitive* reads: `BulkTransferFileDownloadedEvent` ("Sensitive enough to audit") fires when an admin re-downloads a single previously-generated file. The simulation, by design, returns full employee/project/timesheet aggregates with bank account numbers across the **entire outstanding pool** (no project filter by default) for **4–6 cycles at once** — a strictly broader disclosure than a single file re-download — and emits nothing.
- **Failure scenario:** A compromised or insider-admin account runs the simulation repeatedly to bulk-harvest employee bank account numbers and amounts across every project (a single `projected_cycle_count=6` call discloses more than any individual export). Because there is no audit event, SOC review sees nothing. After exfiltration, the only forensic signal is an HTTP access log entry indistinguishable from any other `/simulate-settlement` call. The plan even mandates *no TanStack Query invalidation* in Phase 4, which is correct for read-only semantics but reinforces the "this call leaves no trace" posture.
- **Evidence:**
  - `backend/internal/domain/events.go:611-618` — `BulkTransferFileDownloadedEvent` exists specifically to audit re-downloads, with the comment "Sensitive enough to audit."
  - `backend/internal/domain/events.go:620-627` — `PayrollHistoriesExportedEvent` audits cross-employee Excel export.
  - `backend/internal/domain/events.go:583-595` — `BulkTransferFileExportedEvent` carries `TransactionsCount` and `TotalAmount` for audit. Every adjacent operation in this domain is audited.
  - `plan.md:113` locked decision #2 — "NO writes. NO status changes. NO file/code rows. NO events." Explicitly forbids the audit event.
  - `phase-06-...md:56` — step 6 verifies *no* `eventBus.Publish` call exists in the new code, treating the absence as success criteria.
- **Suggested fix:** Read-only ≠ audit-free. Add a new `SettlementSimulationRunEvent` (carrying actor user_id, project_ids, projected_cycle_count, total_eligible_count, verdict, snapshot_epoch — **no** PII, no account numbers) and publish it on every successful simulation. Audit events are not "writes" in the data-mutation sense the plan is protecting against; they are observability. Update Phase 6 step 6 to *require* exactly one audit event per call, not zero.

---

## Finding 7: Full-pool verdict runs an unscoped scan — `for_month=null, project_ids=[]` returns every outstanding timesheet in the system
- **Severity:** Medium
- **Location:** Phase 2, "Verdict computation (full-pool)" step 1; Phase 3 API contract (`project_ids: []` = all)
- **Flaw:** The full-pool scan calls `planner.Plan(ctx, reqWithNoDateFilter)` with `ProjectIDs=[]`, `EmployeeIDs=[]`, `ForMonth=""`, and **no** date filter. This selects every approved timesheet with `payment_status IN (pending, failed)` across all projects, all employees, all time. The plan calls this "the entire outstanding pool" and treats the unbounded result as a feature.
- **Failure scenario:**
  1. **Performance / DoS:** On a multi-year deployment with hundreds of thousands of historical failed/pending timesheets (the schema has no archival — `DeletedAt` is soft-delete only, `backend/internal/domain/timesheet.go:61`), the full-pool scan plus 4–6 cycle calls each re-run `aggregateTimesheetData`'s per-employee `GetByID` and per-project `GetByID` (N+1 pattern preserved from `export_service.go:376,400`). The result is a multi-second, multi-hundred-MB response that blocks a DB connection and can be triggered by any admin indefinitely.
  2. **Data minimization:** The response will contain names, project names, amounts, and (per Finding 2) raw bank account numbers for employees the admin has no business reason to inspect. Even if admins are trusted, "the admin clicked simulate" is not a legitimate basis to dump the entire historical payroll ledger into one HTTP response.
- **Evidence:**
  - `phase-02-...md:135-138` — `fullPool := planner.Plan(ctx, reqWithNoDateFilter)` with no project/date scoping.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:111,376,400` — production `Plan()` issues per-employee `GetByID` and per-project `GetByID` in a loop. Confirmed N+1; the plan reuses this verbatim ("cut-paste, no edits").
  - `backend/internal/domain/timesheet.go:61` — `DeletedAt gorm.DeletedAt` (soft delete). Historical rows never physically drop, so the pool is monotonically growing.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:96-100` — production filter is `TimesheetStatus=approved AND PaymentStatus IN (pending, failed)` with no upper date bound when monthly mode is used.
- **Suggested fix:** Bound the full-pool scan: require at least one of `project_ids`, `employee_ids`, or `for_month` to be set (return 400 otherwise — "specify a scope"), OR cap the full-pool query to e.g. 90 days back. Add a row-count guard in the handler that returns 413/422 if the projected result exceeds a threshold (e.g. 50k rows) rather than serializing the whole ledger. Document the cap in the API contract.

---

## Finding 8: Snapshot epoch computed over preloaded structs is unreliable — stale-snapshot guard can be bypassed or false-positive
- **Severity:** Medium
- **Location:** Phase 1, section "`SnapshotEpoch` computation"
- **Flaw:** The primary computation walks preloaded structs (`ts.UpdatedAt`, `employeeData`, `projectData`, `assignmentCache`) and takes the max. The plan acknowledges preload reliability is in doubt and offers a fallback (`SELECT MAX(updated_at) FROM timesheets WHERE id IN (?)`) but does not commit to it. The fallback itself is incomplete: it covers `timesheets` only, not employees/projects/project_employees — yet those are exactly the rows whose mutation invalidates the snapshot (a changed `BankAccountNumber` or `PaymentSchedule` reshapes the export without touching any timesheet's `updated_at`).
- **Failure scenario:**
  1. **Bypass:** Admin edits an employee's `bank_account_number` (or a project-employee's `payment_schedule`) at T+1. The timesheet rows are untouched. The simulation's `snapshot_epoch` reflects only timesheet `updated_at`, so it equals the pre-edit value. The real export's `Plan()` recomputes `SnapshotEpoch` the same way — also equal to the pre-edit value. The 409 guard therefore passes (no drift detected), the export proceeds, and the simulation result the admin relied on is silently stale (different selection due to the changed bank/schedule).
  2. **False positive:** If preloading is inconsistent (GORM does not preload `UpdatedAt` on `Employee` unless explicitly requested — and `aggregateTimesheetData` only fetches a hand-built `excel.Employee`, not the full domain struct with `UpdatedAt`), the max may be computed over zero-valued `time.Time{}` for employees, yielding `snapshot_epoch = max(timesheets only)`. This is deterministic but does not match the documented semantics.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:382-396` — production aggregation builds `excel.Employee{ID, Fullname, BankAccountNumber, BankAccountName, Bank}` — **no** `UpdatedAt` field. The plan's "walk employeeData for updated_at" cannot work against this struct.
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:405-409` — same for `excel.Project{ID, Name}` — no `UpdatedAt`.
  - `phase-01-...md:96-105` — computation pseudocode as quoted, with the acknowledged fallback.
- **Suggested fix:** Commit to the explicit-SQL fallback and extend it: a single query that returns `GREATEST(MAX(timesheets.updated_at), MAX(employees.updated_at), MAX(projects.updated_at), MAX(project_employees.updated_at))` over the in-scope IDs. Add a Phase 5 unit test that mutates an employee's bank account (not a timesheet) and asserts the 409 path fires — without this, acceptance criterion #6 ("real export cannot silently proceed using stale simulation results") is not actually met for the most common real-world mutation.

---

## Finding 9: `for_month` query parameter is accepted but never validated — date injection into the planner
- **Severity:** Medium
- **Location:** Phase 3, "API Contract (final)" Request table (`for_month` `string`, "if set, projects the 4 cycles of that month"); Phase 2 `cloneRequestForCycle`/`planner.Plan` reuses the existing `ResolveMonthlyRange`
- **Flaw:** `for_month` is documented as `YYYY-MM` but the request DTO carries no validation tag. The existing `ExportBulkTransferRequest.ForMonth` is also unvalidated at the DTO layer (binding tag is empty), and validation happens inside `periodCalculator.ResolveMonthlyRange`. The simulation will pass `for_month` straight through to `planner.Plan()`. If `ResolveMonthlyRange` does not strictly reject malformed input, an attacker can probe parser behavior with values like `2026-13`, `0000-00`, `2026-1`, `2026; DROP TABLE`, etc. Even if the parser rejects these, the *interaction* between `for_month` and `projected_cycle_count` is undefined — the cycle model in `payroll_cycle.go` walks forward from `CycleContaining(clock.Now())`, so what does `for_month` even mean when the cycle helper doesn't read it?
- **Failure scenario:** An admin submits `for_month=2026-07-31` (full date instead of YYYY-MM). The planner's `time.Parse` may accept it (depending on the format string used), producing an unexpected `monthStart` that selects a different timesheet set than the simulation's `CycleContaining` logic assumes. The verdict then mis-reports coverage. Worse, `for_month` and the cycle-walk are two independent date-resolutions in the same call — the plan never reconciles them.
- **Evidence:**
  - `backend/internal/app/dto/payroll.go:15` — `ForMonth string json:"for_month,omitempty"` — no `binding:"required,datetime=2006-01"` or similar tag.
  - `phase-03-...md:97-98` — Request table documents the format but the DTO carries no validator.
  - `phase-02-...md:69-70` — cycle loop uses `cloneRequestForCycle(req, cycle)` to set weekly mode and drop `ForMonth`, but the **full-pool scan** at step 1 (line 137) reuses `req` with `ForMonth` intact if the caller set it. So the full-pool scan can run in monthly mode while the per-cycle scans run in weekly mode — an inconsistency the plan does not address.
- **Suggested fix:** (a) Add `binding:"omitempty,datetime=2006-01"` to `ForMonth` in the simulation request DTO and unit-test rejection of `2026-7`, `2026-13`, `202607`, empty-after-trim. (b) Decide and document whether `for_month` overrides the cycle-walk start (in which case `payroll_cycle.go` must read it) or is incompatible with cycle projection (in which case return 400 if both are set). As written, the two parameters silently interact.

---

## Finding 10: Phase 5 parity test runs a real mutating export inside the test DB — cross-test contamination risk
- **Severity:** Medium
- **Location:** Phase 5, section "The parity test (most important — design carefully)"; Risk Assessment "test order dependency"
- **Flaw:** The parity test deliberately calls the real `POST /api/v1/payrolls/export-bulk-transfer` against the integration DB, which (per `export_service.go:165,184,507`) calls `saveBulkTransferFile` (writes `bulk_transfer_files` + `transaction_codes.CreateBatch`) and `publishExportAudit`. The plan acknowledges this ("this DOES mutate") and proposes ordering (sim first, then export). But the plan also mandates a "no-mutation" test that runs in the same suite. The risk is not just ordering — it is that the parity test's writes persist in the shared integration DB and can pollute *other* flows (e.g. `flow_bulk_transfer.go`, `flow_manual_bulk_transfer.go`) that count `bulk_transfer_files` or list upload histories.
- **Failure scenario:** The parity test runs, leaves behind N `transaction_codes` rows and one `bulk_transfer_files` row tagged to the test project. A later (or parallel) `flow_bulk_transfer.go` assertion that counts pending uploads now sees an unexpected extra row and fails — intermittently, depending on test ordering. The plan's mitigation ("scope each mutating test to its own project/timesheet set and clean up") is stated but has no enforcement: there is no teardown specified, and the existing `flow_manual_bulk_transfer.go` pattern (cited as the reference) does not clean up `transaction_codes`.
- **Evidence:**
  - `backend/internal/app/services/payroll/bulktransfer/export_service.go:165,184,507` — confirmed write sites: `saveBulkTransferFile` and `transactionCodeRepo.CreateBatch` and `publishExportAudit`.
  - `phase-05-...md:114-125` — pseudocode runs real export and parses the resulting file.
  - `phase-05-...md:194-199` — Risk: "test order dependency — parity test mutates." Mitigation: "order the suite so mutations come last, OR scope each mutating test to its own project/timesheet set and clean up." No actual cleanup code is specified.
  - `backend/tests/integration/main.go` — single shared suite; no per-test isolation framework visible.
- **Suggested fix:** Either (a) run the parity test inside a transaction that is explicitly rolled back at the end (acceptable even though it's a "real" export — the rollback provides the isolation), or (b) dedicate a reserved project ID and add a `t.Cleanup` that deletes `transaction_codes` and `bulk_transfer_files` rows for that project. Add a Phase 5 success criterion: "after the parity test, `bulk_transfer_files` and `transaction_codes` row counts equal pre-test counts" — i.e. the parity test must itself be no-mutation with respect to the rest of the suite.

---

## Summary of severities
- **Critical (1):** Finding 3 — `OP_LOSS` classification unimplementable; headline business output cannot be produced.
- **High (5):** Findings 1 (auth claim wrong), 2 (raw PAN in API), 5 (ReadOnly pseudocode won't compile), 6 (no audit trail), 8 (snapshot epoch unreliable) — each independently can ship a defect that passes CI but fails in production.
- **Medium (4):** Findings 4 (409 oracle), 7 (unscoped full-pool scan), 9 (`for_month` validation gap), 10 (parity test contamination).

The plan reads as polished and internally consistent, but its security claims are not load-bearing against the actual codebase. The most dangerous pattern is **conditional mitigations presented as defaults** ("confirm during implementation", "if it's broader, add a check", "if GORM doesn't preload reliably, fall back"). Each such conditional is a defect deferred to a deadline-pressured implementer. Convert every one of them to a mandate with an automated test, or this plan will produce a feature that passes its own suite while leaking PII, mis-classifying op-loss, and leaving no forensic trail.
