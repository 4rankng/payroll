<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# services — Application Service Layer

## Purpose
Contains 26+ service packages, each encapsulating a specific business domain's use case orchestration. Services coordinate between domain entities, repositories, event bus, and infrastructure adapters to implement complete business workflows. This is where cross-cutting concerns like audit logging, event emission, and authorization checks are handled.

## Key Files
| File | Description |
|------|-------------|
| `bcc_import_service.go` | BCC bank statement file import — parses Excel, auto-creates employees, imports timesheets (24.7K) |
| `bcc_import_service_test.go` | BCC import unit tests |
| `wallet_service.go` | Wallet operations — balance synchronization, topup, payment processing, IPN handling (15.8K) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `advance_payment/` | Advance payment lifecycle — requests, fee calculation, reconciliation |
| `asset/` | Asset management — tracking and import |
| `attendance/` | Attendance tracking — check-in/out, geofence validation |
| `audit/` | Audit log service — file import/export event recording |
| `auth/` | Authentication — JWT token generation, login, password management |
| `cleanup/` | Data cleanup service — stale data removal |
| `config/` | Dynamic configuration — runtime settings management |
| `dashboard/` | Dashboard analytics — aggregated statistics for admin/employee views |
| `db_export/` | Database export — full DB dump functionality |
| `disbursement/` | Payment disbursement orchestration — provider interaction |
| `employee/` | Employee CRUD — profile management, user account creation |
| `excel/` | Excel processing — file parsing and generation utilities |
| `flex_pay/` | FlexPay service — flexible advance payment workflows |
| `infrastructure/` | Cross-cutting infrastructure services — audit logging, balance service |
| `ledger/` | Ledger service — double-entry accounting operations |
| `loan/` | Loan management — disbursement, repayment schedules |
| `notification/` | Notification service — in-app and push notification delivery |
| `orchestration/` | Workflow orchestration — multi-step business process coordination |
| `payroll/` | Payroll processing — salary calculation, payroll generation |
| `ports/` | Application-level port interfaces |
| `project/` | Project management — CRUD, employee assignment, payrate configuration |
| `push/` | Push notification service — Web Push (VAPID) for Chrome/Safari |
| `reporting/` | Report generation — payroll reports, financial summaries |
| `scheduler/` | Scheduler service — cron job management |
| `settlement/` | Settlement processing — payment settlement lifecycle |
| `timesheet/` | Timesheet management — CRUD, bulk operations, approval workflow |
| `user/` | User management — admin and employee user accounts |

## For AI Agents

### Working In This Directory
- Each service package encapsulates a bounded context with its own service struct and constructor
- Services receive dependencies via constructor injection (defined in `bootstrap/services/init.go`)
- Use domain errors (`domain.New*Error()`) for business rule violations
- Emit domain events via `EventBus.Publish()` for side effects
- Use `go func() { ... }()` for non-blocking audit event emission
- The `infrastructure/` subdirectory contains shared services used across multiple domains (audit, balance)

### Testing Requirements
- Service-level unit tests should be placed alongside the service code
- Integration tests in `tests/integration/` exercise services through HTTP

### Common Patterns
```go
// Service struct with injected dependencies
type MyService struct {
    repo         domain.MyRepository
    eventBus     domain.EventBus
    auditSvc     *infrastructure.AuditService
    clock        clock.Clock
}

// Constructor (called from bootstrap/services/init.go)
func NewMyService(repo domain.MyRepository, eventBus domain.EventBus, ...) *MyService {
    return &MyService{repo: repo, eventBus: eventBus, ...}
}

// Non-blocking audit emission
if s.auditSvc != nil {
    go func() { _ = s.auditSvc.LogFileImport(ctx, dataType, count, filename) }()
}
```

## Dependencies

### Internal
- `internal/domain` — entities, ports, events, errors
- `internal/domain/services` — domain services for pure business logic
- `internal/pkg/clock` — business clock
- `internal/app/dto` — data transfer objects
- `internal/app/lib` — temporal date services

### External
- `hibiken/asynq` — task enqueuing for background jobs
- Various Excel/PDF libraries for file generation

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
