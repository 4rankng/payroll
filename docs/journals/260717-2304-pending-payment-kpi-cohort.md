# Pending-payment KPI had three different meanings

**Date**: 2026-07-17 23:04
**Severity**: High
**Component**: Timesheet summary / payroll export / frontend filters
**Status**: Resolved

## What Happened

The UI KPI labeled `Chờ thanh toán` was not backed by a single cohort. The summary path was effectively using `approved | pending_approval` with `payment_status != paid`, while the grouped table and payroll export were drifting toward `approved + pending`. That made the number look stable while the underlying record set disagreed. The screenshot that finally exposed it showed a summary-only record: the KPI counted it, but the grouped table did not.

The fix is to stop pretending this is a generic status filter. We now have a canonical backend cohort for pending payment, a `pending_payment` API token with deterministic mixed-token handling, the summary query and bulk-transfer planner both reuse that cohort, and the frontend filter union is explicit instead of guessing from `status`/`payment_status` strings.

## The Brutal Truth

This was a classic “same label, different math” mess. The frustrating part is that the UI made it look coherent right up until the screenshot proved the table and summary were lying to each other. We were one bad export away from arguing about numbers that were never comparable in the first place.

## Technical Details

- Root cause: pending-payment logic was duplicated and inconsistent across summary, analytics, export planning, and frontend filter typing.
- Canonical helper added: `domain.NewPendingPaymentTimesheetFilters()` returns `approved + (pending | failed)`.
- `TimesheetResponseService.ParseTimesheetFilters()` now treats `pending_payment` as a full cohort and refuses to let mixed tokens broaden it.
- `TimesheetAnalyticsRepository.GetSummaryStats()` now uses the canonical cohort instead of `approved | pending_approval` plus `payment_status != paid`.
- `ExportPlanner.selectAndAggregate()` now reuses the same cohort for normal and forced payroll selection.
- Frontend filter typing moved to `TimesheetStatusFilter`, with `buildTimesheetStatusFilters()` and tests covering the `pending_payment` token explicitly.

## What We Tried

- I traced the mismatch through the summary query, grouped table, and export planner until the divergent predicates were obvious.
- I rejected a local-only UI workaround because that would have left backend export math wrong.
- I also rejected broadening the cohort in place; the bug was not “too narrow,” it was “inconsistent.”

## Root Cause Analysis

We let one business concept fragment into three code paths and two type systems. The summary was written as if `pending_payment` meant “not paid yet,” the export planner treated it as payroll-eligible rows, and the frontend typed it as just another status. That is a design failure, not a typo. It happened because nobody forced a single source of truth for this cohort until the mismatch became visible.

## Lessons Learned

- If a KPI and a table are supposed to answer the same question, they must share the same predicate, not just the same label.
- Business cohorts need named helpers, not repeated filter literals.
- Mixed-token handling should be deterministic by design, or it will silently widen scope.
- Frontend unions must mirror backend contracts exactly, or the UI will invent its own reality.

## Next Steps

- Keep the regression tests in place for the response-service cohort mapping and frontend filter helper; the analytics query and export planner stay aligned by reusing the same domain helper.
- Verification is good enough for this fix: affected backend packages pass race tests; frontend passes `lint`, `typecheck`, `build`, and 131 tests across 26 files.
- Known limitation: the full backend suite still has unrelated existing failures, and the local API integration check was unavailable because the server on `:8080` was down.
