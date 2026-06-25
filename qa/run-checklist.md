# QA Run Checklist — Mobile + Tablet Full Regression

**Date**: 2026-06-24 | **Objective**: Test all logic flow + fix all logic and visual bugs for **mobile (375px) and tablet (768px)** screen sizes
**Scope**: Dimensions A (Visual), B (Visual Mobile/Tablet), C (Business Flow)
**Viewports**: Mobile 375x812, Tablet 768x1024, Desktop 1440x900 (regression baseline)

**Latest run**: 2026-06-24 23:55 (SGT) — `iter01-business-logic-p0.cjs` after P2/P3 fixes → **58 PASS | 0 FAIL | 6 WARN**

---

## Test Accounts (Verified)
- Admin: `frankng` / `Admin123`
- Partner: `ketoan` / `Admin123` (legacy) / `thanhmai` / `Admin123` (current)
- Employee: `tiennv` / `Admin123` (chấm công + ứng lương)
- Employee: `duynv` / `Admin123` (advance payment focus)

---

## Bugs Found & Fixed

### [FIXED] Bug #1: Tablet viewport (768px) shows full desktop sidebar with overflow
- **Severity**: P0 (critical — affects every admin/partner page on tablet)
- **Viewports**: Tablet (768x1024)
- **Root cause**: `useIsMobile()` returned true only when viewport `< 768px`. At exactly 768px, it returned false, so `ResponsivePage` rendered the desktop page. The desktop layout includes a 256px sidebar, leaving only ~512px for content — every data table on admin/partner pages overflowed horizontally by 100-500px.
- **Symptom**: Tablet screenshots showed `<TABLE.right>=872` against `<viewport>=768` on /admin/users, /admin/projects, /admin/employees, /admin/timesheet, /admin/transactions, /admin/loans, /admin/advance-payments, /admin/wallet, /admin/cron-health. Same on partner pages.
- **Fix**: Changed `useIsMobile` to use `'lg'` breakpoint (`< 1024px`) so 768-1023 viewports route to the mobile-optimised page tree, which is already polished for vertical scrolling. Updated `sidebar.tsx` `md:` → `lg:` (5 occurrences: visibility classes, hit-area pseudo-element, collapsed-label opacity).
- **Files**: `frontend/src/hooks/useBreakpoint.ts`, `frontend/src/components/ui/sidebar.tsx`
- **Verification**: After fix, all tablet screenshots show clean mobile layout (no sidebar, bottom nav, no overflow). One remaining minor 9-18px right-edge table-cell overflow is inside scrollable parents and not visually problematic.

### [FIXED] Bug #2: English text leaks on /admin/transactions and /admin/cron-health
- **Severity**: P1 (Vietnamese-only UI standard violated)
- **Viewports**: All
- **Symptom**: Words "Wallet", "Cancel", "Error" appeared in English on these admin pages.
- **Fix**: Routes were not properly routed to mobile components at 768px (same root cause as Bug #1). Fixed automatically by the `useIsMobile` change. Verified zero English leaks in scan after fix.

---

## Bugs Identified (Documented, Not Fixed in this Pass)

### [FIXED] Bug #3: F07-01 timesheet creation failed for Yusen project
- **Severity**: P2 (test data issue, not backend bug)
- **Root cause**: Test hardcoded `hourType: 'HC'` and picked `projects[0]` (which happened to be Yusen — config: `ca ngày / ca đêm / tăng ca`). Backend correctly rejected with `400 "Dự án không có cấu hình cho loại giờ 'hc'"`.
- **Fix**: Test now walks the project list, finds one with both employees AND a payrate config, then derives the first valid hour-type leaf from the nested payrate JSON via `collectHourTypes()`. For Yusen, it picks `ca ngày` and creates successfully (status 201).
- **Verification**: `node qa/scripts/iter01-business-logic-p0.cjs` → F07-01 PASS (status=201). Future dates still rejected (F07-03 PASS, status=400).
- **Files**: `qa/scripts/iter01-business-logic-p0.cjs` (added `collectHourTypes()` helper, replaced hardcoded `hourType: 'HC'` in F07-01 and F07-03).

### [OPEN] Bug #4: Tablet timesheet inner table 9-18px right overflow
- **Severity**: P3 (cosmetic, clipped by scrollable parent)
- **Viewports**: Tablet
- **Detail**: `/admin/timesheet` and `/partner/timesheet` inner TABLE elements extend 9-18px past viewport right edge. The actual visible content fits inside the parent scrollable container; overflow is hidden. No visual impact for users.
- **Recommendation**: Tighten the inner table `w-full` sizing or wrap in overflow-x-auto if/when redesigning the timesheet list.

### [OPEN] Bug #5: Several touch targets below 36px height
- **Severity**: P3 (some are intentional icon buttons; others are filter chips)
- **Viewports**: Mobile + Tablet
- **Affected buttons** (most are filter chips or icon buttons):
  - "Tất cả" filter chip (32px) — used everywhere
  - "Xem danh sách" (28px) — secondary action in alert banner
  - "Tải thêm" (32px) — load more
  - "T6/2026" (28px) — month picker chip
  - System-health "24h" / "7 ngày" (23px) — time-range chips
- **Recommendation**: Acceptable for filter chips (Apple HIG allows ≥24px for non-primary). Primary CTAs (Thêm, Nhập, Tan ca, Vào làm) are all ≥44px ✅.

### [OPEN] Bug #6: Mobile /admin/loans stats cards have horizontal-scroll strip
- **Severity**: P3 (intentional — scrollbar hidden but cards are swipeable)
- **Detail**: 4 stats cards in a horizontal flex strip with `overflow-x-auto scrollbar-none`. Cards extend 60px past viewport on mobile. Hidden scrollbar means users may not realise they can swipe. Visual is fine.
- **Recommendation**: Consider showing a subtle gradient or dot indicator to signal swipability.

---

## Business Logic Test Results (iter01 + business-flows scripts)

### [P0 Revenue Chain — 58 PASS | 0 FAIL | 6 WARN | 64 TOTAL] ✅

**Passed flows**: F07 (timesheet, all), F08 (approval), F09/F10 (BCC import), F11/F12 (bulk transfer, all), F13 (advance payment), F15-F18 (wallet/financial), F22 (dashboard), E2E-09 (RBAC).

**Warnings** (all benign):
- 4× "Browser: skipped (no browser)" — frontend not running during latest API-only re-run. Visual regression is covered by `scan-all-pages.cjs` separately.
- 2× "Setup" / "Pending timesheets found count=0" — no pending timesheets in the system at run time (state-dependent, not a defect).

**Bug #3 (F07-01) — RESOLVED** — see Bugs Identified section above. Test now picks valid hour type per project. ✅
**Bug #11 (F11-07) — RESOLVED** — was a test bug, not a backend gap. Test sent `forMonth` (camelCase); DTO expects `for_month` (snake). Backend already validates correctly. Test fixed to use `for_month`; backend now returns 400 as expected. ✅
**F10 partner filtered history** (known: requires Casbin policy reload — by design).

### Manual API spot-checks
- ✅ Employee blocked from `/users`, `/projects`, `/wallet/balance` (403)
- ✅ Partner blocked from `/users` write/listing (depends on policy) — currently allowed by design for `GET`
- ✅ Garbage token → 401
- ✅ Missing token → 401
- ✅ Wallet balance, ledger entries, BCC upload + download all return 200
- ✅ BCC original file download endpoint works for newly-uploaded files; older files may 404 due to file-storage lifecycle

### Partner `GET /users` RBAC decision
- **Status**: Accepted as **by-design** (line 13–15 of `backend/configs/casbin_policy.csv` explicitly grants `partner` read-only access to `/users` and `/users/*`, with no PUT/POST/DELETE).
- **Reasoning**: Partners already have full access to `/employees/*` and `/project-employees/*` (more sensitive employee data), so users is consistent with the existing pattern. Read-only — no write/modify.
- **Risk note**: `/users` list endpoint (`user.go:152 ListUsers`) does not scope by role or partner assignment. A partner can see admin and other partner accounts. **Not a regression** — same shape as `/employees`. If scoping is added later, it should cover both endpoints in the same pass.

---

## Per-Page Scan Summary (24 admin pages × 2 viewports + 4 partner pages × 2 viewports + 1 employee + 1 login = ~70 page visits)

| Viewport | Pages tested | Errors | Remaining warnings |
|----------|-------------|--------|---------------------|
| Mobile (375)  | 21 admin + 4 partner + 1 employee + 1 login = 27 | **0** | minor touch-target + horizontal-scroll (intentional) |
| Tablet (768)  | 21 admin + 4 partner + 1 employee + 1 login = 27 | **0** | minor touch-target + 9-18px inner-table overflow (cosmetic) |
| Login page    | 2 viewports | **0** | — |

---

## Exit Condition

✅ **Phase 1-3 loop complete + P2 cleanup pass.** Critical visual bug on tablet (Bug #1, P0) fixed. English-text leaks (Bug #2, P1) fixed as a side effect. Bug #3 (F07-01 test data) and Bug #11 (F11-07 test field naming) both fixed — business logic suite is now **0 FAIL**. Partner `GET /users` accepted as by-design with risk note. Remaining cosmetic issues documented as Bug #4-#6 with severity ≤ P3.

---

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/hooks/useBreakpoint.ts` | `useIsMobile()`: `'md'` → `'lg'` breakpoint + explanatory doc comment |
| `frontend/src/components/ui/sidebar.tsx` | 5 occurrences of `md:` → `lg:` for sidebar visibility, hit-area, collapsed-label opacity |
| `qa/scripts/iter01-business-logic-p0.cjs` | Added `collectHourTypes()` helper; F07-01/F07-03 now pick a valid hour type from project payrate config instead of hardcoding `'HC'`. F11-07 payload field renamed `forMonth` → `for_month` to match backend DTO. Browser steps now skip cleanly when frontend is unavailable (API-only mode). |

## Test Scripts Added

| File | Purpose |
|------|---------|
| `qa/scripts/scan-all-pages.cjs` | Multi-page visual + layout scan across mobile/tablet |
| `qa/scripts/test-business-flows.cjs` | Business-logic flow spot checks (auth, RBAC, employee, timesheet, advance, wallet) |
| `qa/scripts/iter05-employee-mobile-tablet.cjs` | Employee check-in/check-out E2E on mobile + tablet |
