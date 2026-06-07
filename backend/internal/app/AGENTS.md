<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# app — Application Layer

## Purpose
The application layer orchestrates use cases by coordinating between domain services, infrastructure adapters, and transport handlers. Contains the DI bootstrap container that wires all dependencies, application services that implement business workflows, DTOs for request/response shaping, temporal date utilities for payroll period calculations, background job workers, and shared utility functions.

## Key Files
| File | Description |
|------|-------------|
| `bcc_import_service.go` | BCC (bank statement) file import service — parses Excel, creates employees, imports timesheets (24.7K) |
| `bcc_import_service_test.go` | BCC import service unit tests |
| `wallet_service.go` | Wallet operations — balance sync, topup, payment processing (15.8K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `bootstrap/` | DI container, route registration, middleware setup, cron scheduler (see `bootstrap/AGENTS.md`) |
| `dto/` | Data Transfer Objects for API request/response shapes (see `dto/AGENTS.md`) |
| `lib/` | Temporal date services — payrate effective date resolution, project assignment temporal logic |
| `services/` | Application service packages — 26+ service packages (see `services/AGENTS.md`) |
| `utils/` | Shared utilities — date range helpers, payroll period calculations |
| `workers/` | asynq background job workers (audit log, bulk transfers, disbursement, imports) |

## For AI Agents

### Working In This Directory
- Application services in `services/` are the main entry point for business logic orchestration
- Services depend on domain ports (interfaces), not concrete infrastructure implementations
- The `bootstrap/` directory is the single source of truth for all dependency wiring
- `lib/` contains temporal services that resolve date-dependent configurations (payrate periods, assignment dates)
- Workers in `workers/` are asynq task handlers — registered in `bootstrap/routes.go`

### Testing Requirements
- Application services should have unit tests within their service package
- BCC import has dedicated tests in `bcc_import_service_test.go`
- Integration tests exercise services through the HTTP layer

### Common Patterns
```go
// Application service pattern
type MyService struct {
    repo         domain.MyRepository
    eventBus     domain.EventBus
    auditService *infrastructure.AuditService
    clock        clock.Clock
}

// Services are constructed in bootstrap/services/init.go
// and injected into handlers via bootstrap/container.go
```

## Dependencies

### Internal
- `internal/domain` — domain entities, ports, events
- `internal/infra` — infrastructure implementations (injected via ports)
- `internal/pkg/clock` — business clock
- `internal/config` — configuration

### External
- `hibiken/asynq` — background job definitions in workers/

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
