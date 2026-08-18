---
phase: 3
title: "Validation and release readiness"
status: pending
priority: P1
effort: "2h"
dependencies: [1, 2]
---

# Phase 3: Validation and release readiness

## Overview

Prove the new visibility semantics end-to-end (including partner scoping and
all four bảng công pages), run the standard regression gates, and update
docs where the endpoint contract changed.

## Requirements

- Functional:
  - E2E-style verification on local dev (`localhost:3000`, admin `frankng` /
    partner `cuongnv`, password `Admin123`): seed an invalid-status employee
    with zero pending work → warning section visible on all four pages with
    the "Sai thông tin" badge.
  - Partner login sees only accessible employees (cross-partner invalid
    employee NOT visible).
- Non-functional:
  - `make api-test` green (30 flow files).
  - `cd backend && go test ./... -race -cover` green for touched packages.
  - `cd frontend && pnpm lint && pnpm type-check` green.
  - Docs updated only where the contract changed: `docs/api.md` row field for
    `missing-bank-details` (add `bank_warning_kind`), plus a line in the
    endpoint description stating invalid rows are not gated on pending work.

## Related Code Files

- Modify: `docs/api.md` (missing-bank-details response field + semantics)
- No code changes expected in this phase — fixes only if gates fail.

## Implementation Steps

1. Backend gates:
   - `cd backend && go build ./... && go test ./internal/infra/persistence/query_builders/... ./internal/transport/http/handlers/employee/... -race`
   - `make api-test`
2. Frontend gates:
   - `cd frontend && pnpm lint && pnpm type-check`
   - `pnpm test -- MissingBankDetailsSection bank-information-warning` (or
     repo-equivalent vitest invocation)
3. Manual verification matrix (local dev, `make dev`):
   | Step | Role | Page | Expectation |
   |------|------|------|-------------|
   | Seed employee E1: status=invalid, all fields set, active project, zero pending | admin | /admin/timesheet (desktop) | Section shows E1, "Sai thông tin" |
   | same | admin | mobile admin timesheet | same |
   | same | partner (owning E1's project) | /partner/timesheets | Section shows E1 |
   | Seed E2: invalid, `last_date` set | admin | timesheet | E2 absent |
   | Seed E3: missing bank no, zero pending | admin | timesheet | E3 absent |
   | Cross-partner E4: invalid, other partner's project | partner | timesheet | E4 absent |
4. Update `docs/api.md` for the response field and semantics sentence.
5. Commit (conventional, no AI references):
   - `feat(employee): always list invalid bank accounts on bang cong warning`
   - Single commit is fine — backend + frontend ship together (deploy-order
   safe per phase 2 fallback).

## Success Criteria

- [x] All gates green (api-test, go test -race, lint, type-check)
- [x] Manual matrix 100% — all six rows verified
- [x] docs/api.md updated
- [x] Committed on `main` per repo convention

## Risk Assessment

- **Pre-existing invalid rows flooding the list on release**: every employee
  ever marked invalid with no pending work appears at once. This is the
  intended "always display" behavior per the user's decision — surface it in
  the release note so it's not mistaken for a bug.
- **api-test flakiness unrelated to this change**: rerun failing flows once;
  if still failing, verify failure exists on `main` before this change
  (baseline check) before investigating.
- **Prod data shape surprises** (e.g. invalid + empty fields from manual DB
  edits): covered by matrix case 9 in phase 1; UI labels them invalid.
