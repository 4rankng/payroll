-- Remove UNIQUE constraint on checksum to allow multiple asset records with same checksum
-- This enables file deduplication where different filenames can point to the same physical file

ALTER TABLE `assets` DROP INDEX `idx_assets_checksum_upload_type`;
