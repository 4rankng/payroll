-- From-date-only model, phase 2: drop the retired to_date bookkeeping.
--
-- APPLY GATE — DO NOT RUN until production has served the from-date-only
-- build through at least one full pay cycle (applied 137 on 2026-10-24; the
-- Kỳ-4 payout lands ~2026-11-01, so earliest sensible date is on/after that
-- payout is confirmed). Two rollback paths depend on this column surviving
-- until then:
--   1. rolling the backend container back to the pre-from-date-only image —
--      the old code reads to_date in every rate-resolution query, and with
--      the column gone those queries hard-error (payroll down, not degraded);
--   2. migration 137's down script re-creates the open_project_id generated
--      column from to_date and cannot run without it.
-- The application never reads or writes to_date since 137/the from-date-only
-- build; responses derive end dates from each row's successor.

SET @drop_check := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND constraint_name = 'chk_payrates_date_order'
        AND constraint_type = 'CHECK'
    ),
    'ALTER TABLE payrates DROP CHECK chk_payrates_date_order',
    'SELECT 1'
  )
);
PREPARE drop_check_stmt FROM @drop_check;
EXECUTE drop_check_stmt;
DEALLOCATE PREPARE drop_check_stmt;

SET @drop_to_date := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND column_name = 'to_date'
    ),
    'ALTER TABLE payrates DROP COLUMN to_date',
    'SELECT 1'
  )
);
PREPARE drop_to_date_stmt FROM @drop_to_date;
EXECUTE drop_to_date_stmt;
DEALLOCATE PREPARE drop_to_date_stmt;
