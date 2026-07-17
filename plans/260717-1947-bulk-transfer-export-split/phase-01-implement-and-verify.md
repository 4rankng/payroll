# Phase 01 — Implement and verify

**Status:** Completed

## Context

The production export already selects, aggregates, validates, creates transaction codes,
and emits its audit event before generating the workbook. This phase changes only the
binary artifact generation and download metadata. It must not introduce a second payroll
selection or amount-calculation path.

## Files to modify

- `backend/internal/app/services/payroll/excel/service.go`
- `backend/internal/app/services/payroll/excel/service_test.go`
- `backend/internal/app/dto/payroll.go`
- `backend/internal/transport/http/handlers/payroll.go`
- `frontend/src/services/api/bulk-transfer.service.ts`
- `backend/tests/integration/flow_manual_bulk_transfer.go`

No database migrations, environment variables, routes, or UI components are required.

## Implementation

1. In the payroll Excel service, introduce a small internal transfer-row representation
   containing the employee-project key, bank fields, transaction code, and the final VND
   payment amount. Calculate that final amount once using the existing configured payment
   percentage and reuse it for both partitioning and workbook cell output.
2. Sort rows deterministically by employee ID and project ID. Partition them atomically
   in one linear pass: append a row to the current workbook only when the new sum is
   strictly less than 500,000,000 VND; otherwise start the next workbook. Do not split a
   row. Predictable O(n) runtime is preferred over minimizing the number of workbooks.
3. Generate one MBank workbook per partition from the same template. Preserve header,
   transaction-code, bank-field, date-range, cycle, skipped-employee, and empty-export
   behavior.
4. When exactly one workbook is produced, return its bytes as the existing XLSX download.
   When multiple workbooks are produced, archive them with stable part filenames such as
   `<base>_part_01.xlsx` and return one ZIP. Use only Go's standard `archive/zip` support.
5. Add internal response metadata for content type and extension without changing the
   request JSON or any serialized public API response. The handler uses this metadata for
   `Content-Type`, `Content-Disposition`, and `Content-Length`.
6. Update the frontend download service to construct the Blob from the response content
   type (with a safe octet-stream fallback) and continue trusting `Content-Disposition`
   for the filename. No component or hook behavior changes.

## Tests

### Focused backend unit tests

- Total 499,999,999 VND produces one partition/workbook.
- A row that would make a workbook total exactly 500,000,000 VND starts another workbook.
- Multi-row data produces multiple partitions whose totals are all below the cap.
- Every row/transaction code occurs exactly once and combined totals are preserved.
- Partitioning is deterministic despite map insertion order.
- Single and empty exports remain XLSX; multi-file output is a readable ZIP whose entries
  are readable XLSX files and whose column-E totals obey the cap.

### Integration coverage

- Extend the manual bulk-transfer download flow to assert the returned content type,
  filename extension, and non-empty binary body. Exercise ZIP behavior when deterministic
  fixture data can exceed the cap without mutating unrelated flows; otherwise keep the
  archive contract fully covered at service level and document why live data cannot safely
  guarantee the threshold.

### Quality gates

1. Focused Go tests for payroll Excel and bulk-transfer packages.
2. `cd backend && go test ./... -v -race -cover`
3. `cd backend && make lint` (or the repository-equivalent Go format/vet/lint gates).
4. `cd frontend && pnpm lint && pnpm type-check && pnpm build`
5. `make api-test` with the local backend/infrastructure available.
6. Mandatory code review for acceptance criteria, business-logic blast radius, public
   contracts, existing patterns, and lint/type/build regressions.
7. `graphify update .`

## Side-effect review

- `ExportPlanner.Plan`, `ExportService.Persist`, transaction-code creation, and audit
  publishing remain behaviorally unchanged.
- Each original validated employee-project key appears in exactly one workbook.
- Result import continues to use the unchanged transaction code stored in each row.
- The endpoint remains `POST /api/v1/payrolls/export-bulk-transfer`; only binary media type
  and extension vary when splitting is necessary.

## Documentation decision

Update the bulk-transfer QA guide to mention XLSX versus ZIP behavior because this is
user-visible. No ADR, architecture document, or database documentation change is needed.

## Completion record

- Implemented strict post-percentage amount validation and deterministic linear splitting.
- Preserved single/empty XLSX behavior; multi-workbook responses are streamed into ZIP.
- Hardened archive labels and sensitive download headers.
- Updated frontend binary handling and manual-export integration assertions.
- Updated `docs/qa/plan/07-bulk-transfer-payment.md`.
- Final architecture, security, and quality verdicts: **APPROVE**.
