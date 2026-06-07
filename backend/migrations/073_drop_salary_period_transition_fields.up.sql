-- Remove dead salary period transition fields — new salary period handles transitions naturally
ALTER TABLE projects DROP COLUMN last_salary_from, DROP COLUMN last_salary_to;
