-- 075_advance_payments_add_salary.up.sql
-- Re-add the `salary` column for the self-check-in advance flow.
-- (The column previously existed and was dropped in 064; it now has a clear purpose.)
--
-- salary = total earned wages from check-in/check-out (100%). The advanceable cap
-- max_adv_amount is then derived as floor(salary * SelfCheckInAdvanceablePercent / 100)
-- = 70% of salary (constant lives in backend/internal/domain/advance_payment.go).
--
-- Applies only to check-in-enabled employees. Admin-upload (BCC) rows leave salary
-- unused and keep their admin-set max_adv_amount (100% of BCC).
--
-- Backfill: historical rows already store the earned amount in max_adv_amount (100%).
-- Seed salary = max_adv_amount so the next checkout's recompute does not lose prior
-- earnings. Idempotent guard (salary = 0) makes this re-runnable.

ALTER TABLE advance_payments
    ADD COLUMN salary BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Total earned salary from check-in/out (100%); max_adv_amount = floor(salary * 70 / 100)';

UPDATE advance_payments
SET salary = max_adv_amount
WHERE salary = 0;
