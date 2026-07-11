# Performance Standards

Performance patterns, optimization guidelines, and history for the payroll system. See [Review Checklist](review-checklist.md) for the pre-merge performance checklist.

## Core Principles

1. **Don't fetch what you don't need** — Use projections (`Select`) to fetch only needed columns.
2. **Don't make N+1 queries** — Use `Preload`, joins, or batch processing.
3. **Paginate everything** — No unbounded list queries.
4. **Cache after commit** — Redis cache invalidation happens after transaction commit, never before. See [ADR-007](../decisions/ADR-007-transaction-manager-unit-of-work.md).
5. **Keep the request path fast** — Events, audit logs, and notifications use non-blocking goroutines.

## Backend Performance

### Database Queries

#### N+1 Elimination

The system has gone through multiple rounds of N+1 elimination:

| Commit | Fix |
|--------|-----|
| `ba953e3` | Phase 4: N+1, proper pagination, projections, NOT EXISTS, dead-code removal |
| `712a551` | Kill 3 N+1 loops: payroll-report, loan-list, loan-schedules |
| `704d085` | Migration 088: add 4 missing query indexes |

**How to avoid N+1:**
```go
// WRONG — N+1 (one query per employee)
employees := repo.FindAll()
for _, e := range employees {
    projects := projectRepo.FindByEmployeeID(e.ID) // N queries!
}

// CORRECT — batch query
employees := repo.FindAll()
ids := extractIDs(employees)
projects := projectRepo.FindByEmployeeIDs(ids) // 1 query
```

Use `Preload` for nested relations:
```go
db.Preload("Projects.Employees").Find(&users)
```

#### Projections

Use `Select` to fetch only needed columns:
```go
db.Model(&Employee{}).Select("id, name, phone").Where("active = ?", true).Find(&results)
```

#### Pagination

All list endpoints support pagination. Use `LIMIT` and `OFFSET`:
```go
db.Offset(offset).Limit(pageSize).Find(&results)
```

In integration tests, always use `pageSize=200` for timesheets to avoid pagination false failures.

#### NOT EXISTS

Use `NOT EXISTS` instead of `NOT IN` for anti-joins (performs better on large datasets):
```sql
SELECT * FROM employees e
WHERE NOT EXISTS (SELECT 1 FROM timesheets t WHERE t.employee_id = e.id AND t.date = ?)
```

### Query Indexes

Performance index migrations: 010, 013, 014, 022-026, 030, 037, 088. When adding a new query pattern:

1. Run `EXPLAIN` on the query.
2. If scanning too many rows, add an index migration (`.up.sql`).
3. Follow the existing naming convention (`add_query_perf_indexes`).

### Caching

- **Redis caching** with post-commit invalidation. See [ADR-007](../decisions/ADR-007-transaction-manager-unit-of-work.md).
- Assignment caching: key pattern `assignment:{projectID}:{employeeID}`.
- Query result caching: `GetSummaryStats` caches/invalidates the hot number (e.g., pending payment amount).
- **No separate backend cache** for the cash-readiness card — `GetSummaryStats` already caches/invalidates the hot number; the cohort is a scoped indexed `GROUP BY`. Avoids stale-cache risk.

### Batch Processing

Use `common/batch_processor.go` for batch insert/update:
```go
batchProcessor := common.NewBatchProcessor(db, 1000) // batch size
batchProcessor.BatchInsert("timesheets", records)
```

### Non-Blocking Operations

Events and audit logs use goroutines, not the request path:
```go
// Audit logging — fire and forget
if h.auditService != nil {
    go func() {
        _ = h.auditService.LogFileImport(ctx, "bcc_timesheet", recordCount, filename)
    }()
}

// Event publishing — non-blocking
if err := h.eventBus.Publish(ctx, event); err != nil {
    logger.Warn("failed to emit event", "error", err)
}
```

### Request Timeout

`RequestTimeout(30s)` middleware enforces a maximum request duration. Long-running operations (bulk transfers, imports) use asynq background jobs instead.

## Frontend Performance

### Code Splitting

- Vite handles automatic code splitting for routes.
- Lazy-loaded routes reduce initial bundle size.

### TanStack Query Caching

```typescript
// Query with stale time to avoid refetching
useQuery({
  queryKey: ['employees', filters],
  queryFn: () => api.employees.list(filters),
  staleTime: 5 * 60 * 1000, // 5 minutes
});

// Mutation with cache invalidation
useMutation({
  mutationFn: (data) => api.employees.create(data),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['employees'] });
  },
});
```

### Memoization

- `useMemo` for expensive computations.
- `useCallback` for handlers passed to memoized children.
- `React.memo` for components that receive stable props.
- Memoize React elements passed to parents via callbacks.

### PWA Caching

- **Workbox** service worker handles asset caching.
- PWA manifest in `frontend/public/`.
- Offline-capable for previously loaded pages.

## Monitoring

- **Prometheus metrics** at `/metrics` endpoint (protected with auth + RBAC).
- **API metrics** (`api_metrics.go` middleware): request latency, status codes, endpoint metrics. Queryable via `/api/v1/metrics/*` endpoints.
- **DB metrics** in `internal/infra/observability/`.
- **CI coverage** artifacts uploaded for both backend and frontend.

## Performance Review Checklist

- [ ] No N+1 queries (check for loop-based repo calls).
- [ ] `EXPLAIN` new queries on large tables.
- [ ] Pagination on list endpoints.
- [ ] Projections (`Select`) where full rows aren't needed.
- [ ] Cache invalidation after commit.
- [ ] Non-blocking events/audit.
- [ ] `useMemo`/`useCallback` for expensive frontend computations.
- [ ] No blocking operations in the request path.
