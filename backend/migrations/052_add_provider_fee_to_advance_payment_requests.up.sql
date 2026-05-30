ALTER TABLE advance_payment_requests ADD COLUMN provider_fee BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER fee;
