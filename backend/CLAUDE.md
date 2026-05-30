# CLAUDE.md - Payroll Backend

> **Long-Term Memory for AI Agents**
> This file provides context, patterns, and commands for working with the payroll backend.

---

## 📋 Project State

### ✅ Recently Completed (2026-05-06)

1. **Audit Log Implementation (Phase 2)** - COMPLETED
   - Status: 15/15 audit events implemented (100%)
   - All file import/export operations now emit audit events
   - Key files:
     - `internal/app/services/infrastructure/audit_service.go` - Created
     - `internal/infra/persistence/asset_repository.go` - Updated to emit AssetCreatedEvent
     - `internal/transport/http/handlers/employee/import_handler.go` - Import audit
     - `internal/transport/http/handlers/employee/employee_export.go` - Export audit
     - `internal/transport/http/handlers/timesheet/timesheet_export.go` - Timesheet & payroll report export audit
     - `internal/transport/http/handlers/advance_payment/file_handler.go` - All FlexPay & advance payment audit
     - `internal/transport/http/handlers/advance_payment/export_employee_handler.go` - FlexPay export audit
     - `internal/transport/http/handlers/db_export_handler.go` - DB export audit
     - `internal/app/services/advance_payment/service.go` - Service-level audit
     - `internal/app/services/advance_payment/admin_flexpay_import.go` - Worker audit
   - Dependency injection updated in `internal/app/bootstrap/services/init.go` and `container.go`

2. **Audit Log Review**
   - Document: `AUDIT_LOG_REVIEW.md`
   - Identified 15 missing audit points
   - All now implemented

3. **Fee Tier Implementation Plan**
   - Document: `FEE_TIER_IMPLEMENTATION_PLAN.md`
   - Plan to implement tiered fee structure (2% → 1.3% for >3.5M VND)
   - NOT YET IMPLEMENTED - Ready for development

### 🎯 Next Steps

1. **Test Audit Implementation**
   - Run integration tests for all import/export operations
   - Verify audit logs in staging environment
   - Query audit logs to confirm all 15 events are working

2. **Implement Fee Tier Structure** (When ready)
   - Follow plan in `FEE_TIER_IMPLEMENTATION_PLAN.md`
   - Update `internal/app/services/config/settings_config.go`
   - Update `internal/app/services/advance_payment/calculator.go`
   - Create migration for new settings

3. **Code Quality**
   - Add unit tests for new audit event emission
   - Increase test coverage from current level to >70%

---

## 🎨 Style Guide

### Architecture Patterns

**Clean Architecture - DDD (Domain-Driven Design)**

```
internal/
├── app/
│   ├── bootstrap/          # Application initialization
│   │   ├── infrastructure/ # Infrastructure setup
│   │   ├── repositories/   # Repository initialization
│   │   ├── services/       # Service initialization
│   │   └── container.go    # DI container
│   ├── services/           # Application services (business logic)
│   │   ├── advance_payment/
│   │   ├── employee/
│   │   ├── timesheet/
│   │   └── infrastructure/ # Cross-cutting services
│   └── dto/                # Data Transfer Objects
├── domain/                 # Domain models and interfaces
│   ├── ports/              # Service ports (interfaces)
│   ├── services/           # Domain services
│   ├── events.go           # Domain events
│   └── *.go                # Domain entities
├── infra/                  # Infrastructure implementations
│   ├── events/             # Event bus & handlers
│   ├── persistence/        # Database repositories
│   ├── storage/            # File storage
│   └── asynq/              # Background jobs
└── transport/
    └── http/
        ├── handlers/       # HTTP handlers
        ├── middleware/     # HTTP middleware
        └── response/       # Response utilities
```

### Code Conventions

**File Naming:**
- Use `snake_case` for file names
- Use `PascalCase` for types/structs
- Use `camelCase` for variables and functions

**Imports:**
- Group imports: stdlib → internal → external
- Use blank lines between groups
- Sort imports alphabetically within groups

**Error Handling:**
- Always use `domain.New*Error()` for domain errors
- Use `slog` for logging (not `fmt.Printf`)
- Never ignore errors unless explicitly with `_`

**Context:**
- Always pass `context.Context` as first parameter
- Use `c.Request.Context()` in handlers
- Use `ctx` variable name consistently

**Database:**
- Use GORM for ORM
- Use `transactions` for multi-step operations
- Use `repository` pattern for data access
- Use `domain` types, not GORM models in handlers

**Event-Driven:**
- Emit events via `EventBus.Publish()`
- Use non-blocking goroutines for event emission
- Create event handlers in `internal/infra/events/`

**Audit Logging:**
- Use `AuditService.LogFileImport()` for imports
- Use `AuditService.LogFileExport()` for exports
- Emit events non-blocking: `go func() { _ = auditService.LogFileExport(...) }()`
- Always include: `data_type`, `record_count`, `file_name`

### Linting Rules

**Golangci-lint Configuration:**
```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode
    - typecheck

linters-settings:
  govet:
    check-shadowing: true
  errcheck:
    check-type-assertions: true
    check-blank: true
  goimports:
    local-prefixes: api-server
```

**Run linter:**
```bash
golangci-lint run ./...
```

### Git Conventions

**Commit Message Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding tests
- `docs`: Documentation
- `chore`: Maintenance tasks
- `audit`: Audit-related changes

**Examples:**
```
feat(advance-payment): implement tiered fee structure

fix(timesheet): correct date parsing in export handler

audit(employee): add import event emission

refactor(bootstrap): consolidate service initialization
```

---

## 🛠️ Command Cheat Sheet

### Development

**Start local development:**
```bash
# Start MySQL and Redis with Docker
docker compose -f docker-compose.dev.yml up -d

# Set environment variables
cp .env.example .env
# Edit .env with your configuration

# Run migrations
go run cmd/migrate/main.go up

# Start server
go run cmd/server/main.go
```

**Run with hot reload:**
```bash
air
```

### Testing

**Run all tests:**
```bash
go test ./... -v -race -cover
```

**Run specific package tests:**
```bash
go test ./internal/app/services/advance_payment/... -v
go test ./internal/transport/http/handlers/timesheet/... -v
```

**Run with coverage:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Run specific test:**
```bash
go test -run TestImportFlexPayFile -v ./internal/app/services/advance_payment/
```

### Building

**Build binary:**
```bash
go build -o bin/payroll-server cmd/server/main.go
```

**Build for Linux:**
```bash
GOOS=linux GOARCH=amd64 go build -o bin/payroll-server-linux cmd/server/main.go
```

**Build Docker image:**
```bash
docker build -t payroll-backend:latest .
```

### Database

**Run migrations:**
```bash
go run cmd/migrate/main.go up
```

**Rollback migrations:**
```bash
go run cmd/migrate/main.go down
```

**Create new migration:**
```bash
go run cmd/migrate/main.go create add_tiered_fee_settings
```

**Connect to MySQL:**
```bash
mysql -h localhost -u payroll_user -p payroll_db
```

**Or use Adminer:**
```bash
# Adminer is available at http://localhost:8081
# Username: root
# Password: (from .env.docker file)
```

### Dependencies

**Install dependencies:**
```bash
go mod download
```

**Tidy dependencies:**
```bash
go mod tidy
```

**Update dependencies:**
```bash
go get -u ./...
go mod tidy
```

**Check for vulnerabilities:**
```bash
govulncheck ./...
```

### Linting & Formatting

**Format code:**
```bash
gofmt -w .
goimports -w .
```

**Run linter:**
```bash
golangci-lint run ./...
```

**Fix linting issues:**
```bash
golangci-lint run --fix ./...
```

### Docker

**Start services:**
```bash
docker compose -f docker-compose.dev.yml up -d
```

**Stop services:**
```bash
docker compose -f docker-compose.dev.yml down
```

**View logs:**
```bash
docker compose -f docker-compose.dev.yml logs -f mysql
docker compose -f docker-compose.dev.yml logs -f redis
```

**Rebuild services:**
```bash
docker compose -f docker-compose.dev.yml up -d --build
```

### Background Jobs (Asynq)

**Start Asynq server:**
```bash
go run cmd/worker/main.go
```

**Check Asynq dashboard:**
```bash
# If enabled, usually at http://localhost:8080/asynq
```

### API Testing

**Test authentication:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}'
```

**Test with JWT:**
```bash
curl http://localhost:8080/api/v1/employees \
  -H "Authorization: Bearer <your-jwt-token>"
```

**Test file upload:**
```bash
curl -X POST http://localhost:8080/api/v1/employees/import \
  -H "Authorization: Bearer <token>" \
  -F "file=@employees.xlsx"
```

### Monitoring & Debugging

**Check application logs:**
```bash
tail -f logs/app.log
```

**Check database connections:**
```bash
# Check if MySQL is running
docker compose -f docker-compose.dev.yml exec mysql mysqladmin ping

# Check if Redis is running
docker compose -f docker-compose.dev.yml exec redis redis-cli ping
```

**Profile CPU:**
```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

**Profile memory:**
```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Deployment

**Build for production:**
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/payroll-server cmd/server/main.go
```

**Run production build:**
```bash
./bin/payroll-server
```

**Deploy to staging/production:**
```bash
# This will depend on your deployment strategy
# Common options: Docker, Kubernetes, SSH deployment
```

### Useful Aliases (add to ~/.bashrc or ~/.zshrc)

```bash
# Payroll Backend aliases
alias pb-start='docker compose -f ~/payroll-backend/docker-compose.dev.yml up -d'
alias pb-stop='docker compose -f ~/payroll-backend/docker-compose.dev.yml down'
alias pb-logs='docker compose -f ~/payroll-backend/docker-compose.dev.yml logs -f'
alias pb-test='cd ~/payroll-backend && go test ./... -v -race'
alias pb-build='cd ~/payroll-backend && go build -o bin/payroll-server cmd/server/main.go'
alias pb-lint='cd ~/payroll-backend && golangci-lint run ./...'
alias pb-migrate='cd ~/payroll-backend && go run cmd/migrate/main.go up'
alias pb-server='cd ~/payroll-backend && go run cmd/server/main.go'
```

---

## 📚 Key Files & Patterns

### Important Files

- **`internal/app/bootstrap/container.go`** - DI container, all handlers initialized here
- **`internal/app/bootstrap/services/init.go`** - Service initialization, dependency wiring
- **`internal/domain/events.go`** - All domain events defined here
- **`internal/infra/events/event_bus.go`** - Event bus implementation
- **`internal/infra/events/audit_event_handler.go`** - Audit event handler
- **`internal/app/services/infrastructure/audit_service.go`** - Audit logging service
- **`migrations/`** - Database migrations
- **`.env.example`** - Environment variables template

### Common Patterns

**1. Creating a New Handler:**
```go
// 1. Define handler struct
type MyHandler struct {
    myService *services.MyService
    auditService *infrastructure.AuditService
}

// 2. Create constructor
func NewMyHandler(
    service *services.MyService,
    auditService *infrastructure.AuditService,
) *MyHandler {
    return &MyHandler{
        myService: service,
        auditService: auditService,
    }
}

// 3. Add to bootstrap/container.go
MyHandler: handlers.NewMyHandler(services.MyService, services.Audit),

// 4. Register routes in routes.go
```

**2. Emitting Audit Events:**
```go
// For imports
if h.auditService != nil {
    go func() {
        _ = h.auditService.LogFileImport(ctx, "data_type", recordCount, filename)
    }()
}

// For exports
if h.auditService != nil {
    go func() {
        _ = h.auditService.LogFileExport(ctx, "data_type", recordCount, filename)
    }()
}
```

**3. Using EventBus:**
```go
// Emit event
event := domain.NewMyEvent(ctx, data)
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}

// Create event handler
type MyEventHandler struct {
    repo domain.MyRepository
}

func (h *MyEventHandler) Handle(ctx context.Context, event domain.Event) error {
    switch e := event.(type) {
    case domain.MyEvent:
        // Handle event
    }
    return nil
}
```

**4. Database Transactions:**
```go
tx := h.db.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

if err := tx.Create(obj).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Commit().Error; err != nil {
    return err
}
```

---

## 🔍 Troubleshooting

### Common Issues

**1. Database connection refused:**
```bash
# Check if MySQL is running
docker compose -f docker-compose.dev.yml ps mysql

# Check logs
docker compose -f docker-compose.dev.yml logs mysql

# Restart MySQL
docker compose -f docker-compose.dev.yml restart mysql
```

**2. Redis connection refused:**
```bash
# Check if Redis is running
docker compose -f docker-compose.dev.yml ps redis

# Check logs
docker compose -f docker-compose.dev.yml logs redis

# Restart Redis
docker compose -f docker-compose.dev.yml restart redis
```

**3. Migration errors:**
```bash
# Check migration status
go run cmd/migrate/main.go version

# Rollback and retry
go run cmd/migrate/main.go down
go run cmd/migrate/main.go up
```

**4. Import/Export not emitting audit events:**
- Check if `auditService` is injected in handler
- Check if event emission code exists
- Check if it's wrapped in goroutine
- Check `AuditService` implementation

**5. Asynq workers not processing:**
```bash
# Check if worker is running
ps aux | grep worker

# Check Asynq dashboard if enabled

# Check Redis queues
redis-cli
> KEYS asynq:*
> LLEN asynq:{queue}:pending
```

---

## 📖 Additional Resources

### Documentation
- [Gin Framework](https://gin-gonic.com/docs/)
- [GORM](https://gorm.io/docs/)
- [Asynq](https://github.com/hibiken/asynq)
- [Go Concurrency Patterns](https://go.dev/doc/effective_go#concurrency)

### Internal Documentation
- `README.md` - Project overview
- `AUDIT_LOG_REVIEW.md` - Audit system review
- `AUDIT_IMPLEMENTATION_COMPLETE.md` - Audit implementation details
- `FEE_TIER_IMPLEMENTATION_PLAN.md` - Fee tier implementation plan

---

## 🔄 Updating This File

**When to update CLAUDE.md:**

1. **After completing major features:** Update "Project State" section
2. **After architectural changes:** Update "Architecture Patterns" section
3. **After adding new commands:** Update "Command Cheat Sheet" section
4. **After changing conventions:** Update "Style Guide" section
5. **After discovering new patterns:** Add to "Common Patterns" section

**Template for updates:**

```markdown
### ✅ Recently Completed (YYYY-MM-DD)

1. **Feature Name** - Status
   - Key files changed
   - Important notes

### 🎯 Next Steps

1. **Next Feature** - Priority
   - Description
```

---

**Last Updated:** 2026-05-06
**Maintained By:** Orbit (AI Agent)
**Version:** 1.0
