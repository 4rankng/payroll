# 04 — Project Management

**Priority**: P1 | **API Prefix**: `/api/v1/projects` | **Risk Level**: Medium-High

## Business Rules

- Project code is unique identifier
- Salary period: `salary_period_from` (day of previous month) and `salary_period_to` (day of current month)
- Off days stored as bitmask (bit0=Sun, bit1=Mon, ..., bit6=Sat)
- `is_flexible` flag enables flexible check-in schedules
- Geofence gates and radius for location-based attendance
- Salary period updates are admin-only (partners get 403)
- Assignment `start_date` cannot be moved backward if approved/paid timesheets exist after new date
- Project status: draft → active → paused → completed → cancelled

## State Machine

```
Project: [draft] → active → paused → completed
                       ↓           ↓
                    cancelled   cancelled

Assignment: [assigned] → removed (with last_date)
                   ↑         ↓
                   └─ re-assigned
```

## Test Scenarios

### F04 — Project CRUD

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F04-01 | Create project with salary period | Happy | POST `/projects` with clientName, name, code, salaryPeriodFrom=26, salaryPeriodTo=25 | 201, project created with salary period |
| F04-02 | Get project by ID | Happy | GET `/projects/:id` | 200, project details with salary period |
| F04-03 | List projects | Happy | GET `/projects` | 200, paginated project list |
| F04-04 | Get project summary | Happy | GET `/projects/summary` | 200, aggregated project stats |
| F04-05 | Update project name | Happy | PUT `/projects/:id` with new name | 200, name updated |
| F04-06 | Create project without required fields | Negative | POST `/projects` without clientName | 400, validation error |
| F04-07 | Create project with duplicate code | Negative | POST `/projects` with existing code | 400, code must be unique |
| F04-08 | Invalid payment schedule | Negative | POST `/projects/:id/employees` with schedule="biweekly" | 400, must be weekly or monthly |
| F04-09 | Delete project | Happy | DELETE `/projects/:id` | 200, soft-deleted |
| F04-10 | Partner cannot update salary period | Edge | Partner user PUT salary period | 403 Forbidden |
| F04-11 | Salary period change with unsettled timesheets | Edge | Update salary period when unsettled timesheets exist | Blocked with error message |
| F04-12 | Project appears in dashboard after creation | Integration | Create project → GET `/dashboard/summary` | Project counted in summary stats |

### F05 — Employee Assignment

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F05-01 | Assign employee to project | Happy | POST `/projects/:id/employees` with employeeID, position, payment_schedule, start_date | 201, assignment created |
| F05-02 | List project employees | Happy | GET `/projects/:id/employees` | 200, employees with positions and schedules |
| F05-03 | Update assignment position | Happy | PUT `/projects/:id/employees/:empId` with new position | 200, position updated |
| F05-04 | Remove employee from project | Happy | DELETE `/projects/:id/employees/:empId` | 200, assignment removed with last_date |
| F05-05 | Assignment start_date validation | Negative | Assign with start_date in invalid format | 400, validation error |
| F05-06 | End_date before start_date | Negative | Update assignment with end_date < start_date | 400, "end_date must be >= start_date" |
| F05-07 | Move start_date backward with paid timesheets | Negative | Move start_date back when paid timesheets after new date | Blocked: "HasNonEditableTimesheetsAfterDate" |
| F05-08 | Re-assign removed employee | Edge | Remove → re-assign same employee | 201, new assignment with new start_date |
| F05-09 | Cache invalidation on assignment update | Integration | Update assignment → validate assignment cache refreshed | Cache key `assignment:{projectID}:{employeeID}` invalidated |
| F05-10 | Assignment visible in employee self-service | Integration | Assign → GET `/employee/me/current-projects` | Project appears in employee's project list |
| F05-11 | Clear end_date via empty string | Edge | Update assignment with end_date="" | 200, end_date cleared (null) |
| F05-12 | Employee with multiple projects | Integration | Assign employee to 2 projects → list both | Employee appears in both project employee lists |

### F06 — Payrate CRUD

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F06-01 | Create payrate with nested rates | Happy | POST `/projects/:id/payrates` with JSON rates `{position: {dayType: {shiftType: amount}}}` | 201, payrate created |
| F06-02 | List project payrates | Happy | GET `/projects/:id/payrates` | 200, payrate list with effective dates |
| F06-03 | Get payrate by ID | Happy | GET `/payrates/:id` | 200, payrate details |
| F06-04 | Update payrate | Happy | PUT `/payrates/:id` with new rates JSON | 200, rates updated |
| F06-05 | Create payrate without effective_from | Negative | POST without effective_from | 400, effective_from required |
| F06-06 | Delete payrate | Happy | DELETE `/payrates/:id` | 200, soft-deleted |
| F06-07 | Payrate used in timesheet calculation | Integration | Create payrate → create timesheet → verify amount | Timesheet amount matches rate × hours |
| F06-08 | Multiple payrates with different effective dates | Integration | Create 2 payrates → timesheet uses correct one | Rate effective on timesheet date is used |
| F06-09 | Update non-existent payrate | Negative | PUT `/payrates/99999` | 404 or appropriate error |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_project_crud.go`
- Integration test: `backend/tests/integration/flow_payrate_crud.go`
