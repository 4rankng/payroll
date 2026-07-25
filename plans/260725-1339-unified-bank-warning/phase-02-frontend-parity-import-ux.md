---
phase: 2
title: Frontend parity + import UX
status: completed
effort: medium
---

# Phase 2: Frontend parity + import UX

## Overview

Present one Vietnamese remediation category consistently and keep it fresh after
asynchronous BCC imports, including partial-success results.

## Implementation Steps

1. Extract a small pure helper that derives a row reason from the employee bank
   fields/status: missing bank, missing account number, missing account holder
   name, or confirmed-invalid reason.
2. Give `MissingBankDetailsSection` the stable title
   `Thông tin ngân hàng không hợp lệ` regardless of row mixture and render the
   precise reason without provider names or internal error strings.
3. Preserve the existing single expandable table, partner-visible scope,
   responsive behavior, and optional employee click behavior.
4. When BCC polling reaches either terminal status, invalidate the employee
   missing/invalid-bank query in addition to import history/timesheets.
5. Render `status='completed' && error_count>0` as partial success with created,
   skipped, and error counts plus the existing grouped actionable details.
   Keep pure success and total failure visually distinct and concise.
6. Add Vitest coverage for the reason helper/section, terminal query
   invalidation, pure success, partial success, and failure.
7. Use the UI designer review to confirm hierarchy and responsive parity; do
   not redesign unrelated timesheet actions or overwrite dirty mobile files.

## File Inventory

- `frontend/src/components/employees/MissingBankDetailsSection.tsx`
- `frontend/src/components/employees/MissingBankDetailsSection.test.tsx`
- `frontend/src/utils/bank-information-warning.ts`
- `frontend/src/utils/bank-information-warning.test.ts`
- `frontend/src/hooks/timesheet/useBCCUploadModal.ts`
- `frontend/src/hooks/timesheet/useBCCUploadModal.test.tsx`
- `frontend/src/components/timesheet/BCCUploadModal.tsx`
- `frontend/src/components/timesheet/BCCUploadModal.test.tsx`
- `frontend/src/lib/queryKeys/index.ts` only if an existing key cannot be reused

## Success Criteria

- [x] One unified section title and one list are used on desktop and mobile.
- [x] Every row explains the exact missing/invalid condition in Vietnamese.
- [x] Terminal BCC jobs refresh the list without reloading the page.
- [x] Partial-success imports never hide `error_count` or `error_detail`.
- [x] No provider name, English copy, or internal error is rendered.
- [x] Existing dirty mobile page/header/test changes remain intact.
