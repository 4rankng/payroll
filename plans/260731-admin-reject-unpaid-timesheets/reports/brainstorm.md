# Approved design

## Decision

The Admin action rejects every timesheet in one project and inclusive custom date range when `payment_status != paid`. This deliberately expands rejection beyond the prior `pending_approval`-only transition. A single reason applies to the batch; Admin may reject rows they created; Partner notifications are out of scope.

## Design

- Use a dedicated server-scoped endpoint instead of submitting IDs from the paginated UI.
- Protect paid rows with a conditional database predicate inside the transaction.
- Preserve historical payment-attempt fields for audit, but clear active approval, edit-request, and force-payroll state.
- Return the actual affected count and publish it with project metadata.
- Reuse the existing responsive date-range picker, searchable project selector, textarea, and destructive dialog conventions.
- Surface the action only in Admin desktop/mobile overflow actions; keep Partner UI and permission behavior unchanged.

## Alternatives rejected

- Rejecting only visible IDs: incomplete across pagination.
- Restricting to `pending_approval`: does not satisfy the approved literal non-paid rule.
- Async job: unnecessary until evidence shows synchronous range updates exceed operational limits.
