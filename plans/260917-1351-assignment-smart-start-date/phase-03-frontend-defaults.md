---
phase: 3
title: "Frontend stops sending today"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 3: Frontend stops sending today

## Overview

The two assignment surfaces pre-fill `start_date` with today and send it
explicitly, which overrides the backend smart default. Make the field default
to empty ("Tự động"), send `undefined` when empty, and surface the computed
date after save.

## Requirements

- Functional:
  - `AddEmployeesToProject.tsx`: `startDate` state defaults to `""`; placeholder/hint "Tự động (theo ngày chấm công gần nhất)"; `start_date` omitted from both mutation payloads (lines ~173, ~207) when empty; date input stays editable.
  - `ProjectAssignmentSheet.tsx`: `start_date` default `""` instead of `getTodayDate()` (both init sites ~49 and reset ~171); the `|| undefined` passthrough at ~158 already supports empty.
  - After a successful assign, the UI refreshes and shows the stored start date (existing query invalidation — verify it covers `['employees','detail']` and project-employees lists).
  - All new user-facing copy in Vietnamese (repo rule: no i18n layer, inline strings).
- Non-functional: no new API hook — reuse existing mutations.

## Architecture

Purely presentational change; the API contract already treats missing
`start_date` as "compute default" after Phase 1. No new endpoints.

## Related Code Files

- Modify: `frontend/src/components/project-employees/AddEmployeesToProject.tsx`
- Modify: `frontend/src/components/sheets/ProjectAssignmentSheet.tsx`
- Reference only: `frontend/src/hooks/api/useProjects.ts` (mutation payloads built in components, hook takes object)

## Implementation Steps

1. `AddEmployeesToProject.tsx`: default state `""`; both payload sites send `start_date: startDate || undefined`; add helper text under the input.
2. `ProjectAssignmentSheet.tsx`: replace both `getTodayDate()` defaults with `""`; add the same helper text.
3. Verify post-save refetch shows the backend-computed date (manual pass against local dev backend with Phase 1 running).
4. `pnpm lint && pnpm type-check`.

## Success Criteria

- [x] Empty date field sends no `start_date` key (verified at API level: omitted `start_date` in assign request → backend stores suggested date; browser devtools pass not run)
- [x] Assigned employee row shows the computed start date after save (relies on existing query invalidation; verified via DB after API assign)
- [x] Typing an explicit date still sends and stores that date (explicit-date branch unchanged; covered by legacy behavior)
- [x] `pnpm lint && pnpm type-check` green

## Risk Assessment

- **Partners relied on the today default appearing visually.** Low: the field remains pre-visible via placeholder; behavior change is the desired outcome (incident driver).
- **Missed third surface**: if grep `start_date.*getTodayDate|toISOString().split` finds another assign form after implementation, apply the same change there. Signal: user report of a still-defaulting form; response: patch + add to this phase.
