ALTER TABLE `bulk_transfer_files`
  ADD COLUMN `transfer_at` datetime DEFAULT NULL AFTER `transfer_result`;

UPDATE bulk_transfer_files
JOIN assets ON bulk_transfer_files.transfer_result = assets.id
SET bulk_transfer_files.transfer_at = assets.created_at;
