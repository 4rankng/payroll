# Bulk-transfer export file-size cap

**Date:** 2026-07-17  
**Status:** Completed  
**Type:** Backward-compatible export enhancement

## Outcome

Manual MBank batch-transfer exports keep the amount in every workbook strictly below
500,000,000 VND. Exports that fit remain a single `.xlsx`; exports requiring two or
more workbooks download as one `.zip` containing the generated `.xlsx` files.

## Confirmed requirements

- The cap is applied to the exact VND amounts written to column E, after the configured
  weekly/monthly payment percentage is applied.
- Employee-project transfer rows remain atomic. The business guarantees one row will
  not reach 500,000,000 VND.
- Each workbook total is at most 499,999,999 VND.
- The existing endpoint, request body, filters, transaction codes, audit event, and
  result-import reconciliation remain compatible.
- Automatic OnePay/9Pay transfers, result import, database schema, configuration, and UI
  layout are out of scope.

## Phases

| Phase | Status | Deliverable |
|---|---|---|
| [01 — Implement and verify](phase-01-implement-and-verify.md) | Completed | Deterministic partitioning, XLSX/ZIP response handling, regression tests |

## Dependencies and touchpoints

- Existing read/write orchestration in
  `backend/internal/app/services/payroll/bulktransfer/export_service.go` stays intact.
- Workbook generation is owned by
  `backend/internal/app/services/payroll/excel/service.go`.
- Binary download metadata flows through `backend/internal/app/dto/payroll.go`,
  `backend/internal/transport/http/handlers/payroll.go`, and
  `frontend/src/services/api/bulk-transfer.service.ts`.
- The implementation must remain compatible with
  `plans/260717-1700-settlement-simulation/`, which reuses the export planner.

## Acceptance criteria

- A calculated export total below 500,000,000 VND returns one valid MBank `.xlsx`.
- A calculated export total at or above 500,000,000 VND is partitioned into valid MBank
  workbooks, every workbook total is below 500,000,000 VND, and the response is a valid
  ZIP containing all workbooks.
- The sum of all workbook row amounts equals the unsplit export sum, with every transfer
  row and transaction code appearing exactly once.
- Empty exports preserve the current single-workbook behavior.
- Existing request and result-import contracts do not change.
- Focused tests, backend unit tests, frontend lint/type-check/build, integration tests,
  and graph update pass without unrelated changes.

## Risks and rollback

- **Amount drift:** partitioning and cell writing must share one calculated row amount;
  tests compare workbook totals and the original total.
- **Nondeterministic Go map order:** rows are sorted by employee ID and project ID before
  deterministic linear sequential partitioning. This may create more workbooks than bin
  packing, but keeps runtime predictable and every workbook safely below the cap.
- **Download compatibility:** the handler supplies the correct extension, MIME type, and
  content length; the frontend uses response headers rather than hard-coded XLSX MIME.
- **Rollback:** revert the export-generation and download metadata changes; no data or
  schema rollback is required.

## Verification outcome

- Focused payroll Excel, bulk-transfer, payroll, handler, and integration-package tests
  pass with race detection where applicable.
- Package-scoped `golangci-lint` and `go vet` report no issues.
- Backend `go build ./...` passes.
- Frontend `pnpm lint` (including TypeScript checking) and `pnpm build` pass.
- Architecture, security, and code-quality validators all approved the final change.
- The live API integration runner reached the local server but could not authenticate
  (`Xin đăng nhập để tiếp tục`); this environment issue is unrelated to export logic.
- The repository-wide backend suite still contains unrelated failures in bootstrap/config,
  NinePay, cache-event race tests, and database-fixture tests. All packages touched by this
  feature pass their focused gates.
