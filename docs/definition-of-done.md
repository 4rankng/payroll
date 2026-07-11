# Definition of Done

A task is complete only when ALL of the following gates pass. Use this checklist before submitting changes for review. See [Review Checklist](standards/review-checklist.md) for the detailed pre-merge checklist and [Testing](testing.md) for test commands.

## Completion Gates

```
Build passes → Tests pass → Lint clean → Type-check clean → No new TODOs →
Migration reviewed → Performance checked → Security reviewed → Docs updated → Mobile responsive
```

### 1. Build Passes

```bash
# Backend
cd backend && go build ./...

# Frontend
cd frontend && pnpm build
```

No compilation errors. No build warnings introduced by the change.

### 2. Tests Pass

```bash
# Backend unit tests
cd backend && go test ./... -v -race -cover

# Integration tests (requires running backend)
make api-test

# Frontend E2E
cd frontend && pnpm test:e2e
```

100% of existing tests must pass. New features must include test coverage. No skipped tests unless explicitly agreed.

### 3. Lint Clean

```bash
# Backend
cd backend && gofmt -l .    # Should output nothing
cd backend && go vet ./...
cd backend && golangci-lint run ./...

# Frontend
cd frontend && pnpm lint    # eslint --max-warnings=0
```

CI enforces `--max-warnings=0` on ESLint and `gofmt -l` on Go. No new lint errors or warnings.

### 4. Type-Check Clean

```bash
cd frontend && pnpm type-check    # tsc -b (checks all referenced projects)
```

No TypeScript errors. Strict mode is enabled — no `any` types.

### 5. No New TODOs

Check for unintended TODO/FIXME/HACK comments in the change:
```bash
git diff --cached | grep -E "^\+.*(TODO|FIXME|HACK)"
```

Existing TODOs in the codebase (only 2 in backend):
- `wallet_service.go:505` — generate Excel using excelize
- `transactions/repository.go:15` — TASK-039: ListByProviderStatuses

New TODOs require explicit team agreement and should reference a task/issue.

### 6. Migration Reviewed

If the change includes a database migration:
- Migration is **idempotent** (safe to run multiple times).
- Recovery/repair scripts use `INSERT IGNORE` or `ON DUPLICATE KEY UPDATE`.
- Migration is applied to demo/prod **before** the backend deploy.
- `EXPLAIN` the affected queries to verify index usage.
- No destructive operations (column drops, table drops) without a backup.

See [Database](database.md) → Migration Rules and [Database Migration Prompt](prompt-library/database-migration.md).

### 7. Performance Checked

- No new N+1 queries (use `Preload`, joins, or batch processing).
- No full table scans on large tables (check `EXPLAIN`).
- No missing pagination on list endpoints.
- Cache invalidation happens **after** transaction commit.
- No blocking operations in request path (use goroutines for events/audit).

See [Performance Standards](standards/performance.md).

### 8. Security Reviewed

- No IDOR vulnerabilities (validate resource ownership).
- Input validation via Zod (frontend) and domain validation (backend).
- No hardcoded secrets (use `.env` files).
- Auth and Casbin policies cover new endpoints.
- No sensitive data in logs (use `slog`, never `fmt.Printf`).

See [Security Standards](standards/security.md).

### 9. Documentation Updated

Update relevant documentation when the change:
- Adds or modifies API endpoints → update [API Reference](api.md)
- Changes database schema → update [Database](database.md)
- Introduces new patterns or conventions → update [Code Standards](code-standards.md)
- Adds architectural decisions → create an [ADR](decisions/README.md)
- Resolves a notable issue → add to [Troubleshooting](troubleshooting.md) or [Lessons](lessons/README.md)

### 10. Mobile Responsive

- Employee views are mobile-first. Test on mobile viewport.
- Use `useIsMobile` hook and mobile-specific component variants.
- Minimum 44px tap targets for interactive elements.
- All user-facing text is in Vietnamese.

See [UI Guidelines](standards/ui-guidelines.md).

## CI Gates

The CI pipeline (`.github/workflows/ci-cd.yml`) automatically enforces gates 1-8 on every PR and push to `main`:

| CI Job | Gates Enforced |
|--------|----------------|
| `backend-lint` | 3 (Go formatting, vet, golangci-lint) |
| `frontend-lint` | 3, 4 (ESLint, TypeScript, Vite build) |
| `backend-test` | 2 (Go tests with coverage) |
| `frontend-test` | 2 (Vitest with coverage) |
| `security` | 8 (gosec, npm audit) |
| `docker-*` | Builds amd64 images (only on push to main) |
| `deploy` | SSH deploy to production (only on push to main) |

## Summary Checklist

- [ ] Build passes (backend + frontend)
- [ ] All tests pass (unit + integration + E2E)
- [ ] No lint errors or warnings
- [ ] TypeScript type-check clean
- [ ] No unintended TODOs/FIXMEs
- [ ] Migration reviewed (if applicable)
- [ ] No performance regressions
- [ ] No security vulnerabilities
- [ ] Documentation updated
- [ ] Mobile responsive verified
