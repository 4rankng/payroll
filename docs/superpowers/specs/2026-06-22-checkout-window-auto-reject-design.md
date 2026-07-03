# Auto-reject attendance when the checkout window closes

**Date:** 2026-06-22
**Scope:** self-checkin / flexible attendance flow only (`IsFlexible` project + `assignment.CheckInEnabled`). Not the admin BCC-upload timesheet flow.
**Status:** Approved.

## Problem

Today, when an employee on the check-in scheme fails to check out within the valid
window `[K-1h, K+4h]`:

- A late checkout attempt is blocked by `validateCheckOutWindow` ("Đã quá giờ tan ca"),
  the transaction rolls back, and **nothing is persisted** — the attendance sits open
  forever and the employee can never complete it.
- The record only flips to `orphaned` (derived) after 18 h, with no reason captured
  and no earning recorded.

We want the system to **auto-reject** the attendance on its own once the checkout
window closes, so the outcome is final, visible, and earns nothing — with no admin
intervention and no late-checkout escape hatch.

## Locked decisions

| Decision | Choice |
|---|---|
| Trigger | **Async, exactly at the deadline.** asynq one-shot task scheduled for `K+4h`, enqueued when the employee checks in. |
| Outcome | **Hard reject, final.** `EarningAmount = 0`, reason recorded, no admin override. No advance quota is touched (checkout — the only place quota accrues — never happened). |
| Legacy records | **Ignore.** New check-ins get the asynq task; pre-existing records keep falling back to the existing 18 h `orphaned` derivation. No backfill, no sweep. |
| Migration | **None.** "Rejected" is a derived state using the existing `SalaryRejectReason` + `EarningAmount` columns. |

`K` = configured shift end, resolved per-record from (payrate, position, check-in time)
via the existing `resolveShift` (±1-day candidates, night-shift aware). The checkout
window upper bound is `K + checkOutUpperGrace` (`checkOutUpperGrace = 4 h`).

## Design

### Trigger — asynq task scheduled at check-in

In `CheckIn`, after the shift is resolved and the check-in window validated (so the
shift is known to be real), compute:

```go
latestCheckout := shift.end.Add(checkOutUpperGrace) // K + 4h
```

In a `domain.RegisterAfterCommit` hook (so the attendance row is durable), enqueue:

```go
EnqueueAutoRejectCheckout(ctx, attendanceID, latestCheckout)
// → asynq.NewTask(TaskAutoRejectCheckout, {attendance_id},
//     asynq.ProcessAt(latestCheckout),
//     asynq.TaskID(fmt.Sprintf("auto-reject-att:%d", attendanceID)),
//     asynq.MaxRetry(cfg.RetryMax), asynq.Queue(QueueDefault))
```

`TaskID` per attendance gives natural dedup; the handler is idempotent.

### Handler — `HandleAutoRejectCheckout`

Registered in the asynq mux. Thin wrapper around `AttendanceService.AutoRejectIfExpired(ctx, attendanceID)`:

1. Load attendance by ID.
2. `CheckOutTime != nil` → skip (employee completed in-window).
3. `SalaryRejectReason != nil` → skip (already rejected).
4. Else → re-resolve the configured shift (payrate + active assignment + check-in time) and set:
   - `EarningAmount = ptr(int64(0))`
   - `SalaryRejectReason = ptr(formatAutoRejectReason(checkInTime, shift.end))` —
     `"Đã hết hạn tan ca (Vào làm: HH:MM; Tan ca: HH:MM (hạn chót HH:MM))"`
   - `Update`.

   Falls back to the legacy generic message when the shift cannot be resolved
   (payrate deleted, assignment ended after check-in): `"Đã hết hạn tan ca —
   bạn đã quá giờ checkout cho ca này. Vui lòng liên hệ quản lý."`

### Representing "rejected" (no migration)

New constant `AttendanceStatusRejected = "rejected"`. A record is *rejected* iff
`CheckOutTime == nil && SalaryRejectReason != nil` — cleanly distinct from
completed-but-unpaid (which has `CheckOutTime != nil`).

`GetStatus(now)` (order matters — rejected before the orphaned fallback):

```
CheckOutTime != nil                              → completed
CheckOutTime == nil && SalaryRejectReason != nil → rejected   // NEW
now - CheckInTime > 18h                          → orphaned   // legacy fallback
otherwise                                        → checked_in
```

`buildFilterQuery`:

- `rejected`:  `check_out_time IS NULL AND salary_reject_reason IS NOT NULL`
- `checked_in`: `check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time >= now-18h`
- `orphaned`:  `check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time <  now-18h`

### CheckOut guard

After loading the attendance, before window validation: if
`attendance.SalaryRejectReason != nil && attendance.CheckOutTime == nil` → return
`"Ca làm việc đã bị tự động từ chối do quá giờ tan ca."`. The existing
`validateCheckOutWindow` upper-bound check still handles the gap before the asynq
task fires.

### DI

New port so `AttendanceService` stays testable without Redis:

```go
type AttendanceTaskEnqueuer interface {
    EnqueueAutoRejectCheckout(ctx context.Context, attendanceID uint, at time.Time) error
}
```

Implemented by the asynq client wrapper; injected into `AttendanceService` via its
constructor; wired in bootstrap. Tests inject a fake that records the call.

### Surfacing

- **Admin list** (`handlers/admin/attendance.go`): already returns `Status`,
  `EarningAmount`, `SalaryRejectReason` — "rejected" flows through automatically.
  The status filter gains a `rejected` option via the `buildFilterQuery` change.
- **Employee view**: status returns `"rejected"` with the reason. A small frontend
  status label for `rejected` may be needed (flagged as a follow-up, out of scope for
  the backend implementation).
- **No cron job** — this is an asynq task, so it does not appear in cron-health.

### Edge cases

- **Night shift / cross-midnight**: `resolveShift` ±1-day candidates yield the correct
  absolute `K+4h`; `ProcessAt` takes an absolute instant.
- **Employee checks out before the task fires**: handler sees `CheckOutTime != nil` → skip.
- **asynq/server down at the deadline**: asynq persists scheduled tasks and runs
  overdue ones on recovery; `MaxRetry` covers handler failures.
- **Shift unresolvable at check-in**: impossible — `CheckIn` already rejects with
  "Chưa cấu hình ca làm việc" when `shift == nil`, so every successful check-in has a
  valid `latestCheckout`.
- **Late checkout after rejection**: blocked by the `CheckOut` guard.

### Testing

- `GetStatus` rejected derivation; `buildFilterQuery` rejected/checked_in/orphaned SQL.
- `AutoRejectIfExpired` idempotency: skip when checked out, skip when already rejected, reject when open.
- `CheckOut` guard blocks a rejected record.
- `CheckIn` enqueues the task at `K+4h` via a fake `AttendanceTaskEnqueuer` (capture
  `(attendanceID, at)`; assert after-commit ordering).
- Night-shift deadline + window math using the existing injected `clock.Clock`.

## Change footprint (small, surgical)

1. `domain/attendance.go` — new `AttendanceStatusRejected` + `GetStatus` branch.
2. `infra/persistence/attendance_repository.go` — `buildFilterQuery` rejected case + tightened cases.
3. `infra/asynq/client.go` — `TaskAutoRejectCheckout` + `EnqueueAutoRejectCheckout`.
4. New port `AttendanceTaskEnqueuer` (domain/ports) + implementor.
5. `app/services/attendance/attendance_service.go` — constructor dep, `CheckIn` after-commit enqueue, `AutoRejectIfExpired`, `CheckOut` guard.
6. asynq mux — register `HandleAutoRejectCheckout`.
7. bootstrap — wire the enqueuer into `AttendanceService` + handler registration.
8. Tests for each of the above.

No migration. No cron job.
