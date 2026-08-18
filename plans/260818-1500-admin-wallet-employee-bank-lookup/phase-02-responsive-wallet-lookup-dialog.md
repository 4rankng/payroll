---
phase: 2
title: Responsive wallet lookup dialog
status: completed
priority: P1
dependencies:
  - 1
effort: medium
---

# Phase 2: Responsive wallet lookup dialog

## Overview

Build one responsive Admin wallet dialog composed from existing Dialog, Command, TanStack Query, and employee-search patterns. Wire the same capability into the separate desktop and mobile wallet pages.

## Requirements

- Vietnamese action label: `Tra cứu tài khoản`.
- Search employees by the established debounced, paginated server API.
- Offer a clearly separated manual-entry mode for custom bank/SWIFT, account
  number, and account name values without implying that they will be saved.
- Single employee selection; no lookup until explicit confirmation.
- Reset stale results when employee changes or dialog closes.
- Render stored bank information separately from the current provider result.
- Clearly show valid, invalid account, holder-name mismatch, unverified, incomplete-data, and provider-failure states.
- Mobile target height at least 44px; dialog wraps long Vietnamese names/account values without horizontal overflow at 390px and 320px.

## Related Code Files

- Create: `frontend/src/components/wallet/EmployeeAccountLookupDialog.tsx`
- Create or modify: a reusable single employee selector under `frontend/src/components/ui/`
- Modify: `frontend/src/config/api.config.ts`
- Modify: `frontend/src/services/api/manual-disbursement.service.ts`
- Modify: `frontend/src/types/api/manual-disbursement.types.ts`
- Modify: `frontend/src/hooks/api/useManualDisbursement.ts`
- Modify: `frontend/src/pages/admin/WalletPage/index.tsx`
- Modify: `frontend/src/pages/mobile/admin/WalletPage/index.tsx`
- Add: focused Vitest/component tests near the new dialog and both wallet page tests.

## Implementation Steps

1. Add typed service and mutation hook for the employee-only payload.
2. Add a single-select employee combobox using the same debounced infinite-query behavior as the established multi-selector; do not regress multi-select consumers.
3. Compose a flat dialog with selection, stored-detail rows, and a semantic result panel.
4. Add desktop header action and mobile action-grid entry using existing button conventions.
5. Test selection, pending/disabled behavior, outcome rendering, reset behavior, and action presence in both wallet paths.

## Success Criteria

- [x] Desktop and mobile Admin wallet expose equivalent lookup capability.
- [x] Employee mode never accepts client-supplied bank/account values.
- [x] Employee mode stays server-authoritative; manual mode is explicitly labeled as one-off and non-persistent.
- [x] Provider-confirmed name is visually distinguishable from the stored name.
- [x] Loading, error, empty, retry, close, and employee-change interactions are deterministic.
- [x] Keyboard focus, accessible names, and mobile touch targets meet project requirements.

## Risk Assessment

- Extending the shared multi-selector could regress payroll transfer workflows. Prefer a separate small single-selector or strictly backward-compatible composition with regression tests.
- Raw provider messages may be technical or English. Map state copy to concise Vietnamese while retaining diagnostic code only where useful.
- Account numbers are sensitive. Display only inside the Admin dialog and never add them to analytics, query keys, or logs.
