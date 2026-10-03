# Backend implementation report: remove project dates + guarded delete + dynamic fee label

Plan: /Users/dev/.claude/plans/i-want-to-remove-robust-harbor.md (Phases 0, 1, 2, backend half of 4)
Date: 2026-10-03 · Status: DONE

## Preflight (Phase 0)

- **Next migration number is 118, not 116.** Migrations 116 (`116_resolve_mixed_source_advance_periods`)
  and 117 (`117_seed_wallet_balance_alert_threshold`) landed after the plan was written. Creating a
  second "116" would collide in golang-migrate, so the new migration is
  `118_drop_project_date_columns.{up,down}.sql`. Everything else follows the plan.
- Sort-sanitizer allowlist (`backend/internal/infra/persistence/common/`): **no** `start_date`/`end_date`
  entries — nothing to remove. (The projects sort whitelist lives in the handler, not in common/.)
- `AutoCompleteProjects` / `GetPendingCompletedProjects`: only self-references (definition + the call
  from `AutoCompleteProjects` itself). Zero external callers → safe to delete. `AutoActivateProjects`
  has exactly one caller (handler `ActivateProjects`), kept and simplified.

## Files changed

| File | Change |
|------|--------|
| `backend/migrations/118_drop_project_date_columns.up.sql` (NEW) | Drops `projects.start_date`/`end_date`; header notes it is destructive and `make backup` gates production. |
| `backend/migrations/118_drop_project_date_columns.down.sql` (NEW) | Re-adds both columns as `date DEFAULT NULL` (schema only, matches migration 001). |
| `backend/internal/domain/project.go` | Removed `StartDate`/`EndDate` fields, `ValidateDates` + its call in `IsValid`; `ProjectRepository.GetPendingActivatedProjects` loses the date param; `GetPendingCompletedProjects` deleted from interface. `ProjectFinancialSummary` date fields kept (reporting range). |
| `backend/internal/domain/audit_diff.go` | Removed the two date compare blocks in `CompareProjects`. `equalTimePtr`/`timePtrToStr` kept — they live in `employee.go` and have other users. |
| `backend/internal/domain/weekly_payment_fee_schedule.go` | Added `FormatWeeklyPaymentFeePercentage(fraction)` with the required `math.Round(fraction*1e6)/1e4` scaling (without it `%g` prints `1.7999999999999998` for 0.018). |
| `backend/internal/domain/weekly_payment_fee_schedule_test.go` | New `TestFormatWeeklyPaymentFeePercentage`: 0.02→"2%", 0.018→"1,8%", 0.013→"1,3%". |
| `backend/internal/domain/project_test.go` | Deleted `TestProject_ValidateDates`; stripped date literals from `TestProject_IsValid`; dropped `time` import. |
| `backend/internal/app/services/project/service.go` | `autoTerminateEmployees` termination date is always `timeutil.StartOfDay(clock.NowUTC())`; `AutoActivateProjects` activates all drafts (no date gate, no "start_date" in logs); `AutoCompleteProjects` deleted; `validateProjectCanBeDeleted` gained the assigned-employees guard — post-filters `GetActiveAssignments` for `LastDate == nil` (repo query intentionally unchanged) and returns `MsgCannotDeleteProjectWithAssignedEmployeesVN`. |
| `backend/internal/infra/persistence/project_repository.go` | `GetPendingActivatedProjects(ctx)` filters `project_status = 'draft'`; `GetPendingCompletedProjects` deleted. |
| `backend/internal/transport/http/handlers/project/project.go` | Removed `timeToStringPtr`, date parsing/default block in Create, `req.StartDate`/`req.EndDate` blocks in Update, date lines in all three response mappers, `start_date`/`end_date` from the ListProjects sort whitelist; ActivateProjects swagger description updated. |
| `backend/internal/app/dto/project.go` | Removed `StartDate`/`EndDate` from `CreateProjectRequest`, `UpdateProjectRequest`, `ProjectResponse`, `ProjectDetailedResponse`. |
| `backend/internal/constants/messages.go` | `MsgCannotDeleteProjectWithAssignedEmployeesVN = "Không thể xóa dự án đang có nhân viên được giao"` right after the approved-timesheets constant. |
| `backend/internal/seed/projects.go` | Removed date generation block + fields; pruned now-unused `clock`/`time` imports. |
| `backend/internal/seed/assignments.go` | Assignment start = clock fallback with join-variation; early-leaver `last_date` derived from `clock.Now()` (project bounds gone). |
| `backend/internal/seed/payrates.go` | Both `fromDate` blocks reduced to the clock fallback. |
| `backend/internal/app/services/advance_payment/admin_flexpay_import.go` | `getOrCreateProject` no longer sets `StartDate`; removed orphaned `now`/`startDate`. |
| `backend/internal/app/services/employee/import_service.go` | Same removal; dropped now-unused `clock` import. |
| `backend/internal/app/services/payroll/report_exporter.go` | New shared `writeFeeLabelCell` (plain `SetCellValue`, preserves template style); whole-payroll exporter writes D4 label from its `feePercentage`. |
| `backend/internal/app/services/payroll/report_by_project_exporter.go` | Summary sheet writes D3 label from `summary.FeePercentage`. |
| `backend/internal/app/services/notification/email_service.go` | `buildPayrollReportMessage` builds `domain.FormatWeeklyPaymentFeePercentage(feePercentValue)` and threads it through `renderPayrollTemplate` (new param + `FeePercent` data key). |
| `backend/templates/email/payroll_statement.html` | Fee label → `Phí dịch vụ ({{.FeePercent}})`. |
| `backend/internal/app/services/payroll/report_exporter_test.go` | Added D4 assertion ("Phí Dịch Vụ (2%)") to the bank test; new 0.018 test asserting D4 = "Phí Dịch Vụ (1,8%)". |
| `backend/internal/app/services/payroll/by_project_bank_test.go` | Added D3 (2%) assertion; new 0.018 test asserting Summary!D3 = "Phí Dịch Vụ (1,8%)". |
| `backend/internal/app/services/notification/payroll_email_template_test.go` *(deviation, see below)* | Updated the positional `renderPayrollTemplate` call for the new parameter; added label assertions "Phí dịch vụ (2%)" in both tests. |
| `backend/tests/integration/flow_project_crud.go` | Create test additionally decodes the raw payload into a local `projectDateFields` struct and asserts `start_date`/`end_date` absent; the bare delete test was replaced by "Delete blocked while employee assigned" (re-assign employee → `DeleteExpectError` → ≥400 + message contains "Không thể xóa dự án đang có nhân viên được giao") and "Delete succeeds after removing the employee" (remove → delete 2xx, zeroes `testProjectID`). |

## Test results

- `go build ./...` — clean.
- `go test ./... -race` — **exit code 0, 89 packages ok, 0 FAIL** (verified with an unpiped run;
  the earlier piped `tail` run's exit code was `tail`'s, not `go test`'s).
- New/updated tests confirmed passing by name:
  `TestFormatWeeklyPaymentFeePercentage`, `TestProject_IsValid`,
  `TestGenerateExcelWritesConfiguredBankInfo` (now incl. D4), `TestGenerateExcelWritesFeeLabelWithConfiguredPercentage`,
  `TestByProjectExcelShowsConfiguredBank` (now incl. D3), `TestByProjectExcelWritesFeeLabelWithConfiguredPercentage`,
  `TestRenderPayrollTemplateUsesResponsiveFinancialLayout`, `TestPayrollTemplateStaysAlignedAfterProviderBranding`.
- `tests/integration` package compiles (vets clean). The new integration cases run only against a live
  backend (`make api-test` after applying migration 118 to dev DB) — not executed here, per instructions
  not to touch any database.

Tail of the full run:

```text
ok  	api-server/internal/transport/http/handlers/advance_payment	1.815s
ok  	api-server/internal/transport/http/handlers/attendance	1.666s
ok  	api-server/internal/transport/http/handlers/employee	1.703s
ok  	api-server/internal/transport/http/handlers/project_employee	1.822s
ok  	api-server/internal/transport/http/handlers/timesheet	1.707s
ok  	api-server/internal/seed	1.686s
ok  	api-server/tests/integration	1.881s
EXIT_CODE:0  (89 × ok, 0 × FAIL)
```

## Leftover-reference sweep (acceptance criterion 6)

`grep` for remaining `StartDate`/`EndDate` in `backend/internal` finds only unrelated owners:
wallet/topup filters, `CompareLoans`, temporal service, `ProjectFinancialSummary` (repo + ledger
handler), payroll periods, BCC import assignment windows, `CurrentProject`/`CurrentProjectAssignment`
(assignment views — `project_employees.start_date`, explicitly out of scope), and `AssignmentResponse`
in the integration client. No `domain.Project` date fields remain.

## Deviations from the plan

1. **Migration numbered 118, not 116** — preflight showed 116/117 already taken; a duplicate version
   number would break golang-migrate. Naming and content otherwise per plan.
2. **`payroll_email_template_test.go` modified** (one file beyond the ownership list). Adding the
   `feePercent` parameter to `renderPayrollTemplate` — required by the plan's email change — breaks
   that same-package test's positional call at compile time, and acceptance criterion 5 requires the
   full suite to pass. The edit is mechanical (new argument + two label assertions); no other session
   owns backend notification files.
3. **Minor autonomy within plan intent** (each confined to owned files): the bare "Delete test project"
   integration case was folded into "Delete succeeds after removing the employee" (the plan's new case
   mirrors and replaces it — keeping both would fail since the project is already deleted); the
   create-response date-absence assertion uses a small local struct in `flow_project_crud.go` so
   `tests/integration/models.go` (not owned) stays untouched; seed assignment `last_date` now derives
   from `clock.Now()` since project bounds no longer exist.

## Notes for the orchestrator

- The migration has **not** been applied to any database (per instructions). Prod rollout still needs
  the manual `make backup` gate from the plan's verification section.
- Bare-API creates still default to `draft`; `/projects/activate` now promotes all drafts — matches the
  plan's locked decisions.
- Unresolved questions: none.
