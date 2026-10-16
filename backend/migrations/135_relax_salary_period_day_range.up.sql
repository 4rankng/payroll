-- Allow full month-end salary period days (29-31) on projects.
-- Days 29-31 are real period boundaries (e.g. a cycle starting on the 29th of
-- the previous month). A day missing in a short month (Feb 29/30/31) rolls
-- forward to the next existing day when the window is resolved. The old 0-28
-- cap rejected these valid configs at the database level.
ALTER TABLE projects
  DROP CHECK chk_salary_period_from_range,
  DROP CHECK chk_salary_period_to_range,
  ADD CONSTRAINT chk_salary_period_from_range CHECK (salary_period_from >= 0 AND salary_period_from <= 31),
  ADD CONSTRAINT chk_salary_period_to_range CHECK (salary_period_to >= 0 AND salary_period_to <= 31);
