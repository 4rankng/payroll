# Plan: Extend/reopen assignment end date (admin)

## Outcome
Admins can extend or clear an assignment's end date (Ngày kết thúc) from the
employee-details UI, even when the assignment has approved/paid timesheets.
Employee 966's situation (came back to LX001, old assignment ended 09-09 with 77
paid timesheets) must be fixable through the UI instead of raw SQL.

## Context (investigated 2026-09-10)
- Backend endpoint `PUT /employees/:id/projects` + `PUT /projects/:id/employees`
  already parse `end_date` (`""` → clear, date → set) — handler
  `employee.go:708-800`, `project_employee_batch.go:444-459`.
- Sole backend blocker: `ValidateAssignmentForUpdate`
  (`domain/services/employee_assignment_service.go:55-99`) blocks ANY update when
  `HasNonEditableTimesheetsAfterDate(start)` is true — 77 approved+paid
  timesheets ≥ 07-02 means everything is blocked.
- Frontend: `EmployeeProjectSection.tsx` edit mode has no end-date field;
  `EmployeeViewDetails.tsx:67` renders the section without `onProjectApply`, so
  Apply silently discards edits (mutation + service exist unused:
  `useUpdateEmployeeProjectAssignment` → `PUT /employees/:id/projects`).

## Invariant (the rule that replaces the blunt guard)
Every approved/paid timesheet covered by the old period must stay covered by the
new period. Only the boundary that actually shrinks is checked:
- Start moved later → block if approved/paid ts in `[oldStart, newStart)`
- End shrunk → block if approved/paid ts in `(newLast, oldLast]` (oldLast nil ⇒ unbounded)
- Extensions/clearing never block; overlap-with-other-assignments rule unchanged.

## Changes
1. `domain/timesheet_types.go` — add `HasNonEditableTimesheetsInRange(from,to *time.Time)` to `TimesheetAssignmentChecker`.
2. `infra/persistence/timesheet_repository.go` — implement the range check (approved OR paid, nil bounds = unbounded).
3. `domain/services/employee_assignment_service.go` — rework update validation to
   delta-scoped guards; load old assignment via repo.GetByID; new pure
   `ValidateAssignmentUpdateWithData(assignment, old, overlapping, counts, startWindowPaid, endWindowPaid)`.
4. `constants/messages.go` — VN message for "cannot drop paid timesheets from period".
5. `EmployeeProjectSection.tsx` — end-date input in edit mode (empty = open-ended,
   helper text), `end_date` in `ProjectChange`, read-mode "Đang làm việc".
6. `EmployeeViewDetails.tsx` — wire `onProjectApply` →
   `useUpdateEmployeeProjectAssignment` (send `end_date: ''` to clear, never null).

## Validation
- New `employee_assignment_service_test.go` — 8 cases (extension/clear allowed
  with paid ts, shrink blocked, clean shrink allowed, start-later blocked,
  start-earlier allowed, overlap rule preserved, set-end-on-open blocked over paid ts).
- `go test ./internal/domain/services/ -race`; `go build ./...`; gofmt.
- Frontend: `npx tsc -p tsconfig.app.json --noEmit` (real gate), `pnpm lint`.
- Live E2E vs running dev backend (employee 966, project 19): extend → 200,
  clear → 200, shrink into paid window → domain error, DB transitions verified.
- Final state left: assignment 1137 open-ended (the user's chosen outcome).

## Non-goals
- No change to create-flow overlap rules (409 on backdated same-project overlap stays).
- No auto-close-on-reassign (option B rejected by user).
- No assignment edit UI on the project side list (`EmployeeProjectsList`) — same
  section component reused later if wanted.

## Validation results (2026-09-10, post-implementation)
- Unit: 9/9 subtests in `employee_assignment_service_test.go` pass; full backend
  suite `go test ./... -race` green; gofmt/vet clean on touched files.
- Frontend: tsc (`tsconfig.app.json`) + eslint clean on touched files.
- Live E2E vs dev backend (frankng), final matrix: set end on open → success;
  shrink into paid window → blocked (new VN message); extend → success; clear →
  success; wrong-owner assignment_id → 404. DB transitions verified after each
  step. Final state: 1137 open-ended, 1069 restored to 06-23→07-01.

## Findings during implementation (2 pre-existing bugs fixed)
1. `GetByProjectAndEmployee` `First()` returned the OLDEST assignment for a
   (project, employee) pair — employee-details edits targeted the June row
   instead of the current one. Fixed via `assignment_id` in
   `UpdateAssignmentRequest` + ownership guard + frontend wiring.
2. DATE-boundary tz exclusion under `loc=Local` (UTC-midnight parse → 07:00
   local) made `date >= ?` exclude same-day rows. Fixed with `DATE(?)` bounds.
   Note: a parallel session converged on the same feature and merged its
   stricter tail-window variant of the guard into this tree; that variant is
   what ships (any finite end must leave no paid timesheet uncovered).
