---
title: Admin and Partner Mobile Functional Parity
description: >-
  Restore the missing Admin and Partner mobile actions, sorting, routing, and
  responsive controls without changing backend contracts or role permissions.
status: completed
priority: P1
branch: main
tags:
  - frontend
  - mobile
  - admin
  - partner
  - parity
blockedBy: []
blocks: []
created: '2026-07-25T10:46:09.671Z'
createdBy: 'ck:plan'
source: skill
---

# Admin and Partner Mobile Functional Parity

## Overview

Audit every breakpoint-specific Admin and Partner route, restore the confirmed
mobile workflow gaps using the same hooks, permissions, mutations, and route
contracts as desktop, then verify the affected workflows at desktop and narrow
mobile widths.

The `/admin/ledger` route now renders the same double-entry ledger workflow at
every viewport. Transaction operations remain available through the separate
`/admin/transactions` route.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Audit and Contract](./phase-01-audit-and-contract.md) | Completed |
| 2 | [Restore Mobile Workflows](./phase-02-restore-mobile-workflows.md) | Completed |
| 3 | [Verification and Review](./phase-03-verification-and-review.md) | Completed |

## Dependencies

- Existing backend APIs, TanStack Query hooks, permissions, and dialog
  components remain authoritative.
- Authenticated browser QA used the local Admin and Partner sessions on
  `http://localhost:3000`.

## Acceptance Criteria

- Every breakpoint-specific Admin and Partner route in `frontend/src/App.tsx`
  has a recorded parity result.
- Confirmed mobile omissions are restored: attendance review, wallet bulk
  transfer upload/progress, settlement simulation, production OnePay export,
  project-scoped Partner timesheet navigation, mobile sorting, complete server
  pagination, and the desktop-equivalent Partner salary-history workflow.
- Mobile controls preserve desktop permissions, data scope, mutation outcomes,
  query parameters, exports, and dialog behavior.
- Affected mobile views have readable wrapping, no page-level horizontal
  overflow, and at least 44px touch targets at 390px and 320px.
- Focused tests, frontend lint/type checks, build, and the repository-required
  API regression suite pass, or unrelated baseline failures are documented.

## Out of Scope

- Backend, API, schema, accounting, and permission changes.
- Expanding either role's capability beyond the desktop contract.
- Broad visual redesign, deployment, or source-control publication.
