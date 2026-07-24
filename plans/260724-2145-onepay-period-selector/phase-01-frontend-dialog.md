---
phase: 1
title: Frontend dialog and wiring
status: completed
---

# Phase 01: Frontend Dialog and Wiring

## Context

- [Plan](plan.md)
- `frontend/src/pages/admin/TimesheetPage/index.tsx`
- `frontend/src/hooks/timesheet/useBulkTransferExportForm.ts`
- `frontend/src/components/timesheet/bulk-transfer-export/BulkTransferDateRangeSection.tsx`

## Overview

Add a small OnePay-specific dialog composed from the existing bulk-transfer form logic and weekly date-range UI, then replace the immediate OnePay export handler with dialog open/submit behavior.

## Requirements

- Vietnamese labels: `Chuyển OnePay`, `Chọn kỳ lương tuần để tạo file chuyển tiền OnePay`, and `Tạo file OnePay`.
- Weekly periods only. Do not expose `for_month`, because it selects the backend's monthly employee cohort.
- Preserve the current page project as the initial project filter.
- Disable date controls and submit while exporting.
- Keep the dialog open after failure; close and reset after success.
- Responsive bottom-sheet behavior and minimum 44px actions.

## Architecture

- New UI component under `frontend/src/components/timesheet/`.
- Reuse `useBulkTransferExportForm` for state, validation, defaults, reset, and payload construction.
- Reuse `BulkTransferDateRangeSection` in weekly mode.
- Extend the form hook only as needed to seed an initial project without changing existing callers.
- Keep `useExportOnePayBulk`, the service, endpoint, and backend untouched.

## Implementation Steps

1. Add the focused OnePay export dialog.
2. Add safe initial-project support to the shared form hook if required.
3. Wire open, submit, success-close, and retry-preserving failure behavior in the admin timesheet page.
4. Add a focused interaction regression test where supported by the current frontend test setup.
5. Run frontend lint, type-check, build, and graph update.

## Todo

- [x] Implement dialog.
- [x] Wire the admin action.
- [x] Verify exact weekly payload.
- [x] Verify desktop and narrow-desktop layout at the component's real route breakpoint.
- [x] Run frontend quality gates.

## Risks

- Accidentally exposing monthly mode would switch employee cohorts.
- Reset logic could erase a retry selection after an API error.
- Current page-level filters must not silently replace the selected payroll period.

All identified risks are covered by component/hook regression tests, controlled success/error handling, and authenticated browser verification.
