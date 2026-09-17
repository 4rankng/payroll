---
phase: 1
title: "Smart-start default across creation paths"
status: pending
priority: P1
effort: "6h"
dependencies: []
---

# Phase 1: Smart-start default across creation paths

## Overview

Add the `SuggestAssignmentStart` rule (last-timesheet + 1 day, fallback 1st of
current month) as a shared service and replace the today-default / monthStart-
literal at every `ProjectEmployee` construction site.

## Requirements

- Functional:
  - New port method `GetLatestDateByEmployee(ctx, employeeID) (*time.Time, error)` returns the employee's most recent timesheet date across ALL projects (nil when none).
  - New domain-service helper `SuggestAssignmentStart(ctx, employeeID) time.Time` implementing the rule; all time math via `clock.Now()` in Asia/Ho_Chi_Minh (ADR-006).
  - Every creation path below uses the helper when no explicit start date is supplied.
  - Explicit start dates are stored unchanged.
- Non-functional:
  - Assignment batch of N employees must not run N+1 sequential queries when avoidable — accept per-employee lookup for N≤50 (current UI batch sizes); note follow-up if profiling shows need.

## Architecture

`TimesheetRepository` port (`internal/domain/timesheet_types.go`) gains
`GetLatestDateByEmployee`. GORM impl: `SELECT MAX(date) FROM timesheets WHERE
employee_id = ? AND deleted_at IS NULL`. Domain service lives in
`internal/domain/services/employee_assignment_service.go` (same file as the
update-validation invariants) with repo dependency already available there.

Callers replace their fallbacks:

| Path | File | Current default | New default |
|------|------|-----------------|-------------|
| UI batch assign | `transport/http/handlers/project_employee/project_employee_batch.go:116` | today | `SuggestAssignmentStart` |
| Employee import (row without start) | `app/services/employee/import_service.go:287,466` | today/zero | `SuggestAssignmentStart` |
| BCC legacy STK auto-create | `app/services/bcc_import_stk.go:186` | monthStart | `SuggestAssignmentStart` |
| BCC multi-position auto-create | `app/services/bcc_import_multi_position.go:318` | monthStart | `SuggestAssignmentStart` |
| BCC weekly-BCC auto-create (2 sites) | `app/services/bcc_import_weekly_bcc.go:316,489` | monthStart | `SuggestAssignmentStart` |
| BCC weekly-payment auto-create (2 sites) | `app/services/bcc_import_weekly_payment.go:298,470` | monthStart | `SuggestAssignmentStart` |
| FlexPay get-or-create | `app/services/advance_payment/admin_flexpay_import.go:415,523` | today | `SuggestAssignmentStart` |

## Related Code Files

- Modify: `backend/internal/domain/timesheet_types.go` (port)
- Modify: `backend/internal/infra/persistence/timesheet_repository.go` (GORM impl — locate exact filename; `grep -rl "GetByEmployeeDateCombos" internal/infra/persistence`)
- Modify: `backend/internal/domain/services/employee_assignment_service.go` (helper)
- Modify: the 7 caller files listed above
- Create: `backend/internal/domain/services/employee_assignment_service_test.go` additions or new test file

## Implementation Steps

1. Add `GetLatestDateByEmployee` to the port; implement in the GORM repo (single indexed query on `employee_id, date`).
2. Add `SuggestAssignmentStart(ctx, employeeID)` to `employee_assignment_service.go` with the rule; unit-test: no history → 1st of month; history on month boundary → lastTS+1; lastTS = today → tomorrow; lastTS in previous month → lastTS+1 even if before current month start (correct: coverage continuity beats month alignment).
3. Wire the UI batch-assign handler (replace `timeutil.StartOfDay(h.clock.NowUTC())`).
4. Wire employee import + FlexPay paths (only when row/request omits start date).
5. Wire the 5 BCC auto-create sites (replace `monthStartDate`/`monthStart` literal).
6. Run `go build ./... && go test ./internal/... -race -count=1`.

## Success Criteria

- [x] `SuggestAssignmentStart` unit tests green covering: nil history, mid-month history, history = today, cross-month history
- [x] `grep -rn "NowUTC()" transport/http/handlers/project_employee/` shows no today-default remains for start_date
- [x] All BCC auto-create sites call the helper (no bare `monthStartDate` literals feeding `StartDate`)
- [x] Explicit-date regression test: assignment created with `start_date=2026-09-23` stores 09-23

## Risk Assessment

- **Rule surprises partners who expect "today".** Mitigation: Phase 3 UI shows the computed date with Vietnamese hint text ("Tự động theo ngày chấm công gần nhất" — final copy in phase 3); start date remains editable before save.
- **Cross-month lastTS+1 earlier than month start** (employee idle since August → start 08-xx). Accepted: coverage continuity is the point; audit log records the source (suggested vs explicit).
- If GORM `MAX(date)` scans slow on large `timesheets` table (4M+ rows), signal: import latency > 2s in `make api-test`; response: batch the lookup for BCC paths (single `WHERE employee_id IN (...)` query).
