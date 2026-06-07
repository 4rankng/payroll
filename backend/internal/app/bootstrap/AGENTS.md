<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# bootstrap — DI Container & Initialization

## Purpose
The composition root for the entire application. Contains the DI container that wires all services, repositories, handlers, and middleware. Defines HTTP routes, registers asynq background job handlers, configures cron scheduler jobs, and sets up the Gin middleware chain. Every dependency in the system is constructed and connected here.

## Key Files
| File | Description |
|------|-------------|
| `container.go` | DI container — constructs all services, repositories, handlers. Returns `Container`, `Handlers`, `Middleware` structs. (18.7K) |
| `routes.go` | HTTP route registration — maps all API endpoints to handlers with middleware. (36.6K) |
| `scheduler_jobs.go` | Cron job definitions — 10+ scheduled jobs (reconciliation, cleanup, polling). (9.4K) |
| `middleware.go` | Middleware chain setup — registers auth, RBAC, logging, security, CORS middleware. (1.7K) |
| `server.go` | Server lifecycle — starts event bus workers, asynq worker, cron scheduler. (3.9K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `infrastructure/` | Infrastructure initialization — database connection, Redis, asynq client, email providers |
| `repositories/` | Repository initialization — constructs all GORM repositories with dependencies |
| `services/` | Service initialization — constructs all application and domain services. Main `init.go` is 30.3K. |

## For AI Agents

### Working In This Directory
- **container.go** is the single source of truth for all dependency wiring
- **routes.go** defines all API endpoints — add new routes here
- **scheduler_jobs.go** defines all cron jobs — 10+ scheduled tasks
- When adding a new service: 1) create in `services/`, 2) wire in `services/init.go`, 3) add handler in `container.go`, 4) register route in `routes.go`
- `ninepay_device_id.go` in `services/` handles 9Pay device ID registration (sandbox only)
- The `AdminUserID` constant is defined here for system-level operations

### Testing Requirements
- Bootstrap is tested indirectly through integration tests
- Any new dependency must be wired correctly or integration tests will fail at startup

### Common Patterns
```go
// In container.go — handler construction
container.Handlers = Handlers{
    EmployeeHandler:   handlers.NewEmployeeHandler(services.EmployeeService, services.Audit),
    TimesheetHandler:  handlers.NewTimesheetHandler(services.TimesheetService, ...),
    // ... 20+ handlers
}

// In routes.go — route registration
api := router.Group("/api/v1")
api.GET("/employees", h.EmployeeHandler.List)
api.POST("/employees", h.EmployeeHandler.Create)
```

## Dependencies

### Internal
- All `internal/app/services/*` — application services
- All `internal/transport/http/handlers/*` — HTTP handlers
- All `internal/infra/persistence/*` — repositories
- `internal/config` — configuration
- `internal/pkg/clock` — business clock

### External
- `gin-gonic/gin` — route groups and middleware
- `hibiken/asynq` — background job mux and scheduler
- `robfig/cron` — cron job scheduling

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
