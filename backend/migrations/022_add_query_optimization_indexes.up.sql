-- Add performance optimization indexes for query improvements
-- Migration: 022_add_query_optimization_indexes.up.sql

-- Projects table indexes for access control and search
-- Note: description is TEXT column, can't be indexed without length prefix
CREATE INDEX idx_projects_search_name ON projects(name);
CREATE INDEX idx_projects_search_code ON projects(code);
CREATE INDEX idx_projects_search_client ON projects(client_name);
CREATE INDEX idx_projects_status_created_by ON projects(project_status, created_by);
CREATE INDEX idx_projects_created_by_dates ON projects(created_by, created_at, updated_at);
CREATE INDEX idx_projects_access_control ON projects(created_by, project_status);

-- Audit logs table indexes - simplified based on actual table structure
CREATE INDEX idx_audit_logs_user_created_at ON audit_logs(user_id, created_at);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- Blacklisted tokens table indexes for expiration and reason filtering
CREATE INDEX idx_blacklisted_tokens_expires ON blacklisted_tokens(expires_at);
CREATE INDEX idx_blacklisted_tokens_reason_date ON blacklisted_tokens(reason, blacklisted_at);
CREATE INDEX idx_blacklisted_tokens_user_expires ON blacklisted_tokens(user_id, expires_at);

-- Project users table for access control optimization
CREATE INDEX idx_project_users_user_access ON project_users(user_id, project_id, deleted_at);

-- Project employees table for partner access control optimization
CREATE INDEX idx_project_employees_employee_access ON project_employees(employee_id, created_by, deleted_at);
CREATE INDEX idx_project_employees_project_created_by ON project_employees(project_id, created_by, deleted_at);

-- Employee users table for access control optimization
CREATE INDEX idx_employee_users_user_employee ON employee_users(user_id, employee_id, deleted_at);