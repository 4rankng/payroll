---
phase: 1
title: "Backend: domain model + migration + validation"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Backend — domain model, migration, validation

## Overview
Add the `ShiftName` value object and `Project.ShiftNames` field, the JSON column via migration `089`, and a `ValidateShiftNames` method that enforces shape, uniqueness, length, count, and payrate-range matching.

## Requirements
- Functional: admin can persist a `[{range, name}]` list on a project; the list is validated against the project's payrate shifts and the same `is_flexible` gate used by geofence.
- Non-functional: zero impact on existing attendance/pricing paths; column is nullable so existing projects are unaffected.

## Architecture
Copy the `GeofenceGate` / `GeofenceGates` precedent exactly:
- value struct `ShiftName{Range, Name string}` with the same GORM tags shape.
- `Project.ShiftNames []ShiftName` with `gorm:"type:json;serializer:json"`.
- `ValidateShiftNames(payrateShiftRanges []string) error` on `*Project`.

## Related Code Files
- Modify: `backend/internal/domain/project.go` — add `ShiftName` struct (after `GeofenceGate`, ~line 16), add `ShiftNames` field on `Project` (after `GeofenceGates`, ~line 47), add `ValidateShiftNames`.
- Create: `backend/migrations/089_add_project_shift_names.up.sql`
- Create: `backend/migrations/089_add_project_shift_names.down.sql`

## Implementation Steps
1. Add `ShiftName` struct near `GeofenceGate` (`project.go:11-16`).
2. Add `ShiftNames []ShiftName \`json:"shift_names" gorm:"type:json;serializer:json"\`` to `Project` after line 47, matching `GeofenceGates`.
3. Write `ValidateShiftNames` mirroring `ValidateGeofenceGates` (`project.go:230-253`):
   - Skip when `!p.IsFlexible` (return nil).
   - Max 20 entries.
   - For each: `range` matches `^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$`; `name` non-empty, ≤50 chars.
   - No duplicate `range` values.
   - When a non-empty `payrateShiftRanges` slice is passed, every `range` must be present in it (else Vietnamese error: `"Ca làm việc %s không khớp với ca trong bảng lương"`).
4. Add the two migration files. `up.sql` = `ALTER TABLE projects ADD COLUMN shift_names JSON NULL;` `down.sql` = `ALTER TABLE projects DROP COLUMN shift_names;`.

## Success Criteria
- [ ] `go build ./...` clean.
- [ ] `go test ./internal/domain/...` green.
- [ ] `ValidateShiftNames` unit tests cover: not-flexible (no-op), empty list OK, max 20 enforced, name required + ≤50, duplicate range rejected, range mismatch rejected, invalid HH:MM rejected.
- [ ] Migration applies cleanly on a fresh MySQL 8 instance.

## Risk Assessment
- **Risk:** Validation needs the payrate's shift ranges, which the domain layer doesn't own. **Mitigation:** `ValidateShiftNames` takes a `payrateShiftRanges []string` arg; the caller (handler/service) computes it by flattening the payrate and reusing `resolveShifts`'s time-range key logic. Domain stays dependency-free.
- **Risk:** Existing projects get a NULL column. **Mitigation:** Column is nullable; GORM reads NULL as an empty slice; validation no-ops when empty.
