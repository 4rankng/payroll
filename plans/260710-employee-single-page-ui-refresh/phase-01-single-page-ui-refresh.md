# Phase 01 — Single-page UI refresh

## Context

The current flexible-pay employee page composes independent cards in this order: check-in, advance form, attendance history, request history, and bank details. This phase retains that data and behavior but improves its hierarchy and eliminates unnecessary nested surfaces.

## Files to modify

- `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx`
- `frontend/src/components/employees/EmployeeMobileShell.tsx`
- `frontend/src/components/employees/EmployeePortalHeader.tsx`
- `frontend/src/components/employees/EmployeeCheckInCard.tsx`
- `frontend/src/components/employees/EmployeeAttendanceHistoryCard.tsx`
- `frontend/src/components/employees/EmployeeAttendanceHistoryCard.test.tsx`
- `frontend/src/components/advance-payment/AdvancePaymentRequestForm.tsx`
- `frontend/src/components/advance-payment/AdvancePaymentHistoryCard.tsx`
- `frontend/src/components/employees/EmployeeBankInfoCard.tsx`
- `frontend/src/styles/base.css`
- `frontend/src/styles/variables.css`

## Implementation steps

1. Refine the employee shell and header: use the neutral canvas, a compact white header, date context, readable type roles, and unchanged notification/account controls.
2. Re-style `EmployeeCheckInCard` presentation for the active state so its status, start time, GPS/location disclosure, and one green `Tan ca` primary action are visually clear. Keep all callback, mutation, cooldown, and dialog code unchanged; retain cancellation as an outlined/destructive secondary action.
3. Flatten `AdvancePaymentRequestForm` into a summary-first section: available balance and pay period, amount field/quick amounts, bank destination, fee/net rows, and the existing `Tiếp tục` flow. Do not change amount normalization, validation, fee calculation, or form payloads.
4. Extend `EmployeeAttendanceHistoryCard` with a collapsed-by-default preview mode that renders the three newest records and summary counts. Add an accessible `Xem toàn bộ lịch chấm công` control to reveal the current full month, month navigation, and the existing inline issue disclosure. Make whole rows navigational only if no current interaction is lost; preserve the established detailed warning content for anomalous records.
5. Reduce decorative cards, shadows, pills, and tinted nesting across request history and bank details. Keep their expand/cancel behavior and all labels/data intact.
6. Adjust the employee semantic CSS variables/classes for the approved contrast hierarchy, restrained border radius/shadows, 14 px body readability, and visible keyboard focus states.
7. Update attendance component tests to prove preview counts, the three-record cap, full-history expansion, and anomaly-detail interaction. Keep existing check-in/advance behavior covered by type-check and existing tests.

## Validation

- `cd frontend && pnpm test:run -- EmployeeAttendanceHistoryCard.test.tsx`
- `cd frontend && pnpm type-check`
- `cd frontend && pnpm lint`
- Inspect the mobile employee portal manually or with its existing development environment at a narrow viewport; verify the full check-in/out, advance form, attendance expansion, request cancellation, bank expansion, notifications, and account menu paths remain reachable.

## Contract and rollback notes

All data remains supplied by the current hooks and services. No payload, route, exported type, API response, or database schema changes are expected. Revert the touched frontend components/styles to roll back the visual change without data migration.
