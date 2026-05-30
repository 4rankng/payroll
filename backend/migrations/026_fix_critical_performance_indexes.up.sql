-- Fix critical performance issues with composite indexes
-- Migration: 026_fix_critical_performance_indexes.up.sql

-- Critical composite index for complex timesheet access control queries
-- This covers the main query pattern combining date filters with access control
CREATE INDEX idx_timesheets_access_complex ON timesheets(
    deleted_at,
    employee_id,
    created_by,
    date,
    timesheet_status,
    payment_status,
    updated_at DESC
);

-- Composite index for project access control with user lookup
-- Optimizes queries filtering by created_by and project_status
CREATE INDEX idx_projects_access_user_lookup ON projects(
    deleted_at,
    created_by,
    project_status,
    created_at DESC
);

-- Composite index for project employees active counting with payment schedule
-- Replaces multiple subqueries with efficient single index access
CREATE INDEX idx_project_employees_active_count ON project_employees(
    project_id,
    last_date,
    payment_schedule,
    deleted_at
);

-- Composite index for project users access lookup
-- Optimizes user access checks for project permissions
CREATE INDEX idx_project_users_access_lookup ON project_users(
    user_id,
    project_id,
    deleted_at
);

-- Additional index for timesheet date range with access control
-- Covers date filtering combined with employee-based access control
CREATE INDEX idx_timesheets_date_access_control ON timesheets(
    date,
    deleted_at,
    employee_id,
    created_by,
    timesheet_status,
    payment_status
);

-- Index for employees with complex access patterns
-- Supports queries joining employees with project_employees and employee_users
CREATE INDEX idx_employees_complex_access ON employees(
    deleted_at,
    created_by,
    id,
    updated_at DESC
);