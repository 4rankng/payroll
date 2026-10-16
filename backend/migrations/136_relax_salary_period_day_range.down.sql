-- Restore the historical 0-28 day range.
-- NOTE: rows configured with days 29-31 after the upgrade must be reset by
-- hand first, otherwise the CHECK re-addition fails on those rows.
ALTER TABLE projects
  DROP CHECK chk_salary_period_from_range,
  DROP CHECK chk_salary_period_to_range,
  ADD CONSTRAINT chk_salary_period_from_range CHECK (salary_period_from >= 0 AND salary_period_from <= 28),
  ADD CONSTRAINT chk_salary_period_to_range CHECK (salary_period_to >= 0 AND salary_period_to <= 28);
