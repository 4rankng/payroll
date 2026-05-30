-- 057_wallet_provider.up.sql

-- wallet_payments ----------------------------------------------------
ALTER TABLE wallet_payments
    ADD COLUMN provider VARCHAR(16) NOT NULL DEFAULT '9pay'
        AFTER invoice_no;

ALTER TABLE wallet_payments
    DROP INDEX uk_wp_invoice_no;
ALTER TABLE wallet_payments
    ADD UNIQUE INDEX uk_wp_provider_invoice_no (provider, invoice_no);
ALTER TABLE wallet_payments
    ADD INDEX idx_wp_provider_status (provider, status, created_at);

-- wallet_topups ------------------------------------------------------
ALTER TABLE wallet_topups
    ADD COLUMN provider VARCHAR(16) NOT NULL DEFAULT '9pay'
        AFTER id;
ALTER TABLE wallet_topups
    ADD INDEX idx_wt_provider_created (provider, created_at);

-- wallet_ipn ---------------------------------------------------------
-- already has provider column; nothing to do.
