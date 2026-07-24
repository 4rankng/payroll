---
title: OnePay period selector
status: completed
priority: P1
effort: small
branch: main
tags: [frontend, onepay, timesheet]
created: 2026-07-24
---

# OnePay Period Selector

## Overview

Open a focused weekly payroll-period dialog from `Chuyển OnePay` instead of exporting the page's whole selected month immediately.

## Phases

| Phase | Status | Detail |
|---|---|---|
| 01 | Completed | [Frontend dialog and wiring](phase-01-frontend-dialog.md) |

## Dependencies

- Existing `BulkTransferDateRangeSection`
- Existing `useBulkTransferExportForm`
- Existing `useExportOnePayBulk` mutation and backend request contract

## Success Criteria

- `Chuyển OnePay` opens a responsive dialog with the same weekly period presets and custom date range behavior as `Chuyển lô`.
- The current page project remains preselected when applicable.
- Submit sends exact weekly `fromDate` and `toDate` values to the existing OnePay export mutation.
- A successful download closes and resets the dialog; an error preserves the chosen period for retry.
- No backend, endpoint, or public request-type changes.
- Frontend lint, type-check, and focused interaction verification pass.
