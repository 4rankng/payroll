---
phase: 6
title: "Tests + build gates"
status: pending
priority: P2
dependencies: [1, 2, 3, 4, 5]
---

# Phase 6: Tests and build gates

## Overview
Run the full test/lint/build matrix required by `AGENTS.md` and add integration-test scenarios for the new feature path, so the change ships behind the repo's standard gates.

## Requirements
- Functional: every new code path has a focused test; existing suites stay green.
- Non-functional: no drop in coverage on touched files.

## Architecture
No new infra — uses the repo's existing test commands.

## Related Code Files
- Add/extend: `backend/internal/domain/project_test.go` — `ValidateShiftNames` table tests.
- Add/extend: `backend/internal/app/services/attendance/attendance_service_test.go` — `ResolveAllShiftWindows` attaches names by range key.
- Add/extend: `backend/tests/integration` — project create/update with `shift_names`; employee profile surfaces `shift_name`.
- Add/extend: `frontend/src/components/employees/EmployeeCheckInCard.test.tsx` — named/fallback/overnight cases (phase 5).

## Implementation Steps
1. Backend unit: domain validation table test (all branches from phase 1).
2. Backend unit: `ResolveAllShiftWindows` with a `names` map returns `ShiftWindow.Name` set per range; unknown ranges get empty name.
3. Backend integration: `POST /projects` with `shift_names` persists and returns; `PUT /projects/:id` validates range mismatch; `GET /auth/me` as employee returns `schedule_windows[].shift_name`.
4. Frontend: update `EmployeeCheckInCard.test.tsx` per phase 5.
5. Run gates:
   - `cd backend && go test ./... -race -cover`
   - `make api-test`
   - `cd frontend && pnpm lint && pnpm type-check && pnpm test run && pnpm build`

## Success Criteria
- [ ] `go test ./... -race -cover` green; new domain + service tests pass.
- [ ] `make api-test` green.
- [ ] `pnpm lint`, `pnpm type-check`, `pnpm test run`, `pnpm build` all green.
- [ ] No new lint warnings in touched files.

## Risk Assessment
- **Risk:** Integration tests need a payrate + project + employee fixture. **Mitigation:** reuse the existing attendance integration fixtures that already set up flexible projects with payrates.
