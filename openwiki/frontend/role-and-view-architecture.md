---
type: frontend
title: Frontend Role and View Architecture
description: How the React SPA separates Admin, Partner, Employee views across desktop and mobile, with strict desktop+mobile parity and inline Vietnamese copy.
tags: [frontend, react, router, roles, responsive, pwa, vietnamese]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-e483fd3285d99d05c7b265cf
    resource: repo://frontend/AGENTS.md
  - id: openwiki-source-4aca1a83c8648805c82f5abc
    resource: repo://frontend/CLAUDE.md
  - id: openwiki-source-4691894e593eb5f2f5eb82b7
    resource: repo://frontend/src/AGENTS.md
  - id: openwiki-source-454c9bcdde0b77b35e0fc994
    resource: repo://frontend/src/App.tsx
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Frontend Role and View Architecture

The frontend is a single React 18 + Vite SPA that serves three roles — Admin, Partner, and Employee — across desktop and mobile viewports. The route table branches by role, and within each role every desktop route has a matching mobile route. Vietnamese strings are inlined in components; there is no i18n layer.

## Routing model

`frontend/src/App.tsx` is the routing hub. It lazy-loads page bundles so the initial download stays small, and uses a `ProtectedRoute` guard plus a `ResponsivePage` switch that renders the desktop or mobile variant based on viewport. Pages are grouped by role:

| Directory | Role | Notes |
|---|---|---|
| `pages/admin/` | Admin | Desktop views (Dashboard, Users, Projects, Employees, Timesheet, Transactions, Loans, AdvancePayments, Wallet, Email, Settings, AuditLog, etc.) |
| `pages/partner/` | Partner | Project-scoped (Projects, Employees, Timesheets, Dashboard, PaymentHistory) |
| `pages/adv-partner/` | AdvPartner | Subset of Admin routes scoped to advance-payment admin |
| `pages/accountant/` | Accountant | Read-only financial surface |
| `pages/employee/` | Employee | Mobile-first; routes through `EmployeeRouter` |
| `pages/mobile/admin/` | Admin (mobile) | One mobile page per Admin desktop surface |
| `pages/mobile/partner/` | Partner (mobile) | One mobile page per Partner desktop surface |
| `pages/mobile/` | top-level | Shared mobile-only utilities |
| `Login.tsx`, `OTPLogin.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx`, `ZaloResetPassword.tsx` | Auth | Top-level public routes |

`AdminLayout` and `PartnerLayout` (in `frontend/src/layouts/`) own the desktop chrome (sidebar, header, command palette). Mobile pages render their own header to preserve 44px touch targets and bottom-sheet-friendly navigation.

## Desktop and mobile parity

This is a non-negotiable project rule, recorded in `frontend/AGENTS.md`. Every feature or bug fix must cover desktop and mobile for both Admin and Partner in the same task. Tracing requirements when changing a route:

1. **Identify the render paths.** Does the route use shared responsive markup, or does it swap to dedicated desktop/mobile pages, components, headers, tables, cards, dialogs, sheets, or navigation? Update every active path.
2. **Align capabilities.** Actions, buttons, filters, states, data, validation, permissions, error handling, and workflow outcomes must remain equivalent across views. Mobile may use cards, bottom sheets, overflow menus, stacked controls — never silently omit desktop functionality. Put space-constrained actions in an accessible menu or sheet.
3. **Share hooks, services, mutations, query keys, authorization rules.** Desktop and mobile cannot drift independently when they consume the same building blocks.
4. **Add regression coverage** when desktop and mobile use separate components or callback wiring. Tests must fail if an Admin or Partner workflow disappears from either view.
5. **Verify at the right widths.** Authenticated Admin and Partner desktop at 1280px or wider; their mobile at 390px. Also test 320px when the content is dense, has long Vietnamese text, full currency values, tables, dialogs, or sheets that may overflow.

Browser QA must confirm feature/action parity, correct data and permissions, readable wrapping, no horizontal scrolling, keyboard and screen-reader semantics, visible focus, and touch targets of at least 44px.

## Vietnamese inline strings

All user-facing copy is in Vietnamese, inlined at the call site. There is no i18n layer. The codebase does not run a translation pipeline; copying strings into a dictionary is forbidden. This means new copy must be reviewed by a Vietnamese speaker before merge, and the same word should be used consistently across pages (e.g., always `Kỳ` for pay-period, never mixing `kỳ`/`Ky`/`cycle`). Examples seen in the source:

- `Tạm ứng` — advance payment screen
- `Chờ thanh toán` — pending payment KPI
- `Phiếu lương` — payslip
- `Kỳ` — pay period (used in route params, column headers, and body copy)

Currency formatting uses VND with no decimal places. Long values wrap, but never split the digits — the visual thousand separators stay intact for readability.

## Data fetching

All API calls go through TanStack Query hooks in `frontend/src/hooks/api/`. Mutations apply optimistic updates and roll back on error; query keys live in `frontend/src/lib/` so desktop and mobile share cache invalidation. The Axios client (`frontend/src/services/api/client.ts`) attaches the JWT, base-URLs against `http://localhost:8080/api/v1` in dev, and unwraps error responses into a typed shape.

Authorization is enforced server-side via Casbin (see [Auth, RBAC, and Casbin](../operations/auth-rbac-and-casbin.md)). The frontend does not duplicate the role checks for security — it uses the role only to decide which routes and chrome to render, not whether to disable an action.

## PWA and web push

`frontend/src/sw.ts` is the service worker (workbox). It registers the PWA install prompt, handles push notifications (web push for employee mobile and admin alerts), and provides offline cache strategies for static assets. Vietnamese notification copy lives in the backend event payloads; the frontend renders whatever the backend sends.

## File responsibility rule

- **`.tsx` files** — UI/UX rendering only. No business logic, no API calls, no data transformation.
- **`.ts` files** — Business logic, data manipulation, API calls, type definitions, utilities.

This separation keeps page components readable and makes the hooks/services layer reusable across desktop and mobile variants.

## Component patterns

- **shadcn/ui primitives** under `components/ui/` are the building blocks — dialogs, sheets, dropdowns, tables, forms. Page components compose these, not raw HTML.
- **Modal registry** (`src/lib/modal-registry-auto.ts`) centralizes modal routes with deep-link support; opening a modal is a navigation event, not a local state toggle.
- **Custom hooks** (`useIsMobile`, `useAuth`, `useCRUD`, `useCatalogs`, `useTripForm`) wrap repeated logic. New shared behavior goes into a hook before being copy-pasted across pages.

## Related pages

- [Quickstart](../quickstart.md) — repo map and entry points.
- [Auth, RBAC, and Casbin](../operations/auth-rbac-and-casbin.md) — server-side authorization that constrains which routes the frontend renders.
- [Testing: Integration and Playwright](../testing/integration-and-playwright.md) — Playwright E2E coverage including desktop/mobile parity assertions.
