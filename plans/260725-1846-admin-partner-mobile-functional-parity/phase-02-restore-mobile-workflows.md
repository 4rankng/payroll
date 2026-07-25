---
phase: 2
title: Restore Mobile Workflows
status: completed
effort: 1.5 days
---

# Phase 2: Restore Mobile Workflows

## Overview

Wire the existing desktop workflows into the active mobile pages using the same
domain hooks and shared dialog components. Keep changes local to frontend
presentation and routing.

## Implementation Steps

1. Add attendance map, approve, and reject actions to the Admin mobile advance
   payment attendance cards; reuse the desktop review hooks and dialogs.
2. Add bulk-transfer upload and batch-progress access to the Admin mobile wallet
   page using `BulkTransferUploadDialog` and `BulkTransferBatchList`.
3. Add settlement simulation to the Admin mobile transaction action sheet and
   render `SettlementSimulationDialog`.
4. Restore mobile sorting for Admin users, Admin projects, and Partner
   employees; preserve the existing API sort keys and direction semantics.
5. Add the Partner project-scoped “Bảng công” shortcut to the shared mobile
   project list without changing Admin behavior.
6. Open the same salary-history workflow from Partner timesheets at every
   viewport, including filters, server paging/infinite loading, sorting, and
   Excel export.
7. Add the production OnePay export to Admin mobile timesheets while preserving
   the existing weekly-period and explicit-project-scope contract.
8. Replace client-only Partner employee/project slicing with real server
   pagination and align project status filtering across breakpoints.
9. Preserve dashboard ledger drill-down parameters and Admin employee add
   deep links on mobile.
10. Make touched controls at least 44px and remove page-level overflow introduced
   by the repaired workflows.
11. Add focused regression tests for every restored action, route, and sorting
   contract.

## Primary Files

- `frontend/src/App.tsx`
- `frontend/src/pages/mobile/admin/AdvancePaymentsPage/index.tsx`
- `frontend/src/pages/mobile/admin/WalletPage/index.tsx`
- `frontend/src/pages/mobile/admin/TransactionsPage/index.tsx`
- `frontend/src/pages/mobile/admin/UsersPage/index.tsx`
- `frontend/src/pages/mobile/admin/ProjectsPage/index.tsx`
- `frontend/src/pages/mobile/partner/EmployeesPage/index.tsx`
- `frontend/src/pages/mobile/partner/ProjectsPage/index.tsx`
- `frontend/src/pages/mobile/partner/TimesheetsPage/index.tsx`
- `frontend/src/components/transaction/TransactionPageHeaderMobile.tsx`
- `frontend/src/components/shared/ProjectMobileList.tsx`
- Focused colocated test files under `frontend/src/`

## Success Criteria

- [x] Attendance review works from the Admin mobile attendance list.
- [x] Bulk-transfer upload and progress work from the Admin mobile wallet.
- [x] Settlement simulation works from the Admin mobile transaction page.
- [x] Admin users/projects and Partner employees expose desktop-equivalent sort.
- [x] Partner project cards open project-scoped timesheets.
- [x] Partner payment history opens the desktop-equivalent salary-history
  workflow on mobile.
- [x] Partner employee and project lists can reach every server page.
- [x] Admin mobile timesheets expose the production OnePay export.
- [x] Dashboard salary drill-downs retain their period on the ledger route.
- [x] Restored actions preserve desktop permission and data contracts.
- [x] Focused regression tests cover the repaired capabilities.
