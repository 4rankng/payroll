# ADR-004: Redis Streams for Event Delivery

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system emits domain events (employee created, timesheet approved, payment settled, etc.) that must be consumed asynchronously by multiple handlers (audit logging, cache invalidation, settlement processing, notification dispatch). The event system must be reliable — events should survive consumer crashes and be delivered at least once.

The system originally used a database-backed outbox pattern (migrations 015, 018) where events were stored in MySQL tables and a poller dispatched them to handlers.

## Decision

Migrate to **Redis Streams** for event delivery. The outbox tables were dropped in migration `062_drop_outbox_tables.up.sql`.

### Architecture

| File | Purpose |
|------|---------|
| `event_bus.go` | In-memory event bus (for testing) |
| `redis_event_bus.go` (14.8K) | Production event bus via Redis streams with reliability |
| `event_bus_with_workers.go` (8.0K) | Worker pool consuming events from Redis streams |
| `event_registry.go` (12.4K) | Maps event type strings to handler functions |
| `sequential_handler.go` | Ensures ordered processing for financial events |

### Event Types

50+ domain event types defined in `domain/events.go`, organized by domain: employee, timesheet, financial, project, system, user, advance_payment_fee, disbursement_fee, payment_cycle.

### Handlers

| Handler | Purpose |
|---------|---------|
| `audit_event_handler.go` | Writes audit log entries |
| `settlement_event_handler.go` | Processes payment settlements |
| `cache_invalidation_handler.go` | Invalidates Redis cache after commits |
| `employee_user_created_handler.go` | Provisions employee user accounts |

### Publishing Pattern

Events are published via `EventBus.Publish()` in non-blocking goroutines:
```go
event := domain.NewEmployeeCreatedEvent(ctx, employee)
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}
```

## Consequences

**Positive:**
- Redis streams provide consumer groups with automatic retry and dead-letter handling.
- Lower latency than database polling (no `SELECT ... FOR UPDATE` loop).
- Redis is already a dependency (used for caching, rate limiting, asynq) — no new infrastructure.
- In-memory event bus for tests is trivially fast.

**Negative:**
- Events are lost if Redis is not persisted (mitigated by Redis AOF persistence).
- The outbox reliability guarantee (transactional event publishing) is relaxed — events are published after the transaction commits, but a crash between commit and publish can lose an event. This is accepted because the financial critical path uses the transaction manager directly, not events.
- Redis stream consumer groups add operational complexity.

## Alternatives Considered

1. **Keep database outbox** — Rejected. Polling added latency and database load. The outbox tables were dropped in migration 062.
2. **Kafka** — Rejected. Over-provisioned for this scale. Would add a new infrastructure component.
3. **NATS JetStream** — Rejected. Redis is already a dependency; NATS would add another.
4. **Synchronous event handling** — Rejected. Would slow down request processing and couple handlers to the request lifecycle.
