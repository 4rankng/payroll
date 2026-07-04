# Phase 1 Money-Integrity Code Review — H5 / H6 / H7

**Scope:** FlexPay settlement per-iteration transaction + idempotency (H5), RetryDisbursement in-flight dedup (H6), MarkReconciled status precondition (H7).
**Repo:** `/Users/dev/Documents/projects/payroll/backend`
**Branch:** `main` (working tree; uncommitted)
**Reviewer posture:** rulebook-first, evidence-based. All claims below are grep/read-verified against the actual codebase, not the plan text.
**Verdict:** **DONE_WITH_CONCERNS** — money-integrity invariants are sound; concerns are operational (migration pairing, log noise, a narrow race window that is documented and acceptable), not correctness defects.

---

## 1. Overall Assessment

The change set is well-scoped and the load-bearing money-integrity claims hold under verification. The per-iteration transaction in H5 correctly participates in the surrounding `db.Transaction` via `domain.WithTransactionContext`, the H7 status-precondition allowlist exactly mirrors the `ReconcilePayment` switch, and H6's nil-safe in-flight guard plus `asynq.TaskID` dedup compose correctly with the existing worker-side double-pay guard. Build is clean, `go vet` clean, all new tests pass. The only failing tests in the repo are the pre-existing `TestCreateWithBudgetCheck_*` FK fixture issue (employee_id 999901/999902), confirmed unrelated.

The background design decisions (asynq.TaskID over UniqueTTL; SHA-256 of sorted request-ID set; per-iteration tx; settleOneTxn mirroring `ApplySettlement`) are the right calls and I am not relitigating them — the verification below confirms they hold.

---

## 2. Verification Matrix (all 7 items requested)

### 2.1 H5 — per-iteration tx participates in `db.Transaction` via txCtx — **VERIFIED**

`settleOneTxn` (`flex_pay_settlement_service.go:223-293`) wraps the work in `s.db.WithContext(ctx).Transaction(func(tx *gorm.DB)...)`, then constructs `txCtx := domain.WithTransactionContext(ctx, &domain.TransactionContext{TX: tx, IsTransactional: true})` and passes `txCtx` to every repo call. Each of the three repos resolves the tx from context:

| Repo / method | Resolves tx from ctx? | Evidence |
|---|---|---|
| `transactionRepository.GetByIDForUpdate` | yes — via `getDB(ctx)` | `transaction_repository.go:61` calls `r.getDB(ctx)`; `getDB` at `:499-501` returns `txCtx.TX.WithContext(ctx)` when ctx carries a tx |
| `transactionRepository.Update` | yes — via `getDB(ctx)` | `transaction_repository.go:226-232`: `db := r.getDB(ctx); db.Save(txn)` |
| `LedgerEntryRepository.CreateTransaction` | yes — explicit branch | `ledger_repository_crud.go:39-41`: `if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil { return r.createEntriesInTx(ctx, txCtx.TX, entries) }` |

This matches the `ApplySettlement` pattern in `internal/infra/events/settlement_event_handler.go:214` (same `db.Transaction` + `WithTransactionContext` + `GetByIDForUpdate` shape). The `FOR UPDATE` row lock in `GetByIDForUpdate` (`clause.Locking{Strength: "UPDATE"}`) serializes concurrent settlements of the same transaction ID, and the deadlock-retry loop (`isDeadlockErr` + 3 attempts with exponential backoff + jitter) handles the cross-transaction deadlock case. **Sound.**

### 2.2 H5 — error propagation: no swallow path returns `Success:true` on a write failure — **VERIFIED**

Tracing `ProcessSettlementFile` end-to-end:

- **Per-iteration failures** are captured into `failedTxnIDs` (`:166`) — the txn rolls back, the loop continues. If `len(failedTxnIDs) > 0`, the result is `Success: false` (`:182`) and the `settlement_uploads` row is **NOT** written (the Create at `:194` is gated behind the `failedTxnIDs == 0` branch). Correct.
- **Ledger/txn write failure inside `settleOneTxn`** returns a wrapped error from the `Transaction` callback, which rolls back the entire iteration and surfaces as a `failedTxnID`. No swallow.
- **Idempotency-row Create failure on full success** (`:194-202`) is the one place a write error is logged-and-continued. This is the documented, correct tradeoff: the books are already correct, and a re-upload is a no-op because the per-iteration status precondition skips already-settled txns. **Acceptable** — explicitly called out in the inline comment at `:190-192`.

One observation worth noting (not a defect): `MarkReceivableSettled` (`:128`) runs **outside** the per-iteration tx, before the txn loop starts. So on a partial-failure re-upload, `receivable_settled_at` is already set for all request IDs and the `WHERE receivable_settled_at IS NULL` guard in the repo (`advance_payment_request_repository.go:506`) makes the re-run a no-op for that column. This is consistent with the "reprocess only failed records" contract — the receivable flag is per-request, the ledger write is per-transaction, and they decouple cleanly.

### 2.3 H6 — guard + TaskID: ordering, 409 mapping, nil-safety — **VERIFIED with one noted race (acceptable)**

`retry_disbursement_handler.go`:

- **Nil-safety** (`:74`): `if h.txWalletPaymentRepo != nil` — the guard is skipped silently if the repo is nil rather than panicking. Correct.
- **409 mapping**: in-flight guard → `response.Conflict` (`:83`); `asynqlib.ErrTaskIDConflict` → `response.Conflict` (`:120`); other enqueue errors → 500 (`:126`). Correct differentiation.
- **Ordering — race window (documented, acceptable)**: Step 3 (`:48-54`) resets `FAILED → APPROVED` on the advance_payment_request **before** the in-flight check at Step 5.5. Two concurrent retry clicks on the same FAILED request could both pass the status check (`IsFailed()` true for both), both reset to APPROVED (idempotent — same target state), and then race on the in-flight check. The loser of that race returns 409, but the advance_payment_request row is already left in APPROVED. This is fine because (a) the worker-side `HasPendingForRecipient` + `request_id` unique index is the actual double-pay guard, and (b) APPROVED is the correct state for a request awaiting disbursement regardless. The TaskID (`disbursement:execute:%d`, `:112`) ensures only one task is ever pending per request ID. **No double-pay risk.**

If you want to tighten this further, move the in-flight check before the status reset (so a FAILED request with a stuck wallet_payment row stays FAILED until reconciliation clears it). Not required for correctness; purely hygiene.

### 2.4 H7 — precondition allowlist matches `ReconcilePayment` switch — **VERIFIED**

`reconcilablePriorStates` in `tx_wallet_payment_repository.go` (just above `MarkReconciled`):

```go
var reconcilablePriorStates = []domaintx.State{
    domaintx.StateFailed,
    domaintx.StateCompleted,
    domaintx.StateAuthorised,
    domaintx.StateVerified,
}
```

`ReconcilePayment` (`wallet_payment_service.go:711-780`) switch cases: `StateFailed` (`:720`), `StateCompleted` (`:743`), `StateAuthorised, StateVerified` (`:753`). Default branch (`:774`) logs-and-skips everything else. **Exact match.** `pending` and `reversed` rows are correctly rejected by the `WHERE status IN ?` precondition (`MarkReconciled` body). `failed → completed` still works because `StateFailed` is in the allowlist and the new status (`StateCompleted`) is passed as a parameter, not filtered. The H7 unit test (`provider_transaction_repository_test.go:347`) pins all six states explicitly.

### 2.5 Migration 085 — **VERIFIED with two concerns**

`085_settlement_uploads.up.sql`:

- `file_hash CHAR(64) NOT NULL` — matches SHA-256 hex output (64 chars). Correct.
- `uploaded_at DATETIME(3)` — matches the rest of the codebase's timestamp precision.
- `request_ids_json JSON NULL` — nullable, matches the domain `gorm:"type:json"` tag (which GORM maps to NULL on empty — acceptable).
- `UNIQUE KEY uq_settlement_uploads_file_hash (file_hash)` — enforces idempotency at the DB level even if two uploads race past the `GetByFileHash` check. Correct.
- `CREATE TABLE IF NOT EXISTS` — idempotent re-run. Good.

**Concern A (operational, blocking-for-process if your runbook requires it):** there is **no paired `085_settlement_uploads.down.sql`**. The convention in `migrations/` is paired files (see 081/084). For an additive audit table this is low-risk, but golang-migrate will refuse a `down` invocation if the file is missing, and consistency matters. Recommend adding a one-line `DROP TABLE IF EXISTS settlement_uploads;` down file.

**Concern B (cosmetic):** `settlement_uploads.id BIGINT UNSIGNED AUTO_INCREMENT` with no `NOT NULL` on `request_ids_json` is fine, but the domain struct's `RequestIDsJSON string` will persist as `""` (empty string) rather than NULL when `requestIDs` is empty. MySQL JSON column accepts an empty string as invalid JSON and will error. In practice `requestIDs` is always non-empty at the Create call site (the early return at `:74-82` skips empty), so this is theoretical — but worth a defensive `if requestIDsJSON == "" { requestIDsJSON = "[]" }` or a nullable type if you want belt-and-suspenders.

### 2.6 Public-contract break — **VERIFIED: none**

- `NewFlexPaySettlementService` signature changed (added `db *gorm.DB` and `uploadRepo domain.SettlementUploadRepository`). Grep confirms exactly **one** call site: `internal/app/bootstrap/services/init.go:318`. Updated. No other callers exist.
- `ProcessSettlementFile(ctx, *excelize.File)` signature is **unchanged** — the hash is computed inside. The three callers (`recon_settlemetn_handler.go:58`, plus the two unrelated `ProcessSettlementFileWithDedup` paths in `settlement/upload_service.go` which belong to a different service) are unaffected. Confirmed.
- `WalletPaymentRepository` interface gained `HasNonTerminalByEntityID`. All implementations updated: production `TxWalletPaymentRepository` plus the two test fakes (`fakeProviderTxRepo`, `fakeOnePayFeeWalletPayments`). Build is clean, so no other implementor exists.

### 2.7 Test fakes returning `false, nil` — **VERIFIED acceptable**

Both fakes (`provider_transaction_service_test.go:130`, `onepay_fee_import_service_test.go:246`) stub `HasNonTerminalByEntityID` to `return false, nil`. Neither test exercises the in-flight guard path — they test fee calculation and stale-authorised polling respectively, where the new method is never called. The real behavior is pinned by `TestRepository_HasNonTerminalByEntityID` in `provider_transaction_repository_test.go:347`, which asserts all six states plus NULL entity_id plus cross-entity isolation. **Acceptable.**

---

## 3. Critical Issues

None. Money-integrity invariants are sound.

## 4. High Priority

None.

## 5. Medium Priority

| # | Finding | Evidence | Fix |
|---|---|---|---|
| M1 | **Missing `085_settlement_uploads.down.sql`** — breaks convention (081/084 are paired) and golang-migrate `down` invocations | `ls migrations/085*` shows only `.up.sql` | Add `migrations/085_settlement_uploads.down.sql` with `DROP TABLE IF EXISTS settlement_uploads;` |
| M2 | **GORM logger logs `record not found` at the repo's `GetByFileHash` miss path** — visible as scary red log lines during normal first-upload operation | `settlement_upload_repository.go:29` First → ErrRecordNotFound → returns nil,nil; GORM's default logger emits the SQL+error before the handler branches on it | Either suppress with `.Session(&gorm.Session{Logger: r.DB.Logger.LogMode(logger.Silent)})` for this specific First, or accept the noise. Same pattern exists in `tx_wallet_payment_repository.go:48`. Cosmetic. |

## 6. Low Priority

| # | Finding | Evidence |
|---|---|---|
| L1 | **H6 status-reset-before-check race window** — two concurrent retry clicks on a FAILED request both reset to APPROVED, then one wins the in-flight guard. The loser returns 409 but leaves the row in APPROVED. No double-pay risk (worker-side `HasPendingForRecipient` + `request_id` unique index is the real guard). | `retry_disbursement_handler.go:48-54` (reset) precedes `:74-86` (guard). Documented inline at `:69-73`. |
| L2 | **`request_ids_json` empty-string edge case** — domain `RequestIDsJSON string` would write invalid JSON (`""`) if the slice were ever empty at Create time. Currently impossible (early return at `:74-82`), but a nullable type or `"[]"` default would harden it. | `settlement_upload.go:17` + `flex_pay_settlement_service.go:193` |
| L3 | **`isDeadlockErr` matches on substring `"1213"`** — brittle if a future MySQL version changes error numbers, but pragmatic and matches existing project conventions. | `flex_pay_settlement_service.go:310-313` |

## 7. Edge Cases Found by Scout

- **Concurrent settlement of the same txn ID across two upload sessions** — `GetByIDForUpdate` with `FOR UPDATE` inside the per-iteration tx serializes these correctly. The loser blocks until the winner commits, then sees `Status != pending` and returns nil (idempotent skip). Verified.
- **Re-upload after partial failure** — `MarkReceivableSettled`'s `WHERE receivable_settled_at IS NULL` guard plus the per-iteration `txn.Status != pending` skip compose correctly: already-settled records are no-ops, failed ones reprocess. Verified.
- **`HasNonTerminalByEntityID` with NULL `entity_id` rows** — SQL equality (`entity_id = ?`) does not match NULL, so NULL-linked in-flight rows are correctly ignored. Pinned by the H6 test at `provider_transaction_repository_test.go:376-386`.
- **`ErrTaskIDConflict` only fires while the task is pending/retrying/scheduled** — once the task reaches a terminal state (success/failed/deleted), the ID is released and a later retry enqueues cleanly. This matches the inline comment at `retry_disbursement_handler.go:106-110`. Verified against asynq semantics.

## 8. Positive Observations (risk calibration only)

- The per-iteration tx design (vs. one tx wrapping the whole loop) is the right call: a single bad record cannot block the rest of the batch, and the per-record status precondition makes re-runs idempotent.
- `reconcilablePriorStates` is sourced from and cross-referenced to `ReconcilePayment` in its doc comment — good traceability for future maintainers.
- The H6 unit test is exhaustive: all six states, NULL entity_id, cross-entity isolation, empty table. This is what "pinning" a precondition should look like.
- `settleOneTxn` mirrors `ApplySettlement` exactly — no parallel reimplementation of the settlement ledger pattern.

## 9. Recommended Actions

1. **(M1)** Add `migrations/085_settlement_uploads.down.sql` (`DROP TABLE IF EXISTS settlement_uploads;`) for convention consistency.
2. **(M2, optional)** Silence GORM's `record not found` logging on the `GetByFileHash` miss path if log noise is a concern.
3. **(L2, optional)** Harden `request_ids_json` against the empty-slice edge case if you want defense in depth.
4. No code changes required for correctness. H5/H6/H7 are production-ready from a money-integrity standpoint.

## 10. Metrics

- Type Coverage: N/A (Go)
- Test Coverage: H5 service + H6/H7 repo tests pass; new tests cover the documented invariants
- Linting Issues: 0 (`go vet` clean, `go build ./internal/...` exit 0)
- Pre-existing failures confirmed unrelated: `TestCreateWithBudgetCheck_*` (FK fixture on employee_id 999901/999902)

## 11. Unresolved Questions

None. All seven verification points are answered with file:line evidence.

---

**Status: DONE_WITH_CONCERNS**
**Summary:** H5/H6/H7 money-integrity invariants are sound and verified against the codebase; the only actionable item is adding the paired `085_settlement_uploads.down.sql` (M1) for migration-convention consistency — no correctness defects found.
**Concerns:** M1 (missing down migration), M2 (GORM log noise on expected miss path), L1 (documented H6 reset-before-check race, no double-pay risk).
