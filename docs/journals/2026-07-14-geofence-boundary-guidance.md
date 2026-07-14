# Geofence boundary guidance for uncertain GPS fixes

**Date:** 2026-07-14 · **Status:** fixed

## Context

An employee was shown as 124 m from Cổng D, inside the configured 150 m radius,
but the GPS accuracy was ±48 m. The frontend correctly classified this as
inside-but-uncertain because the accuracy envelope crossed the geofence boundary,
yet it only told users outside the radius to move closer. This case therefore
showed a generic GPS accuracy warning without the useful next action.

## What changed

`checkInGeofenceGuidance` now produces movement instructions for both outside and
inside-but-uncertain states. For the uncertain state, it directs the employee
toward the center of the nearest named gate and asks them to retry.
`EmployeeCheckInCard` uses that shared instruction while acquiring a location and
when presenting a rejected low-confidence attempt. The pre-submit card and mobile
action dock also prioritize the movement instruction when GPS has stopped watching,
so this state cannot fall through to the misleading "Sẵn sàng vào làm" view.

Regression tests reproduce the observed values exactly: 124 m distance, 150 m
radius, and ±48 m accuracy at Cổng D. They verify that the state remains
`inaccurate`, the employee receives the center-directed instruction, and the full
check-in card does not render the ready state.

The movement instruction is limited to zone-scale uncertainty (accuracy no worse
than the configured radius). A worker already at the gate with genuinely poor GPS
keeps the existing open-sky and hold-still recovery guidance.

## Decision

The backend remains the geofence authority. Its validation rules and API contract
were not changed; the fix is frontend guidance only. Moving toward the center of
the nearest gate increases the GPS uncertainty margin without weakening attendance
controls or pretending an uncertain fix is valid.

## Verification

- Focused frontend suite: 23/23 tests passed.
- Frontend lint and TypeScript checks passed.
- Frontend production build passed.
- Focused backend attendance and response tests passed.
- The root Makefile has no `api-test` target, so the root-level command documented
  for that target was not used.

No evergreen documentation changed because the geofence behavior and public
contract are unchanged.

## Reflection

Location guidance should explain the action that improves confidence, not merely
describe sensor quality. Keeping the instruction derived from the same geofence
status used by the card prevents the outside and boundary cases from drifting
apart again.
