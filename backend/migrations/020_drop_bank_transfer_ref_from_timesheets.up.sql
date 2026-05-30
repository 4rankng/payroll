-- Drop bank_transfer_ref column and its index from timesheets
ALTER TABLE `timesheets`
  DROP INDEX `idx_bank_transfer_ref`,
  DROP COLUMN `bank_transfer_ref`;

