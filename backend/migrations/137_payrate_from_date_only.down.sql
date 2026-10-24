-- Restore the open-config bookkeeping constraints removed by
-- 137_payrate_from_date_only.up.sql. The surviving from-date invariant
-- (live_from_date + uq_payrates_live_project_from_date) is kept: resolution
-- depends on it.
SET @add_open_column := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND column_name = 'open_project_id'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD COLUMN open_project_id BIGINT UNSIGNED GENERATED ALWAYS AS (CASE WHEN (deleted_at IS NULL AND to_date IS NULL) THEN project_id ELSE NULL END) STORED'
  )
);
PREPARE add_open_column_stmt FROM @add_open_column;
EXECUTE add_open_column_stmt;
DEALLOCATE PREPARE add_open_column_stmt;

SET @add_open_index := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND index_name = 'uq_payrates_one_open_per_project'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD UNIQUE INDEX uq_payrates_one_open_per_project (open_project_id)'
  )
);
PREPARE add_open_index_stmt FROM @add_open_index;
EXECUTE add_open_index_stmt;
DEALLOCATE PREPARE add_open_index_stmt;
