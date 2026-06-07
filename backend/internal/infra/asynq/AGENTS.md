<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# asynq — Background Job Workers

## Purpose
Implements the background job processing system using asynq (Redis-backed task queue). Contains the asynq client for enqueuing tasks, a mux router for task type dispatch, task handler functions, and an HTTP monitoring server. Tasks include bulk transfer execution, disbursement processing, IPN handling, status polling, employee imports, and audit log writing.

## Key Files
| File | Description |
|------|-------------|
| `client.go` | Asynq client — enqueues tasks with retry policies, scheduling, and unique constraints (7.8K) |
| `handlers.go` | Task handlers — dispatches task types to specific worker functions (11.0K) |
| `mux.go` | Mux router — registers task type → handler mappings (3.7K) |
| `server.go` | Asynq server — configures worker pool, concurrency, queues, starts the HTTP monitoring server (1.9K) |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- Task types are defined as constants and registered in `mux.go`
- Client enqueues tasks via `client.Enqueue(task)` with configurable retry and timeout
- Handlers receive `asynq.Task` and return error (nil = success, error = retry)
- The server runs as a goroutine started by `bootstrap/server.go`
- Monitoring available via HTTP (asynq dashboard)
- Common task patterns: immediate execution, scheduled execution, unique deduplication

### Testing Requirements
- Test by enqueuing tasks and verifying side effects
- Integration tests poll for task completion rather than checking immediately
- Use `make api-test` to verify worker processing

### Common Patterns
```go
// Enqueue a task
task := asynq.NewTask(TaskType, payload)
client.Enqueue(task, asynq.MaxRetry(3), asynq.Timeout(5*time.Minute))

// Handle a task
func HandleMyTask(ctx context.Context, t *asynq.Task) error {
    var payload MyPayload
    json.Unmarshal(t.Payload(), &payload)
    // Process...
    return nil // or error for retry
}

// Register in mux.go
mux.HandleFunc(TaskType, HandleMyTask)
```

## Dependencies

### Internal
- `internal/app/workers/` — worker implementations that use this client
- `internal/infra/observability` — logging
- `internal/config` — Redis connection configuration

### External
- `hibiken/asynq` — task queue library
- `redis/go-redis` — underlying Redis connection

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
