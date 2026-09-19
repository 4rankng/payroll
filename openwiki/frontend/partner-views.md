---
type: frontend
title: Partner Views (Desktop + Mobile)
description: Partner role pages — project-scoped employee lists, timesheet approval, advance-payment request review, and the read-only payment history view, with the parallel mobile/partner subtree.
tags: [frontend, partner, desktop, mobile, project-scoped, parity]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-98b4fef5bee3b5a0d880f16b
    resource: repo://docs/api.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-212e82cfb34e7f4ad7a8df4d
    resource: repo://frontend/src/pages/adv-partner/UsersPage/index.tsx
  - id: openwiki-source-535745922d81573df6c01146
    resource: repo://frontend/src/pages/mobile/partner/DashboardPage/index.tsx
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Frontend: Partner Views

The partner role ("đối tác") is a project-scoped admin. A partner sees only the projects they own and only the employees and timesheets within those projects. The scoping is enforced server-side in two layers — Casbin policy for endpoint-level access and partner-scope handler checks (`isScopedPartnerRole` + `CanUserAccessProject` / `CanUserAccessEmployee`) — so the frontend never has to filter a list the server should not have returned. The advanced-partner role (`adv-partner`) extends the partner role with additional write rights.

Every partner page ships in two parallel trees: `pages/partner/` (desktop) and `pages/mobile/partner/` (mobile).

## Layout primitives

- `frontend/src/components/PartnerSidebar.tsx` — the desktop partner sidebar. Project selector lives here; selecting a project scopes the rest of the UI.
- `MobileBottomNav.tsx` — shared with admin; mobile partner uses the same bottom nav.
- `ResponsivePage.tsx` — the dual-layout wrapper.
- `ProtectedRoute.tsx` — auth/RBAC guard. Partner routes use Casbin `partner` (and `adv-partner` where applicable) policies.

The partner shell reads the active project from a context (`contexts/`) so the sidebar selector and the page-level data hooks stay in sync.

## Desktop partner pages (`frontend/src/pages/partner/`)

| Page | What it does |
|------|-------------|
| `DashboardPage` | Project dashboard scoped to the selected project: pending timesheets, recent advances, payment backlog, project cash-readiness. |
| `ProjectsPage` | The list of projects owned by this partner. Switching project re-scopes every page. |
| `EmployeesPage` | Employees assigned to projects owned by the partner. No access to employees outside the partner's scope. |
| `TimesheetsPage` | Timesheet approval queue, bulk approve/reject, BCC import. The partner approves timesheets for their projects; the admin performs final disbursement. |
| `PaymentHistoryPage` | Read-only completed bank transfers for the partner's projects, grouped by employee and fixed weekly cycle (cycle 1 = days 1-7, etc.). |

## Mobile partner subtree (`frontend/src/pages/mobile/partner/`)

| Mobile page | Mirrors |
|-------------|---------|
| `DashboardPage` | Desktop dashboard |
| `ProjectsPage` | Desktop projects |
| `EmployeesPage` | Desktop employees |
| `TimesheetsPage` | Desktop timesheet |

The mobile tree is intentionally narrower than the desktop tree because the partner does not run payroll or read the full payment history from a phone — those workflows are desktop-only.

## Advanced partner (`frontend/src/pages/adv-partner/`)

`UsersPage` is the only advanced-partner page in this tree. It lets an advanced partner manage the user accounts of employees in their projects (invite, deactivate, reset password) without going through the central admin.

## Reusable components (`frontend/src/components/partner/` and `partner-dashboard/`)

Reusable widgets scoped to the partner role: project selector, scoped employee table, the approval-queue card, the per-project dashboard tiles. Components are shared between desktop and mobile pages where the layout permits.

## Project scoping model

Server-side enforcement (per `docs/system-architecture.md` and `architecture/transport-http.md`):

1. **Casbin policy** (`backend/configs/casbin_policy.csv`) — endpoint-level allow/deny for `partner` and `adv-partner` roles.
2. **Handler-level checks** — `isScopedPartnerRole(ctx)` plus `CanUserAccessProject(ctx, projectID)` and `CanUserAccessEmployee(ctx, employeeID)` predicates from the user service. These run inside the handler before any data is returned.

Both layers are required. The Casbin policy decides which endpoints a partner can call; the handler-level checks decide which rows within that endpoint the partner can see. The partner-views frontend never re-implements this filter.

The bank-transfer history endpoint (`/api/v1/payrolls/bank-transfer-histories`) is the canonical example: the partner role gets the same response shape as the admin, but the server filters to accessible projects and returns only those rows.

## Vietnamese-only copy

All copy is Vietnamese inline. Project selector label, page titles, action buttons, and error messages are written once per page. Examples: `Duyệt chấm công`, `Yêu cầu tạm ứng`, `Lịch sử thanh toán`.

## Tests

- Page-level unit tests for each partner page (`index.test.tsx` siblings).
- Playwright runs at 1280px desktop, 390px mobile, 320px narrow, scoped to the partner role.
- Mobile/desktop parity check: the project-scoping UI surfaces must produce the same filter set.

## Relationships

- Admin role — `frontend/admin-views.md`. The admin reviews partner-approval outcomes and runs the disbursement.
- Employee role — `frontend/employee-mobile.md`. The partner approves employee timesheets and advance requests.
- Backend enforcement — `architecture/transport-http.md` and `integrations/auth-rbac.md`.
- Backend capabilities — `features/timesheet.md`, `features/flexpay.md`, `features/salary-disbursement.md`.
