# Visual Explanation: Check-In / Check-Out Flow

> Self-checkin attendance for **flexible** projects. Employee uses the mobile app
> to check in at a gate, work their shift, and check out — earning a flat per-shift
> wage only when both the check-in and check-out land inside configured windows.
> Source of truth: `backend/internal/app/services/attendance/attendance_service.go`
> (values below reflect the current working tree, where `checkOutUpperGrace = 4h`).

## Overview

The flow spans four layers, plus an asynchronous safety net:

| Layer | Component | Role |
|---|---|---|
| Mobile | `EmployeeCheckInCard`, `geolocation.ts` | Capture GPS (watchPosition convergence), POST to API |
| Handler | `attendance.Handler` (`transport/http/handlers/attendance/attendance.go`) | Resolve `Employee.ID` from JWT, build `GeoReading`, map response, log failed attempts |
| Service | `attendance.AttendanceService` | Business rules: project, geofence, shift, windows, earning, auto-reject scheduling |
| Async | `AutoRejectCheckoutWorker` (asynq) | At K+4h, reject shifts left open with no checkout |

**Time anchors (the whole flow is built on these):**
- **T** = configured shift **start**
- **K** = configured shift **end**
- All times via `clock.Now()` in `Asia/Ho_Chi_Minh` (`time.Local`).

## Quick View (ASCII)

### Component pipeline

```
 ┌──────────┐  POST /check-in   ┌────────────┐                    ┌──────────────────┐
 │  Mobile  │ ────────────────▶ │  Handler   │ ──service.CheckIn▶ │ AttendanceService│
 │  (GPS)   │ ◀──── 200/4xx ─── │            │                    │  + transaction   │
 └──────────┘                   └────┬───────┘                    └────────┬─────────┘
   watchPosition                     │ recordFailedAttempt                  │
   convergence                (best-effort, fire-and-forget)              │
                                     ▼                                       ▼
                          ┌──────────────────┐   validateGeofence   ┌─────────────┐
                          │ FailedAttemptRepo│ ◀────────────────── │ Project /    │
                          │ (admin dashboard)│                      │ Payrate repos│
                          └──────────────────┘                      └─────────────┘

   On commit ──▶ asynq task scheduled at K+4h ──▶ AutoRejectCheckoutWorker
                                                       │ AutoRejectIfExpired
                                                       ▼
                                          if CheckOutTime == nil → earning 0 + rejected
```

### Time-window timeline (the heart of the flow)

```
              CHECK-IN window            CHECK-OUT window
              (T-1h ──────── T+1h)       [K-1h ──────── K ──────── K+4h]
                      open                         closed

  ─────┬───────┬───────────────┬────────┬────────┬────────┬──────────┬────────▶ time
       T-1h     T              T+1h     K-1h      K        K+?        K+4h
                                                              │
                                          (checkout still allowed anywhere in [K-1h, K+4h])
                                                              │
                                                  auto-reject fires ◀── asynq, only if no checkout
```

- **Check-in** valid strictly inside `(T-1h, T+1h)` — `checkInShiftWindow = 1h`.
- **Check-out** valid across `[K-1h, K+4h]` — `checkOutLowerGrace = 1h`, `checkOutUpperGrace = 4h`.
- A shift **earns** only when the attendance fits **both** windows simultaneously (`calculateEarningAmount` matches a shift satisfying `checkInFitsShift` **and** `checkOutFitsShift`).

## Detailed Flow

### 1. Check-In — `AttendanceService.CheckIn`

```mermaid
flowchart TD
    Start(["POST /api/v1/mobile/attendance/check-in<br/>{project_id?, lat, lng, accuracy, gps_at}"]) --> Proj["resolveProject<br/>(given id, or auto-detect)"]
    Proj --> ProjCheck{"exactly one<br/>flexible project<br/>& assignment active<br/>& check_in_enabled?"}
    ProjCheck -->|"0 or >1 / disabled"| ErrPerm["ValidationError → 422<br/>(handler logs failed attempt)"]
    ProjCheck -->|"yes"| Geo["validateGeofence(project, geo)<br/>accuracy &le; radius AND<br/>dist+accuracy &le; radius"]
    Geo -->|"outside / too inaccurate"| ErrGeo["ValidationError → 422"]
    Geo -->|"inside a gate"| Dup{"checked in today<br/>&amp; not a prior<br/>no-salary checkout?"}
    Dup -->|"yes"| ErrDup["ValidationError: đã vào làm"]
    Dup -->|"no"| Shift["resolveShift(payrate, position, now)<br/>generate &plusmn;1-day candidates,<br/>closestShift picks nearest"]
    Shift --> HasShift{"shift resolved?"}
    HasShift -->|"nil config"| ErrShift["ValidationError: chưa cấu hình ca"]
    HasShift -->|"ok"| Win{"checkInFitsShift?<br/>now in (T-1h, T+1h)"}
    Win -->|"no"| ErrWin["ValidationError: ngoài khung giờ vào"]
    Win -->|"yes"| Create["attendanceRepo.Create<br/>{Date, CheckInTime, gate, geo}"]
    Create --> Sched["RegisterAfterCommit:<br/>EnqueueAutoReject(id, K+4h)"]
    Sched --> Commit[("TX commit")]
    Commit --> Resp["200 OK<br/>salary_status = pending"]
```

### 2. Check-Out — `AttendanceService.CheckOut`

```mermaid
flowchart TD
    Start(["POST /api/v1/mobile/attendance/check-out<br/>{lat, lng, accuracy, gps_at, confirm_no_salary}"]) --> Find["Load today's attendance<br/>(fallback yesterday — night shift)"]
    Find --> Has{"open record found?"}
    Has -->|"none"| ErrNone["ValidationError: không tìm thấy vào làm"]
    Has -->|"found"| Done{"IsCompleted / already rejected<br/>(SalaryRejectReason) / orphaned?"}
    Done -->|"yes"| ErrDone["ValidationError: ca đã đóng"]
    Done -->|"open"| Shift2["resolveShift from CheckInTime"]
    Shift2 --> Win{"checkOutFitsShift?<br/>now in [K-1h, K+4h]"}
    Win -->|"yes"| Geo2["validateGeofence (checkout)<br/>same contract as check-in"]
    Win -->|"no"| Confirm{"confirm_no_salary == true?"}
    Confirm -->|"no"| ErrWin2["ValidationError: ngoài khung tan ca"]
    Confirm -->|"yes"| Force["forcedNoSalary<br/>earning = 0 (audited override)"]
    Force --> Geo2
    Geo2 -->|"fail"| ErrGeo2["ValidationError → 422"]
    Geo2 -->|"ok"| Earn["calculateEarningAmount<br/>flat amount if a shift fits BOTH windows;<br/>else earning 0 + reject reason"]
    Earn --> Write["Set CheckOutTime, CheckOutGate, geo;<br/>write earning_amount / reason"]
    Write --> Commit[("TX commit")]
    Commit --> Resp["200 OK<br/>salary_status = recorded | not_recorded"]
```

### 3. Auto-Reject safety net — async, fired once at K+4h

```mermaid
sequenceDiagram
    autonumber
    participant CI as CheckIn (in TX)
    participant TX as Transaction
    participant Enq as TaskEnqueuer (asynq)
    participant W as AutoRejectCheckoutWorker
    participant Svc as AutoRejectIfExpired
    CI->>TX: RegisterAfterCommit(EnqueueAutoReject id, K+4h)
    CI->>TX: commit (attendance row now durable)
    TX->>Enq: EnqueueAutoRejectCheckout(id, deadline=K+4h)
    Note over Enq: task dormant until K+4h
    Note over Enq,W: ⏰ at K+4h — asynq delivers
    Enq->>W: ProcessJob(attendanceID)
    W->>Svc: AutoRejectIfExpired(ctx, id)
    alt CheckOutTime still nil
        Svc->>Svc: earning = 0<br/>SalaryRejectReason = auto-reject
    else already checked out / rejected
        Svc-->>W: nil (idempotent)
    end
    Svc-->>W: nil
```

### 4. Attendance lifecycle

```mermaid
stateDiagram-v2
    [*] --> CheckedIn: CheckIn passes geofence + (T-1h,T+1h)
    CheckedIn --> Completed: CheckOut in [K-1h,K+4h] → earning recorded
    CheckedIn --> NoSalary: CheckOut outside window + confirm_no_salary (earning 0)
    CheckedIn --> AutoRejected: K+4h reached, no checkout (asynq)
    Completed --> [*]
    NoSalary --> [*]
    AutoRejected --> [*]
    note right of CheckedIn
        mapToResponse surfaces salary_status:
        pending / recorded / not_recorded
    end note
```

## Key Concepts

1. **T and K, not wall-clock duration.** There is no "worked N hours" math. Earning is a **flat per-shift amount** from the payrate config; the windows only gate *whether* the shift counts. A checkout the gate accepts is *guaranteed* to earn, because `calculateEarningAmount` reuses the exact same `checkInFitsShift` / `checkOutFitsShift` predicates the gates use.

2. **Geofence accuracy gate (anti-spoof).** `validateGeofence` rejects a fix whose uncertainty circle can't fit inside the gate: `dist + accuracy ≤ radius`. `accuracy > radius` is rejected outright; an on-gate coordinate with low accuracy is held as "inside but uncertain" and rejected. `accuracy == 0` (legacy client, unknown) skips the uncertainty check so legitimate fixes still pass. *(Device-reported GPS can still be spoofed — this only stops sloppy/misreported fixes.)*

3. **Shift resolution uses ±1-day candidates, not a rollback hack.** `resolveShifts` generates each configured shift starting on the day before, the day of, and the day after check-in (cross-midnight aware), then `closestShift` prefers a shift that actually *contains* the check-in before falling back to nearest start. This anchors night shifts and early arrivals (e.g. 19:50 into a 20:00–04:00 shift) to the correct calendar day.

4. **No fallback when config is missing.** If no payrate/position/shift resolves (`shift == nil`), the action is **rejected** — it never silently falls back to a fixed duration. The same is true for a project with no `GeofenceGates`.

5. **One check-in per day — with one escape hatch.** A second check-in is rejected unless the prior record was a *confirmed-no-salary* checkout, in which case the employee may start a fresh shift.

6. **Auto-reject is scheduled, not polled.** The task is enqueued at K+4h inside the check-in transaction but only **after commit** (`RegisterAfterCommit`), so it never fires for a rolled-back check-in. `AutoRejectIfExpired` is idempotent, so asynq retries are safe; a checkout that lands before K+4h makes the task a no-op.

7. **Failed attempts are forensic, never blocking.** When a check-in/checkout fails validation, the handler fire-and-forgets a row to `AttendanceFailedAttemptRepository` (using a detached 3s context so the write survives response cancellation). This feeds the admin **Check-In Health** dashboard; it never changes the response to the employee.

## Code Anchors

| Concern | Location |
|---|---|
| HTTP endpoints + failed-attempt logging | `backend/internal/transport/http/handlers/attendance/attendance.go` (`CheckIn` ~L153, `CheckOut` ~L187, `LogDeviceAttempt` ~L240) |
| Window constants | `attendance_service.go` L18–27 (`checkInShiftWindow=1h`, `checkOutLowerGrace=1h`, `checkOutUpperGrace=4h`) |
| Geofence accuracy gate | `attendance_service.go` `validateGeofence` ~L290 |
| Shift resolution (±1-day candidates) | `attendance_service.go` `resolveShifts` ~L160, `closestShift` ~L237, `resolveShift` ~L267 |
| Check-in / checkout orchestration | `attendance_service.go` `CheckIn` L406, `CheckOut` L501 |
| Earning match (both-window predicate) | `attendance_service.go` `calculateEarningAmount` L897 |
| Auto-reject scheduling | `attendance_service.go` ~L482 (`RegisterAfterCommit` → `EnqueueAutoRejectCheckout`) |
| Auto-reject worker | `backend/internal/app/workers/auto_reject_checkout_worker.go` (`AutoRejectCheckoutWorker.ProcessJob`) |
| Salary-status response mapping | `attendance.go` `mapToResponse` ~L115 |
