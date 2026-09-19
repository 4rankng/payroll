---
type: integration
title: Event Bus (Redis Streams)
description: Domain events declared in events.go, per-aggregate factories in event_factory_*.go, the publish-after-commit outbox discipline, Redis Streams as the transport, and the handler set that consumes them.
tags: [integration, redis-streams, events, outbox, audit, cache-invalidation]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-a5c9de3ea4280bad6936ca7f
    resource: repo://backend/internal/domain/AGENTS.md
  - id: openwiki-source-43bb266a2ea235aca5021e3e
    resource: repo://backend/internal/domain/events.go
  - id: openwiki-source-3d0b00c530be91b05c860949
    resource: repo://backend/internal/domain/transaction_manager.go
  - id: openwiki-source-b535e2fba2a70e12d8e4a462
    resource: repo://backend/internal/infra/events/audit_event_handler.go
  - id: openwiki-source-c53833878841a0e871de883f
    resource: repo://backend/migrations/062_drop_outbox_tables.up.sql
  - id: openwiki-source-675cdb8aa785fd3b58b5747f
    resource: repo://docs/decisions/ADR-004-redis-streams-event-bus.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Integration: Event Bus (Redis Streams)

The event bus delivers cross-aggregate side effects (audit logs, cache invalidation, settlement, notification fan-out) without coupling producers and consumers. It is Redis Streams-backed per ADR-004. The original database outbox tables were dropped in migration 062; events now flow Redis-only.

## Where events live

- `internal/domain/events.go` — 50+ typed event structs.
- `internal/domain/event_factory_*.go` — per-aggregate factories that produce events from a domain context.

The factories take the current domain entity plus a context and return a fully-populated event. They are the only place that constructs an event; producers never hand-roll event literals.

| Factory | Concern |
|---------|---------|
| `event_factory_base.go` | Base factory; shared creation logic |
| `event_factory_employee.go` | Hire, update, assignment changes |
| `event_factory_timesheet.go` | Create, update, approve, bulk |
| `event_factory_financial.go` | Transactions, settlements, wallet |
| `event_factory_project.go` | CRUD, configuration changes |
| `event_factory_system.go` | Imports, exports, cron, health |
| `event_factory_user.go` | Login, logout, profile |
| `event_factory_advance_payment_fee.go` | Advance-payment fee events |
| `event_factory_disbursement_fee.go` | Disbursement fee events |
| `payment_cycle_event.go` | Salary-period transitions |

## Publishing — post-commit, never inside

The publishing rule mirrors ADR-007's cache discipline: events are persisted in the same DB transaction as the state change, then published to Redis Streams after the transaction commits. The pattern:

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.repo.Save(txCtx, entity); err != nil { return err }
    event := domain.NewEmployeeCreatedEvent(txCtx, entity)
    s.outbox.Persist(txCtx, event)   // inside the transaction
    return nil
})
// After this returns, publish via after-commit callback
domain.RegisterAfterCommit(ctx, func() {
    _ = s.eventBus.Publish(ctx, event)
})
```

If the process dies between commit and publish, the outbox row is still there and a sweeper republishes. The financial critical path does not depend on event delivery for correctness (wallet payments are reconciled by IPN).

## Runtime

`internal/infra/events/` (ADR-004):

- `redis_event_bus.go` — production bus; `Publish` writes to the Redis stream.
- `event_bus.go` — in-memory bus for tests (deterministic).
- `event_bus_with_workers.go` — consumer pool reading from Redis Streams.
- `event_registry.go` — maps event type strings to handler functions.
- `sequential_handler.go` — orders financial events so settlement cannot overtake a payment.
- `logs/` — stream consumer logs.

The consumer pool uses Redis Streams consumer groups for automatic retry and dead-letter handling. Lower latency than DB polling; no `SELECT ... FOR UPDATE` loop.

## Handlers

The handler set lives in `internal/infra/events/handlers/` and is wired in the consumer pool:

| Handler | Purpose |
|---------|---------|
| `audit_event_handler.go` | Writes audit-log rows from events. |
| `settlement_event_handler.go` | Processes wallet-payment settlements. |
| `cache_invalidation_handler.go` | Invalidates Redis cache after commits (ADR-007 driver). |
| `employee_user_created_handler.go` | Provisions employee users after hire events. |
| `cash_forecast_accuracy_handler.go` | Keeps forecast accuracy metrics current. |

Handlers are stateless. They consume an event, do their work (often delegating to an application service), and ack. Slow or failing work is retried via the Redis consumer-group machinery.

## Subscribing

The bus exposes two subscription shapes:

- `Subscribe(eventType, handler)` — typed; only events of that type are routed.
- `SubscribeAll(handler)` — global; every event reaches the handler. Used by the audit handler and the cache invalidator.

The wiring is in `bootstrap/`. New events need both the type declaration and a factory method, plus a subscriber line.

## Operational notes

- Events are lost if Redis is not persisted; mitigated by Redis AOF persistence.
- The outbox reliability guarantee is relaxed (events publish after commit). Accepted because financial correctness does not depend on event delivery.
- Stream consumer groups add operational complexity but enable automatic retry and dead-letter handling without a separate broker.

## Relationships

- Domain layer — `architecture/domain-layer.md`. Where events and factories are defined.
- Infrastructure — `architecture/infrastructure.md`. Where the bus, registry, and handlers live.
- Background jobs — `integrations/background-jobs.md`. Workers can also consume events; the two are not in conflict.
- Application services — `architecture/application-services.md`. Services register after-commit callbacks and publish.
