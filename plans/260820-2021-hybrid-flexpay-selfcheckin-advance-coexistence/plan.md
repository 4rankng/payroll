---
title: "Hybrid FlexPay-SelfCheckIn Advance Coexistence"
description: "Let hybrid employees (prior-month FlexPay + current-month self-checkin) request the previous salary period during the days 1-8 tail, keep checking in normally, and defer check-in enablement activation to day 1 of the next month."
status: completed
priority: P1
effort: "medium"
tags: [feature, backend, frontend, advance-payment, attendance, migration]
blockedBy: []
blocks: []
created: 2026-08-20
---

# Hybrid FlexPay-SelfCheckIn Advance Coexistence

## Overview

A flexible employee with July salary data via admin-upload FlexPay who is registered for self-checkin in August must, on Aug 5, still be able to request an advance for the July period — while continuing to check in/out normally under the self-checkin scheme. Today the frontend hard-switches her to the checkin-only surface (July becomes invisible), and the backend skips the request window wholesale for checkin-enabled employees (July stays requestable any day of August — a permissiveness hole).

Additionally, enabling self-checkin for an employee must **activate on day 1 of the next month** (enable 29 Aug → active 1 Sep), never mid-month — mirroring the existing deferred `PendingPaymentSchedule` pattern.

**Locked decisions** (user, 2026-08-20):
- Scope: **HOLD** — tail parity + dual surface + tests. No admin reporting extras.
- July requestable during **days 1–8** of August only (FlexPay tail parity, `RequestCutoffDay = 8`).
- Day-1 enable is **strict next-month**: enable Aug 1 → active Sep 1.
- Pending window shows **countdown** to the employee + admin **cancel** option.

## Background / evidence

Scout report: [`plans/reports/scout-260820-2016-hybrid-flexpay-checkin-coexistence.md`](../reports/scout-260820-2016-hybrid-flexpay-checkin-coexistence.md)

- Regular FlexPay flow: `backend/internal/app/services/advance_payment/service.go` — `CreateRequest` (~226; window-check skip at ~245 for `hasCheckInEnabled`), `isRequestMonthAllowed` (~303; days 1–8 → prev month only), `GetEmployeeAdvanceInfo` (~62; already returns ALL quota months for checkin-enabled employees via `GetMonthsByEmployee`).
- Checkin flow: `backend/internal/app/services/advance_payment/checkin_advance.go` — `SelfCheckInAdvanceWindowOpenDay = 10`, `isCheckInRequestWindowOpen` (current calendar month only).
- Frontend hard switch: `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx:83-105`.
- Deferred-activation pattern to mirror: `backend/internal/app/services/project/employee_service.go` — `RequestPaymentScheduleChange` (~539), `ApplyPendingScheduleChanges` (~708, cron-registered in `bootstrap/scheduler_jobs.go`), entity fields `PendingPaymentSchedule`/`ScheduleEffectiveFrom` on `domain.ProjectEmployee`.
- Instant toggle today: `employee_service.go` `ToggleCheckInEnabled` (~795) — disable also calls `ZeroOutQuota` (current month onward).
- Attendance gate reads the active flag: `attendance/attendance_checkin.go:46`.
- Budget check is per-`forMonth` (`CreateWithBudgetCheck`) — an Aug-5 July request correctly draws July quota. No data-model change needed for tail requests.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Hybrid employee can request July period on Aug 1–8 (FlexPay tail parity, backend + frontend) | P1 |
| 2 | Checkin/out in August unaffected (regression-verified, no behavior change) | P1 |
| 3 | Check-in enablement activates day 1 of next month, strict (day-1 enable → next month) | P1 |
| 4 | Pending window: employee sees activation countdown; admin can cancel pending enable | P2 |
| 5 | Close the backend permissiveness hole (no July requests after day 8 for hybrids) | P1 |

## Non-goals

- No change to the checkin advance window itself (day 10, current month).
- No admin reporting/badges beyond the pending-cancel UI.
- No tail-boundary configurability (constant, not a settings key).
- No day-9 dead-zone fix (July tail closed, Aug window opens day 10 — same as regular FlexPay; consistent).
- No change to disable semantics (stays instant with `ZeroOutQuota`).

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Migration: pending check-in columns](./phase-01-start.md) | Completed |
| 2 | [Deferred check-in activation](./phase-02-deferred-check-in-activation.md) | Completed |
| 3 | [Hybrid tail parity backend](./phase-03-hybrid-tail-parity-backend.md) | Completed |
| 4 | [Frontend dual surface](./phase-04-frontend-dual-surface.md) | Completed |
| 5 | [Tests and regression](./phase-05-tests-and-regression.md) | Completed |

Phase 1 → 2 sequential (columns then activation logic); Phase 3 independent (no migration needed — can run in parallel); Phase 4 depends on 2+3; Phase 5 spans all (test files touched per phase, consolidated gate at the end).

## Success Criteria

- [x] Hybrid employee requests July on Aug 5 (day within 1–8) → success, quota drawn from July rows
- [x] Hybrid employee requests July on Aug 9+ → rejected with cutoff message
- [x] Hybrid employee check-in/out in August → unchanged success
- [x] Enable check-in 29 Aug → pending; employee check-in attempt pre-Sep-1 → distinct "kích hoạt từ 01/09" message; active Sep 1 via cron
- [x] Enable check-in Aug 1 → active Sep 1 (strict)
- [x] Admin can cancel pending enable before activation
- [x] `cd backend && go test ./internal/app/services/... ./internal/app/services/advance_payment/...` green
- [x] `cd frontend && pnpm lint && pnpm type-check && pnpm test` green
- [x] `make api-test` green (30 flow files)

## Risks

| Risk | Mitigation |
|------|------------|
| Migration on `project_employees` (prod data) | Additive `ALTER ADD COLUMN` (authorized pattern); nullable columns, no backfill needed |
| Cron apply race with manual toggle | Activation updates same row via repo `Update` in tx; last-write-wins acceptable (admin intent) |
| Frontend dual-query complexity | Backend already returns all quota months; frontend only re-enables the regular query during tail |
| Mid-rollback: pending rows stranded | Activation query is date-driven; stranded pendings apply on next run after redeploy |

## Deployment notes

- Migration must run before backend deploy (standing rule: `make deploy` never runs migrations — apply `.up.sql` manually per `docs/deployment-guide.md`).
- Phase 3 (tail parity) is a behavior tightening — deploy together with Phase 4 so hybrid employees aren't blocked in UI before the fix is visible.
