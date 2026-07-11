# Prompt: Feature Implementation

Use when adding a new backend feature following the DDD/Clean Architecture pattern.

## Prompt

```
Goal: Implement [FEATURE_NAME] — [ONE_SENTENCE_DESCRIPTION]

Context to gather:
1. Read backend/AGENTS.md and backend/internal/AGENTS.md for architecture rules
2. Read the AGENTS.md in the target domain directory (e.g., backend/internal/domain/AGENTS.md)
3. Read docs/decisions/ADR-001-ddd-clean-architecture.md for layer boundary rules
4. Find a similar existing feature to use as a reference pattern
5. Read docs/code-standards.md → "Adding a New Feature (Backend)" section

Constraints:
- Domain layer (internal/domain/) has ZERO framework imports — no Gin, no GORM
- All business time uses clock.Clock injection — never time.Now()
- Use domain error constructors (NewNotFoundError, NewValidationError, etc.)
- Events published via EventBus.Publish() in non-blocking goroutines
- Cache invalidation AFTER transaction commit
- Commit format: <type>(<scope>): <subject>
- Vietnamese for all user-facing text

Output format:
Implement the feature across these layers (in order):
1. Define domain entity in internal/domain/[entity].go
2. Add repository interface in internal/domain/ports/repository.go
3. Implement repository in internal/infra/persistence/[entity]_repository.go
   - Use db.WithContext(ctx)
   - Handle gorm.ErrRecordNotFound → domain.NewNotFoundError()
   - Map via model.ToDomain()
4. Create app service in internal/app/services/[domain]/
5. Add DTOs in internal/app/dto/
6. Add handler in internal/transport/http/handlers/[domain]/
7. Wire into DI container in internal/app/bootstrap/services/init.go and container.go
8. Register routes in internal/app/bootstrap/routes_[domain].go
9. Add Casbin policy row in configs/casbin_policy.csv
10. Add unit tests (co-located *_test.go with FakeClock)
11. Add integration test flow in backend/tests/integration/

Verification:
- go build ./... passes
- go test ./... -race -cover passes
- make api-test passes
- gofmt -l . outputs nothing
- golangci-lint run ./... is clean
- New endpoint is accessible with correct role
```

## Example Usage

```
Goal: Implement expense tracking — allow admins to record project expenses with receipts

Context to gather:
[... same as above ...]

Constraints:
[... same as above ...]
```
