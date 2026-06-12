# 05 — Timesheet Operations

**Priority**: P0 | **API Prefix**: `/api/v1/timesheets` | **Risk Level**: Critical

## Business Rules

- Timesheets track hours worked per employee per day per hour type
- Hour types: HC (ca ngày), OT150 (tăng ca 150%), OT200, OT300, NIGHT (đêm), HOLIDAY (lễ)
- Position determines applicable rates from project payrate
- Three-level payrate: position → day_type → shift_type → amount
- Day types: "ngày thường" (weekday), "cuối tuần" (weekend), "ngày lễ" (holiday)
- Maximum 12 hours/day triggers warning (orange), >16 hours triggers error (red)
- Bulk approve available for admin
- Paid timesheets cannot be deleted (API blocks)
- Re-uploading BCC uses upsert logic (latest wins for unapproved)
- Assignment start_date respected — cannot enter timesheet before assignment start
- `pageSize=200` recommended for test polling to avoid pagination issues

## State Machine

```
Timesheet: [draft] → pending_approval → approved → paid
                                    ↘ rejected
                                        ↗ (re-submit)
paid timesheets: IMMUTABLE (cannot delete/edit)
```

## Test Scenarios

### F07 — Timesheet Entry

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F07-01 | Create single timesheet entry | Happy | POST `/timesheets` with employee_id, project_id, date, hours, hour_type | 201, entry created |
| F07-02 | Create timesheet for past date | Happy | POST with date = yesterday | 201, accepted |
| F07-03 | Create timesheet for future date | Negative | POST with date = tomorrow | 400, future dates not allowed |
| F07-04 | Create timesheet before assignment start | Negative | POST with date before employee assignment start_date | 400, "thời gian phân công nhân viên bắt đầu sau ngày chấm công" |
| F07-05 | Bulk create timesheet entries | Happy | POST `/timesheets/bulk` with multiple entries | 201, all entries created |
| F07-06 | Get timesheet summary | Happy | GET `/timesheets/summary?project_id=X` | 200, aggregated hours and amounts |
| F07-07 | Get grouped timesheets | Happy | GET `/timesheets?group_by=employee` | 200, grouped by employee with totals |
| F07-08 | Delete unapproved timesheet | Happy | DELETE `/timesheets/:id` | 200, entry deleted |
| F07-09 | Delete paid timesheet | Negative | DELETE `/timesheets/:id` (paid) | 400/403, "cannot delete paid timesheet" |
| F07-10 | Timesheet with >12 hours | Edge | Create entry with 13 hours | Accepted with warning (exceeded status) |
| F07-11 | Timesheet with >16 hours | Edge | Create entry with 17 hours | Accepted with excessive warning |
| F07-12 | Duplicate timesheet same employee/date/type | Edge | Create two entries with same key | Upsert behavior: second updates first |
| F07-13 | Timesheet date respects assignment start | Integration | findAvailableDateWithMin → create timesheet | Date is >= assignment start_date |
| F07-14 | Timesheet amount calculated from payrate | Integration | Create timesheet → verify amount | Amount = rate × hours from project payrate |
| F07-15 | Timesheet export template | Integration | GET `/timesheets/export-template` | 200, binary Excel template |

### F08 — Timesheet Approval

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F08-01 | Bulk approve timesheets | Happy | POST `/timesheets/bulk-approve` with list of IDs | 200, all entries move to "approved" |
| F08-02 | Reject timesheet | Happy | POST `/timesheets/:id/reject` with reason | 200, entry moves to "rejected" |
| F08-03 | Re-submit rejected timesheet | Happy | POST `/timesheets` with same employee/date | 201, new entry replaces rejected |
| F08-04 | Approve already approved | Negative | Approve already approved timesheet | Idempotent — no error, no double approval |
| F08-05 | Approve paid timesheet | Negative | Approve already paid timesheet | Rejected — paid is immutable |
| F08-06 | Partner cannot approve | Negative | Partner user attempts bulk-approve | 403 Forbidden |
| F08-07 | Approved timesheet appears in bulk transfer | Integration | Approve → initiate bulk transfer | Approved timesheets included in transfer |
| F08-08 | Approval updates project financials | Integration | Approve timesheets → GET project summary | pending_payable_vnd increases |
| F08-09 | Re-upload after approval | Negative | Upload BCC after timesheets approved | Status=failed, original approved entries preserved |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_timesheet_extended.go`
- Unit test: `backend/internal/domain/timesheet_test.go`
- Key helper: `findAvailableDateWithMin()` for date selection in tests
