-- 057_wallet_provider.down.sql

-- wallet_topups
ALTER TABLE wallet_topups DROP INDEX idx_wt_provider_created;
ALTER TABLE wallet_topups DROP COLUMN provider;

-- wallet_payments
ALTER TABLE wallet_payments DROP INDEX idx_wp_provider_status;
ALTER TABLE wallet_payments DROP INDEX uk_wp_provider_invoice_no;
ALTER TABLE wallet_payments ADD UNIQUE INDEX uk_wp_invoice_no (invoice_no);
ALTER TABLE wallet_payments DROP COLUMN provider;
