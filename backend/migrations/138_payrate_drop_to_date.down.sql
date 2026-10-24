-- Restore what 138_payrate_drop_to_date.up.sql removed.
SET @add_to_date := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND column_name = 'to_date'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD COLUMN to_date DATE NULL'
  )
);
PREPARE add_to_date_stmt FROM @add_to_date;
EXECUTE add_to_date_stmt;
DEALLOCATE PREPARE add_to_date_stmt;

SET @add_check := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.table_constraints
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND constraint_name = 'chk_payrates_date_order'
        AND constraint_type = 'CHECK'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD CONSTRAINT chk_payrates_date_order CHECK (to_date IS NULL OR to_date >= from_date)'
  )
);
PREPARE add_check_stmt FROM @add_check;
EXECUTE add_check_stmt;
DEALLOCATE PREPARE add_check_stmt;
