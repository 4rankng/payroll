-- Reverses 110_add_employees_mobile_index.up.sql.
-- The index carries no constraints, so dropping it cannot fail on data.
DROP INDEX idx_employees_mobile ON employees;
