-- Clean up duplicate and unused indexes across all tables
-- Migration: 025_cleanup_duplicate_and_unused_indexes.up.sql

-- Phase 1: Remove exact duplicate indexes (ZERO RISK - same functionality)
-- These are completely redundant indexes that serve no unique purpose

-- audit_logs table - remove duplicate created_at index
DROP INDEX idx_audit_logs_created ON audit_logs;

-- api_metrics table - remove duplicate single-column indexes
DROP INDEX idx_called_at ON api_metrics;
DROP INDEX idx_endpoint_id ON api_metrics;
DROP INDEX idx_user_id ON api_metrics;

-- bulk_transfer_files table - remove duplicate deleted_at index
DROP INDEX idx_deleted_at ON bulk_transfer_files;

-- settlements table - remove duplicate deleted_at index
DROP INDEX idx_settlements_deleted_at ON settlements;

-- timesheet_edit_requests table - remove duplicate deleted_at index
DROP INDEX idx_timesheet_edit_requests_deleted_at ON timesheet_edit_requests;

-- transactions table - remove duplicate deleted_at index
DROP INDEX idx_transactions_deleted_at ON transactions;

-- Phase 2: Remove redundant non-unique indexes (LOW RISK - covered by unique indexes)
-- These indexes are redundant because unique indexes provide better coverage

-- api_endpoints table - remove non-unique method_path index (covered by unique uk_method_path)
DROP INDEX idx_method_path ON api_endpoints;

-- employee_users table - remove non-unique composite indexes (covered by unique uk_employee_user)
DROP INDEX idx_employee_users_employee_user_deleted ON employee_users;
DROP INDEX idx_employee_users_user_employee ON employee_users;

-- Phase 3: Remove confirmed unused indexes (MEDIUM RISK - based on performance schema data)
-- These indexes show 0 reads despite database activity

-- api_metrics table - remove unused composite indexes (high insert, 0 reads)
DROP INDEX idx_api_metrics_called_at ON api_metrics;
DROP INDEX idx_api_metrics_endpoint_called_at ON api_metrics;
DROP INDEX idx_api_metrics_status_called_at ON api_metrics;
DROP INDEX idx_api_metrics_user_called_at ON api_metrics;

-- audit_logs table - remove all non-PK indexes (write-only pattern, 0 reads)
DROP INDEX idx_audit_logs_created_at ON audit_logs;
DROP INDEX idx_audit_logs_user_created ON audit_logs;
DROP INDEX idx_audit_logs_user_created_at ON audit_logs;

-- timesheets table - remove unused specialized indexes (0 reads despite high usage)
DROP INDEX idx_timesheets_allowed_edit ON timesheets;
DROP INDEX idx_timesheets_date_deleted ON timesheets;
DROP INDEX idx_timesheets_date_status ON timesheets;
DROP INDEX idx_timesheets_employee_created ON timesheets;
DROP INDEX idx_timesheets_employee_date ON timesheets;

-- Various tables - remove unused foreign key indexes with 0 reads
DROP INDEX fk_employee_users_granted_by_user ON employee_users;

-- Various tables - remove unused soft-delete indexes with 0 reads
DROP INDEX idx_projects_deleted_at ON projects;

-- Phase 4: Consolidate overlapping composite indexes (HIGH RISK - requires careful analysis)
-- project_employees table - remove overlapping composite indexes, keep essential ones
DROP INDEX idx_project_employees_employee_creator_deleted ON project_employees;

-- ledger_entries table - remove overlapping date/account combinations, keep most used order
DROP INDEX idx_ledger_account_date_deleted ON ledger_entries;

-- Additional unused indexes identified from performance schema
DROP INDEX idx_projects_access_control ON projects;