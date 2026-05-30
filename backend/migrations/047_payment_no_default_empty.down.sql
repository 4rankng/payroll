-- +migrate Down
ALTER TABLE provider_transactions
    MODIFY COLUMN payment_no VARCHAR(128) DEFAULT NULL;
