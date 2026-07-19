ALTER TABLE wallet_payments
    DROP INDEX idx_wp_vfic_code,
    DROP INDEX idx_wp_bulk_batch,
    DROP COLUMN sweeper_retry_count,
    DROP COLUMN enqueue_state,
    DROP COLUMN vfic_code,
    DROP COLUMN bulk_transfer_order,
    DROP COLUMN bulk_transfer_batch_id;
