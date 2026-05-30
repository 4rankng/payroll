-- Fix remaining slow query performance issues
-- Migration: 023_fix_slow_queries_performance.up.sql

-- Add index for users.role query optimization
CREATE INDEX idx_users_role ON users(role, deleted_at);

-- Add specialized index for ledger entries date filtering (cumulative balance queries)
CREATE INDEX idx_ledger_entries_date_account ON ledger_entries(date, account, deleted_at);

-- Add composite index for ledger entries equity calculations by owner
CREATE INDEX idx_ledger_entries_equity_owner ON ledger_entries(account, created_by, date, deleted_at);