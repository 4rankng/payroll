-- Core composite index for grouped query JOIN pattern
CREATE INDEX idx_timesheets_employee_project_date
  ON timesheets(employee_id, project_id, date, deleted_at);

-- Covers employee + date range queries + status filtering
CREATE INDEX idx_timesheets_employee_date_status
  ON timesheets(employee_id, date, deleted_at, timesheet_status);

-- Covers EXISTS subqueries in access control
CREATE INDEX idx_project_users_access
  ON project_users(project_id, user_id, deleted_at);

-- Covers correlated subquery for employee_code (ORDER BY created_at DESC LIMIT 1)
CREATE INDEX idx_project_employees_employee_created
  ON project_employees(employee_id, created_at DESC, deleted_at);
