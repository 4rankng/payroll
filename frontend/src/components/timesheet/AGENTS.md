<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# timesheet — Timesheet Feature Components

## Purpose

Components for the timesheet management feature: entry tables, calendar view, bulk import/export, approval status badges, grouped tables by employee, BCC upload, mobile entry forms, email history, and edit request handling. This is one of the largest component domains in the application.

## Key Files

| File | Description |
|------|-------------|
| `TimesheetEntryTable.tsx` | Main desktop timesheet entry table with inline editing |
| `TimesheetGroupedTable.tsx` | Timesheet entries grouped by employee with collapsible sections |
| `TimesheetListTable.tsx` | Simplified timesheet list view |
| `TimesheetCalendarView.tsx` | Calendar-based timesheet visualization |
| `TimesheetContext.tsx` | React context providing shared timesheet state and actions |
| `TimesheetFilters.tsx` | Filter bar (date range, status, employee, project) |
| `TimesheetProjectFilter.tsx` | Project-specific filter dropdown |
| `TimesheetStatusBadge.tsx` | Status badge with color coding (pending/approved/paid/rejected) |
| `TimesheetSummaryCards.tsx` | Summary statistic cards (total hours, entries, amounts) |
| `TimesheetDateNavigation.tsx` | Date range navigation (prev/next period) |
| `TimesheetDateHeader.tsx` | Date column header with day-of-week |
| `TimesheetDisplaySection.tsx` | Conditional display section wrapper |
| `TimesheetEmptyState.tsx` | Empty state when no timesheets found |
| `TimesheetPageHeader.tsx` | Page header with title and action buttons |
| `TimesheetMonthSelector.tsx` | Monthly period selector |
| `TimesheetMobileList.tsx` | Mobile-optimized timesheet list |
| `TimesheetEntryMobile.tsx` | Mobile timesheet entry form |
| `MobileHourEntry.tsx` | Mobile hour input component |
| `MobileTimesheetSummary.tsx` | Mobile summary card |
| `TimesheetImportDialog.tsx` | Excel/CSV import dialog |
| `TimesheetsExportDialog.tsx` | Export dialog (Excel, CSV, PDF) |
| `TimesheetTemplateExportDialog.tsx` | Template download for bulk import |
| `PayrollReportExportDialog.tsx` | Payroll report export dialog |
| `PayrollReportEmailDialog.tsx` | Email payroll report dialog |
| `BCCUploadModal.tsx` | BCC (Bang Cham Cong) timesheet upload modal |
| `BulkTransferExportDialog.tsx` | Bulk transfer export dialog |
| `BulkTransferResultUploadDialog.tsx` | Bulk transfer result upload dialog |
| `MarkExternallyPaidDialog.tsx` | Mark timesheets as externally paid |
| `EditRequestBadge.tsx` | Edit request status badge |
| `EditRequestTable.tsx` | Edit request review table |
| `EmailHistorySheet.tsx` | Email history slide-over panel |
| `UploadHistorySheet.tsx` | Upload history slide-over panel |
| `EmployeeDropdownAdapter.tsx` | Employee dropdown adapter for timesheet filters |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `ChuyenLoDialog/` | Chuyen Lo (transfer type) dialog components |
| `bulk-transfer-export/` | Bulk transfer export utilities |
| `components/` | Shared timesheet sub-components |
| `mobile/` | Mobile-specific timesheet components |
| `utils/` | Timesheet-specific utility functions |

## For AI Agents

### Working In This Directory

- `TimesheetContext.tsx` provides shared state — consume it via `useTimesheetContext()`.
- Timesheet status flow: `pending` -> `approved` -> `paid` (or `rejected` at any stage).
- BCC uploads parse Vietnamese-format Excel files with specific column layouts.
- Mobile components use card-based layouts instead of tables.
- All date handling uses `dateHelpers.ts` for timezone consistency (Asia/Ho_Chi_Minh).

### Testing Requirements

- E2E tests in `tests/e2e/timesheet.spec.ts` cover CRUD and approval flows.
- Test file imports with the BCC upload dialog using sample Excel files.

### Common Patterns

- **Grouped table**: Entries grouped by employee, with expand/collapse and subtotals.
- **Import pattern**: Upload -> parse -> validate -> preview -> confirm import.
- **Status badges**: `TimesheetStatusBadge` maps status enum to color and label.

## Dependencies

### Internal
- `../../hooks/api/useTimesheets.ts` for data fetching
- `../../utils/timesheetHelpers.ts` for formatting and display
- `../../utils/timesheetTransformers.ts` for data transformation
- `../../lib/queryKeys.ts` for cache keys
- `../ui/` for base components (Table, Dialog, Badge, etc.)

### External
- date-fns, xlsx (SheetJS), react-hook-form

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
