-- 069_wallet_payments_resolution_source.up.sql
-- Track what triggered a terminal state transition: ipn, status_inquiry, manual, or reconciliation.

ALTER TABLE wallet_payments
    ADD COLUMN resolution_source VARCHAR(32) NULL
        COMMENT 'ipn, status_inquiry, manual, or reconciliation'
        AFTER reconciled_at;

-- Backfill: every row already in a terminal state was resolved by IPN
-- (the only resolution path that existed before this column).
UPDATE wallet_payments
    SET resolution_source = 'ipn'
 WHERE status IN ('completed', 'failed', 'reversed')
   AND resolution_source IS NULL;
