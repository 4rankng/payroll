---
type: architecture
title: Transaction Manager and Outbox
description: Atomicity across GORM writes, the after-commit hook for cache invalidation and event publication, and the event bus replacing the historical outbox table.
tags: [transactions, unit-of-work, outbox, redis-streams, event-bus, cache-invalidation]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-3d0b00c530be91b05c860949
    resource: repo://backend/internal/domain/transaction_manager.go
  - id: openwiki-source-588f14ac700cd570a4ba3ab8
    resource: repo://backend/internal/infra/transaction/gorm_transaction_manager.go
  - id: openwiki-source-675cdb8aa785fd3b58b5747f
    resource: repo://docs/decisions/ADR-004-redis-streams-event-bus.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Transaction Manager and Outbox

The payroll backend runs multi-step database operations that must be atomic — create a timesheet, post the matching ledger entries, publish a settlement event — and must be safe to roll back without leaving partial state. The unit-of-work pattern delivers that atomicity, and the after-commit hook makes downstream side effects (cache invalidation, event publication) safe. The event bus has migrated from a database outbox to Redis Streams for lower latency at the cost of a relaxed durability guarantee.

## Domain interface

`domain.TransactionManager` (`repo://backend/internal/domain/transaction_manager.go#L10-L16`) defines two methods:

- `WithTransaction(ctx, fn)` — execute the function inside a DB transaction; commit if `fn` returns nil, otherwise roll back.
- `WithTransactionResult(ctx, fn)` — same, but propagates a typed result from `fn` to the caller.

A helper `RegisterAfterCommit(ctx, fn)` (`repo://backend/internal/domain/transaction_manager.go#L32-L40`) lets callers queue work that must run only if the surrounding transaction commits. When not in a transaction, the callback runs immediately in a goroutine. `RunAfterCommitCallbacks` on `TransactionContext` is what the transaction manager invokes on successful commit (`repo://backend/internal/domain/transaction_manager.go#L43-L49`).

`WithoutTransactionContext` lets a caller shadow any inherited transaction context — important for post-commit work that should not reuse an already-committed TX.

## GORM implementation

`GormTransactionManager` (`repo://backend/internal/infra/transaction/gorm_transaction_manager.go#L24-L45`) wraps `db.WithContext(ctx).Transaction(...)`. It detects an existing transaction by checking the context, and if one is present it joins it via GORM's savepoint semantics rather than starting a new transaction. The unit test suite (`gorm_transaction_manager_test.go`) covers commit, rollback, and nested-rollback-via-savepoint cases.

GORM itself handles auto-rollback on error from the closure; the implementation is deliberately thin so the transaction manager contract is honored without surprising the caller.

## The cache-invalidation-after-commit invariant

The most important rule this pattern enforces: **cache invalidation happens after `WithTransaction` returns successfully, never inside it.** Stated in `docs/decisions/ADR-007-transaction-manager-unit-of-work.md` and reinforced in the bootstrap conventions:

```go
err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
    return s.repo.Save(txCtx, entity)
})
if err == nil {
    s.cache.Invalidate(ctx, cacheKey)  // AFTER commit — safe
}
```

Invalidating inside the transaction creates a window where a concurrent read can populate the cache with pre-commit data, then commit fails and the cache stays warm with phantom state. Code review enforces this; the lint rule will not catch it.

The same principle applies to `RegisterAfterCommit` for handlers that need a guaranteed after-commit hook without manually re-checking the error.

## Event publishing via Redis Streams

The system originally published events through a database-backed outbox table (`outbox_events`, migration 015), where a poller dispatched rows to handlers. Migrations 062+ dropped the outbox tables, and the system now publishes to **Redis Streams** as documented in `docs/decisions/ADR-004-redis-streams-event-bus.md`.

The event bus lives at `backend/internal/infra/events/`:

- `event_bus.go` — in-memory bus used in unit tests.
- `redis_event_bus.go` — production Redis Streams implementation with consumer-group retry and dead-letter handling.
- `event_bus_with_workers.go` — worker pool that drains streams and dispatches to registered handlers.
- `event_registry.go` — maps event-type strings to handler functions; supports 50+ domain event types from `domain/events.go`.
- `sequential_handler.go` — orders financial events so settlement-driven side effects run in the right order.

### Trade-off: relaxed durability

ADR-004 records that the migration to Redis Streams relaxed the strict transactional-event guarantee. A crash between MySQL commit and Redis publish will lose an event. This is accepted because the financially critical path uses the transaction manager directly (not events) for cache invalidation and ledger posts; events are best-effort for downstream concerns like audit logging, notifications, and forecast recalculation.

### Publish pattern

```go
event := domain.NewEmployeeCreatedEvent(ctx, employee)
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}
```

Publishing is non-blocking by convention; the worker pool consumes asynchronously. Failures are logged but do not abort the calling request — that would couple event-bus availability to user-facing operations.

## Event handlers

| Handler | Purpose |
|---|---|
| `cache_invalidation_handler.go` | Subscribes to mutation events and runs cache invalidation after commit. |
| `audit_event_handler.go` | Writes audit log entries from event payloads. |
| `settlement_event_handler.go` | Drives settlement bookkeeping when a payment settles. |
| `employee_user_created_handler.go` | Provisions an employee user record when an employee is created. |
| `cash_forecast_accuracy_handler.go` | Recomputes forecast accuracy snapshots. |

The cache invalidation handler is the canonical example of the after-commit invariant in event-driven form: it consumes mutation events from Redis Streams and invalidates Redis keys — running strictly after the originating transaction has already committed, because events are published post-commit.

## What does not belong inside a transaction

- Cache invalidation (use `RegisterAfterCommit` or run outside `WithTransaction`).
- HTTP calls to external providers (OnePay, 9Pay, Resend) — these are scheduled through asynq and run in worker processes.
- Long-running computations or anything that could hold a row lock for more than a few hundred milliseconds.

Keep transactions short: write the rows, post the ledger entries, register the after-commit work, and return. Move everything else to events or background workers.

## Related pages

- [Architecture Overview](./overview.md) — the four layers and the request lifecycle.
- [Double-Entry Ledger and Chart of Accounts](./double-entry-ledger.md) — ledger writes go through the transaction manager.
- [Caching and Cache Invalidation](../operations/caching-and-cache-invalidation.md) — how the after-commit invariant is operationalized.
