---
phase: 1
title: "Backend: query relaxation"
status: pending
priority: P1
effort: "3h"
dependencies: []
---

# Phase 1: Backend — query relaxation

## Overview

Restructure `buildMissingBankDetailsBaseQuery` so employees with
`bank_account_status = 'invalid'` are listed **unconditionally** (subject only
to the active-project assignment rule), while missing-info employees keep the
existing pending-work gate. Expose a `bank_warning_kind` field in the list
response so the frontend can label rows without duplicating predicate logic.

## Requirements

- Functional:
  - Invalid-status employees appear in `GET /employees/missing-bank-details`
    with no pending timesheet and no PENDING advance request.
  - Invalid-status employees with an ended assignment (`last_date` set, or no
    active project) do NOT appear.
  - Missing-info employees (any empty bank field, status ≠ invalid) still
    require a pending timesheet or PENDING advance request AND active-project
    assignment.
  - `unverified` employees never appear (unchanged).
  - Partner scoping via `applyFilters` (`AccessibleBy`) is untouched and still
    applies to the whole query — both invalid and missing rows.
  - Response rows carry a machine-readable warning kind: `invalid` or
    `missing`, derived server-side from the same conditions as the predicate.
- Non-functional:
  - Single SQL query shape (no N+1); reuse the existing subquery pattern.
  - Index-friendly: if the OR-structure defeats indexes, the acceptable
    fallback is a UNION of two queries — decide during implementation with
    EXPLAIN on the dev DB, not by default.

## Architecture

Current predicate (`employee_project_query_builder.go:228`):

```sql
(
  employees.bank_id IS NULL
  OR COALESCE(TRIM(employees.bank_account_number), '') = ''
  OR COALESCE(TRIM(employees.bank_account_name), '') = ''
  OR employees.bank_account_status = 'invalid'
)
-- AND id IN (active project subquery)          ← applies to ALL rows today
-- AND (EXISTS pending timesheet OR EXISTS PENDING apr)  ← applies to ALL rows today
```

Target semantics:

```sql
(
  employees.bank_account_status = 'invalid'          -- branch A: always
    AND employees.id IN (active_project_subquery)
)
OR (
  (missing-field predicate)                          -- branch B: unchanged gating
    AND employees.id IN (active_project_subquery)
    AND (EXISTS pending timesheet OR EXISTS PENDING apr)
)
```

Implementation notes:

- Keep `missingBankDetailsPredicate` as the missing-fields-only constant
  (drop the `OR bank_account_status = 'invalid'` term from it).
- Build the combined OR as a single `query.Where(...)` with grouped
  conditions so GORM emits one WHERE clause; do not chain two separate
  `Where` calls (that would AND them).
- An invalid row is never also "missing" per the validation pipeline: a
  confirmed-invalid verdict requires all three fields present
  (`Validate` returns `okValidation` when fields are incomplete — see
  `bank_validation.go:126`). So the branches are mutually exclusive by
  construction; the `bank_warning_kind` derivation can rely on
  `status = 'invalid'` alone.
- `BuildMissingBankDetailsCountQuery` shares the base query, so the count
  badge updates automatically.

## Related Code Files

- Modify: `backend/internal/infra/persistence/query_builders/employee_project_query_builder.go`
  - `missingBankDetailsPredicate` constant (narrow to missing-fields only)
  - `buildMissingBankDetailsBaseQuery` (restructure into grouped OR)
- Modify: `backend/internal/transport/http/handlers/employee/employee_list.go`
  - `GetEmployeesMissingBankDetails` response mapping: add `bank_warning_kind`
    (`"invalid" | "missing"`) per row, derived from
    `emp.BankAccountStatus == domain.BankAccountStatusInvalid`
- Reference (no change): `backend/internal/app/services/employee/bank_validation.go`
  (status semantics), `backend/internal/domain/employee.go:380`
  (`BankAccountStatus*` constants)
- Reference (no change): `backend/internal/infra/persistence/employee_repository_queries.go:99`
  (calls the builder; contract unchanged)
- Tests:
  - `backend/internal/infra/persistence/query_builders/employee_project_query_builder_test.go`
    — add cases per the matrix below
  - `backend/internal/app/services/employee/bank_validation_test.go` —
    unchanged, must keep passing

## Implementation Steps

1. Narrow `missingBankDetailsPredicate` to the three missing-field
   conditions (remove the `bank_account_status = 'invalid'` term).
2. In `buildMissingBankDetailsBaseQuery`, wrap conditions into the grouped OR
   shown above. Keep `applyFilters` first so partner scoping ANDs with the
   whole group.
3. Update the doc comment on the function to describe the two-branch
   semantics (invalid = always subject to active project; missing = pending
   gate) and why `unverified` stays excluded.
4. Handler: add `bank_warning_kind` to the response rows. Follow the existing
   `bankAccountStatusFields` helper style (`employee_list.go:46`) — add a
   sibling helper `bankWarningKind(e *domain.Employee) string` returning
   `"invalid"` or `"missing"`.
5. Run `go build ./...` and the builder tests.

## Test Scenario Matrix (builder tests)

| # | Status | Bank fields | Pending work | Active project | Expected in list |
|---|--------|-------------|--------------|----------------|------------------|
| 1 | invalid | complete | none | yes | ✅ appears (NEW behavior) |
| 2 | invalid | complete | none | no (last_date set) | ❌ hidden |
| 3 | invalid | complete | has pending | yes | ✅ appears |
| 4 | missing | incomplete | none | yes | ❌ hidden (unchanged) |
| 5 | missing | incomplete | has pending | yes | ✅ appears (unchanged) |
| 6 | unverified | complete | none | yes | ❌ hidden (unchanged) |
| 7 | unverified | complete | has pending | yes | ❌ hidden (unchanged) |
| 8 | valid | complete | none | yes | ❌ hidden (unchanged) |
| 9 | invalid + missing fields (edge: manual DB edit) | incomplete | none | yes | ✅ appears as invalid (branch A wins) |

Case 9 documents precedence: if somehow both conditions hold, the row is
returned and labeled by status. Assert `kind = invalid`.

Check the builder test file's existing pattern first (sqlmock vs live DB) and
follow it rather than introducing a new one.

## Success Criteria

- [x] Builder emits the grouped-OR WHERE clause (verified via existing test
  pattern or EXPLAIN on dev DB)
- [x] All 9 matrix cases pass
- [x] Existing missing-bank tests still pass (cases 4-8 are regression guards)
- [x] `bank_warning_kind` present on all rows of the endpoint response
- [x] `go vet ./...` clean for touched packages

## Risk Assessment

- **OR-grouping + GORM**: accidental AND of two `Where` calls would silently
  drop invalid rows. Mitigation: single `Where` with explicit parentheses;
  test case 1 catches it.
- **Partner scope interplay**: scoping must apply to both branches. It does —
  `applyFilters` runs before the predicate. Integration test in phase 3
  re-verifies with a partner token.
- **Count/list drift**: count query reuses base query, so no drift possible
  by construction.
