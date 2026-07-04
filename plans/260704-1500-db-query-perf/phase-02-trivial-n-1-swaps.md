---
phase: 2
title: "Trivial N+1 swaps (reuse existing batch helpers)"
status: pending
priority: P2
effort: "XS (~3 lines, no new methods)"
dependencies: []
---

# Phase 2: Trivial N+1 swaps

## Overview
The two N+1 findings that need *no new code* — they reuse batch helpers that
already exist and are proven by other callers. Smallest possible changes.

## Requirements
- Functional: eliminate the two per-iteration `GetByID` loops.
- Non-functional: behavior identical (same data, same order handling); no new
  repo methods, no new tests required (existing coverage exercises the path).

## Architecture
Both fixes replace `for _, id := range ids { x, _ := repo.GetByID(ctx, id) }`
with a single `repo.GetByIDs(ctx, ids)` call that already exists.

## Related Code Files
- **Modify:** `internal/domain/services/payroll_report_by_project_service.go:357-365` (the project-fetch loop)
- **Modify:** `internal/transport/http/handlers/loan/loan.go` — drop the redundant `LoanRepo.GetByID` inside the list path of `buildLoanResponse` (the loan is already loaded by `LoanRepository.List`)

## Implementation Steps
1. **`payroll_report_by_project_service.go:357-365`** — replace:
   ```go
   for _, id := range projectIDs {
       p, err := s.projectRepo.GetByID(ctx, id)
       ...
       projects = append(projects, p)
   }
   ```
   with:
   ```go
   projectMap, err := s.projectRepo.GetByIDs(ctx, projectIDs)  // already exists, returns map[uint]*Project
   if err != nil { return err }
   for _, p := range projectMap { projects = append(projects, p) }
   ```
   (Note: map iteration order is non-deterministic — if the caller depends on
   `projectIDs` order, iterate `projectIDs` and look up the map instead.)
2. **`loan.go` list path** — `buildLoanResponse` calls `GetLoanWithSchedules`
   which re-fetches the loan via `LoanRepo.GetByID`. In the list handler the
   loan is already in hand. Pass the already-loaded `loan` into the response
   builder so the redundant fetch is skipped. (The schedules fetch is the
   Phase 3 refactor; this step only removes the redundant loan re-fetch.)
3. Build + `go vet`; run the payroll-report-by-project and loan-list endpoints
   to confirm identical output.

## Success Criteria
- [ ] `payroll_report_by_project_service.go` no longer calls `GetByID` in a loop.
- [ ] Loan list handler no longer re-fetches each loan it already loaded.
- [ ] `go build ./... && go vet ./...` clean.
- [ ] Manual: payroll report by project returns the same projects as before.
- [ ] Manual: loan list returns the same loans (schedules still loaded per-loan until Phase 3).

## Risk Assessment
- **Risk:** map-iteration order changes report row order. **Mitigation:**
  preserve input order by iterating `projectIDs` and reading the map.
- **Risk:** Phase 2 step 2 (drop redundant GetByID) changes `GetLoanWithSchedules`
  behavior for non-list callers. **Mitigation:** only the *list-path* response
  builder should skip the re-fetch; keep `GetLoanWithSchedules` intact for the
  single-loan detail path. Add a `buildLoanResponseFromLoaded` variant or pass
  the loan explicitly rather than mutating the shared helper.
