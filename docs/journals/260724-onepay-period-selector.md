# OnePay Period Selector

**Date**: 2026-07-24 21:59 +08
**Severity**: High
**Component**: Admin timesheet OnePay export
**Status**: Resolved

## What Happened

We stopped the OnePay export from firing immediately and moved it behind the same weekly period selector used by `Chuyển lô`. The dialog now opens with the current page project preselected when possible, stays weekly-only, and closes only after a successful export. The backend request contract did not change.

The ugly part was the project-scope loading race. The dialog could open before project data was fully available, so the shared form could lose the intended page project on first render and send the wrong scope. That is exactly the kind of bug that looks like “just UI state” and becomes a finance mistake if nobody notices.

## The Brutal Truth

This was frustrating because the bug was not dramatic. It was quiet, stateful, and easy to miss until it was already turning into bad payroll output. The real risk was shipping a OnePay action that looked correct in the browser but could still build the wrong cohort underneath it. That is not a cosmetic defect. That is the sort of thing that burns trust.

## Technical Details

- `BulkTransferExportDialog` now supports `mode="onepay"` and a OnePay-specific copy path.
- `TimesheetPage` opens the dialog instead of calling the mutation directly.
- `useBulkTransferExportForm` now accepts `initialProjectIds` and reapplies them when the dialog opens, which fixes the loading race.
- The form now preserves weekly-only behavior and avoids exposing `for_month`, which would switch the backend to the monthly employee cohort.
- Verification: 43 files / 206 frontend tests, lint passed with 0 errors and 3 generated coverage warnings, build passed, authenticated browser check at 1280x1024 showed no overflow or runtime errors, backend integration ran 254 pass / 4 unrelated failures / 15 skipped.

## What We Tried

- Reusing the existing bulk-transfer form and weekly range section instead of inventing a new OnePay-only path.
- Passing the current page project into the dialog so the first render matches the page filter.
- Keeping the submit flow on the existing `useExportOnePayBulk` mutation instead of changing the API surface.

## Root Cause Analysis

The root mistake was assuming the page-level project filter and the dialog’s internal form state would stay aligned by default. They do not. The form had to own the project default explicitly, and that default had to survive async project loading. Once that was obvious, the fix was straightforward: seed the form from the page selection, reapply it on open, and keep the export contract weekly-only so we never fall back into the monthly cohort path.

## Lessons Learned

- Do not trust cross-component state to stay in sync during async loading.
- If a payroll action can silently change the employee cohort, the UI needs a hard guard, not a polite label.
- Reusing the existing mutation was the right call. A new backend path would have added risk without solving the bug.

## Next Steps

- Watch the OnePay flow for any follow-up cohort mismatch reports from admin users.
- If the dialog is reused elsewhere, keep the `initialProjectIds` seeding behavior intact and do not reintroduce monthly mode.
- Owner: frontend timesheet/admin flow. Timeline: already implemented, with the current verification set passed.
