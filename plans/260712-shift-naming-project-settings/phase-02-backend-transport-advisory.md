---
phase: 2
title: "Backend: transport + advisory propagation"
status: pending
priority: P1
dependencies: [1]
---

# Phase 2: Backend — transport DTOs, handler wiring, advisory name propagation

## Overview
Expose `shift_names` through create/update project DTOs and handlers (mirroring `geofence_gates`), and thread the admin-chosen name through the advisory chain so the employee profile carries `shift_name` per `schedule_window`.

## Requirements
- Functional: `POST/PUT /projects` accept and return `shift_names`; `GET /auth/me` employee profile's `schedule_windows[].shift_name` is populated when the project has a name for that range.
- Non-functional: additive only — no breaking change to existing DTO fields or attendance validation.

## Architecture
1. **Transport (mirror `GeofenceGates`):** add `ShiftNames []ShiftNameRequest` to `CreateProjectRequest` and `UpdateProjectRequest` (`backend/internal/app/dto/project.go:29-30, 48-50`); add to `ProjectResponse`/`ProjectDetailedResponse`; add `ShiftNameRequest{Range, Name string}` and `shiftNamesToDTO`/`shiftNamesFromDTO` helpers next to the geofence ones in `project.go`.
2. **Handler wiring (`handlers/project/project.go`):** in both `CreateProject` (~line 444) and `UpdateProject` (~line 808), after the geofence block, map `req.ShiftNames` → `project.ShiftNames`; compute the payrate's configured shift ranges (flatten payrate, reuse the `"HH:MM-HH:MM"` key extraction from `resolveShifts`) and call `project.ValidateShiftNames(ranges)`.
3. **Advisory propagation:** add `Name string` to `attendance.ShiftWindow` and `employee.ScheduleWindowInfo`; set it in `ResolveAllShiftWindows` (`attendance_service.go:1368`) by looking up `project.ShiftNames` keyed on the existing `"HH:MM-HH:MM"` string built at line 1375; thread the project (or its `ShiftNames` map) through `populateShiftWindow` → `formatScheduleWindows`. Add `ShiftName string` to `dto.ScheduleWindowInfo` and the mapper in `employee_profile.go:153-168`.

## Related Code Files
- Modify: `backend/internal/app/dto/project.go` — `ShiftNameRequest`, add `ShiftNames` to all 4 project DTO structs (next to `GeofenceGates` lines 29-31, 48-50, 67-69, 167-169).
- Modify: `backend/internal/transport/http/handlers/project/project.go` — DTO mapping (~lines 69-71, 94-96, 148-150) + handler blocks in Create (~444) and Update (~808); add `shiftNamesToDTO`/`shiftNamesFromDTO`.
- Modify: `backend/internal/app/services/attendance/attendance_service.go` — `Name` field on `ShiftWindow` (~line 193), populate in `ResolveAllShiftWindows` (~line 1376). Export a helper `ExtractShiftRanges(flattened) []string` so the handler can validate without reaching into internals.
- Modify: `backend/internal/app/services/employee/profile_service.go` — `Name` on `ScheduleWindowInfo` (~line 60); pass `project.ShiftNames` into `populateShiftWindow`/`ResolveAllShiftWindows`; set in `formatScheduleWindow`.
- Modify: `backend/internal/app/dto/employee_profile.go` — `ShiftName string json:"shift_name,omitempty"` on `ScheduleWindowInfo` (line 44).
- Modify: `backend/internal/transport/http/handlers/employee_profile.go` — map the new field in `mapScheduleWindow` (~line 164).

## Implementation Steps
1. Add DTO fields and request/response types; add the two mapping helpers next to `geofenceGatesToDTO`.
2. In `ResolveAllShiftWindows`, build the range key (already done at line 1375), look up `nameByRange[key]`, and set `ShiftWindow.Name`.
3. Change `ResolveAllShiftWindows` signature to accept a `names map[string]string` (or a `[]domain.ShiftName`) — update its 1 caller (`profile_service.go:250`). Keep `resolveShift`/`resolveShiftForAttendance` untouched (validation path doesn't need names).
4. In `profile_service.populateShiftWindow`, fetch `project.ShiftNames` (already has the project from line ~200) and pass it through.
5. Wire create/update handlers: map DTO → domain, call `ExtractShiftRanges` + `ValidateShiftNames`, return `shift_names` in responses.

## Success Criteria
- [ ] `POST /projects` and `PUT /projects/:id` accept and echo `shift_names`.
- [ ] Invalid `range` (not in payrate) → 400 with the Vietnamese mismatch error.
- [ ] `GET /auth/me` (employee) returns `schedule_windows[].shift_name` matching the admin config for the resolved position.
- [ ] Existing geofence + payrate flows unchanged (focused tests pass).
- [ ] `go vet ./...` clean.

## Risk Assessment
- **Risk:** Changing `ResolveAllShiftWindows` signature touches the profile path. **Mitigation:** only one production caller; update its test fixtures too. Keep `resolveShift` (validation) untouched.
- **Risk:** Shift ranges differ between positions in the payrate. **Mitigation:** `ExtractShiftRanges` filters by the employee's effective position (same logic as `resolveShifts` lines 213-235), so names match the position the employee actually sees.
- **Risk:** Admin names a range that no position uses. **Mitigation:** validation compares against the union of all positions' ranges so the config is accepted; it simply won't surface for positions that lack it.
