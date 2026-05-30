-- Remove description, reference, and url columns from ledger_entries table
-- These fields are being removed to simplify the ledger entry structure

ALTER TABLE `ledger_entries` DROP COLUMN `description`;
ALTER TABLE `ledger_entries` DROP COLUMN `reference`;
ALTER TABLE `ledger_entries` DROP COLUMN `url`;
