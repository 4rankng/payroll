---
type: frontend
title: Frontend Shared Platform Layer
description: TanStack Query data flow, hooks and services, the shadcn/ui + Tailwind design system, PWA / service worker, build and runtime, and the shared contexts and components used across all roles.
tags: [frontend, tanstack-query, shadcn, tailwind, pwa, hooks, services, contexts]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-e483fd3285d99d05c7b265cf
    resource: repo://frontend/AGENTS.md
  - id: openwiki-source-a980ed9d12d6044272290a99
    resource: repo://frontend/components.json
  - id: openwiki-source-a121f2c9d809d163dff0602e
    resource: repo://frontend/src/contexts/AuthContext.tsx
  - id: openwiki-source-222220f6e0103665a200647a
    resource: repo://frontend/src/schemas/modalSchemas.ts
  - id: openwiki-source-6d5b92cea3eba0cd0e8adcae
    resource: repo://frontend/src/services/api/client.ts
  - id: openwiki-source-94818d82bdf61ffa53d6fc57
    resource: repo://frontend/src/sw.ts
  - id: openwiki-source-378e3cf05ab0d05d335c68d5
    resource: repo://frontend/vite.config.ts
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Frontend: Shared Platform Layer

This page covers what every role's pages share: data fetching, state contexts, the UI primitive library, the PWA shell, and the build/runtime. Role-specific pages live under `frontend/src/pages/{admin,partner,employee,adv-partner,accountant}/` plus the parallel `pages/mobile/{admin,partner}/`; see `frontend/admin-views.md`, `frontend/partner-views.md`, and `frontend/employee-mobile.md`.

## Bootstrap

- `frontend/src/main.tsx` — the entry. Mounts React, sets up the router and providers (TanStack Query, auth, error boundary, PWA registration).
- `frontend/src/App.tsx` — top-level router + role guards. The `App.admin-ledger-route.test.tsx` covers the routing surface.
- `frontend/index.html` — the SPA shell with the PWA `<link>` tags.
- `frontend/components.json` — the shadcn/ui project config (style: default, baseColor: slate, cssVariables: true, TS+JSX). The path aliases (`@/components`, `@/hooks`, `@/lib`, `@/utils`) are declared here for IDE and build-tool use.
- `frontend/Dockerfile` — multi-stage production build. The output is a static SPA served by nginx in the production docker-compose.

## Data fetching — TanStack Query

`@tanstack/react-query` is the only API data-fetching surface. All requests go through hooks under `src/hooks/api/` (one hook file per resource: `useTimesheets`, `useEmployees`, `useProjects`, etc.). The conventions:

- Query keys are resource-prefixed and stable; mutations invalidate by the same keys.
- Mutations are optimistic where safe (`useApproveTimesheet`, `useRequestAdvance`). Optimistic updates are paired with rollback-on-error.
- Errors are typed; the transport translates `domain.*Error` into a discriminated union (`api/src/types.ts`).
- `useSuspenseQuery` is used where the page is wrapped in `<Suspense>`.
- `useCRUD` (under `src/hooks/`) wraps the four CRUD verbs in one place; new resources that follow the standard shape reuse it.

The Axios client (`frontend/src/services/api/client.ts`) handles JWT injection, refresh, and the `X-Request-Id` header for correlation with backend logs.

## Services and hooks

- `frontend/src/services/` — non-hook service modules. `attendance.ts` and the `api/` subdirectory carry the resource clients; they expose typed `get`, `list`, `create`, `update`, `remove` methods that hooks wrap.
- `frontend/src/hooks/` — `api/` for the TanStack Query hooks, plus utilities (`useAuth`, `useCatalogs`, `useTripForm`, `useObservedWidth`, `useIsMobile`).
- `frontend/src/schemas/` — Zod schemas mirroring the backend DTOs. Forms use `react-hook-form` + `@hookform/resolvers/zod` for end-to-end typed validation.

## Contexts

`frontend/src/contexts/`:

- `AuthContext.tsx` — current user, role, JWT refresh, login/logout.
- `UserPreferencesContext.tsx` — theme, locale-preferences, project scope.
- `AppStateContext.tsx` — global app flags.
- `MetadataContext.tsx` — runtime metadata (project list, current cycle).
- `BankAccountWarningContext.tsx` — surfaces missing-bank-detail warnings to the right surface.
- `BottomNavContext.tsx` — drives the mobile bottom nav.
- `CommandPaletteContext.tsx` — Cmd-K command palette.
- `ShortcutContext.tsx` — keyboard shortcuts.
- `index.tsx` — barrel re-export.

Contexts are intentionally narrow; cross-cutting state that is heavy should move into TanStack Query cache or a Zustand store (Zustand is used per `docs/system-architecture.md`).

## UI primitives — shadcn/ui + Tailwind

`frontend/src/components/ui/` is the shadcn/ui primitive library (Radix-based). Every component is owned in-tree (no runtime registry):

- `accordion`, `alert`, `alert-dialog`, `aspect-ratio`, `async-searchable-dropdown`, `auth-loading`, `avatar`, `badge`, `bank-selector`, `breadcrumb`, … plus domain-specific primitives (`animated-currency`, `animated-hamburger`, `ErrorBoundary`).

The design system is Navy & Gold on Tailwind CSS with CSS custom properties (`frontend/src/index.css`). Tokens are defined in `frontend/tailwind.config.ts`; the design language is documented in `frontend/CLAUDE.md`.

`frontend/src/components/shared/` is the cross-role component layer (`AccentStripCard`, `AdminPageFrame`, `ContextStrip`, `EarningsPill`, `EmptyState`, `FilterBar`, `FilterPill`, `GroupedStatCard`, `InfoPanel`, `InlineAlert`, `InlineStatStrip`, …). Pages compose these rather than building local variants.

## PWA — service worker and manifest

- `frontend/src/sw.ts` — the Workbox-driven service worker. It caches the app shell (vite-plugin-pwa), the last-read timesheet summary, and the static icons. It also handles web-push events from the browser Push API (`features/push-notifications.md`).
- `frontend/pwa-assets.config.ts` — PWA assets (icons, splash) config.
- `frontend/public/manifest.webmanifest` — the manifest served at `/manifest.webmanifest`.
- `frontend/public/` — static assets, PWA icons, generated Workbox assets.

The service worker registration is wired in `main.tsx` via `vite-plugin-pwa`. Old service workers are auto-unregistered on major version bumps.

## Build and runtime

- `frontend/vite.config.ts` — Vite 6 config with `@/` aliases matching `components.json`, plus PWA plugin.
- `frontend/tsconfig.json` — TypeScript strict mode.
- `frontend/eslint.config.js` — flat ESLint config.
- `frontend/postcss.config.js` — PostCSS for Tailwind.
- `frontend/package.json` — scripts: `pnpm dev`, `pnpm build`, `pnpm lint`, `pnpm lint:fix`, `pnpm type-check`, `pnpm test:e2e`.
- `frontend/Makefile` — mirrors the npm scripts and adds the deploy wiring.
- `frontend/Dockerfile` — multi-stage build; the production image serves the static bundle behind nginx.
- `frontend/fix-dates.mjs` — a one-off date-fix utility (not in the production bundle; listed under `.openwikiignore`).

## Mobile detection and responsive

`useIsMobile` (`frontend/src/hooks/`) is the canonical viewport hook. Combined with `ResponsivePage.tsx` it drives the dual-layout pattern. The breakpoint thresholds are 390px (typical phone) and 320px (narrow phone); both are exercised in Playwright runs.

## Vietnamese-only copy

All UI text is Vietnamese. No i18n abstraction. Strings live inline in components or in DTO tables for server-supplied labels. The convention is enforced by code review.

## Tests

- Unit tests (Vitest) live alongside the components: `frontend/src/components/ui/*test.tsx`, `frontend/src/components/shared/*test.tsx`, `frontend/src/pages/**/index.test.tsx`, `frontend/src/App.admin-ledger-route.test.tsx`.
- E2E tests (Playwright) live under `frontend/tests/` with `frontend/playwright.config.ts` configuring browsers, viewports (1280px, 390px, 320px), and the admin/partner/employee matrix.
- Testplan scenarios live in `testplan/`.

## Relationships

- Admin role — `frontend/admin-views.md`.
- Partner role — `frontend/partner-views.md`.
- Employee mobile-first — `frontend/employee-mobile.md`.
- PWA push — `features/push-notifications.md`.
- Backend API client configuration — `architecture/transport-http.md`.
