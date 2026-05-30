-- Add revenue tracking columns to timesheets table
ALTER TABLE timesheets
ADD COLUMN revenue_receivable BIGINT UNSIGNED DEFAULT 0 COMMENT 'Revenue to receive from client for advance cash service fee',
ADD COLUMN revenue_paid TINYINT(1) DEFAULT 0 COMMENT 'Whether revenue has been received from client';

-- Fix reversed_transaction_id column type to match id column for foreign key constraint
-- Change from bigint to bigint unsigned

ALTER TABLE `transactions`
MODIFY COLUMN `reversed_transaction_id` bigint unsigned DEFAULT NULL;

-- Add foreign key constraint for self-referencing transactions
ALTER TABLE `transactions`
ADD CONSTRAINT `fk_transactions_reversed_by_transaction`
FOREIGN KEY (`reversed_transaction_id`)
REFERENCES `transactions`(`id`)
ON DELETE SET NULL;
