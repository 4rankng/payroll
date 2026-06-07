<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# pkg — Shared Utility Packages

## Purpose
Contains shared utility packages used across the entire application. These packages have no business logic and no dependency on domain or application layers. They provide reusable infrastructure: centralized business clock, constants, database helpers, retry logic, input validation, HTTP utilities, geo/IP services, password hashing, time formatting, and context helpers.

## Key Files
_None at this level — all code is in subdirectories._

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `clock/` | Centralized business clock — all business time flows through here (see `clock/AGENTS.md`) |
| `constants/` | Application-wide constants — cache TTLs, date formats, business rules |
| `context/` | Context helpers — extracting user info, tenant ID from request context |
| `db/` | Database helpers — batch operations, batch loader, temporal helpers |
| `geo/` | Geolocation utilities — geofence calculations |
| `httputils/` | HTTP utilities — client helpers, request building |
| `ipgeo/` | IP geolocation — IP-to-location lookup |
| `password/` | Password utilities — bcrypt hashing, validation |
| `retry/` | Retry logic — exponential backoff with jitter (5.9K, tested) |
| `scopes/` | Query scope helpers — GORM scope functions for common filters |
| `services/` | Shared service utilities |
| `tenantqueue/` | Tenant-aware queue — per-tenant job queue management |
| `timeutil/` | Time utilities — date formatting, parsing helpers |
| `ua/` | User agent parsing — browser/device detection |
| `utils/` | General utilities — date ranges, payroll period calculations |
| `validation/` | Input validation — reusable validation rules (4.3K, tested) |

## For AI Agents

### Working In This Directory
- These packages are **infrastructure utilities** — no business logic
- Always import `clock` from here for business time: `clock.Now()` not `time.Now()`
- Use `retry` for any external service calls that may fail transiently
- Use `validation` for common input validation patterns
- Use `db/batch_helper.go` for bulk database operations
- Constants in `constants/` define cache TTLs, date formats, and other shared values

### Testing Requirements
- `retry/retry_test.go` — tests exponential backoff behavior
- `validation/validators_test.go` — tests validation rules
- `db/batch_loader_test.go` — tests batch loading
- `password/hash_test.go` — tests hashing and verification

### Common Patterns
```go
// Clock usage (ALWAYS for business logic)
now := clock.Now()           // Asia/Ho_Chi_Minh timezone
utc := clock.NowUTC()        // UTC

// Retry with exponential backoff
err := retry.Do(ctx, func() error {
    return externalService.Call(ctx, payload)
}, retry.MaxRetries(3))

// Validation
if err := validation.ValidateEmail(email); err != nil {
    return domain.NewValidationError("invalid email")
}
```

## Dependencies

### Internal
_None_ — these are the lowest-level internal packages

### External
- `gorm.io/gorm` — used by `db/` and `scopes/`
- `redis/go-redis` — used by `tenantqueue/`
- `golang.org/x/crypto` — used by `password/` for bcrypt

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
