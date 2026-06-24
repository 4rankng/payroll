# QA Run Checklist — Mobile + Tablet Full Regression

**Date**: 2026-06-24 | **Objective**: Test all logic flow + fix all logic and visual bugs for **mobile (375px) and tablet (768px)** screen sizes
**Scope**: Dimensions A (Visual), B (Visual Mobile/Tablet), C (Business Flow)
**Viewports**: Mobile 375x812, Tablet 768x1024, Desktop 1440x900 (regression baseline)

---

## Test Accounts (Verified)
- Admin: `frankng` / `Admin123`
- Partner: `ketoan` / `Admin123`
- Employee: `tiennv` / `Admin123` (chấm công + ứng lương)
- Employee: `duynv` / `Admin123` (advance payment focus)

---

## Execution Items (P0 — Critical Revenue Chain)

### [in_progress] Item 1: Employee Check-in/Check-out Flow (Mobile + Tablet)
- Roles: Employee (tiennv)
- Viewports: mobile (375x812), tablet (768x1024)
- Dimensions: B + C
- Touch targets ≥44px on mobile, ≥44px on tablet
- Verify: Vào làm → Tan ca → Hoàn thành state transitions
- Verify: timer counts up during work
- Verify: no English "Check In/Check Out" leakage
- Verify: topbar fills viewport, no horizontal overflow
- Flow doc: Employee flow spec

### [ ] Item 2: Employee Advance Payment Flow (Mobile + Tablet)
- Roles: Employee (duynv)
- Viewports: mobile, tablet
- Dimensions: B + C
- Verify: amount input → request → confirm → cancel cycle
- Verify: lịch sử yêu cầu updates after action
- Verify: Vietnamese-only labels (no English fallback)

### [ ] Item 3: Admin Dashboard (Mobile + Tablet)
- Roles: Admin (frankng)
- Viewports: mobile, tablet
- Dimensions: B
- Verify: bottom nav (if any) renders correctly
- Verify: cards stack/wrap properly at 768px
- Verify: no horizontal scroll on either viewport

### [ ] Item 4: Partner Dashboard + Timesheets (Mobile + Tablet)
- Roles: Partner (ketoan)
- Viewports: mobile, tablet
- Dimensions: B + C
- Verify: timesheet list/table responsive
- Verify: payment history accessible

### [ ] Item 5: Admin Timesheet Operations (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B + C
- Verify: timesheet table responsive / scrollable

### [ ] Item 6: Admin Projects / Employees CRUD (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B + C

### [ ] Item 7: Admin Advance Payments Management (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B + C

### [ ] Item 8: Admin Wallet + Transactions (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B + C

### [ ] Item 9: Admin Loans + Lenders (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B

### [ ] Item 10: Admin Settings + Audit + System Health (Mobile + Tablet)
- Roles: Admin
- Viewports: mobile, tablet
- Dimensions: B

### [ ] Item 11: Cross-Role Access Control (All viewports)
- Roles: all
- Dimensions: C
- Verify: employee blocked from admin endpoints; partner scoped data; admin full access

### [ ] Item 12: Login Page (Mobile + Tablet)
- All roles
- Viewports: mobile, tablet
- Dimensions: B
- Verify: form usable, no overflow

---

## Bug Tracking

### Bug #1: ___
- Severity: P0/P1/P2
- Viewport: mobile/tablet
- Description:
- Status: open/in-progress/fixed/verified

### Bug #2: ___
- ...

### Bug #3: ___
- ...

---
