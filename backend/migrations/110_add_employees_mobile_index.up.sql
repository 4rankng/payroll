-- Add a lookup index for employees.mobile.
--
-- Why
-- ---
-- The employee mobile number is an authentication identifier, not just contact
-- data: it drives Zalo password reset (zaloreset resolves employee -> user by
-- phone), phone-based login (auth_service falls back to employees.mobile), and
-- FlexPay ZNS delivery. Every one of those paths filters `WHERE mobile = ?`, and
-- `employees.mobile` had no index at all — EXPLAIN reported `type: ALL` over the
-- whole table (1.491 rows at the time of writing, growing with headcount), while
-- the equivalent `users.mobile` column carries `unique_user_mobile_deleted_at`.
--
-- Why a *plain* index (not unique)
-- --------------------------------
-- Duplicate mobile numbers exist in production data today (21 groups / 43 rows
-- at the time of writing), and the schema deliberately permits them: identity is
-- CCCD, and a shared family phone is a legitimate entry. A unique index would
-- either reject live data or force a data-cleanup decision that is not a
-- prerequisite for this performance fix. Ambiguity is instead handled at the
-- read path: `EmployeeRepository.GetByMobile` fails closed with a conflict error
-- when one number maps to several employees, so no caller can silently act on an
-- arbitrary match.
--
-- Reversibility
-- -------------
-- Plain index, no data change: `DROP INDEX idx_employees_mobile ON employees;`
-- restores the previous state exactly.
--
-- Safe to re-run: the index is added only while absent.

SET @schema_110 = (
    SELECT IF(
        COUNT(*) = 0,
        'ALTER TABLE employees ADD INDEX idx_employees_mobile (mobile)',
        'SELECT ''idx_employees_mobile already present'''
    )
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'employees'
      AND index_name = 'idx_employees_mobile'
);

PREPARE schema_110_stmt FROM @schema_110;
EXECUTE schema_110_stmt;
DEALLOCATE PREPARE schema_110_stmt;

-- Verify: index present and leading on mobile.
SELECT
    (SELECT COUNT(*) FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'employees'
        AND index_name = 'idx_employees_mobile' AND seq_in_index = 1 AND column_name = 'mobile') AS mobile_index_present;
