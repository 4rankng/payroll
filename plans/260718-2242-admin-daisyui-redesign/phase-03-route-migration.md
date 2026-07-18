---
phase: 3
title: "Route Migration"
status: pending
effort: ""
priority: P1
dependencies: [1, 2]
---

# Phase 3: Route Migration

## Overview

Migrate every admin archetype and subpage to the shared system, correcting only visual hierarchy and responsive composition while keeping all workflows intact.

## Requirements

- Functional: all routes, redirects, query-backed tabs, local tabs, modal deep links, aliases, mobile-only subpages, and parameterized editors remain reachable and functional.
- Non-functional: consistent page anatomy, exception-first information order, stable desktop/tablet/mobile layout, and no decorative dashboard clutter.

## Related Code Files

- Modify: relevant files under `frontend/src/pages/admin/**`, `frontend/src/pages/mobile/admin/**`, and their feature components.
- Preserve: `frontend/src/App.tsx` route contracts and `frontend/src/components/modals/ModalRouter.tsx` deep-link behavior unless a styling hook is strictly required.

## Implementation Steps

1. Overview: reorganize dashboard presentation into attention, core totals, finance/workforce, and collapsible secondary mobile analysis without changing metrics.
2. Directories: migrate users, projects, employees, and loans to consistent header → summary → filters → data surface.
3. Payroll execution: migrate timesheet and advance-payment routes/subpages, preserving bulk actions, imports/exports, histories, and dialogs.
4. Financial control: migrate payment history, transactions/ledger, wallet, and all reconciliation/settlement/disbursement overlays.
5. Editors/settings: migrate settings tabs, email composer, notification surfaces, and pay-rate editor modes.
6. Observability: migrate system health, cron health, audit log, activity, lenders, and other routed mobile-only subpages.
7. Sweep every route for hardcoded page-local visual exceptions and normalize only those that conflict with the system.

## Success Criteria

- [ ] All 22 declared route/redirect surfaces and modal subpages inherit the system.
- [ ] All actions, fields, filters, tabs, exports, imports, dialogs, and navigation callbacks remain available.
- [ ] Mobile dashboard prioritizes attention and avoids an unstructured permanent card stack.
- [ ] High-risk financial and payroll workflows show clear amount/status context and confirmation hierarchy.
- [ ] No route-specific horizontal overflow at 390/768/1440px.

## Risk Assessment

Financial mutation paths and desktop/mobile parity are the highest risks. Styling work must not alter hooks or handlers; verify action inventory and state before/after for each high-risk route.
