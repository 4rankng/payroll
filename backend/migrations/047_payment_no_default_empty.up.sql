-- +migrate Up
--
-- Make provider_transactions.payment_no NOT NULL DEFAULT ''.
--
-- 9pay calls this value `payment_no` in the Postman collection but
-- `invoice_no` in some support correspondence — same field, different
-- vocabulary. We standardize on `payment_no` to match the wire shape.
--
-- The previous schema (NULL default) tracked "no provider reference yet"
-- as NULL, but this surfaced two operational papercuts:
--
--   1) The 9pay synchronous /disbursement/create response carries
--      payment_no=0 on the initial accept (before 9pay assigns a real
--      reference), which the provider layer used to coerce into the
--      literal string "0" — polluting the payment_no idempotency key
--      with rows that all share "0". The provider layer now treats 0
--      as no-ref, but the DB column should make the empty-default
--      semantics explicit so future writers don't have to remember the
--      NULL convention.
--
--   2) Mixed NULL / "" / "0" semantics across rows make grep / SELECT
--      WHERE payment_no = ? checks brittle. Standardize on the empty
--      string for "not yet known"; the IPN writes the real value when
--      it arrives.
--
-- Backfill any existing NULL or "0" rows to '' before the constraint
-- change so the ALTER does not fail.
UPDATE provider_transactions
SET    payment_no = ''
WHERE  payment_no IS NULL
   OR  payment_no = '0';

ALTER TABLE provider_transactions
    MODIFY COLUMN payment_no VARCHAR(128) NOT NULL DEFAULT '';
