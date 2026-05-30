-- Migration: Fix slow query performance for project listing and token validation
-- This migration adds critical indexes to optimize the two slow queries identified:
-- 1. Project listing with employee counts (274ms)
-- 2. Blacklisted token validation (236ms)

-- ============================================================================
-- BLACKLISTED_TOKENS TABLE INDEXES
-- ============================================================================

-- Composite index for token validation queries
-- Optimizes: IsBlacklisted method in security_repository.go:50
-- Query pattern: WHERE token_jti = ? AND expires_at > ?
CREATE INDEX idx_blacklisted_tokens_jti_expires ON blacklisted_tokens(token_jti, expires_at);

-- ============================================================================
-- PROJECT_EMPLOYEES TABLE INDEXES
-- ============================================================================

-- Composite index for project employee count queries with payment schedule
-- Optimizes: Project listing queries that group by payment_schedule
-- Query pattern: WHERE project_id = ? AND last_date IS NULL AND deleted_at IS NULL GROUP BY project_id
CREATE INDEX idx_project_employees_project_schedule_active ON project_employees(project_id, payment_schedule, last_date, deleted_at);

-- Separate index for active employees by project (covers the main query pattern)
-- Optimizes: Subqueries in project listing for employee counts
-- Query pattern: WHERE project_id = ? AND last_date IS NULL AND deleted_at IS NULL
CREATE INDEX idx_project_employees_project_active ON project_employees(project_id, last_date, deleted_at);

-- ============================================================================
-- PROJECTS TABLE INDEXES
-- ============================================================================

-- Index for project sorting by creation date
-- Optimizes: ORDER BY projects.created_at desc in project listing
CREATE INDEX idx_projects_created_at ON projects(created_at);

-- ============================================================================
-- TIMESHEETS TABLE INDEXES
-- ============================================================================

-- Composite index for timesheet date range queries with deleted_at
-- Optimizes: timesheet listing with date filtering (292ms)
-- Query pattern: WHERE timesheets.deleted_at IS NULL AND timesheets.date >= ? AND timesheets.date <= ?
CREATE INDEX idx_timesheets_date_deleted ON timesheets(date, deleted_at);

-- Composite index for employee access control queries
-- Optimizes: EXISTS subqueries in timesheet access control
-- Query pattern: WHERE pe.employee_id = e.id AND pe.created_by = ? AND pe.deleted_at IS NULL
CREATE INDEX idx_project_employees_employee_creator_deleted ON project_employees(employee_id, created_by, deleted_at);

-- Composite index for employee user access queries
-- Optimizes: EXISTS subqueries for employee user permissions
-- Query pattern: WHERE eu.employee_id = e.id AND eu.user_id = ? AND eu.deleted_at IS NULL
CREATE INDEX idx_employee_users_employee_user_deleted ON employee_users(employee_id, user_id, deleted_at);

-- Composite index for timesheet queries with employee and date
-- Optimizes: Complex timesheet queries with access control + date filtering
-- Query pattern: WHERE timesheets.employee_id = ? AND timesheets.date >= ? AND timesheets.date <= ?
CREATE INDEX idx_timesheets_employee_date_deleted ON timesheets(employee_id, date, deleted_at);