# DB Performance: N+1 Query Elimination

**Date:** 2026-07-04
**Source:** Commits `ba953e3`, `712a551`, `704d085`
**Tags:** [performance, database, n-plus-1, indexing, gorm]

## Context

Several backend endpoints were slow under load. Investigation revealed N+1 query patterns: for each item in a list, a separate query was issued to fetch related data. With 100+ employees, this meant 100+ additional queries per request.

Additionally, some queries were doing full table scans because indexes were missing on commonly-filtered columns.

## Decision / Outcome

Three rounds of N+1 elimination and index optimization:

### Round 1 (commit `ba953e3` — Phase 4)
- Replaced loop-based repository calls with batch queries.
- Added proper pagination (`LIMIT`/`OFFSET`) on all list endpoints.
- Added projections (`Select` only needed columns instead of full rows).
- Replaced `NOT IN` subqueries with `NOT EXISTS` for better performance.
- Removed dead code that was making unnecessary queries.

### Round 2 (commit `712a551`)
- Killed 3 specific N+1 loops:
  - Payroll report: was querying per-employee payrates in a loop.
  - Loan list: was querying per-loan repayment schedules in a loop.
  - Loan schedules: was querying per-schedule interest calculations in a loop.

### Round 3 (commit `704d085` — Migration 088)
- Added 4 missing query indexes identified by `EXPLAIN` analysis.

## Lesson

**N+1 is the most common performance killer in ORM-based applications.** The pattern is seductive because it's easy to write:

```go
// N+1 — looks clean, performs terribly
for _, emp := range employees {
    projects := repo.FindByEmployeeID(emp.ID) // 1 query per employee
}
```

The fix is always one of:
1. **Batch query:** Fetch all related data in one query using `WHERE id IN (...)`.
2. **Preload:** Let GORM preload relations: `db.Preload("Projects").Find(&employees)`.
3. **Join:** Use a SQL join to fetch everything in one query.

**Generalized rules:**

1. **Any loop containing a repository call is suspicious.** Review it immediately — can the data be fetched in a single batch query?

2. **`EXPLAIN` is your friend.** Before optimizing, run `EXPLAIN` on the query. If `type` is `ALL` (full table scan), you need an index. If `rows` is much larger than the result set, the query is inefficient.

3. **Projections matter.** Fetching `SELECT *` when you only need `id, name` wastes memory and bandwidth. Use `Select("id, name")`.

4. **`NOT EXISTS` beats `NOT IN`.** For anti-joins ("find employees with no timesheet"), `NOT EXISTS` performs better than `NOT IN`, especially on large datasets.

5. **Indexes are migrations.** Adding an index requires a `.up.sql` migration file. Name it descriptively (`add_query_perf_indexes`).

6. **Performance optimization is iterative.** This took 3 rounds over multiple days. Each round used `EXPLAIN` and metrics to identify the next bottleneck. Don't try to fix everything at once.

## References

- Commit `ba953e3`: perf(db): Phase 4 — N+1, proper pagination, projections, NOT EXISTS, dead-code
- Commit `712a551`: perf(db): kill 3 N+1 loops — payroll-report, loan-list, loan-schedules
- Commit `704d085`: perf(db): migration 088 — add 4 missing query indexes
- Migration 088: `add_query_perf_indexes`
- [Performance Standards](../standards/performance.md)
- [Performance Optimization Prompt](../prompt-library/performance-optimization.md)
