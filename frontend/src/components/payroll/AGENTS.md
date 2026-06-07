<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# payroll — Payroll Processing Components

## Purpose

Components for payroll processing including payment history tables, filters, and detail sheets. Supports viewing completed payroll runs with their associated transactions and payment statuses.

## Key Files

| File | Description |
|------|-------------|
| `PaymentHistoryTable.tsx` | Table displaying payroll payment history |
| `PaymentHistoryFilters.tsx` | Filter bar for payment history (date range, status, employee) |
| `PaymentHistorySheet.tsx` | Detail slide-over for individual payment records |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `mobile/` | Mobile-specific payment history components |

## For AI Agents

### Working In This Directory

- Payment history is read-only — payroll runs are triggered from the backend.
- Filters use the same date range picker pattern as timesheets.

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Table + Filter + Sheet pattern**: List table with filter bar, click row opens detail sheet.

## Dependencies

### Internal
- `../../hooks/api/usePayrolls.ts` for data fetching
- `../ui/` for base components

### External
- date-fns

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
