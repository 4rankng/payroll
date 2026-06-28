-- Order matters: fk_attendances_employee (employee_id -> employees.id) requires an index
-- whose leading column is employee_id. uq_employee_date is the only such index, so it cannot
-- be dropped until the replacement idx_attendances_employee_date exists. Create first, then drop.
CREATE INDEX idx_attendances_employee_date ON attendances (employee_id, date);
ALTER TABLE attendances DROP INDEX uq_employee_date;
