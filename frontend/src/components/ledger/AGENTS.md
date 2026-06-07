<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# ledger — Double-Entry Ledger Components

## Purpose

Components for the double-entry ledger system including entry tables, detail sheets, filters, reversal dialogs, summary cards, and a cash flow chart. Supports viewing ledger entries with debit/credit amounts, account types, and reversal workflow for corrections.

## Key Files

| File | Description |
|------|-------------|
| `LedgerEntriesTable.tsx` | Main ledger entries table with sorting and filtering |
| `LedgerEntryDetailsSheet.tsx` | Detail slide-over for individual ledger entries |
| `LedgerFilters.tsx` | Filter bar (date range, account type, amount range, reference) |
| `LedgerPageHeader.tsx` | Page header with export and period navigation |
| `LedgerSummaryCard.tsx` | Summary card showing total debits, credits, and balance |
| `LedgerMobileList.tsx` | Mobile-optimized ledger entry list |
| `DoubleEntryModal.tsx` | Modal showing double-entry breakdown (debit/credit pairs) |
| `ReversalDialog.tsx` | Reversal dialog for correcting ledger entries |
| `CashFlowChart.tsx` | Cash flow chart showing income vs. expenses over time |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `mobile/` | Mobile-specific ledger components |

## For AI Agents

### Working In This Directory

- Ledger entries are immutable — corrections use the reversal pattern (create offsetting entry).
- `DoubleEntryModal.tsx` shows the paired debit/credit entries for a transaction.
- `CashFlowChart.tsx` uses Recharts for visualization.
- Filters support complex queries across date ranges and account types.

### Testing Requirements

- Run `pnpm type-check` after changes.
- Ledger data is backend-generated; frontend is read-only except for reversals.

### Common Patterns

- **Reversal pattern**: Select entry -> confirm reversal -> backend creates offsetting pair.
- **Summary + Table pattern**: Summary cards above the filterable entry table.

## Dependencies

### Internal
- `../../hooks/api/` for ledger data fetching
- `../../utils/formatters.ts` for currency formatting
- `../ui/` for base components

### External
- Recharts, date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
