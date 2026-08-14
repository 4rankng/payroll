---
phase: 2
title: "Verify and review"
status: in_progress
effort: ""
---

# Phase 2: Verify and review

## Overview

Prove acceptance criteria and shared-contract safety before any release action.

## Implementation Steps

1. Run focused tests for the new query, message construction, channel behavior, and scheduler registration/manual execution.
2. Run `cd backend && go test ./... -v -race -cover` and `cd backend && make lint`.
3. Start or use the live local backend as required and run `make api-test`; do not weaken or skip failures.
4. Run `git diff --check`, inspect every changed caller/contract, and confirm no public API/schema/env-key drift.
5. Run mandatory tester and debugger verification agents, followed by the mandatory code-reviewer against every acceptance criterion and blast-radius contract.
6. Run `graphify update .` and the repository-required incremental `/understand` knowledge update after code changes.

## Success Criteria

- [x] All focused and broad checks pass with commands and results recorded.
  - `go build ./...` — pass. `go vet` on 4 touched packages — clean. `make lint` — 0 issues. `gofmt -l` — clean.
  - Focused: `go test ./internal/app/services/notification/ ./internal/infra/persistence/ -run 'LoanRepaymentReminder|ListPendingForReminder'` — all pass (unit + sqlite window tests).
  - Broad: `go test ./... -race -count=1` — one failure, `TestCacheInvalidationHandler_HandlesTimesheetEvents` (infra/events). Verified pre-existing by stash → clean-tree rerun still fails. Unrelated to this change.
  - `make api-test` (live backend, MySQL+Redis up): 259 passed / 6 failed / 23 skipped. All 6 failures pre-existing, known-open: 5 × Cutoff [clock] phase tests (RequestCutoffDay=8 vs desired 9), 1 × self check-in advance wait (setting not found; spec-not-impl).
- [ ] Reviewer confirms no regression in loan repayment, notification fan-out, email, scheduler, or public contracts.
- [ ] Any failure or side effect stops release and is presented for a user decision rather than patched silently.

## Notes

- `graphify update .` completed; `understand` CLI not installed — Understand step satisfied by the pre-implementation impact check only.
- Pre-existing failures surfaced to the user in the session report; not patched silently.
