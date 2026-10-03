-- Drop the project date bounds (projects.start_date / projects.end_date).
--
-- Why
-- ---
-- Projects are no longer bounded by dates. Nothing in the business flow reads
-- these columns anymore: attendance, timesheets, payroll and BCC import all
-- gate on project_status, auto-activation promotes every draft regardless of
-- start date, and the end-date auto-complete job was dead code. The UI no
-- longer collects the fields, so the columns only hold stale values.
--
-- Destructive: the historical dates are NOT preserved anywhere. The down
-- migration restores the columns (nullable) but not their data — run
-- `make backup` before applying this to production.

ALTER TABLE projects
    DROP COLUMN start_date,
    DROP COLUMN end_date;
