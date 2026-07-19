---
phase: 2
title: "Phase B: Batch unbatched GetByIDs/FindByCodes"
status: pending
priority: P1
effort: "M"
dependencies: []
---

# Phase B: Batch unbatched GetByIDs/FindByCodes

## Overview

Six repository methods issue a single unbounded `WHERE id IN (?)` query. Each is a latent production-breaking bug: at >1000 IDs MySQL rejects the packet, and at hundreds of IDs the preload-heavy variants (`transactions` with 4 preloads, `advance_payment_requests` with 3) produce large joined result sets. Add 1000-item batching, mirroring the existing correct pattern in `employee_repository_crud.go:35`.

## Background / why

The audit (partner dashboard burst) found these methods unbatched:

| File:Line | Method | Preloads | Worst caller |
|---|---|---|---|
| `transaction_repository.go:110` | `GetByIDs(ids []uint)` | Asset, Creator, LedgerEntries, Settlements | settlement linker (per-upload) |
| `project_repository.go:49` | `GetByIDs(ids []uint)` | none | `payroll/service.go:473` (bank-transfer-histories) |
| `user_repository.go:81` | `GetByIDs(ids []uint)` | none | audit/dashboard converters |
| `advance_payment_request_repository.go:388` | `GetByIDs(ids []uint64)` | Employee, Bank, FlexPayRequest | settlement upload |
| `advance_payment_request_repository.go:410` | `GetByIDsLean(ids []uint64)` | none | wallet settlement worker |
| `transaction_code_repository.go:63` | `FindByCodes(codes []string)` | none | `payroll/service.go:339` |

The existing `BatchProcessor.ProcessInBatches` (`common/batch_processor.go`) is reflection-based and awkward for "fetch and accumulate into a typed slice". A typed generic helper will read more clearly and reuse cleanly across all six sites.

## Architecture

```
common/batch_processor.go (or new chunk.go)
  └─ func Chunk[T any](ctx, db, ids []uint, size int, fn func(tx, batch) ([]T, error)) ([]T, error)
  └─ func ChunkStrings[T any](ctx, db, keys []string, size int, fn) ([]T, error)

transaction_repository.go        GetByIDs         ──► Chunk[Transaction] with preloads inside fn
project_repository.go            GetByIDs         ──► Chunk[Project] building map[uint]*Project
user_repository.go               GetByIDs         ──► Chunk[User] building map[uint]*User
advance_payment_request_repository.go GetByIDs    ──► Chunk[AdvancePaymentRequest] with preloads
advance_payment_request_repository.go GetByIDsLean ──► Chunk[AdvancePaymentRequest] no preloads
transaction_code_repository.go   FindByCodes     ──► ChunkStrings[TransactionCode]
```

## Requirements

- **Functional:** every method returns the same rows it would have returned with a single `WHERE id IN (?)`, regardless of input size. Order need not match the input order (callers build maps; verify none rely on order).
- **Non-functional:** each individual query ≤ 1000 IDs; total result accumulated in memory. No behavior change for small inputs (≤1000 IDs → one query, identical to today).

## Related code files

- Modify: `backend/internal/infra/persistence/common/batch_processor.go` — add `Chunk[T]` and `ChunkStrings[T]` helpers.
- Modify: `backend/internal/infra/persistence/transaction_repository.go` — `GetByIDs`.
- Modify: `backend/internal/infra/persistence/project_repository.go` — `GetByIDs`.
- Modify: `backend/internal/infra/persistence/user_repository.go` — `GetByIDs`.
- Modify: `backend/internal/infra/persistence/advance_payment_request_repository.go` — `GetByIDs`, `GetByIDsLean`.
- Modify: `backend/internal/infra/persistence/transaction_code_repository.go` — `FindByCodes`.
- Modify: `backend/internal/infra/persistence/query_builders/employee_query_builder.go` — make the silent-truncation bug fail loudly (B3).

## Implementation steps

1. **Add typed helpers** in `common/batch_processor.go`:
   ```go
   const defaultChunkSize = 1000 // safe under MySQL's max_allowed_packet for typical IDs

   // Chunk runs fn over ids in chunks of size, concatenating the returned slices.
   // Use for "SELECT ... WHERE id IN (?)" patterns.
   func Chunk[T any](ctx context.Context, db *gorm.DB, ids []uint,
       size int, fn func(tx *gorm.DB, batch []uint) ([]T, error)) ([]T, error)

   // ChunkStrings is the string-keyed analogue (e.g. transaction codes).
   func ChunkStrings[T any](ctx context.Context, db *gorm.DB, keys []string,
       size int, fn func(tx *gorm.DB, batch []string) ([]T, error)) ([]T, error)
   ```
   Both short-circuit on empty input and default `size` to `defaultChunkSize` when ≤ 0.

2. **`transaction_repository.go:110`** — replace the single `Find` with `Chunk[domain.Transaction]`, moving the `Select(allFields).Preload(...)` chain inside `fn`. Preserve the post-fetch `calculateSettledAmount` loop.

3. **`project_repository.go:49`** — `Chunk[domain.Project]`, accumulate into the existing `map[uint]*domain.Project` return.

4. **`user_repository.go:81`** — `Chunk[domain.User]`, accumulate into `map[uint]*domain.User`.

5. **`advance_payment_request_repository.go`** — both `GetByIDs` and `GetByIDsLean` use `Chunk[domain.AdvancePaymentRequest]`; keep the preload difference between them.

6. **`transaction_code_repository.go:63`** — `FindByCodes` uses `ChunkStrings[domain.TransactionCode]`.

7. **B3 — Make `employee_query_builder.go:230` truncation fail loudly:** the `if len(ids) > 1000` branch currently returns a query for the first 1000 IDs silently. Change it to return an error (or log-and-continue with an explicit `error` return) so a future direct caller does not get silent data loss. The existing `employee_repository_crud.go` caller is unaffected because it pre-chunks to 1000.

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] Existing unit/integration tests for the 6 repos pass unchanged
- [ ] New table-driven test: each method with a 2500-ID input returns the same rows as the union of 3×1000-ID calls
- [ ] `BuildGetByIDsQuery` returns an error (not silent truncation) when given >1000 IDs directly

## Risk assessment

- **Result ordering:** GORM `Find` with `WHERE id IN (?)` does not guarantee order. Audit each caller — most build maps, but verify no caller assumes input-order output. The transaction repo's post-fetch loop is order-independent.
- **Preload fan-out:** batching preloads is safe — each batch is an independent query with its own preload queries; results merge naturally.
- **Generic helper compilation:** Go 1.22+ supports the generics fine; ensure the `common` package has no import cycle with repos (it must not import domain types — the generic `[T any]` keeps it domain-agnostic).
- **`FindByCodes` contract:** callers may dedupe codes before passing; chunking preserves any duplicates the single query would have returned.
