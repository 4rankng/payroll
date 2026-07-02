<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# mobile — Mobile-Specific Pages

## Purpose

Mobile-optimized page components for admin and partner roles. These are separate route trees (under `/mobile/admin/*` and `/mobile/partner/*`) that mirror the desktop pages with touch-friendly layouts, card-based data displays, and simplified navigation. Pages use the `useIsMobile` hook and mobile-specific components from `components/*/mobile/` directories.

## Key Files

| File | Description |
|------|-------------|
| `index.tsx` | Mobile layout router directing to admin or partner mobile routes |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `admin/` | Mobile admin pages (mirrors `pages/admin/`) |
| `partner/` | Mobile partner pages (mirrors `pages/partner/`) |

### admin/

| File | Description |
|------|-------------|
| `DashboardPage/index.tsx` | Mobile admin dashboard |
| `EmployeesPage/index.tsx` | Mobile employee list |
| `ProjectsPage/index.tsx` | Mobile project list |
| `TimesheetPage/index.tsx` | Mobile timesheet view |
| `AdvancePaymentsPage/index.tsx` | Mobile advance payment list |
| `AuditLogPage/index.tsx` | Mobile audit log |
| `CronHealthPage/index.tsx` | Mobile cron health |
| `LedgerEntriesPage/index.tsx` | Mobile ledger entries |
| `LoansPage/index.tsx` | Mobile loan management |
| `SendNotificationPage/index.tsx` | Mobile notification sender |
| `SettingsPage/index.tsx` | Mobile settings |
| `SystemHealthPage/index.tsx` | Mobile system health |
| `TransactionsPage/index.tsx` | Mobile transactions |
| `UsersPage/index.tsx` | Mobile user management |
| `WalletPage/index.tsx` | Mobile wallet view |

### partner/

| File | Description |
|------|-------------|
| `DashboardPage/index.tsx` | Mobile partner dashboard |
| `EmployeesPage/index.tsx` | Mobile partner employee list |
| `ProjectsPage/index.tsx` | Mobile partner project list |
| `TimesheetsPage/index.tsx` | Mobile partner timesheet view |

## For AI Agents

### Working In This Directory

- Mobile pages are **separate routes**, not responsive wrappers — they have their own URLs under `/mobile/`.
- Use mobile-specific components from `components/*/mobile/` subdirectories.
- Touch targets must be at least 44px.
- Navigation uses bottom tab bar instead of sidebar.
- Data displays use card layouts instead of tables.

### Testing Requirements

- Test on mobile viewports using Playwright's device emulation.
- Run `pnpm type-check` after changes.

### Common Patterns

- **Card-based lists**: Data displayed as vertical card lists instead of tables.
- **Bottom navigation**: `MobileBottomNav.tsx` replaces sidebar.
- **Swipe gestures**: Sheets and lists use swipe-to-close patterns.

## Dependencies

### Internal
- `../../components/` for mobile components
- `../../hooks/` for data hooks (same as desktop)
- `../../components/MobileBottomNav.tsx` for bottom navigation

### External
- React Router v6, TanStack Query

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
