<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# employee — Employee Role Pages

## Purpose

Page-level route components for the Employee role (flexible pay employees). Employees can view their project assignments, check in/out for work, view their earnings quota, and request advance payments against earned wages.

## Key Files

| File | Description |
|------|-------------|
| `EmployeeRouter/index.tsx` | Router component that directs to the correct employee page based on employee type |
| `EmployeePage/index.tsx` | Standard employee page (view-only project assignments) |
| `FlexiblePayEmployeePage/index.tsx` | Flexible pay employee page with check-in/out, earnings, and advance payment request |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `EmployeeRouter/` | Employee type router |
| `EmployeePage/` | Standard employee view |
| `FlexiblePayEmployeePage/` | Flexible pay employee interactive page |

## For AI Agents

### Working In This Directory

- `EmployeeRouter` determines which page to render based on the employee's project type.
- Flexible pay employees can check in/out and request advance payments.
- Standard employees can only view their project assignments and timesheet history.
- Protected by `ProtectedRoute` checking for employee role.

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Employee type routing**: Router reads employee type from auth context and renders the appropriate page.
- **Check-in/out**: Flexible employees use a toggle button that calls the attendance API.

## Dependencies

### Internal
- `../../components/employees/` for employee components
- `../../components/advance-payment/` for advance payment request form
- `../../hooks/api/` for employee portal and attendance hooks
- `../../contexts/AuthContext.tsx` for employee identity

### External
- React Router v6, TanStack Query

<!-- MANUAL: Any manually added notes below this line is preserved on regeneration -->
