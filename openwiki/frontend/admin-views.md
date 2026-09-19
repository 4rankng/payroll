---
type: frontend
title: Admin Views (Desktop + Mobile)
description: Admin role pages — employee and project CRUD, payroll/ledger/audit read views, system and cron health, settings, and the dual desktop+mobile delivery that every admin change must cover.
tags: [frontend, admin, desktop, mobile, parity, sidebar]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-966141f45871bd214897c8bf
    resource: repo://frontend/src/components/AdminSidebar.tsx
  - id: openwiki-source-347b64241a95768d317d461e
    resource: repo://frontend/src/components/MobileBottomNav.tsx
  - id: openwiki-source-58e80110e3c82917d0ff4e0c
    resource: repo://frontend/src/components/ResponsivePage.tsx
  - id: openwiki-source-eddf02e0f8b63b3cc01e7e90
    resource: repo://frontend/src/pages/admin/list-error-parity.test.tsx
  - id: openwiki-source-758947eded897b4dc3796fc2
    resource: repo://frontend/src/pages/mobile/admin/LedgerEntriesPage/index.tsx
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Frontend: Admin Views

The admin role owns the operational surface of the application: employee and project CRUD, payroll run, ledger/audit reading, settings, system and cron health, wallet top-ups, FlexPay reconciliation, and reports. Every admin page ships in two parallel trees — the desktop layout and the `pages/mobile/admin/` subtree — and every change must cover both.

## Layout primitives

- `frontend/src/components/AdminSidebar.tsx` — the desktop admin sidebar. Owns the navigation labels (Vietnamese) and the role gates.
- `frontend/src/components/MobileBottomNav.tsx` — the mobile bottom navigation shared by admin and partner.
- `frontend/src/components/ResponsivePage.tsx` — the dual-layout wrapper. Renders desktop-only, mobile-only, or shared content based on viewport. Touch targets ≥44px on mobile.
- `frontend/src/components/ProtectedRoute.tsx` — the auth/RBAC guard. Admin routes use Casbin `admin` policies.

The admin shell sets up TanStack Query providers, error boundaries (`ErrorBoundary.tsx`), the `EmailPromptGate`, the `ForceChangePasswordDialog`, and the `EnvironmentBanner`.

## Desktop admin pages (`frontend/src/pages/admin/`)

| Page directory | Capability |
|---------------|-----------|
| `DashboardPage` | Operational dashboard: pending payment backlog, recent activity, cash-readiness summary. |
| `EmployeesPage` | Employee CRUD, assignment management, payroll per employee, summary, missing bank details. |
| `ProjectsPage` | Project CRUD, payrates per project, employee assignment, activation/deactivation, pending check-in toggles. |
| `TimesheetPage` | Timesheet CRUD, bulk approve/reject, BCC import, export, payroll report, cash-readiness. |
| `AdvancePaymentsPage` | FlexPay requests, fee schedule (`AdvancePaymentFeeSchedule`), bulk import, reconciliation. |
| `PaymentHistoryPage` | Read-only bank-transfer history (per cycle, per project, per employee). |
| `ManualDisbursementPage` | Manual wallet top-up entry. |
| `WalletPage` | Wallet balance, sync balance, top-ups, payments, IPN history. |
| `TransactionsPage` | Ledger transactions and the full double-entry view. |
| `LoansPage` | Loan management (read). |
| `AuditLogPage` | Audit log search and read. |
| `SystemHealthPage` | System health, DB metrics, Prometheus. |
| `CronHealthPage` | Cron job status (per-scheduler). |
| `PayrateEditPage` | Payrate per project / per position. |
| `UsersPage` | User CRUD, activities, password reset. |
| `SettingsPage` | Runtime settings (`zalo.enabled`, fee schedule, advance hold hours, etc.). |
| `EmailPage` | Email templates and outbound log. |
| `SendNotificationPage` | Manual push / in-app notification. |
| `DisbursementFeeSchedule`, `AdvancePaymentFeeSchedule`, `WeeklyPaymentFeeSchedule` | Fee schedule editors (admin/managed). |

The directory also carries `list-error-parity.test.tsx` — the desktop/mobile list-error parity contract test that ensures the mobile twin renders the same error states.

## Mobile admin subtree (`frontend/src/pages/mobile/admin/`)

The mobile admin pages mirror the desktop capability map at 390px and 320px breakpoints:

| Mobile page | Mirrors |
|-------------|---------|
| `DashboardPage` | Desktop dashboard |
| `EmployeesPage` | Desktop employees |
| `ProjectsPage` | Desktop projects |
| `TimesheetPage` | Desktop timesheet |
| `AdvancePaymentsPage` | Desktop FlexPay |
| `WalletPage` | Desktop wallet |
| `LedgerEntriesPage` | Desktop ledger (mobile has its own page here, since the dense table does not fit) |
| `PayrateEditPage` | Desktop payrate editor |
| `LoansPage` | Desktop loans |
| `AuditLogPage` | Desktop audit |
| `SystemHealthPage` | Desktop system health |
| `CronHealthPage` | Desktop cron health |
| `SendNotificationPage` | Desktop send notification |
| `SettingsPage` | Desktop settings |
| `UsersPage` | Desktop users |

The mobile tree uses `MobileBottomNav.tsx` for navigation and consumes the same TanStack Query hooks as the desktop tree.

## Components (`frontend/src/components/admin/` and `admin-dashboard/`)

Reusable admin widgets: dashboard cards, the approval queue, the BCC import dropzone, the audit-log filter bar, the system-health chart, and the cron-health table. They are shared between desktop and mobile pages; the layout primitives (sidebar vs bottom nav, table vs card list) are not shared because the two viewports are genuinely different experiences.

## Desktop + mobile parity rule

Every admin change must cover both viewports. This is enforced by:

- The page-list diff between `pages/admin/` and `pages/mobile/admin/` — a missing twin is a code-review red flag.
- The list-error parity test (`list-error-parity.test.tsx`).
- The viewports covered in `frontend/tests/` Playwright runs (1280px desktop, 390px mobile, 320px narrow).

When a feature genuinely does not need a mobile twin (e.g., bulk-import settings), the desktop route is gated by viewport so the mobile app does not surface a broken link. The decision is recorded in the PR description and revisited at the next mobile redesign.

## Vietnamese-only UI

All copy is Vietnamese; there is no i18n layer. Strings live inline in components or in DTO tables for server-supplied labels. Changes to copy are coordinated with the admin to land in the right file.

## Relationships

- Shared platform layer — `frontend/shared-platform.md` (TanStack Query, PWA, design system).
- Partner role — `frontend/partner-views.md`.
- Mobile-first employee flows — `frontend/employee-mobile.md`.
- Backend capabilities the admin pages exercise: `features/timesheet.md`, `features/salary-disbursement.md`, `features/wallet-ledger.md`, `features/flexpay.md`, `integrations/auth-rbac.md`, `integrations/event-bus.md`.
