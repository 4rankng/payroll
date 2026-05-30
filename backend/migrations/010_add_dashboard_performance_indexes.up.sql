-- Migration: Add indexes for dashboard performance optimization
-- This migration adds critical indexes to improve dashboard query performance

-- Index for ledger_entries financial queries
-- Optimizes: GetTotalByAccountType, GetMonthlyFinancialsBatch
CREATE INDEX idx_ledger_account_date_deleted ON ledger_entries(account, date, deleted_at);

-- Index for project_employees active employee queries
-- Optimizes: CountWorkingEmployees, GetCurrentProjectsForEmployees
CREATE INDEX idx_project_employees_active ON project_employees(last_date, deleted_at, start_date);


