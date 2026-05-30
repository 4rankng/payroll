ALTER TABLE `bulk_transfer_files`
  ADD COLUMN `completed_count` int NOT NULL DEFAULT 0 AFTER `transactions_count`,
  ADD COLUMN `failed_count` int NOT NULL DEFAULT 0 AFTER `completed_count`;

