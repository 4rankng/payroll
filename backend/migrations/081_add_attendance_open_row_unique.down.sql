-- Reverse of 081. Drop the partial unique first (it references open_key), then the column.
ALTER TABLE attendances DROP INDEX uq_attendances_employee_open;
ALTER TABLE attendances DROP COLUMN open_key;
