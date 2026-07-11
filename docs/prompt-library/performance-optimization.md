# Prompt: Performance Optimization

Use when profiling and optimizing slow endpoints or queries.

## Prompt

```
Goal: Optimize [ENDPOINT/QUERY/COMPONENT] — [CURRENT_PERFORMANCE_ISSUE]

Context to gather:
1. Read docs/standards/performance.md for patterns and history
2. Read docs/lessons/2026-07-04-db-performance-n-plus-1-elimination.md for past fixes
3. Identify the slow query or code path:
   - Check /api/v1/metrics/* for latency data
   - Check Prometheus metrics at /metrics
   - Add timing logs if needed: logger.Info("query took", "duration_ms", elapsed)
4. Run EXPLAIN on the query:
   docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "EXPLAIN SELECT ..."
5. Check for N+1 patterns: loop-based repository calls
6. Check existing indexes: migrations 010-088

Constraints:
- Don't change the API contract (response shape, status codes)
- Don't sacrifice correctness for speed
- Cache invalidation must happen AFTER transaction commit
- New indexes require a migration (.up.sql file)
- No raw SQL in application code — use GORM query builders

Optimization steps (in priority order):
1. Eliminate N+1 queries — use Preload, batch queries, or joins
2. Add projections — Select only needed columns
3. Add pagination if missing
4. Add database indexes if EXPLAIN shows full table scans
5. Add Redis caching for read-heavy, rarely-changing data
6. Move expensive operations to asynq background jobs
7. Use NOT EXISTS instead of NOT IN for anti-joins

Output format:
1. Before: current query/code, EXPLAIN output, measured latency
2. Root cause: what makes it slow (N+1, missing index, full scan, etc.)
3. Fix: what changed and why
4. After: new EXPLAIN output, measured latency
5. Migration: if adding an index, create the .up.sql file

Verification:
- All existing tests pass: go test ./... -race
- Integration tests pass: make api-test
- API response is identical (same shape, same data)
- EXPLAIN shows index usage (type: ref/eq_ref, not ALL)
```

## N+1 Detection

```bash
# Find potential N+1 patterns (loop + repo call)
grep -rn "for.*range" backend/internal/app/services/ --include="*.go" -A5 | grep -i "repo\.\|Find\|Get\|List"
```

## Index Check

```sql
-- Check existing indexes on a table
SHOW INDEX FROM timesheets;

-- Check query plan
EXPLAIN SELECT * FROM timesheets WHERE employee_id = ? AND date BETWEEN ? AND ?;
-- Look for: type=ALL (bad), type=ref (good), key=NULL (bad)
```
