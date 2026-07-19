# Runbook: Wallet Bulk Transfer — Stuck `completing` Recovery

## Symptom

A bulk-transfer batch shows `status=completing` for more than 10 minutes in
`/admin/wallet` or directly in the `bulk_transfer_batches` table, and no
Expense transaction has been created (i.e. `ledger_txn_id IS NULL`).

## Root cause

The `markRowTerminal` worker transitioned the batch to `completing` and
enqueued `wallet:book_batch_ledger`, but the ledger booking task failed
(e.g. `txnSvc.CreateTransaction` errored on a DB connection blip) AND the
`completing`-recovery cron hasn't yet run (it fires `@every 5m`, only acts
on rows older than 10min).

## Automatic recovery

The `wallet:bulk_completing_recovery` cron handles this automatically —
no operator action required for transient failures. Wait ≤15min and the
batch should self-heal to `completed`.

## Manual recovery

If the cron itself is failing (Redis down, worker panic), run the booking
manually:

### Option A — re-trigger via SQL (preferred for ops)

```sql
-- Inspect the stuck batch.
SELECT id, filename, status, fee_booked_at, ledger_txn_id, total_fee, updated_at
FROM bulk_transfer_batches
WHERE status = 'completing' AND ledger_txn_id IS NULL;

-- Compute the expected fee (should match total_fee once booked).
SELECT SUM(fee) AS expected_fee
FROM wallet_payments
WHERE bulk_transfer_batch_id = <BATCH_ID>
  AND status IN ('completed', 'failed');
```

If `expected_fee = 0`, the batch had no real transfers reach the OnePay
endpoint — flip it to `completed` directly:

```sql
UPDATE bulk_transfer_batches
SET status = 'completed', total_fee = 0, completed_at = NOW()
WHERE id = <BATCH_ID> AND ledger_txn_id IS NULL;
```

If `expected_fee > 0`, create the Expense transaction via the admin UI
(`/admin/transactions` → New → type=`expense`, party=`OnePay`,
amount=`expected_fee`, description includes the batch id) then link it:

```sql
UPDATE bulk_transfer_batches
SET status = 'completed',
    total_fee = <expected_fee>,
    ledger_txn_id = <NEW_TXN_ID>,
    completed_at = NOW()
WHERE id = <BATCH_ID> AND ledger_txn_id IS NULL;
```

### Option B — restart the worker

If Redis was down, restart the asynq server. On next boot the
`completing`-recovery cron picks up the stuck batch within 5min.

## Idempotency note

Re-running `ProcessBookBatchLedger` on a batch whose `ledger_txn_id` is
already set is a no-op (the handler early-returns). However, the
**CRITICAL window** — between `txnSvc.CreateTransaction` committing and
`batchRepo.Update` committing — can produce a duplicate Expense transaction
if the recovery cron fires after a txn was created but before the link was
stamped. Detect this case with:

```sql
SELECT id, amount, description, created_at
FROM transactions
WHERE party = 'OnePay'
  AND description LIKE '%batch #<BATCH_ID>%'
ORDER BY created_at;
```

If two rows exist, reverse the duplicate (mark one as `reversed` in the
admin UI — the reverse/refund workflow is out of scope for this feature
and handled via existing transaction management).
