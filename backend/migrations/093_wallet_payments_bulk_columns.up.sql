-- Bulk-transfer linkage columns on wallet_payments.
--
-- Each row produced by the bulk-transfer worker (wallet:bulk_transfer_row)
-- stamps these so we can SUM(fee) GROUP BY bulk_transfer_batch_id when
-- booking the aggregate Expense ledger entry, and so the KQ Excel can
-- render each row in original input order via bulk_transfer_order.
--
-- `vfic_code` mirrors request_id for filtered lookup (request_id already
-- has idx_wp_request_id UNIQUE; this secondary index is for admin search
-- by VFIC without hitting the unique index).
--
-- `enqueue_state` defaults to 'enqueued' because by the time a
-- wallet_payments row exists, the worker has already accepted the task
-- (the sweeper operates at the batch level, not the row level).
ALTER TABLE wallet_payments
    ADD COLUMN bulk_transfer_batch_id BIGINT UNSIGNED NULL,
    ADD COLUMN bulk_transfer_order    INT UNSIGNED NULL,
    ADD COLUMN vfic_code              VARCHAR(32) NULL,
    ADD COLUMN enqueue_state          VARCHAR(16) NULL DEFAULT 'enqueued',
    ADD COLUMN sweeper_retry_count    INT UNSIGNED NOT NULL DEFAULT 0,
    ADD INDEX idx_wp_bulk_batch (bulk_transfer_batch_id),
    ADD INDEX idx_wp_vfic_code (vfic_code);
