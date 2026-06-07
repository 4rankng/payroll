<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# partner — Partner Role Pages

## Purpose

Page-level route components for the Partner role. Partners have a scoped view: they can see their own projects, employees assigned to those projects, and timesheets for those employees. Partners cannot access admin features like user management, ledger, or system settings.

## Key Files

| File | Description |
|------|-------------|
| `DashboardPage/index.tsx` | Partner dashboard showing project overview and stats |
| `EmployeesPage/index.tsx` | Partner employee list (scoped to partner's projects) |
| `TimesheetsPage/index.tsx` | Partner timesheet view (scoped to partner's employees) |
| `ProjectsPage/index.tsx` | Partner project list and detail |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `DashboardPage/` | Partner dashboard |
| `EmployeesPage/` | Partner-scoped employee list |
| `ProjectsPage/` | Partner-scoped project list |
| `TimesheetsPage/` | Partner-scoped timesheet view |

## For AI Agents

### Working In This Directory

- Partner pages are **scoped** — they only show data belonging to the partner's projects.
- Partners have read-only access to most data; limited write access for timesheet approvals.
- Pages use the same components as admin but with partner-specific hooks that filter data.
- Protected by `ProtectedRoute` checking for partner role.

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Scoped queries**: Partner hooks pass partner ID filter to API calls.
- **Simplified layout**: No sidebar settings, no admin navigation items.

## Dependencies

### Internal
- `../../components/partner-dashboard/` for dashboard components
- `../../components/partner-employees/` for employee list
- `../../components/partner-projects/` for project list
- `../../components/partner-timesheet/` for timesheet views
- `../../hooks/` for partner-scoped data hooks

### External
- React Router v6, TanStack Query

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
