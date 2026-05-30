# AGENTS.md - Payroll Backend

> **Long-Term Memory for AI Agents**
> This file provides context, patterns, and commands for working with the payroll backend.

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
