-- Drop the durable self check-in start date.
--
-- Data loss is limited to the recorded start day: activations recorded in
-- `check_in_start_date` after this migration are simply forgotten, and
-- `check_in_effective_from` (pending enables) is untouched.
ALTER TABLE project_employees
  DROP COLUMN check_in_start_date;
