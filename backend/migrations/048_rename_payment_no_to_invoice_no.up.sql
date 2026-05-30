-- +migrate Up
--
-- Rename provider_transactions.payment_no → invoice_no.
--
-- 9pay's support correspondence consistently calls this field
-- `invoice_no`, while the Postman collection / public API docs call
-- it `payment_no`. They are the same value. We standardize internally
-- on `invoice_no` to match the human-facing 9pay vocabulary; the wire
-- layer (ninepay.* JSON tags) keeps `payment_no` unchanged because
-- that is what 9pay's API actually emits.
--
-- The CHANGE COLUMN form preserves all data — every existing row's
-- value is carried over to the new column verbatim. The DEFAULT '' /
-- NOT NULL constraint is kept (introduced in migration 047) so the
-- empty-string semantics for "not yet known" stays in force.
ALTER TABLE provider_transactions
    CHANGE COLUMN payment_no invoice_no VARCHAR(128) NOT NULL DEFAULT '';

ALTER TABLE provider_transactions
    RENAME INDEX idx_pt_payment_no TO idx_pt_invoice_no;
