---
phase: 5
title: "Tests and regression"
status: pending
priority: P1
effort: "4h"
dependencies: [2, 3, 4]
---

# Phase 5: Tests and regression

## Overview

Consolidated test matrix for all behavior introduced by Phases 2–4, plus full regression gates. Unit tests are written alongside each phase; this phase verifies coverage completeness and runs the gates.

## Requirements

- Functional: every Success Criterion row in plan.md maps to at least one automated test (backend unit, frontend unit, or api-test flow).
- Non-functional: tests use injected clock (`clock.Now` pattern — never `time.Now()`); deterministic (no sleep-based waits on cron; call the apply function directly).

## Test Matrix

### Backend — deferred activation (`employee_service` + attendance)

| # | Scenario | Assert |
|---|----------|--------|
| T1 | Toggle enable Jul 29 | pending=true, effective=Aug 1, `CheckInEnabled` still false |
| T2 | Toggle enable Aug 1 (strict) | effective=Sep 1 |
| T3 | Apply cron, effective due | active=true, pending cleared, event published |
| T4 | Apply cron, not due | row untouched |
| T5 | Enable then disable same day | pending cleared, NOT active, `ZeroOutQuota` NOT called |
| T6 | Disable active row | `ZeroOutQuota(current month)` called (regression) |
| T7 | Cancel pending endpoint | clears; second cancel → 400 |
| T8 | Check-in attempt while pending | countdown VN message, no attendance row |
| T9 | Toggle enable without project payrate | 400 (existing validation intact) |

### Backend — tail parity (`advance_payment`)

| # | Scenario | Assert |
|---|----------|--------|
| T10 | Hybrid requests prev month, day 5 | accepted; budget from prev-month rows |
| T11 | Hybrid requests prev month, day 9 | rejected (cutoff msg) |
| T12 | Hybrid requests prev month, day 31 | rejected (closes hole) |
| T13 | Hybrid requests current month, day 5 via regular endpoint | rejected (tail = prev only) |
| T14 | Non-hybrid prev month, day 5 | accepted (regression, unchanged) |
| T15 | Non-hybrid current month, day 25 | accepted (regression, unchanged) |
| T16 | Hybrid locked-gap quota-missing fallback | cutoff message (the kept `len==0` carve-out) |

### Frontend

| # | Scenario | Assert |
|---|----------|--------|
| T17 | `isPriorMonthRequestable` day 8 + quota | true |
| T18 | `isPriorMonthRequestable` day 9 / no quota | false |
| T19 | `getInitialEmployeeAdvanceMonth` checkin + tail + quota | prev month |
| T20 | Request form routes prev-month submit to regular mutation | correct endpoint called |
| T21 | Checkin card pending state | countdown rendered, action disabled |
| T22 | Admin toggle pending badge + cancel | badge shows date; cancel invalidates list |

### Integration

- [x] `make api-test` — all 30 flow files green (watch: advance request flows, attendance flows)

## Related Code Files

- Create: `backend/internal/app/services/project/employee_service_checkin_pending_test.go`
- Create: `backend/internal/app/services/advance_payment/service_hybrid_window_test.go`
- Modify: `backend/internal/app/services/attendance/attendance_checkin_test.go` (pending-reject case)
- Modify: `frontend/src/utils/advancePaymentHelpers.test.ts`
- Modify: `frontend/src/components/advance-payment/AdvancePaymentRequestForm.test.tsx`
- Create/modify: admin toggle component test (pending badge)

## Implementation Steps

1. Sweep T1–T22 against what Phases 2–4 already wrote; add the missing ones.
2. Inject fake clock where the service uses `clock.Now()` (follow existing test patterns — `clock` package is mockable via its Global/functional-field design; see `advance_payment/config.go` function-fields).
3. Run gates: `cd backend && go test ./... -race -count=1`; `cd frontend && pnpm lint && pnpm type-check && pnpm test`.
4. `make api-test` against a running backend (`make dev` + `make db`).
5. Fix regressions; do not weaken tests.

## Success Criteria

- [x] T1–T22 all present and green
- [x] `go test ./... -race` green
- [x] `pnpm lint && pnpm type-check && pnpm test` green
- [x] `make api-test` green
- [x] No `time.Now()` in new domain logic (grep check)

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Cron apply untestable without scheduler | Call `ApplyPendingCheckInEnables(ctx)` directly with injected clock — no asynq dependency |
| api-test env drift (migrations) | Run migration (Phase 1) locally first; api-test hits live local backend |
| Test flakiness around month boundaries | All date math via injected clock — no real "today" in tests |
