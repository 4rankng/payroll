-- 070_wallet_payments_provider_status_updated_idx.up.sql
-- Supports ListStaleAuthorised which filters on (provider, status, updated_at)
-- and orders by updated_at ASC. Existing idx_wp_provider_status uses created_at
-- instead of updated_at, so MySQL cannot use it for the updated_at filter/order.

CREATE INDEX idx_wp_provider_status_updated
    ON wallet_payments (provider, status, updated_at);
