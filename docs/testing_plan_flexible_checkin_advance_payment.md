# Testing Plan: Flexible Check-in → Check-out → Advance Payment Flow

## Overview

This document defines integration test scenarios for the flexible employee attendance and advance payment workflow. Tests are implemented as a new `flow_attendance.go` file in `backend/tests/integration/`, following the existing test infrastructure patterns.

**Test file**: `backend/tests/integration/flow_attendance.go`
**Test models to add**: Attendance DTOs in `backend/tests/integration/models.go`

---

## Test Infrastructure

### Required Test Data

The test requires:
- **Flexible project** with `IsFlexible = true`, geofence gates configured, payrate with shift definitions
- **Flexible employee** with `check_in_enabled = true`, user account for API auth
- **Non-flexible employee** for negative tests
- **Employee with check_in_enabled = false** for permission denial tests

Discovery should extend `discoverTestData()` to find:
```go
// Add to TestData struct
FlexibleProject      *ProjectResponse
FlexibleEmployee     *EmployeeDetailedResponse
FlexibleEmpToken     string
NonFlexibleProject   *ProjectResponse
DisabledCheckInEmp   *EmployeeDetailedResponse
DisabledCheckInToken string
```

### Required DTOs (add to models.go)

```go
// --- Attendance ---

type CheckInRequest struct {
    Lat       float64 `json:"lat"`
    Lng       float64 `json:"lng"`
    ProjectID uint    `json:"project_id,omitempty"`
}

type CheckOutRequest struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
}

type AttendanceResponse struct {
    ID            uint     `json:"id"`
    ProjectID     uint     `json:"project_id"`
    EmployeeID    uint     `json:"employee_id"`
    Date          string   `json:"date"`
    CheckInTime   string   `json:"check_in_time"`
    CheckInGate   string   `json:"check_in_gate"`
    CheckOutTime  *string  `json:"check_out_time"`
    CheckOutGate  *string  `json:"check_out_gate"`
    EarningAmount *int64   `json:"earning_amount"`
    Status        string   `json:"status"`
}
```

### Clock Manipulation Pattern

Attendance tests require precise time control. Use the existing clock helpers:

```go
// Set server to a specific date/time for deterministic testing
adminClient := client.WithToken(data.AdminToken)
defer func() { _ = ResetServerTime(adminClient) }()

// Set to a work day at 07:50 (before shift start)
err := SetServerTime(adminClient, time.Date(2026, 6, 8, 7, 50, 0, 0, vnLoc))
```

### Database Cleanup

After each test scenario, clean up attendance records to avoid data pollution:

```go
func cleanupAttendance(client *APIClient, projectID, employeeID uint) {
    // Direct DB cleanup since no delete API exists for attendance
    _ = execCommand("docker", "exec", "payroll-mysql", "mysql", "-uroot", "-prootpassword",
        "payroll_db", "-e", fmt.Sprintf(
            "DELETE FROM attendances WHERE project_id=%d AND employee_id=%d",
            projectID, employeeID))
}
```

---

## Test Scenarios

### Group 1: Check-In Tests

#### 1.1 Check-in Success (Happy Path)

**Setup**: Server time set to workday at 07:50 (before typical 08:00 shift)

```
POST /api/v1/me/advance-payment/today → nil (no existing attendance)
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": <gate_lat>, "lng": <gate_lng>, "project_id": <flexible_project_id> }
  Expected: 200 OK
  Assert:
    - response.status = "checked_in"
    - response.project_id = <flexible_project_id>
    - response.check_in_time is set
    - response.check_out_time is null
    - response.earning_amount is null
    - response.check_in_gate matches a configured gate name
```

**Cleanup**: Delete the attendance record

#### 1.2 Check-in Without project_id (Auto-detect)

```
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": <gate_lat>, "lng": <gate_lng> }
  Expected: 200 OK (auto-detects the employee's single flexible project)
  Assert: response.project_id = <flexible_project_id>
```

**Cleanup**: Delete the attendance record

#### 1.3 Double Check-in Rejected

**Setup**: Check in first, then try again

```
POST /api/v1/mobile/attendance/check-in → 200 OK (first check-in)
POST /api/v1/mobile/attendance/check-in → Expect error
  Body: { "lat": <gate_lat>, "lng": <gate_lng> }
  Expected: 400 Bad Request
  Assert: error message contains "đã check-in trong ngày"
```

**Cleanup**: Delete the attendance record

#### 1.4 Check-in Outside Geofence Rejected

```
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": 1.0, "lng": 1.0, "project_id": <flexible_project_id> }
  Expected: 400 Bad Request
  Assert: error message contains "ngoài khu vực chấm công" or "vị trí check-in"
```

#### 1.5 Check-in With check_in_enabled = false

**Setup**: Use employee with disabled check-in, or admin disables check-in first

```
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": <gate_lat>, "lng": <gate_lng>, "project_id": <flexible_project_id> }
  Expected: 400 Bad Request
  Assert: error message contains "chưa được cấp quyền chấm công"
```

#### 1.6 Check-in to Non-Flexible Project Rejected

```
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": <gate_lat>, "lng": <gate_lng>, "project_id": <non_flexible_project_id> }
  Expected: 400 Bad Request
  Assert: error message contains "không hỗ trợ chấm công linh hoạt"
```

#### 1.7 Multiple Flexible Projects Without project_id

**Setup**: Employee belongs to 2+ flexible projects

```
POST /api/v1/mobile/attendance/check-in
  Body: { "lat": <gate_lat>, "lng": <gate_lng> }  // no project_id
  Expected: 400 Bad Request
  Assert: error message contains "nhiều dự án linh hoạt"
```

---

### Group 2: Shift Time Validation Tests (±1 Hour)

These tests verify the ±1 hour shift time window rejection behavior at both check-in and check-out.

#### 2.1 Check-in Within ±1h Window — Accept

```
Shift: 08:00-17:00, tolerance ±1h → valid window [07:00, 09:00]

SetServerTime → 07:01 (59min before shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓

Cleanup, then:
SetServerTime → 08:59 (59min after shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓

Cleanup, then:
SetServerTime → 08:00 (exact shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓
```

#### 2.2 Check-in Outside ±1h Window — Reject

```
Shift: 08:00-17:00, tolerance ±1h → valid window [07:00, 09:00]

SetServerTime → 06:59 (61min before shift start)
POST /api/v1/mobile/attendance/check-in → 400 Bad Request
  Assert: error message contains "không trong giờ check-in"

Cleanup, then:
SetServerTime → 09:01 (61min after shift start)
POST /api/v1/mobile/attendance/check-in → 400 Bad Request
  Assert: error message contains "không trong giờ check-in"
```

#### 2.3 Check-in No Payrate — Reject

```
Setup: Use a date where no payrate is configured for the project

SetServerTime → date with no payrate
POST /api/v1/mobile/attendance/check-in → 400 Bad Request
  Assert: error message contains "mức lương" or "ca làm việc"
```

#### 2.4 Check-in Multiple Shifts — Match Any

```
Payrate has shifts: 08:00-17:00 AND 14:00-22:00

SetServerTime → 13:01 (within ±1h of 14:00 shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓

Cleanup, then:
SetServerTime → 12:59 (61min before 14:00)
POST /api/v1/mobile/attendance/check-in → 400 Bad Request (not within ±1h of either shift)
```

#### 2.5 Night Shift Check-in — Within Window

```
Shift: 22:00-06:00, tolerance ±1h → valid window [21:00, 23:00]

SetServerTime → 21:01 (59min before shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓

Cleanup, then:
SetServerTime → 22:59 (59min after shift start)
POST /api/v1/mobile/attendance/check-in → 200 OK ✓
```

#### 2.6 Night Shift Check-in — Outside Window

```
Shift: 22:00-06:00, tolerance ±1h → valid window [21:00, 23:00]

SetServerTime → 20:59 (61min before shift start)
POST /api/v1/mobile/attendance/check-in → 400 Bad Request

Cleanup, then:
SetServerTime → 23:01 (61min after shift start)
POST /api/v1/mobile/attendance/check-in → 400 Bad Request
```

#### 2.7 Check-out Within ±1h Window — Accept

```
Shift: 08:00-17:00, tolerance ±1h → valid window [16:00, 18:00]

Setup: Check-in at 07:50, advance to 16:01
POST /api/v1/mobile/attendance/check-out → 200 OK ✓

Cleanup, then:
Check-in at 07:50, advance to 17:59
POST /api/v1/mobile/attendance/check-out → 200 OK ✓
```

#### 2.8 Check-out Outside ±1h Window — Reject

```
Shift: 08:00-17:00, tolerance ±1h → valid window [16:00, 18:00]

Setup: Check-in at 07:50, advance to 15:59
POST /api/v1/mobile/attendance/check-out → 400 Bad Request
  Assert: error message contains "không trong giờ check-out"

Cleanup, then:
Check-in at 07:50, advance to 18:01
POST /api/v1/mobile/attendance/check-out → 400 Bad Request
```

#### 2.9 Night Shift Check-out — Window Validation

```
Shift: 22:00-06:00, tolerance ±1h → valid window [05:00, 07:00]

Setup: Check-in at 21:50, advance to 05:01 (59min before shift end)
POST /api/v1/mobile/attendance/check-out → 200 OK ✓

Cleanup, then:
Check-in at 21:50, advance to 06:59 (59min after shift end)
POST /api/v1/mobile/attendance/check-out → 200 OK ✓

Cleanup, then:
Check-in at 21:50, advance to 04:59 (61min before shift end)
POST /api/v1/mobile/attendance/check-out → 400 Bad Request

Cleanup, then:
Check-in at 21:50, advance to 07:01 (61min after shift end)
POST /api/v1/mobile/attendance/check-out → 400 Bad Request
```

#### 2.10 Boundary: ±1h Exactly at the Edge

```
Shift: 08:00-17:00

Check-in at exactly 07:00 (1h before start) → Accept
Check-in at exactly 09:00 (1h after start) → Accept
Check-out at exactly 16:00 (1h before end) → Accept
Check-out at exactly 18:00 (1h after end) → Accept
```

---

### Group 3: Check-Out Tests

#### 3.1 Check-out Success (Happy Path — Full Shift Coverage)

**Setup**: Check in at 07:50, advance clock to 17:10, check out

```
Step 1: SetServerTime → 2026-06-08 07:50:00
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 9h20m (now at 17:10)
Step 4: POST /api/v1/mobile/attendance/check-out
  Body: { "lat": <gate_lat>, "lng": <gate_lng> }
  Expected: 200 OK
  Assert:
    - response.status = "completed"
    - response.check_out_time is set
    - response.earning_amount > 0 (shift 08:00-17:00 fully covered: 07:50 <= 08:00 AND 17:10 >= 17:00)
    - response.earning_amount matches the payrate for the matched shift
```

**Cleanup**: Delete attendance and advance_payment records

#### 3.2 Check-out With Zero Earning (Late Arrival)

**Setup**: Check in at 09:30 (after shift start 08:00), advance to 17:10, check out

```
Step 1: SetServerTime → 2026-06-08 09:30:00
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 7h40m (now at 17:10)
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 200 OK
  Assert:
    - response.status = "completed"
    - response.earning_amount = 0 or null (09:30 > 08:00 — does NOT cover shift start)
    - NO advance_payment record created (earning = 0)
```

**Cleanup**: Delete attendance record

#### 3.3 Check-out With Zero Earning (Early Departure)

**Setup**: Check in at 07:50, advance to 15:00 (before shift end 17:00), check out

```
Step 1: SetServerTime → 2026-06-08 07:50:00
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 7h10m (now at 15:00)
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 200 OK
  Assert:
    - response.status = "completed"
    - response.earning_amount = 0 or null (15:00 < 17:00 — does NOT cover shift end)
    - NO advance_payment record created
```

**Cleanup**: Delete attendance record

#### 3.4 Check-out Without Check-in Rejected

```
POST /api/v1/mobile/attendance/check-out
  Body: { "lat": <gate_lat>, "lng": <gate_lng> }
  Expected: 400 Bad Request
  Assert: error message contains "không tìm thấy thông tin check-in"
```

#### 3.5 Double Check-out Rejected

**Setup**: Check in, check out successfully, then try to check out again

```
Step 1: Check-in → 200 OK
Step 2: Check-out → 200 OK
Step 3: POST /api/v1/mobile/attendance/check-out → Expect error
  Expected: 400 Bad Request
  Assert: error message contains "đã check-out"
```

**Cleanup**: Delete attendance and advance_payment records

#### 3.6 Orphaned Check-out Rejected

**Setup**: Check in, advance time by >18 hours, try to check out

```
Step 1: SetServerTime → 2026-06-08 08:00:00
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 19h (now at 03:00 next day)
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 400 Bad Request
  Assert: error message contains "quá hạn check-out"
```

**Cleanup**: Delete attendance record

#### 3.7 Check-out Outside Geofence Rejected

**Setup**: Check in at valid location, try to check out at invalid location

```
Step 1: Check-in → 200 OK (valid location)
Step 2: AdvanceServerTime → 9h
Step 3: POST /api/v1/mobile/attendance/check-out
  Body: { "lat": 1.0, "lng": 1.0 }
  Expected: 400 Bad Request
  Assert: error message contains "ngoài khu vực"
```

**Cleanup**: Delete attendance record

#### 3.8 Check-out With check_in_enabled Disabled Mid-Session

**Setup**: Check in, admin disables check-in, try to check out

```
Step 1: Check-in → 200 OK
Step 2: Admin PATCH /api/v1/projects/:id/employees/:empId/checkin-enabled
        Body: { "check_in_enabled": false }
Step 3: AdvanceServerTime → 9h
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 400 Bad Request
  Assert: error message contains "vô hiệu hóa"
```

**Cleanup**: Re-enable check_in, delete attendance record

---

### Group 4: Night Shift Tests

#### 4.1 Night Shift — Same Calendar Day Check-out

**Setup**: Check in at 21:50 for 22:00-06:00 shift, advance to 06:10 same "next day"

```
Step 1: SetServerTime → 2026-06-08 21:50:00
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 8h20m (now at 06:10 on June 9)
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 200 OK
  Assert:
    - response.status = "completed"
    - response.earning_amount > 0 (21:50 <= 22:00 AND 06:10 >= 06:00)
```

> **Note**: The check-out will look up yesterday's check-in record when today's is not found (night shift support in CheckOut step 1).

**Cleanup**: Delete attendance and advance_payment records

#### 4.2 Night Shift — Zero Earning (Late Arrival)

```
Step 1: SetServerTime → 2026-06-08 22:30:00 (after shift start 22:00)
Step 2: POST /api/v1/mobile/attendance/check-in → 200 OK
Step 3: AdvanceServerTime → 5h40m (now at 04:10 on June 9)
Step 4: POST /api/v1/mobile/attendance/check-out
  Expected: 200 OK
  Assert: response.earning_amount = 0 (22:30 > 22:00 — late arrival)
```

**Cleanup**: Delete attendance record

---

### Group 5: Advance Payment Quota Tests

#### 5.1 Quota Accumulated After Check-out

**Setup**: Check in, check out with full shift coverage, verify advance payment quota

```
Step 1: Complete check-in + check-out (full shift coverage, earning > 0)
Step 2: GET /api/v1/me/advance-payment
  Expected: 200 OK
  Assert:
    - response.maxAdvanceAmount >= earningAmount from check-out
    - response.remainingAmount > 0
```

**Cleanup**: Delete attendance and advance_payment records

#### 5.2 Quota NOT Accumulated After Zero-Earning Check-out

**Setup**: Check in, check out with zero earning (late arrival), verify no quota change

```
Step 1: Record initial advance payment quota (or note it doesn't exist)
Step 2: Complete check-in + check-out with zero earning
Step 3: GET /api/v1/me/advance-payment
  Assert: maxAdvanceAmount unchanged (no new advance_payment record created)
```

**Cleanup**: Delete attendance record

#### 5.3 Multiple Check-outs Accumulate Quota

**Setup**: Complete 2 check-in/check-out cycles on different days, verify cumulative quota

```
Day 1: SetServerTime → Day 1 07:50, check-in, advance to 17:10, check-out
       Verify quota = shift_rate_1
Day 2: SetServerTime → Day 2 07:50, check-in, advance to 17:10, check-out
       Verify quota = shift_rate_1 + shift_rate_2
```

> **Note**: Must use different calendar days. Clock must be reset between days.

**Cleanup**: Delete all attendance and advance_payment records

---

### Group 6: Advance Payment Request Tests (Attendance-earned Quota)

#### 6.1 Request Advance After Earning

**Setup**: Complete check-in/check-out to earn quota, then request advance

```
Step 1: Complete check-in + check-out (full shift, earning > 0)
Step 2: Record earning_amount and verify quota
Step 3: POST /api/v1/me/advance-payment/request
  Body: { "amount": <less_than_earning> }
  Expected: 200 OK
  Assert:
    - response.status = "pending"
    - response.request_amount = requested amount
    - response.fee > 0
    - response.net_amount = request_amount - fee
```

**Cleanup**: Cancel request, delete attendance and advance_payment records

#### 6.2 Request Exceeding Earned Quota Rejected

**Setup**: Complete check-in/check-out, then request MORE than earned

```
Step 1: Complete check-in + check-out (earning = shift_rate, e.g., 500,000)
Step 2: POST /api/v1/me/advance-payment/request
  Body: { "amount": 600000 }  // > 500,000 earned
  Expected: 400 Bad Request
  Assert: error message contains "vượt hạn mức"
```

**Cleanup**: Delete attendance and advance_payment records

#### 6.3 Request During Locked Gap Rejected

**Setup**: Set time to day 15 (locked gap, days 11-20)

```
Step 1: SetServerTime → day 15 of month
Step 2: POST /api/v1/me/advance-payment/request
  Body: { "amount": 100000 }
  Expected: 400 Bad Request
  Assert: error message contains cutoff or locked period info
```

---

### Group 7: Combined End-to-End Flow

This is the **primary real-life simulation** — a flexible employee's complete daily cycle.

#### 7.1 Full Day: Check-in → Work → Check-out → Request Advance

```
=== SETUP ===
Admin client: SetServerTime → 2026-06-08 07:45:00 (Monday, workday)
Employee client: authenticated as flexible employee

=== PHASE 1: CHECK-IN ===
1. GET /api/v1/mobile/attendance/today
   Assert: response.data = null (no attendance yet)

2. POST /api/v1/mobile/attendance/check-in
   Body: { "lat": <gate_lat>, "lng": <gate_lng> }
   Assert: 200 OK, status = "checked_in"

3. GET /api/v1/mobile/attendance/today
   Assert: response.data.status = "checked_in", check_out_time = null

=== PHASE 2: WORK (SIMULATE TIME PASSING) ===
4. Admin: AdvanceServerTime → 9h30m (now 17:15)

=== PHASE 3: CHECK-OUT ===
5. POST /api/v1/mobile/attendance/check-out
   Body: { "lat": <gate_lat>, "lng": <gate_lng> }
   Assert: 200 OK, status = "completed"
   Assert: earning_amount > 0 (full shift 08:00-17:00 covered)
   Record: earning_amount = EARNED

6. GET /api/v1/mobile/attendance/today
   Assert: response.data.status = "completed"
   Assert: response.data.earning_amount = EARNED

=== PHASE 4: VERIFY QUOTA ===
7. GET /api/v1/me/advance-payment
   Assert: maxAdvanceAmount >= EARNED
   Assert: remainingAmount >= EARNED

=== PHASE 5: REQUEST ADVANCE PAYMENT ===
8. POST /api/v1/me/advance-payment/calculate-fee
   Body: { "amount": <half_of_EARNED> }
   Record: fee, netAmount

9. POST /api/v1/me/advance-payment/request
   Body: { "amount": <half_of_EARNED> }
   Assert: 200 OK, status = "pending"
   Record: request_id

=== PHASE 6: VERIFY BUDGET AFTER REQUEST ===
10. GET /api/v1/me/advance-payment
    Assert: pendingAmount = <half_of_EARNED>
    Assert: remainingAmount = EARNED - <half_of_EARNED>

=== PHASE 7: CANCEL AND VERIFY RESTORED BUDGET ===
11. POST /api/v1/me/advance-payment/request/{request_id}/cancel
    Assert: 200 OK

12. GET /api/v1/me/advance-payment
    Assert: pendingAmount = 0
    Assert: remainingAmount = EARNED (restored)

=== CLEANUP ===
13. Admin: ResetServerTime
14. DB cleanup: attendance + advance_payment + advance_payment_requests
```

#### 7.2 Two-Day Earning Accumulation

```
=== DAY 1 ===
1. SetServerTime → 2026-06-08 07:50
2. Check-in → check-out (full shift) → record EARNED_1
3. Verify quota = EARNED_1

=== DAY 2 ===
4. SetServerTime → 2026-06-09 07:50
5. Check-in → check-out (full shift) → record EARNED_2
6. Verify quota = EARNED_1 + EARNED_2

=== REQUEST ===
7. Request advance for (EARNED_1 + EARNED_2 - 1) → should succeed
8. Request advance for 1 more → should fail (over budget)

=== CLEANUP ===
9. Cancel all requests, reset time, DB cleanup
```

---

### Group 8: Edge Cases

#### 8.1 Check-in at Exactly Shift Start

```
SetServerTime → 08:00:00 (exactly shift start)
Check-in → 200 OK
AdvanceServerTime → 9h
Check-out at 17:00 → earning > 0 (08:00 <= 08:00 ✓, 17:00 >= 17:00 ✓)
```

#### 8.2 Check-out at Exactly Shift End

```
Check-in at 07:50
AdvanceServerTime → shift_length (e.g., 9h10m to reach 17:00)
Check-out at 17:00 → earning > 0 (07:50 <= 08:00 ✓, 17:00 >= 17:00 ✓)
```

#### 8.3 Check-in 1 Second Before Shift Start

```
SetServerTime → 07:59:59
Check-in → 200 OK
Advance to 17:00
Check-out → earning > 0 (07:59:59 <= 08:00 ✓)
```

#### 8.4 Check-in 1 Second After Shift Start

```
SetServerTime → 08:00:01
Check-in → 200 OK (check-in is NOT rejected)
Advance to 17:01
Check-out → earning = 0 (08:00:01 > 08:00, does NOT cover shift start)
```

#### 8.5 Check-out 1 Second Before Shift End

```
Check-in at 07:50
Advance to 16:59:59
Check-out → 200 OK (check-out is NOT rejected)
Assert: earning = 0 (16:59:59 < 17:00, does NOT cover shift end)
```

#### 8.6 Check-out 1 Second After Shift End

```
Check-in at 07:50
Advance to 17:00:01
Check-out → earning > 0 (07:50 <= 08:00 ✓, 17:00:01 >= 17:00 ✓)
```

#### 8.7 Attendance History

```
Complete 3 check-in/check-out cycles on different days
GET /api/v1/mobile/attendance/history?limit=20
Assert: response contains 3+ records, ordered by date desc
```

#### 8.8 Admin Views Attendance

```
Admin: GET /api/v1/admin/attendances?employee_id=<id>&pageSize=20
Assert: contains the attendance records created in tests
```

#### 8.9 Advance Payment Request Below Minimum

```
POST /api/v1/me/advance-payment/request
Body: { "amount": 5000 }
Expected: 400 Bad Request (minimum is 10,000 VND)
```

---

## Implementation Skeleton

```go
package main

import (
    "fmt"
    "time"
)

const flowAttendance = "Attendance"

// vnLoc is the Vietnam timezone for clock manipulation
var vnLoc = time.FixedZone("ICT", 7*3600)

func runAttendanceTests(client *APIClient, data *TestData, reporter *Reporter) {
    reporter.PrintSection("FLOW: Attendance + Advance Payment")

    adminClient := client.WithToken(data.AdminToken)

    // Skip if no flexible employee with user account
    if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
        reporter.Skip(flowAttendance, "All tests", "no flexible employee user account available")
        return
    }

    empClient := client.WithToken(data.EmployeeTokenForAdv)
    projectID := data.FlexibleProject.ID
    empID := data.EmployeeForAdvance.ID

    // Gate coordinates from project geofence configuration
    gateLat, gateLng := getGateCoords(data.FlexibleProject)

    // Always reset clock after tests
    defer func() {
        _ = ResetServerTime(adminClient)
        cleanupAttendanceDB(projectID, empID)
    }()

    // --- Group 1: Check-in Tests ---
    reporter.RunTest(flowAttendance, "Check-in success", func() error { /* 1.1 */ })
    reporter.RunTest(flowAttendance, "Check-in auto-detect project", func() error { /* 1.2 */ })
    reporter.RunTest(flowAttendance, "Double check-in rejected", func() error { /* 1.3 */ })
    reporter.RunTest(flowAttendance, "Check-in outside geofence rejected", func() error { /* 1.4 */ })
    reporter.RunTest(flowAttendance, "Check-in with disabled check_in_enabled", func() error { /* 1.5 */ })

    // --- Group 2: Shift Time Validation Tests (±1h) ---
    reporter.RunTest(flowAttendance, "Check-in within ±1h window", func() error { /* 2.1 */ })
    reporter.RunTest(flowAttendance, "Check-in outside ±1h window rejected", func() error { /* 2.2 */ })
    reporter.RunTest(flowAttendance, "Check-in no payrate rejected", func() error { /* 2.3 */ })
    reporter.RunTest(flowAttendance, "Check-in multiple shifts match any", func() error { /* 2.4 */ })
    reporter.RunTest(flowAttendance, "Night shift check-in window", func() error { /* 2.5 */ })
    reporter.RunTest(flowAttendance, "Night shift check-in outside window", func() error { /* 2.6 */ })
    reporter.RunTest(flowAttendance, "Check-out within ±1h window", func() error { /* 2.7 */ })
    reporter.RunTest(flowAttendance, "Check-out outside ±1h window rejected", func() error { /* 2.8 */ })
    reporter.RunTest(flowAttendance, "Night shift check-out window", func() error { /* 2.9 */ })
    reporter.RunTest(flowAttendance, "Boundary ±1h exact edge", func() error { /* 2.10 */ })

    // --- Group 3: Check-out Tests ---
    reporter.RunTest(flowAttendance, "Check-out with full shift earning", func() error { /* 3.1 */ })
    reporter.RunTest(flowAttendance, "Check-out zero earning (late arrival)", func() error { /* 3.2 */ })
    reporter.RunTest(flowAttendance, "Check-out zero earning (early departure)", func() error { /* 3.3 */ })
    reporter.RunTest(flowAttendance, "Check-out without check-in rejected", func() error { /* 3.4 */ })
    reporter.RunTest(flowAttendance, "Double check-out rejected", func() error { /* 3.5 */ })
    reporter.RunTest(flowAttendance, "Orphaned check-out rejected (>18h)", func() error { /* 3.6 */ })
    reporter.RunTest(flowAttendance, "Check-in disabled mid-session", func() error { /* 3.8 */ })

    // --- Group 4: Night Shift Tests ---
    reporter.RunTest(flowAttendance, "Night shift full coverage earning", func() error { /* 4.1 */ })
    reporter.RunTest(flowAttendance, "Night shift zero earning (late)", func() error { /* 4.2 */ })

    // --- Group 5: Quota Tests ---
    reporter.RunTest(flowAttendance, "Quota accumulated after earning", func() error { /* 5.1 */ })
    reporter.RunTest(flowAttendance, "Quota NOT accumulated after zero earning", func() error { /* 5.2 */ })
    reporter.RunTest(flowAttendance, "Multiple check-outs accumulate quota", func() error { /* 5.3 */ })

    // --- Group 6: Advance Request Tests ---
    reporter.RunTest(flowAttendance, "Request advance after earning", func() error { /* 6.1 */ })
    reporter.RunTest(flowAttendance, "Request exceeding quota rejected", func() error { /* 6.2 */ })

    // --- Group 7: End-to-End ---
    reporter.RunTest(flowAttendance, "E2E: full day cycle", func() error { /* 7.1 */ })
    reporter.RunTest(flowAttendance, "E2E: two-day accumulation", func() error { /* 7.2 */ })

    // --- Group 8: Edge Cases ---
    reporter.RunTest(flowAttendance, "Boundary: exact shift start/end", func() error { /* 8.1-8.6 */ })
    reporter.RunTest(flowAttendance, "Attendance history", func() error { /* 8.7 */ })
    reporter.RunTest(flowAttendance, "Admin attendance view", func() error { /* 8.8 */ })
}

func cleanupAttendanceDB(projectID, employeeID uint) {
    // Delete attendance, advance_payment_requests, then advance_payments
    // Order matters due to foreign key constraints
    _ = execCommand("docker", "exec", "payroll-mysql", "mysql", "-uroot", "-prootpassword",
        "payroll_db", "-e", fmt.Sprintf(
            "DELETE apr FROM advance_payment_requests apr "+
                "JOIN advance_payments ap ON apr.adv_pay_id = ap.id "+
                "WHERE ap.project_id=%d AND ap.employee_id=%d AND ap.for_month = DATE_FORMAT(NOW(), '%%Y-%%m')",
            projectID, employeeID))
    _ = execCommand("docker", "exec", "payroll-mysql", "mysql", "-uroot", "-prootpassword",
        "payroll_db", "-e", fmt.Sprintf(
            "DELETE FROM advance_payments WHERE project_id=%d AND employee_id=%d AND for_month = DATE_FORMAT(NOW(), '%%Y-%%m')",
            projectID, employeeID))
    _ = execCommand("docker", "exec", "payroll-mysql", "mysql", "-uroot", "-prootpassword",
        "payroll_db", "-e", fmt.Sprintf(
            "DELETE FROM attendances WHERE project_id=%d AND employee_id=%d",
            projectID, employeeID))
}

func getGateCoords(project *ProjectResponse) (lat, lng float64) {
    // Return coordinates from project's geofence gates
    // Fallback to configured test coordinates
    return 10.12345, 106.12345 // TODO: read from project or config
}
```

---

## Test Registration in main.go

Add after the advance payment flow phase (around line 60 in `main.go`):

```go
// Phase X: Attendance + Advance Payment (check-in → check-out → request)
runAttendanceTests(client, &data, reporter)
```

**Phase ordering**: Attendance tests should run BEFORE `runAdvancePaymentTests` because:
1. They test the attendance earning mechanism that feeds the advance payment quota
2. They clean up after themselves, not interfering with existing advance payment tests
3. They need clock manipulation which may affect other tests if ordered incorrectly

---

## Summary

| Group | Scenarios | Tests | Key API Endpoints |
|-------|-----------|-------|-------------------|
| 1. Check-in | Success, auto-detect, double, geofence, disabled | 7 | `POST /attendance/check-in` |
| 2. Shift validation (±1h) | Check-in window accept/reject, check-out window accept/reject, night shift, no payrate, boundary | 10 | `POST /attendance/check-in`, `POST /attendance/check-out` |
| 3. Check-out | Full earning, zero earning, no check-in, double, orphaned, geofence, disabled | 8 | `POST /attendance/check-out` |
| 4. Night shift | Full coverage, zero earning | 2 | `POST /attendance/check-out` |
| 5. Quota | Accumulated, not accumulated, multi-day | 3 | `GET /advance-payment` |
| 6. Advance request | After earning, over quota, locked gap | 3 | `POST /advance-payment/request` |
| 7. End-to-end | Full day cycle, two-day accumulation | 2 | All attendance + advance endpoints |
| 8. Edge cases | Boundary times, history, admin view | 3+ | All attendance endpoints |
| **Total** | | **~38** | |

All tests use **clock manipulation** (`SetServerTime`/`AdvanceServerTime`/`ResetServerTime`) for deterministic time-based validation, and **direct DB cleanup** for test isolation.
