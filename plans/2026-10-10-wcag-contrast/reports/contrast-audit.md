# WCAG 2.2 AA — Static contrast audit

**Generated**: 2026-10-10T10:45:54.707Z
**Files scanned**: 1272
**Findings**: 40 (paired 3, inherited-surface 37; light 35, dark 5)

## Methodology

Two pairings are evaluated:

- **paired** — `text-X` and `bg-Y` land on the same source line (same JSX className or CSS rule). The exact pair is checked.
- **inherit** — `text-X` with no explicit bg on the same line. Evaluated against every declared surface (white / card / muted / background / tints) and flagged on the worst.

`dark:` variants are evaluated against the dark surface set.

## Thresholds

| Kind | Required | WCAG |
|---|---|---|
| Text (normal) | 4.5:1 | 1.4.3 AA |
| Non-text (borders, focus rings) | 3.0:1 | 1.4.11 AA |

## Worst offenders

| Ratio | Theme | Text | On | Suggested fix | Location |
|---|---|---|---|---|---|
| 1 | dark | `text-slate-900` | `dark-bg` | `n/a` | frontend/src/lib/typography.ts:212 |
| 1 | dark | `text-slate-900` | `dark-bg` | `n/a` | frontend/src/lib/typography.ts:213 |
| 1.01 | light | `text-emerald-100/70` | `muted` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:216 |
| 1.01 | light | `text-slate-100` | `muted` | `n/a` | frontend/src/lib/typography.ts:212 |
| 1.06 | light | `text-emerald-200` | `bg-emerald-500/20` | `emerald-700` | frontend/src/components/timesheet/components/useTimesheetEntryForm.ts:375 |
| 1.07 | light | `text-amber-200` | `bg-amber-500/20` | `amber-700` | frontend/src/components/timesheet/components/useTimesheetEntryForm.ts:374 |
| 1.09 | light | `text-emerald-100/70` | `white` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:216 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/advance-payment/AdvancePaymentEmailDialog.tsx:204 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/notifications/NotificationDetailModal.tsx:54 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/timesheet/BulkTransferExportDialog.tsx:188 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/timesheet/ChuyenLoDialog/index.tsx:211 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/timesheet/PayrollReportEmailDialog.tsx:173 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/timesheet/components/TimesheetEntryModal.tsx:54 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/timesheet/components/TimesheetEntryModal.tsx:55 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/transaction/BulkTransferHistoryDetailDialog.tsx:156 |
| 1.1 | light | `text-emerald-200/75` | `muted` | `n/a` | frontend/src/components/ui/dialog.tsx:116 |
| 1.1 | light | `text-slate-100` | `white` | `n/a` | frontend/src/lib/typography.ts:212 |
| 1.11 | light | `text-red-200` | `bg-red-500/20` | `red-600` | frontend/src/components/timesheet/components/useTimesheetEntryForm.ts:376 |
| 1.12 | light | `text-slate-200` | `muted` | `n/a` | frontend/src/components/advance-payment/EmployeeAdvancePaymentDetailSheet.tsx:334 |
| 1.12 | light | `text-slate-200` | `muted` | `n/a` | frontend/src/components/advance-payment/EmployeeAdvancePaymentDetailSheet.tsx:341 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/advance-payment/AdvancePaymentEmailDialog.tsx:204 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/notifications/NotificationDetailModal.tsx:54 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/timesheet/BulkTransferExportDialog.tsx:188 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/timesheet/ChuyenLoDialog/index.tsx:211 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/timesheet/PayrollReportEmailDialog.tsx:173 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/timesheet/components/TimesheetEntryModal.tsx:54 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/timesheet/components/TimesheetEntryModal.tsx:55 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/transaction/BulkTransferHistoryDetailDialog.tsx:156 |
| 1.21 | light | `text-emerald-200/75` | `white` | `n/a` | frontend/src/components/ui/dialog.tsx:116 |
| 1.22 | dark | `text-slate-900` | `dark-card` | `n/a` | frontend/src/lib/typography.ts:212 |
| 1.22 | dark | `text-slate-900` | `dark-card` | `n/a` | frontend/src/lib/typography.ts:213 |
| 1.23 | light | `text-slate-200` | `white` | `n/a` | frontend/src/components/advance-payment/EmployeeAdvancePaymentDetailSheet.tsx:334 |
| 1.23 | light | `text-slate-200` | `white` | `n/a` | frontend/src/components/advance-payment/EmployeeAdvancePaymentDetailSheet.tsx:341 |
| 1.35 | light | `text-slate-300` | `muted` | `n/a` | frontend/src/components/transaction/SettlementCard.tsx:65 |
| 1.38 | light | `text-emerald-300` | `muted` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:209 |
| 1.48 | light | `text-slate-300` | `white` | `n/a` | frontend/src/components/transaction/SettlementCard.tsx:65 |
| 1.52 | light | `text-emerald-300` | `white` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:209 |
| 1.72 | light | `text-red-300` | `muted` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:209 |
| 1.9 | light | `text-red-300` | `white` | `n/a` | frontend/src/components/modals/SettleTransactionModal.tsx:209 |
| 4.27 | dark | `text-rose-300/70` | `dark-card` | `n/a` | frontend/src/components/timesheet/BCCUploadModal.tsx:301 |

## By file

### frontend/src/lib/typography.ts (6)

- L212 `text-slate-900` on `dark-bg` = **1** (needs 4.5) → `n/a` [dark/inherit]
- L213 `text-slate-900` on `dark-bg` = **1** (needs 4.5) → `n/a` [dark/inherit]
- L212 `text-slate-100` on `muted` = **1.01** (needs 4.5) → `n/a` [light/inherit]
- L212 `text-slate-100` on `white` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L212 `text-slate-900` on `dark-card` = **1.22** (needs 4.5) → `n/a` [dark/inherit]
- L213 `text-slate-900` on `dark-card` = **1.22** (needs 4.5) → `n/a` [dark/inherit]

### frontend/src/components/modals/SettleTransactionModal.tsx (6)

- L216 `text-emerald-100/70` on `muted` = **1.01** (needs 4.5) → `n/a` [light/inherit]
- L216 `text-emerald-100/70` on `white` = **1.09** (needs 4.5) → `n/a` [light/inherit]
- L209 `text-emerald-300` on `muted` = **1.38** (needs 4.5) → `n/a` [light/inherit]
- L209 `text-emerald-300` on `white` = **1.52** (needs 4.5) → `n/a` [light/inherit]
- L209 `text-red-300` on `muted` = **1.72** (needs 4.5) → `n/a` [light/inherit]
- L209 `text-red-300` on `white` = **1.9** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/components/TimesheetEntryModal.tsx (4)

- L54 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L55 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L54 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]
- L55 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/advance-payment/EmployeeAdvancePaymentDetailSheet.tsx (4)

- L334 `text-slate-200` on `muted` = **1.12** (needs 4.5) → `n/a` [light/inherit]
- L341 `text-slate-200` on `muted` = **1.12** (needs 4.5) → `n/a` [light/inherit]
- L334 `text-slate-200` on `white` = **1.23** (needs 4.5) → `n/a` [light/inherit]
- L341 `text-slate-200` on `white` = **1.23** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/components/useTimesheetEntryForm.ts (3)

- L375 `text-emerald-200` on `bg-emerald-500/20` = **1.06** (needs 4.5) → `emerald-700` [light/paired]
- L374 `text-amber-200` on `bg-amber-500/20` = **1.07** (needs 4.5) → `amber-700` [light/paired]
- L376 `text-red-200` on `bg-red-500/20` = **1.11** (needs 4.5) → `red-600` [light/paired]

### frontend/src/components/advance-payment/AdvancePaymentEmailDialog.tsx (2)

- L204 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L204 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/notifications/NotificationDetailModal.tsx (2)

- L54 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L54 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/BulkTransferExportDialog.tsx (2)

- L188 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L188 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/ChuyenLoDialog/index.tsx (2)

- L211 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L211 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/PayrollReportEmailDialog.tsx (2)

- L173 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L173 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/transaction/BulkTransferHistoryDetailDialog.tsx (2)

- L156 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L156 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/ui/dialog.tsx (2)

- L116 `text-emerald-200/75` on `muted` = **1.1** (needs 4.5) → `n/a` [light/inherit]
- L116 `text-emerald-200/75` on `white` = **1.21** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/transaction/SettlementCard.tsx (2)

- L65 `text-slate-300` on `muted` = **1.35** (needs 4.5) → `n/a` [light/inherit]
- L65 `text-slate-300` on `white` = **1.48** (needs 4.5) → `n/a` [light/inherit]

### frontend/src/components/timesheet/BCCUploadModal.tsx (1)

- L301 `text-rose-300/70` on `dark-card` = **4.27** (needs 4.5) → `n/a` [dark/inherit]
