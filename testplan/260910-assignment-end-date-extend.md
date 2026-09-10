# Test Plan: Admin can extend or clear project assignment end date

Ticket: `eda63892` — relax paid-timesheet update guard in `ValidateAssignmentForUpdate`.
Rule under test: an update may only pull a boundary inward when the dropped window holds
no approved/paid timesheet; extending or clearing the end date never blocks;
same-project overlap check (with timesheets on the other assignment) unchanged;
assignments without paid timesheets keep all current edit freedoms.

Fixture reference (dev): employee 966, assignment 1137, project LX001 — 77
approved+paid timesheets all ≤ 2026-08-26, weekly pay, window 2026-07-02 → 2026-09-09
(now open-ended via one-off SQL; this feature productizes it).

Written BEFORE any verification run (feedback_testplan_before_testing).

## A. Unit tests — `backend/internal/domain/services/employee_assignment_service_test.go` (new)

### A1. Pure guard `ValidateAssignmentUpdateWithData` (no repo deps)
| # | Case | Inputs | Expected |
|---|------|--------|----------|
| U1 | Extend end, paid ts exist | both window flags false | nil (allowed) |
| U2 | Clear end (nil LastDate), paid ts exist | flags false | nil |
| U3 | Shrink end into paid range | endWindowHasPaid=true | ValidationError, `MsgAssignmentUpdateExcludesPaidTimesheetsVN` |
| U4 | Start moved later into paid range | startWindowHasPaid=true | ValidationError, same message |
| U5 | Start window dirty blocks even if end clean | start=true, end=false | ValidationError |
| U6 | Overlap: same project, other assignment has timesheets | overlapping list + count>0 | ConflictError `MsgAssignmentOverlapVN` |
| U7 | Overlap: different project, other has timesheets | other.ProjectID ≠ | nil (cross-project overlap OK) |
| U8 | Overlap: same project, other has NO timesheets | count=0 | nil |
| U9 | Overlap hit is the assignment itself | other.ID == assignment.ID | nil |
| U10 | No paid ts anywhere (fast path) | flags false, no overlaps | nil |

### A2. Window arithmetic in `ValidateAssignmentForUpdate` (fakes capture query bounds)

**Semantics note (resolved during implementation):** the end-side guard implements the PM
spec literally — *no approved/paid timesheet may fall outside the resulting window* — so ANY
finite new end (shrunk, unchanged, or extended) queries `[newEnd+1d, ∞)`; clearing the end
(nil) never queries. The start side stays delta-scoped (`[oldStart, newStart-1d]`) because
paid work of *earlier* assignments legitimately sits before this assignment's start.
Consequence (accepted by spec): an extension that stops short of an existing paid timesheet
beyond the old end is rejected until the admin extends past it or clears the end.

| # | Case | old → new | Expected range passed to `HasNonEditableTimesheetsInRange` |
|---|------|-----------|----------------------------------------------------------|
| W1 | Start moved later (end cleared to isolate) | start 07-02 → 08-01 | exactly [07-02, 07-31] (newStart exclusive) |
| W3 | Start moved earlier (end cleared) | start 07-02 → 06-15 | NO query |
| W4 | End shrunk | last 09-09 → 08-31 | [09-01, nil] (unbounded upper) |
| W5 | End extended | last 09-09 → 12-31 | [2027-01-01, nil] — extension still verified |
| W6 | End cleared | last 09-09 → nil | NO end query |
| W7 | End unchanged (finite) | last 09-09 → 09-09 | [09-10, nil] |
| W8 | Open-ended shrunk to finite | last nil → 08-31 | [09-01, nil] |
| W9 | Error propagation | checker returns error | error surfaces (not swallowed) |
| W10 | Paid ts in dropped start window | W1 bounds, checker=true | ValidationError |
| W11 | Both boundaries shrink clean | start →07-15, end →08-31 | exactly 2 queries: [07-02, 07-14] + [09-01, nil] |
| W12 | Insufficient extension (paid ts beyond new end) | last 09-09 → 09-30, checker=true | ValidationError (AC3) |

### A3. Existing validation preserved
| # | Case | Expected |
|---|------|----------|
| V1 | last_date < start_date | ValidationError (ValidateDates, unchanged VN message) |
| V2 | Employee not found | NotFoundError |

## B. Integration — extend `backend/tests/integration/flow_project_crud.go`
Fixtures created by the flow itself (fresh employee, assignment 2026-05-01 open-ended);
approved+paid timesheet rows planted via `docker exec payroll-mysql` (db_helper helpers
`plantPaidTimesheet` / `deleteTimesheetsForPair` / `getAssignmentDateColumn`); DB state
asserted on disk after every case, including after every rejection.
| # | Case | Steps | Expected |
|---|------|-------|----------|
| I1 | Setup | create employee, assign (start 2026-05-01), plant PAID ts 2026-06-15 | assignment id > 0 |
| I2 | Extend end over paid ts | PUT end_date 2026-08-31 | 200; API + DB last_date=2026-08-31 |
| I3 | Boundary inclusivity | plant PAID ts exactly 2026-07-10, PUT end_date 2026-07-10 | 200 (ts ON end date stays covered); DB last_date=2026-07-10 |
| I4 | Insufficient extension (must-fix regression) | plant PAID ts 2026-08-20, PUT end_date 2026-08-15 | 4xx; DB last_date still 2026-07-10 |
| I5 | Shrink stranding paid ts | PUT end_date 2026-06-01 | 4xx; DB last_date still 2026-07-10 |
| I6 | Clear end over paid ts | PUT end_date "" | 200; API last_date null; DB NULL |
| I7 | Start move stranding paid ts | PUT start_date 2026-06-20 (drops strip with ts 06-15) | 4xx; DB start_date still 2026-05-01 |
| I8 | Create-overlap regression (AC9) | POST re-assign same employee start 2026-06-01 over open assignment with ts | 409 |
| I9 | Cleanup | delete planted ts, hard-delete assignment row, delete employee | clean |

## C. Regression gates
- `cd backend && go build ./...`
- `go test ./internal/domain/services/... -race`
- `go test ./... -race` (full unit suite)
- `make api-test` (all flow files incl. project CRUD + employee CRUD)

## D. Out of scope for this plan
- Frontend UI verification (localhost:3000) — owned by FullStack/frontend step.
- Prod data (no prod DB writes ever).

## Results
- 2026-09-10 unit: `go test ./... -race -count=1` — PASS (full backend suite, incl. new
  TestValidateAssignmentForUpdate table [9 cases], TestValidateAssignmentForUpdateWindowBounds
  [12 subtests], TestValidateAssignmentUpdateWithDataOverlapVariants, BaseValidation).
- Architect must-fix (unbounded end-guard) verified by W5/W7/W12 + integration I4.
- Mock blocker resolved: `mocks/mock_timesheet_repo.go` already carries
  HasNonEditableTimesheetsInRange (regenerated by fullstack).
- Integration `make api-test`: 3 consecutive runs. Run 1: 298/324 passed, 3 failed (names
  lost to log truncation; cause judged environmental — FullStack was concurrently doing live
  UI verification against the same dev backend; several discovery tests assert global counts).
  Runs 2 and 3: **Total 324 | Passed 301 | Failed 0 | Skipped 23** each — stable full pass,
  including all 9 new paid-assignment cases (I1–I9).
- Rejections verified by status AND on-disk DB state (last_date/start_date unchanged after
  each 4xx); successes verified in API response AND DB.
- Incident fixture: assignment 1137 (project 19, employee 966) confirmed OPEN-ended in dev
  after all runs — user's chosen final state preserved.
