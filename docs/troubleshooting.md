# Trououbleshooting

Known recurring issues, their causes, and fixes. Extracted from git history, AGENTS.md files, and operational experience. See [Testing](testing.md) for test-specific issues.

## Backend

### Database connection refused

**Symptom:** Backend fails to start with `dial tcp: connection refused` on `:3306`.

**Cause:** MySQL Docker container not running, or `make db` was not executed.

**Fix:**
```bash
make db    # Starts MySQL + Redis + Adminer + sandbox mocks
# Verify:
docker exec payroll-mysql mysql -uroot -prootpassword -e "SELECT 1"
```

### Redis connection refused

**Symptom:** Event bus, asynq, or caching errors with `connection refused` on `:6379`.

**Cause:** Redis container not running.

**Fix:**
```bash
make db    # Starts Redis alongside MySQL
# Verify:
docker exec payroll-redis redis-cli ping   # Should return PONG
```

### Migration errors

**Symptom:** Backend fails on startup with SQL errors referencing missing columns/tables.

**Cause:** Database schema is behind the code. A migration was added but not applied.

**Fix:**
```bash
# Apply migrations locally
cd backend && go run cmd/migrate/main.go up

# For demo/prod, apply before deploy (see deployment-guide.md):
cat backend/migrations/XXX_name.up.sql | ssh root@demo.tingting.vip \
  "docker exec -i payroll-mysql mysql -u root -p'<password>' payroll_db"
```

**Note:** `schema_migrations` table may be empty on dump-seeded environments. Manual DDL application does not require a version row. Always prefer code-level fixes over schema migrations when possible. Recovery/repair scripts must be idempotent.

### asynq workers not processing

**Symptom:** Background jobs (bulk transfer, disbursement, audit) are enqueued but never execute.

**Cause:** asynq worker process not started, or Redis queue is stale.

**Fix:**
1. Check that the backend server is running (workers start as goroutines via `bootstrap/server.go`).
2. Check Redis queues: `docker exec payroll-redis redis-cli LRLEN asynq:default`
3. Restart the backend — workers re-register on startup via `mux.go`.
4. Inspect asynq monitoring endpoint if enabled.

### Import/Export not emitting audit events

**Symptom:** File import/export operations complete but no audit log entry appears.

**Cause:** `auditService` not injected in the handler or service.

**Fix:** Verify DI wiring in `internal/app/bootstrap/services/init.go` and `container.go`. The audit service must be passed to the handler constructor. See `backend/CLAUDE.md` → Audit Log Implementation.

## Frontend

### Dual lockfile pitfall (yarn.lock vs pnpm-lock.yaml)

**Symptom:** `make deploy` or `make demo` fails during Docker build with yarn dependency errors. Local dev works fine with pnpm.

**Cause:** Frontend has both `yarn.lock` (used by Docker build) and `pnpm-lock.yaml` (used by local dev). Running `pnpm add` locally updates `pnpm-lock.yaml` but leaves `yarn.lock` stale.

**Fix:** Regenerate `yarn.lock` from `package.json`. Work in an isolated temp directory to avoid `node_modules` conflicts:
```bash
# In a temp dir, copy package.json, run yarn install, copy yarn.lock back
mkdir -p /tmp/yarn-sync && cp frontend/package.json /tmp/yarn-sync/
cd /tmp/yarn-sync && yarn install && cp yarn.lock /path/to/payroll/frontend/
```

See [Deployment Guide](deployment-guide.md) → Dual Lockfile Pitfall.

### Frontend dev server won't start

**Symptom:** `pnpm dev` fails or hangs.

**Fix:**
```bash
cd frontend
pnpm install        # Ensure deps are installed
pnpm dev            # Starts Vite on :5173
```

Check that the backend is running on `:8080` (API proxy target). Check `frontend/.env` for correct `VITE_API_URL`.

## Testing Issues

### Test data pollution

**Symptom:** Integration tests fail with "date already occupied" or unexpected data from prior runs.

**Cause:** Integration tests create real records against the dev database. Over time, available dates fill up.

**Fix:**
```bash
# Clean timesheet records to free up dates
docker exec payroll-mysql mysql -uroot -prootpassword payroll_db \
  -e "DELETE FROM timesheets WHERE date >= CURDATE()"
```

Use `pageSize=200` when querying timesheets in tests to avoid pagination false failures.

### Clock pollution between tests

**Symptom:** Tests fail intermittently because the server clock is frozen at an unexpected time.

**Cause:** A previous test called `SetServerTime()` without calling `ResetServerTime()`.

**Fix:** Always pair clock manipulation with reset:
```go
SetServerTime(client, specificTime)
defer ResetServerTime(client)
```

## Performance Issues

### N+1 queries

**Symptom:** Dashboard or list endpoints are slow; many database round-trips per request.

**Cause:** Loop-based repository calls instead of batch queries.

**History:** Multiple rounds of N+1 elimination (commits `ba953e3`, `712a551`, `704d085`):
- `ba953e3` — Phase 4: N+1, proper pagination, projections, NOT EXISTS, dead-code removal
- `712a551` — Kill 3 N+1 loops: payroll-report, loan-list, loan-schedules
- `704d085` — Migration 088: add 4 missing query indexes

**Fix:** Use `Preload` or explicit joins. Use `common/batch_processor.go` for batch operations. Use projections (`Select`) to avoid loading full rows. Add indexes via migration if `EXPLAIN` shows full table scans. See [Performance Standards](standards/performance.md).

### Missing query indexes

**Symptom:** Slow queries on large tables.

**Fix:** Check `EXPLAIN` for the query. If scanning too many rows, add an index migration. Index migrations: 010, 013, 014, 022-026, 030, 037, 088. Follow the same `.up.sql` naming convention.

## Security Issues

### IDOR (Insecure Direct Object Reference)

**Symptom:** A user can access another user's resources by changing an ID in the URL.

**History:** Commit `619f9cd` — red-team fix closing IDOR, spray, and token-revocation gaps.

**Fix:** Always validate resource ownership in the handler/service layer. Use Casbin policies for role-based access. Never trust client-provided IDs without server-side ownership checks. See [Security Standards](standards/security.md).

### Token revocation not working

**Symptom:** Logged-out users can still make API calls with old tokens.

**Cause:** Token blacklist not checked, or `invalid_before` timestamp not enforced.

**Fix:** Ensure `blacklisted_tokens` table is checked in auth middleware (migration 086 added `invalid_before`). Password reset and logout should both blacklist the current token and set `invalid_before`.

## Deployment Issues

### Docker build fails on arm64

**Symptom:** `docker buildx build` fails or produces wrong architecture.

**Cause:** Building without `--platform linux/amd64` on an arm64 machine (e.g. Apple Silicon).

**Fix:** Always use `--platform linux/amd64`:
```bash
docker buildx build --platform linux/amd64 --tag <image> --push .
```

The Makefile `push` targets already include this flag. See [Deployment Guide](deployment-guide.md).

### Demo DB out of sync

**Symptom:** Demo server returns errors after a schema change.

**Fix:**
```bash
make demo-db    # Dumps local dev DB and restores to demo server
# Also resets all demo user passwords to Admin123
```

This does NOT touch demo Docker images — only the database contents.
