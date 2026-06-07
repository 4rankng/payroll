<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# wallet — Wallet Components

## Purpose

Wallet balance display component showing available balance, pending amounts, and total balance for the admin's system wallet.

## Key Files

| File | Description |
|------|-------------|
| `WalletBalanceCard.tsx` | Card displaying wallet balance breakdown (available, pending, total) |

## For AI Agents

### Working In This Directory

- Wallet balance card shows three amounts: available balance, pending (held for in-progress transfers), and total.
- Balance is fetched via `useQuery` with real-time updates after disbursement events.

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Balance display**: VND-formatted amounts with color coding (green for available, amber for pending).

## Dependencies

### Internal
- `../../hooks/api/` for wallet data fetching
- `../../utils/formatters.ts` for currency formatting
- `../ui/` for Card component

### External
- None beyond React

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
