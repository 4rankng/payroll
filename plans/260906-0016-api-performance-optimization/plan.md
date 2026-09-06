# API Performance Optimization — Slow Endpoints (7-day latency report)

**Status:** COMPLETE — reviewed, review findings fixed, verified
**Date:** 2026-09-06
**Mode:** /ak-cook --auto

## Outcome

Reduce P95/avg latency for the 10 slowest endpoints (7-day window) using the
existing Redis cache + event-driven invalidation infrastructure. Explicit user
ask: cache `/api/v1/projects` in Redis with correct invalidation on mutation.

## Scout evidence (key facts)

- `CacheService` (Redis, `internal/app/services/infrastructure/cache_service.go`):
  Get/Set/Delete/DeletePattern + `GenerateDashboardCacheKey`.
- `CacheInvalidationHandler` (`internal/infra/events/cache_invalidation_handler.go`)
  subscribes to the event bus and clears `projects:list*`, `timesheets:list:*`,
  `dashboard:*` etc. on Project/ProjectEmployee/Timesheet/Employee/Ledger/
  Transaction/Payrate/Settlement events → invalidation after commit (ADR-007
  satisfied: events publish post-commit).
- `/api/v1/projects` already microcached (`ListProjectsWithEmployeeCount` +
  `CountProjects`, TTL 15s) but: (a) 15s TTL misses most requests,
  (b) `projects:count*` key is NEVER invalidated (pattern gap — count cache
  can go stale for the full TTL after project create/delete).
- Dashboard analytics **not cached at all**: `GetBankUsageAllProjects` (P95 3.6s),
  `GetProjectProfitability` (2.7s), `GetProjectWeeklyProfit` (2.1s),
  `GetMonthlyFinancials` (1.9s). All admin-only (routes_admin.go), no per-user
  variance → single-key cache safe.
- `/api/v1/dashboard/summary` already cached (60s). `/api/v1/timesheets` list
  already cached (15s).
- `/api/v1/timesheets/grouped` (1,101 calls/7d — highest slow-endpoint traffic):
  `ListGroupedByEmployee` uncached; 3 heavy queries + relationship loads.
  `generateSummaryStatsCacheKey` includes partner-scope fields → safe key base.
- `ProjectListCacheTTL = 15s` in `internal/constants/cache.go`.

## Phases

### Phase 1 — Cache dashboard analytics (60s TTL, `dashboard:*` keys) ✅ DONE
1. `bank_usage.go` `GetBankUsageAllProjects` → `dashboard:bank_usage_all_projects`
2. `project_profitability.go` `GetProjectProfitability` → `dashboard:project_profitability`
3. `project_profitability.go` `GetProjectWeeklyProfit(days)` → `dashboard:project_weekly_profit:{days}`
4. `service.go` `GetMonthlyFinancials` → `dashboard:monthly_financials:{period}`
Pattern: exact copy of `GetDashboardSummary` flow (Get → miss → compute → Set with
`DashboardSummaryCacheTTL`). Invalidated by existing `dashboard:*` event patterns.

### Phase 2 — /api/v1/projects strengthening ✅ DONE
5. `constants/cache.go`: `ProjectListCacheTTL` 15s → 60s (event invalidation exists).
6. `cache_invalidation_handler.go`: add `projects:count*` to `Project` and
   `ProjectEmployee` cases (fixes count-cache staleness gap).

### Phase 3 — Grouped timesheets microcache ✅ DONE
7. `timesheet/service.go` `ListGroupedByEmployee`: 15s TTL cache, key derived from
   `generateSummaryStatsCacheKey` with prefix `timesheets:list:grouped` →
   invalidated by existing `timesheets:list:*` pattern. Wrapper struct
   {Groups, Timesheets, Total}. Skip SET on empty result (repo convention).

### Phase 4 — Test-suite repair (found during verification) ✅ DONE
8. `project_employee_repository_test.go`: pre-existing nightly-flaky test
   (fails 00:00–07:00 +07:00): inserted start_date in local time while query
   compares `StartOfDay(clock.NowUTC())`; SQLite string comparison breaks when
   local-date-minus-1 equals UTC-date. Fixed by pushing fixture dates 1 more
   day into the past (tz-robust). Unrelated to caching change; fixed per
   fix-all convention.

## Verification results

- `go build ./...` clean; `go vet` clean on all touched packages.
- Unit tests: dashboard ✓, timesheet ✓, infra/events ✓, persistence ✓ (after Phase 4).
- Integration `api-test`: 304 total / 281 passed / 0 failed / 23 skipped
  (skips are pre-existing env-gated).
- Full unit suite: only failure was the Phase 4 flaky test (now green, 3/3).

## Code review (code-reviewer subagent) — findings & resolution

- **BLOCKER (fixed)**: `generateSummaryStatsCacheKey` omitted `filters.Search`
  while grouped/list/summary queries honor it → different searches shared one
  cache key within TTL. Fixed by adding a Search branch (normalized like the
  project list key). This also repaired the pre-existing collision in the
  `ListTimesheets` microcache. Live-verified: `search=nguy` → 20 groups;
  `search=zzzznonexistent` immediately after → 0 groups (pre-fix: stale 20).
- **WARNING (fixed)**: `BulkTransfer*` events matched no `CanHandle` substring
  ("Transfer" ≠ "Transaction") → disbursement completion only expired caches by
  TTL. Added `BulkTransfer` case (superset: dashboard/timesheets/transactions
  patterns, ordered before "Transaction" since BulkTransferTransactionCreated
  matches both). Direct `BulkUpdatePaymentStatus` calls still publish no event —
  TTL-bounded 60s, same design bound as the pre-existing summary cache.
- **Correction to premise**: `/api/v1/dashboard/*` is NOT admin-only — Casbin
  grants partner GET (configs/casbin_policy.csv:83). Pre-existing authorization
  surface, unchanged here. RESOLVED 2026-09-06: user confirmed partner access
  to all-projects financials is intended.
- Non-blocking notes: monthly-financials caches identical payload under up to 3
  period keys (harmless); TTL race on compute-vs-invalidate is the accepted
  pattern; employee rename leaves grouped names stale ≤15s (TTL-bounded, same
  as existing list cache).
- Reviewer independently verified build/vet/tests and JSON-roundtrip fidelity
  of every cached shape. Verdict: DONE_WITH_CONCERNS → all concerns addressed
  above.

## Follow-up invalidation gaps closed (post-review, user-directed fix-all)

- `app/workers/bulk_transfer_worker.go`: payment-status writes invalidated
  `timesheets:list/summary` directly but not `dashboard:*` → added.
- `app/services/payroll/service.go` `MarkExternallyPaid`: published no event
  (manual "paid externally" flow changes paid amounts) → now publishes the
  previously-unused `BulkTransferPaymentStatusUpdatedEvent`, which the new
  `BulkTransfer` invalidation case catches (dashboard/timesheets/transactions).
  Added `events` field to PayrollService (bus was already a constructor param,
  previously only forwarded to the bulktransfer sub-service).
- `BulkPaymentService.UpdatePaymentStatuses` has no active callers (dead
  interface path) — no change needed.
- Re-verified after these edits: full unit suite zero failures; api-test
  304/281/0/23.

## Live smoke evidence (local dev, air-rebuilt binary)

| Endpoint | miss | hit |
|---|---|---|
| dashboard/bank-usage/projects | 177ms | 3.6ms |
| dashboard/project-profitability | 247ms | 3.1ms |
| dashboard/project-weekly-profit?days=30 | 30ms | 2.2ms |
| dashboard/monthly-financials | 29ms | 2.1ms |
| projects | 7.4ms | 2.5ms |
| timesheets/grouped | 226ms | 81ms (JSON deserialize floor) |

Local DB is small; prod misses (P95 0.9–3.6s) collapse to the hit floor.
Payloads byte-identical across miss/hit.

## Evaluated, declined (with reason)

- `/api/v1/timesheets/export` (5 calls/7d): inherently heavy export; caching
  inappropriate at this traffic.
- `/api/v1/timesheets/cash-readiness` (68 calls/7d ≈ 10/day): advisory forecast
  depends on wallet balance (no wallet event invalidation exists); cache would
  not help first load which is the actual UX cost. Needs query/math optimization
  — follow-up, out of scope.
- `/api/v1/timesheets` + `/dashboard/summary`: already cached; leave TTLs
  (control-center freshness / 60s respectively).
- Latent bug noted (NOT fixed here, behavior change): `GetMonthlyFinancials`
  ignores `req.Period` and always computes 12 months. Cache key includes period
  so honoring it later stays correct.

## Acceptance criteria

1. All 4 dashboard analytics endpoints served from Redis on repeat; keys under
   `dashboard:*`; project mutations / timesheet writes clear them via events.
2. `/api/v1/projects` list+count cached 60s; `projects:count*` invalidated on
   Project/ProjectEmployee events alongside `projects:list*`.
3. `/api/v1/timesheets/grouped` cached 15s; partner-scoped keys (no cross-partner
   leakage); invalidated with `timesheets:list:*`.
4. `go build ./...` + `go vet` clean; unit tests in touched packages pass;
   `make api-test` green.
5. Public API contracts unchanged (same JSON responses).
6. code-reviewer subagent finds no regression.

## Verification

- `cd backend && go build ./... && go test ./internal/app/services/dashboard/... ./internal/app/services/timesheet/... ./internal/infra/events/...`
- `make api-test` from repo root.
