-- Migration: 045_add_tester_user_role.up.sql
--
-- Adds a fourth value 'tester' to the users.role enum. The tester role is
-- a constrained, QA-only role scoped to the 9pay disbursement debug rig
-- (/api/v1/admin/_debug/disbursement/*) and the basic auth flow. Casbin
-- denies it everything else by default; see configs/casbin_policy.csv for
-- the explicit allow list.
--
-- Idempotent: ALTER TABLE … MODIFY with the existing values + the new one
-- is safe to re-run; MySQL is a no-op when the column type already matches.

ALTER TABLE `users`
MODIFY COLUMN `role` enum('admin','partner','employee','tester')
    CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
    NOT NULL DEFAULT 'employee';
