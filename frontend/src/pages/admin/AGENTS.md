<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# admin — Admin Role Pages

## Purpose

Page-level route components for the Admin role. Admins have full access to all features: dashboard, employees, projects, timesheets, payrates, payroll, advance payments, ledger, transactions, wallet, users, loans, audit logs, system health, cron monitoring, settings, and notifications. Each page is a composition layer that wires up components with data-fetching hooks.

## Key Files

| File | Description |
|------|-------------|
| `index.tsx` | Admin layout with sidebar navigation and role guard |
| `AdvPartnerView.tsx` | Advance payment partner management view |
| `types.ts` | Admin page-level type definitions |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `DashboardPage/` | Admin dashboard with KPIs, charts, recent activities |
| `EmployeesPage/` | Employee list, create, import, detail views |
| `ProjectsPage/` | Project list, create, detail with assignments |
| `TimesheetPage/` | Timesheet entry, approval, export, import |
| `PayrateEditPage/` | Payrate editor (flexible and matrix modes) |
| `AdvancePaymentsPage/` | Advance payment management and approval |
| `TransactionsPage/` | Transaction history and settlement tracking |
| `WalletPage/` | Wallet balance and disbursement management |
| `ManualDisbursementPage/` | Shared manual disbursement form/dialog components used by wallet flows |
| `UsersPage/` | User management with role assignments |
| `LoansPage/` | Loan management and repayment tracking |
| `AuditLogPage/` | Audit log viewer with filtering |
| `SystemHealthPage/` | System health monitoring dashboard |
| `CronHealthPage/` | Cron job status and health monitoring |
| `SettingsPage/` | Application settings configuration |
| `SendNotificationPage/` | Push notification sender |
| `components/` | Shared admin page components |

### DashboardPage/

| File | Description |
|------|-------------|
| `index.tsx` | Dashboard page with widget grid |
| `DashboardMasonryCard.tsx` | Dashboard card wrapper |
| `constants.ts` | Dashboard layout constants |

### EmployeesPage/

| File | Description |
|------|-------------|
| `index.tsx` | Employee list page with filters and actions |
| `utils.ts` | Employee page utility functions |

### ProjectsPage/

| File | Description |
|------|-------------|
| `index.tsx` | Project list page with filters |
| `utils.ts` | Project page utility functions |

### TimesheetPage/

| File | Description |
|------|-------------|
| `index.tsx` | Timesheet page with entry table and approval actions |
| `utils.ts` | Timesheet page utility functions |

### PayrateEditPage/

| File | Description |
|------|-------------|
| `index.tsx` | Payrate editor page loading the correct editor mode |
| `constants.ts` | Payrate page constants |
| `types.ts` | Payrate page types |
| `utils.ts` | Payrate page utility functions |

### AdvancePaymentsPage/

| File | Description |
|------|-------------|
| `index.tsx` | Advance payment list with status filters |
| `utils.ts` | Advance payment page utilities |

### TransactionsPage/

| File | Description |
|------|-------------|
| `index.tsx` | Transaction history at `/admin/transactions` |
| `constants.ts` | Transaction page constants |
| `utils.ts` | Transaction page utility functions |

### WalletPage/

| File | Description |
|------|-------------|
| `index.tsx` | Wallet page with balance and transaction tabs |

### ManualDisbursementPage/

| File | Description |
|------|-------------|
| `ConfirmManualDisbursementDialog.tsx` | Confirmation dialog |
| `ManualDisbursementForm.tsx` | Disbursement form component |
| `RecentTransfers.tsx` | Recent transfer list |
| `ReconciliationDownloadDialog.tsx` | Reconciliation download dialog |
| `TransactionStatusPanel.tsx` | Transaction status display |
| `helpers.ts` | Disbursement helper functions |

### UsersPage/

| File | Description |
|------|-------------|
| `index.tsx` | User management page |

### LoansPage/

| File | Description |
|------|-------------|
| `index.tsx` | Loan management page |

### AuditLogPage/

| File | Description |
|------|-------------|
| `index.tsx` | Audit log list page |
| `AuditLogCard.tsx` | Audit log entry card |
| `AuditLogDetailSheet.tsx` | Audit log detail slide-over |
| `AuditLogFilters.tsx` | Audit log filter bar |
| `MetadataRenderer.tsx` | Metadata JSON renderer |
| `utils.ts` | Audit log utility functions |

### SystemHealthPage/

| File | Description |
|------|-------------|
| `index.tsx` | System health dashboard |

### CronHealthPage/

| File | Description |
|------|-------------|
| `index.tsx` | Cron health monitoring page |
| `utils.ts` | Cron health utility functions |

### SettingsPage/

| File | Description |
|------|-------------|
| `index.tsx` | Settings page |

### SendNotificationPage/

| File | Description |
|------|-------------|
| `index.tsx` | Send notification page |

## For AI Agents

### Working In This Directory

- Pages are composition layers — do not add business logic here.
- Each page corresponds to a route in `App.tsx` under `/admin/*`.
- Pages import components from `../../components/` and hooks from `../../hooks/`.
- Admin pages are wrapped with `ProtectedRoute` checking for admin role.

### Testing Requirements

- E2E tests cover dashboard, employees, projects, and timesheets.
- Run `pnpm type-check` after changes.

### Common Patterns

- **Page structure**: Import PageHeader + Filters + DataDisplay, wire up with `useQuery` hook.
- **Utils file**: Page-level utility functions (label mappers, formatters) live in `utils.ts`.

## Dependencies

### Internal
- `../../components/` for all UI components
- `../../hooks/api/` for data fetching
- `../../lib/` for permissions and navigation
- `../../utils/` for formatting helpers

### External
- React Router v6, TanStack Query

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
