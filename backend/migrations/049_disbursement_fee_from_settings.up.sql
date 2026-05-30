-- +migrate Up
--
-- Disbursement fee bookkeeping move: drop the redundant charged_amount
-- column, make fee NOT NULL DEFAULT 0, and seed the new
-- disbursement_fee_vnd setting.
--
-- Why charged_amount goes:
--   9pay's IPN payload reports an `amount` field that always equals our
--   `requested_amount` (verified empirically against production rows —
--   see commit history). The 200 VND per-transfer fee 9pay deducts is
--   settled on the merchant balance and is not surfaced on the IPN, so
--   `charged_amount` carried no information beyond `requested_amount`.
--   Drop it; callers that previously summed total_charged_amount get
--   total_requested_amount instead.
--
-- Why fee becomes NOT NULL DEFAULT 0:
--   The new flow stamps the disbursement-provider fee at INSERT time
--   from the disbursement_fee_vnd setting (default 200 VND). Historical
--   rows wrote NULL into fee because the IPN doesn't report it; backfill
--   them to 0 so the NOT NULL constraint can be applied without losing
--   any rows. Future writers always set a numeric value at INSERT.
--
-- Why disbursement_fee_vnd lives in settings:
--   The fee is configurable per deployment and may change if 9pay
--   adjusts pricing. Stamping it on the row at INSERT time means each
--   historical transaction preserves the fee in effect at the time of
--   the transfer, even if the setting is updated later.

UPDATE provider_transactions SET fee = 0 WHERE fee IS NULL;

ALTER TABLE provider_transactions
    DROP COLUMN charged_amount,
    MODIFY COLUMN fee BIGINT NOT NULL DEFAULT 0;

-- Seed the disbursement-provider fee at 200 VND (current 9pay rate).
-- Idempotent: re-running the migration leaves an admin-edited value
-- alone. ON DUPLICATE KEY UPDATE `key` = `key` is the standard "no-op
-- on conflict" pattern used by migration 044.
INSERT INTO settings (`key`, `value`, `value_type`)
VALUES ('disbursement_fee_vnd', '200', 'number')
ON DUPLICATE KEY UPDATE `key` = `key`;
