-- +migrate Down
ALTER TABLE advance_payments ADD COLUMN salary BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Full salary amount for the month';
