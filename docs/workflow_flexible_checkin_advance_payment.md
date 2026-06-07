# Flexible Employee: Check-in/Check-out → Timesheet → Advance Payment Workflow

## Overview

Flexible (linh hoạt) employees have a self-service workflow where they **check-in at a project site**, **check-out when done**, and can immediately **request an advance payment** against their earned wages — all from their mobile phone. No admin or partner intervention needed for daily attendance.

## Actors

| Actor | Role |
|-------|------|
| **Flexible Employee** | Self-service check-in/out + advance payment requests via mobile PWA |
| **Admin** | Imports flex-pay employee list, manages bank transfer results, reconciles |
| **System** | Auto-calculates earning amount on check-out, accumulates advance quota |

## Prerequisites

1. **Project must be flexible**: `project.IsFlexible = true`
2. **Employee assignment has `check_in_enabled = true`**: Admin enables this per employee-project assignment
3. **Geofence gates configured**: Project must have geofence gates with lat/lng coordinates and a radius
4. **Payrate configured**: Active payrate for the project matching `position.shift.HH:MM-HH:MM` time ranges

---

## Step-by-Step Workflow

### Step 1: Employee Check-In

**Endpoint**: `POST /api/v1/mobile/attendance/check-in`
**Handler**: `backend/internal/transport/http/handlers/attendance/attendance.go:CheckIn`
**Service**: `backend/internal/app/services/attendance/attendance_service.go:CheckIn`

**Request body**:
```json
{ "lat": 10.12345, "lng": 106.12345, "project_id": 0 }
```
- `project_id` is optional (0 = auto-detect from employee's active flexible projects)

**Validation sequence** (all within a single DB transaction):
1. **Resolve project** — If `project_id` omitted, auto-detect the employee's single active flexible project. If multiple flexible projects exist, employee must specify. Project must have `IsFlexible = true`.
2. **Check `check_in_enabled`** — Validates the employee's assignment has check-in enabled. Returns "Bạn chưa được cấp quyền chấm công" if disabled.
3. **Shift time validation (±1 hour)** — Fetches active payrate for the project and validates that the current time falls within ±1 hour of any configured shift's **start time** for the employee's position. If no payrate exists or no shift matches, the check-in is **rejected** — no attendance record is saved (see [Shift Time Validation Rules](#shift-time-validation-rules)).
4. **Geofence validation** — Haversine distance from each gate; must be within `project.GeofenceRadiusMeters`. Returns error if no gates configured or outside all gate radii.
5. **One check-in per day** — Enforces 1 attendance record per employee per calendar day. Checks both today and yesterday (for night shift support).
6. **Create attendance record** — `status = "checked_in"`, stores check-in time, coords, gate name.

**Result**: `Attendance` record with `CheckInTime`, `CheckInGate`, `status = "checked_in"`

### Step 2: Employee Check-Out

**Endpoint**: `POST /api/v1/mobile/attendance/check-out`
**Handler**: `backend/internal/transport/http/handlers/attendance/attendance.go:CheckOut`
**Service**: `backend/internal/app/services/attendance/attendance_service.go:CheckOut`

**Request body**:
```json
{ "lat": 10.12345, "lng": 106.12345 }
```
- No `project_id` needed — determined from the existing check-in record

**Validation sequence** (all within a single DB transaction):
1. **Load active check-in** — Finds today's unchecked-out record. Falls back to yesterday's record for night shift support. Returns error if no valid check-in found.
2. **Prevent double check-out** — Rejects if `CheckOutTime` is already set (`IsCompleted()`).
3. **Orphaned check** — Rejects if check-in was more than 18 hours ago (`GetStatus(now) == "orphaned"`). Returns "Ca làm việc đã quá hạn check-out".
4. **Geofence validation** — Same as check-in: loads project, validates coordinates against configured gates.
5. **Update CheckOutTime and coords** — Sets `CheckOutTime`, coordinates, and gate name on the attendance record.
6. **Re-check `check_in_enabled`** — Guard added to prevent quota accumulation if admin disabled check-in between check-in and check-out. Returns "Chấm công đã bị vô hiệu hóa" if disabled.
7. **Shift time validation (±1 hour)** — Validates that the current time falls within ±1 hour of any configured shift's **end time** for the employee's position. If no shift matches, the check-out is **rejected** — the attendance record is not updated (see [Shift Time Validation Rules](#shift-time-validation-rules)).
8. **Calculate earning amount** — Matches `checkInTime → checkOutTime` against payrate shift definitions (see [Earning Calculation Rules](#earning-calculation-rules)). On calculation failure, defaults to 0 (logs warning).
9. **Update attendance record** — Persists `EarningAmount`, sets `status = "completed"`.
10. **Accumulate advance payment quota** — Only if `earningAmount > 0`:
   - Looks up (or creates) `AdvancePayment` record for `employee + project + current month`
   - If no record exists: creates new `AdvancePayment` with `MaxAdvAmount = earningAmount`
   - If record exists: atomically increments via `IncrementMaxAdvAmount(id, earningAmount)` → `UPDATE advance_payments SET max_adv_amount = max_adv_amount + ?`
   - This quota becomes immediately available for advance payment requests

**Result**: `Attendance` record with `EarningAmount`, `status = "completed"`, and advance quota incremented (if earning > 0)

### Step 3: Employee Requests Advance Payment

**Endpoint**: `POST /api/v1/me/advance-payment/request`
**Handler**: `backend/internal/transport/http/handlers/advance_payment/create_request_handler.go`
**Service**: `backend/internal/app/services/advance_payment/service.go:CreateRequest`

**Request body**:
```json
{ "amount": 200000, "forMonth": "2026-06" }
```
- Minimum amount: 10,000 VND
- `forMonth` defaults to current advance period month

**Validation sequence**:
1. **Minimum amount check** — `requestAmount < 10000` returns error.
2. **Flexible schedule check** — Employee must have an active flexible assignment with `PaymentScheduleFlexible` and `LastDate == nil`.
3. **Three-phase window rule** — If `IsInLockedGap(now)` (days 11-20), request is rejected (see [Three-Phase Calendar Window](#three-phase-calendar-window)).
4. **Month validity check** — If before cutoff (days 1-10): allows `forMonth == previousMonth OR forMonth == currentMonth`. Otherwise (days 21+): only allows `forMonth == currentMonth`.
5. **Advance payment record check** — Fetches `AdvancePayment` records for employee+month. If none exist, returns "salary info not found" error.
6. **Fee calculation** — Server-side via `FeeResolver.ResolveFee(ctx, amount, at)`. Defensive cap: if fee >= amount, caps to `(amount, 0)`. Otherwise: `netAmount = requestAmount - fee`.
7. **Provider transfer limits** — Net amount (after fee) must be within provider min/max.
8. **Atomic budget check** — `CreateWithBudgetCheck` prevents TOCTOU race conditions:
   - Locks employee's `advance_payments` rows with `SELECT ... FOR UPDATE`
   - Calculates: `usedTotal = SUM(pending + approved + completed request amounts)`
   - Calculates: `remaining = maxAdvAmount - usedTotal`
   - Rejects if `usedTotal + requestAmount > maxAdvAmount`

**Result**: `AdvancePaymentRequest` with `status = "pending"`, fee and net amount calculated

### Step 4: Admin Processes Payment

Admin approves pending requests → exports approved requests → sends to bank → uploads bank result → system marks as `completed`/`failed`. Employees can cancel requests while in `pending` or `approved` status.

---

## Three-Phase Calendar Window

The advance payment system uses a **monthly cycle** with three phases:

```
Day of Month:  1  ...  10  |  11  ...  20  |  21  ...  31
               ────────────   ─────────────   ─────────────
Phase:         Before Cutoff  Locked Gap     New Period
               (can request)  (blocked)      (can request)
```

| Phase | Days | Can Request | Available Months |
|-------|------|-------------|-----------------|
| **Before Cutoff** | 1–10 | ✅ Yes | Previous month + Current month |
| **Locked Gap** | 11–20 | ❌ No | — (admin processing period) |
| **New Period** | 21–31 | ✅ Yes | Current month |

**Constants**:
- `PeriodCycleStartDay = 21` (start of new period)
- `RequestCutoffDay = 10` (end of previous period's window)

**Source**: `backend/internal/pkg/clock/advance_payment.go`

---

## Shift Time Validation Rules

Both check-in and check-out times are validated against configured shift times with a **±1 hour tolerance window**:

- **Check-in** must fall within ±1 hour of any shift's **start time** for the employee's position
- **Check-out** must fall within ±1 hour of any shift's **end time** for the employee's position
- **REJECT** if either condition fails — no attendance data is saved/updated
- Validation is independent per operation: check-in validates against shift starts, check-out validates against shift ends

**Tolerance constant**: `shiftTimeTolerance = 1 * time.Hour`

### How it works

The system fetches the active payrate for the project and parses all shift definitions (`position.dayType.HH:MM-HH:MM`) matching the employee's position. It then computes the circular distance (in minutes) between the current time and each shift boundary. If any shift is within tolerance, the check-in/check-out proceeds.

### Day Shift Examples (08:00 – 17:00)

**Check-in window**: [07:00 – 09:00] (±1h of shift start 08:00)

| Check-in Time | Verdict | Reason |
|---------------|---------|--------|
| 08:00 | ✅ ACCEPT | Exact shift start |
| 07:01 | ✅ ACCEPT | Within ±1h (59min before) |
| 08:59 | ✅ ACCEPT | Within ±1h (59min after) |
| 06:59 | ❌ REJECT | 61min before shift start |
| 09:01 | ❌ REJECT | 61min after shift start |

**Check-out window**: [16:00 – 18:00] (±1h of shift end 17:00)

| Check-out Time | Verdict | Reason |
|----------------|---------|--------|
| 17:00 | ✅ ACCEPT | Exact shift end |
| 16:01 | ✅ ACCEPT | Within ±1h (59min before) |
| 17:59 | ✅ ACCEPT | Within ±1h (59min after) |
| 15:59 | ❌ REJECT | 61min before shift end |
| 18:01 | ❌ REJECT | 61min after shift end |

### Night Shift Examples (22:00 – 06:00, crosses midnight)

**Check-in window**: [21:00 – 23:00] (±1h of shift start 22:00)

| Check-in Time | Verdict | Reason |
|---------------|---------|--------|
| 22:00 | ✅ ACCEPT | Exact shift start |
| 21:01 | ✅ ACCEPT | Within ±1h (59min before) |
| 22:59 | ✅ ACCEPT | Within ±1h (59min after) |
| 20:59 | ❌ REJECT | 61min before shift start |
| 23:01 | ❌ REJECT | 61min after shift start |

**Check-out window**: [05:00 – 07:00] (±1h of shift end 06:00, wraps across midnight)

| Check-out Time | Verdict | Reason |
|----------------|---------|--------|
| 06:00 | ✅ ACCEPT | Exact shift end |
| 05:01 | ✅ ACCEPT | Within ±1h (59min before) |
| 06:59 | ✅ ACCEPT | Within ±1h (59min after) |
| 04:59 | ❌ REJECT | 61min before shift end |
| 07:01 | ❌ REJECT | 61min after shift end |

> **Note**: Shift time validation is a guardrail, not an earning guarantee. Passing validation does NOT guarantee earning > 0. The earning calculation (full-shift coverage: `checkIn <= shiftStart AND checkOut >= shiftEnd`) remains independent. For example, checking in at 08:59 and out at 17:00 passes both validations but earns 0 (08:59 > 08:00 — doesn't cover shift start).

---

## Earning Calculation Rules

When an employee checks out, the system calculates the earning amount by matching the actual check-in/check-out times against configured payrate shift definitions.

**Matching logic** (`calculateEarningAmount` in `attendance_service.go:314`):

1. Flattens the payrate structure into key-value pairs where key format is `position.dayType.HH:MM-HH:MM`
2. For each entry matching the employee's position:
   - Parses the time range (e.g., `08:00-17:00`)
   - Builds real shift start/end times on the check-in date
   - Handles night shifts crossing midnight (adds 24h to end time)
   - For night shifts where check-in is before shiftStart (e.g., 00:30 check-in for 22:00-06:00 shift), rolls shift times back 24 hours
3. **Match condition**: `checkIn <= shiftStart AND checkOut >= shiftEnd`
   - The employee must have been present for the **entire** shift duration
   - Arriving on time or early + leaving on time or late = earns the shift rate
   - Arriving late or leaving early = **no match** for that shift → earning = 0
4. Returns the matched shift's payrate amount, or 0 if no shift matches

### Day Shift Example (08:00 – 17:00, rate: 500,000 VND)

| Check-in | Check-out | Earning | Reason |
|----------|-----------|---------|--------|
| 08:00 | 17:00 | 500,000 | Exact shift coverage |
| 07:30 | 17:30 | 500,000 | Early arrival, late departure — still covers full shift |
| 06:00 | 18:00 | 500,000 | Very early/late — still covers full shift |
| 08:30 | 17:00 | 0 | Late arrival (after shiftStart) — does NOT cover full shift |
| 08:00 | 16:00 | 0 | Early departure (before shiftEnd) — does NOT cover full shift |
| 09:00 | 16:00 | 0 | Both late arrival AND early departure |

### Night Shift Example (22:00 – 06:00, rate: 600,000 VND)

| Check-in | Check-out | Earning | Reason |
|----------|-----------|---------|--------|
| 22:00 | 06:00 (next day) | 600,000 | Exact shift coverage |
| 21:30 | 06:30 (next day) | 600,000 | Early/late — still covers full shift |
| 22:30 | 06:00 (next day) | 0 | Late arrival — does NOT cover full shift |
| 22:00 | 05:00 (next day) | 0 | Early departure — does NOT cover full shift |

> **Important**: Unlike the check-in/out validation (which only checks geofence and daily limits), the earning calculation is purely about shift coverage. There is no time window tolerance — either the employee's presence covers the entire shift or they earn nothing for that shift.

---

## Data Flow Diagram

```
┌─────────────┐     POST /check-in      ┌──────────────────┐
│  Employee    │ ───────────────────────▶│  Attendance       │
│  Mobile App  │     (lat, lng)          │  Service          │
│  (PWA)       │                         │  (CheckIn)        │
│              │     POST /check-out     │                    │
│              │ ───────────────────────▶│  (CheckOut)       │
└──────┬───────┘     (lat, lng)          └────────┬──────────┘
       │                                          │
       │                                          │ On check-out:
       │                                          │ If earning > 0:
       │                                          │   AdvancePayment.MaxAdvAmount += earning
       │                                          ▼
       │                              ┌───────────────────────┐
       │                              │  advance_payments      │
       │                              │  table (quota)         │
       │                              └───────────┬────────────┘
       │                                          │
       │    POST /advance-payment/request         │
       │ ────────────────────────────────────────▶│
       │    { amount, forMonth }                  │
       │                                          │ Budget check:
       │                                          │ remaining = MaxAdv - (pending + approved + completed)
       │                                          │ remaining >= amount?
       │                                          ▼
       │                              ┌───────────────────────┐
       │                              │  advance_payment_      │
       │                              │  requests table        │
       │                              │  (status: pending)     │
       │                              └───────────────────────┘
```

---

## Key Domain Entities

### Attendance (`backend/internal/domain/attendance.go`)
- **Statuses**: `checked_in` → `completed` | `orphaned` (if >18h without check-out)
- **EarningAmount**: Set on check-out based on payrate shift matching. Nullable `*int64` — null until check-out.
- **"orphaned" is computed dynamically** — `GetStatus(now)` checks `now - CheckInTime > 18h`. Not stored in DB.

### AdvancePayment (`backend/internal/domain/advance_payment.go`)
- Tracks **quota** per employee + project + month
- `MaxAdvAmount` = total earned from check-outs + admin flex-pay import
- Accumulated via `IncrementMaxAdvAmount` on each check-out (atomic SQL increment)
- Can be zeroed out via `ZeroOutQuota` when admin disables check-in

### AdvancePaymentRequest (`backend/internal/domain/advance_payment_request.go`)
- **Statuses**: `pending` → `approved` → `completed` | `failed`; `cancelled` available from both `pending` and `approved`
- Stores `request_amount`, `fee`, `net_amount`
- Budget check sums `PENDING + APPROVED + COMPLETED` amounts against `MaxAdvAmount`

---

## Frontend Components

| Component | File | Purpose |
|-----------|------|---------|
| **FlexiblePayEmployeePage** | `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx` | Main employee self-service page |
| **EmployeeCheckInCard** | `frontend/src/components/employees/EmployeeCheckInCard.tsx` | Check-in/out UI with geolocation |
| **EmployeeAttendanceHistoryCard** | `frontend/src/components/employees/EmployeeAttendanceHistoryCard.tsx` | Recent attendance history |
| **AdvancePaymentLimitCard** | `frontend/src/components/advance-payment/AdvancePaymentLimitCard.tsx` | Shows quota, used/remaining |
| **AdvancePaymentRequestForm** | `frontend/src/components/advance-payment/AdvancePaymentRequestForm.tsx` | Amount input with live fee calc |
| **AdvancePaymentConfirmSheet** | `frontend/src/components/advance-payment/AdvancePaymentConfirmSheet.tsx` | Confirm before submitting |
| **AdvancePaymentHistoryCard** | `frontend/src/components/advance-payment/AdvancePaymentHistoryCard.tsx` | Past requests + cancel pending |

**Conditional rendering**: Check-in/out UI only shown when `profile.check_in_enabled === true`

---

## API Endpoints (Employee-facing)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/v1/mobile/attendance/check-in` | Check in at project |
| POST | `/api/v1/mobile/attendance/check-out` | Check out from project |
| GET | `/api/v1/mobile/attendance/today` | Get today's attendance status |
| GET | `/api/v1/mobile/attendance/history` | Attendance history (paginated, default 20) |
| GET | `/api/v1/me/advance-payment` | Get advance payment info (quota) |
| POST | `/api/v1/me/advance-payment/request` | Request advance payment |
| POST | `/api/v1/me/advance-payment/calculate-fee` | Preview fee |
| GET | `/api/v1/me/advance-payment/history` | Request history |
| POST | `/api/v1/me/advance-payment/request/:id/cancel` | Cancel pending request |

---

## API Endpoints (Admin-facing)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/admin/attendances` | List all attendances (filtered) |
| GET | `/api/v1/admin/attendances/:id` | Get attendance by ID |
| PATCH | `/api/v1/projects/:id/employees/:employeeId/checkin-enabled` | Toggle check-in for employee |
| PATCH | `/api/v1/projects/:id/employees/checkin-enabled/bulk` | Bulk toggle check-in |

---

## Key Design Decisions

1. **Quota accumulates in real-time** — Each check-out immediately increments `MaxAdvAmount`, making earnings available for advance requests without admin intervention
2. **Geofence enforcement** — Both check-in and check-out require being within project's configured radius
3. **Shift-based earning (full-coverage match)** — Earning is determined by whether the employee's check-in/out times fully cover a configured payrate shift (`checkIn <= shiftStart AND checkOut >= shiftEnd`). The entire shift must be covered. If no shift matches, earning = 0.
4. **Shift time validation (±1 hour)** — Check-in and check-out times must fall within ±1 hour of the configured shift start/end times respectively. Attendance records outside these windows are rejected entirely — no data is persisted. This prevents phantom check-ins at random hours that could never earn money.
5. **Atomic budget check** — `CreateWithBudgetCheck` uses `SELECT ... FOR UPDATE` to prevent race conditions when multiple requests arrive simultaneously. Budget formula: `remaining = MaxAdvAmount - (pending + approved + completed amounts)`.
6. **Three-phase window** — Locks requests during days 11-20 to allow admin time to process bank transfers
7. **Night shift support** — Check-out can match against yesterday's check-in; shifts crossing midnight handled; earning calculation adjusts shift times for midnight boundary
8. **Guard against mid-session disable** — Check-out re-validates `check_in_enabled` to prevent quota accumulation if admin disabled check-in between employee's check-in and check-out

---

## Known Limitations

1. **One check-in per day is global** — The unique constraint is on `(employee_id, date)` without `project_id`, so an employee can only check in once per calendar day across all projects
2. **`orphaned` status is derived at read-time** — Not stored in DB; `GetStatus(now)` computes it dynamically by checking `now - CheckInTime > 18h`. The `GetOrphanCandidates` repository method exists but has no callers (no cleanup cron)
3. **No GPS spoofing prevention** — The system trusts client-reported lat/lng without accuracy checks, altitude verification, or anti-fraud measures
4. **No audit logging for attendance** — Unlike advance payment imports (which use `AuditService`), attendance check-in/out operations have no structured audit trail
5. **No earning correction mechanism** — `AdvancePaymentRepository` has `IncrementMaxAdvAmount` and `ZeroOutQuota` but no `DecrementMaxAdvAmount`. If an attendance earning is incorrect, there is no admin endpoint to correct it; the advance payment quota remains inflated
6. **Timezone is hardcoded** — All business time uses `Asia/Ho_Chi_Minh` via `clock.Now()`. The "today" boundary for check-in/out is always Vietnam time regardless of employee location
7. **Zero earning can still occur within valid window** — Even with ±1h shift validation, an employee who checks in late within the window (e.g., 08:59 for an 08:00 shift) and checks out on time will pass both validations but earn 0 (doesn't cover full shift). No warning is returned — the employee may not realize they earned nothing until checking their advance payment quota
8. **Earning calculation failure is silent** — If `calculateEarningAmount` returns an error (e.g., malformed payrate config), the system logs a warning and sets earning to 0 rather than failing the check-out
9. **No payrate = check-in rejected** — If no active payrate is configured for the project on the check-in date, the check-in is rejected entirely. This is a new constraint (previously check-in did not require payrate)
