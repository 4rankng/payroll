# Outside-window card white surface

Status: planned

## Goal

Make the employee check-in card shown before the check-in window use a pure white outer surface.

## Phase

| Phase | Status | Description |
| --- | --- | --- |
| 01 | pending | Apply and verify the isolated surface styling change. |

## Acceptance criteria

- The outside-window card in `EmployeeCheckInCard.tsx` uses `bg-white`.
- Its outer border and shadow remain neutral.
- Amber stays limited to the inner clock/status/countdown elements.
- Attendance, GPS, map, API, and component-prop behavior remain unchanged.
- The focused component test and frontend lint/type check pass.

## Dependencies

- None; this is an isolated frontend styling change.

## Rollback

Revert the single outer-card background utility if the warmer ivory treatment is preferred later.

## Details

- [Phase 01: white surface](phase-01-white-surface.md)
