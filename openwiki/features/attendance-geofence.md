---
type: feature
title: Attendance & Geofence
description: Self check-in/out with geofence validation, the immutable quota-credit hold, deferred enable switches, and the auto-reject sweepers that protect earning integrity.
tags: [feature, attendance, geofence, check-in, hold, quota, sweep]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-1135e40d3c2c5613c842dd6b
    resource: repo://backend/internal/app/services/attendance/attempt_classifier.go
  - id: openwiki-source-79f746d8550b1309e3186aed
    resource: repo://backend/internal/app/workers/auto_reject_checkout_worker.go
  - id: openwiki-source-021f7f9379129decb4e6e69e
    resource: repo://backend/internal/app/workers/auto_reject_sweep_worker.go
  - id: openwiki-source-43a2edab4fe54570e6cbe8d2
    resource: repo://backend/internal/app/workers/credit_quota_worker.go
  - id: openwiki-source-63ca859df3e630a62a4417f4
    resource: repo://backend/internal/domain/attendance.go
  - id: openwiki-source-da0867ee883cf3b9195fa043
    resource: repo://backend/migrations/101_seed_self_check_in_advance_hold_hours.up.sql
  - id: openwiki-source-4c5117867fd7bf51d2bf86a4
    resource: repo://backend/migrations/102_add_attendance_quota_credit_eligible_at.up.sql
  - id: openwiki-source-7814a631d4046f6bc84c871c
    resource: repo://backend/migrations/104_project_employees_pending_check_in.up.sql
  - id: openwiki-source-22c979da92ff10b8f954b8e0
    resource: repo://backend/migrations/105_project_employees_advance_request_enabled.up.sql
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Feature: Attendance & Geofence

The attendance feature records flexi-employee check-ins and check-outs at a project site, validates them against geofence gates, computes earnings, and routes those earnings into the advance-payment quota pool after a configurable hold. Admin review and dispute resolution are part of the same lifecycle.

## Lifecycle

```
pending → checked_in → completed → (auto-rejected | admin-reviewed)
```

- `checked_in` — the employee has tapped in at a geofence gate.
- `completed` — the employee has tapped out; earning has been computed and queued for quota credit.
- `rejected` — the checkout window closed without a checkout (auto-rejected by the sweeper; earning is 0).
- `admin-reviewed` — an admin has approved or rejected a disputed row via `/admin/attendances/:id/approve|reject`; `review_action` is `approved` or `rejected` and mirrors `GetStatus()`.

Status constants live in `internal/domain/attendance.go`:

```
AttendanceStatusCheckedIn  = "checked_in"
AttendanceStatusCompleted  = "completed"
AttendanceStatusOrphaned   = "orphaned"
AttendanceStatusRejected   = "rejected"   // CheckOutTime nil && SalaryRejectReason not nil
```

A `GeoReading` struct carries the device-reported `Lat`, `Lng`, `Accuracy` (meters), and `GpsAt` (device fix timestamp). `AccuracyPtr()` returns a pointer for the GORM column when the value is meaningful, otherwise `nil` (zero means "unknown").

## Geofence validation

`internal/app/services/attendance/attendance_geofence.go` enforces that a check-in or check-out happened at a valid gate for the project. Geofence gate definitions live in migration 068. The check-in attempt records the gate code (`CheckInGate`) and the gate used at checkout (`CheckOutGate`) for audit. Attempts outside the gates are rejected at the service layer with a typed domain error so the transport layer translates them to a clean 4xx response.

## Advance-hold hours (admin-controlled)

Between self-checkout and quota credit, the system waits an admin-configured number of hours so a disputed row can be reviewed before its earning enters the advance pool. The setting is seeded at 24 hours in migration 101 (`settings.key = 'self_check_in_advance_hold_hours'`, value `'24'`, type `number`). Admin can change it; subsequent self-checkouts use the new value.

To protect earnings already in flight, the hold is **immutable per row**: at self-checkout the system computes and stores `quota_credit_eligible_at = check_out_time + hold_hours` (migration 102). The worker and the recovery sweep read only that column, never the current setting, so an admin tightening the hold cannot release earnings that were already waiting, and loosening it cannot accelerate already-waiting earnings. The column is indexed (`idx_attendances_quota_credit_eligible`) for the sweeper query.

## Quota credit

`credit_quota_worker.go` (asynq) reads `attendances` rows where `quota_credited_at IS NULL AND earning_amount > 0 AND quota_credit_eligible_at <= clock.Now()`, writes the earnings into the advance-payment quota pool (`advance_payments.salary` / `max_adv_amount`), and stamps `quota_credited_at` (the idempotency key). The worker is registered in `bootstrap/routes.go` and runs alongside the rest of the asynq queue.

Because `quota_credit_eligible_at` is per-row and immutable, the worker can be retried freely and the recovery sweep can backfill missed runs without double-crediting.

## Auto-reject sweepers

Two workers guard rows whose checkout never arrived:

- `auto_reject_checkout_worker.go` — closes individual checkout windows for a specific gate / project. The window is `[K-1h, K+4h]`; rows past `K+4h` with no checkout are marked `rejected` (earning 0).
- `auto_reject_sweep_worker.go` — the project-wide sweeper that runs on a schedule and closes every stale `checked_in` row. This is the safety net behind the per-gate worker.

A row with `CheckOutTime == nil && SalaryRejectReason != nil` is by definition `rejected`.

## Project-employee flags (deferred activation + kill switches)

Migration 104 introduced deferred self-checkin activation. Enabling check-in for a project-employee pair takes effect on day 1 of the next month (`pending_check_in_enabled`, `check_in_effective_from` columns on `project_employees`). This mirrors the existing `pending_payment_schedule` / `schedule_effective_from` pair used elsewhere. Enable on 29 Aug → active 1 Sep. Disable is instant (no deferral); `DELETE` cancels a pending enable.

Migration 105 added `advance_request_enabled` (default 1) to `project_employees` as a per-employee kill switch for new advance requests (both the regular flow and the self-check-in flow). Setting it to 0 blocks new requests only — existing pending or approved requests are untouched. The frontend label is "tạm ngừng ứng lương".

## Concurrency

Check-in is the hot path: the same employee can retry the network on a slow connection, and two taps can race. `attendance_checkin.go` and `attendance_checkin_concurrency_test.go` cover the race condition — only the first valid tap creates the row; subsequent attempts are idempotent on (employee, project, date). `attempt_classifier.go` distinguishes legitimate retries from genuine duplicates.

## Admin review

Admins open `/admin/attendances/:id/approve|reject` to dispute-resolve rows. The handler writes `review_action` (`approved` | `rejected`) and stamps the audit fields (`review_*`). A `nil` `review_action` means "never reviewed"; the system-derived `GetStatus()` remains authoritative until an admin explicitly overrides it. See migration 083 for the audit columns.

## Relationships

- **Advance payments / FlexPay** — earnings banked into the quota pool feed `advance_payments.salary` and constrain `max_adv_amount`. See `features/flexpay.md`.
- **Timesheets** — attendance can be a source of timesheet hours for flexi employees; the timesheet import pipeline (BCC) can also backdate project assignments when an attendance proves earlier worked days. See `features/timesheet.md` and `features/bcc-import.md`.
- **Mobile flows** — check-in/out is the primary employee-mobile flow. See `frontend/employee-mobile.md`.
