# ADR-006: Clock Injection Pattern

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system is highly time-sensitive: timesheet check-in/out, pay cycle calculations, fee schedules, settlement deadlines, and forecast horizons all depend on the current time. The system operates in the `Asia/Ho_Chi_Minh` timezone.

For testing, the system must be able to "freeze" or "advance" time to test scenarios like:
- What happens if an employee checks in at 11 PM?
- What does the cash-readiness forecast show 3 days before payday?
- How does the system behave at month-end boundaries?

Using `time.Now()` directly makes this impossible — there is no way to control time in tests without mocking every call site.

## Decision

All business time uses an injected `clock.Clock` interface from `internal/pkg/clock/`. Never use `time.Now()` for domain logic.

### The Clock Interface

```go
type Clock interface {
    Now() time.Time       // Returns Asia/Ho_Chi_Minh time
    NowUTC() time.Time    // Returns UTC time
    TodayStart() time.Time
    TodayEnd() time.Time
    UnixNow() int64
}
```

### Implementations

| Implementation | Usage |
|---------------|-------|
| `RealClock` | Production. Wraps `time.Now()`. |
| `FakeClock` | Tests. Frozen at a specific time via `NewFake(t)`. |
| `AutoFake` | Tests. Starts unfrozen like RealClock until `Set()`/`Advance()` freezes it. Prevents outbox/cron workers from getting stuck with stale time. |

### DI Injection

The clock is injected via the DI container (`internal/app/bootstrap/`). Services receive it as a constructor parameter.

### Admin Clock Endpoints (Non-Prod Only)

For integration tests, admin endpoints allow server-side time manipulation:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/admin/clock/time` | Get current server time |
| POST | `/api/v1/admin/clock/set` | Set server to specific time |
| POST | `/api/v1/admin/clock/advance` | Advance time by duration |
| POST | `/api/v1/admin/clock/reset` | Reset to real time |

These endpoints are **not mounted in production**.

### Exclusions

The rule has narrow exceptions: event bus duration measurements and health check response-time measurement use `time.Now()` directly. These are infrastructure measurements, not business logic.

## Consequences

**Positive:**
- Deterministic time in tests — no flaky tests due to clock drift.
- Integration tests can simulate any time scenario via HTTP endpoints.
- Timezone is centralized — no risk of UTC vs. local time bugs.
- `AutoFake` prevents background workers from hanging during tests.

**Negative:**
- Every service that needs time must accept a `Clock` parameter (boilerplate).
- Developers must remember to use `clk.Now()` instead of `time.Now()` — code review enforces this.
- Admin clock endpoints must be carefully excluded from production builds.

## Alternatives Considered

1. **`time.Now()` with monkey-patching** — Rejected. Fragile, not thread-safe, and Go does not support monkey-patching in the standard toolchain.
2. **Package-level `clock.Now()` function** — Rejected. Global state makes it impossible to run parallel tests with different times.
3. **Context-based time** — Rejected. Overly complex; DI injection is simpler and more explicit.
