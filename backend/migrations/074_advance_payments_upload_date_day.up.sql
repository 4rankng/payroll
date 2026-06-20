-- 074_advance_payments_upload_date_day.up.sql
-- upload_date historically stored the upload CALENDAR MONTH (YYYY-MM, VARCHAR(7))
-- even though the column is named "date". Widen it to the actual upload DAY
-- (YYYY-MM-DD, VARCHAR(10)) so two upload rounds for the same for_month on
-- different days can coexist under the (employee, project, for_month, upload_date)
-- unique key instead of colliding and overwriting each other's max_adv_amount.
--
-- Backfill is safe + idempotent: upload_date-month always equals created_at-month
-- across both write sites (FlexPay import and the attendance path), so deriving
-- the day from created_at can never create a unique-key collision. The
-- CHAR_LENGTH(upload_date) = 7 guard makes this re-runnable.

ALTER TABLE advance_payments
    MODIFY COLUMN upload_date VARCHAR(10) NOT NULL COMMENT 'Upload date in YYYY-MM-DD format';

UPDATE advance_payments
SET upload_date = DATE_FORMAT(created_at, '%Y-%m-%d')
WHERE CHAR_LENGTH(upload_date) = 7;
