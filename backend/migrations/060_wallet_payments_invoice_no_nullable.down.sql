-- 060_wallet_payments_invoice_no_nullable.down.sql
ALTER TABLE wallet_payments
    MODIFY COLUMN invoice_no VARCHAR(128) NOT NULL DEFAULT '';
