---
phase: 1
title: "Migration 088 — missing indexes"
status: pending
priority: P1
effort: "XS (1 migration file, 0 code changes)"
dependencies: []
---

# Phase 1: Migration 088 — missing indexes

## Overview
Pure additive SQL — 4 indexes that fix 2 High-severity full-scan endpoints
(audit_logs by entity_id, transactions reversed_transaction_id subquery) and 2
Medium hot-path queries (users.last_login, transactions status+created_at). Zero
application code changes, zero regression risk.

## Requirements
- Functional: 4 new indexes, each verified non-duplicative against `migrations/*.up.sql`.
- Non-functional: applies online (InnoDB `CREATE INDEX` → `ALGORITHM=INPLACE`,
  LOCK=NONE for these); safe on a live demo DB. Idempotent down migration.

## Architecture
All four are secondary indexes on growing tables where current queries do full
scans:
- `audit_logs.entity_id` — dropped in migration 008, never restored (034 revamp
  built `idx_action_entity (action, entity_type)` which can't serve a lone
  `entity_id = ?` predicate). Compound with `created_at` since the audit list
  always sorts by recency.
- `transactions.reversed_transaction_id` — FK exists but no index; the
  `NOT IN (SELECT reversed_transaction_id ...)` subquery scans the whole table
  twice per list/count request.
- `users.last_login` — queried as `IS NULL` (never-logged-in job) and date
  ranges (dashboard activity); zero index exists.
- `transactions (status, created_at)` — the common admin list/export combo;
  only single-column indexes exist today.

## Related Code Files
- **Create:** `backend/migrations/088_add_query_perf_indexes.up.sql`
- **Create:** `backend/migrations/088_add_query_perf_indexes.down.sql`

## Implementation Steps
1. Write `088_add_query_perf_indexes.up.sql`:
   ```sql
   -- 088: query-performance indexes surfaced by ck:debug 2026-07-04.
   -- All verified non-duplicative against the full migration chain.
   CREATE INDEX idx_audit_logs_entity_id          ON audit_logs (entity_id, created_at);
   CREATE INDEX idx_transactions_reversed_txn_id  ON transactions (reversed_transaction_id);
   CREATE INDEX idx_users_last_login              ON users (last_login);
   CREATE INDEX idx_transactions_status_created   ON transactions (status, created_at);
   ```
2. Write the down migration dropping all four (reverse order).
3. Mirror the comment style of 086/087 (one-line purpose + the audit reference).
4. Apply on local dev DB; run the affected endpoints and `EXPLAIN` to confirm
   the optimizer picks the new indexes:
   - `GET /audit/logs?entityId=1` → should show `key=idx_audit_logs_entity_id`
   - `GET /transactions` → the `NOT IN` subquery should use `idx_transactions_reversed_txn_id`

## Success Criteria
- [ ] Migration 088 applies cleanly on local dev DB.
- [ ] Down migration drops all four without error.
- [ ] `EXPLAIN` on the audit-logs-by-entity query shows the new index in use.
- [ ] `go build ./...` still clean (no code touched).
- [ ] `make api-test` green (no regressions from index-only change).

## Risk Assessment
- **Risk:** index bloat / write slowdown. **Mitigation:** 4 secondary indexes on
  tables with moderate write volume; InnoDB online add. Acceptable — these
  tables are read-heavy (audit_logs, transactions) or low-write (users).
- **Risk:** `idx_audit_logs_entity_id` partially overlaps `idx_action_entity`.
  **Mitigation:** they serve *different* predicates (entity_id-only vs
  action+entity_type); both justified. Keep both.
- **NOT IN → NOT EXISTS rewrite** is deferred to Phase 4 (code change); the
  index alone speeds up the inner subquery scan here.
