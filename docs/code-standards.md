# Code Standards

Conventions observed in the payroll monorepo codebase. These are patterns already in use -- follow them when adding new code.

See [System Architecture](system-architecture.md) for layer diagrams and data flow.

## Backend Conventions

### DDD Layer Boundaries

Dependency direction: `transport -> app -> domain <- infra`

- `domain/` has zero framework imports (no Gin, no GORM). Only Go stdlib and internal packages.
- `app/services/` orchestrates use cases. Depends on domain ports and domain services.
- `infra/persistence/` implements domain port interfaces. Depends on GORM.
- `transport/http/handlers/` depends on app services and domain types only -- never touches GORM models directly.

### Naming

| Element | Convention | Example |
|---------|-----------|---------|
| Files | `snake_case.go` | `advance_payment_repository.go` |
| Types/Structs | `PascalCase` | `AdvancePayment`, `EmployeeRepository` |
| Variables/Functions | `camelCase` | `calculateFee`, `employeeID` |
| Interfaces | `PascalCase` (no `I` prefix) | `Clock`, `EmployeeRepository` |
| Test files | `*_test.go` (co-located) | `attendance_service_test.go` |
| Test functions | `Test<What><Condition>` | `TestCheckInConcurrentFirstCheckInAllowsOnlyOneOpenRow` |

### Clock

All business time uses `clock.Clock` from `internal/pkg/clock`. Injected via DI.

```go
// Correct -- use injected clock
func (s *AttendanceService) CheckIn(ctx context.Context, clk clock.Clock, ...) error {
    now := clk.Now() // Returns Asia/Ho_Chi_Minh time
}

// Wrong -- never use time.Now() for business logic
func (s *Service) DoSomething() error {
    now := time.Now() // FORBIDDEN
}
```

The `Clock` interface provides: `Now()`, `NowUTC()`, `TodayStart()`, `TodayEnd()`, `UnixNow()`. A `FakeClock` is available for testing.

### Calendar Dates and Database Timezones

Treat `YYYY-MM-DD` business dates differently from timestamps:

- Parse payroll and timesheet dates with `timeutil.ParseBusinessDate()`. Do not use `time.Parse("2006-01-02", ...)`, which silently creates UTC midnight.
- SQL `DATE` filters must implement `UsesCalendarDates() bool`. The shared persistence filter then binds timezone-free `YYYY-MM-DD` strings and uses a half-open `[from, to+1 day)` interval.
- Use UTC only for real instants such as event timestamps, provider timestamps, and audit timestamps.
- Boundary tests must use a non-UTC timezone and assert that both the first and last requested calendar days are included.

This separation prevents the MySQL connection location from shifting a date-only boundary and omitting an entire day.

### Error Handling

Domain errors use `domain.New*Error()` constructors from `internal/domain/errors.go`:

```go
return domain.NewNotFoundError("employee not found")
return domain.NewValidationError("invalid date range")
return domain.NewForbiddenError("action not allowed")
return domain.NewConflictError("timesheet already exists")
return domain.NewInternalError("database error", err)
```

Sentry-style context can be attached: `.WithContext("key", value)`.

Logging uses `slog` via `internal/infra/observability/logger.go` -- never `fmt.Printf` or `log.Printf`.

### Events

Emit via `EventBus.Publish()` in non-blocking goroutines:

```go
event := domain.NewEmployeeCreatedEvent(ctx, employee)
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}
```

Event handlers live in `internal/infra/events/` and implement `domain.EventHandler`.

### Transactions

Use the transaction manager from `internal/infra/transaction/` for multi-step operations. Repository methods accept `*gorm.DB` or use the injected transaction manager.

### Audit Logging

Non-blocking, fire-and-forget pattern:

```go
if h.auditService != nil {
    go func() {
        _ = h.auditService.LogFileImport(ctx, "bcc_timesheet", recordCount, filename)
    }()
}
```

### Adding a New Feature (Backend)

1. Define domain entity in `internal/domain/<entity>.go`
2. Add repository interface in `internal/domain/ports/repository.go`
3. Implement repository in `internal/infra/persistence/<entity>_repository.go`
4. Create app service in `internal/app/services/<domain>/`
5. Add handler in `internal/transport/http/handlers/<domain>/`
6. Wire into DI container in `internal/app/bootstrap/container.go`
7. Register routes in `internal/app/bootstrap/routes.go` (or equivalent)
8. Add Casbin policy row in `configs/casbin_policy.csv`

### Testing

- Tests co-located with source (`*_test.go` in the same package)
- Backend unit tests: `cd backend && go test ./... -v -race -cover`
- Test helpers use `FakeClock` for deterministic time
- Integration tests: `make api-test` (requires running backend)
- Integration test helpers: `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`

## Frontend Conventions

### File Responsibility

- `.tsx` files: UI rendering only. No business logic, no API calls, no data transforms.
- `.ts` files: Business logic, API calls, type definitions, utilities.

### Component Patterns

- shadcn/ui primitives from `@/components/ui/` -- small, composable, accessible-first.
- Domain components in `src/components/<domain>/` -- grouped by business area.
- Barrel exports: every component directory has an `index.ts` re-exporting public API.
- No duplicate versions of the same component (enforce single best version).

### TanStack Query

All API data fetching goes through hooks in `src/hooks/api/`:

```typescript
// Query hook
export function useEmployees(filters: EmployeeFilters) {
  return useQuery({
    queryKey: ['employees', filters],
    queryFn: () => api.employees.list(filters),
  });
}

// Mutation with cache invalidation
export function useCreateEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateEmployeeInput) => api.employees.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
    },
  });
}
```

### Imports

Order: React -> UI libs -> icons -> hooks -> local components -> types. Use `@/` path aliases.

### UI Text

All user-facing text is in Vietnamese. No i18n layer -- strings are inline in components.

### Mobile-First

Employee views are mobile-first. Use `useIsMobile` hook and `useBreakpoint`. Mobile-specific components exist alongside desktop variants (e.g., `employee-table-desktop.tsx` / `employee-table-mobile.tsx`).

### Forms

`react-hook-form` with `zod` validation via `@hookform/resolvers/zod`.

## Shared Conventions

### Git Commits

Format: `<type>(<scope>): <subject>`

Types: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `audit`

Examples:
```
feat(attendance): add geofence validation for checkout
fix(timesheet): correct date parsing in export handler
audit(employee): add import event emission
refactor(bootstrap): consolidate service initialization
```

No AI references in commit messages.

### Branch Strategy

Work on `main` directly. No worktrees or feature branches by convention.

### Database

- GORM for all access. No raw SQL in application code.
- Repository pattern with domain types at service/handler boundaries.
- Migrations in `backend/migrations/` as `.up.sql` files.

### TypeScript

Strict mode enabled. No `any` types.

### No Hardcoded Secrets

Secrets come from `.env` files. Never default secrets in `getEnv()` functions.
