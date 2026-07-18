---
phase: 2
title: "Backend Pipeline & Ledger"
status: pending
priority: P1
dependencies: [1]
---

# Phase 2: Backend Pipeline & Ledger

## Overview

The orchestration service: receives the uploaded bytes → parses (Phase 1) → duplicate-checks by content hash + per-row VFIC code → creates `wallet_payments` rows in status `pending` with fee stamped from DB schedule → enqueues one asynq task per row → each task calls `WalletPaymentService.Initiate` (which calls OnePay via the queued provider) → on batch completion, books one aggregate Expense ledger transaction. Also exposes a status endpoint for the frontend to poll.

## Requirements

- **Functional**: End-to-end pipeline from bytes → initiated transfers → settled ledger entry. Per-row tracking via `wallet_payments` FSM (existing `pending → verified → authorised → completed|failed`). Duplicate rejection at two levels: batch content hash, and per-row VFIC code against existing non-failed rows.
- **Non-functional**: OnePay's 3 TPS throttle is enforced by the existing `queuedProvider` — no new rate-limiting. Batch of 50 rows completes in ≤30s after OnePay responses. All long work happens in asynq workers; the upload HTTP handler returns within 2s with a `batch_id`.

## Architecture

### Service: `WalletBulkTransferService`

New service in `backend/internal/app/services/wallet_bulk/service.go`. Sibling to (does not extend) `NinePayBulkTransferService` — that one is timesheet-driven and tightly coupled to 9Pay's batch API; this one is file-driven and uses OnePay's per-transfer API.

```go
type WalletBulkTransferService struct {
    batchRepo       BulkTransferBatchRepository
    walletPaymentSvc *disbursement.WalletPaymentService
    feeProvider     disbursement.DisbursementFeeProvider
    bankRepo        domain.BankRepository
    fileStorage     ports.FileStorage
    assetRepo       domain.AssetRepository
    asynqClient     BulkTransferEnqueuer
    ledgerPort      ports.LedgerPort
    txnSvc          TransactionService     // for the aggregate Expense booking
    txMgr           TransactionManager
    clock           clock.Clock
    logger          *slog.Logger
}
```

### Pipeline (Upload handler)

```
POST /wallet/bulk-transfer/upload  (multipart "file")
  ├─ 1. Parse bytes via YeuCauChuyenTienParser → []BulkTransferRow
  ├─ 2. Compute content_hash = SHA-256(normalized row tuples)
  ├─ 3. Check batchRepo.GetByContentHash → if exists, return 409 ConflictError
  ├─ 4. For each row.VFICCode: check wallet_payments by RequestID
  │     if any exists in status ∈ {pending, verified, authorised, completed}
  │     → return 409 with []conflicting VFIC codes (NO rows created, NO OnePay calls)
  ├─ 5. Resolve bank code → SWIFT via bankRepo (failure → 400 ErrUnresolvedBank)
  ├─ 6. Store uploaded bytes via fileStorage → asset_id
  ├─ 7. Create BulkTransferBatch row (status='processing', total_count=N, transfer_amount=Σamount, source='wallet_upload', cycle='flexible')
  ├─ 8. For each row (in original STT order):
  │     - Resolve fee via feeProvider.GetDisbursementFeeVND(ctx, "1pay")
  │     - Build WalletPayment with:
  │         RequestID = "WALLET-BULK-{batchID}-{VFICCode}"  (unique)
  │         RecipientName        = row.AccountName
  │         RecipientAccountNo   = row.AccountNo
  │         RecipientBank        = row.Bank                 (Vietnamese display name e.g. "Quân đội (MB)")
  │         RequestedAmount      = row.Amount
  │         Fee                  = resolved fee (e.g. 3850)
  │         Status               = pending
  │         BulkTransferBatchID  = batch.ID
  │         BulkTransferOrder    = row.OrderNo              (preserves STT for KQ output)
  │         Description          = row.PaymentDetail
  │         CreatedBy            = admin userID
  │     - Create via walletPaymentRepository.Create
  ├─ 9. For each created wallet_payment: asynqClient.EnqueueBulkTransferRow(payload{requestID, batchID})
  └─ 10. Return 202 { batch_id, total_count, transfer_amount, estimated_fee_total }
```

**Critical ordering**: All validation (steps 1-5) happens BEFORE any DB write or OnePay call. Step 4 is the duplicate guard — if any VFIC conflicts, zero rows are created. This is essential because OnePay charges 3,850 VND per call regardless of outcome.

### Per-row worker

New asynq worker `wallet_bulk_transfer_row_worker.go` (mirror existing `bulk_transfer_ninepay_execute_worker.go`):

```go
func (w *WalletBulkTransferRowWorker) ProcessJob(ctx, t *asynq.Task) error {
    var p RowTaskPayload
    asynqlib.Unmarshal(t.Payload(), &p)
    // Delegates entirely to the existing WalletPaymentService.Initiate flow:
    //   InitiateTransfer → RecordSyncResponse → (IPN updates later or status poll)
    _, err := w.walletPaymentSvc.Initiate(ctx, disbursement.InitiateInput{
        RequestID:         p.RequestID,
        Amount:            p.Amount,
        BankCode:          p.BankCode,
        AccountNo:         p.AccountNo,
        AccountName:       p.AccountName,
        Description:       p.Description,
    })
    // err handling: ErrPreflightValidation → mark row failed (no fee was charged);
    //                ErrDuplicatePaymentInProgress → skip silently;
    //                network/OnePay errors → leave row in 'authorised' for IPN reconciliation
    return err   // asynq will retry on transient errors per retry policy
}
```

This **reuses the existing `WalletPaymentService.Initiate`** (see `disbursement/wallet_payment_service.go:125-218`) — it already handles provider selection, fee stamping (we override by pre-stamping), duplicate guard, and FSM. The worker is a thin async wrapper.

### Batch completion + ledger booking

Two possible triggers for "batch complete":
1. **Polling**: A separate asynq task (`bulk-batch-completion-checker`) runs every 30s, finds batches in status `processing` where `success_count + failed_count == total_count`, transitions them to `completed`.
2. **Inline**: Each row worker, after terminal state, attempts an atomic completion via `batchRepo.UpdateWithLock`.

**Use option 2 (inline completion via lock)** — simpler, no extra scheduler. Pseudocode in row worker's terminal hook:

```go
// After wallet_payment reaches completed|failed:
batchRepo.UpdateWithLock(ctx, batchID, func(b *BulkTransferBatch) error {
    // Idempotency guard: if another row already finalized this batch, bail.
    if b.status == "completed" || b.LedgerTxnID != nil {
        return nil
    }
    b.success_count = countByStatus(ctx, batchID, "completed")
    b.failed_count   = countByStatus(ctx, batchID, "failed")
    if b.success_count + b.failed_count == b.total_count {
        b.status = "completed"
        b.completed_at = clock.Now()
        return bookBatchLedger(ctx, b)   // see below
    }
    return nil
})
```

**The `if b.status == "completed" || b.LedgerTxnID != nil` guard is mandatory** — without it, all N terminal hooks in a batch would each call `bookBatchLedger` (because counts stay at total after the first call). This satisfies Phase 5 Test 6 (idempotency).

### Ledger booking — `bookBatchLedger`

Mirror the existing aggregate pattern in `settlement/onepay_fee_import_service.go:114-121`. Create ONE settled Expense transaction:

```go
txn := &domain.Transaction{
    Description:     fmt.Sprintf("Phí OnePay đợt chuyển tiền %s - %s", batch.Filename, batch.ID),
    TransactionType: domain.TransactionTypeExpense,
    Amount:          batch.total_fee,        // Σ fee across ALL rows (success + failure)
    Party:           "OnePay",
    Status:          domain.TransactionStatusSettled,
    CreatedBy:       batch.CreatedBy,
}
txnID := txnSvc.CreateTransaction(ctx, txn)   // produces Debit Expense / Credit Cash ledger entries
batchRepo.Update(ctx, batchID, ledger_txn_id=txnID)
```

`batch.total_fee` is computed at booking time by `SELECT SUM(fee) FROM wallet_payments WHERE bulk_transfer_batch_id = ? AND status IN ('completed','failed')` — this naturally includes both successful and failed rows (per the locked decision: OnePay charges either way). Pre-flight rejections (status `failed` with NULL/non-numeric `error_code`, per migration 077) have `fee=0` already and contribute nothing.

### Fee source (locked)

`feeProvider.GetDisbursementFeeVND(ctx, "1pay")` resolves the active `DisbursementFeeScheduleEntry` for provider `1pay`. Migration 058 seeded 3,300 VND as a placeholder; **migration 093 (Phase 1) updates the active entry to 3,850 VND** (contract rate, per Dieu 6.3.b). The code reads from the schedule, never hardcoded. Tests in Phase 5 explicitly seed 3,850 in setup to make assertions deterministic.

### Status endpoint

```
GET /wallet/bulk-transfer/batches/:id
→ {
    id, filename, status,
    total_count, success_count, failed_count,
    transfer_amount, total_fee,
    created_at, completed_at,
    ledger_txn_id,
    rows: [
      { order_no, account_no, account_name, bank, amount, fee, status, ft_number, error_message }
      ...
    ]
  }
```

Frontend polls this every 5s while `status == 'processing'`.

## Related Code Files

- **Create**: `backend/internal/app/services/wallet_bulk/service.go` — `WalletBulkTransferService`.
- **Create**: `backend/internal/app/services/wallet_bulk/ledger.go` — `bookBatchLedger` helper.
- **Create**: `backend/internal/app/services/wallet_bulk/service_test.go` — unit tests with mocked repos.
- **Create**: `backend/internal/app/workers/wallet_bulk_transfer_row_worker.go` — asynq worker.
- **Create**: `backend/internal/app/services/wallet_bulk/queue.go` — `BulkTransferEnqueuer` impl (or extend existing `bulktransfer/config.go`).
- **Modify**: `backend/internal/app/bootstrap/services/init.go` — instantiate `WalletBulkTransferService`, inject into worker.
- **Modify**: `backend/internal/app/bootstrap/container.go` — wire asynq client to the new enqueuer.
- **Modify**: `backend/internal/app/bootstrap/routes_wallet.go` — add `/wallet/bulk-transfer/*` routes (the handler file is created in Phase 3).
- **Reuse (no change)**: `backend/internal/app/services/disbursement/wallet_payment_service.go:125` (`Initiate`).
- **Reuse (no change)**: `backend/internal/infra/disbursement/onepay/queue.go` (`QueuedProvider` 3 TPS throttle).
- **Reuse (no change)**: `backend/internal/app/services/settlement/onepay_fee_import_service.go` (pattern reference).

## Implementation Steps

1. **Define the service interface** in `wallet_bulk/service.go`:
   ```go
   type Service interface {
       Upload(ctx context.Context, fileData []byte, filename string, userID uint64) (*UploadResult, error)
       GetBatch(ctx context.Context, id uint64) (*BatchDetail, error)
       ListBatches(ctx context.Context, limit, offset int) ([]*BulkTransferBatch, int64, error)
       DownloadKQ(ctx context.Context, id uint64) ([]byte, string, error)   // impl in Phase 3
   }
   ```
   (`DownloadKQ` is a stub returning `ErrNotImplemented` in Phase 2 — filled in Phase 3.)

2. **Implement `Upload`** following the 10-step pipeline above. Each step returns typed errors for HTTP mapping (Phase 3).

3. **Implement content-hash computation** in `wallet_bulk/checksum.go`:
   - Normalize each row: lowercase VFIC code, trim whitespace, sort by AccountNo+Amount.
   - `SHA-256(JSON.stringify(sorted_normalized_rows))`.
   - This is distinct from the existing `bulktransfer/checksum.go` (which hashes `BulkTransferFileData`).

4. **Implement per-row VFIC duplicate check**: query `wallet_payments WHERE request_id LIKE 'WALLET-BULK-%-{vfic}' AND status IN ('pending','verified','authorised','completed')`. The `RequestID` prefix convention makes this query indexable.

5. **Write the asynq worker + enqueuer**:
   - **5a. `wallet_bulk/queue.go`** — `BulkTransferRowEnqueuer` impl wrapping `asynq.Client`:
     ```go
     type asynqRowEnqueuer struct { client *asynq.Client }
     func (e *asynqRowEnqueuer) EnqueueBulkTransferRow(p RowTaskPayload) error {
         payload, _ := json.Marshal(p)
         task := asynq.NewTask("wallet:bulk_transfer_row", payload)
         _, err := e.client.Enqueue(task, asynq.MaxRetry(5), asynq.BackoffStrategy(asynq.ExponentialBackoff))
         return err
     }
     ```
   - **5b. `app/workers/wallet_bulk_transfer_row_worker.go`** — `ProcessJob(ctx, t)`:
     - Unmarshal `RowTaskPayload{ RequestID, BatchID, Amount, BankCode, AccountNo, AccountName, Description }`.
     - Call `walletPaymentSvc.Initiate(...)` → on success, run inline batch-completion check (`markRowTerminal`).
     - Error handling: `ErrPreflightValidation` → mark row failed (no fee); `ErrDuplicatePaymentInProgress` → skip; transient errors → return err for asynq retry.
   - Retry policy: 5 retries with exponential backoff (matches existing asynq patterns).

6. **Implement `bookBatchLedger`** in `wallet_bulk/ledger.go`:
   - Query `SELECT SUM(fee) FROM wallet_payments WHERE bulk_transfer_batch_id = ? AND status IN ('completed','failed')`.
   - If `total == 0` (all rows pre-flight rejected), skip ledger booking — log and return.
   - Else create the Expense transaction via `txnSvc.CreateTransaction`, capture `txnID`, update batch.

7. **Wire bootstrap**:
   - In `init.go` around line 464 (after `walletPaymentService` is constructed), instantiate `WalletBulkTransferService` with the right deps.
   - In `container.go` (around line 183-193, near the existing `NinePayBulkTransfer` wiring): instantiate `asynqRowEnqueuer` with the asynq client, set it on the service via `SetAsynqClient`, and register the row worker with the asynq mux.

8. **Unit tests** (`service_test.go`) with mocks for all repos:
   - Happy path: 2-row file → batch created, 2 `wallet_payments` rows, 2 tasks enqueued.
   - Content-hash duplicate → 409 before any write.
   - Per-row VFIC duplicate → 409 with conflict list before any write.
   - Unresolved bank → 400 before any write.
   - Ledger booking: mock all rows completed with fee 3850 → assert one Expense txn created with amount = 2 × 3850 = 7700.
   - Ledger booking with zero fees (all pre-flight rejections) → no txn created.

9. **Run `make api-test`** — confirm bootstrap wiring compiles and existing flows still pass.

## Success Criteria

- [ ] `POST /wallet/bulk-transfer/upload` (once Phase 3 wires the route) parses, validates, creates the batch + wallet_payments rows, enqueues per-row tasks, returns 202 with `batch_id` within 2s for typical files.
- [ ] Duplicate content-hash upload returns 409 with `{"error":"duplicate_batch","existing_batch_id":...}`.
- [ ] Upload where any VFIC code matches an existing non-failed `wallet_payments` row returns 409 with the conflict list, and **zero rows are created** (verify with `SELECT COUNT(*) FROM wallet_payments WHERE bulk_transfer_batch_id IS NULL`-style check in test).
- [ ] Each enqueued row triggers exactly one `WalletPaymentService.Initiate` call → OnePay `PUT /funds_transfers` (verified by mock OnePay in integration test).
- [ ] On all rows reaching terminal state, batch transitions to `completed` and exactly ONE Expense transaction is created with amount = `Σ fee` across both successful and failed rows.
- [ ] If all rows fail at pre-flight (no OnePay call), no ledger transaction is created.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... -v -race -cover` passes with ≥80% coverage.
- [ ] `make api-test` green.

## Risk Assessment

- **Risk (HIGH)**: OnePay charges 3,850 VND per row even on failure — a malformed upload could burn significant money. **Mitigation**: (a) all validation runs before any OnePay call; (b) duplicate guard at content-hash AND per-VFIC levels; (c) integration test verifies zero OnePay calls on validation failure.
- **Risk**: Race condition where two concurrent uploads of the same file both pass the content-hash check. **Mitigation**: `bulk_transfer_batches.content_hash` has a UNIQUE constraint — second insert fails with duplicate-key, mapped to 409.
- **Risk**: Row worker crashes mid-batch → batch stuck in `processing`. **Mitigation**: Add a stale-batch sweeper cron (low priority, can be Phase 5 follow-up) that re-checks `success_count + failed_count == total_count` every 5min for batches older than 10min.
- **Risk**: `WalletPaymentService.Initiate` re-stamps fee from schedule, ignoring our pre-stamped value. **Mitigation**: Verify in step 8 unit test that the final `wallet_payments.fee` equals our pre-stamped value. If `Initiate` overrides, pass fee explicitly via `InitiateInput` (extend the struct if needed — minor change to `disbursement/wallet_payment_service.go:98`).
- **Risk**: IPN webhook arrives before row worker returns → double FSM transition. **Mitigation**: Existing FSM is idempotent on terminal states (see `transactions/state_machine.go:75-93` — `completed → ipn_reversed → reversed` is the only post-terminal transition). No new mitigation needed.
