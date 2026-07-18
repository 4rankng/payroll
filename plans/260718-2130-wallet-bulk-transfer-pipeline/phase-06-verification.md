---
phase: 6
title: "Verification"
status: pending
priority: P2
dependencies: [1, 2, 3, 4, 5]
---

# Phase 5: Verification

## Overview

End-to-end integration tests proving the v6 pipeline. Uses **synchronous worker dispatcher** in a `_test.go` file (no `//go:build integration` tag — red-team v2 Security H6: the codebase has zero build-tag precedent and the Makefile doesn't pass `-tags=integration`; Go's `_test.go` convention prevents production compilation cleanly).

## Requirements

- All AC verified by automated tests.
- Tests run ≤30s with sync dispatcher.

## Architecture

### Synchronous worker dispatcher (`sync_dispatcher_test.go`)

```go
// File: backend/internal/app/services/wallet_bulk/sync_dispatcher_test.go
// _test.go suffix prevents production compilation (red-team v2 Security H6).

package wallet_bulk

type SyncDispatcher struct {
    rowWorker      *WalletBulkTransferRowWorker
    bookLedgerSvc *WalletBulkTransferService
}

func (d *SyncDispatcher) EnqueueBulkTransferRow(p RowTaskPayload) error {
    task := asynqlib.NewTask("wallet:bulk_transfer_row", mustJSON(p))
    return d.rowWorker.ProcessJob(context.Background(), task)
}

func (d *SyncDispatcher) EnqueueBookBatchLedger(p BookLedgerPayload) error {
    task := asynqlib.NewTask("wallet:book_batch_ledger", mustJSON(p))
    return d.bookLedgerSvc.ProcessBookBatchLedger(context.Background(), task)
}
```

Integration tests inject this. No Redis.

### Mock OnePay provider

- Success for `AccountNo` not starting with `9999` or `8888`.
- Failed (`Status=Failed`) for `9999*`.
- `ErrPreflightValidation` for `8888*`.
- Track call count.

## Integration Test Scenarios

### Test 1: Happy path (v6 full pipeline)

1. Seed `DisbursementFeeScheduleEntry` for `1pay` with `fee_vnd=3850`.
2. POST `/wallet/bulk-transfer/upload` with `yeu_cau_chuyen_tien_with_swift.xlsx` (2 rows).
3. Assert 202 with `batch_id`, `total_count=2`, `estimated_fee_total=7700`.
4. Sync dispatcher runs both row workers. Each:
   - `Initiate` → creates row with `fee=3850`.
   - `Status == StatePending` → proceeds.
   - `RecordAccountCheck(Verified:true)` → `verified`.
   - `provider.InitiateTransfer` → mock returns `Status=Pending`.
   - `RecordSyncResponse(Accepted:true, InvoiceNo:ProviderRef)` → `authorised`.
5. Manually fire mock IPN → both rows → `completed`.
6. Worker's `OnEnterCompleted` hook → `markRowTerminal`.
7. `markRowTerminal` → lock tx commits `fee_booked_at` + `status=completing`, returns `shouldBook=true`.
8. Sync dispatcher runs `book_batch_ledger` → creates Expense txn with `amount=7700`.
9. Assert: 2 wallet_payments rows with `fee=3850`, `bulk_transfer_batch_id=id`, `status=completed`. 1 transactions row `type=expense`, `party=OnePay`, `amount=7700`, `status=settled`. `bulk_transfer_batches.ledger_txn_id` set.
10. GET `/wallet/bulk-transfer/batches/:id/kq` → 2 data rows show `Thành công` with FT in column I.

### Test 2: Duplicate content-hash rejection

Upload same file twice → 409, no new rows, no OnePay call.

### Test 3: Duplicate VFIC rejection (historical + new)

1. Pre-seed `wallet_payments` row with `request_id='VFIC_TEST_A'`, `status='completed'`.
2. Upload file with `VFIC_TEST_A` in row 1.
3. Assert 409 `duplicate_vfic` with conflict list. No new rows. No OnePay call.
4. Concurrent-upload test: two goroutines uploading same new file. Exactly one succeeds; one hits `idx_wp_request_id` UNIQUE violation (mapped to 409).

### Test 4: Partial failure ledger booking

3 rows: 2 normal + 1 with `AccountNo='9999...'` (mock Failed).
- Assert `success_count=2`, `failed_count=1`, `total_fee=3 × 3850=11550`.
- Assert 1 Expense txn `amount=11550`.
- KQ shows 2 `Thành công` + 1 `Thất bại`.

### Test 5: All-preflight-failure (no fee, no ledger)

Rows with `AccountNo='8888*'` (mock `ErrPreflightValidation`).
- Worker calls `markRowFailed(feeWaived=true)` → `RecordSyncResponse(Accepted:false, FeeWaived:true)` → fee zeroed.
- Assert `failed_count=N`, `total_fee=0`. No Expense txn. status=completed.

### Test 6: Idempotency of ledger booking + `completing` transition

1. Drive batch to completion.
2. Assert batch transitions through `status='completing'`.
3. Assert `fee_booked_at` BEFORE Expense txn's `created_at`.
4. Re-call `ProcessBookBatchLedger` → no duplicate txn.

### Test 7: 3 TPS throttle respected

10-row Excel. Use real `QueuedProvider` wrapping mock (NOT bare mock — must NOT bypass throttle).
- Assert total time ≥ 3s.

### Test 8: KQ format byte-level comparison

Generate KQ for Test 1 batch. Parse with `excelize`. Assert cell-by-cell:
- A1 title, A2 ref no `WB<id>`, A3 date format.
- Row 4 headers exact match.
- Row 5 H=`Thành công`, I=FT number.
- Column G=3850 for all rows.
- Column D = Vietnamese display name (e.g. `"Quân đội (MB)"`), NOT SWIFT.

### Test 9: KQ "Đang chờ FT" fallback

1. Build row that completed via IPN but IPN didn't carry `ProviderRef` → `invoice_no` still VFIC.
2. Generate KQ. Assert column I = `"Đang chờ FT"`.

### Test 10: Backend file-size limit + ZIP magic + multipart cap

- POST 11MB file → 400 `file_too_large`.
- POST PDF renamed `.xlsx` → 400 `invalid_file_type`.
- POST file exceeding `MaxMultipartMemory` → connection terminated mid-stream.

### Test 11: Outbox sweeper recovery

1. Upload batch, mock `EnqueueBulkTransferRow` to fail.
2. Assert `bulk_transfer_batches.enqueue_state='pending'`, no row tasks ran.
3. Invoke `ProcessStaleEnqueueSweeper` with cutoff = `clock.Now().Add(-3min)`.
4. Assert sweeper reads `batch.data` JSON, re-enqueues tasks, flips `enqueue_state='enqueued'`.
5. Retry-cap test: re-invoke sweeper 4 times for same row; assert 4th invocation skips (count > 3).

### Test 12: Completing-batch recovery

1. Mock `txnSvc.CreateTransaction` to fail on first call.
2. Drive batch to `completing` (lock tx commits `fee_booked_at`, but `book_batch_ledger` fails).
3. Assert batch stuck in `completing`, `ledger_txn_id` NULL.
4. Restore `txnSvc` mock to succeed.
5. Invoke `ProcessCompletingRecovery` with cutoff = `clock.Now().Add(-11min)`.
6. Assert recovery re-enqueues `book_batch_ledger`; booking succeeds; batch → `completed`.

### Test 13: StatePending guard prevents double-processing

1. Run worker Steps 1-3 on a row → `verified`.
2. Simulate crash: don't run Steps 4-5.
3. Re-enqueue the row.
4. Assert worker sees `Status == StatePending` is FALSE → returns nil early. No double-RecordAccountCheck. No FSM violation.

### Test 14: ErrFeeResolution → SkipRetry

1. Delete fee schedule entry (simulate misconfiguration).
2. Upload + run worker.
3. Assert `Initiate` returns `ErrFeeResolution`.
4. Assert worker returns `asynq.SkipRetry` wrapped error.

### Test 15: Missing SWIFT column rejection

Upload the legacy `yeu_cau_chuyen_tien.xlsx` (no SWIFT column).
- Assert 400 `missing_swift_column`.

### Test 16: Exporter — happy path (Stage 1)

1. Seed: 3 approved timesheets for cycle "weekly" with employees that have `bank_id` set (bank records with SWIFT codes `MBBEVNVX`, `BFTVVNVX`, `TCBVVNVX`).
2. POST `/api/v1/payrolls/export-onepay-bulk` with `{cycle: "weekly", project_id: X, from_date, to_date}`.
3. Assert 200 + blob response with `Content-Disposition: attachment; filename="Yeu_cau_chuyen_tien_weekly_<timestamp>.xlsx"`.
4. Parse the blob with `excelize.OpenReader`:
   - Assert sheet name `eMB_BulkPayment`.
   - Assert header row 2 has 7 columns including `Mã SWIFT\n(SWIFT Code)`.
   - Assert 3 data rows (3+) with correct AccountNo, AccountName, Bank (Vietnamese), SwiftCode, Amount, PaymentDetail (contains `VFIC...`).
5. Assert `transaction_codes` table has 3 new rows linking VFIC → timesheet IDs.

### Test 17: Exporter — skip employees with missing bank info

1. Seed: 3 employees. Employee #2 has `bank_id = nil`.
2. POST `/api/v1/payrolls/export-onepay-bulk`.
3. Assert response includes `X-Skipped-Count: 1` header.
4. Parse blob: assert 2 data rows (employee #2 skipped).
5. Assert `transaction_codes` table has only 2 new rows.

### Test 18: End-to-end — export then upload

1. Run Test 16 (export) → save the blob to a temp file.
2. POST `/api/v1/wallet/bulk-transfer/upload` with the exported file.
3. Assert 202 with `batch_id`, `total_count=3`.
4. Run sync dispatcher → all 3 rows reach `completed` via mock OnePay.
5. Assert: 3 `wallet_payments` rows with `bulk_transfer_batch_id=id`, `fee=3850`, `request_id` matching the VFIC codes from step 1.
6. Assert: 1 Expense txn with `amount=3 × 3850=11550`.
7. Download KQ → assert 3 rows showing `Thành công` with FT numbers.

This test proves the two stages work together: the file the exporter produces is consumable by the wallet pipeline without manual intervention.

## Related Code Files

- **Create**: `backend/tests/integration/wallet_bulk_transfer_test.go` — Tests 1-15, 18.
- **Create**: `backend/tests/integration/onepay_exporter_test.go` — Tests 16-17.
- **Create**: `backend/internal/app/services/wallet_bulk/sync_dispatcher_test.go` (test helper; `_test.go` suffix).
- **Create**: Anonymized `yeu_cau_chuyen_tien_with_swift.xlsx` fixture (or generate via Test 16 and reuse).
- **Create**: `backend/testdata/kq_chuyen_tien_reference.xlsx`.
- **Create**: `docs/runbooks/wallet-bulk-transfer-smoke-test.md` + `docs/runbooks/wallet-bulk-transfer-stuck-completing-recovery.md`.

## Implementation Steps

1. Build anonymized fixture: copy `~/Downloads/Yeu cau chuyen tien.xlsx`, add `Mã SWIFT` column with real SWIFT codes from migration 059 (MB→`MBBEVNVX`, VCB→`BFTVVNVX`), replace names/accounts with test data.
2. Write sync dispatcher.
3. Set up mock OnePay provider.
4. Write Test 1 first.
5. Write Tests 2-15 in order.
6. Run `make api-test`.
7. Write smoke-test + stuck-completing recovery runbooks.
8. Frontend: `pnpm lint && pnpm type-check`.
9. Update `HANDOFF.md`.

## Success Criteria

- [ ] All 15 integration tests pass under `make api-test`.
- [ ] Tests run ≤30s using sync dispatcher (no real asynq boot).
- [ ] Test 1 confirms fee stamps correctly via `Initiate`.
- [ ] Test 1 confirms `Accepted` flag set correctly.
- [ ] Test 3 confirms existing `idx_wp_request_id` UNIQUE backstop works.
- [ ] Test 5 confirms preflight rejections have fee zeroed via `FeeWaived: true`.
- [ ] Test 6 confirms `completing` transition + idempotent booking.
- [ ] Test 9 confirms KQ `"Đang chờ FT"` fallback.
- [ ] Test 11 confirms outbox sweeper reads `batch.data` JSON + retry cap.
- [ ] Test 12 confirms completing-batch recovery.
- [ ] Test 13 confirms StatePending guard.
- [ ] Test 14 confirms ErrFeeResolution → SkipRetry.
- [ ] Test 15 confirms missing SWIFT column rejection.
- [ ] Test fixtures anonymized + committed.
- [ ] `make api-test` overall green.
- [ ] `pnpm lint && pnpm type-check` pass.
- [ ] Smoke-test + stuck-completing recovery runbooks written.

## Risk Assessment

- **Risk**: Mock provider doesn't perfectly emulate OnePay. **Mitigation**: explicit modes (success/failed/preflight).
- **Risk**: Test fixtures collide with dev DB. **Mitigation**: deterministic VFIC generation in tests; per-test truncation.
- **Risk**: KQ byte-comparison brittle. **Mitigation**: compare parsed cell values, regex on timestamps.
