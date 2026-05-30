-- +migrate Down
--
-- Restore the charged_amount column and the nullable fee column. Note
-- that data lost on the up migration cannot be recovered: rows whose
-- fee was NULL pre-migration are now 0, and charged_amount values are
-- gone entirely. The down path is structural-only.

ALTER TABLE provider_transactions
    ADD COLUMN charged_amount BIGINT DEFAULT NULL AFTER requested_amount,
    MODIFY COLUMN fee BIGINT DEFAULT NULL;

DELETE FROM settings WHERE `key` = 'disbursement_fee_vnd';
