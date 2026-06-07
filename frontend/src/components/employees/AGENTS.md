<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# employees — Employee Management Components

## Purpose

Components for employee CRUD, import, display, and status management. Includes the employee list with filters, import modal for bulk Excel uploads, detail cards (bank info, attendance, check-in), mobile card layouts, and the employee portal header for employee-role views.

## Key Files

| File | Description |
|------|-------------|
| `EmployeeImportModal.tsx` | Excel bulk import modal with validation preview and error display |
| `EmployeeListContent.tsx` | Employee list content with pagination and infinite scroll |
| `EmployeeFiltersBar.tsx` | Filter bar (status, project, search) |
| `EmployeeStatusFilter.tsx` | Status filter dropdown (active/inactive) |
| `EmployeePageHeader.tsx` | Page header with add/import actions |
| `EmployeeMobileCard.tsx` | Mobile employee card layout |
| `EmployeeSummaryCard.tsx` | Employee summary stats card |
| `EmployeeSummaryCards.tsx` | Grid of summary statistic cards |
| `EmployeeProjectsList.tsx` | List of projects assigned to an employee |
| `EmployeePortalHeader.tsx` | Header for employee portal (employee role) |
| `EmployeeAttendanceHistoryCard.tsx` | Attendance history card for employee detail |
| `EmployeeBankInfoCard.tsx` | Bank information display card |
| `EmployeeCheckInCard.tsx` | Check-in status card for flexible employees |
| `EmployeeEmptyStates.tsx` | Empty state variants for employee views |
| `MissingBankDetailsSection.tsx` | Warning section for employees without bank details |
| `InfiniteScrollInfo.tsx` | Info component showing infinite scroll status |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `details/` | Employee detail view components |
| `ChangePasswordSheet.tsx` | Password change sheet for employee settings |

## For AI Agents

### Working In This Directory

- Employee import uses Excel parsing with specific Vietnamese column mappings.
- Employee list supports both paginated table view and infinite scroll.
- Mobile views use card layouts (`EmployeeMobileCard`) instead of table rows.
- Employee status flow: active -> inactive (with assignment cleanup).

### Testing Requirements

- E2E tests in `tests/e2e/employees.spec.ts` cover CRUD and import.
- Run `pnpm type-check` after changes.

### Common Patterns

- **Import pattern**: Upload -> parse -> validate rows -> show preview -> confirm -> batch create.
- **List pattern**: Filters + summary cards + paginated/infinite list.
- **Detail pattern**: Detail sheet opened from list with tabs for different info sections.

## Dependencies

### Internal
- `../../hooks/api/useEmployees.ts` for data fetching
- `../../utils/employeeHelpers.ts` for display helpers
- `../ui/` for base components

### External
- xlsx (SheetJS), react-hook-form

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
