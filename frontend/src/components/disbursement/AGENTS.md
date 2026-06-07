<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# disbursement — Disbursement Components

## Purpose

Components for the disbursement workflow: creating manual disbursements, wallet top-ups, reconciliation uploads, sending money, and displaying wallet transactions. The wallet transactions list is the main hub for viewing all financial activity (disbursements, top-ups, settlements).

## Key Files

| File | Description |
|------|-------------|
| `WalletTransactionsList.tsx` | Main wallet transactions list with filtering, pagination, and status display |
| `CreateManualDisbursementDialog.tsx` | Dialog for creating a manual disbursement to an employee |
| `CreateTopupDialog.tsx` | Dialog for creating a wallet top-up |
| `SendMoneyDialog.tsx` | Dialog for sending money to a bank account |
| `ReconcileUploadDialog.tsx` | Reconciliation file upload dialog |
| `WalletBalanceCard.tsx` | Wallet balance card for the disbursement page |
| `WalletPaymentsTable.tsx` | Payments sub-table within wallet transactions |
| `WalletTopupsTable.tsx` | Top-ups sub-table within wallet transactions |

## For AI Agents

### Working In This Directory

- `WalletTransactionsList.tsx` is a large component handling multiple transaction types with tab navigation.
- Disbursement flow: create -> approve -> transfer -> upload result -> complete.
- Wallet top-ups add funds to the system wallet.
- Reconciliation uploads match transfer results to disbursement records.

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Transaction list**: Tab-based (all/payments/top-ups) with shared filter bar.
- **Dialog pattern**: Form in dialog -> mutation on submit -> invalidate wallet cache.

## Dependencies

### Internal
- `../../hooks/api/` for disbursement mutations and wallet data
- `../../utils/formatters.ts` for currency formatting
- `../ui/` for base components

### External
- react-hook-form, date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
