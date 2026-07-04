---
phase: 3
title: "Loan-schedule batch refactor"
status: pending
priority: P1
effort: "S (1 new repo method + handler wiring)"
dependencies: []
---

# Phase 3: Loan-schedule batch refactor

## Overview
The High-severity loan-list N+1: `GET /loans` (page up to 100) fires
`RepaymentScheduleRepo.GetByLoanID` once per loan → up to 100 extra queries.
Add one batch helper (`GetByLoanIDs`) mirroring the existing
`ListByTransactionIDs` pattern, and rewire the list handler to use it.

## Requirements
- Functional: loan list endpoint loads all schedules in one query (plus the loan
  list query), not N+1.
- Non-functional: identical response shape; the single-loan detail path is
  unchanged (still uses `GetLoanWithSchedules`).

## Architecture
Mirror `LoanRepaymentScheduleRepository.ListByTransactionIDs(ctx, transactionIDs)`
(`loan_repayment_schedule_repository.go:85`) — it already demonstrates the exact
batch pattern this repo uses. New method:
```go
func (r *LoanRepaymentScheduleRepository) GetByLoanIDs(ctx context.Context, loanIDs []uint) (map[uint][]*domain.LoanRepaymentSchedule, error)
```
Handler collects `loanIDs` from the loaded `loans` slice, calls `GetByLoanIDs`
once, then passes the map into a new `buildLoanResponses(ctx, loans, scheduleMap)`
that does pure map lookup per loan.

## Related Code Files
- **Modify:** `internal/infra/persistence/loan_repayment_schedule_repository.go` — add `GetByLoanIDs`
- **Modify:** `internal/domain/loan_repayment_schedule_repository.go` (interface) — add the method to the interface
- **Modify:** `internal/transport/http/handlers/loan/loan.go` — new `buildLoanResponses` batch builder; `ListLoans` handler uses it
- **Create:** `internal/infra/persistence/loan_repayment_schedule_repository_test.go` — extend with a `GetByLoanIDs` test (or add to the existing test file if present)

## Implementation Steps
1. **Add `GetByLoanIDs` to the interface** (`domain/loan_repayment_schedule_repository.go`).
2. **Implement** in `loan_repayment_schedule_repository.go`:
   ```go
   func (r *LoanRepaymentScheduleRepository) GetByLoanIDs(ctx context.Context, loanIDs []uint) (map[uint][]*domain.LoanRepaymentSchedule, error) {
       if len(loanIDs) == 0 { return map[uint][]*domain.LoanRepaymentSchedule{}, nil }
       var schedules []*domain.LoanRepaymentSchedule
       if err := r.DB.WithContext(ctx).Where("loan_id IN ?", loanIDs).Order("loan_id, due_date").Find(&schedules).Error; err != nil {
           return nil, err
       }
       out := make(map[uint][]*domain.LoanRepaymentSchedule, len(loanIDs))
       for _, s := range schedules { out[s.LoanID] = append(out[s.LoanID], s) }
       return out, nil
   }
   ```
   (Mirror the column name + ordering from the existing `GetByLoanID`.)
3. **Add `buildLoanResponses(ctx, loans, scheduleMap)`** in `loan.go` — iterate
   loans, look up schedules from the map, build `[]dto.LoanResponse`. Reuse the
   per-loan field-mapping logic from the existing `buildLoanResponse` to avoid
   drift (extract the field-mapping into a `populateLoanResponse(loan, schedules)`
   helper that both the single and batch paths call).
4. **Rewire `ListLoans`** (`loan.go:325`) to collect `loanIDs`, call
   `GetByLoanIDs`, then `buildLoanResponses`.
5. **Test:** add a `GetByLoanIDs` repo test (round-trip: create 3 loans with
   schedules, batch-fetch, assert map shape). If an integration test for the
   loan list endpoint exists, confirm it still passes.

## Success Criteria
- [ ] `GetByLoanIDs` exists and returns the expected map shape (empty-input returns empty map, no error).
- [ ] `GET /loans` (page=1, pageSize=100) issues exactly 1 schedules query (verify via GORM Info logs or a query counter).
- [ ] Single-loan detail path (`GetLoanWithSchedules`) is unchanged.
- [ ] `go build ./... && go vet ./...` clean.
- [ ] New repo test passes; existing loan tests pass.

## Risk Assessment
- **Risk:** schedule ordering changes (batch `ORDER BY loan_id, due_date` vs
  per-loan `ORDER BY due_date`). **Mitigation:** ensure the per-loan slice is
  sorted by `due_date` within each map entry (the `ORDER BY loan_id, due_date`
  already guarantees this).
- **Risk:** empty `loanIDs` slice → `WHERE loan_id IN ()` SQL error. **Mitigation:**
  guard with the early-return shown in the implementation.
- **Risk:** response-builder drift between single and batch paths. **Mitigation:**
  extract the shared `populateLoanResponse` helper so both call the same code.
