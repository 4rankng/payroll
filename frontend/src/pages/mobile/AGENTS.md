<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-10-28 -->

# mobile — Mobile Page Variants

## Purpose

Mobile (<768px) variants of the admin and partner pages. These are NOT
separate route trees: every surface lives at the SAME URL as its desktop
counterpart, and `App.tsx` swaps the variant through
`<ResponsivePage desktopComponent={...} mobileComponent={...} />`
(`components/ResponsivePage.tsx`, `useIsMobile` breakpoint). The mobile
component renders inside a `.mobile-page` viewport wrapper; the desktop
component renders bare.

## Key Files

| File | Description |
|------|-------------|
| `components/ResponsivePage.tsx` (repo `src/`) | Width-based variant swap used by every responsive route in `App.tsx` |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `admin/` | Mobile admin page variants (pairs with `pages/admin/`) |
| `partner/` | Mobile partner page variants (pairs with `pages/partner/`) |

### admin/

| File | Description |
|------|-------------|
| `DashboardPage/index.tsx` | Mobile admin dashboard |
| `DashboardPage/ActivityUsersPage.tsx` | Active-users drill-down (paired with desktop `pages/admin/DashboardPage/ActivityUsersPage.tsx`) |
| `ApprovalsPage/index.tsx` | Mobile approval queue |
| `EmployeesPage/index.tsx` | Mobile employee list |
| `ProjectsPage/index.tsx` | Mobile project list |
| `TimesheetPage/index.tsx` | Mobile timesheet view |
| `AdvancePaymentsPage/index.tsx` | Mobile advance payment list |
| `AdvancePaymentsPage/EmployeeListPage.tsx` | FlexPay employee-limit drill-down (paired with desktop `pages/admin/AdvancePaymentsPage/EmployeeListPage.tsx`) |
| `AuditLogPage/index.tsx` | Mobile audit log |
| `CronHealthPage/index.tsx` | Mobile cron health |
| `LedgerEntriesPage/index.tsx` | Mobile ledger (Transaction model, shared with desktop) |
| `LoansPage/index.tsx` | Mobile loan management |
| `LoansPage/LendersPage.tsx` | Lenders drill-down (paired with desktop `pages/admin/LoansPage/LendersPage.tsx`) |
| `PayrateEditPage/` | Mobile payrate edit |
| `SendNotificationPage/index.tsx` | Mobile notification sender |
| `SettingsPage/index.tsx` | Mobile settings |
| `SystemHealthPage/index.tsx` | Mobile system health |
| `UsersPage/index.tsx` | Mobile user management |
| `WalletPage/index.tsx` | Mobile wallet view |

### partner/

| File | Description |
|------|-------------|
| `DashboardPage/index.tsx` | Mobile partner dashboard |
| `EmployeesPage/index.tsx` | Mobile partner employee list |
| `ProjectsPage/index.tsx` | Mobile project list |
| `TimesheetsPage/index.tsx` | Mobile partner timesheet view |
| `TimesheetsPage/PaymentHistoryPage.tsx` | Payment-history mobile variant (paired with desktop `pages/partner/PaymentHistoryPage`) |

## For AI Agents

### Working In This Directory

- Mobile pages are **route variants**, not routes: they are wired in `App.tsx`
  via `ResponsivePage`. NEVER add `/mobile/*` URLs.
- When adding a surface, add BOTH variants and wire them as a
  `ResponsivePage` pair; drill-down pages that only exist here need a desktop
  dense-table counterpart under `pages/admin|partner` too (doctrine: ≥768px =
  dense table, <768px = card list).
- Shared mobile chrome lives in `components/shared/` (`MobilePageHeader`,
  `MobilePageShell`, `MobileSearchInput`, `MobileStatStrip`,
  `MobilePagination`, …); shared sheets/detail overlays live in
  `components/**` and may serve both widths.
- Touch targets must be at least 44px; inputs ≥16px (no iOS zoom); card rows
  wrap — never hard-clip text.
- Navigation uses the bottom tab bar instead of the sidebar; fixed bottom
  chrome needs a content clearance rail so it never covers text/links.
