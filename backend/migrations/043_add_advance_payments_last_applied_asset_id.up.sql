-- +migrate Up

-- Per-row idempotency for the flexible payroll top-up pipeline. Each row
-- stamps the asset (uploaded file) that most recently contributed to it.
-- Re-applying the same file is a no-op: the upsert's ON DUPLICATE KEY
-- UPDATE compares advance_payments.last_applied_asset_id to the incoming
-- asset id and skips the salary / max_adv_amount addition when they match.
-- ON DELETE SET NULL keeps payment data intact if an asset is later
-- archived.
ALTER TABLE advance_payments
ADD COLUMN last_applied_asset_id BIGINT UNSIGNED NULL DEFAULT NULL
    COMMENT 'assets.id of the most recent flex-pay file applied to this row; the upsert skips top-up when the incoming asset matches',
ADD CONSTRAINT fk_adv_pay_last_applied_asset FOREIGN KEY (last_applied_asset_id)
    REFERENCES assets(id) ON DELETE SET NULL;
