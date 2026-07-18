---
title: Admin daisyUI Redesign
description: >-
  Redesign every admin route and subpage with a scoped daisyUI-based ledger
  system while preserving all payroll workflows, deep links, and responsive
  behavior.
status: complete
priority: P2
branch: main
tags:
  - frontend
  - admin
  - ui
  - daisyui
  - responsive
  - accessibility
blockedBy: []
blocks: []
created: '2026-07-18T14:42:35.425Z'
createdBy: 'ck:plan'
source: skill
---

# Admin daisyUI Redesign

## Overview

Apply one production-grade admin visual system across all routes under `/admin`. The direction preserves the original TingTing information architecture, colored logo and white wordmark, and compact proportions while improving hierarchy, responsive behavior, tabular financial values, and operational clarity.

The daisyUI MCP v5 snippets are the component and theme reference. Because the application is on Tailwind CSS 3.4, implementation uses a namespaced compatible daisyUI layer instead of a risky Tailwind 4 migration. Existing Radix/shadcn behavior remains authoritative for focus-managed dialogs, sheets, selects, and URL-driven modals.

## Scope

- In: every reachable `/admin` page, responsive desktop/mobile pair, routed mobile-only subpage, settings tab, editor, dialog/sheet surface, loading/empty/error state, and the shared admin shell.
- In: admin-scoped theme tokens, sidebar/bottom navigation, page headers, summaries, filters, tables/lists, pagination, tabs, forms, status treatments, and responsive spacing.
- Out: backend/API/schema changes, business-rule changes, new product features, route renaming, modal contract changes, and redesign of partner/employee experiences.
- Preserve: role guards, query parameters, deep-linked modal/`returnTo` behavior, sorting/filtering/pagination, mutation availability, cache invalidation, keyboard flows, and Vietnamese copy.

## Acceptance Criteria

- Every declared `/admin` route, redirect target, mobile-only subpage, editor, and routed modal inherits the new admin design system without broken navigation or missing actions.
- Desktop (1440px), tablet (768px), and mobile (390px) layouts have no horizontal overflow, clipped controls, unsafe nested scrolling, or touch targets below 44px on mobile.
- The shared shell follows the daisyUI drawer/menu/dock model; recurring UI follows MCP-derived navbar, stats, table, badge, filter, tabs, modal, alert, and skeleton patterns.
- daisyUI styles are namespaced/admin-scoped and do not visually alter partner or employee routes.
- Financial values use tabular numerals; status never relies on color alone; focus states, keyboard behavior, labels, and reduced-motion handling remain accessible.
- Timesheet approvals/transfers, wallet actions, transaction settlement, pay-rate editing, settings, notification/email flows, cron toggles, and audit drill-down retain their current behavior.
- `pnpm lint`, TypeScript checks, production build, focused tests, and authenticated route-by-route visual QA pass.
- `graphify update .` completes after implementation.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Design System and Shell](./phase-01-design-system-and-shell.md) | Complete |
| 2 | [Shared Admin Primitives](./phase-02-shared-admin-primitives.md) | Complete |
| 3 | [Route Migration](./phase-03-route-migration.md) | Complete |
| 4 | [Visual QA and Verification](./phase-04-visual-qa-and-verification.md) | Complete |

## Dependencies

- No blocking plan dependency. The pending wallet bulk-transfer plan touches `/admin/wallet`; its already-present upload/disbursement UI is a behavior-preservation touchpoint, not a blocker.
- Phase 2 depends on Phase 1; Phase 3 depends on Phases 1-2; Phase 4 depends on all implementation phases.

## Key Touchpoints

- Theme/build: `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/tailwind.config.ts`, `frontend/src/styles/*`.
- Shell: `frontend/src/layouts/AdminLayout.tsx`, `frontend/src/components/AdminSidebar.tsx`, `frontend/src/components/MobileBottomNav.tsx`, `frontend/src/components/ResponsivePage.tsx`.
- Shared UI: page/mobile headers, responsive tables, cards, buttons, badges, tabs, filters, pagination, dialogs, sheets, and loading/empty states.
- Route-specific gaps: dashboard, transactions/ledger, timesheet, advance payments, settings, wallet, observability, email, and pay-rate editor components.

## Risk Controls

- Use visual wrappers and CSS/component composition; do not rewrite hooks, services, mutation handlers, or route state.
- Keep Radix primitives for interaction semantics and add daisyUI presentation around them.
- Validate representative high-risk workflows before broad route screenshots.
- Compare partner/employee screenshots or DOM tokens to confirm style isolation.
