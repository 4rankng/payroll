# Admin-named shifts in project settings

**Date:** 2026-07-13 · **Status:** shipped (all 6 phases)

## Summary

For `is_flexible` (check-in-enabled) projects, admins can now assign display
names to the shift time-ranges already defined in the payrate — e.g.
`"Ca làm": "09:00-18:00"`, `"Ca đêm": "21:00-05:00"`. The name surfaces on the
employee attendance card, with a small **"Qua đêm"** (Moon) badge when a shift
crosses midnight. Pricing, validation, and the payrate model are untouched.

## Decision

Two choices from the user interview, both picked for lowest risk:

1. **Name existing payrate shifts** — rejected the "standalone shifts in settings"
   alternative. The payrate stays the single source of truth for shift times and
   pricing; admin only attaches a display name per detected range. Nothing that
   touches money moves.
2. **Name + overnight badge** — show the admin-given name plus a "Qua đêm" badge
   for cross-midnight ranges, rather than a separate shift-type concept.

## Implementation

Architectural approach was "copy `geofence_gates`." That existing `[]struct{...}`
GORM `serializer:json` column on `projects` was the perfect precedent — it gave
a proven template for storage, DTO, handler wiring, validation, and the
inline-edit UI section. Reusing the pattern minimized surface area.

- `domain.ShiftName{Range, Name}` + `Project.ShiftNames` JSON column
  (`gorm:"type:json;serializer:json"`), mirroring `geofence_gates`.
- `(*Project).ValidateShiftNames(payrateShiftRanges []string)` — shape,
  uniqueness, length (≤50), count (≤20), and range-must-match-payrate.
- Migration `089_add_project_shift_names.{up,down}.sql` — nullable JSON column.
- DTOs: `ShiftNameRequest` + `shift_names` on all 4 project DTOs.
- Advisory name propagation chain:
  `ShiftWindow.Name` → `employee.ScheduleWindowInfo.ShiftName`
  → `dto.ScheduleWindowInfo.ShiftName` + top-level `ShiftName` on
  `EmployeeProfileResponse`. `ResolveAllShiftWindows` now takes a
  `names map[string]string` arg.

The `"HH:MM-HH:MM"` range key is the natural join — `resolveShifts` already
builds that key from the payrate, so attaching the name there had no matching
ambiguity.

**Create vs Update validation split:** on Create the payrate doesn't exist yet,
so `shift_names` gets shape validation only; the range↔payrate match check runs
on Update via the new `attendance.ExtractShiftRanges`, once the payrate is
configured.

Frontend: `ShiftNamesSection.tsx` in `ProjectDetailsSheet` copies the
`GeofenceSection` inline-edit pattern (detect ranges from the active payrate,
local state, Save/Cancel, `useUpdateProject`). `AttendanceReference` tab labels
use `schedule.shift_name` with fallback to `Ca ngày` / `Ca đêm`.

This is an **additive, non-breaking change**. The `schedule_windows` chain is
display-only, so adding a field is safe; no validation logic moved and the
payrate/pricing code is completely untouched.

## Tests

- 9 new backend domain tests (`ValidateShiftNames` branches + `IsValidShiftRange`).
- 2 new attendance service tests (`ResolveAllShiftWindows` name attachment +
  `ExtractShiftRanges` dedup).
- 2 new frontend tests (named shift labels; overnight badge).
- Backend: build clean, `go vet` clean, changed-package tests green with `-race`.
- Frontend: `pnpm lint` (eslint + tsc) clean, 110/110 tests pass, production
  build succeeds.

## Notes

Pre-existing unrelated test failures remain on the baseline and are **not**
caused by this work: `ninepay` provider tests and
`advance_payment_request_repository.TestCreateWithBudgetCheck_*`. Confirmed via
`git diff --name-only` that they live in untouched packages; they're documented
in the prior plan `2026-07-09-attendance-checkout-geofence-fixes`.

This was a clean, low-risk implementation. Nothing worth flagging as a setback.
