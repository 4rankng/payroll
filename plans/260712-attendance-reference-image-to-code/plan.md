# Attendance reference redesign

Status: complete

## Goal

Translate the selected employee attendance reference into the live mobile check-in card while preserving live GPS, map, check-in, and check-out behaviour.

## Scope

- Replace the long timing list with a selectable shift summary.
- Show the first two checkpoints with a compact remaining-count affordance.
- Keep existing API contracts and the live map/action flow unchanged.

## Acceptance criteria

- Employees can select a configured shift and see its work, check-in, and check-out windows.
- Overnight windows remain explicitly marked as ending the following day.
- Checkpoint names remain available in the compact reference.
- Focused tests, lint, build, and the employee component suite pass.

## Validation

- Focused attendance-reference tests pass.
- Full frontend suite passes: 19 files / 94 tests.
- Frontend lint, type-check, and production build pass.
