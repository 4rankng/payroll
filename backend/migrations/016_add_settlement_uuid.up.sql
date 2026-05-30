-- Add settlement_uuid column to settlements table for idempotency
ALTER TABLE `settlements`
ADD COLUMN `settlement_uuid` varchar(36) DEFAULT NULL COMMENT 'UUID for idempotency' AFTER `id`,
ADD UNIQUE KEY `idx_settlement_uuid` (`settlement_uuid`);
