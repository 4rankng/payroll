<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# config — Application Configuration

## Purpose

Application-level configuration including API base URL, route definitions, table column configurations for each domain (employees, projects, users, partner views), and role-specific table/column settings. Contains both static configuration data and some presentation-level components for table rendering.

## Key Files

| File | Description |
|------|-------------|
| `api.config.ts` | API base URL configuration, endpoint paths, and HTTP client settings |
| `constants.ts` | Application-wide constants (page sizes, status labels, etc.) |
| `actions.ts` | Action button configurations for table rows and page headers |
| `project-table-columns.tsx` | Column definitions for the admin project data table |
| `project-table-mobile.tsx` | Mobile-optimized project list card layout |
| `partner-employee-columns.tsx` | Column definitions for partner employee tables |
| `partner-employee-table-columns.tsx` | Partner employee table column configuration |
| `partner-employee-table-desktop.tsx` | Desktop partner employee table component |
| `partner-employee-table-mobile.tsx` | Mobile partner employee list component |
| `partner-project-columns.tsx` | Column definitions for partner project tables |
| `partner-project-mobile.tsx` | Mobile partner project card layout |
| `partner-timesheet-columns.tsx` | Column definitions for partner timesheet tables |
| `partner-timesheet-mobile.tsx` | Mobile partner timesheet list component |
| `employee-table-desktop.tsx` | Desktop employee data table |
| `employee-table-mobile.tsx` | Mobile employee list card layout |
| `user-table-columns.tsx` | Column definitions for user management tables |
| `user-table-mobile.tsx` | Mobile user list card layout |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `dashboard/` | Dashboard-specific configuration and table columns |
| `partner-dashboard/` | Partner dashboard configuration |
| `partner-employees/` | Partner employee view configuration |
| `partner-timesheet/` | Partner timesheet view configuration |

## For AI Agents

### Working In This Directory

- `api.config.ts` defines all API endpoint paths — check here before hard-coding URLs.
- Table column files define the column structure, sorting, and filtering for data tables.
- When adding new table views, create a column config file here and import it in the page.
- Mobile table variants render card-style layouts instead of traditional table rows.

### Testing Requirements

- Run `pnpm type-check` to verify column definition types.

### Common Patterns

- **Column definition**: Uses TanStack Table column helper pattern with `accessorKey`, `header`, `cell` renderers.
- **Mobile variant**: Separate `*-mobile.tsx` file renders the same data as cards for small screens.

## Dependencies

### Internal
- `../types/` for entity type definitions
- `../components/ui/` for table and badge components

### External
- TanStack Table (column definitions), React Router

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
