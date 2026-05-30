-- Optimize timesheet query performance
-- Migration: 025_optimize_timesheet_performance.up.sql

-- Critical composite index for timesheet access control queries
-- This index covers the main query pattern for filtering by edit status and ordering
CREATE INDEX idx_timesheets_access_control_edit ON timesheets(
    deleted_at,
    allowed_edit,
    request_edit_id,
    updated_at DESC
);

-- Additional index for timesheet date filtering with status
CREATE INDEX idx_timesheets_date_status ON timesheets(
    deleted_at,
    date,
    timesheet_status,
    payment_status
);

-- Index for employee access control optimization
CREATE INDEX idx_employees_access_control ON employees(
    id,
    deleted_at,
    created_by
);

-- Index for project employees partner access (supports EXISTS subqueries)
CREATE INDEX idx_project_employees_partner_access ON project_employees(
    employee_id,
    created_by,
    deleted_at
);

-- Index for employee users partner access (supports EXISTS subqueries)
CREATE INDEX idx_employee_users_partner_access ON employee_users(
    employee_id,
    user_id,
    deleted_at
);

-- Optimized index for timesheet sorting by updated_at with filtering
CREATE INDEX idx_timesheets_updated_at_filter ON timesheets(
    deleted_at,
    updated_at DESC,
    allowed_edit,
    request_edit_id
);