---
phase: 2
title: "Deferred check-in activation"
status: pending
priority: P1
effort: "6h"
dependencies: [1]
---

# Phase 2: Deferred check-in activation

## Overview

Enable check-in for an employee → goes pending, activates day 1 of next month (strict: enabling on the 1st still defers to the following month). Admin can cancel while pending. Employee sees a countdown. Disable stays instant (with `ZeroOutQuota`).

## Requirements

- Functional:
  - `ToggleCheckInEnabled(enabled=true)` writes `PendingCheckInEnabled=true` + `CheckInEffectiveFrom = first day of NEXT month` (from `clock.Now()`), leaves `CheckInEnabled=false`.
  - Cron applies due pendings (`CheckInEffectiveFrom <= today`): sets `CheckInEnabled=true`, clears pending fields.
  - `ToggleCheckInEnabled(enabled=false)` on a pending row cancels the pending (clears fields) — and on an active row, current instant disable + `ZeroOutQuota` path unchanged.
  - Cancel endpoint: clears pending fields only; rejects if not pending.
  - Employee profile response gains pending state + activation date (for countdown).
  - Check-in attempt while pending → distinct VN message: "Dịch vụ tự chấm công sẽ kích hoạt từ {dd/MM}."
- Non-functional: all writes transactional; reuse `clock.Now()` (Asia/Ho_Chi_Minh); strict activation (no same-month day-1 exception).

## Architecture

Mirrors `RequestPaymentScheduleChange` → `ApplyPendingScheduleChanges`. Toggle handler computes effective date (pattern already at `project_employee.go:323`: `time.Date(now.Year(), now.Month()+1, 1, ...)`); service persists pending; daily cron folds application into the existing scheduler job registration.

```
admin PATCH checkin-enabled{true}
  → service: pending=true, effective=firstOfNextMonth(now), CheckInEnabled stays false
  → cron (daily): WHERE pending AND effective <= today → CheckInEnabled=true, clear pending
  → admin DELETE pending (cancel): clear pending fields (only if pending)
```

## Related Code Files

- Modify: `backend/internal/app/services/project/employee_service.go` — `ToggleCheckInEnabled` (~795), `BulkToggleCheckInEnabled` (~843), new `CancelPendingCheckInEnable`, new `ApplyPendingCheckInEnables` (or fold into `ApplyPendingScheduleChanges` loop)
- Modify: `backend/internal/domain/project_employee.go` — helper `ApplyPendingCheckIn() bool` + `RequestCheckInChange(effective time.Time)` (mirror schedule methods)
- Modify: `backend/internal/transport/http/handlers/project_employee/project_employee.go` — `ToggleCheckInEnabled` (~468) unchanged route; new `CancelPendingCheckInEnable` handler; response DTO gains `PendingCheckInEnabled`/`CheckInEffectiveFrom`
- Modify: `backend/internal/app/bootstrap/scheduler_jobs.go` — register/fold apply job
- Modify: `backend/internal/app/bootstrap/routes_*` — route for cancel (mirror `payment-schedule` DELETE)
- Modify: `backend/internal/domain/employee.go` — `CurrentProject` struct L483 gains `PendingCheckInEnabled *bool`, `CheckInEffectiveFrom *time.Time` + wherever `CheckInEnabled` is populated for `/me` profile
- Modify: `backend/internal/app/services/attendance/attendance_checkin.go` — pending rejection branch (~L46, distinct message)
- Reference: `frontend/src/services/api/project-employee.service.ts` `toggleCheckInEnabled` (~298) — API shape consumers

## Implementation Steps

1. Domain helpers on `ProjectEmployee` (mirror `RequestScheduleChange`/`ApplyPendingSchedule`).
2. Rework `ToggleCheckInEnabled`: `enabled=true` → validate payrate (existing check) → write pending; `enabled=false` → if pending, clear pending; else existing instant-disable path (incl. `ZeroOutQuota`).
3. `BulkToggleCheckInEnabled`: same branching per employee.
4. `ApplyPendingCheckInEnables(ctx)`: select due pendings, apply in tx, publish `ProjectEmployeeUpdatedEvent` per row (mirror schedule-apply loop).
5. Register in `scheduler_jobs.go` (daily, alongside `ApplyPendingScheduleChanges`).
6. Cancel handler + route (admin-only, mirrors cancel-schedule-change).
7. Profile/`CurrentProject` payload: pending fields surfaced in `/me` endpoints.
8. Attendance `CheckIn`: after `!assignment.CheckInEnabled` reject, if `PendingCheckInEnabled` → return countdown message instead.
9. `go build ./... && go test ./internal/app/services/project/...`.

## Success Criteria

- [x] Enable Jul 29 → row pending, effective 2026-08-01; cron on Aug 1 → active
- [x] Enable Aug 1 → effective Sep 1 (strict next-month)
- [x] Enable + disable same day → pending cleared, `CheckInEnabled` stays false, NO `ZeroOutQuota` call
- [x] Cancel endpoint clears pending; 400 when nothing pending
- [x] Check-in attempt while pending → "kích hoạt từ 01/09" message
- [x] Disable on active row → `ZeroOutQuota` still fires (regression)

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Cron misses a day (server down on the 1st) | Apply query is `effective <= today`, not `==` — self-healing on next run |
| Frontend toggle UI shows stale "on" after pending write | Response/list must surface pending badge (Phase 4 wires UI); backend returns pending fields now |
| `ZeroOutQuota` accidentally fired for cancel-pending | Explicit branch: pending-clear path must NOT touch quota (test) |
| Admin confusion mid-month (toggle appears not to work) | Phase 4 adds pending badge + activation date next to toggle; handler response message states activation date |
