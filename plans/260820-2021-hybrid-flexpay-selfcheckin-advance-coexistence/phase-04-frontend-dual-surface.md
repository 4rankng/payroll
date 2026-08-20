---
phase: 4
title: "Frontend dual surface"
status: pending
priority: P1
effort: "6h"
dependencies: [2, 3]
---

# Phase 4: Frontend dual surface

## Overview

Replace the `isCheckIn` hard switch on `FlexiblePayEmployeePage` with a month-aware dual surface: during the days 1–8 tail a checkin-enabled employee with prior-month quota sees and can request that prior period (via the regular flow), while the checkin surface keeps serving the current month. Plus the admin pending badge/cancel UI and the employee activation countdown.

## Requirements

- Functional:
  - For checkin-enabled employees: ALSO fetch `/me/advance-payment` (regular info); during tail days 1–8, when prior-month quota exists, the month navigator allows selecting the prior month and the request form submits to the regular endpoint for that month.
  - Current-month view still uses `/me/check-in-advance` + `/me/check-in-advance/request` (unchanged).
  - Month → endpoint routing: selected month === current calendar month → checkin mutation; selected month === prior month (during tail) → regular mutation.
  - Pending activation: `EmployeeCheckInCard` shows "Kích hoạt từ {dd/MM}" when `pendingCheckInEnabled` (service not yet active); check-in attempts blocked pre-activation with that message.
  - Admin project-employee list: pending badge + activation date next to the toggle (mirror `PaymentScheduleToggle` pending UI) + cancel action.
- Non-functional: no UI redesign — reuse existing cards/forms; user-facing text Vietnamese; full ₫ amounts (no K/tr).

## Architecture

```
FlexiblePayEmployeePage
  profile.check_in_enabled = true
  ├─ month = current    → /me/check-in-advance (info + request)   [existing]
  └─ month = prev (tail days 1–8 AND prev-month quota in regular info.quotas)
        → regular /me/advance-payment info quota for that month
        → submit via requestAdvancePayment({amount, forMonth: prevMonth})
```

Key reuse: `GetEmployeeAdvanceInfo` already returns ALL earned-quota months for checkin-enabled employees (`service.go:90-98`) — the regular info query is the only new fetch; no new backend endpoint needed for display.

## Related Code Files

- Modify: `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx` — dual queries (~83-105), month routing, initial-month resolve (~116-138)
- Modify: `frontend/src/utils/advancePaymentHelpers.ts` — `getInitialEmployeeAdvanceMonth` (L321: currently returns undefined for checkin; extend for tail+quota), possibly new `isPriorMonthRequestable(now, quotas)` helper (pure, testable)
- Modify: `frontend/src/components/advance-payment/AdvancePaymentRequestForm.tsx` — accept per-month canRequest/quota source; month chip shows period range (existing `formatPayrollMonthRange`)
- Modify: `frontend/src/hooks/api/useAdvancePayments.ts` — `useAdvancePaymentInfo({enabled})` already parameterized; conditional invalidation on submit
- Modify: `frontend/src/components/employees/EmployeeCheckInCard.tsx` — pending countdown state
- Modify: `frontend/src/types/api/` — employee profile type gains `pendingCheckInEnabled`, `checkInEffectiveFrom`; project-employee assignment type gains pending fields
- Modify: `frontend/src/components/project-employees/CheckInToggle.tsx` (or equivalent toggle component) — pending badge + cancel
- Modify: `frontend/src/hooks/api/useProjectEmployees.ts` + `frontend/src/services/api/project-employee.service.ts` — cancel pending checkin enable mutation
- Tests: `frontend/src/utils/advancePaymentHelpers.test.ts`, `frontend/src/components/advance-payment/AdvancePaymentRequestForm.test.tsx`

## Implementation Steps

1. Types: add pending fields to profile + assignment types.
2. Helpers: `isPriorMonthRequestable(selectedMonth, currentMonth, quotas)` — tail-day check client-side (day ≤ 8) AND prior-month quota present; unit tests.
3. `FlexiblePayEmployeePage`: fetch regular info when `isCheckInEnabled`; compute which surface owns the selected month; route `handleConfirmSubmit` to the right mutation; invalidate both queries after submit.
4. Month navigator: allow prior month during tail (navigator currently bounds by `MAX_MONTHS_BACK` + `isPastAdvancePaymentPeriod`); guard with the new helper.
5. `AdvancePaymentRequestForm`: props gain the resolved quota/canRequest for the selected month (already mostly month-driven via `getAdvanceQuotaSummaryForMonth`).
6. `EmployeeCheckInCard`: pending branch — disabled action + countdown text.
7. Admin: toggle shows pending badge ("kích hoạt 01/09") + cancel button → new mutation → invalidate list.
8. `pnpm lint && pnpm type-check && pnpm test`.

## Success Criteria

- [x] Hybrid on Aug 5 (July quota > 0): navigator offers July; July view shows quota from regular info; submit creates request (regular endpoint); success toast; history refetches
- [x] Hybrid on Aug 9: July not selectable; August checkin surface unchanged
- [x] Hybrid on Aug 5 with July quota = 0: July not offered
- [x] Pending enable (29 Aug): employee checkin card shows "Kích hoạt từ 01/09"; admin list shows badge; cancel clears it
- [x] Non-checkin flexible employee: page identical to today (regression)
- [x] `pnpm lint`, `pnpm type-check`, `pnpm test` green

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Client clock skew on tail-day check | Backend re-validates window (`isRequestMonthAllowed`); client check is UX-only — worst case a rejection toast |
| Two info queries double-fetch cost | Both cached by TanStack; only checkin-enabled employees pay it |
| Mutation routing sends wrong endpoint for month | Single helper decides endpoint by `(selectedMonth === currentCalMonth)`; unit-tested; backend rejects mismatches anyway |
| Initial-month effect fights navigator | Extend `getInitialEmployeeAdvanceMonth` carefully; keep `initialMonthResolvedRef` guard; covered by existing tests + new cases |
| Mobile parity (mobile admin pages) | Admin pending badge must appear in mobile project-employee list too — check both list components (feedback: mobile/desktop sync) |
