# 15 — End-to-End Cross-Module Scenarios

**Priority**: P0 | **Scope**: Full system | **Risk Level**: Critical

These scenarios test complete business journeys that span multiple modules, validating data flows correctly across system boundaries.

---

## E2E-01: Monthly Payroll Revenue Chain

**Description**: Complete monthly payroll cycle from project setup through payment and financial reporting.

**Modules**: Project → Employee → Assignment → Payrate → Timesheet → Approval → Bulk Transfer → Wallet → Ledger → Dashboard

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Create project with salary period (26th–25th) | Project | Project exists with status=active |
| 2 | Create employee with bank details | Employee | Employee has valid bank_id + account |
| 3 | Assign employee to project (monthly schedule, position="Nhân viên") | Assignment | Assignment created with start_date |
| 4 | Create payrate: position→day_type→shift_type→amount | Payrate | Payrate effective from assignment start_date |
| 5 | Create timesheet entries for the month | Timesheet | Entries created with correct amounts |
| 6 | Verify timesheet amounts match payrate × hours | Timesheet+Payrate | Amount = rate × hours for each entry |
| 7 | Bulk approve all timesheets | Approval | All entries status=approved |
| 8 | Verify project pending_payable_vnd increased | Project | Financial summary updated |
| 9 | Initiate bulk transfer (manual or auto) | Bulk Transfer | Transfer initiated |
| 10 | Verify timesheets payment_status='paid' | Timesheet | All timesheets marked paid |
| 11 | Verify wallet balance updated | Wallet | Payment records created |
| 12 | Verify ledger entries created (double-entry) | Ledger | Debit/credit entries balanced |
| 13 | Verify project total_payout_vnd increased | Project | Financial summary updated |
| 14 | Verify dashboard reflects new data | Dashboard | All dashboard views show updated data |
| 15 | Verify audit trail complete | Audit | All actions logged with correct timestamps |

**Pass Criteria**: All 15 steps complete without errors. Financial totals balance. Dashboard data consistent.

---

## E2E-02: Weekly Payroll with BCC Import

**Description**: Weekly payroll using BCC Excel upload for timesheet creation.

**Modules**: Project → Employee → Payrate → BCC Import → Approval → Bulk Transfer → Wallet

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Create weekly project | Project | payment_schedule=weekly |
| 2 | Assign employees with weekly schedule | Assignment | Multiple employees assigned |
| 3 | Create payrate with weekday/weekend rates | Payrate | Different rates for weekday vs weekend |
| 4 | Upload standard BCC Excel file | BCC Import | Import completed, timesheets created |
| 5 | Verify auto-created employees (unknown CCCD) | Employee | New employees created with CCCD |
| 6 | Verify timesheet amounts calculated correctly | Timesheet | Rate × hours for each position |
| 7 | Re-upload same BCC file (latest wins) | BCC Import | Unapproved entries overwritten, no duplicates |
| 8 | Bulk approve all timesheets | Approval | All entries status=approved |
| 9 | Estimate bulk transfer fee | Bulk Transfer | Fee estimate returned |
| 10 | Initiate weekly bulk transfer | Bulk Transfer | Transfer processed |
| 11 | Poll for payment completion | Timesheet | payment_status='paid' for all |
| 12 | Verify wallet payment records | Wallet | Payments match timesheet totals |

**Pass Criteria**: BCC import creates correct timesheets. Re-upload overwrites unapproved. Payment completes end-to-end.

---

## E2E-03: Weekly BCC Import (LGD Format)

**Description**: BCC import using BCC-<shiftType> sheet format (BCC-HC, BCC-OT150).

**Modules**: BCC Weekly Import → Payrate → Timesheet → Approval → Payment

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Set up project with weekly schedule | Project | Project ready for BCC import |
| 2 | Create payrate with shift type rates | Payrate | HC, OT150 rates defined |
| 3 | Upload BCC LGD.xlsx with BCC-HC sheet | BCC Import | HC timesheets created |
| 4 | Upload same file with BCC-OT150 sheet | BCC Import | OT150 timesheets added |
| 5 | Verify weekday vs weekend rate application | Timesheet | Different rates applied by day type |
| 6 | Verify employee matching via CCCD | Employee | Correct employees matched |
| 7 | Re-upload after approval fails gracefully | BCC Import | Status=failed, approved preserved |
| 8 | Complete payment cycle | Payment | All paid timesheets unchanged by re-upload |

**Pass Criteria**: Both shift types processed. Day type rates correct. Re-upload safety verified.

---

## E2E-04: Advance Payment Full Lifecycle

**Description**: Employee requests advance payment within the request window, disbursement completes, and reconciliation settles.

**Modules**: FlexPay Import → Clock → Advance Payment → Disbursement → Wallet → Sao Ke

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Upload bang luong Excel for current month | FlexPay Import | Import completed, quotas calculated |
| 2 | Set clock to day 5 (Phase 1: OPEN for previous month) | Clock | Clock frozen at day 5 |
| 3 | Employee requests advance payment | Advance Payment | Request created (PENDING) |
| 4 | Verify fee preview matches actual fee | Advance Payment | Fee calculation correct |
| 5 | Wait for poller to claim request | Advance Payment | Status → APPROVED |
| 6 | Wait for disbursement to complete | Disbursement | Status → COMPLETED |
| 7 | Verify wallet payment record | Wallet | Payment with correct amount |
| 8 | Verify employee completedAmount | Employee | Amount increased |
| 9 | Reset clock to real time | Clock | Auto-advance restored |
| 10 | Set clock to day 25 (Phase 3: OPEN for current month) | Clock | Clock at day 25 |
| 11 | Employee requests second advance | Advance Payment | Second request created |
| 12 | Complete second advance | Disbursement | COMPLETED |
| 13 | Send reconciliation email | Sao Ke | Outstanding advances cancelled |
| 14 | Verify all advances settled | Advance Payment | No outstanding requests |
| 15 | Verify ledger entries for all advances | Ledger | Double-entry balanced |

**Pass Criteria**: Both phases work. Disbursement completes. Reconciliation cancels outstanding. Ledger balanced.

---

## E2E-05: Financial Management Chain

**Description**: Create financial transactions, settle them, manage loans, and verify ledger integrity.

**Modules**: Transaction → Settlement → Ledger → Loan → Dashboard

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Create expense transaction | Transaction | Transaction + ledger entry created |
| 2 | Create revenue transaction | Transaction | Transaction + ledger entry created |
| 3 | Verify double-entry for both transactions | Ledger | Debit/credit balanced for each |
| 4 | Partial settle expense (50%) | Settlement | Transaction status → partially_settled |
| 5 | Full settle expense (remaining 50%) | Settlement | Transaction status → settled |
| 6 | Reverse revenue transaction | Transaction | Status → reversed, mirror entries created |
| 7 | Verify reversal mirror entries | Ledger | Reversal has opposite debit/credit |
| 8 | Create lender | Lender | Lender created |
| 9 | Create loan from lender | Loan | Loan created (pending) |
| 10 | Disburse loan | Loan | Status → disbursed, ledger entry created |
| 11 | Partial repay loan | Loan | Status → partially_repaid |
| 12 | Full repay loan | Loan | Status → fully_repaid |
| 13 | Get cash flow summary | Ledger | All transactions + loans reflected |
| 14 | Verify dashboard financial report | Dashboard | Consistent with ledger totals |

**Pass Criteria**: All financial operations create correct ledger entries. Cash flow summary accurate. Dashboard consistent.

---

## E2E-06: Employee Removal and Visibility

**Description**: Remove employee from project mid-cycle, verify data integrity and advance payment visibility.

**Modules**: Project → Assignment → Advance Payment → Timesheet → Visibility

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Assign employee to project | Assignment | Employee assigned with start_date |
| 2 | Create timesheets for current month | Timesheet | Entries created |
| 3 | Upload bang luong for current month | FlexPay Import | Advance quota set |
| 4 | Request advance payment for employee | Advance Payment | Request created |
| 5 | Remove employee from project | Assignment | Assignment removed with last_date |
| 6 | Verify employee visible for original forMonth | Advance Payment | Employee still in list for that month |
| 7 | Verify employee NOT visible for current month | Advance Payment | Employee excluded from current month |
| 8 | Verify historical timesheets preserved | Timesheet | Past entries still accessible |
| 9 | Re-add employee to project | Assignment | New assignment created |
| 10 | Verify employee visible again | Advance Payment | Employee in list for current month |

**Pass Criteria**: Removal preserves historical data. Visibility rules enforced. Re-add restores access.

---

## E2E-07: Sao Ke Email and Settlement

**Description**: Send payroll report email, settle, then send reconciliation email.

**Modules**: Assets → Sao Ke → Advance Payment → Email

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Upload sao ke Excel file | Assets | Asset created |
| 2 | Send payroll report email with attachment | Sao Ke | Email sent with saoKeAssetId |
| 3 | Verify email history entry | Sao Ke | Email in history with correct metadata |
| 4 | Download attached asset | Assets | Binary matches uploaded file |
| 5 | Settle payroll email | Sao Ke | Email marked as settled |
| 6 | Re-settle (idempotency check) | Sao Ke | 200, no duplicate processing |
| 7 | Create advance requests | Advance Payment | Multiple PENDING requests |
| 8 | Send reconciliation email | Sao Ke | Outstanding advances cancelled |
| 9 | Verify advances cancelled | Advance Payment | No outstanding requests |
| 10 | Export reconciliation preview | Sao Ke | Binary Excel with reconciliation data |

**Pass Criteria**: Email flow works end-to-end. Settlement idempotent. Reconciliation cancels outstanding advances.

---

## E2E-08: Data Integrity Under Re-Upload

**Description**: Verify system integrity when BCC files are re-uploaded multiple times with different data.

**Modules**: BCC Import → Timesheet → Approval → Payment

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Upload BCC v1 (5 employees, 7 days) | BCC Import | 35 timesheet entries created |
| 2 | Verify all entries are draft/unapproved | Timesheet | All status = draft |
| 3 | Bulk approve all entries | Approval | All status = approved |
| 4 | Re-upload BCC v2 (updated hours for same employees) | BCC Import | Approved entries preserved, no duplicates |
| 5 | Verify approved entries unchanged | Timesheet | Original entries still approved |
| 6 | Initiate and complete bulk transfer | Payment | All approved entries paid |
| 7 | Re-upload BCC v3 after payment | BCC Import | Paid entries untouched |
| 8 | Verify no duplicate timesheets | Timesheet | Unique count = expected count |
| 9 | Verify financial totals consistent | Wallet/Ledger | No double-counting |

**Pass Criteria**: Re-upload never corrupts approved/paid data. No duplicates. Financial totals consistent.

---

## E2E-09: Cross-Role Access Control

**Description**: Verify admin vs partner role restrictions across all modules.

**Modules**: Auth → All modules

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Admin accesses all endpoints | Auth | 200 for all admin-accessible endpoints |
| 2 | Partner cannot access admin-only endpoints | Auth | 403 for admin routes |
| 3 | Partner can only see own projects | Dashboard | Only partner's projects visible |
| 4 | Partner can only see own uploads | BCC Import | Only partner's project uploads listed |
| 5 | Partner cannot update salary period | Project | 403 on salary period update |
| 6 | Partner cannot approve timesheets | Timesheet | 403 on bulk-approve |
| 7 | Employee self-service scoped to own data | Self-Service | Only own timesheets/payments visible |
| 8 | Unauthenticated access blocked | Auth | 401 on all protected endpoints |

**Pass Criteria**: Role boundaries enforced everywhere. No data leakage between roles.

---

## E2E-10: Clock-Dependent Business Rules

**Description**: Verify all time-dependent business rules respond correctly to clock manipulation.

**Modules**: Clock → Advance Payment → Timesheet → Cron

| Step | Action | Module | Verification |
|------|--------|--------|-------------|
| 1 | Set clock to day 5 (Phase 1 open) | Clock | Time confirmed |
| 2 | Create advance request → succeeds | Advance Payment | 201 |
| 3 | Set clock to day 15 (Phase 2 locked) | Clock | Time confirmed |
| 4 | Create advance request → blocked | Advance Payment | 400, locked |
| 5 | Set clock to day 25 (Phase 3, no bang luong) | Clock | Time confirmed |
| 6 | Create advance request → blocked | Advance Payment | 400, no bang luong |
| 7 | Upload bang luong for current month | FlexPay Import | Import completed |
| 8 | Create advance request → succeeds | Advance Payment | 201 |
| 9 | Create timesheet for future date → blocked | Timesheet | 400, future date |
| 10 | Reset clock to real time | Clock | Auto-advance restored |

**Pass Criteria**: All 3 phases of advance payment window work correctly with clock manipulation.

---

## Execution Summary

| Scenario | Priority | Est. Duration | Modules | Key Risk |
|----------|----------|---------------|---------|----------|
| E2E-01 | P0 | 15 min | 8 | Financial accuracy |
| E2E-02 | P0 | 10 min | 6 | BCC parsing |
| E2E-03 | P0 | 10 min | 5 | Weekly BCC format |
| E2E-04 | P0 | 20 min | 6 | Advance lifecycle |
| E2E-05 | P1 | 15 min | 5 | Ledger integrity |
| E2E-06 | P1 | 10 min | 5 | Data preservation |
| E2E-07 | P1 | 10 min | 4 | Email + settlement |
| E2E-08 | P0 | 10 min | 4 | Data integrity |
| E2E-09 | P0 | 10 min | All | Access control |
| E2E-10 | P0 | 10 min | 4 | Time-dependent rules |
| **Total** | | **~2 hours** | | |

## Existing Integration Test Coverage

All E2E scenarios have corresponding integration test flows at `backend/tests/integration/flow_*.go`. The integration tests are registered in `backend/tests/integration/main.go` and can be run with `make api-test`.
