---
title: Admin reject unpaid timesheets by project date range
description: >-
  Admin-only, server-scoped rejection of every non-paid timesheet for one
  project and inclusive custom date range.
status: completed
priority: P1
branch: main
tags: []
blockedBy: []
blocks: []
created: '2026-07-31T11:33:02.981Z'
createdBy: 'ck:plan'
source: skill
---

# Admin reject unpaid timesheets by project date range

## Overview

Add a destructive Admin action on desktop and mobile. The server, not the paginated UI, selects matching rows. Every payment state except `paid` is eligible; rows that become paid concurrently remain protected.

## Acceptance Criteria

- Admin chooses exactly one project, inclusive start/end dates, and a required shared rejection reason.
- Matching `pending_approval` and `approved` rows with `payment_status != paid` become rejected; paid, other-project, and out-of-range rows remain unchanged.
- The write is atomic and conditional, returns the actual affected count, records the project/count in the bulk audit event, and invalidates existing timesheet/dashboard caches.
- Desktop and mobile Admin expose the same action; Partner desktop/mobile do not expose it and remain denied by the API.
- Historical payment-attempt fields remain available for audit; rejection removes the timesheet from payroll eligibility and clears active approval/edit/force-payroll state.
- If an already-started provider transfer settles after rejection, the timesheet remains rejected while the authoritative payment result is recorded; if payment settles first, rejection does not match it.
- UI remains usable at 1280px, 390px, and 320px with 44px mobile controls and no overflow.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [domain-api](./phase-01-domain-api.md) | Completed |
| 2 | [admin-ui](./phase-02-admin-ui.md) | Completed |
| 3 | [verification](./phase-03-verification.md) | Completed |

## Dependencies

- Existing timesheet repository, transaction manager, bulk rejection audit event, and cache invalidation subscribers.
- Existing Shadcn dialog/date-range/project-selection primitives; no new UI library dependency.
