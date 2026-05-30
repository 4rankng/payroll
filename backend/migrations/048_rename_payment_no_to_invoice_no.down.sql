-- +migrate Down
ALTER TABLE provider_transactions
    CHANGE COLUMN invoice_no payment_no VARCHAR(128) NOT NULL DEFAULT '';
