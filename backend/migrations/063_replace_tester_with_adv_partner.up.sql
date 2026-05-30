-- Migration: 063_replace_tester_with_adv_partner.up.sql
--
-- Replaces the unused 'tester' role with 'adv_partner' in the users.role enum.
-- adv_partner is for external partners managing the advance payment pipeline.
--
-- Prerequisite: any existing tester users must be reassigned or deleted before running.
-- This migration will fail if rows with role='tester' still exist.

-- Safety check: fail fast if tester users still exist
SET @tester_count = (SELECT COUNT(*) FROM `users` WHERE `role` = 'tester');
SET @sql = IF(@tester_count > 0,
    CONCAT('SELECT CONCAT("ABORT: ', @tester_count, ' users still have role=tester. Reassign or delete them first.") AS msg'),
    'SELECT "No tester users found, proceeding" AS msg'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Only alter if tester users don't exist
ALTER TABLE `users`
MODIFY COLUMN `role` enum('admin','partner','employee','adv_partner')
    CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
    NOT NULL DEFAULT 'employee';
