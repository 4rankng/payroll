# Phase 2: Align API and UI

## Change

Replace the misleading `confirmed_payable` response field with
`observed_approved`. Keep `expected_total` and the P50–P95 band as target-Kỳ
totals. Preserve the card's visual format and responsive two-endpoint range.

## Validation

- Backend DTO and frontend type match.
- Frontend lint and TypeScript checks pass.
