---
phase: 1
title: Audit and Contract
status: completed
effort: 0.5 day
---

# Phase 1: Audit and Contract

## Overview

Build a route-by-route capability matrix from the actual responsive routing and
compare actions, filters, sorting, data scope, permissions, mutations, exports,
dialogs, and navigation.

## Implementation Steps

1. Inventory every Admin and Partner route selected through `DesktopOnly` and
   `MobileOnly` in `frontend/src/App.tsx`.
2. Trace each pair through its page hook, shared components, and mutations.
3. Classify each route as equivalent, mobile superset, partial, or conflicting.
4. Record confirmed repair scope and explicitly isolate the conflicting
   `/admin/ledger` route.
5. Identify responsive risks for 390px and 320px, including touch targets,
   wrapping, and horizontal overflow.

## Success Criteria

- [x] Admin route matrix is complete.
- [x] Partner route matrix is complete.
- [x] Confirmed gaps are distinguished from intentional mobile drill-downs.
- [x] `/admin/ledger` conflict is recorded without silently removing features.
- [ ] Phase status and graph-backed implementation touchpoints are finalized.
