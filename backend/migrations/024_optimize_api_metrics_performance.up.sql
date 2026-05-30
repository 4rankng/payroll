-- Optimize API metrics performance by removing duplicate indexes
-- Migration: 024_optimize_api_metrics_performance.up.sql

-- Drop duplicate indexes on api_metrics table
DROP INDEX idx_user_id;
DROP INDEX idx_called_at;
DROP INDEX idx_api_metrics_user_id;
DROP INDEX idx_api_metrics_called_at;
DROP INDEX idx_api_metrics_endpoint_id;

-- Keep only essential indexes for query performance
-- These indexes should already exist, but recreate if needed
CREATE INDEX idx_api_metrics_endpoint_called_at ON api_metrics(endpoint_id, called_at);
CREATE INDEX idx_api_metrics_user_called_at ON api_metrics(user_id, called_at);
CREATE INDEX idx_api_metrics_status_called_at ON api_metrics(status_code, called_at);