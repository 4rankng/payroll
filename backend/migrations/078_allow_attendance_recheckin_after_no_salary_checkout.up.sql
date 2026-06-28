ALTER TABLE attendances DROP INDEX uq_employee_date;
CREATE INDEX idx_attendances_employee_date ON attendances (employee_id, date);
