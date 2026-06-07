<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# pages — Page-Level Route Components

## Purpose

Top-level route components organized by user role. Each page is a composition layer that wires up components from `components/` with hooks from `hooks/` and displays routed views. Three role directories (`admin/`, `partner/`, `employee/`) plus `mobile/` for mobile-optimized views and root-level pages (`Login`, `NotFound`).

## Key Files (Root Level)

| File | Description |
|------|-------------|
| `Index.tsx` | Root redirect page |
| `Login.tsx` | Login page with authentication form and session handling |
| `NotFound.tsx` | 404 page with navigation back to dashboard |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `admin/` | Admin role pages — dashboard, employees, timesheets, projects, etc. (see `admin/AGENTS.md`) |
| `partner/` | Partner role pages — dashboard, employees, timesheets (see `partner/AGENTS.md`) |
| `employee/` | Employee role pages — flexible pay portal (see `employee/AGENTS.md`) |
| `mobile/` | Mobile-optimized pages mirroring admin and partner views (see `mobile/AGENTS.md`) |

## For AI Agents

### Working In This Directory

- Pages are **composition layers** — they should not contain business logic or inline components.
- Each page imports components from `components/` and hooks from `hooks/`.
- Pages must not define new UI primitives; create them in `components/` instead.
- Mobile pages in `mobile/` are separate route trees, not responsive wrappers of desktop pages.

### Testing Requirements

- E2E tests cover login, employees, projects, and timesheets (`tests/e2e/`).
- Run `pnpm type-check` to verify route type safety.

### Common Patterns

- **Page structure**: Pages typically import a `PageHeader`, filters, and a data table/list.
- **Route params**: Pages use `useParams()` for entity IDs, `useSearchParams()` for filters.
- **Lazy loading**: Heavy pages use `React.lazy()` with `Suspense` boundaries.
- **Role guards**: Each role directory is wrapped with `ProtectedRoute` that checks user role.

## Dependencies

### Internal
- `../components/` for all UI components
- `../hooks/` for data fetching and state management
- `../lib/` for permissions and modal navigation
- `../contexts/` for auth context

### External
- React Router v6 (`useParams`, `useSearchParams`, `useNavigate`)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
