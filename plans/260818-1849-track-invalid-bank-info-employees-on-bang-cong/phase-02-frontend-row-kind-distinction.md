---
phase: 2
title: "Frontend: distinguish invalid vs missing rows"
status: pending
priority: P2
effort: "3h"
dependencies: [1]
---

# Phase 2: Frontend — distinguish invalid vs missing rows

## Overview

Consume the new `bank_warning_kind` field and visually separate
OnePay-confirmed-invalid employees ("Sai thông tin") from missing-info
employees ("Thiếu thông tin") inside `MissingBankDetailsSection`, so the
always-visible invalid population is clearly understood as *wrong data needing
a fix*, not just incomplete data. Component is already mounted on all four
bảng công pages — no page-level changes.

## Requirements

- Functional:
  - Each row shows a kind badge: "Sai thông tin" (rose/red tone — matches the
    existing `BankAccountWarningContext` rose semantics for confirmed-invalid)
    vs "Thiếu thông tin" (current amber tone).
  - Header shows split counts when both populations exist, e.g.
    "Sai thông tin: 3 · Thiếu thông tin: 1"; single-kind lists keep the
    current single-badge layout.
  - Reason column keeps rendering `getBankInformationWarningReason` (which
    already returns the OnePay reason for invalid rows).
  - Click-to-open employee sheet behavior unchanged (`onEmployeeClick`).
  - Fallback: if `bank_warning_kind` is absent (stale backend during deploy
    window), derive client-side via the util classifier — never crash, never
    show a blank badge.
- Non-functional:
  - No new dependency; reuse `Badge`, existing table markup, amber/rose
    palette already in the codebase.
  - All text Vietnamese, inline (project convention — no i18n layer).
  - Mobile: section is shared between desktop and mobile pages already;
    verify the badge renders acceptably at narrow widths (table already has
    `min-w-[760px]` + horizontal scroll container).

## Architecture

Data flow:

```
GET /employees/missing-bank-details
  → Employee type gains optional bank_warning_kind?: 'invalid' | 'missing'
  → MissingBankDetailsSection groups rows by kind (useMemo)
  → row badge + header counts from groups
```

- Type: add `bank_warning_kind?: "invalid" | "missing"` to the Employee
  response type used by the missing-bank hooks
  (`frontend/src/types/api/employee.types.ts`).
- Util: extend `frontend/src/utils/bank-information-warning.ts` with
  `getBankInformationWarningKind(employee): 'invalid' | 'missing'` — keys off
  `bank_account_status === 'invalid'`, mirroring the backend predicate; the
  component prefers the server field and falls back to this classifier.
- Component: `MissingBankDetailsSection.tsx` — group rows, render badges,
  split counts. Keep the collapsible behavior (`isExpanded`) exactly as-is.
- Hooks: `useMissingBankDetails` / `useEmployeesWithMissingBankDetails`
  need no changes (they already pass Employee objects through).

## Related Code Files

- Modify: `frontend/src/types/api/employee.types.ts` (add optional field)
- Modify: `frontend/src/utils/bank-information-warning.ts` (add classifier)
- Modify: `frontend/src/components/employees/MissingBankDetailsSection.tsx`
  (grouping + badges + split counts)
- Modify: `frontend/src/components/employees/MissingBankDetailsSection.test.tsx`
  (new cases)
- Modify: `frontend/src/utils/bank-information-warning.test.ts` (classifier
  cases) — check it exists first; if the util has no test file, add minimal
  cases to the section test instead of creating a new file (smallest surface).
- No changes: the four timesheet pages, hooks, API config, cache
  invalidation registry (endpoint and query key unchanged).

## Implementation Steps

1. Add the optional `bank_warning_kind` field to the Employee type.
2. Add `getBankInformationWarningKind` to the util; reuse the existing
   `Pick<>` input pattern from `getBankInformationWarningReason`.
3. Update the section component:
   - `useMemo` group rows into `{ invalid: [...], missing: [...] }` using
     server field with util fallback.
   - Row: kind badge before the reason chip (rose for invalid, amber for
     missing).
   - Header: two Badges with counts when both kinds present.
4. Update tests:
   - Invalid row (server field) → rose badge, "Sai thông tin".
   - Missing row without server field → fallback classifier → amber badge.
   - Mixed list → header shows both counts; both groups ordered invalid-first
     (fix-now data outranks incomplete data).
   - Empty list → section hidden (existing behavior, guard test).
5. `pnpm lint && pnpm type-check` and run the component tests.

## Success Criteria

- [x] Invalid rows show "Sai thông tin" badge + OnePay reason; missing rows
  show "Thiếu thông tin" + specific missing-field reason
- [x] Header shows split counts for mixed lists
- [x] Fallback classifier works when `bank_warning_kind` is undefined
- [x] All existing section tests pass; new cases added
- [x] `pnpm lint && pnpm type-check` clean

## Risk Assessment

- **Deploy-order coupling**: new frontend against old backend (field absent)
  → fallback classifier covers it; old frontend against new backend → field
  ignored, today's behavior. Safe both directions.
- **Amber-box redesign scope creep**: keep the outer amber container; only
  the badges/counts change. Redesigning the whole section is out of scope.
- **Sort stability**: grouping must not reorder rows within a kind
  (server order preserved; invalid group first).
