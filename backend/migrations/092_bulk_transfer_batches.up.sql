-- bulk_transfer_batches: one row per admin upload of a "Yêu cầu chuyển tiền"
-- .xlsx (exported from /admin/timesheet via the "Chuyển OnePay" button).
-- Parsed rows are persisted in `data` JSON so the outbox sweeper and the KQ
-- generator can re-read them without re-parsing the (potentially large) Excel.
--
-- Lifecycle:
--   pending → processing → completing → completed
--                                  ↘ failed
-- `enqueue_state` is the batch-level outbox: 'pending' until per-row asynq
-- tasks have been enqueued, then 'enqueued'. A sweeper cron flips stuck
-- 'pending' rows back to enqueued using the persisted `data`.
--
-- Fee accounting:
--   total_fee = SUM(wallet_payments.fee) over rows in this batch.
--   ledger_txn_id points at the single Expense transaction that books the
--   aggregate OnePay fee. NULL until the book_batch_ledger task runs.
CREATE TABLE bulk_transfer_batches (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    filename         VARCHAR(255) NOT NULL,
    content_hash     CHAR(64) NOT NULL,
    source           VARCHAR(16) NOT NULL DEFAULT 'wallet_upload',
    status           VARCHAR(32) NOT NULL DEFAULT 'pending',
    enqueue_state    VARCHAR(16) NOT NULL DEFAULT 'pending',
    total_count      INT NOT NULL DEFAULT 0,
    success_count    INT NOT NULL DEFAULT 0,
    failed_count     INT NOT NULL DEFAULT 0,
    transfer_amount  BIGINT NOT NULL DEFAULT 0,
    total_fee        BIGINT NOT NULL DEFAULT 0,
    ledger_txn_id    BIGINT UNSIGNED NULL,
    fee_booked_at    DATETIME(3) NULL,
    data             JSON NOT NULL,
    asset_id         BIGINT UNSIGNED NULL,
    created_by       BIGINT UNSIGNED NOT NULL,
    created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    completed_at     DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_bulk_transfer_batches_content_hash (content_hash),
    KEY idx_bulk_transfer_batches_status (status),
    KEY idx_bulk_transfer_batches_enqueue_state (enqueue_state),
    KEY idx_bulk_transfer_batches_ledger_txn_id (ledger_txn_id),
    KEY idx_bulk_transfer_batches_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
