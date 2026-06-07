<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# projects — Project Management Components

## Purpose

Components for project display, filtering, status badges, and configuration. Includes project status badges, payment type badges, salary period fields, off-days picker, mobile list views, and statistics.

## Key Files

| File | Description |
|------|-------------|
| `ProjectPageHeader.tsx` | Page header with add project action |
| `ProjectStatusBadge.tsx` | Badge showing project status (active/paused/completed) |
| `ProjectPaymentTypeBadge.tsx` | Badge for payment type (flexible/monthly/weekly) |
| `ProjectFilters.tsx` | Filter bar (status, payment type, search) |
| `ProjectStats.tsx` | Project statistics summary |
| `ProjectMobileList.tsx` | Mobile-optimized project list cards |
| `SalaryPeriodFields.tsx` | Salary period configuration fields (start date, cycle) |
| `OffDaysPicker.tsx` | Off-days selection component for project schedule |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `details/` | Project detail view components |

## For AI Agents

### Working In This Directory

- Project type determines the timesheet/payrate editing mode (flexible vs. matrix).
- Salary period fields only show for monthly/weekly projects, not flexible.
- Off-days picker affects timesheet validation on the backend.

### Testing Requirements

- E2E tests in `tests/e2e/projects.spec.ts` cover CRUD.
- Run `pnpm type-check` after changes.

### Common Patterns

- **Badge pattern**: Status and type badges map enum values to colors and labels.
- **Filter + List pattern**: FilterBar + DataTable or MobileList.

## Dependencies

### Internal
- `../../hooks/api/useProjects.ts` for data fetching
- `../../utils/projectHelpers.ts` for display helpers
- `../ui/` for base components

### External
- date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
