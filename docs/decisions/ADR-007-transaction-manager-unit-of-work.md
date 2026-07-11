# ADR-007: Transaction Manager (Unit of Work) Pattern

**Date:** 2026-06-07
**Status:** Accepted

## Context

The payroll system performs multi-step database operations that must be atomic: e.g., creating a timesheet, updating a ledger entry, and publishing an event. If any step fails, all prior changes must roll back.

Additionally, cache invalidation must happen **after** the transaction commits — never before. If a cache is invalidated mid-transaction, a concurrent read could populate the cache with stale (pre-commit) data.

## Decision

Use a **Transaction Manager (Unit of Work)** pattern via `GormTransactionManager`, which implements the `domain.TransactionManager` interface.

### Usage

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.repo.Save(txCtx, entity); err != nil { return err }
    if err := s.otherRepo.Update(txCtx, other); err != nil { return err }
    return nil // auto-commit
})
// After this returns without error, cache invalidation can safely happen
```

Key files:
- `internal/infra/transaction/gorm_transaction_manager.go` (3.7K)
- `internal/domain/transaction_manager.go` — interface definition
- `gorm_transaction_manager_test.go` (7.4K) — commit/rollback tests

### Behavior

- Automatic rollback on panic via deferred recovery.
- The transaction context (`txCtx`) is propagated to all repository calls within the transaction.
- Repositories accept `*gorm.DB` or use the injected transaction manager — they detect whether they're in a transaction by checking the context.

### Cache Invalidation Rule

Cache invalidation must happen **after** `RunInTransaction` returns successfully:

```go
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    return s.repo.Save(txCtx, entity)
})
if err == nil {
    s.cache.Invalidate(ctx, cacheKey)  // AFTER commit — safe
}
```

Invalidating before or during the transaction risks a concurrent reader repopulating the cache with pre-commit data.

## Consequences

**Positive:**
- Multi-step operations are atomic — no partial writes on failure.
- Cache consistency is maintained (no stale cache after rollback).
- The transaction boundary is explicit in the service layer.
- Repositories don't need to know about transactions — they just use the context.

**Negative:**
- Long-running transactions can hold locks. Services must be designed to keep transactions short.
- The developer must remember to invalidate cache after the transaction, not inside it. Code review enforces this.

## Alternatives Considered

1. **Manual `db.Begin()` / `Commit()` / `Rollback()`** — Rejected. Error-prone; forgetting `Rollback()` on an error path leaks connections. The transaction manager handles this automatically.
2. **GORM's `db.Transaction(fn)`** — Used internally by `GormTransactionManager`, but the domain interface (`TransactionManager`) keeps the domain layer decoupled from GORM.
3. **Saga pattern** — Considered for cross-service transactions, but this is a monolith. Sagas add unnecessary complexity here.
