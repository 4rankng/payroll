# Architecture Decision Records (ADRs)

Records of significant architectural decisions made in the payroll project. Each ADR captures the context, decision, consequences, and alternatives considered.

## What is an ADR?

An ADR documents **why** a technical decision was made. It prevents the team (and AI agents) from proposing changes that conflict with established architecture. When a decision is revisited, a new ADR supersedes the old one — both are kept for history.

## ADR Format

```
# ADR-XXX: Title

**Date:** YYYY-MM-DD
**Status:** Accepted | Superseded by ADR-YYY | Deprecated
**Supersedes:** ADR-ZZZ (if applicable)

## Context
Why this decision was needed. What problem were we solving?

## Decision
What we decided. Be precise.

## Consequences
What are the implications? Positive, negative, neutral.

## Alternatives Considered
What other options existed? Why were they rejected?
```

## Index

| ADR | Decision | Status |
|-----|----------|--------|
| [ADR-001](ADR-001-ddd-clean-architecture.md) | DDD with strict layer dependencies | Accepted |
| [ADR-002](ADR-002-go-gin-gorm-stack.md) | Go + Gin + GORM + MySQL 8 stack | Accepted |
| [ADR-003](ADR-003-payment-provider-abstraction.md) | Provider-agnostic disbursement interface | Accepted |
| [ADR-004](ADR-004-redis-streams-event-bus.md) | Redis streams for event delivery | Accepted |
| [ADR-005](ADR-005-asynq-background-jobs.md) | asynq for background job processing | Accepted |
| [ADR-006](ADR-006-clock-injection-pattern.md) | Clock injection for deterministic time | Accepted |
| [ADR-007](ADR-007-transaction-manager-unit-of-work.md) | Transaction manager (Unit of Work) pattern | Accepted |
| [ADR-008](ADR-008-casbin-rbac-authorization.md) | Casbin RBAC with deny-override | Accepted |
| [ADR-009](ADR-009-wallet-double-entry-ledger.md) | Wallet double-entry ledger with state machine | Accepted |
| [ADR-010](ADR-010-statistical-forecast-over-ml.md) | Statistical baseline over ML for forecasting | Accepted |

## When to Write a New ADR

Write an ADR when making a decision that:
- Introduces a new technology, framework, or pattern.
- Changes how a significant component works.
- Establishes a constraint that future development must follow.
- Resolves a trade-off where reasonable alternatives existed.

You do **not** need an ADR for:
- Adding a new endpoint following existing patterns.
- Bug fixes.
- Refactoring that doesn't change architecture.
- Routine dependency updates.

## Numbering

ADRs are numbered sequentially (001, 002, ...). Once assigned, a number is never reused. If a decision is superseded, mark it as "Superseded by ADR-XXX" and create a new ADR — do not edit the original decision.
