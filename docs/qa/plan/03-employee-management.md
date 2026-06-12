# 03 — Employee Management

**Priority**: P1 | **API Prefix**: `/api/v1/employees` | **Risk Level**: Medium

## Business Rules

- CCCD (Citizen ID) is unique identifier (12 digits)
- Employee can be assigned to multiple projects
- Bank details: bank_id (FK to banks table), bank_account_number, bank_account_name
- Soft-delete with GORM DeletedAt
- Employee self-service available when linked to a User via user_id

## Test Scenarios

### F03 — Employee CRUD

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F03-01 | Create employee with all fields | Happy | POST `/employees` with fullname, CCCD, address, mobile | 201, employee created with all fields |
| F03-02 | Get employee by ID | Happy | GET `/employees/:id` | 200, returns full employee details |
| F03-03 | Get employee by CCCD | Happy | GET `/employees?cccd=xxx` | 200, returns matching employee |
| F03-04 | List all employees | Happy | GET `/employees` | 200, paginated employee list |
| F03-05 | Get employee summary | Happy | GET `/employees/summary` | 200, aggregated summary (total, by status, etc.) |
| F03-06 | Update employee bank details | Happy | PUT `/employees/:id` with bank_id, account_number, account_name | 200, bank details updated |
| F03-07 | Create employee without CCCD | Negative | POST `/employees` without CCCD field | 400, "CCCD là bắt buộc" |
| F03-08 | Create employee with duplicate CCCD | Negative | POST `/employees` with existing CCCD | 400, duplicate CCCD rejected |
| F03-09 | Update bank details without bank_id | Negative | PUT `/employees/:id` with account_number but no bank_id | 400, bank_id required when bank details provided |
| F03-10 | Delete employee removes from listings | Edge | DELETE `/employees/:id` → GET `/employees` | Employee no longer in list |
| F03-11 | Export employees | Edge | GET `/employees/export` | 200, binary Excel file with all employees |
| F03-12 | Employee appears in project after assignment | Integration | Assign employee to project → GET `/projects/:id/employees` | Employee listed with assignment details |
| F03-13 | Employee current projects | Integration | Assign employee → GET `/employees/:id/current-projects` | Returns project list with positions, start dates |

### F30 — Employee Self-Service

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F30-01 | Get my profile | Happy | GET `/employee/me/profile` with employee-linked user token | 200, returns own employee profile |
| F30-02 | Get my timesheets | Happy | GET `/employee/me/timesheets` | 200, returns only own timesheets |
| F30-03 | Get my summary | Happy | GET `/employee/me/summary` | 200, own work summary |
| F30-04 | Get my payroll history | Happy | GET `/employee/me/payroll` | 200, own payroll records |
| F30-05 | Get my advance payment info | Happy | GET `/employee/me/advance-payment` | 200, own advance payment status/eligibility |
| F30-06 | Cannot access other employee data | Negative | Try to query another employee's timesheets | 403 or filtered to own data only |
| F30-07 | Self-service without employee link | Edge | User without employee_id calls self-service | 404 or appropriate error |
| F30-08 | Timesheets span multiple projects | Integration | Employee on 2 projects → get my timesheets | Returns timesheets from both projects |
| F30-09 | Advance payment info reflects FlexPay import | Integration | Upload bang luong → employee checks advance info | Max advance amount reflects imported data |

### F03-Auto — BCC Auto-Employee Creation

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| AE-01 | Auto-create employee from STK sheet | Happy | Upload BCC with unknown CCCD | New employee created, timesheets linked |
| AE-02 | Auto-create matches existing CCCD | Edge | Upload BCC with known CCCD | No duplicate, existing employee used |
| AE-03 | Auto-create employee name normalization | Edge | BCC has different name format for same CCCD | Name cross-check logged, existing employee matched by CCCD |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_employee_crud.go`
- Integration test: `backend/tests/integration/flow_employee_self_service.go`
