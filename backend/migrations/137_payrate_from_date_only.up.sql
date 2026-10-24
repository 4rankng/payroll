-- From-date-only payrate model: stop materializing end dates.
--
-- to_date is no longer written by the application; a config's end is implied
-- by its successor (resolution: latest from_date <= work date). This migration
-- removes the constraints that enforced the old open-config bookkeeping and
-- codifies the one surviving invariant (one live row per project per
-- from_date) for environments that do not have it yet.
--
-- DEPLOY ORDER: apply BEFORE serving the new build. The old build closes the
-- predecessor before inserting a replacement (safe under either schema); the
-- new build inserts open-ended rows without closing anything, which would
-- violate uq_payrates_one_open_per_project while that index still exists.
--
-- Idempotent by design: every statement is guarded via INFORMATION_SCHEMA, so
-- it can run on environments where the prod schema was applied manually.

SET @drop_open_index := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND index_name = 'uq_payrates_one_open_per_project'
    ),
    'ALTER TABLE payrates DROP INDEX uq_payrates_one_open_per_project',
    'SELECT 1'
  )
);
PREPARE drop_open_index_stmt FROM @drop_open_index;
EXECUTE drop_open_index_stmt;
DEALLOCATE PREPARE drop_open_index_stmt;

SET @drop_open_column := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND column_name = 'open_project_id'
    ),
    'ALTER TABLE payrates DROP COLUMN open_project_id',
    'SELECT 1'
  )
);
PREPARE drop_open_column_stmt FROM @drop_open_column;
EXECUTE drop_open_column_stmt;
DEALLOCATE PREPARE drop_open_column_stmt;

-- Surviving invariant: at most one live row per (project, from_date). The
-- generated column maps soft-deleted rows to NULL so the unique index only
-- sees live ones.
SET @add_live_column := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.columns
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND column_name = 'live_from_date'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD COLUMN live_from_date DATE GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN from_date ELSE NULL END) STORED'
  )
);
PREPARE add_live_column_stmt FROM @add_live_column;
EXECUTE add_live_column_stmt;
DEALLOCATE PREPARE add_live_column_stmt;

SET @add_live_index := (
  SELECT IF(
    EXISTS(
      SELECT 1 FROM information_schema.statistics
      WHERE table_schema = DATABASE() AND table_name = 'payrates'
        AND index_name = 'uq_payrates_live_project_from_date'
    ),
    'SELECT 1',
    'ALTER TABLE payrates ADD UNIQUE INDEX uq_payrates_live_project_from_date (project_id, live_from_date)'
  )
);
PREPARE add_live_index_stmt FROM @add_live_index;
EXECUTE add_live_index_stmt;
DEALLOCATE PREPARE add_live_index_stmt;
