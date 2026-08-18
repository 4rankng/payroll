# Timesheet hour display precision

## Status

COMPLETE

## Scope

- Render active Admin and Partner timesheet hours with exactly two decimal places.
- Keep persisted hours, sums, API contracts, and unrelated in-progress work unchanged.

## Implementation

1. Add a pure two-decimal hours formatter beside the shared timesheet grouping helpers.
2. Use it in the shared desktop grouped table and active mobile list.
3. Add focused formatter coverage, then run frontend checks and refresh the graph.

## Acceptance criteria

- `85.53999999999999` displays as `85,54h`.
- Whole-hour values display trailing zeros, such as `36,00h`.
- Admin and Partner retain the same data and interactions on desktop and mobile.

## Verification

- `pnpm vitest run src/components/timesheet/utils/timesheetGrouping.test.ts` — 3/3 passing.
- `pnpm lint` — ESLint and TypeScript passing.
- `git diff --check` — no whitespace errors.
- `graphify update .` — completed (with the pre-existing missing-skill warning).
