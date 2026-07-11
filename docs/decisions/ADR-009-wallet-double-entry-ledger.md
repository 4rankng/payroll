# ADR-009: Wallet Double-Entry Ledger with State Machine

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system manages a wallet that holds funds for advance payments (FlexPay) and salary disbursements. Financial integrity is non-negotiable: every VND must be accounted for. The wallet processes top-ups (adding funds), payments (disbursing funds), and IPNs (Instant Payment Notifications from payment providers).

The wallet has complex state transitions: a payment can be `pending → processing → completed → failed`, with specific rules about which transitions are allowed. Invalid state transitions (e.g., completing a payment that's already failed) must be prevented structurally, not just by convention.

## Decision

Implement the wallet as a **self-contained aggregate** with:
1. **Double-entry ledger** accounting (`ledger.go` + `accounting_rules.go`).
2. **State machine** using `github.com/qmuntal/stateless`.

### Wallet Aggregate Structure

`backend/internal/domain/wallet/` (7 files):

| File | Purpose |
|------|---------|
| `wallet.go` | Aggregate root, state configuration |
| `wallet_balance.go` | Balance calculation and sync |
| `wallet_payment.go` | Payment state transitions |
| `wallet_topup.go` | Top-up processing |
| `wallet_ipn.go` | IPN handling and verification |
| `service.go` | Wallet service orchestration |
| `repository.go` | Repository interface (within the aggregate) |
| `forecast.go` | Demand forecast |

### Double-Entry Ledger

Every financial operation creates paired ledger entries (debit + credit). The ledger enforces that debits always equal credits. Accounting rules are defined in `accounting_rules.go` (7.6K) and validated with **property-based tests** (`accounting_rules_test.go` using `gopter`).

Key tables: `wallet_payments` (migration 054), `wallet_topups` (054), `wallet_ipn` (055).

### State Machine

The `qmuntal/stateless` library configures valid state transitions. Invalid transitions return an error rather than silently failing. This prevents bugs like:
- Completing a payment that was already failed.
- Processing a top-up that hasn't been verified.
- Settling a payment before the IPN arrives.

## Consequences

**Positive:**
- Financial integrity is enforced structurally — the ledger cannot be unbalanced.
- State machine prevents invalid transitions at the type level, not just via comments.
- The wallet aggregate is self-contained — its repository interface is scoped within the aggregate, not in the global `domain/ports/`.
- Property-based tests verify accounting invariants across random inputs.

**Negative:**
- The state machine configuration has a learning curve.
- Double-entry means every operation writes 2+ rows — higher write volume.
- The wallet aggregate is large and tightly coupled internally.

## Alternatives Considered

1. **Single-entry accounting** — Rejected. No way to verify financial integrity. Unacceptable for a system handling hundreds of millions of VND.
2. **Event sourcing** — Considered. Would provide a full audit trail by default, but adds significant complexity (snapshots, replay, projection). The double-entry ledger with audit logs provides sufficient integrity at lower complexity.
3. **Manual state checks (`if status == X`)** — Rejected. Error-prone and easy to bypass. The state machine makes invalid transitions impossible.
4. **External accounting system** — Rejected. Over-provisioned for this scale. The ledger is internal and sufficient.
