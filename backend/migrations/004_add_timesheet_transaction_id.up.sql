-- Add transaction_id column to timesheets table
ALTER TABLE `timesheets`
  ADD COLUMN `transaction_id` BIGINT UNSIGNED NULL DEFAULT NULL
    COMMENT 'Linked revenue transaction for this timesheet',
  ADD KEY `idx_timesheets_transaction_id` (`transaction_id`),
  ADD CONSTRAINT `fk_timesheets_transaction`
    FOREIGN KEY (`transaction_id`)
    REFERENCES `transactions` (`id`)
    ON DELETE SET NULL;

