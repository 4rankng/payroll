---
phase: 2
title: "Convert long-list dropdowns"
status: todo
priority: P1
effort: "4h"
dependencies: [phase-01-build-searchableselect-primitive]
---

# Phase 2: Convert long-list dropdowns

## Overview

Swap every long/dynamic-list plain `Select` to `SearchableSelect`, preserving
exact selection semantics (values stringified, defaults incl. "all" kept,
disabled options respected).

## Requirements

- Instance table (classify in-flight with the rule: dynamic list or can
  exceed ~8 options → convert; static enum ≤8 → keep `Select`):

| File | Dropdown(s) | List |
|---|---|---|
| `components/timesheet/UploadHistorySheet.tsx` | project filter | projects (~37) |
| `components/timesheet/TimesheetProjectFilter.tsx` | project filter | projects |
| `components/admin-dashboard/BankTransferBreakdownCard.tsx` | project | projects |
| `components/advance-payment/CheckInSettingsPage.tsx:~250` | flexible projects | projects |
| `components/ledger/DoubleEntryModal.tsx` | account + project | accountOptions + projects |
| `components/ledger/mobile/LedgerFiltersMobile.tsx` | account + project | accountOptions + projects |
| `components/sheets/LenderDisbursementSection.tsx` | lender | lenders |
| `components/wallet/EmployeeAccountLookupDialog.tsx` | bank | banks (swift_code) |
| `components/payroll/PaymentHistoryFilters.tsx` | project + employee | projects + employees |
| `components/sheets/timesheet-entry/mobile/MobileTimesheetEntry.tsx` | project + employee | both dynamic |
| `components/sheets/timesheet-entry/components/EmployeeSelector.tsx` | employee | availableEmployees |
| `pages/mobile/admin/EmployeesPage/index.tsx` | project | projects |
| `pages/mobile/admin/LoansPage/index.tsx` | lender | lenders |
| `pages/mobile/partner/EmployeesPage/index.tsx` | project (mirror) | projects |
| `components/transaction/ExportTransactionDialog.tsx` | txn type | dynamic metadata |
| `components/transaction/mobile/TransactionFiltersMobile.tsx` | txn type | dynamic metadata |
| `components/attendance/AdminCreateCheckInDialog.tsx` | shift | shifts (dynamic) |

- In-flight classification: spot-check these on touch — if a dropdown listed
  above turns out to be a ≤8 static enum, keep `Select` and note it in the
  report; if a not-listed dropdown in a touched file holds a dynamic list,
  convert it too.
- Preserve: "Tất cả" pseudo-options, placeholders, clear/disabled states,
  controlled-value wiring, layout classes (move layout classes to the
  trigger's `triggerClassName`).
- Sheet/Dialog-context instances (DoubleEntryModal, EmployeeAccountLookupDialog,
  MobileTimesheetEntry, UploadHistorySheet) use `modal` popover (default).

## Related Code Files

- Create: none
- Modify: the ~17 files above
- Reference: `ui/searchable-select.tsx` (phase-01), `multi-searchable-dropdown.tsx`

## Implementation Steps

1. Devs work in 3 disjoint clusters (parallel):
   - **dev-A (timesheet)**: UploadHistorySheet, TimesheetProjectFilter,
     BankTransferBreakdownCard, CheckInSettingsPage, MobileTimesheetEntry,
     timesheet-entry/EmployeeSelector
   - **dev-B (finance/ledger)**: DoubleEntryModal, LedgerFiltersMobile,
     LenderDisbursementSection, EmployeeAccountLookupDialog, PaymentHistoryFilters
   - **dev-C (users/txn)**: ExportTransactionDialog, TransactionFiltersMobile,
     mobile admin EmployeesPage, mobile admin LoansPage, mobile partner
     EmployeesPage, AdminCreateCheckInDialog
2. Per instance: map `SelectTrigger/SelectValue/SelectContent/SelectItem` →
   `SearchableSelect` with `options` built from the same source array;
   stringified values; keep surrounding layout classes on the trigger.
3. Run `pnpm lint` after each file.

## Success Criteria

- [ ] Every table row converted (or explicitly reclassified with reason);
      `pnpm lint` green.
- [ ] Browser-verified: UploadHistorySheet project filter (37 options,
      diacritic-insensitive filter), DoubleEntryModal account select, one
      mobile surface (filter + select works, sheet modal traps focus
      correctly), placeholder/"Tất cả" defaults intact.
- [ ] Reclassified-to-keep list documented in the report.
