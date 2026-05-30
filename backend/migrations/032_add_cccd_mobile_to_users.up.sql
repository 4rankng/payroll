-- Add CCCD and mobile columns to users table for admin/partner login
-- CCCD: unique per active user (soft-delete aware), used as login identifier
-- Mobile: unique per active user (soft-delete aware), used as login identifier

ALTER TABLE `users`
  ADD COLUMN `cccd`   varchar(20)  DEFAULT NULL COMMENT 'Citizen ID for admin/partner login',
  ADD COLUMN `mobile` varchar(15)  DEFAULT NULL COMMENT 'Mobile number for admin/partner login';

-- Unique index on CCCD (NULL values are excluded from uniqueness in MySQL)
ALTER TABLE `users`
  ADD UNIQUE KEY `unique_user_cccd_deleted_at` (`cccd`, `deleted_at`);

-- Regular index on mobile for fast lookup
ALTER TABLE `users`
  ADD KEY `idx_users_mobile` (`mobile`);

-- Add unique constraint on mobile for users table
-- Mobile must be unique per active user (soft-delete aware) to be used safely as a login identifier
-- NULL values are excluded from uniqueness in MySQL composite unique indexes

ALTER TABLE `users`
  DROP KEY `idx_users_mobile`,
  ADD UNIQUE KEY `unique_user_mobile_deleted_at` (`mobile`, `deleted_at`);
