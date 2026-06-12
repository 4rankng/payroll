# 06 — BCC Import (Partner Timesheet Upload)

**Priority**: P0 | **API Prefix**: `/api/v1/timesheets/partner-import` | **Risk Level**: High

## Business Rules

- Partner uploads BCC Excel files to create timesheets for their project
- `project_id` and `for_month` are required parameters
- Auto-creates employees from BCC sheets (STK rows with unknown CCCD)
- Latest upload wins — re-upload overwrites unapproved entries (stale cleanup)
- Two-layer duplicate protection: (1) stale cleanup deletes unapproved matching employee+date+hourType, (2) BulkCreateTimesheets does paytype-keyed upsert
- Re-upload after bulk-approve is rejected (status=failed)
- Paid/approved timesheets are never modified by re-upload
- Two BCC formats: Standard (multi-position) and Weekly (BCC-<shiftType> sheets)

## Standard BCC Format

- Multi-position sheets with employee rows
- Payrate lookup: position → day_type → shift_type → amount
- Employee matching by CCCD with STK name cross-check

## Weekly BCC Format (BCC LGD)

- Sheet naming: `BCC-<shiftType>` (e.g., `BCC-HC`, `BCC-OT150`)
- Also supports `EPE<MM.YYYY>.xlsx` (STK format)
- Shift type extracted case-insensitively from sheet name (BCC- prefix)
- Date handling: day-of-month from Excel cell (not strict year/month matching)
- Day type: weekday vs weekend only (no holiday detection)
- Rate lookup via dot-path: `position.dayType.shiftType`

## State Machine

```
Upload: [uploaded] → processing → completed
                                ↘ failed
Re-upload: [uploaded] → completed (overwrites unapproved)
           [uploaded] → failed (if timesheets already approved)
```

## Test Scenarios

### F09 — Standard BCC Import

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F09-01 | Upload valid BCC file | Happy | POST `/partner-import` with project_id, for_month, BCC Excel | 200, import job created, timesheets generated |
| F09-02 | Get import detail | Happy | GET `/partner-import/:id` | 200, import details with file metadata |
| F09-03 | Download original file | Happy | GET `/partner-import/:id/download` | 200, binary Excel file (original upload) |
| F09-04 | List upload history (partner view) | Happy | GET `/partner-import?project_id=X` (partner token) | 200, only partner's project uploads |
| F09-05 | List upload history (admin view) | Happy | GET `/partner-import` (admin token) | 200, all project uploads visible |
| F09-06 | Upload without project_id | Negative | POST without project_id | 400, project_id required |
| F09-07 | Upload without for_month | Negative | POST without for_month | 400, for_month required |
| F09-08 | Upload with invalid file format | Negative | POST with non-Excel file | 400, parsing error |
| F09-09 | Re-upload overwrites unapproved | Edge | Upload → approve none → re-upload | Unapproved entries overwritten, no duplicates |
| F09-10 | Re-upload after approval fails | Edge | Upload → bulk approve → re-upload | Status=failed, approved entries preserved |
| F09-11 | Auto-create employee from BCC | Integration | Upload BCC with unknown CCCD | New employee created, timesheets linked |
| F09-12 | BCC import triggers audit event | Integration | Upload BCC → check audit logs | Import event logged with file details |

### F10 — Weekly BCC Import

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F10-01 | Upload BCC-HC sheet | Happy | POST with BCC LGD.xlsx containing BCC-HC sheet | 200, HC timesheets created |
| F10-02 | Upload BCC-OT150 sheet | Happy | POST with BCC LGD.xlsx containing BCC-OT150 sheet | 200, OT150 timesheets created |
| F10-03 | Upload multi-sheet BCC (HC + OT) | Happy | POST with file containing BCC-HC and BCC-OT150 | 200, both shift types processed |
| F10-04 | Upload EPE06.2026.xlsx (STK format) | Happy | POST with STK format file | 200, timesheets created from STK rows |
| F10-05 | Weekly BCC date handling | Edge | Upload with dates spanning month boundary | day-of-month used correctly, no year/month strict match |
| F10-06 | Case-insensitive sheet name | Edge | Upload with BCC-hc (lowercase) | Sheet matched case-insensitively |
| F10-07 | Empty BCC sheet | Edge | Upload with BCC-HC sheet containing no data | Completed with 0 timesheets (no error) |
| F10-08 | Re-upload weekly BCC | Edge | Upload same file twice | Second upload overwrites unapproved entries |
| F10-09 | Weekly BCC rate lookup | Integration | Upload → verify timesheet amounts | Rate = payrate position.weekday/weekend.HC |
| F10-10 | Employee matching via CCCD | Integration | Upload with known CCCD but different name format | Matched by CCCD, name cross-check logged |
| F10-11 | Day type: weekday vs weekend | Integration | Upload with weekday + weekend dates | Weekday rates vs weekend rates applied correctly |
| F10-12 | Paid timesheets not affected by re-upload | Integration | Upload → approve → pay → re-upload | Paid entries untouched, new entries created for unpaid |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_bcc_import.go`
- Integration test: `backend/tests/integration/flow_bcc_weekly_import.go`
- Unit test: `backend/internal/app/services/bcc_import_service_test.go`
- Unit test: `backend/internal/app/services/bcc_import_weekly_test.go`
- Key constants: `MAX_HOURS_PER_DAY = 12`, `MAX_HOURS_EXCESSIVE = 16`
