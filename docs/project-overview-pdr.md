# Project Overview & Product Development Requirements

## Purpose

Payroll management system for Vietnamese companies. Replaces manual spreadsheet-based payroll tracking with an automated system handling timesheets, advance payments, disbursements, attendance tracking, and financial reporting.

## Problem Space

Vietnamese companies manage employees across multiple projects with complex pay structures:

- Employees assigned to projects with varying pay rates and schedules
- Weekly and monthly timesheet tracking with approval workflows
- Employees need advance payment access between salary cycles
- Disbursement to bank accounts requires integration with Vietnamese payment providers
- Partners (project managers) need project-scoped visibility without full admin access
- Mobile workforce requires mobile-first self-service (check-in/out, view earnings)

## Roles & Personas

| Role | Vietnamese | Scope |
|------|-----------|-------|
| **Admin** | Quan tri | Full system access: employees, projects, timesheets, payroll, disbursements, settings, audit logs |
| **Partner** | Doi tac | Project-scoped: view/manage own projects, employee assignments, timesheets, and timesheet-related payroll reports |
| **Employee** | Nhan vien | Self-service (mobile-first): view own timesheets, earnings, advance payment requests, attendance check-in/out |

## Functional Scope

### Timesheet Management
- Weekly and monthly timesheet creation via BCC (Bo cong chiec / Timesheet Workbook) file upload
- Bulk timesheet operations (create, preview, update, delete)
- Approval workflows with edit-request flow
- Timesheet validation: daily hour limits, assignment conflicts, day-type constraints
- Payroll report (sao ke) generation and email delivery

### Advance Payments (FlexPay)
- Employees request advance payments against earned wages
- Tiered fee structure on disbursements
- Admin approval workflow: Pending -> Approved -> Disbursed / Rejected
- Retry mechanism for failed disbursements
- Wallet balance tracks available funds (earnings minus pending advances)

### Salary Disbursement
- Provider-agnostic architecture: OnePay (production), 9Pay (sandbox/dev)
- Bulk transfer support for batch disbursements
- IPN (Instant Payment Notification) webhook handling with IP whitelist
- Disbursement polling for status tracking
- Fee charged only at transfer execution (zero on preflight)
- Read-only bank transfer history screen for Admin and Partner roles, grouped by employee and fixed weekly cycle, showing completed transfer references and amounts only

### Attendance & Check-In/Out
- Employee self check-in/out with GPS geofence validation
- Time windows: check-in (shift start +/- 1h), check-out (shift start + K to shift start + K+3h)
- Automatic rejection of incomplete attendance (no check-out within window)
- Admin review workflow for device-GPS-failed check-ins
- Check-in health dashboard with drill-down (successful/failed attempts)

### Wallet & Ledger Accounting
- Double-entry ledger: every financial movement creates debit + credit entries
- Wallet aggregate with state machine (top-up, payment, settlement)
- Available balance = total earnings - pending advance payments
- Transaction codes for audit trail

### Project & Employee Management
- Projects with salary period configuration (start day / end day)
- Employee assignment to projects with pay rates (daily/hourly, multiple positions)
- Bank account management via STK sheet upload
- Auto-employee creation from STK data

### Notifications & Communication
- Web push notifications (PWA with VAPID keys)
- Email delivery via Resend (production) or sandbox provider
- Audit logging for file imports/exports

## Non-Goals

- Multi-company/tenant support beyond the single-deployment model
- Full general-ledger accounting (system tracks payroll flows only)
- Vietnamese tax calculation or statutory deductions
- Employee onboarding workflows (employees created via import or admin action)
- Legacy browser support

## Domain Glossary

| Term | Vietnamese | Definition |
|------|-----------|-----------|
| **FlexPay** | - | Advance payment system allowing employees to request early payout of earned wages |
| **BCC** | Bo Cong Chiec | Timesheet workbook file uploaded by admin containing weekly/monthly hour records |
| **STK** | So Tai Khoan | Bank account sheet used to import/update employee bank account details |
| **Sao ke** | Sao ke | Payroll report / payslip document generated per salary period |
| **Timesheet** | Bang cham cong | Record of hours worked by an employee on a project for a specific date |
| **Salary Period** | Ky luong | Configurable date range within a month defining the payroll cycle (e.g., 21st of prev month to 20th of current month) |
| **Advance Payment** | Tam ung | Early wage disbursement to employee, subject to approval and fee |
| **Partner** | Doi tac | Project-level manager with scoped access to assigned projects |
| **Disbursement** | Chi tra | Transfer of funds to employee bank accounts via payment provider |
| **Wallet** | Vi | Virtual balance tracking earned wages, top-ups, and payment deductions |
| **Ledger** | So cai | Double-entry accounting record of all financial transactions |
| **Settlement** | Quyet toan | Final payroll calculation and fund allocation for a salary period |
| **Check-in/Check-out** | Cham vao / Cham ra | Employee attendance recording via mobile GPS with geofence validation |
| **IPN** | - | Instant Payment Notification — webhook from payment provider confirming transfer status |
| **Geofence** | - | GPS boundary around a project location used to validate employee location at check-in/out |

## Technical Constraints

- All business time must use `clock.Now()` (Asia/Ho_Chi_Minh), never `time.Now()`
- Production server is x86_64/amd64 only — no arm64 Docker images
- Production payment provider is OnePay; 9Pay for sandbox/dev only
- GORM for all database access — no raw SQL in application code
- Repository pattern with domain types at handler/service boundaries
- Event-driven architecture with EventBus for domain events
