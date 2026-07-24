-- +migrate Down
ALTER TABLE `employees`
  DROP COLUMN `bank_account_validated_at`,
  DROP COLUMN `bank_account_invalid_reason`,
  DROP COLUMN `bank_account_status`;
