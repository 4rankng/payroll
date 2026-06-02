-- Add last_salary_from and last_salary_to columns to handle salary period transitions
-- These fields store the previous salary period configuration when a project transitions
-- from one pay cycle to another (e.g., Lear: 21->20 to 1->end-of-month)
ALTER TABLE projects
ADD COLUMN last_salary_from INT NULL COMMENT 'Previous salary period start day during transition (NULL = no transition)',
ADD COLUMN last_salary_to INT NULL COMMENT 'Previous salary period end day during transition (NULL = no transition)';
