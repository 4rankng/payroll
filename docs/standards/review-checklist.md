# Review Checklist

Pre-merge checklist for every change. Use this before submitting a PR or asking for review. See [Definition of Done](../definition-of-done.md) for the full completion gates.

## Security

- [ ] **No IDOR** — Every endpoint that takes a resource ID validates ownership. Users cannot access another user's resources by changing an ID.
- [ ] **Authz covers new endpoints** — New routes have a Casbin policy row in `configs/casbin_policy.csv`. Default is deny.
- [ ] **Input validation** — Backend validates via domain rules; frontend validates via Zod schemas.
- [ ] **No hardcoded secrets** — Secrets come from `.env` files. Never default secrets in `getEnv()` functions.
- [ ] **No sensitive data in logs** — Use `slog` (`internal/infra/observability/logger.go`), never `fmt.Printf` or `log.Printf`.
- [ ] **Token revocation** — Logout and password reset both blacklist the token and set `invalid_before`.

See [Security Standards](security.md) for threat models.

## Performance

- [ ] **No N+1 queries** — Use `Preload`, explicit joins, or batch processing. Check for loop-based repository calls.
- [ ] **Pagination on list endpoints** — All list endpoints support pagination. Default page size is reasonable.
- [ ] **Projections** — Use `Select` to fetch only needed columns, not full rows.
- [ ] **Indexes** — If adding a new query pattern, run `EXPLAIN` and add an index migration if needed.
- [ ] **Cache invalidation after commit** — Never invalidate cache inside a transaction. See [ADR-007](../decisions/ADR-007-transaction-manager-unit-of-work.md).
- [ ] **Non-blocking operations** — Events and audit logs use goroutines, not the request path.

See [Performance Standards](performance.md) for optimization history.

## Accessibility

- [ ] **ARIA roles** — Interactive elements have appropriate ARIA roles.
- [ ] **Focus states** — Focus is visible and logical (tab order follows visual order).
- [ ] **Keyboard navigation** — All interactive elements are operable via keyboard.
- [ ] **44px tap targets** — Minimum 44px for touch targets on mobile.
- [ ] **Color contrast** — Meets WebAIM Contrast Checker standards.

See [UI Guidelines](ui-guidelines.md).

## Logging

- [ ] **Use `slog`** — Never `fmt.Printf`, `log.Printf`, or `console.log` in production code.
- [ ] **Structured logging** — Key-value pairs, not string interpolation.
- [ ] **No secrets in logs** — JWT tokens, passwords, bank account numbers are never logged.

## Tests

- [ ] **Unit tests** — New domain logic has co-located `*_test.go` tests.
- [ ] **Integration tests** — New API endpoints have a flow test in `backend/tests/integration/`.
- [ ] **FakeClock** — Tests use `FakeClock` or `AutoFake`, never `time.Now()`. See [ADR-006](../decisions/ADR-006-clock-injection-pattern.md).
- [ ] **Existing tests pass** — `make api-test` and `go test ./... -race` are green.
- [ ] **No skipped tests** — Unless explicitly agreed with the team.

See [Testing Strategy](../testing.md).

## Error Handling

- [ ] **Domain errors** — Use `domain.New*Error()` constructors (`NotFoundError`, `ValidationError`, `ForbiddenError`, `ConflictError`, `InternalError`).
- [ ] **No swallowed errors** — Every error is either returned, logged, or handled. No `_ = err`.
- [ ] **Error context** — Use `.WithContext("key", value)` for Sentry-style context on domain errors.

## Backward Compatibility

- [ ] **Public contracts unchanged** — Function signatures, exported types, API responses, DB schemas, env vars, and config keys are backward compatible unless intentionally breaking.
- [ ] **Breaking changes called out** — If a contract must change, document it in the PR description and update [API Reference](../api.md) or [Database](../database.md).

## Database Migration Safety

- [ ] **Idempotent** — Migration is safe to run multiple times (`INSERT IGNORE`, `ON DUPLICATE KEY UPDATE`).
- [ ] **Reversible** — If the migration adds a column, it can be removed without data loss (or the loss is documented and accepted).
- [ ] **Applied before deploy** — Migration is applied to demo/prod before the backend deploy.
- [ ] **`EXPLAIN` checked** — New queries are verified to use indexes.

See [Database Migration Prompt](../prompt-library/database-migration.md).

## Mobile Responsive

- [ ] **Mobile-first** — Employee views are tested on mobile viewport.
- [ ] **`useIsMobile` hook** — Mobile-specific variants use the hook, not CSS media queries alone.
- [ ] **Vietnamese text** — All user-facing text is in Vietnamese.

## Documentation

- [ ] **API changes** — [API Reference](../api.md) updated.
- [ ] **Schema changes** — [Database](../database.md) updated.
- [ ] **New patterns** — [Code Standards](../code-standards.md) updated.
- [ ] **Architectural decisions** — New [ADR](../decisions/README.md) created.
- [ ] **Notable fixes** — [Troubleshooting](../troubleshooting.md) or [Lessons](../lessons/README.md) updated.
