<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# clock — Centralized Business Clock

## Purpose
The single source of truth for all business time in the application. All business logic must use `clock.Now()` (returns Asia/Ho_Chi_Minh timezone) or `clock.NowUTC()` instead of `time.Now()`. Provides a `Clock` interface for dependency injection, with `RealClock` for production and `FakeClock`/`AutoFake` for testing. Non-production environments expose admin endpoints to manipulate time for testing.

## Key Files
| File | Description |
|------|-------------|
| `clock.go` | Clock interface, `RealClock`, `FakeClock`, `AutoFake` implementations. Global `Now()` and `NowUTC()` functions. `DefaultLocation = Asia/Ho_Chi_Minh`. Fake clock supports `Set()`, `Advance()`, `Reset()`, `IsFrozen()` (5.4K) |
| `advance_payment.go` | Advance payment clock helpers — time-specific calculations for advance payment windows (3.9K) |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- **CRITICAL:** Never use `time.Now()` in business logic — always use `clock.Now()` or inject `Clock` interface
- `clock.Now()` returns time in `Asia/Ho_Chi_Minh` timezone
- `clock.NowUTC()` returns UTC time
- Handlers receive `clock.Clock` via DI — use the injected clock, not the global
- Infrastructure/services that don't have DI use the global `clock.Now()`
- **Exclusions:** Event bus duration measurements and health check response-time measurement use `time.Now()` directly

### FakeClock Behavior (for tests)
- `NewFake(t)` — frozen at given time
- `NewAutoFake()` — starts unfrozen, behaves like `RealClock` until `Set()`/`Advance()` freezes it
- `Reset()` — unfreezes, restores auto-advance behavior (important for background workers)
- `IsFrozen()` — reports whether clock is manually set
- Prevents outbox/cron workers from getting stuck with stale time after clock manipulation tests

### Admin Endpoints (non-prod only)
- `GET/POST /api/v1/admin/clock/*` — set, advance, reset, get current time
- Requires auth + Casbin authorization
- Not mounted in production (returns 404)

### Integration Test Helpers
Located in `tests/integration/clock_helper.go`:
- `SetServerTime(client, time)` — set server to specific time
- `AdvanceServerTime(client, duration)` — advance time by duration
- `ResetServerTime(client)` — unfreeze, restore auto-advance
- `GetServerTime(client)` — get current server time

### Testing Requirements
- Clock manipulation tests must call `Reset()` afterward to avoid polluting other tests
- Integration tests verify clock manipulation via admin endpoints

### Common Patterns
```go
// Global usage (infrastructure/services)
now := clock.Now()

// DI usage (handlers/services)
type MyService struct {
    clock clock.Clock
}

func (s *MyService) DoSomething(ctx context.Context) {
    now := s.clock.Now() // always use injected clock
}

// Test setup with fake clock
fakeClock := clock.NewAutoFake()
svc := NewMyService(fakeClock)

// Manipulate time in tests
fakeClock.Advance(24 * time.Hour)
defer fakeClock.Reset() // always reset
```

## Dependencies

### Internal
_None_ — this is a foundational package with no internal dependencies

### External
- Standard library `time` package only

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
