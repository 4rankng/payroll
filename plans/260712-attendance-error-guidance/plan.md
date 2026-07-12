# Attendance Error Guidance

Status: complete

## Goal

Make check-in timing failures understandable and location failures immediately actionable for check-in-enabled employees.

## Phases

1. Add structured timing and location guidance to backend attendance rejections without changing their Vietnamese messages.
2. Add a frontend timing-error classifier and focused tests.
3. Present a closable timing dialog and automatically expand checkpoint guidance on GPS/geofence failures.
4. Move the viewport to the expanded checkpoint map after a GPS/geofence failure so the guidance is immediately visible.
5. Expose the full advisory shift/check-in/checkout schedule and every checkpoint name in the employee attendance card.
6. Run backend and frontend tests, lint/type-check, production build, and inspect attendance action paths.

## Validation

- Focused backend attendance and HTTP-response tests pass.
- Focused backend attendance and employee-profile tests pass.
- Frontend unit suite passes (91 tests), as do lint/type-check and the production build.

## Dependencies

- Existing backend error messages and employee-profile shift-window fields remain unchanged.
- Existing `EmployeeLocationMap` remains the sole checkpoint-map implementation.

## Acceptance Criteria

- The backend attaches optional machine-readable timing and checkpoint guidance while preserving existing error messages.
- A rejected **Vào làm** or **Tan ca** caused by a timing window displays a closable Vietnamese dialog with the relevant allowed time range.
- GPS accuracy, permission, and geofence failures automatically show the nearest-checkpoint map; it retains the employee’s current location when available.
- GPS accuracy, permission, and geofence failures scroll the employee directly to the expanded nearest-checkpoint map.
- Leaflet map layers remain below the fixed employee action dock.
- The attendance card displays every configured shift, its allowed check-in and checkout windows, and all configured checkpoint names.
- When an employee is checked in, the shown checkout window is anchored to that attendance's original check-in rather than the current clock time.
- Existing checkout no-salary confirmation remains available only for its current confirmed-checkout case.
- No schema or attendance-policy changes.
