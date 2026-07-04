---
title: "Backend DB query performance fixes (indexes + N+1 + unbounded)"
description: "Turns the ck:debug Phase-1 root-cause findings (2026-07-04) into ordered, independently-shippable action items. 4 missing indexes (additive, one migration), 2 trivial N+1 swaps to existing batch helpers, 1 isolated loan-schedule batch refactor, and a final phase for the remaining N+1/unbounded/projection cleanups. Root cause is established for every item with file:line evidence; this plan is execution scaffolding, not design. All proposed indexes verified non-duplicative against migrations/*.up.sql."
status: pending
priority: P2
branch: "main"
tags: [performance, database, mysql, indexes, n-plus-1, ck-debug-2026-07-04]
blockedBy: []
blocks: []
created: "2026-07-04T12:19:05.497Z"
createdBy: "ck:plan"
source: skill
---

# Backend DB query performance fixes (indexes + N+1 + unbounded)

## Overview

A `ck:debug` Phase-1 investigation (2026-07-04) found the backend's hot paths
(timesheets, dashboard, payroll bulk report, advance-payment list) are well-
architected — they already use a `TimesheetRelationshipLoader` (parallel errgroup
+ IN), `GetByIDs` batch helpers, and Preload. **The problems are elsewhere**:
4 missing indexes, 3 N+1 loops, 2 unbounded queries, and a few wide projections.

This plan converts those findings into **4 independently-shippable batches**.
Each batch can land, build, test, and deploy on its own — no cross-phase
dependencies. The phases are ordered by impact-per-effort, not by hard sequence.

### Grouping rationale (why these 4 batches)
- **Phase 1 — Indexes:** pure additive SQL, zero code risk, fixes the two
  High-severity full-scan endpoints. Ships first, biggest bang-for-buck.
- **Phase 2 — Trivial N+1 swaps:** the two findings that reuse *existing* batch
  helpers (`ProjectRepository.GetByIDs`, removing redundant `GetByID`). One-line
  to few-line changes, no new repo methods.
- **Phase 3 — Loan-schedule batch refactor:** the one finding that needs a *new*
  repo method (`GetByLoanIDs`). Isolated so the new helper gets its own review.
- **Phase 4 — Remaining cleanups:** the advance-payment sao-ke N+1, unbounded
  employee/user queries, the wide-projection fix, and the latent dead-code trap.
  Lower individual impact; batch them to amortize the build/test cycle.

## Phases

| Phase | Name | Effort | Ships independently? |
|-------|------|--------|----------------------|
| 1 | [Migration 088 indexes](./phase-01-migration-088-indexes.md) | XS (1 migration) | ✅ Yes |
| 2 | [Trivial N+1 swaps](./phase-02-trivial-n-1-swaps.md) | XS (~3 lines) | ✅ Yes |
| 3 | [Loan-schedule batch refactor](./phase-03-loan-schedule-batch-refactor.md) | S (1 repo method + handler) | ✅ Yes |
| 4 | [Remaining N+1 + unbounded + projections](./phase-04-remaining-n-1-unbounded-projections.md) | M (several small fixes) | ✅ Yes |

## Findings → phase map

| Finding | Severity | Phase |
|---------|----------|-------|
| #2 `audit_logs.entity_id` missing index | 🔴 High | 1 |
| #3 `transactions.reversed_transaction_id` missing index + NOT IN→NOT EXISTS | 🔴 High | 1 |
| #7 `users.last_login` missing index | 🟡 Medium | 1 |
| #8 `transactions (status, created_at)` composite missing | 🟡 Medium | 1 |
| #5 payroll-report-by-project N+1 → swap to `ProjectRepository.GetByIDs` | 🟡 Medium | 2 |
| #1 loan-list N+1 (schedules per loan) → new `GetByLoanIDs` | 🔴 High | 3 |
| #6 advance-payment sao-ke N+1 | 🟡 Medium | 4 |
| #4 `GetActiveEmployees` unbounded | 🔴 High | 4 |
| #9 `FindUsersWithNullLastLogin` unbounded+unindexed | 🟡 Medium | 4 |
| #10 `GetByIDsWithoutRelations` SELECT * | 🟡 Medium | 4 |
| dead-code `ResponseBuilder.BuildDetailedEmployeeResponse` triple-N+1 | 🟢 Low | 4 |
| transactions export hardcoded Limit:10000 + optional date range | 🟢 Low | 4 |

## Constraints

- Latest migration is `087` (OTP lockout). New migration = **088**.
- Every proposed index was verified absent from `migrations/*.up.sql` by the
  audit (none duplicate existing indexes — including the audit_logs revamp in
  migration 034 that *dropped* the entity_id index in 008 and never restored it).
- Existing batch helpers available to reuse: `EmployeeRepository.GetByIDs`,
  `ProjectRepository.GetByIDs` (returns `map[uint]*Project`),
  `ProjectEmployeeRepository.GetActiveAssignmentsByProjectsAndEmployees`,
  `LoanRepaymentScheduleRepository.ListByTransactionIDs` (the pattern to mirror
  for the new `GetByLoanIDs`).
- Pre-existing test failures (`persistence`, `ninepay`, `bootstrap-repositories`)
  need a live DB and are **unrelated** to these changes — confirmed by stash test
  in the security-fix work. Don't gate on them.

## Verification posture

- `go build ./...` + `go vet ./...` clean after each phase.
- `go test ./internal/infra/persistence/loan_repayment_schedule_repository_test.go`
  for Phase 3's new helper (existing test file already covers this repo).
- `make api-test` per AGENTS.md before deploy (integration tests).
- Manual spot-check: `EXPLAIN` the affected queries before/after index migration
  to confirm the optimizer picks the new index.

## Out of scope

- WebAuthn, OTP, security-remediation work (separate plans).
- Larger architectural refactors (e.g. read-replica routing, materialized views).
- The inner per-project timesheet query in `payroll_report_by_project_service`
  (genuinely variable per-project date ranges — flagged as a secondary concern,
  not a clean N+1).
