---
phase: 1
title: "Backend shift-window DTO"
status: pending
priority: P1
effort: "S (0.5d)"
dependencies: []
---

# Phase 1: Backend shift-window DTO

## Overview
Add the resolved shift start/end + valid check-in/check-out window bounds to the
employee profile response, so the frontend can determine "is now a valid
check-in time" without a server round-trip per tap.

## Requirements
- Functional: `EmployeeProfileResponse` gains `shift_start`, `shift_end`,
  `check_in_window_start`, `check_in_window_end` (ISO 8601 "HH:MM" strings),
  plus `next_shift_start` (for the outside-window hint when no shift is active
  today). All nullable — null when no shift is configured for the employee's
  position/project today.
- Non-functional: purely additive (new optional fields); no breaking change to
  existing consumers. The server's `validateCheckInWindow` stays authoritative —
  these fields are informational (display hint), not a new gate.

## Architecture
The shift resolution logic already exists in `AttendanceService`:
- `resolveShifts` (`attendance_service.go:168-234`) parses payrate shift keys
  ("HH:MM-HH:MM") for the employee's position + project.
- `closestShift` (`:236-260`) picks the nearest shift to "now".
- `validateCheckInWindow` (`:98-109`) computes the ±1h window.

This phase reuses that resolution and surfaces it in the profile DTO. The
natural home is `EmployeeScheduleInfo` / `GetEmployeeScheduleInfo`
(`profile_service.go:91-175`), which already resolves `check_in_target`.

### DTO shape (additive)
```go
// In EmployeeProfileResponse or a nested ScheduleInfo struct:
ShiftStart        *string `json:"shift_start,omitempty"`         // "08:00"
ShiftEnd          *string `json:"shift_end,omitempty"`           // "20:00"
CheckInWindowStart  *string `json:"check_in_window_start,omitempty"`  // "07:00" (shift-1h)
CheckInWindowEnd    *string `json:"check_in_window_end,omitempty"`    // "09:00" (shift+1h)
NextShiftStart     *string `json:"next_shift_start,omitempty"`   // for outside-window hint
```
All `*string` (nullable) so employees with no shift configured today get nulls
(the frontend treats null as "no timing gate / always allow").

## Related Code Files
- **Modify:** `backend/internal/app/dto/employee_profile.go` — add the shift-window fields.
- **Modify:** `backend/internal/app/services/employee/profile_service.go` — resolve the shift + window in `GetEmployeeScheduleInfo` (reuse `resolveShifts`/`closestShift` logic from `attendance_service.go`, or call into the attendance service).
- **Modify:** `frontend/src/types/api/auth.types.ts` — add the fields to `EmployeeProfile` / `CheckInTarget`.

## Implementation Steps
1. **Add the DTO fields** to `EmployeeProfileResponse` (or a nested
   `ScheduleWindow` struct on it — prefer the latter to avoid field sprawl).
   All nullable (`*string` or `*Time`).
2. **Resolve the shift** in `GetEmployeeScheduleInfo` (`profile_service.go`):
   - The method already loads the project + assignment. Add a call to resolve
     the payrate shifts for the employee's position + project (reuse the
     attendance service's `resolveShifts` / payrate repository).
   - Pick `closestShift(now)` to get today's applicable shift.
   - Compute `checkInWindowStart = shiftStart - 1h`, `checkInWindowEnd = shiftStart + 1h`.
   - If no shift configured → all fields null.
3. **Populate the DTO** from the resolved shift.
4. **Frontend type** (`auth.types.ts`): add the fields to `EmployeeProfile`.
5. **Test**: verify the profile endpoint returns the shift window for an
   employee with a configured shift, and nulls for one without.

## Success Criteria
- [ ] `GET /mobile/employee-profile` (or the profile endpoint) returns
      `shift_start`, `shift_end`, `check_in_window_start`, `check_in_window_end`
      for an employee with a configured shift.
- [ ] Returns nulls for an employee with no shift configured.
- [ ] `go build ./...` clean; existing attendance tests pass.
- [ ] Frontend type updated; `tsc --noEmit` clean.

## Risk Assessment
- **Risk:** `resolveShifts` lives on `AttendanceService`, not `EmployeeProfileService`.
  **Mitigation:** either inject `AttendanceService` (or the payrate repo) into
  `EmployeeProfileService`, or extract the shift-resolution into a shared helper.
  Prefer the payrate repo (lighter dependency) if the resolution is just parsing
  shift keys.
- **Risk:** shift times are timezone-sensitive. **Mitigation:** use `clock.Now()`
  (Asia/Ho_Chi_Minh, the centralized clock) for "now" and format times in that
  zone. The "HH:MM" format is zone-agnostic for display; the frontend compares
  against the device's local time (also VN for these employees).
- **Risk:** multiple shifts per day (e.g. day + night). **Mitigation:** the
  `closestShift` logic already handles this — pick the nearest one. The DTO
  surfaces the *current* applicable shift only.
