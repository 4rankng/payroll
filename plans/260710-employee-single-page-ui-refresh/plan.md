---
title: "Employee single-page UI refresh"
description: "Improve the flexible-pay employee portal's information hierarchy, readability, and accessibility while retaining its existing single-page structure and behavior."
status: pending_approval
created: "2026-07-10"
scope: frontend
---

# Employee single-page UI refresh

## Outcome

The existing `/employee` flexible-pay portal remains a single page. Its home view gives priority to the active shift, presents FlexPay without excessive nested surfaces, and initially shows only three attendance records. Employees can expand the current-month attendance history in place; request history and bank details remain available below without becoming competing visual anchors.

## Constraints

- No bottom navigation, new routes, backend, database, API, or query-contract changes.
- Preserve check-in/out GPS and geofence behavior, cancellation confirmations, notification sheet, password/logout actions, advance amount/fee validation, request confirmation/cancellation, and bank-information display.
- Use Vietnamese text, existing React/Tailwind/shadcn patterns, strict TypeScript, and at least 44 px interactive targets.
- Use accessible employee colors: `#101828`, `#475467`, `#667085`, `#07883F`, plus existing semantic warning/danger colors.

## Phases

1. [Single-page UI refresh](./phase-01-single-page-ui-refresh.md) — update reusable employee portal components, add attendance preview/full-history behavior, and verify the changed UI.

## Acceptance criteria

- The employee portal remains one page at `/employee` with no new tab navigation or routes.
- The header removes the low-value greeting, shows the employee name/date context, and preserves notification and account controls.
- A currently checked-in employee sees the active-shift card as the most prominent section; `Tan ca` uses the accessible employee green and `Hủy ca` stays a clearly secondary destructive action behind its existing confirmation.
- FlexPay shows the available balance and period clearly, then the existing amount input, fee preview, and confirmation flow without card-inside-card visual clutter.
- Attendance is initially limited to three records with a clear `Xem toàn bộ lịch chấm công` control that reveals the existing full current-month data in the same page.
- Attendance status is understandable through text and icon/color treatment; warning detail remains available only when a real anomaly exists.
- Request history and bank details are still accessible and keep their existing API-driven behavior.
- Existing public frontend/API contracts are unchanged; targeted tests plus frontend type-check and lint pass.

## Dependencies and risks

- The active `EmployeeCheckInCard` contains recent GPS/geofence and checkout-cooldown work. Visual changes must not modify its mutation inputs, guards, or confirmation paths.
- The attendance history query currently requests a full month and powers its month controls. The preview must be a presentation-only slice of that data.
- Rollback is a frontend-only revert of the touched components and styles.
