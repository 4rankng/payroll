-- +migrate Up
ALTER TABLE bulk_transfer_files
    ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'manual',
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'exported',
    ADD COLUMN uploaded_at TIMESTAMP NULL;

ALTER TABLE wallet_payments ADD COLUMN batch_id VARCHAR(64) DEFAULT NULL;

ALTER TABLE wallet_payments ADD INDEX idx_wallet_payments_batch_id (batch_id);
