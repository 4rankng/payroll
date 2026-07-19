---
phase: 2
title: 'Phase B: Batch unbatched GetByIDs/FindByCodes'
status: completed
priority: P1
effort: M
dependencies: []
---

# Phase B: Batch unbatched GetByIDs/FindByCodes

## Overview

Six repository methods issue a single unbounded `WHERE id IN (?)` query. Each is a latent production-breaking bug: at >1000 IDs MySQL rejects the packet, and at hundreds of IDs the preload-heavy variants produce large joined result sets. Add 1000-item batching, mirroring the existing correct pattern in `employee_repository_crud.go:35`.

> **Red-team revision (findings F6, F8):**
> - **F6 (Critical):** `transactionRepository.GetByIDs` uses `r.db`, not `r.getDB(ctx)` — it ignores transaction context today. Chunking widens the MVCC snapshot window mid-transaction. **Precondition:** fix `GetByIDs` to honor `getDB(ctx)` BEFORE batching so all chunks share the caller's snapshot.
> - **F8 (Medium):** the original B7 ("make `BuildGetByIDsQuery` truncation fail loudly") would change the return signature from `*gorm.DB` to `(*gorm.DB, error)`, breaking the only existing caller at compile time. **Dropped.** Keep silent truncation; add a metrics counter instead.

## Background / why

The audit found these methods unbatched:

| File:Line | Method | Preloads | Worst caller | Tx-aware? |
|---|---|---|---|---|
| `transaction_repository.go:110` | `GetByIDs(ids []uint)` | Asset, Creator, LedgerEntries, Settlements | settlement linker (per-upload, **inside tx**) | ❌ uses `r.db` |
| `project_repository.go:49` | `GetByIDs(ids []uint)` | none | `payroll/service.go:473` | (verify) |
| `user_repository.go:81` | `GetByIDs(ids []uint)` | none | audit/dashboard | (verify) |
| `advance_payment_request_repository.go:388` | `GetByIDs(ids []uint64)` | Employee, Bank, FlexPayRequest | settlement upload | ✅ uses `getDB(ctx)` |
| `advance_payment_request_repository.go:410` | `GetByIDsLean(ids []uint64)` | none | wallet settlement worker | ✅ uses `getDB(ctx)` |
| `transaction_code_repository.go:63` | `FindByCodes(codes []string)` | none | `payroll/service.go:339` | (verify) |

The existing `BatchProcessor.ProcessInBatches` (`common/batch_processor.go`) is reflection-based and awkward for "fetch and accumulate into a typed slice". A typed generic helper reads more clearly and reuses across all six sites.

## Architecture

```
common/batch_processor.go (or new chunk.go)
  └─ func Chunk[T any](ctx, db *gorm.DB, ids []uint, size int, fn func(tx, batch) ([]T, error)) ([]T, error)
  └─ func ChunkStrings[T any](ctx, db *gorm.DB, keys []string, size int, fn) ([]T, error)

PRECONDITION (F6): transaction_repository.GetByIDs switches from r.db → r.getDB(ctx)
                   before batching, so tx-scoped callers share one snapshot across chunks.

transaction_repository.go        GetByIDs         ──► Chunk[Transaction] with preloads inside fn
project_repository.go            GetByIDs         ──► Chunk[Project] building map[uint]*Project
user_repository.go               GetByIDs         ──► Chunk[User] building map[uint]*User
advance_payment_request_repository.go GetByIDs    ──► Chunk[AdvancePaymentRequest] with preloads
advance_payment_request_repository.go GetByIDsLean ──► Chunk[AdvancePaymentRequest] no preloads
transaction_code_repository.go   FindByCodes     ──► ChunkStrings[TransactionCode]
```

## Requirements

- **Functional:** every method returns the same rows it would have returned with a single `WHERE id IN (?)`, regardless of input size. Order need not match input order (callers build maps — verify per method).
- **Non-functional:** each individual query ≤ 1000 IDs; total result accumulated in memory. No behavior change for small inputs (≤1000 IDs → one query, identical to today). **Transaction-scoped callers see a consistent snapshot across all chunks (F6).**

## Related code files

- Modify: `backend/internal/infra/persistence/common/batch_processor.go` — add `Chunk[T]` and `ChunkStrings[T]` helpers.
- Modify: `backend/internal/infra/persistence/transaction_repository.go` — **first** switch `GetByIDs` from `r.db` to `r.getDB(ctx)` (F6 precondition), **then** batch.
- Modify: `backend/internal/infra/persistence/project_repository.go` — `GetByIDs`.
- Modify: `backend/internal/infra/persistence/user_repository.go` — `GetByIDs`.
- Modify: `backend/internal/infra/persistence/advance_payment_request_repository.go` — `GetByIDs`, `GetByIDsLean`.
- Modify: `backend/internal/infra/persistence/transaction_code_repository.go` — `FindByCodes`.
- **NOT modified (red-team F8):** `query_builders/employee_query_builder.go` — the silent-truncation bug stays as-is. Add a structured log + metrics counter on the truncation branch so ops can detect overflows without breaking the `*gorm.DB` return signature.

## Implementation steps

1. **Add typed helpers** in `common/batch_processor.go`:
   ```go
   const defaultChunkSize = 1000 // safe under MySQL's max_allowed_packet for typical IDs

   // Chunk runs fn over ids in chunks of size, concatenating the returned slices.
   // `db` MUST be the context-and-transaction-aware session handle (r.getDB(ctx)),
   // not bare r.db — otherwise chunks run outside the caller's transaction and see
   // inconsistent snapshots (red-team F6).
   func Chunk[T any](ctx context.Context, db *gorm.DB, ids []uint,
       size int, fn func(tx *gorm.DB, batch []uint) ([]T, error)) ([]T, error)

   // ChunkStrings is the string-keyed analogue (e.g. transaction codes).
   func ChunkStrings[T any](ctx context.Context, db *gorm.DB, keys []string,
       size int, fn func(tx *gorm.DB, batch []string) ([]T, error)) ([]T, error)
   ```
   Both short-circuit on empty input and default `size` to `defaultChunkSize` when ≤ 0. `common` has no import cycle risk — it already imports only `context, fmt, reflect, observability, slog, gorm` (verified, no `domain`/`repositories` imports).

2. **F6 precondition — fix `transaction_repository.go:110`:** change `r.db.WithContext(ctx)` → `r.getDB(ctx)`. Verify `getDB` exists on the receiver (it does — `transaction_repository.go:497-504`, used by `IncrementAmount`, `UpdateStatus`). This makes `GetByIDs` honor `domain.TransactionContext` for the first time. Without this, chunking a settlement-validation read mid-transaction would widen the snapshot window (red-team F6).

3. **Batch `transaction_repository.GetByIDs`:** replace the single `Find` with `Chunk[domain.Transaction]`, moving the `Select(allFields).Preload(...)` chain inside `fn`. Preserve the post-fetch `calculateSettledAmount` loop — it accumulates per-transaction by ID, so order-independence holds (red-team M2: the plan now cites this explicitly).

4. **Batch the remaining 5 methods** (`project`, `user`, `advance_payment_request` ×2, `transaction_code`). Each: pass `r.getDB(ctx)` (or equivalent tx-aware handle) to `Chunk`; accumulate into existing map/slice return types.

5. **F8 — observability for `employee_query_builder.go:230` truncation (NOT a signature change):**
   - Keep the `if len(ids) > batchSize` branch returning `query.Where("id IN ?", ids[:batchSize])` (do NOT change the `*gorm.DB` return type).
   - Add `observability.GetLogger().Warn("employee_query_builder_truncated_ids", "requested", len(ids), "cap", batchSize)` and `observability.IncrementCounter("employee_query_builder_truncation_total")` on the truncation branch.
   - Ops can alert on the counter; future direct callers who pass >1000 IDs are detected without a compile break.

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] Existing unit/integration tests for the 6 repos pass unchanged
- [ ] New table-driven test: each method with a 2500-ID input returns the same rows as the union of 3×1000-ID calls
- [ ] **F6 regression test (new):** `transactionRepository.GetByIDs` called inside `db.Transaction(...)` while another goroutine mutates a chunk-2 row returns a consistent snapshot (all chunks see the same MVCC view)
- [ ] **F6 regression test:** `GetByIDs` called with a `domain.TransactionContext` uses the transaction's `*gorm.DB` (assert via a test double that the tx handle reaches `Chunk`)
- [ ] `employee_query_builder` truncation emits a warning log + metric (no behavior change)

## Risk assessment

- **Transaction snapshot widening (red-team F6 — accepted):** the precondition (switch to `getDB(ctx)`) closes this. The regression test asserts it.
- **Result ordering:** GORM `Find` with `WHERE id IN (?)` does not guarantee order. Audited: `transaction_repository.calculateSettledAmount` accumulates by ID (order-independent); the other 5 methods build maps. None rely on input-order output.
- **Preload fan-out:** batching preloads is safe — each batch is an independent query with its own preload queries; results merge naturally.
- **Generic helper compilation:** Go 1.22+ (verify `go.mod` version before implementing). The `[T any]` constraint keeps `common` domain-agnostic — no import cycle.
- **`BuildGetByIDsQuery` truncation (red-team F8 — accepted):** keeping the silent truncation with metrics is safer than a signature change that breaks the only caller. Ops visibility is the goal; future callers are alerted via the counter.
