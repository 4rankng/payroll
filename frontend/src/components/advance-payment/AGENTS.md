<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# advance-payment — Advance Payment / FlexPay Components

## Purpose

Components for the advance payment (FlexPay) feature that allows employees to request early payment of earned wages. Includes request forms, approval dialogs, transfer/result upload dialogs, payment history, statistics strips, fee schedules, and both admin and partner views.

## Key Files

| File | Description |
|------|-------------|
| `AdvancePaymentRequestForm.tsx` | Employee advance payment request form with amount and date selection |
| `AdvancePaymentConfirmSheet.tsx` | Confirmation sheet for approving/rejecting advance requests |
| `AdvancePaymentTransferDialog.tsx` | Transfer execution dialog for approved advances |
| `AdvancePaymentResultUploadDialog.tsx` | Upload transfer result file dialog |
| `AdvancePaymentExportDialog.tsx` | Export advance payment list dialog |
| `AdvancePaymentEmailDialog.tsx` | Email advance payment report dialog |
| `AdvancePaymentStatsStrip.tsx` | Statistics strip (total requested, approved, transferred) |
| `AdvancePaymentLimitCard.tsx` | Card showing employee's advance payment limit and usage |
| `AdvancePaymentHistoryCard.tsx` | History card showing past advance payments |
| `AdvancePaymentPageHeaderMobile.tsx` | Mobile page header for advance payment section |
| `AdvancePaymentMobileList.tsx` | Mobile-optimized advance payment list |
| `AdvancePaymentMobileEmployeeList.tsx` | Mobile employee list for advance payment selection |
| `EmployeeAdvancePaymentDetailSheet.tsx` | Detailed sheet for individual employee advance payments |
| `CheckInBulkDialog.tsx` | Bulk check-in dialog for flexible employees |
| `FileDropZone.tsx` | File drag-and-drop upload zone |
| `FileHistoryCard.tsx` | File upload history card |
| `FileHistorySheet.tsx` | File history slide-over panel |
| `FlexibleEmployeeListUploadDialog.tsx` | Upload flexible employee list dialog |
| `ImportPayrollDialog.tsx` | Import payroll data dialog |
| `StatementDialog.tsx` | Bank statement dialog |
| `StatusFilterBar.tsx` | Status filter tabs for advance payment list |
| `TransactionSummaryStrip.tsx` | Transaction amount summary |
| `TreasuryFeePanel.tsx` | Treasury fee display and configuration panel |
| `UnifiedFileCard.tsx` | Unified file card for upload history |
| `UploadAndSettleDialog.tsx` | Upload and settle advance payment dialog |
| `AdvPartnerHeroStrip.tsx` | Partner view hero strip |
| `AdvPartnerMetricsStrip.tsx` | Partner view metrics display |
| `AdvPartnerStatusOverview.tsx` | Partner status overview cards |
| `table-config.tsx` | Table column configuration for advance payment list |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `actions/` | Action button components for advance payment rows |

## For AI Agents

### Working In This Directory

- Advance payment lifecycle: `requested` -> `approved` -> `transferring` -> `completed` (or `rejected`).
- Flexible employees earn quota via check-in/out; advance amount is bounded by earned quota.
- Treasury fees are configurable via `TreasuryFeePanel`.
- File uploads (transfer results, statements) use `FileDropZone` and `FileHistorySheet`.
- Partner views (`AdvPartner*` components) show read-only summaries.

### Testing Requirements

- Run `pnpm type-check` after changes.
- Advance payment flow affects wallet balance — verify with backend integration tests.

### Common Patterns

- **Request form**: Amount + date selection with limit validation.
- **File upload**: Drop zone -> preview -> confirm -> history tracking.
- **Status filter**: Tab bar filtering by payment status.

## Dependencies

### Internal
- `../../hooks/api/useAdvancePayments.ts` for data fetching
- `../../utils/advancePaymentHelpers.ts` for calculations
- `../ui/` for base components

### External
- react-hook-form, date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
