# Scout Report — Hybrid FlexPay→Self-CheckIn Coexistence + Deferred Check-In Activation

Date: 2026-08-20 · Trigger: `/ak-scout` (following `/ak-plan` scope Qs, answered: HOLD SCOPE, July-tail = days 1–8)

## Target behavior (scenario)

Flexible employee, July salary period via admin-upload FlexPay (NO self-checkin in July). Registered self-checkin for August period.

1. Jul 25: request advance for July period → ✅ works today
2. Aug 5 (tail days 1–8): STILL request advance for July period (FlexPay tail rule)
3. Aug 1–31: check-in/out normally under self-checkin scheme (August period; advance window opens day 10)
4. **NEW RULE:** enabling check-in activates **day 1 of next month** — enable 29 Aug → active 1 Sep. Never mid-month.

## Current behavior vs target

| # | Behavior | Status |
|---|----------|--------|
| 1 | Jul 25 request July (regular flow) | ✅ works (day ≥ 20 → current period) |
| 2a | Aug 5 request July — backend regular endpoint | ⚠️ accepted, but by accident: window check skipped wholesale for `hasCheckInEnabled` (`service.go:245`) → July actually requestable ANY day of August (permissiveness hole vs days-1–8 tail) |
| 2b | Aug 5 request July — frontend | ❌ **hard switch** `FlexiblePayEmployeePage/index.tsx:83` `isCheckIn = profile.check_in_enabled` → regular `/me/advance-payment` query disabled; page binds `/me/check-in-advance` (current calendar month only). July invisible + unrequestable the moment check-in is enabled |
| 3 | Aug check-in/out | ✅ attendance flow independent of advance windows |
| 4 | Deferred activation | ❌ `ToggleCheckInEnabled` applies **instantly**. No pending/effective-date mechanism for the check-in flag |

## Relevant files

### Backend — advance flows
- `backend/internal/app/services/advance_payment/service.go` — regular FlexPay flow: `CreateRequest` (~226; window-check skip at ~245), `isRequestMonthAllowed` (~303; tail = prev month, days 1–8), `GetEmployeeAdvanceInfo` (~62; **already returns ALL quota months via `GetMonthsByEmployee` when check-in-enabled** — July+Aug both at API level), `getAdvanceEligibility` (~386; `hasFlexible` + `hasCheckInEnabled`, active flexible assignments only)
- `backend/internal/app/services/advance_payment/checkin_advance.go` — self-checkin flow: `SelfCheckInAdvanceWindowOpenDay = 10` (L32), `isCheckInRequestWindowOpen` (L77; current-month-only, prior periods locked), `GetCheckInAdvanceInfo` (L85), `CreateCheckInAdvanceRequest` (L174)
- `backend/internal/app/services/advance_payment/month_resolver.go` — `RequestCutoffDay=8`, `PeriodCycleStartDay=20`, `IsBeforeCutoff`, `IsInLockedGap`; clock twins in `backend/internal/pkg/clock/advance_payment.go`
- `backend/internal/app/services/attendance/attendance_checkin.go` — `CheckIn` gate L46: `!assignment.CheckInEnabled → reject` (reads active flag; pending state naturally rejected — needs distinct messaging)
- `backend/internal/domain/advance_payment_request.go` — `CreateWithBudgetCheck`: **budget sums per `forMonth`** → Aug-5 July request correctly draws July quota. No change needed.

### Backend — enablement + deferred-activation pattern (reuse this)
- `backend/internal/app/services/project/employee_service.go`
  - `ToggleCheckInEnabled` (~795): instant apply; **disable → `ZeroOutQuota(current month onward)`**; requires active payrate before enabling
  - `RequestPaymentScheduleChange` (~539): **the pattern to mirror** — not-immediate → `assignment.RequestScheduleChange(newSchedule, firstDayOfNextMonth)` stores `PendingPaymentSchedule` + `ScheduleEffectiveFrom`
  - `ApplyPendingScheduleChanges` (~708): bulk-applies due pending changes; registered as cron in `backend/internal/app/bootstrap/scheduler_jobs.go`
- `backend/internal/domain/project_employee.go` (entity — `PendingPaymentSchedule`, `ScheduleEffectiveFrom`, `ApplyPendingSchedule`, `RequestScheduleChange` live here; **no check-in pending counterpart exists**)
- `backend/internal/transport/http/handlers/project_employee/project_employee.go` — `ToggleCheckInEnabled` (~468), `BulkToggleCheckInEnabled` (~511), `RequestPaymentScheduleChange` (~308; handler already computes `firstDayOfNextMonth` L323)
- `backend/internal/transport/http/handlers/advance_payment/checkin_advance_handler.go` — dedicated `/me/check-in-advance` endpoints
- `backend/internal/infra/persistence/check_in_scope.go` — `checkInEnabledScope` SQL EXISTS (`pe.check_in_enabled = 1`) used by dashboards; pending state auto-excluded
- `backend/internal/domain/employee.go` — `CurrentProject` struct L483: `CheckInEnabled`, `PendingPaymentSchedule`, `ScheduleEffectiveFrom` (profile payload shape)

### Frontend
- `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx` — **hard switch L83–105** (`isCheckIn = check_in_enabled`; `useAdvancePaymentInfo({enabled: !isCheckIn})`); initial-month resolve L116 (`getInitialEmployeeAdvanceMonth` returns undefined for check-in)
- `frontend/src/utils/advancePaymentHelpers.ts` — `getInitialEmployeeAdvanceMonth` (L321), `isPastAdvancePaymentPeriod` (L312), rollover-day const (~L123); tests: `advancePaymentHelpers.test.ts`
- `frontend/src/components/advance-payment/AdvancePaymentRequestForm.tsx` — form, `forMonth` submit, `getAdvanceQuotaSummaryForMonth`; tests exist
- `frontend/src/hooks/api/useAdvancePayments.ts` — `useCheckInAdvanceInfo`, `useRequestCheckInAdvance`, `useAdvancePaymentInfo`
- `frontend/src/components/employees/EmployeeCheckInCard.tsx` — check-in UI gated on `profile.check_in_enabled`
- `frontend/src/components/project-employees/` — admin `CheckInToggle` + `PaymentScheduleToggle` (pending-badge UI pattern to mirror)
- `frontend/src/services/api/project-employee.service.ts` — `toggleCheckInEnabled` (~298)

## Key facts / gotchas

1. Quota keyed per `for_month`; July rows persist after rollover; budget check scoped to requested month → tail requests are pure window/UI problems, no data problem.
2. `GetEmployeeAdvanceInfo` already lists **every earned-quota month** for check-in-enabled employees — frontend just never calls it for them.
3. Backend hole: with `hasCheckInEnabled`, regular endpoint accepts July on Aug 31 too. Chosen fix (user decision): enforce FlexPay tail parity (days 1–8, prev month only) on regular endpoint for hybrids.
4. Dead zone day 9: July tail closes day 8, August checkin window opens day 10 → day 9 requestable for neither. Same as non-hybrid employees; consistent, note in plan only.
5. `ZeroOutQuota` on disable is instant + current-month-onward. With deferred enable: enabling Aug 29 (active Sep 1) must NOT touch August FlexPay quota; disable semantics stay instant (verify with user).
6. Deferred enable keeps `getAdvanceEligibility` clean: `hasCheckInEnabled` flips only at day 1 → no mid-month hybrid weirdness; hybrid exists exactly at month boundaries where periods are clean anyway.
7. Profile `check_in_enabled` must reflect **active** state only (else frontend switch flips early). Expose activation date for countdown UI (optional).
8. Migration needed for pending-checkin columns (additive `ALTER ADD COLUMN` — authorized deploy pattern; code-first alternative not viable since pending state must survive restarts).
9. Cron already exists (`scheduler_jobs.go`) — fold pending-checkin application into `ApplyPendingScheduleChanges` job or sibling registration; no new infra.
10. `checkInEnabledScope` dashboards ignore pending employees — acceptable (they appear once active); optionally show "sắp kích hoạt" badge in admin project-employee list.

## Implementation shape (sketch for plan)

- **P1 Backend — deferred activation:** migration `ADD pending_check_in_enabled TINYINT NULL, check_in_effective_from DATE NULL` on `project_employees`; `ToggleCheckInEnabled(enable=true)` → write pending + `firstDayOfNextMonth`; extend cron to apply due pendings; disable path unchanged; distinct reject message for pending check-in attempts; admin list shows pending badge + cancel.
- **P2 Backend — hybrid tail parity:** regular `CreateRequest`: for `hasCheckInEnabled` hybrids enforce `isRequestMonthAllowed` (prev month, days 1–8) instead of skipping.
- **P3 Frontend — dual surface:** check-in-enabled employees with prior-month quota during tail: fetch BOTH info queries; month navigator allows prev month during tail; prev-month submits → regular endpoint; checkin card countdown when pending.
- **P4 Tests:** window unit tests (isRequestMonthAllowed hybrid, isCheckInRequestWindowOpen), toggle→pending→activate cron test, ZeroOutQuota non-interaction, frontend helper tests, `make api-test`.

## Unresolved questions

1. Enable on day 1 of month M (e.g. Aug 1) — activate Sep 1 (strict "always next month") or Aug-run already counts? Rule text says always day 1 next month → Sep 1. Confirm.
2. Disable stays instant + `ZeroOutQuota` from current month? (Current spec; deferred activation changes nothing on disable side?)
3. During pending period (enabled 29 Aug, before Sep 1) employee opens app: show countdown "dịch vụ tự chấm công kích hoạt từ 01/09"? (Recommended yes; needs profile field.)
4. Cancelling a pending check-in enable — allowed until activation day? (Mirror cancel-pending-schedule-change.)
5. Hybrid tail parity on regular endpoint (days 1–8) — already answered HOLD/days-1–8 in plan scope Qs; re-confirm unaffected by deferred activation. (Analysis: unaffected — activation day 1 precedes tail.)
