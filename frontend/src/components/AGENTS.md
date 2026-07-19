<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# components — React Component Library

## Purpose

Reusable React components organized by feature domain. Contains shadcn/ui base primitives (`ui/`), feature-specific component sets (timesheet, payrates, advance-payment, etc.), role-specific views (admin, partner, partner-dashboard), shared layout components (sidebars, headers), and the modal/dialog system. Components follow the `.tsx` for UI / `.ts` for logic separation rule strictly.

## Key Files (Root Level)

| File | Description |
|------|-------------|
| `AdminSidebar.tsx` | Admin role sidebar navigation with collapsible sections |
| `PartnerSidebar.tsx` | Partner role sidebar navigation |
| `MobileBottomNav.tsx` | Mobile bottom navigation bar with role-based tabs |
| `ProtectedRoute.tsx` | Route guard — redirects unauthenticated users to login |
| `ErrorBoundary.tsx` | React error boundary with error reporting UI |
| `EnvironmentBanner.tsx` | Banner showing current environment (dev/staging/prod) |
| `ResponsivePage.tsx` | Page wrapper that adapts layout for mobile/desktop |
| `SidebarToggle.tsx` | Sidebar collapse/expand toggle button |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `ui/` | shadcn/ui base components and custom UI primitives (see `ui/AGENTS.md`) |
| `timesheet/` | Timesheet feature components (see `timesheet/AGENTS.md`) |
| `payrates/` | Payrate editor — flexible + matrix editors (see `payrates/AGENTS.md`) |
| `advance-payment/` | Advance payment / FlexPay components (see `advance-payment/AGENTS.md`) |
| `payroll/` | Payroll processing components (see `payroll/AGENTS.md`) |
| `employees/` | Employee management (see `employees/AGENTS.md`) |
| `projects/` | Project management (see `projects/AGENTS.md`) |
| `sheets/` | Sheet/slide-over panels for CRUD forms (see `sheets/AGENTS.md`) |
| `wallet/` | Wallet balance display (see `wallet/AGENTS.md`) |
| `ledger/` | Double-entry ledger components (see `ledger/AGENTS.md`) |
| `disbursement/` | Disbursement and wallet transactions (see `disbursement/AGENTS.md`) |
| `settings/` | Settings page components (see `settings/AGENTS.md`) |
| `transaction/` | Transaction history, settlement, export dialogs |
| `notifications/` | Notification provider, badge, sheet, push toggle |
| `header/` | Desktop and mobile headers, command palette, quick actions |
| `modals/` | Centralized modal components (ModalRouter, ModalRenderer, etc.) |
| `dialogs/` | Reusable dialog components (confirmation, password reset, etc.) |
| `shared/` | Shared utility components (PageHeader, FilterBar, StatsCards, etc.) |
| `dashboard/` | Admin dashboard widgets and charts |
| `admin-dashboard/` | Dashboard hero cards, KPI cards, financial charts |
| `partner-dashboard/` | Partner-specific dashboard components (employee list sheet, workforce overview card) |
| `admin/` | Admin-specific fee schedule components and partner project header |
| `partner/` | Partner-specific components |
| `project-employees/` | Add employees to project, check-in toggle, payment schedule |
| `approvals/` | Approval flow components |
| `loans/` | Loan management components |
| `lenders/` | Lender management components |
| `attendance/` | Attendance tracking service |
| `cron-health/` | Cron job monitoring cards and tables |
| `system-health/` | System health monitoring (errors, latency, users) |
| `users/` | User management table columns |
| `premium/` | Premium dashboard hero, empty states, stat strip |

## For AI Agents

### Working In This Directory

- Every component directory has its own `AGENTS.md` with specific patterns.
- **Never hard-code components in pages** — create reusable components here first.
- Use `ui/` as base primitives, build feature components on top.
- Mobile variants go in `mobile/` subdirectories within each feature folder.
- All text must be in Vietnamese.

### Testing Requirements

- Components are tested via E2E tests in `tests/` (Playwright).
- Run `pnpm type-check` and `pnpm lint` before committing.

### Common Patterns

- **Sheet pattern**: Feature CRUD forms use `Sheet` (slide-over) components in `sheets/`.
- **Modal pattern**: `modals/ModalRouter.tsx` maps URL slugs to modal components for deep-linking.
- **Mobile variants**: Components with mobile-specific views use `MobileXxx.tsx` files or `mobile/` subdirectories.
- **Barrel exports**: Feature directories export via `index.ts`.

## Dependencies

### Internal
- `../hooks/api/` for data-fetching hooks
- `../types/` for TypeScript interfaces
- `../utils/` for formatting and helper functions
- `../lib/` for query keys, modal system, permissions
- `../contexts/` for auth, app state, user preferences

### External
- shadcn/ui + Radix UI primitives, Tailwind CSS, Lucide icons, Recharts, date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
