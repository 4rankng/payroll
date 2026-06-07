<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# events — Event Bus & Handlers

## Purpose
Implements the event-driven architecture using Redis streams for reliable event delivery. Contains the event bus (in-memory for testing, Redis for production), event handlers that process domain events for side effects (audit logging, cache invalidation, settlement processing, employee-user linking), and the event registry that maps event types to handlers.

## Key Files
| File | Description |
|------|-------------|
| `event_bus.go` | In-memory event bus — simple synchronous dispatch for testing (2.7K) |
| `redis_event_bus.go` | Redis event bus — production event publishing via Redis streams with outbox reliability (14.8K) |
| `event_bus_with_workers.go` | Event bus worker pool — consumes events from Redis streams, dispatches to handlers (8.0K) |
| `event_registry.go` | Event registry — maps event type strings to handler functions, manages handler subscriptions (12.4K) |
| `audit_event_handler.go` | Audit event handler — processes audit events, writes audit log entries (15.8K) |
| `settlement_event_handler.go` | Settlement event handler — processes payment settlement lifecycle events (14.7K) |
| `cache_invalidation_handler.go` | Cache invalidation handler — invalidates Redis cache on data changes (5.2K) |
| `employee_user_created_handler.go` | Employee-user handler — creates user accounts when employees are hired (2.6K) |
| `sequential_handler.go` | Sequential handler — wraps handlers to ensure ordered processing (1.2K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `logs/` | Event processing logs (gitignored) |

## For AI Agents

### Working In This Directory
- **Redis event bus** is the production implementation; `InMemoryEventBus` is for testing
- Events are published via `EventBus.Publish(ctx, event)` — always pass context
- The outbox pattern ensures events are not lost even if Redis is temporarily unavailable
- Event handlers implement `domain.EventHandler` interface with `Handle(ctx, event) error`
- Register new handlers in `event_registry.go` using `registry.Register(eventType, handler)`
- The `sequential_handler.go` ensures certain events (e.g., financial) are processed in order
- Worker pool (`event_bus_with_workers.go`) runs as goroutines started by the bootstrap server

### Testing Requirements
- Event handler tests co-located: `audit_event_handler_*_test.go`, `cache_invalidation_handler_test.go`
- Event registry has basic tests: `event_registry_test.go`
- Integration tests verify end-to-end event flow through Redis

### Common Patterns
```go
// Publishing an event
event := domain.NewTimesheetCreatedEvent(ctx, timesheet)
if err := eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}

// Creating a new event handler
type MyEventHandler struct {
    repo domain.MyRepository
}

func (h *MyEventHandler) Handle(ctx context.Context, event domain.Event) error {
    switch e := event.(type) {
    case domain.MyEvent:
        return h.handleMyEvent(ctx, e)
    }
    return nil
}

// Register in event_registry.go
registry.Register("my_event", myHandler)
```

## Dependencies

### Internal
- `internal/domain` — event types, `EventHandler` interface, `EventBus` interface
- `internal/infra/observability` — structured logging
- `internal/infra/persistence` — repositories for event side effects
- `internal/pkg/clock` — timestamp generation

### External
- `redis/go-redis` — Redis streams for event transport
- `google/uuid` — event ID generation

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
