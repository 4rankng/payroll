-- +migrate Up

-- Add settlement_transaction_id to link requests to their settlement transaction
-- This provides full traceability: Settlement File → Request IDs → Transaction → Ledger Entries
ALTER TABLE advance_payment_requests
ADD COLUMN settlement_transaction_id BIGINT UNSIGNED NULL COMMENT 'Links to transaction for tracing settlement from settlement file to ledger entries',
ADD INDEX idx_apr_settlement_txn (settlement_transaction_id),
ADD CONSTRAINT fk_apr_settlement_txn FOREIGN KEY (settlement_transaction_id)
    REFERENCES transactions(id) ON DELETE SET NULL;
