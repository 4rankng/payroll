---
type: operations
title: Caching and Cache Invalidation
description: Redis cache usage, key strategy, the commit-after-invalidate invariant, and the audit-context interaction with the event-driven invalidation handler.
tags: [cache, redis, invalidation, transaction-manager, events]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-8428ae454c01a22e3f11de48
    resource: repo://backend/internal/app/services/infrastructure/cache_service.go
  - id: openwiki-source-2ca8c3b6dec9c23eff82e165
    resource: repo://backend/internal/app/services/infrastructure/import_progress_service.go
  - id: openwiki-source-e11d452ed15f5bbe429f4f93
    resource: repo://backend/internal/domain/services.go
  - id: openwiki-source-3d0b00c530be91b05c860949
    resource: repo://backend/internal/domain/transaction_manager.go
  - id: openwiki-source-ec61acad9618adbf769f1fc8
    resource: repo://backend/internal/infra/cache/nonce_store.go
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Caching and Cache Invalidation

Redis is used as the read-side cache for hot aggregates (banks, settings, transactions, timesheets, dashboard tiles) and as the backing store for several security tokens (OTP pending sessions, password-reset tokens, Zalo OA reset tokens, OIDC nonces). Two invariants are non-negotiable:

1. **Cache invalidation happens after the database transaction commits — never before.** (ADR-007)
2. **The invalidation handler subscribes to mutation events on Redis Streams, so invalidation runs strictly post-commit by construction.** (ADR-004)

These two together make "stale cache after rollback" structurally impossible rather than merely discouraged.

## Cache service

`domain.CacheServiceUseCase` (`repo://backend/internal/domain/services.go`) is the abstract interface every application service uses:

- `Get(ctx, key, dest)` / `Set(ctx, key, value, ttl)` / `Delete(ctx, key)` / `DeletePattern(ctx, pattern)`
- `Exists(ctx, key)` and `FlushAll(ctx)`
- Key-generation helpers: `GenerateBankCacheKey`, `GenerateSettingsCacheKey`, `GenerateTransactionCacheKey`, `GenerateTimesheetCacheKey`, `GenerateDashboardCacheKey`
- Operational: `GetRedisStats`

The implementation lives at `backend/internal/app/services/infrastructure/cache_service.go`. Centralizing key generation in the cache service means callers cannot accidentally construct a malformed key, and renaming a key space is a single-file change.

The cache handler at `backend/internal/transport/http/handlers/cache.go` exposes `DELETE /api/v1/cache` for the admin to flush everything in a recovery scenario (corrupt state from a bad import, manual rebuild after a config change, etc.). It is admin-gated by Casbin.

## Commit-after-invalidate

The canonical pattern, from ADR-007:

```go
err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
    return s.repo.Save(txCtx, entity)
})
if err == nil {
    s.cache.Invalidate(ctx, cacheKey)  // AFTER commit — safe
}
```

Invalidating inside the transaction opens a window: a concurrent reader sees a missing cache, hits MySQL, and populates the cache with pre-commit state. If the transaction then rolls back, the cache stays warm with phantom data until TTL expiry.

The domain layer ships `RegisterAfterCommit(ctx, fn)` (`repo://backend/internal/domain/transaction_manager.go#L32-L40`) as the canonical hook. It queues the callback inside the transaction; if no transaction is active, the callback fires immediately in a goroutine. `RunAfterCommitCallbacks` on `TransactionContext` is what the transaction manager invokes after a successful commit. Use it instead of remembering to invalidate outside the closure.

## Event-driven invalidation

For mutations that need to invalidate many cache keys at once (a bulk transfer completing, an employee being updated), the cache invalidation handler subscribes to Redis Streams via the event bus. The flow:

1. The application service publishes a mutation event (`EmployeeUpdated`, `BulkTransferCompleted`, `SettingsChanged`, etc.).
2. `cache_invalidation_handler.go` in `backend/internal/infra/events/` consumes the event and runs the corresponding cache deletes.
3. Because events are published post-commit (see [Transaction Manager and Outbox](../architecture/transaction-manager-and-outbox.md)), the invalidation runs strictly after the originating transaction has committed.

This is the structural enforcement of the invariant: there is no way for the invalidation to race the transaction because the event channel itself is the synchronization point.

## What belongs in the cache

- **Banks** — small, read-mostly reference data. TTL in hours; invalidate on admin edit.
- **Settings** — system configuration; aggressive invalidation on settings change, otherwise long TTL.
- **Transactions / Ledger summaries** — invalidated on every write to the underlying tables.
- **Timesheets** — invalidated on creation, update, approval, or bulk import.
- **Dashboard tiles** — invalidated when the underlying data changes; the dashboard handler also subscribes to forecast accuracy events to recompute the cash-readiness tile.
- **Import progress** (`import_progress_service.go`) — short-lived, scoped to a single asset ID, with 24h TTL safety net.

What does **not** go in the cache: anything that participates in the financial critical path. Wallet balance, ledger entries, and settlement rows are always read fresh from MySQL because the cost of a stale read exceeds the cost of a query.

## Security-token caches

`backend/internal/infra/cache/` holds dedicated stores that are not general-purpose cache:

- `nonce_store.go` — OIDC `id_token` nonce replay defense. Each login's nonce is consumed once; a second presentation is rejected.
- `otp_pending_store.go` — pending OTP login sessions.
- `password_reset_token_store.go` — password-reset tokens.
- `zalo_reset_store.go` — Zalo OA password-reset sessions.

Each store scopes its key namespace and TTL tightly. Reuse the cache service only for non-secret data; tokens go through these dedicated stores so a `FLUSHALL` on the general cache cannot accidentally wipe security state.

## N+1 defense

The 2026-07-04 lesson (`docs/lessons/2026-07-04-db-performance-n-plus-1-elimination.md`) is the canonical postmortem for the kind of regression the cache is supposed to prevent: a hot path that issued N+1 queries on cache miss. The fix combined `cache.Get` with batched lookups (`GetEmployeesByIDs` in the employee repository) so a single cache fill serves N reads. When adding a new cached aggregate, audit the miss path for N+1 risk before shipping.

## Operational guidance

- **TTL choice** — long enough that the cache earns its keep, short enough that a missed invalidation is bounded. Banks and settings can take hours; dashboard tiles minutes.
- **Pattern deletes** — use `DeletePattern` only when the key space is bounded and you understand the fan-out. A `bank:*` pattern delete is fine; a `user:*` pattern delete is not.
- **Observability** — `cache.GetRedisStats` returns hit/miss counts and memory pressure. Track these in the dashboard; a sudden drop in hit ratio usually means invalidation over-firing or a key-space migration.
- **Recovery** — `DELETE /api/v1/cache` (admin-only) is the catch-all when state diverges. It is also the right tool to run after restoring a database from backup, so subsequent reads do not see pre-restore cached aggregates.

## Related pages

- [Architecture Overview](../architecture/overview.md) — request lifecycle and middleware ordering.
- [Transaction Manager and Outbox](../architecture/transaction-manager-and-outbox.md) — the `RegisterAfterCommit` hook and event bus.
- [Auth, RBAC, and Casbin](./auth-rbac-and-casbin.md) — interaction with the audit context middleware that publishes events.
