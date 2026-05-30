-- 060_wallet_payments_invoice_no_nullable.up.sql
-- Make invoice_no nullable so the composite unique index uk_wp_provider_invoice_no
-- does not block concurrent rows that have not yet received a provider reference.
-- MySQL unique indexes allow multiple NULL values per key.

ALTER TABLE wallet_payments
    MODIFY COLUMN invoice_no VARCHAR(128) NULL DEFAULT NULL;

UPDATE wallet_payments SET invoice_no = NULL WHERE invoice_no = '';
