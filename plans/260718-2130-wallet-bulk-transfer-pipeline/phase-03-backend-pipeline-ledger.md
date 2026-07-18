---
phase: 3
title: "Backend Pipeline, Worker & Ledger"
status: pending
priority: P1
dependencies: [1]
---

# Phase 2: Backend Pipeline, Worker & Ledger

## Overview

Orchestration service that receives uploaded bytes → parses → dup-checks → creates `bulk_transfer_batches` row with parsed rows in `data` JSON (does NOT create `wallet_payments`) → enqueues per-row asynq tasks. Each task drives the **full 5-step transfer flow mirroring `disbursement_execute_worker.go` line-for-line**, including all canonical error branches and guards.

On batch completion, `wallet:book_batch_ledger` asynq task books ONE Expense transaction via `txnSvc.CreateTransaction` (3 returns), idempotent via `ledger_txn_id IS NULL`. `wallet:bulk_completing_recovery` cron handles stuck batches; `wallet:bulk_stale_enqueue_sweeper` cron handles outbox recovery.

## Requirements

- **Functional**: End-to-end pipeline bytes → initiated transfers → settled ledger. Worker mirrors canonical pattern exactly. Outbox + recovery crons. Idempotent ledger booking.
- **Non-functional**: 3 TPS throttle via `queuedProvider`. Upload returns within 2s with `batch_id`. Realistic terminal SLA: ≥90% in 60s via IPN, remainder in 10min via poller.

## Architecture

### Service: `WalletBulkTransferService`

```go
type WalletBulkTransferService struct {
    batchRepo   BulkTransferBatchRepository
    fileStorage ports.FileStorage
    assetRepo   domain.AssetRepository
    asynqClient BulkTransferEnqueuer
    txnSvc      *settlement.TransactionService
    txMgr       TransactionManager
    parser      *YeuCauChuyenTienParser
    clock       clock.Clock
    logger      *slog.Logger
}
```

No `paymentRepo` field — service does NOT write `wallet_payments` rows directly. Worker does, via `WalletPaymentService.Initiate`.

### Pipeline (Upload handler) — no wallet_payments writes

```
POST /wallet/bulk-transfer/upload  (multipart "file")
  ├─ 1. http.MaxBytesReader(10<<20+512) BEFORE c.FormFile
  ├─ 2. fileHeader.Size > 10<<20 → 400 file_too_large
  ├─ 3. ZIP magic sniff (50 4B 03/05/07 04/06/08) → else 400 invalid_file_type
  ├─ 4. Read bytes via io.LimitReader(file, 10<<20+1)
  ├─ 5. Parse bytes via parser.Parse(ctx, bytes) → []BulkTransferRow
  │     (parser reads SWIFT directly; no BankRepository)
  ├─ 6. Sanitize filename: filepath.Base + strip control chars + cap 128 + reject <>"'
  ├─ 7. Compute content_hash = SHA-256(normalized row tuples including all fields)
  ├─ 8. Check batchRepo.GetByContentHash → if exists, return 409 ErrDuplicateBatch
  ├─ 9. For each row: check wallet_payments WHERE request_id = row.VFICCode
  │     AND status IN ('pending','verified','authorised')
  │     if any exists → return 409 ErrDuplicateVFIC with conflict list
  │     (DB existing idx_wp_request_id UNIQUE is the TOCTOU backstop)
  ├─ 10. Store uploaded bytes via fileStorage → asset_id
  ├─ 11. Create BulkTransferBatch (status='processing', total_count=N,
  │      transfer_amount=Σamount, enqueue_state='pending',
  │      data = JSON-encoded []BulkTransferRow,  ← parsed rows persisted HERE
  │      filename = sanitized, asset_id, created_by=userID)
  ├─ 12. For each row: asynqClient.EnqueueBulkTransferRow(payload{batch_id, row})
  │      (payload carries the FULL BulkTransferRow so worker doesn't re-parse)
  ├─ 13. UPDATE bulk_transfer_batches SET enqueue_state='enqueued' WHERE id=?
  │      (if process crashes between step 11 and 13, sweeper reads batch.data
  │       and re-enqueues — see Stale Enqueue Sweeper)
  └─ 14. Return 202 { batch_id, total_count, transfer_amount, estimated_fee_total }
```

**Critical**: Service NEVER calls `paymentRepo.Create` or `WalletPaymentService.Initiate`. Worker is sole insertion point. Ensures `Initiate`'s fee-stamp from schedule runs correctly.

### Per-row worker — EXACT canonical pattern

Reference: `backend/internal/app/workers/disbursement_execute_worker.go:73-237`. **Copy structure line-for-line**.

```go
type WalletBulkTransferRowWorker struct {
    walletPaymentSvc *disbursement.WalletPaymentService
    registry         *disbursement.Registry
    paymentRepo      domaintx.WalletPaymentRepository  // for ensureBulkBatchLink + counts
    batchRepo        BulkTransferBatchRepository
    asynqClient      BulkTransferEnqueuer
    logger           *slog.Logger
}

func (w *WalletBulkTransferRowWorker) ProcessJob(ctx context.Context, t *asynqlib.Task) error {
    var p RowTaskPayload
    if err := asynqlib.Unmarshal(t.Payload(), &p); err != nil { return fmt.Errorf("unmarshal: %w", err) }

    logger := w.logger.With("batch_id", p.BatchID, "vfic", p.Row.VFICCode)

    // === Step 1: Initiate wallet_payment (sole insertion point — fee stamps here) ===
    row, err := w.walletPaymentSvc.Initiate(ctx, disbursement.InitiateInput{
        RequestID:          p.Row.VFICCode,         // ≤20 chars
        RequestedAmount:    p.Row.Amount,
        RecipientName:      p.Row.AccountName,
        RecipientAccountNo: p.Row.AccountNo,
        RecipientBank:      p.Row.SwiftCode,        // SWIFT — Initiate accepts it directly
        Description:        p.Row.PaymentDetail,
        // EntityID left nil — bulk transfers aren't linked to advance_payment_requests.
    })
    if err != nil {
        // === CANONICAL TERMINAL-ERROR BRANCHES (mirror disbursement_execute_worker.go:114-122) ===
        if errors.Is(err, disbursement.ErrDuplicatePaymentInProgress) {
            return nil  // already in-flight
        }
        if errors.Is(err, disbursement.ErrFeeResolution) {
            // Fee schedule misconfigured — don't retry, surface immediately (red-team v2 S-8)
            return fmt.Errorf("initiate: %w: fee config gap", asynqlib.SkipRetry)
        }
        return fmt.Errorf("initiate: %w", err)
    }

    // === CANONICAL STATE GUARD (mirror disbursement_execute_worker.go:127-131) ===
    if row.Status != domaintx.StatePending {
        logger.Info("wallet bulk: skipping — already processed", "status", row.Status)
        w.ensureBulkBatchLink(ctx, row.ID, p.BatchID, p.Row)
        return w.markRowTerminal(ctx, p.BatchID)
    }

    w.ensureBulkBatchLink(ctx, row.ID, p.BatchID, p.Row)

    // === Step 2: Get active provider ===
    provider, err := w.registry.Active(ctx)
    if err != nil { return fmt.Errorf("no active provider: %w", err) }

    // === Step 3: Record account check — auto-verify ===
    _, err = w.walletPaymentSvc.RecordAccountCheck(ctx, p.Row.VFICCode, disbursement.AccountCheckOutcome{
        Verified: true,
    })
    if err != nil {
        return w.markRowFailed(ctx, p.BatchID, p.Row.VFICCode, err, false)
    }

    // === Step 4: Initiate transfer via provider (rate-limited by queuedProvider) ===
    result, err := provider.InitiateTransfer(ctx, infrastructure.TransferRequest{
        RequestID:   p.Row.VFICCode,
        Amount:      p.Row.Amount,
        SwiftCode:   p.Row.SwiftCode,
        AccountNo:   p.Row.AccountNo,
        AccountName: p.Row.AccountName,
        Description: p.Row.PaymentDetail,
        AccountType: infrastructure.AccountTypeBankAccount,
    })
    if err != nil {
        if errors.Is(err, infrastructure.ErrPreflightValidation) {
            return w.markRowFailed(ctx, p.BatchID, p.Row.VFICCode, err, true /*FeeWaived*/)
        }
        return fmt.Errorf("initiate_transfer: %w", err)
    }

    // === Step 5: Record sync response — EXACT canonical field mapping ===
    // SyncResult has ONLY: Accepted, InvoiceNo, RawErrorCode, RawMessage, FeeWaived.
    _, err = w.walletPaymentSvc.RecordSyncResponse(ctx, p.Row.VFICCode, disbursement.SyncResult{
        // CRITICAL (red-team v2 R2-1): Accepted must be set explicitly.
        Accepted:     result.Status == infrastructure.TransferStatusPending || result.Status == infrastructure.TransferStatusSuccess,
        InvoiceNo:    result.ProviderRef,   // canonical mapping
        RawErrorCode: result.RawErrorCode,
        RawMessage:   result.RawMessage,
    })
    if err != nil {
        return fmt.Errorf("record_sync_response: %w", err)
    }

    // Step 6: terminal state reached later via IPN webhook or status-inquiry poller.
    return nil
}

// ensureBulkBatchLink stamps bulk_transfer_batch_id + order + vfic_code + entity_id.
// Idempotent — safe to call on retry.
//
// entity_id resolution (Validation Decision V6 — enable push notifications):
// The existing notifyEmployee FSM hook requires entity_id to look up the employee
// via advance_payment_requests. For bulk transfers (no advance_payment_request),
// we resolve entity_id differently — but the simplest path is to leave entity_id
// nil and instead extend notifyEmployee to handle nil-entity_id rows by looking
// up employee via recipient_account_no. See plan.md "Notification Path" section.
//
// For now, ensureBulkBatchLink stamps everything EXCEPT entity_id; the notification
// extension is a Phase 3 follow-up task documented in plan.md.
func (w *WalletBulkTransferRowWorker) ensureBulkBatchLink(ctx, rowID uint64, batchID uint64, row BulkTransferRow) {
    w.paymentRepo.UpdateBulkBatchLink(ctx, rowID, batchID, uint(row.OrderNo), row.VFICCode)
}

// TODO (Phase 3 follow-up — Notification Path per Validation V6):
// Once notifyEmployee is extended to handle bulk rows (Option A in plan.md:
// lookup employee by recipient_account_no when entity_id is nil), no entity_id
// stamping is needed — the notification hook will resolve employee from the
// wallet_payment's recipient_account_no directly.

// markRowFailed routes a row through the FSM via RecordSyncResponse,
// NOT via direct UpdateStatus (red-team v2 Security C5).
// feeWaived: true when the failure was a preflight rejection.
func (w *WalletBulkTransferRowWorker) markRowFailed(ctx context.Context, batchID uint64, requestID string, cause error, feeWaived bool) error {
    _, err := w.walletPaymentSvc.RecordSyncResponse(ctx, requestID, disbursement.SyncResult{
        Accepted:     false,             // TriggerReject
        RawErrorCode: extractErrorCode(cause),
        RawMessage:   cause.Error(),
        FeeWaived:    feeWaived,
    })
    if err != nil {
        return fmt.Errorf("markRowFailed: record_sync_response: %w", err)
    }
    return nil
}

// extractErrorCode parses a OnePay/infrastructure error for its raw error code.
// TODO: implement based on canonical worker's pattern.
func extractErrorCode(err error) string { /* TODO */ return "" }
```

### markRowTerminal — atomic via `UpdateWithLock` returning `(batch, shouldBook, err)`

```go
func (w *WalletBulkTransferRowWorker) markRowTerminal(ctx context.Context, batchID uint64) error {
    batch, shouldBook, err := w.batchRepo.UpdateWithLock(ctx, batchID, func(b *BulkTransferBatch) (bool, error) {
        if b.Status == "completed" || b.Status == "failed" || b.FeeBookedAt != nil {
            return false, nil
        }
        successCount, _ := w.paymentRepo.CountByBatchAndStatuses(ctx, batchID, []string{"completed"})
        failedCount, _   := w.paymentRepo.CountByBatchAndStatuses(ctx, batchID, []string{"failed"})
        b.SuccessCount = successCount
        b.FailedCount = failedCount
        if successCount + failedCount == int64(b.TotalCount) {
            now := w.clock.Now()
            b.FeeBookedAt = &now
            b.Status = "completing"
            return true, nil
        }
        return false, nil
    })
    if err != nil { return err }
    if !shouldBook { return nil }
    return w.asynqClient.EnqueueBookBatchLedger(BookLedgerPayload{BatchID: batchID})
}
```

### Book-batch-ledger asynq task (`wallet:book_batch_ledger`)

```go
func (s *WalletBulkTransferService) ProcessBookBatchLedger(ctx context.Context, t *asynqlib.Task) error {
    var p BookLedgerPayload
    asynqlib.Unmarshal(t.Payload(), &p)

    batch, err := s.batchRepo.GetByID(ctx, p.BatchID)
    if err != nil { return err }

    if batch.LedgerTxnID != nil || batch.Status == "completed" {
        return nil
    }

    // Sum fees for ALL terminal rows. No resolution_source filter (red-team v2 H7).
    totalFee, err := s.paymentRepo.SumFeeByBatchAndStatuses(ctx, batch.ID, []string{"completed", "failed"})
    if err != nil { return err }
    batch.TotalFee = totalFee

    if totalFee == 0 {
        return s.batchRepo.Update(ctx, &BulkTransferBatch{
            ID: batch.ID, Status: "completed", CompletedAt: s.clock.Now(), TotalFee: 0,
        })
    }

    // CreateTransaction returns 3 values (red-team v2 Assumption Destroyer):
    createdTxn, _, err := s.txnSvc.CreateTransaction(ctx, &domain.Transaction{
        Description:     fmt.Sprintf("Phí OnePay đợt chuyển tiền %s - batch #%d", batch.Filename, batch.ID),
        TransactionType: domain.TransactionTypeExpense,
        Amount:          totalFee,
        Party:           "OnePay",
        Status:          domain.TransactionStatusSettled,
        CreatedBy:       batch.CreatedBy,
    })
    if err != nil { return err }

    return s.batchRepo.Update(ctx, &BulkTransferBatch{
        ID: batch.ID, LedgerTxnID: &createdTxn.ID, Status: "completed",
        CompletedAt: s.clock.Now(), TotalFee: totalFee,
    })
}
```

Idempotent: re-enqueueing for completed batch hits `LedgerTxnID != nil` → nil.

### Stale enqueue sweeper (`wallet:bulk_stale_enqueue_sweeper`, @every 1m)

Reads parsed rows from `bulk_transfer_batches.data` JSON — no dependency on `wallet_payments`. Retry cap at 3 per row via `sweeper_retry_count`.

### Completing-batch recovery (`wallet:bulk_completing_recovery`, @every 5m)

Re-attempts booking for batches stuck in `completing` >10min. Idempotent.

### OTP gating

`OTP_ENABLE` defaults to `false` (`config.go:421`). Bulk upload is money-moving. Log startup warning if `OTP_ENABLE=false`:

```go
if !cfg.OTP.Enabled {
    logger.Warn("SECURITY: /wallet/bulk-transfer routes registered without OTP enforcement. " +
        "Set OTP_ENABLE=true in production before enabling this feature.")
}
```

## Related Code Files

- **Create**: `backend/internal/app/services/wallet_bulk/service.go` — `WalletBulkTransferService`.
- **Create**: `backend/internal/app/services/wallet_bulk/queue.go` — `BulkTransferEnqueuer`.
- **Create**: `backend/internal/app/services/wallet_bulk/types.go` — payloads + sentinels.
- **Create**: `backend/internal/app/workers/wallet_bulk_transfer_row_worker.go` — 5-step worker.
- **Create**: `backend/internal/app/services/wallet_bulk/service_test.go`.
- **Modify**: `backend/internal/domain/transactions/repository.go` — add `CountByBatchAndStatuses`, `SumFeeByBatchAndStatuses`, `UpdateBulkBatchLink`, `IncrementSweeperRetry`, `ListByEnqueueState`, `UpdateEnqueueState`, `ListByStatusAndOlderThan`.
- **Modify (Notification Path — Validation V6)**: `backend/internal/app/services/disbursement/wallet_payment_service.go` — extend `notifyEmployee` to handle `entity_id == nil` rows by looking up employee via `recipient_account_no → employees.bank_account_number`. Apply to all notification call sites (OnEnterCompleted, OnEnterFailed hooks already exist). Mirrors the Option A design from `plan.md` "Notification Path" section. Add a unit test asserting a bulk row with `entity_id=nil` + valid `recipient_account_no` triggers a notification.
- **Modify**: `backend/internal/infra/persistence/tx_wallet_payment_repository.go` — implement the new methods.
- **Modify**: `backend/internal/app/bootstrap/services/init.go`, `container.go` — wire service + register 3 asynq tasks.
- **Reuse (no change)**: `WalletPaymentService.Initiate`, `RecordAccountCheck`, `RecordSyncResponse`, `RecordIPN`.
- **Reuse (no change)**: `QueuedProvider` 3 TPS throttle.

## Implementation Steps

1. Define service interface + types.

2. Implement `Upload` per 14-step pipeline.

3. Implement content-hash normalization (include all parsed fields).

4. Implement per-row VFIC duplicate check via `request_id` equality.

5. **Write the worker** per canonical pseudocode. Reference `disbursement_execute_worker.go:73-237` line-for-line. Critical checkpoints: StatePending guard, Accepted flag, InvoiceNo mapping, ErrFeeResolution→SkipRetry, ErrPreflightValidation→markRowFailed(feeWaived=true), markRowFailed via FSM.

6. Implement `extractErrorCode(err)` — mirror canonical worker pattern.

7. Write enqueuer with both methods.

8. Implement `ProcessBookBatchLedger` per pseudocode. Use 3-return `CreateTransaction`; capture `createdTxn.ID`.

9. Implement `ProcessStaleEnqueueSweeper` + `ProcessCompletingRecovery`.

10. Wire bootstrap + asynq mux + OTP startup warning.

11. Unit tests covering: happy path, dup content-hash, dup VFIC, Accepted flag mapping, ErrFeeResolution→SkipRetry, ErrPreflightValidation→markRowFailed(feeWaived=true), StatePending guard, booking idempotency, zero-fee batch.

12. Run `make api-test`.

## Success Criteria

- [ ] Upload parses + validates + creates batch with `data` JSON; **does NOT create wallet_payments**.
- [ ] Backend enforces 10MB + ZIP magic + `http.MaxBytesReader` + multipart memory cap.
- [ ] Duplicate content-hash → 409.
- [ ] Per-row VFIC duplicate → 409 via `request_id` equality.
- [ ] Worker calls `Initiate` (sole insertion point, fee stamps correctly), guards `StatePending`, runs full 5-step flow.
- [ ] `SyncResult` uses only real fields with `Accepted` explicitly mapped.
- [ ] `ErrFeeResolution` → `asynq.SkipRetry`.
- [ ] `ErrPreflightValidation` → `markRowFailed(feeWaived=true)` via FSM.
- [ ] `markRowFailed` routes through `RecordSyncResponse(Accepted:false)`, NOT direct UpdateStatus.
- [ ] RequestID = VFIC code (≤20 chars).
- [ ] On all rows terminal: `book_batch_ledger` task enqueued → `fee_booked_at` + `status=completing` committed in lock tx → ledger booked outside lock → `ledger_txn_id` set.
- [ ] Booking uses 3-return `CreateTransaction`; captures `createdTxn.ID`.
- [ ] Idempotent re-booking via `ledger_txn_id IS NULL`.
- [ ] Zero-fee batch → no txn, status=completed.
- [ ] `completing` recovery cron re-attempts booking for batches stuck >10min.
- [ ] Stale enqueue sweeper reads `batch.data` JSON; retry cap at 3.
- [ ] OTP startup warning logged if `OTP_ENABLE=false`.
- [ ] `cd backend && go test ./internal/app/services/wallet_bulk/... -v -race -cover` ≥80%.

## Risk Assessment

- **Risk (HIGH)**: OnePay charges per row even on failure. **Mitigation**: validation before any OnePay call; `idx_wp_request_id` UNIQUE backstop; size + ZIP magic + `UnzipSizeLimit`; `ErrFeeResolution` skips retry.
- **Risk**: Worker dies between batch INSERT commit and asynq enqueue. **Mitigation**: Stale enqueue sweeper reads `batch.data`; retry cap at 3.
- **Risk**: IPN before RecordSyncResponse → IPN discarded, row strands in `authorised`. **Mitigation**: Status-inquiry poller recovers.
- **Risk**: Status-inquiry poller caps completion at ~20min/50 stranded rows. **Mitigation**: Documented in AC.
- **Risk**: Ledger write fails after `fee_booked_at` committed. **Mitigation**: `completing` recovery cron every 5min.
- **Risk**: Lock held across ledger write. **Mitigation**: Booking in separate asynq task AFTER lock release.
- **Risk**: Reconcile-flow collision. **Mitigation**: Documented; recommend not running reconcile while bulk batch in-flight.
- **Risk (mitigated)**: ~~`notifyEmployee` early-returns when `EntityID==nil`~~ — **Validation V6 reversed this**: push notifications ARE required for bulk transfers. **Mitigation**: Phase 3 extends `notifyEmployee` to handle `entity_id=nil` rows by looking up employee via `recipient_account_no → employees.bank_account_number` (Option A in plan.md "Notification Path"). Test 18 step 8 verifies notifications fire for bulk-completed rows.
