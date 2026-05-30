-- Migration: Optimize query performance with composite indexes
-- This migration adds composite indexes to improve query performance for slow endpoints
-- Based on performance analysis showing 100-250ms response times on employee and timesheet queries

-- ============================================================================
-- TIMESHEETS TABLE INDEXES
-- ============================================================================

-- Index for employee profile queries (employee_id + created_at)
-- Optimizes: /employees/:id/summary, /employees/:id/timesheet endpoints
-- Query pattern: WHERE employee_id = ? ORDER BY created_at DESC
CREATE INDEX idx_timesheets_employee_created ON timesheets(employee_id, created_at);

-- Index for project-based timesheet queries (project_id + created_at)
-- Optimizes: /projects/:id/timesheets, project summary endpoints
-- Query pattern: WHERE project_id = ? ORDER BY created_at DESC
CREATE INDEX idx_timesheets_project_created ON timesheets(project_id, created_at);

-- Index for paid salary queries (payment_status + paid_at)
-- Optimizes: Dashboard summary, GetPaidSalaryForMonth
-- Query pattern: WHERE payment_status = 'paid' AND paid_at BETWEEN ? AND ?
CREATE INDEX idx_timesheets_payment_paid_at ON timesheets(payment_status, paid_at);

-- Index for employee salary aggregation (employee_id + payment_status + paid_at)
-- Optimizes: GetPaidSalaryByEmployee for dashboard average salary calculations
-- Query pattern: WHERE employee_id = ? AND payment_status = 'paid' AND paid_at BETWEEN ? AND ?
CREATE INDEX idx_timesheets_employee_payment_paid ON timesheets(employee_id, payment_status, paid_at);

-- ============================================================================
-- EMPLOYEES TABLE INDEXES
-- ============================================================================

-- Index for dashboard new employees queries
-- Optimizes: /dashboard/new-employees endpoint (avg 85ms → target <30ms)
-- Query pattern: WHERE created_at BETWEEN ? AND ? ORDER BY created_at DESC
CREATE INDEX idx_employees_created_at ON employees(created_at);

-- ============================================================================
-- PROJECT_EMPLOYEES TABLE INDEXES
-- ============================================================================

-- Index for getting current projects by employee
-- Optimizes: GetCurrentProjectsForEmployees batch query for dashboard
-- Query pattern: WHERE employee_id IN (?) AND last_date IS NULL
CREATE INDEX idx_project_employees_employee_last_date ON project_employees(employee_id, last_date);
