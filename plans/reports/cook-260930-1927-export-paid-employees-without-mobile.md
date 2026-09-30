# Report: Export Paid Employees Without Mobile

- **Plan:** `plans/260930-1841-export-paid-employees-without-mobile/plan.md`
- **Date:** 2026-09-30
- **Status:** DONE (with one local-dev data caveat below)

## Outcome

Admin can export an Excel list of employees who received salary (paid
timesheets) or a completed FlexPay advance within the last N months but have no
mobile number on file, to judge which unreachable employees are still active
from payment recency.

## Changes

Backend:
- `backend/internal/domain/employee_paid_activity.go` — new read model.
- `backend/internal/domain/employee.go` — `EmployeeRepository.GetPaidWithoutMobile`.
- `backend/internal/infra/persistence/employee_repository_queries.go` — union query
  over paid timesheets + COMPLETED advance_payment_requests, LEFT JOIN aggregates,
  working-status flag, project names, ordered by last activity.
- `backend/internal/app/dto/export.go` — `ExportEmployeesPaidWithoutMobileRequest` (months 1–24).
- `backend/internal/app/services/employee/query.go` — service pass-through.
- `backend/internal/transport/http/handlers/employee/employee_export_paid_without_mobile.go` — admin-only
  handler, clock-based window (ADR-006), Vietnamese xlsx, audit event.
- `backend/internal/app/bootstrap/routes_employee.go` — `POST /api/v1/employees/export-paid-without-mobile`.
- `backend/internal/constants/messages.go` — `MsgExportAdminOnlyVN`.
- `backend/mocks/mock_employee_repo.go` — regenerated (mockgen).

Frontend:
- `frontend/src/config/api.config.ts` — endpoint constant.
- `frontend/src/services/api/employee.service.ts` — POST-blob export method.
- `frontend/src/hooks/employees/useEmployeeExport.ts` — `exportPaidWithoutMobile`.
- `frontend/src/components/modals/ExportPaidNoMobileModal.tsx` — months dialog (1/2/3/6/12/24).
- `frontend/src/components/employees/EmployeePageHeader.tsx` — admin-only "NV chưa có SĐT" action.
- `frontend/src/pages/admin/EmployeesPage/index.tsx` — wiring.

## Verification

- `go build ./...` clean; `go vet` clean on touched packages; gofmt applied.
- Unit tests: all touched backend packages pass; regenerated mocks satisfy test files.
- New MySQL-backed repo test `TestEmployeeRepository_GetPaidWithoutMobile` passes,
  pinning union/mobile/soft-delete/aggregate/boundary semantics (skips without `PAYROLL_TEST_DB_DSN`
  or default local MySQL; fixtures isolated in the 214000000xxx band).
- End-to-end on live backend: admin `POST {months:3}` → 200, valid xlsx with all
  Vietnamese headers; partner → 403.
- `pnpm type-check` exit 0; scoped eslint exit 0.
- `make api-test`: 312/343 passed, 29 skipped, 2 failed — both failures are
  settings-flow tests hitting pre-existing data conflicts with the synced dev DB
  (`settings` table already contains `transfer_bank_visible`, duplicate-key on create;
  GET returns 19 rows then 500s in the settings layer). The settings domain is untouched
  by this diff.

## Caveats

1. **Local dev DB rows deleted by an earlier fixture-cleanup range.** The first
   version of the repo test used IDs 999901–999999, a band shared with the
   integration suite's fixtures and crossed by live auto-increment sequences
   (timesheets max id ≈ 999924). The cleanup likely deleted some recent real rows
   in `timesheets`, `advance_payment_requests`, `project_employees`, and
   `payrates` in that band. This is the local synced copy only — production is
   unaffected. Run `make restore` to re-sync the local dev DB from the latest
   OneDrive backup if this copy matters.
2. `payroll-mysql` container was started for testing and left running.
3. The api-server started for end-to-end checks was stopped after testing.
