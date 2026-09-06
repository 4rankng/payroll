---
phase: 3
title: "Employee portal"
status: completed
priority: P1
effort: "1d"
dependencies: [1]
---

# Phase 3: Employee portal

## Overview
The employee-facing half: data hook, presentational bottom sheet, orchestrator card
with the three-stage dismissal state machine, mounted on both home pages.

## Requirements
- Functional: sheet auto-opens once per campaign version; dismiss → compact card;
  dismiss card → hidden; CTA taps fire the click mutation then act (`tel:` /
  `window.open` noopener) without gating navigation on the mutation.
- Non-functional: `employee-type-*` classes + `--employee-*` tokens only (emerald
  accent family); one Card nesting level; `localStorage["employee_ad_state"]` storing
  `{bannerId, version, stage}`; 60s staleTime on the query.

## Related Code Files
- Create: `frontend/src/types/api/ad-banner.types.ts`,
  `frontend/src/services/api/ad-banner.service.ts`,
  `frontend/src/hooks/api/useEmployeeAdBanner.ts`,
  `frontend/src/components/employees/EmployeeAdSheet.tsx` (+ test),
  `EmployeeAdBanner.tsx` (+ test)
- Modify: `frontend/src/lib/queryKeys.ts`,
  `frontend/src/pages/employee/EmployeePage/index.tsx` (sibling before the
  `space-y-5` div ~:178),
  `frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx` (sibling ABOVE the
  content grid ~:318 — grid children have explicit col/row placement)

## Implementation Steps
1. Types + service + hook + query keys.
2. `EmployeeAdSheet` (controlled `isOpen/onClose`, `side={isMobile ? "bottom" :
   "right"}` — pattern `NotificationSheet.tsx`).
3. `EmployeeAdBanner` orchestrator: query + localStorage state machine + auto-open.
4. Component tests (three-stage machine, version-bump re-display, CTA behavior).
5. Mount on both pages.

## Success Criteria
- [x] `pnpm vitest run` green for new tests
- [x] `npx tsc -p tsconfig.app.json --noEmit` clean (the real type gate)
- [x] Manual: seeded campaign renders on BOTH pages (spec §Employee Portal is the
      highest-risk-omission item)

## Risk Assessment
Mounting inside the flexpay grid instead of above it breaks the pinned grid layout —
the acceptance check on the flexpay page catches it visually. localStorage state must
key on `updated_at` (version), not just id, or edits never re-show — covered by the
component test.
