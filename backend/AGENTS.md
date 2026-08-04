<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# Backend — Payroll API Server

## Purpose
Go backend for the payroll management system. Follows Domain-Driven Design (DDD) with clean architecture layers: domain entities and business rules at the core, application services orchestrating use cases, infrastructure adapters for external systems, and HTTP transport for API delivery. The server handles employee management, timesheet tracking, advance payments (FlexPay), wallet operations with double-entry ledger accounting, and payment disbursement via OnePay (production) and 9Pay (sandbox).

## Architecture

```
internal/
├── adapters/              # Anti-corruption layer (ports for external contracts)
├── app/
│   ├── bootstrap/         # DI container & initialization wiring
│   │   ├── infrastructure/# Infrastructure setup (DB, Redis, asynq)
│   │   ├── repositories/  # Repository initialization
│   │   ├── services/      # Service initialization & wiring
│   │   └── container.go   # Root DI container
│   ├── dto/               # Data Transfer Objects (request/response shapes)
│   ├── lib/               # Temporal date services (payrate, assignments)
│   ├── services/          # Application services (26+ service packages)
│   ├── utils/             # Shared utilities (date ranges, payroll periods)
│   └── workers/           # asynq background job workers
├── config/                # Environment configuration (env-based)
├── domain/                # Domain layer: entities, value objects, events, ports
│   ├── ports/             # Repository & service interfaces
│   ├── services/          # Domain services (pure business logic)
│   ├── specs/             # Business specification objects
│   ├── transactions/      # Transaction management abstractions
│   └── wallet/            # Wallet aggregate (state machine, payments, IPN)
├── infra/                 # Infrastructure implementations
│   ├── asynq/             # Background job client/handlers/mux (asynq)
│   ├── disbursement/      # Payment provider adapters (9Pay, OnePay)
│   ├── email/             # Email sending (Resend + sandbox providers)
│   ├── events/            # Event bus (Redis streams + outbox pattern)
│   ├── observability/     # Logging (slog), DB metrics, Prometheus metrics
│   ├── persistence/       # GORM repositories + query builders
│   ├── storage/           # File storage abstraction
│   └── transaction/       # GORM transaction manager / unit of work
├── pkg/                   # Shared utility packages (clock, constants, retry, etc.)
├── seed/                  # Database seed data
└── transport/
    └── http/              # HTTP API layer
        ├── handlers/      # Request handlers (grouped by domain)
        ├── helpers/        # Request parsing, pagination, formatting
        ├── middleware/     # Auth, RBAC (Casbin), rate limiting, security
        ├── response/       # Error translation & response helpers
        └── validation/    # Request validation
```

**Data flow:** HTTP request → middleware → handler → app service → domain service → repository → GORM/MySQL. Events published via EventBus → Redis streams → outbox → async handlers.

## Key Files
| File | Description |
|------|-------------|
| `Makefile` | Build, test, lint, docker, deploy commands |
| `go.mod` | Module definition (`api-server`) and dependencies |
| `Dockerfile` | Multi-stage production build (amd64 only) |
| `docker-compose.dev.yml` | Local MySQL + Redis + Adminer |
| `.air.toml` | Hot-reload configuration |
| `.golangci.yml` | Linter configuration |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `cmd/` | Entry points (see `cmd/AGENTS.md`) |
| `configs/` | Casbin RBAC configuration files (see `configs/AGENTS.md`) |
| `internal/` | Core application source code (see `internal/AGENTS.md`) |
| `migrations/` | Database SQL migration files (see `migrations/AGENTS.md`) |
| `tests/` | Integration test suite (see `tests/AGENTS.md`) |
| `scripts/` | Utility scripts (currently empty) |

## For AI Agents

### Working In This Directory
- **Never run the backend directly** — it runs on hot-reload via `air`
- **Always run `make lint`** after implementation changes
- **Always run `make api-test`** to ensure no regression after new features
- Module name is `api-server` (import path: `api-server/internal/...`)
- Production server is **x86_64/amd64** — never build arm64 images
- Work on the `main` branch directly

### Code Conventions

**File naming:** `snake_case` for files, `PascalCase` for types/structs, `camelCase` for variables/functions.

**Imports:** Group as stdlib → internal → external. Sort alphabetically within groups. Separate with blank lines.

**Error handling:** Use `domain.New*Error()` for domain errors. Use `slog` for logging (not `fmt.Printf`). Never ignore errors unless explicitly with `_`.

**Context:** Always pass `context.Context` as first parameter. Use `c.Request.Context()` in handlers. Use `ctx` variable name consistently.

**Database:** Use GORM for ORM. Use `transaction` manager for multi-step operations. Use repository pattern for data access. Use domain types (not GORM models) in handlers.

**Clock:** All business time uses `clock.Now()` (Asia/Ho_Chi_Minh timezone). Never use `time.Now()` for business logic. Inject `clock.Clock` via DI in handlers/services.

**Event-driven:** Emit events via `EventBus.Publish()`. Use non-blocking goroutines for event emission. Create event handlers in `internal/infra/events/`.

**Audit logging:** Use `AuditService.LogFileImport()` / `LogFileExport()`. Emit non-blocking: `go func() { _ = auditService.LogFileExport(...) }()`. Always include: `data_type`, `record_count`, `file_name`.

**Logging:** Use `internal/infra/observability/logger.go` with `logger.Info` and above. Do not use debug level. Log at critical junctions.

### Testing Requirements
- Run `make api-test` for integration tests after any feature change
- Update tests in `backend/tests/integration/` with new test scenarios
- Use `pageSize=200` when querying timesheets in tests
- Use `findAvailableDateWithMin` for date selection in tests
- Integration test helpers: `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`, `GetServerTime`
- Pre-test DB cleanup when needed: `docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "DELETE FROM timesheets WHERE ..."`

### Common Patterns

**Creating a new handler:**
```go
// 1. Define handler struct
type MyHandler struct {
    myService    *services.MyService
    auditService *infrastructure.AuditService
}

// 2. Create constructor
func NewMyHandler(service *services.MyService, auditService *infrastructure.AuditService) *MyHandler {
    return &MyHandler{myService: service, auditService: auditService}
}

// 3. Add to bootstrap/container.go
MyHandler: handlers.NewMyHandler(services.MyService, services.Audit)

// 4. Register routes in bootstrap/routes.go
```

**Emitting audit events:**
```go
if h.auditService != nil {
    go func() { _ = h.auditService.LogFileImport(ctx, "data_type", recordCount, filename) }()
}
```

**Using EventBus:**
```go
event := domain.NewMyEvent(ctx, data)
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}
```

**Zalo ZNS password reset (employee mobile channel):**
- `internal/infra/zalo/` — stateless ZNS protocol client (phone norm, param clamp, OAuth v4, `-124` retry). Ported from tuyennhanvien.vn PHP. DB-agnostic; takes credentials via `CredentialSource` interface.
- `internal/app/services/zaloreset/` — the reset service (mobile lookup, code-gen, async ZNS dispatch, atomic password update). Role-gated to `employee`; anti-enumeration via dummy sessions.
- `internal/app/services/zaloconnect/` — admin-managed OA connection (DB-backed via `settings` table: `zalo.enabled`, `zalo.credentials`). Implements `zalo.CredentialSource`. Owns the OAuth v4 connect flow + runtime toggle (hot — no redeploy).
- Admin UI: `/admin/settings?tab=zalo` — credentials form, "Kết nối Zalo" OAuth, enable/disable toggle, token refresh. See `docs/runbooks/zalo-oa-connect.md`.
- Env vars (`ZALO_*`) are **bootstrap-only seed**; on first boot they populate the `settings` rows; thereafter the admin UI is authoritative.
- ADR: `docs/decisions/ADR-011-zalo-otp-password-reset.md`.

### Git Conventions

**Commit format:** `<type>(<scope>): <subject>`

**Types:** `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `audit`

**Examples:**
```
feat(advance-payment): implement tiered fee structure
fix(timesheet): correct date parsing in export handler
audit(employee): add import event emission
```

## Dependencies

### Internal
- `internal/domain` — core entities and business rules
- `internal/app/services` — application service orchestration
- `internal/infra/persistence` — database access layer
- `internal/infra/events` — event publishing and handling
- `internal/transport/http` — API endpoint delivery
- `internal/pkg/clock` — centralized business clock

### External
| Package | Purpose |
|---------|---------|
| `gin-gonic/gin` | HTTP framework |
| `gorm.io/gorm` | ORM (MySQL driver) |
| `redis/go-redis` | Redis client (caching, streams, queues) |
| `hibiken/asynq` | Background job processing |
| `casbin/casbin` | RBAC authorization |
| `golang-jwt/jwt` | JWT authentication |
| `resend/resend-go` | Email sending (Resend provider) |
| `go-chi/captcha` | CAPTCHA generation |

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
