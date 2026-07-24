---
title: OnePay period selector completion report
status: completed
created: 2026-07-24
---

# OnePay Period Selector Completion Report

## Summary

The admin `Chuyển OnePay` action now opens the established bulk-transfer period form in a weekly-only OnePay mode. It preserves the page's current project, sends exact weekly dates through the existing OnePay mutation, closes after success, and retains the selection after failure.

## Acceptance Evidence

| Requirement | Evidence |
|---|---|
| Period dialog opens from OnePay action | Authenticated localhost QA on `/admin/timesheet` |
| Same weekly presets/custom dates as `Chuyển lô` | Shared `BulkTransferDateRangeSection` |
| Monthly cohort cannot be selected | OnePay component test asserts no monthly UI and no `for_month` |
| Current project preserved during async project loading | Hook regression covers `projects: []` with `initialProjectIds: [42]` |
| Controls locked during export | Component regression covers disabled submit and period controls |
| No horizontal overflow | Browser checks at 1280px and 1024px |
| No backend/API contract change | Independent re-review |

## Verification

- Frontend unit tests: 43 files, 206 tests passed.
- Focused OnePay tests: 2 files, 4 tests passed.
- TypeScript: passed.
- Lint: 0 errors; 3 existing warnings in generated coverage helpers.
- Production build: passed; existing large-chunk warning remains.
- Browser: authenticated QA passed at 1280px and 1024px with no console errors or horizontal overflow.
- Knowledge graph: updated.
- Backend integration: 254 passed, 4 unrelated existing failures, 15 skipped. Failures were asset upload/lookup/download and transaction export missing `fromDate`.

## Documentation

- No evergreen docs required changes.
- Technical journal: `docs/journals/260724-onepay-period-selector.md`.

## Unresolved Questions

- None for this feature.
