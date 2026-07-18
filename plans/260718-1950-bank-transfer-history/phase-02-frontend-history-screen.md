---
phase: 2
title: Frontend history screen
status: completed
priority: P1
dependencies:
  - 1
---

# Phase 2: Frontend history screen

## Overview

Add one shared responsive bank-transfer history screen and expose it to both Admin and Partner using existing payroll history patterns.

## Requirements

- Functional: month, cycle, project, employee/search filters; completed employee-cycle list; visible bank-reference/amount lines; total amount.
- UX: Vietnamese copy, desktop table, mobile cards, loading/error/empty states, no reconciliation terminology.
- Security: frontend does not attempt to broaden or filter Partner scope.

## Related Code Files

- Modify: `frontend/src/config/api.config.ts`
- Modify: `frontend/src/types/api/payroll.types.ts`
- Modify: `frontend/src/services/api/bulk-transfer.service.ts` or the existing payroll API service boundary.
- Modify: `frontend/src/hooks/api/usePayrolls.ts`
- Create/modify shared components under `frontend/src/components/payroll/`.
- Modify Admin/Partner routes and navigation entry points.

## Implementation Steps

1. Add typed API client and TanStack Query hook.
2. Build the shared responsive history content from existing table/card/filter primitives.
3. Render each bank reference and amount directly in the record, followed by the cycle total.
4. Add Admin and Partner routes/entry points while preserving current timesheet workflows.
5. Add focused component tests for split references, empty/loading/error states, and mobile rendering.

## Success Criteria

- [x] Admin and Partner can open the history screen.
- [x] Example split payment renders two references and total `1.998.000 ₫`.
- [x] Only weekly-cycle language appears.
- [x] Desktop and mobile layouts remain usable without hiding bank references.

## Risk Assessment

Reference strings can be long; render them with tabular figures and safe wrapping. Avoid hiding the key evidence in a secondary dialog.
