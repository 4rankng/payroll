---
type: frontend
title: Employee Mobile-First Views
description: The mobile-first employee experience — check-in/out with geofence, salary slip, advance request, profile, OTP/Zalo login, and the PWA shell that delivers it offline-capable on a phone.
tags: [frontend, employee, mobile, pwa, geofence, attendance, flexpay, otp]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-0561ebb6fa3a9d3f78919d06
    resource: repo://backend/internal/domain/advance_payment_fee_schedule.go
  - id: openwiki-source-a5c9de3ea4280bad6936ca7f
    resource: repo://backend/internal/domain/AGENTS.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-c988eccdb5077bfadc4bcd00
    resource: repo://frontend/src/components/advance-payment/AdvancePaymentConfirmSheet.tsx
  - id: openwiki-source-5800c9b9d173fd82d12e6ad1
    resource: repo://frontend/src/components/attendance/AttendanceMapDialog.tsx
  - id: openwiki-source-347b64241a95768d317d461e
    resource: repo://frontend/src/components/MobileBottomNav.tsx
  - id: openwiki-source-94818d82bdf61ffa53d6fc57
    resource: repo://frontend/src/sw.ts
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Frontend: Employee Mobile-First Views

The employee role is mobile-first. Every employee screen ships for 390px (typical phone) and 320px (narrow phone / content density) breakpoints, with a 44px minimum touch target. The PWA shell lets a user open the app from a home-screen icon, work offline, and receive web-push notifications.

## Layout

- `MobileBottomNav.tsx` — the bottom navigation with the four primary flows (Check-in, Salary, Advance, Profile).
- `ResponsivePage.tsx` — the dual-layout wrapper. Employee pages are always rendered inside it so the layout never collapses to a desktop layout.
- `sw.ts` — the service worker that caches the app shell and handles web-push events. See `features/push-notifications.md`.

The employee router (`frontend/src/pages/employee/EmployeeRouter/index.tsx`) mounts the role and dispatches between the page-level entry points.

## Pages

| Page | What it does |
|------|-------------|
| `EmployeePage/index.tsx` | Employee home — next payday, recent timesheets, recent advances, profile shortcut. |
| `EmployeeRouter/index.tsx` | Router mount for the employee role. |
| `FlexiblePayEmployeePage/index.tsx` | The FlexPay flow: request, history, salary notification read-back. |

`Login.tsx` and `OTPLogin.tsx` are the role-agn entry points; the employee path also supports Zalo OA OTP (`OTPLogin.tsx`) and Zalo reset (`ZaloResetPassword.tsx`). The reset and OTP flows consume `internal/infra/zalo/` via `internal/app/services/zaloreset/` and `internal/app/services/zaloconnect/`. See `integrations/zalo-otp.md`.

## Reusable components

### Attendance

`frontend/src/components/attendance/`:

- `AttendanceMapDialog.tsx` — map preview of the geofence.
- `AdminCreateCheckInDialog.tsx` — admin-side manual check-in (used by partner too).
- `AdminAttendanceTableConfig.tsx` — table config for the admin/partner view.
- `AdminAttendanceReviewDialogs.tsx` — admin dispute resolution.

The employee-side check-in component reads GPS (`navigator.geolocation`), posts to `/api/v1/attendance/check-in`, and renders the result with a success/error toast. The check-out component is the symmetric pair. Concurrency (two taps racing) is handled by the backend per `(employee, project, date)`. See `features/attendance-geofence.md`.

### Advance payment

`frontend/src/components/advance-payment/`:

- `AdvancePaymentConfirmSheet.tsx` — the confirmation sheet for a new advance request.
- `AdvancePaymentHistoryCard.tsx` — history of the employee's advances.
- `AdvancePaymentMobileEmployeeList.tsx`, `AdvancePaymentMobileList.tsx` — list variants for mobile.
- `AdvancePaymentEmailDialog.tsx` — admin email dialog (not employee-facing).
- `AdvancePaymentExportDialog.tsx` — admin export dialog.
- `AdvancePaymentPageHeaderMobile.tsx` — the mobile page header.
- `actions/` — small action components (request, cancel).

The request sheet shows the fee preview (which always matches what the disbursement-time fee calculator produces — see `features/flexpay.md`) and the eligible quota from attendance earnings. On submit it posts to `/api/v1/advance-payments/request` and renders the success or the kill-switch error (`tạm ngừng ứng lương`).

## Mobile-first specifics

- **Breakpoints** — 390px primary, 320px narrow. Both verified by Playwright.
- **Touch targets** — every interactive element ≥44px.
- **Inputs** — `inputmode="decimal"` for VND amounts; Vietnamese keyboard hints where helpful.
- **Geolocation** — the check-in flow requests `geolocation` permission on first use; explains the why.
- **Offline** — service worker caches the shell and the last-read timesheet summary so the home screen renders even with no network; submission surfaces a queued state.

## Vietnamese-only copy

All UI is Vietnamese. Labels such as `tạm ngừng ứng lương`, `Chấm công`, `Tạm ứng`, `Phiếu lương`, `Hồ sơ` are inline in the components.

## Backend features exercised

- **Attendance** — `features/attendance-geofence.md`. The employee check-in/out flow drives the `attendances` table and the quota-credit worker.
- **FlexPay** — `features/flexpay.md`. The advance request flow drives `advance_payment_requests`, the fee schedule, and the wallet disbursement.
- **Push notifications** — `features/push-notifications.md`. The PWA service worker receives push events and renders them with `Notification`.
- **Zalo OTP** — `integrations/zalo-otp.md`. The OTP login and password reset channels.

## Tests

- `EmployeePage/index.test.tsx`, `EmployeeRouter/index.test.tsx`, `FlexiblePayEmployeePage/index.test.tsx` — page-level coverage.
- `frontend/src/components/attendance/AdminCreateCheckInDialog.test.tsx`, `AdminAttendanceTableConfig.test.tsx` — admin attendance dialogs.
- `frontend/src/components/advance-payment/AdvancePaymentHistoryCard.test.tsx`, `AdvancePaymentPageHeaderMobile.test.tsx` — mobile UI contracts.
- Playwright runs at 390px and 320px cover the live behavior.

## Relationships

- Shared platform layer — `frontend/shared-platform.md` (TanStack Query, PWA, design system, auth context).
- Admin role — `frontend/admin-views.md`. The admin pages provide the dispute-resolution UI for employee-side issues.
- Partner role — `frontend/partner-views.md`. The partner is the project-scoped admin who approves employee requests.
