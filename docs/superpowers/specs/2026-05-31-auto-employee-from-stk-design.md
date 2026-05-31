# Auto-Create Employees from STK Sheet During BCC Import

**Date:** 2026-05-31
**Status:** Approved

## Context

When partners upload a BCC (bang cham cong) Excel file, the system currently requires all employees to already exist in the project. Employees not found are reported as errors ("nhan vien khong tim thay trong he thong") and skipped. This forces manual employee creation before every import.

The partner Excel files contain a second sheet "STK" with employee bank account information. This sheet has enough data (CCCD, full name, bank account, bank name) to auto-create employee profiles, user accounts, and project assignments before processing timesheet entries.

## Design

### Flow

Integrated into the existing `BCCImportService.ProcessUpload()` — no new endpoints.

```
Upload .xlsx
  → Parse BCC sheet (existing)
  → Parse STK sheet (NEW)
  → ensureEmployeesFromSTK():
       For each STK row where CCCD not in system:
         → Create employee profile
         → Create user account (auto-generated username + default password)
         → Assign to project (weekly, "pho thong", start=today)
       For each STK row where employee exists but lacks bank info:
         → Fill bank details from STK
  → Refresh byCCCD/byCode lookup maps
  → Existing BCC matching + timesheet creation (unchanged)
```

### STK Sheet Layout (confirmed)

Columns: B=CCCD, C=FullName, D=BankAccount, E=BankName, F=Note. First 3 rows are headers. Parser already exists at `stk_parser.go`.

### Employee Creation

Reuses the pattern from `ImportService.getOrCreateEmployee()`:

| Field | Source |
|-------|--------|
| `Fullname` | STK `FullName` |
| `CCCD` | STK `CCCD` |
| `BankAccountNumber` | STK `BankAccount` |
| `BankAccountName` | STK `FullName` |
| `BankID` | Resolved via `bankpkg.MapName(STK.BankName)` |
| `CreatedBy` | Uploader user ID |
| `UserID` | Auto-created via `EmployeeUserService.CreateUserForEmployee()` |

### Assignment Defaults

| Field | Value |
|-------|-------|
| `Position` | `"pho thong"` |
| `PaymentSchedule` | `"weekly"` |
| `StartDate` | `clock.Now()` (today) |
| `CreatedBy` | Uploader user ID |

### Bank Info Update Policy

For existing employees without banking: fill from STK data (BankAccount, BankAccountName, BankID). If employee already has bank info: leave unchanged.

### Error Handling

- STK sheet not found → skip silently, proceed with BCC import as before
- Individual employee creation fails → log error, skip that row, continue with rest
- Bank name not resolvable → create employee without bank info (non-blocking)
- All STK employees fail → BCC import still proceeds for existing employees

### Files to Modify

1. **`internal/app/services/bcc_import_service.go`**
   - Add `EmployeeRepo` and `BankRepo` to struct and constructor
   - Add `ensureEmployeesFromSTK(ctx, xf, projectID, uploaderID)` method
   - Call it between payrate lookup and assignment loading in `ProcessUpload`

2. **`internal/app/services/excel/stk_parser.go`** — no changes (already correct)

3. **`internal/app/bootstrap/services/init.go`** — pass `EmployeeRepo` + `BankRepo` to `NewBCCImportService`

### Integration Points

- `excel.ParseSTKSheet(xf)` — existing parser, returns `(nil, nil)` when no STK sheet
- `employee.EmployeeService` — already injected in `BCCImportService`
- `employee.EmployeeUserService` — already injected in `BCCImportService`
- `bankpkg.MapName()` — existing bank name resolver
- `utils.GenerateUsername()` — existing username generator

### Verification

1. Unit test: `ensureEmployeesFromSTK` with mock repos — verify create/assign/skip logic
2. Integration test: upload BCC file with STK sheet containing new employees → verify employees created, assigned, timesheets imported
3. Integration test: upload BCC file without STK sheet → verify existing behavior unchanged
4. Integration test: upload BCC file where some STK employees exist, some don't → verify partial creation works
5. Run `make api-test` to ensure no regression
