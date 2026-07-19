---
title: Employee Mobile Polish
description: >-
  Polish the employee self-service mobile experience with a scoped
  daisyUI-compatible presentation layer, unified states, and regression-safe
  attendance actions.
status: completed
priority: P2
branch: main
tags:
  - feature
  - frontend
  - ui
  - employee
  - daisyui
  - accessibility
blockedBy: []
blocks: []
created: '2026-07-19T05:15:37.598Z'
createdBy: 'ck:plan'
source: skill
---

# Employee Mobile Polish

## Overview

Refine `/employee` into one calm, pay-first mobile experience for regular, flexible, and check-in-enabled employees. Reuse the existing employee tokens, data models, hooks, routes, and Radix/shadcn interaction semantics. Because installed daisyUI classes are `ct-`-prefixed and rooted under `[data-admin-ui]`, employee UI uses semantic CSS inspired by daisyUI `card`, `stats`, `list`, `avatar`, and `btn` contracts—never daisyUI classes outside the admin root.

## Scope

- In: employee shell/header, loading/error states, wage/quota summaries, timesheet/attendance/request histories, bank destination, focus treatment, safe areas, and the fixed attendance action toolbar.
- Out: backend/API/business-rule changes, new routes or bottom navigation, dark mode, map-provider changes, background location, charts/gamification, and admin/partner redesign.
- Preserve: Vietnamese copy, employee-type routing, month/query state, mutations, GPS/geofence rules, advance calculations, deep-linked sheets, and cached history within the same authenticated principal; clear it across logout/principal change.

## Key Decisions

- Keep Tailwind 3.4 + daisyUI 4.12.24; no Tailwind 4/daisyUI 5 migration.
- Keep employee styling isolated from the completed admin daisyUI system. Existing `--employee-*` tokens remain authoritative.
- Do not add unprefixed or `ct-*` daisyUI classes to employee markup; reproduce only the selected component structure through employee-owned semantic classes and existing shadcn primitives.
- Retain Radix/shadcn for dropdowns, sheets, dialogs, focus trapping, dismissal, and trigger focus restoration.
- Treat `EmployeeAttendanceActionDock` as `role="toolbar"`, not daisyUI `dock`; it contains actions, not navigation.
- Use one 32px primary money value, 44px targets, semantic status text/icons, safe-area padding, and reduced-motion handling.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Employee Design Foundation and Shell](./phase-01-employee-design-foundation-and-shell.md) | Completed |
| 2 | [Core Employee Surface Polish](./phase-02-core-employee-surface-polish.md) | Completed |
| 3 | [Attendance Dock Accessibility and Visual QA](./phase-03-attendance-dock-accessibility-and-visual-qa.md) | Completed |

## Dependencies

- No blocking cross-plan dependency.
- The completed admin daisyUI redesign is a non-regression boundary; employee styles must not leak into admin or partner routes.
- Phase 2 depends on Phase 1. Phase 3 depends on Phases 1 and 2.

## Acceptance Criteria

- Regular, flexible, and check-in employee variants share one coherent shell while keeping their existing workflows and hierarchy.
- Loading, empty, error, offline, mutation, success, attendance, and payment states remain understandable without color alone.
- Fixed actions never cover anchored content or browser safe areas at 320px through desktop widths and landscape.
- Keyboard, 200% text zoom, reduced motion, and screen-reader flows remain usable; every mobile action is at least 44x44px.
- Focused Vitest, lint, explicit TypeScript check, build, employee Playwright, `make api-test`, and route-by-route visual QA pass.

## Visual Reference

- [Employee mobile concept](./assets/employee-mobile-concept.png) — direction only. Keep the pay hierarchy, status rows, and two-action toolbar; reject its invented avatar, new four-tab navigation, and map-first layout.

## Red Team Review

- Session 2026-07-19: 12 evidence-backed findings; 10 accepted, 2 rejected. Severity: 1 Critical, 8 High, 3 Medium.
- Accepted: cross-user cache isolation, guarded profile/attendance retries, cancellation deduplication, dynamic toolbar sizing, synthetic/redacted E2E artifacts, enforceable employee CSS strategy, working Playwright server contract, explicit shell state variants, and removal of stale viewport scope.
- Rejected: bank masking (existing self-service full-display/copy contract; separate product decision) and map-provider privacy migration (pre-existing user-opened provider contract; no eager-load change in this scope).

### Whole-Plan Consistency Sweep

- Files reread: `plan.md` and all three phase files.
- Decision deltas checked: 10. Reconciled stale references: 10. Unresolved contradictions: 0.
