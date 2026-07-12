# Employee Meaningful Motion

Status: complete

## Approved scope

Implement five CSS/React micro-interactions for the mobile employee portal:

1. One-shot GPS accuracy confirmation when a fresh sample first reaches the required threshold.
2. One-shot nearest-checkpoint route and marker emphasis after an outside-geofence failure.
3. One-shot accepted check-in/check-out confirmation with the recorded time.
4. One-shot transition when the check-in window opens.
5. Advance-request success state that collapses to a concise submitted summary.

## Constraints

- No animation dependency or backend contract change.
- Transform/opacity only for motion; no continuous decorative effects.
- Respect `prefers-reduced-motion`.
- Preserve attendance and advance-payment behavior, including server-side validation.

## Validation

- Focused component tests for each state.
- Full frontend test suite (94 tests), lint/type-check, and production build.
