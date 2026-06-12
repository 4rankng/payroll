# QA Run Checklist — Business Logic Verification

**Date**: 2026-06-12 | **Objective**: Verify business logic of payroll app (Dimension C)
**Baseline**: `make api-test` — 492 PASS, 6 FAIL (all 6 = missing BCC sample file, not business logic bugs)

## Known Pre-existing Issues
- [ ] `docs/timesheets-excel/BCC LƯỜNG DỰ ÁN EVA Sample.xlsx` missing → 6 standard BCC import tests fail

## Test Execution Items (P0 Critical — Revenue Chain)

### Item 1: Timesheet CRUD & Validation (Desktop, Admin role)
- [ ] F07-01: Create single timesheet entry → 201
- [ ] F07-03: Future date rejected → 400
- [ ] F07-04: Before assignment start rejected → 400
- [ ] F07-05: Bulk create → 201
- [ ] F07-09: Delete paid timesheet → blocked
- [ ] F07-14: Amount = rate × hours (verify calculation)
- **Flow doc**: QA Plan 05

### Item 2: Timesheet Approval Flow (Desktop, Admin role)
- [ ] F08-01: Bulk approve → status changes to approved
- [ ] F08-06: Partner cannot approve → 403
- [ ] F08-08: Approval updates project pending_payable_vnd
- [ ] F08-09: Re-upload BCC after approval → rejected
- **Flow doc**: QA Plan 05

### Item 3: Weekly BCC Import (Desktop, Partner role)
- [ ] F10-01: Upload BCC-HC sheet → timesheets created
- [ ] F10-09: Rate lookup correct (weekday vs weekend)
- [ ] F10-11: Day type rates applied correctly
- [ ] F10-08: Re-upload overwrites unapproved
- **Flow doc**: QA Plan 06

### Item 4: Bulk Transfer & Payment (Desktop, Admin role)
- [ ] F11-01: Check auto-bulk-transfer config
- [ ] F11-02: Estimate fee for weekly period
- [ ] F11-07: Both forMonth + date range → 400
- [ ] F11-08: fromDate without toDate → 400
- [ ] F12-01: Export bulk transfer Excel
- **Flow doc**: QA Plan 07

### Item 5: Advance Payment Lifecycle (Desktop, Employee + Admin)
- [ ] F13-01: Get advance payment info
- [ ] F13-02: Calculate fee preview
- [ ] F13-03: Create advance request → PENDING
- [ ] F13-05: Cancel PENDING request
- [ ] F13-17: Request during locked period → blocked
- **Flow doc**: QA Plan 08

### Item 6: Wallet & Financial (Desktop, Admin role)
- [ ] F15-01: Get wallet balance
- [ ] F16-02: Create expense transaction + double-entry
- [ ] F16-06: Partial settle transaction
- [ ] F17-05: Settlement date in future → rejected
- [ ] F18-07: Reverse entry creates mirror
- **Flow doc**: QA Plan 09

### Item 7: Cross-Role Access Control (Desktop, Multi-role)
- [ ] E2E-09: Admin accesses all → 200
- [ ] E2E-09: Partner blocked from admin endpoints → 403
- [ ] E2E-09: Employee scoped to own data
- **Flow doc**: QA Plan 15

## Test Execution Items (P1 High)

### Item 8: Employee CRUD & Assignment (Desktop, Admin role)
- [ ] F03: Employee CRUD happy path
- [ ] F05: Assign/remove employee from project
- **Flow doc**: QA Plan 03, 04

### Item 9: Loan Management (Desktop, Admin role)
- [ ] F19: Create lender, loan, disburse, repay
- **Flow doc**: QA Plan 10

### Item 10: Sao Ke & Statements (Desktop, Admin role)
- [ ] F21: Send payroll email, settle, reconciliation
- **Flow doc**: QA Plan 11
