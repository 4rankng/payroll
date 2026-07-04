---
phase: 4
title: "Remaining N+1 + unbounded queries + projections"
status: pending
priority: P2
effort: "M (several small, isolated fixes)"
dependencies: []
---

# Phase 4: Remaining N+1 + unbounded queries + projections

## Overview
The remaining cleanups batched to amortize the build/test cycle: one Medium
N+1 (advance-payment sao-ke), two unbounded-query caps, one wide-projection fix,
one `NOT IN → NOT EXISTS` rewrite (uses the Phase-1 index), the export date-range
guard, and the dead-code trap cleanup.

## Requirements
- Functional: each finding addressed per the ck:debug report.
- Non-functional: no behavior change for legitimate callers; caps only bound
  worst-case query size.

## Architecture
Each fix is isolated to one file/service. No new tables, no new indexes
(Phase 1 already added the indexes these fixes rely on).

## Related Code Files
- **Modify:** `internal/app/services/advance_payment/reconcile_service.go` (N+1 #6)
- **Modify:** `internal/infra/persistence/employee_repository_queries.go:301` (unbounded #4)
- **Modify:** `internal/infra/persistence/user_repository.go:247,388` (unbounded #9 + ListByRole cap)
- **Modify:** `internal/infra/persistence/repositories/timesheet_query_repository.go:81` (SELECT * #10)
- **Modify:** `internal/infra/persistence/transaction_repository.go:210,387` (NOT IN → NOT EXISTS)
- **Modify:** `internal/transport/http/handlers/transaction.go:667` (export date-range guard)
- **Delete or guard:** `internal/transport/http/handlers/employee/response_builder.go:95` (`BuildDetailedEmployeeResponse` — dead code, triple N+1)

## Implementation Steps
1. **#6 — advance-payment sao-ke N+1** (`reconcile_service.go:378-419`):
   pre-collect unique `employeeIDs` + `projectIDs` from `paidRequests` before
   the loop; call `EmployeeRepository.GetByIDs` + `ProjectEmployeeRepository.GetActiveAssignmentsByProjectsAndEmployees`
   once each; index results into maps; the loop body becomes pure map lookups.
2. **#4 — `GetActiveEmployees` unbounded** (`employee_repository_queries.go:301`):
   add `.Limit(common.DefaultMaxResults)` (the constant already used by
   `project_repository.go:252,263` for the same purpose). If callers need full
   enumeration, add a paginated variant — but cap the default to prevent DoS.
3. **#9 — `FindUsersWithNullLastLogin` + `ListByRole`** (`user_repository.go:388,247`):
   add `.Limit(common.DefaultMaxResults)` to both. (Index added in Phase 1.)
4. **#10 — wide projection** (`timesheet_query_repository.go:81`):
   add an explicit `.Select("id, project_id, employee_id, date, timesheet_status, payment_status, amount, paid_amount, paid_at")`
   to `GetByIDsWithoutRelations` (validation-only caller doesn't need the rest).
5. **NOT IN → NOT EXISTS** (`transaction_repository.go:210,387`):
   rewrite `AND id NOT IN (SELECT reversed_transaction_id FROM transactions WHERE reversed_transaction_id IS NOT NULL)`
   to `AND NOT EXISTS (SELECT 1 FROM transactions r WHERE r.reversed_transaction_id = transactions.id)`
   so the Phase-1 `idx_transactions_reversed_txn_id` index is actually used.
6. **Export date-range guard** (`handlers/transaction.go:667`): reject export
   requests missing `fromDate` with a 400 (currently allows unbounded 10k export).
7. **Dead-code trap** (`employee/response_builder.go:95`): grep confirms
   `BuildDetailedEmployeeResponse` has no callers. Delete it (and `BuildEmployeeResponse`
   if also uncalled) — a triple-N+1 landmine. If a future feature needs it,
   write the batch variant first.
8. Build + vet + run affected endpoints.

## Success Criteria
- [ ] `reconcile_service.go` no longer queries per (employee, project) pair in the loop.
- [ ] `GetActiveEmployees`, `ListByRole`, `FindUsersWithNullLastLogin` all capped.
- [ ] `GetByIDsWithoutRelations` uses an explicit column list.
- [ ] Transaction list/count uses `NOT EXISTS` and the Phase-1 index (EXPLAIN confirms).
- [ ] Transactions export requires a `fromDate`.
- [ ] `BuildDetailedEmployeeResponse` removed (or guarded with a batch variant).
- [ ] `go build ./... && go vet ./...` clean; `make api-test` green.

## Risk Assessment
- **Risk:** `GetActiveEmployees` cap breaks a caller that expects full enumeration.
  **Mitigation:** grep callers first; if a caller genuinely needs all, give it a
  paginated variant rather than silently truncating. `common.DefaultMaxResults`
  is generous (check its value — if 1000+, the cap is a safety net not a real
  bound for legitimate use).
- **Risk:** `NOT IN → NOT EXISTS` changes NULL semantics. **Mitigation:** the
  subquery already filters `WHERE reversed_transaction_id IS NOT NULL`, so NULLs
  are excluded either way; `NOT EXISTS` is semantically equivalent here.
- **Risk:** deleting `BuildDetailedEmployeeResponse` breaks something the grep
  missed. **Mitigation:** use `git grep -n BuildDetailedEmployeeResponse` across
  the whole repo (including tests) before deleting; if any caller exists, gate
  it instead.
