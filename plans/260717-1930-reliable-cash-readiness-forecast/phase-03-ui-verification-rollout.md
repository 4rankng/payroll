# Phase 3: UI, verification, and rollout gate

**Status:** Completed

## Files

- `frontend/src/types/api/cash-readiness.types.ts`
- `frontend/src/components/timesheet/CashReadinessCard.tsx`
- frontend forecast display helpers if transformation logic is needed
- backend focused tests and `backend/tests/integration` cash-readiness flow
- forecast ADR/journal/API documentation

## Implementation

1. Keep the expected payout visible, but make the recommended reserve the cash
   planning number and label both clearly in Vietnamese.
2. Show deterministic approved amount, pending exposure, future estimate, a
   central interval, sample count, and measured/learning/uncalibrated state.
3. Replace basis-count wording such as “Độ tin cậy vừa” with production accuracy
   language. Do not claim reliability before the measurement gate passes.
4. Verify loading/error/no-history/uncalibrated/measured branches on the shared
   component used by desktop and mobile.
5. Run focused Go tests first, then backend package tests and lint, frontend
   type-check/lint/build, `make api-test`, and `graphify update .`.
6. Run mandatory code review for advisory invariants, public-contract
   compatibility, financial correctness, time boundaries, and event idempotency.

## Rollout

The upgraded forecast starts in shadow/learning state. It becomes “dependable”
only through measured metrics after at least 24 resolved forecasts for the same
decision horizon; no code or configuration automatically funds the wallet.
