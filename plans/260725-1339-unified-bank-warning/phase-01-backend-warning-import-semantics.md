---
phase: 1
title: Backend warning + import semantics
status: completed
effort: medium
---

# Phase 1: Backend warning + import semantics

## Overview

Make the unified warning query complete and give BCC employee creation an
explicit allow-and-flag path without weakening strict manual writes.

## Implementation Steps

1. Add a shared bank-completeness helper or equivalent explicit predicate so
   missing `bank_id`, account number, or account holder name are all classified
   as incomplete.
2. Update `EmployeeProjectQueryBuilder` missing-bank list/count predicates to
   include any incomplete required field or `bank_account_status='invalid'`;
   keep complete `unverified` rows excluded.
3. Introduce an import-specific employee creation path that performs the same
   normalization, domain validation, bank reference checks, duplicate checks,
   persistence, events, and validation status folding as normal creation, but
   persists confirmed-invalid bank results instead of returning
   `BANK_ACCOUNT_INVALID`.
4. Route standard, weekly, and multi-position BCC auto-creation through that
   path. Keep `UpdateBankInfo` as the existing allow-and-flag chokepoint for
   existing employees.
5. Keep manual `CreateEmployee` and `UpdateEmployee` on the strict
   reject-before-persist path.
6. Add focused backend tests for incomplete-field permutations, invalid and
   unverified statuses, import creation persistence, and strict manual
   regression behavior.

## File Inventory

- `backend/internal/infra/persistence/query_builders/employee_project_query_builder.go`
- `backend/internal/infra/persistence/query_builders/employee_project_query_builder_test.go`
- `backend/internal/app/services/employee/service.go`
- `backend/internal/app/services/employee/bank_validation.go`
- `backend/internal/app/services/employee/bank_validation_test.go`
- `backend/internal/app/services/bcc_import_service.go`
- `backend/internal/app/services/bcc_import_weekly.go`
- `backend/internal/app/services/bcc_import_multi_position.go`
- `backend/internal/app/services/bcc_import_helpers_test.go`
- `backend/internal/app/services/bcc_import_service_test.go`
- `backend/internal/app/services/bcc_import_weekly_test.go`

## Success Criteria

- [x] Every missing required bank-field permutation appears in the unified list.
- [x] Complete confirmed-invalid rows appear with their persisted reason.
- [x] Complete `unverified` rows do not appear.
- [x] BCC creates invalid-bank employees and continues timesheet import.
- [x] Manual invalid create/update still rejects without persistence.
- [x] No schema or public response contract changes.
