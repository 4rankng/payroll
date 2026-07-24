-- +migrate Up
--
-- employees: track OnePay bank-account validation outcome.
--
-- bank_account_status: 'valid' (default; also covers all pre-existing rows
--   per assumption that existing accounts are already valid), 'invalid'
--   (OnePay confirmed the account is bad or the holder name mismatches),
--   'unverified' (OnePay unreachable / timed out — fail-open, not shown in
--   the warning list).
--
-- Only rows with status = 'invalid' are surfaced in the
-- /employees/missing-bank-details list (alongside truly missing info).
-- 'unverified' is deliberately excluded so OnePay outages do not flood
-- admins with false alarms.
ALTER TABLE `employees`
  ADD COLUMN `bank_account_status` VARCHAR(20) NOT NULL DEFAULT 'valid'
    COMMENT 'valid|invalid|unverified — outcome of OnePay account check',
  ADD COLUMN `bank_account_invalid_reason` VARCHAR(500) NULL
    COMMENT 'Vietnamese human-readable reason when status=invalid',
  ADD COLUMN `bank_account_validated_at` DATETIME NULL
    COMMENT 'When the last OnePay validation was performed';
