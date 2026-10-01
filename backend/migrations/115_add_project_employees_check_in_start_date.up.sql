-- Record the day the self check-in service started for each assignment.
--
-- Why
-- ---
-- `check_in_effective_from` (migration 104) only exists while an enable is
-- waiting: ApplyPendingCheckIn clears it the moment the row activates. So once
-- an employee is live under self check-in there is no column left that says
-- when it started, and the admin roster export cannot report a start date.
--
-- `check_in_start_date` is the durable counterpart: it is written when the
-- service activates (immediately for a "this month" start, or by the nightly
-- pending sweep for a "next month" start) and is never cleared on disable, so
-- the export keeps reporting the real start day of every enabled employee.
--
-- Backfill
-- --------
-- Rows that are already enabled predate this column, so their start day is
-- recovered from the earliest attendance they ever recorded for the
-- assignment — the first day the employee could actually have used the
-- service. Enabled rows with no attendance keep NULL: the start day is
-- genuinely unknown and the export leaves that cell empty rather than
-- inventing a date. Pending rows keep NULL as well; their start date is
-- CheckInEffectiveFrom until the sweep activates them.
--
-- Safe to re-run: the column is added only while absent, and the backfill only
-- touches rows whose start date is still NULL.

SET @schema_115 = (
    SELECT IF(
        COUNT(*) = 0,
        'ALTER TABLE project_employees ADD COLUMN check_in_start_date DATE NULL DEFAULT NULL COMMENT ''Day the self check-in service started (day 1 of the chosen month)''',
        'SELECT ''check_in_start_date already present'''
    )
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'project_employees'
      AND column_name = 'check_in_start_date'
);

PREPARE schema_115_stmt FROM @schema_115;
EXECUTE schema_115_stmt;
DEALLOCATE PREPARE schema_115_stmt;

UPDATE project_employees AS pe
LEFT JOIN (
    SELECT project_id, employee_id, DATE(MIN(check_in_time)) AS first_check_in
    FROM attendances
    GROUP BY project_id, employee_id
) AS first_use
    ON first_use.project_id = pe.project_id
   AND first_use.employee_id = pe.employee_id
SET pe.check_in_start_date = first_use.first_check_in
WHERE pe.check_in_enabled = 1
  AND pe.check_in_start_date IS NULL;

-- Verify: every enabled row with attendance history has a start date.
SELECT
    (SELECT COUNT(*)
       FROM project_employees
      WHERE check_in_enabled = 1 AND check_in_start_date IS NOT NULL) AS enabled_with_start_date,
    (SELECT COUNT(*)
       FROM project_employees
      WHERE check_in_enabled = 1 AND check_in_start_date IS NULL) AS enabled_without_start_date;
