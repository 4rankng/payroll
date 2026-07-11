# ADR-005: asynq for Background Job Processing

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system needs to execute long-running tasks outside the HTTP request lifecycle: bulk payment transfers, payment disbursement processing, IPN handling, status polling, employee imports, and audit log writing. These tasks need retry policies, scheduling, uniqueness constraints, and monitoring.

## Decision

Use **asynq** (`github.com/hibiken/asynq v0.26.0`), a Redis-backed task queue library.

### Architecture

| File | Purpose |
|------|---------|
| `client.go` (7.8K) | Enqueues tasks with retry policies, scheduling, unique constraints |
| `handlers.go` (11.0K) | Dispatches task types to worker functions |
| `mux.go` (3.7K) | Registers task type → handler mappings |
| `server.go` (1.9K) | Worker pool, concurrency, queues, HTTP monitoring |

All files in `backend/internal/infra/asynq/`. Workers are defined in `internal/app/workers/`.

### Task Types

- Bulk transfer execution
- Disbursement processing
- IPN (Instant Payment Notification) handling
- Status polling (OnePay)
- Employee imports
- Audit log writing

### Server Lifecycle

The asynq server runs as a goroutine started by `bootstrap/server.go`. It registers handlers via `mux.go` and processes tasks from Redis-backed queues.

## Consequences

**Positive:**
- Redis is already a dependency — no new infrastructure.
- Built-in retry with exponential backoff.
- Task uniqueness prevents duplicate processing (e.g., don't enqueue the same bulk transfer twice).
- HTTP monitoring endpoint for inspecting queue health.
- Scheduled tasks support (e.g., "run this in 5 minutes").

**Negative:**
- Redis must be persisted (AOF) for task durability.
- No built-in dead-letter queue (must be implemented manually or via max-retry).
- Task payloads are JSON-encoded — large payloads (e.g., bulk imports) may need to be stored elsewhere with a reference passed in the task.

## Alternatives Considered

1. **Go channels / goroutines** — Rejected. No persistence, no retry, no monitoring. Process restart loses all in-flight tasks.
2. **Celery (Python)** — Rejected. Would introduce Python into the stack. The backend is pure Go.
3. **River (PostgreSQL-backed)** — Rejected. Would add write load to the primary database. Redis is better suited for transient task data.
4. **Custom job table + poller** — Rejected. Reinvents what asynq provides. asynq is well-maintained and purpose-built for this use case.
