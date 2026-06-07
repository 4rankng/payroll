<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# transaction — Database Transaction Management

## Purpose
Implements the transaction management abstraction using GORM. Provides a `GormTransactionManager` that implements the `domain.TransactionManager` interface and a `GormUnitOfWork` for the unit of work pattern. Enables application services to execute multi-step database operations atomically without coupling to GORM directly.

## Key Files
| File | Description |
|------|-------------|
| `gorm_transaction_manager.go` | Transaction manager — `Begin()`, `Commit()`, `Rollback()` wrappers around GORM DB transactions. Implements `domain.TransactionManager` and `domain.UnitOfWork` interfaces (3.7K) |
| `gorm_transaction_manager_test.go` | Transaction manager tests — verifies commit, rollback, and nested transaction behavior (7.4K) |

## Subdirectories
_None_

## For AI Agents

### Working In This Directory
- Implements interfaces from `domain/transaction_manager.go`
- Use `domain.TransactionManager` in application services, never reference GORM directly
- Unit of work pattern: operations within `RunInTransaction(ctx, fn)` are atomic
- Automatic rollback on panic via deferred recovery
- Cache invalidation should happen **after transaction commit**, not before

### Testing Requirements
- Tests verify commit, rollback, and error scenarios
- Transaction behavior tested in integration tests

### Common Patterns
```go
// Using transaction manager in application service
err := s.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.repo.Save(txCtx, entity); err != nil {
        return err // auto-rollback
    }
    if err := s.otherRepo.Update(txCtx, other); err != nil {
        return err // auto-rollback
    }
    return nil // auto-commit
})

// IMPORTANT: Cache invalidation AFTER commit
s.cache.Delete(key) // outside transaction
```

## Dependencies

### Internal
- `internal/domain` — `TransactionManager` and `UnitOfWork` interfaces

### External
- `gorm.io/gorm` — underlying transaction mechanism

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
