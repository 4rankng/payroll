<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# utils — Utility Functions

## Purpose

Pure utility functions and helper modules for date formatting, number formatting, cache updates, Vietnamese text processing, file operations, and domain-specific transformations. These are `.ts` files containing no React hooks or UI code — only pure functions and constants.

## Key Files

| File | Description |
|------|-------------|
| `dateHelpers.ts` | Date formatting, parsing, and manipulation using date-fns |
| `weekPeriodHelpers.ts` | Weekly period calculations (start/end of week, period ranges) |
| `monthPeriodHelpers.ts` | Monthly period calculations |
| `date-range.utils.ts` | Date range selection utilities |
| `formatters.ts` | Number, currency, and percentage formatting |
| `vietnamese.ts` | Vietnamese text processing (normalization, accent handling) |
| `vietnameseHelpers.ts` | Vietnamese string comparison and search helpers |
| `vietnameseNormalization.ts` | Vietnamese diacritics normalization for search |
| `numericInput.ts` | Numeric input formatting and validation |
| `validators.ts` | Form validation utility functions |
| `cacheUpdates.ts` | TanStack Query cache update helper functions |
| `error-handler.ts` | Centralized error handling and user-friendly error messages |
| `file-download.ts` | File download trigger utilities |
| `file-upload.ts` | File upload validation and processing |
| `file-naming.ts` | File naming convention helpers |
| `fileGrouping.ts` | File grouping and categorization logic |
| `xls-export.ts` | Excel export utility (SheetJS) |
| `import-errors.ts` | Import error message formatting |
| `projectHelpers.ts` | Project-related helper functions (status labels, type labels) |
| `employeeHelpers.ts` | Employee-related helpers |
| `partnerProjectHelpers.ts` | Partner-scoped project helpers |
| `timesheetHelpers.ts` | Timesheet display helpers (status labels, duration formatting) |
| `timesheetTransformers.ts` | Timesheet data transformation for tables and charts |
| `timesheet.utils.ts` | Timesheet utility functions |
| `timesheetBulkHelpers.tsx` | Bulk timesheet operation UI helper components |
| `timesheetEditRequests.ts` | Timesheet edit request helpers |
| `advancePaymentHelpers.ts` | Advance payment calculation and display helpers |
| `approvalHelpers.ts` | Approval workflow helpers |
| `loanHelpers.ts` | Loan calculation helpers (amortization, interest) |
| `badge-styles.ts` | Badge color/style mapping for status badges |
| `avatarHelpers.ts` | Avatar URL and initials helpers |
| `userHelpers.ts` | User display helpers (role labels, name formatting) |
| `notification-helpers.ts` | Notification formatting helpers |
| `notification-navigation.ts` | Notification click navigation routing |
| `notification-styles.ts` | Notification type-to-style mapping |
| `sorting.ts` | Generic sorting utilities with Vietnamese support |
| `email.ts` | Email validation and formatting |
| `cron-human.ts` | Cron expression to human-readable text (Vietnamese) |
| `scheduleSummary.ts` | Schedule summary formatting |
| `dom-safety.ts` | DOM safety utilities (XSS prevention, safe HTML) |
| `environment.ts` | Environment detection utilities |
| `vibration.ts` | Haptic feedback utility for mobile devices |
| `payrateHelpers.ts` | Payrate calculation helpers |
| `payrate-test.ts` | Payrate test data and fixtures |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `employeePortal/` | Employee portal-specific utility functions |
| `employees/` | Employee domain utility functions |
| `excel/` | Excel import/export processing utilities |
| `pdf/` | PDF generation utilities |
| `timesheet/` | Timesheet-specific utility functions |

## For AI Agents

### Working In This Directory

- All files here must be pure functions — no React hooks, no side effects, no DOM access (except `dom-safety.ts`).
- Vietnamese text processing must go through `vietnamese.ts` / `vietnameseHelpers.ts` for proper accent handling.
- Date formatting must use `dateHelpers.ts` (which wraps date-fns) for consistency.
- Cache update helpers in `cacheUpdates.ts` are used by mutation hooks to update TanStack Query cache.

### Testing Requirements

- `weekPeriodHelpers.test.ts` exists as a unit test example.
- Pure functions here are easy to unit test — prefer testing utilities over testing components.

### Common Patterns

- **Helper naming**: Domain + "Helpers" (e.g., `timesheetHelpers`, `projectHelpers`).
- **Formatter naming**: Purpose-based (e.g., `formatters.ts` for generic, `vietnamese.ts` for locale-specific).
- **Import pattern**: `import { formatVND, formatDate } from '@/utils/formatters'`.

## Dependencies

### Internal
- `../types/` for TypeScript interfaces
- `../lib/` for query key constants (used by `cacheUpdates.ts`)

### External
- date-fns, xlsx (SheetJS)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
