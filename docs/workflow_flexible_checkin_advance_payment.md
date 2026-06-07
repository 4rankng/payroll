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
4. **Payrate configured**: Active payrate for the project matching `position.shift.checkin-checkout` time ranges

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

**Validation sequence**:
1. **Resolve project** — If `project_id` omitted, auto-detect the employee's single active flexible project. If multiple flexible projects exist, employee must specify.
2. **Check `check_in_enabled`** — Validates the employee's assignment has check-in enabled
3. **Geofence validation** — Haversine distance from each gate; must be within `project.GeofenceRadiusMeters`
4. **One check-in per day** — Enforces 1 attendance record per employee per calendar day
5. **Shift time validation** — Check-in time must fall within ±1 hour of a configured shift's start time (see [Shift Time Validation Rules](#shift-time-validation-rules)). If no shift matches, the check-in is **rejected** — no attendance record is saved.
6. **Create attendance record** — `status = "checked_in"`, stores check-in time, coords, gate name

**Result**: `Attendance` record with `CheckInTime`, `CheckInGate`, `status = "checked_in"`

### Step 2: Employee Check-Out

**Endpoint**: `POST /api/v1/mobile/attendance/check-out`
**Handler**: `backend/internal/transport/http/handlers/attendance/attendance.go:CheckOut`
**Service**: `backend/internal/app/services/attendance/attendance_service.go:CheckOut`

**Request body**:
```json
{ "lat": 10.12345, "lng": 106.12345 }
```

**Validation sequence**:
1. **Load active check-in** — Finds today's (or yesterday's for night shifts) unchecked-out record
2. **Prevent double check-out** — Rejects if already `completed` or `orphaned` (>18h since check-in)
3. **Geofence validation** — Same as check-in
4. **Shift time validation** — Check-out time must fall within ±1 hour of the matched shift's end time (see [Shift Time Validation Rules](#shift-time-validation-rules)). If the check-out time is outside the allowed window, the check-out is **rejected** — the attendance record is not updated.
5. **Calculate earning amount** — Matches `checkInTime → checkOutTime` against payrate shift definitions:
   - Payrate keys follow pattern: `position.dayType.HH:MM-HH:MM`
   - If the worked period fully covers a shift's start→end, employee earns that shift's rate
   - Night shifts crossing midnight are handled
6. **Update attendance record** — Sets `CheckOutTime`, coords, gate, `EarningAmount`, `status = "completed"`
7. **Accumulate advance payment quota** — This is the critical step:
   - Looks up (or creates) `AdvancePayment` record for `employee + project + current month`
   - `AdvancePayment.MaxAdvAmount += earningAmount` (via `IncrementMaxAdvAmount`)
   - This quota becomes immediately available for advance payment requests

**Result**: `Attendance` record with `EarningAmount`, `status = "completed"`, and advance quota incremented

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
1. **Flexible schedule check** — Employee must have flexible payment schedule
2. **Three-phase window rule** (see below)
3. **Quota check** — `remaining = MaxAdvAmount - CompletedAmount - PendingAmount` must be ≥ requested amount
4. **Fee calculation** — Server-side: `fee = amount * feePercentage / 100` (min fee enforced)
5. **Provider transfer limits** — Net amount (after fee) must be within provider min/max
6. **Atomic budget check** — `CreateWithBudgetCheck` prevents TOCTOU race conditions

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

- **Check-in** must fall within `[shift_start - 1h, shift_start + 1h]`
- **Check-out** must fall within `[shift_end - 1h, shift_end + 1h]`
- **REJECT** if either condition fails — no attendance data is saved to the database

### Day Shift Examples (08:00 – 17:00)

| Check-in | Check-out | Verdict | Reason |
|----------|-----------|---------|--------|
| 08:00 | 17:00 | ✅ ACCEPT | Exact shift times |
| 07:00 | 18:00 | ✅ ACCEPT | Both within ±1h window |
| 09:00 | 16:00 | ✅ ACCEPT | Both within ±1h window |
| 08:50 | 17:00 | ✅ ACCEPT | Check-in within window, exact departure |
| 09:00 | 16:45 | ✅ ACCEPT | Both within ±1h window |
| 06:59 | 17:00 | ❌ REJECT | Check-in 1m before window (07:00 boundary) |
| 09:01 | 17:00 | ❌ REJECT | Check-in 1m after window (09:00 boundary) |
| 09:01 | 16:45 | ❌ REJECT | Check-in 1m after window |
| 08:00 | 15:59 | ❌ REJECT | Check-out 1m before window (16:00 boundary) |
| 08:00 | 18:01 | ❌ REJECT | Check-out 1m after window (18:00 boundary) |

### Night Shift Examples (20:00 – 05:00, crosses midnight)

| Check-in | Check-out | Verdict | Reason |
|----------|-----------|---------|--------|
| 20:00 | 05:00 | ✅ ACCEPT | Exact shift times |
| 19:00 | 06:00 | ✅ ACCEPT | Both within ±1h window |
| 21:00 | 04:00 | ✅ ACCEPT | Both within ±1h window |
| 18:59 | 05:00 | ❌ REJECT | Check-in 1m before window (19:00 boundary) |
| 21:01 | 05:00 | ❌ REJECT | Check-in 1m after window (21:00 boundary) |
| 20:00 | 03:59 | ❌ REJECT | Check-out 1m before window (04:00 boundary) |
| 20:00 | 06:01 | ❌ REJECT | Check-out 1m after window (06:00 boundary) |

**Implementation**: Validation occurs in the service layer before any database write. The payrate's shift definitions (`position.dayType.HH:MM-HH:MM`) serve as the source of truth for valid time windows.

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
       │                                          │ AdvancePayment.MaxAdvAmount += earning
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
- **EarningAmount**: Set on check-out based on payrate shift matching

### AdvancePayment (`backend/internal/domain/advance_payment.go`)
- Tracks **quota** per employee + project + month
- `MaxAdvAmount` = total earned from check-outs + admin flex-pay import
- Accumulated via `IncrementMaxAdvAmount` on each check-out

### AdvancePaymentRequest (`backend/internal/domain/advance_payment_request.go`)
- **Statuses**: `pending` → `approved` → `completed` | `failed`; `cancelled` available from both `pending` and `approved`
- Stores `request_amount`, `fee`, `net_amount`

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
| GET | `/api/v1/mobile/attendance/history` | Attendance history |
| GET | `/api/v1/me/advance-payment` | Get advance payment info (quota) |
| POST | `/api/v1/me/advance-payment/request` | Request advance payment |
| POST | `/api/v1/me/advance-payment/calculate-fee` | Preview fee |
| GET | `/api/v1/me/advance-payment/history` | Request history |
| POST | `/api/v1/me/advance-payment/request/:id/cancel` | Cancel pending request |

---

## Key Design Decisions

1. **Quota accumulates in real-time** — Each check-out immediately increments `MaxAdvAmount`, making earnings available for advance requests without admin intervention
2. **Geofence enforcement** — Both check-in and check-out require being within project's configured radius
3. **Shift-based earning** — Earning is determined by matching actual check-in/out times against configured payrate shifts, not by hours worked
4. **Shift time validation (±1 hour)** — Check-in and check-out times must fall within ±1 hour of the configured shift start/end times. Attendance records outside these windows are rejected entirely — no data is persisted. This prevents employees from checking in too early/late or checking out too early/late relative to their assigned shift.
5. **Atomic budget check** — `CreateWithBudgetCheck` prevents race conditions when multiple requests come in simultaneously
6. **Three-phase window** — Locks requests during days 11-20 to allow admin time to process bank transfers
7. **Night shift support** — Check-out can match against yesterday's check-in; shifts crossing midnight handled; ±1h window accounts for midnight boundary

---

## Known Limitations

1. **One check-in per day is global** — The unique constraint is on `(employee_id, date)` without `project_id`, so an employee can only check in once per calendar day across all projects
2. **`orphaned` status is derived at read-time** — Not stored in DB; `GetStatus(now)` computes it dynamically by checking `now - CheckInTime > 18h`. The `GetOrphanCandidates` repository method exists but has no callers (no cleanup cron)
3. **No GPS spoofing prevention** — The system trusts client-reported lat/lng without accuracy checks, altitude verification, or anti-fraud measures
4. **No audit logging for attendance** — Unlike advance payment imports (which use `AuditService`), attendance check-in/out operations have no structured audit trail
5. **No earning correction mechanism** — `AdvancePaymentRepository` has `IncrementMaxAdvAmount` and `ZeroOutQuota` but no `DecrementMaxAdvAmount`. If an attendance earning is incorrect, there is no admin endpoint to correct it; the advance payment quota remains inflated
6. **Timezone is hardcoded** — All business time uses `Asia/Ho_Chi_Minh` via `clock.Now()`. The "today" boundary for check-in/out is always Vietnam time regardless of employee location
