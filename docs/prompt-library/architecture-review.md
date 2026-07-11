# Prompt: Architecture Review

Use when auditing layer boundaries, dependency direction, and architectural compliance.

## Prompt

```
Goal: Architecture review of [MODULE/COMPONENT/PR] — [SCOPE]

Context to gather:
1. Read docs/decisions/ADR-001-ddd-clean-architecture.md for layer rules
2. Read docs/decisions/ADR-002-go-gin-gorm-stack.md for stack
3. Read backend/internal/AGENTS.md for directory structure
4. Read docs/code-standards.md for conventions
5. Read docs/standards/review-checklist.md

Review checklist:

1. LAYER BOUNDARIES
   - [ ] domain/ has ZERO framework imports (no Gin, no GORM)
       grep -rn "gin\|gorm\|redis" backend/internal/domain/ --include="*.go"
       (should return nothing except test files)
   - [ ] transport/http/handlers/ depends on app services and domain types only
       — never imports infra/persistence or GORM models
   - [ ] app/services/ depends on domain ports, not concrete infra implementations
   - [ ] infra/ implements domain/ports/ interfaces

2. DEPENDENCY DIRECTION
   - [ ] Dependencies point inward: transport → app → domain ← infra
   - [ ] No circular imports
   - [ ] Domain does not depend on app or infra

3. CLOCK INJECTION (ADR-006)
   - [ ] No time.Now() in domain or app layer
       grep -rn "time\.Now()" backend/internal/domain/ backend/internal/app/ --include="*.go"
       (should return nothing)
   - [ ] All services that need time accept clock.Clock via DI

4. ERROR HANDLING
   - [ ] Domain errors use domain.New*Error() constructors
   - [ ] No fmt.Printf or log.Printf (use slog)
   - [ ] gorm.ErrRecordNotFound translated to domain.NewNotFoundError()

5. EVENT BUS (ADR-004)
   - [ ] Events published via EventBus.Publish() in non-blocking goroutines
   - [ ] Event handlers in internal/infra/events/ implement domain.EventHandler

6. TRANSACTIONS (ADR-007)
   - [ ] Multi-step operations use txManager.RunInTransaction()
   - [ ] Cache invalidation AFTER commit, not before

7. CASBIN (ADR-008)
   - [ ] New endpoints have Casbin policy rows
   - [ ] No manual role checks (if role == "admin") — use Casbin

8. TESTING
   - [ ] Tests use FakeClock, not time.Now()
   - [ ] Tests co-located with source (*_test.go)
   - [ ] No skipped tests without explanation

Output format:
1. Findings table: | Layer | File | Violation | Severity |
2. For each violation: what's wrong, why it matters, recommended fix
3. Overall assessment: compliant / needs fixes / non-compliant
4. Priority list of fixes (if any)
```

## Quick Boundary Check

```bash
# Domain should NOT import Gin or GORM
grep -rn "gin-gonic\|gorm.io" backend/internal/domain/ --include="*.go"
# Expected: no output (except maybe test files)

# Handlers should NOT import GORM or persistence directly
grep -rn "gorm.io\|infra/persistence" backend/internal/transport/http/handlers/ --include="*.go"
# Expected: no output

# Domain should NOT import time.Now() directly
grep -rn "time\.Now()" backend/internal/domain/ --include="*.go" | grep -v "_test.go"
# Expected: no output

# Check for circular imports
cd backend && go vet ./...
```
