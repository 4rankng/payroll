<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# cmd — Entry Points

## Purpose
Contains the application entry points. The primary server (`api-server`) starts the HTTP API, background workers, event bus, and scheduled jobs. The `hashpw` utility generates bcrypt hashes for password setup. The `seed-temp` directory is a placeholder.

## Key Files
| File | Description |
|------|-------------|
| `api-server/main.go` | Main server entry point. Initializes config, bootstrap container, HTTP server, asynq worker, event bus, and cron scheduler. Defines `API_SERVER_VERSION` (currently `v1.10.0`). Includes health check endpoint. |
| `hashpw/main.go` | CLI utility to generate bcrypt password hashes for admin user setup |
| `seed-temp/` | Empty placeholder directory |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `api-server/` | Production API server (Gin + asynq + cron) |
| `hashpw/` | Password hash generation utility |

## For AI Agents

### Working In This Directory
- The server version is defined in `api-server/main.go` as `API_SERVER_VERSION`
- Bootstrap happens via `bootstrap.NewContainer(cfg, version)` which wires all dependencies
- Server startup order: config → observability → bootstrap container → HTTP routes → asynq mux → cron scheduler → event bus workers
- Never modify entry points unless adding a new top-level startup concern

### Testing Requirements
- Integration tests start the full server via `tests/integration/client.go`
- Server must be running for integration tests (hot-reload via `air`)

### Common Patterns
```go
// Server startup sequence in api-server/main.go
cfg := config.NewConfig()
container, _ := bootstrap.NewContainer(cfg, version)
go container.StartEventBusWorkers()  // non-blocking
go container.StartWorker()           // asynq worker
container.StartScheduler()           // cron jobs
container.Router.Run(fmt.Sprintf(":%s", cfg.AppConfig.Port))
```

## Dependencies

### Internal
- `internal/config` — environment configuration
- `internal/app/bootstrap` — DI container and all wiring
- `internal/infra/observability` — logger initialization

### External
- `gin-gonic/gin` — HTTP server
- `hibiken/asynq` — background job server

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
