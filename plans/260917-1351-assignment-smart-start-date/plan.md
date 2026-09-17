---
title: "assignment-smart-start-date"
description: "Auto-set project-assignment start_date from the employee's last recorded timesheet (fallback: 1st of current month) across all assignment-creation paths, plus a self-healing backdate inside BCC imports, so timesheet uploads for past dates are never rejected by ValidateEmployeeAssignment."
status: completed
priority: P1
effort: "2d"
tags: [backend, frontend, timesheet, assignment, bcc-import]
created: 2026-09-17
---

# assignment-smart-start-date

## Overview

Prod incident 2026-09-17 (project CBS/65): a partner manually created an employee
and assignment at 10:02; the UI sent `start_date = today (09-17)`. The weekly BCC
upload `LUONGTUAN_CBS_KY2-09_v1.xlsx` covering Sept 10–14 was then rejected 8× by
`ValidateEmployeeAssignment` ("thời gian phân công bắt đầu sau ngày chấm công"),
each time rolling back 101 valid rows with a generic error naming no employee or
date. The partner eventually self-recovered by backdating the assignment to 09-09
at 12:42; the 12:43 re-upload succeeded (110 rows).

Root causes addressed by this plan:

1. **Assignment start defaults are wrong for payroll reality.** Backend defaults
   missing `start_date` to today (`project_employee_batch.go:116`), and the
   frontend pre-fills today and sends it explicitly
   (`AddEmployeesToProject.tsx:167,173`, `ProjectAssignmentSheet.tsx:49`). BCC
   files always cover past dates.
2. **Imports don't self-heal.** When a file proves an employee worked before the
   assignment start, the import fails wholesale instead of aligning the
   assignment with the evidence.

Design decision (user-approved 2026-09-17): combine smart default
(last-timesheet + 1 day), month-start fallback, and self-healing import; apply to
ALL assignment-creation paths.

## The Rule

```
SuggestAssignmentStart(employeeID, now in Asia/Ho_Chi_Minh):
    if lastTS := timesheetRepo.GetLatestDateByEmployee(employeeID); lastTS != nil:
        return startOfDay(lastTS + 1 day)        # no cross-project day overlap
    return firstOfMonth(now)                     # absorbs same-month BCC files
```

- Explicit `start_date` provided by a user/import row still wins (contract
  unchanged); the rule only replaces *defaults*.
- Timesheet dates are always ≤ today (future dates rejected by validation), so
  `lastTS + 1` never lands beyond tomorrow.

**Self-heal rule (BCC imports):** before building entries, for each employee in
the file whose earliest entry date `E` precedes their assignment `StartDate`,
backdate `StartDate` to `E` in its own committed transaction, then invalidate the
`assignment:{projectID}:{employeeID}` cache **after commit** (ADR-007). Backdating
only *extends coverage backward* — it touches no existing timesheet rows and
violates no invariant from the end-date plan (`260910-1435-assignment-end-date-
extend`), whose guards only protect *shrinking* boundaries.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | New assignments never default to a start date that blocks same-month timesheet uploads | P1 |
| 2 | An employee's assignments across projects never overlap day coverage | P2 |
| 3 | BCC imports self-heal assignment start dates instead of failing wholesale | P1 |
| 4 | No regression in assignment-update validation (explicit dates still honored) | P1 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Smart-start default across creation paths](./phase-01-smart-start-default.md) | Completed |
| 2 | [Self-healing BCC import backdate](./phase-02-import-self-heal.md) | Completed |
| 3 | [Frontend stops sending today](./phase-03-frontend-defaults.md) | Completed |
| 4 | [End-to-end verification + docs](./phase-04-verify-docs.md) | Completed |

## Success Criteria

- [x] Assigning an employee with no `start_date` (UI, import, auto-create) yields `lastTS+1` or 1st-of-month — never today-when-month-already-started
- [x] Replaying today's incident scenario (create assignment today → upload file covering earlier dates) **succeeds** on both weekly formats and legacy format
- [x] Assignment cache invalidation happens after commit; verified by a same-request re-validation test
- [x] `cd backend && go test ./... -race` green; `make api-test` green; `pnpm lint && pnpm type-check` green
- [x] Explicit `start_date` values are stored unchanged (regression test)

## Out of Scope

- Changing `ValidateEmployeeAssignment` semantics or its error messages
- Fixing the 1,258 historical timesheets-before-start rows (separate ops task, projects 7/10/19/15/17/11/67)
- The per-row error surfacing fix for weekly BCC imports (already fixed on branch `investigate-prod-upload-fail`, see `plans/reports/debug-260917-1301-weekly-bcc-upload-failure.md`)

## Related Work

- Report: `plans/reports/debug-260917-1301-weekly-bcc-upload-failure.md` (incident root cause)
- Related plan (no dependency): `plans/260910-1435-assignment-end-date-extend/` — same validation service, opposite boundary (end-date shrinking guards)
