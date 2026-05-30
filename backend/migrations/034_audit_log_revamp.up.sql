-- Migration: 034_audit_log_revamp.up.sql
-- Add structured audit log columns and indexes

-- Step 1: Add nullable columns first
ALTER TABLE audit_logs
    ADD COLUMN action      VARCHAR(50)      NULL,
    ADD COLUMN entity_type VARCHAR(50)      NULL,
    ADD COLUMN entity_id   BIGINT UNSIGNED  NULL,
    ADD COLUMN ip_address  VARCHAR(45)      NULL,
    ADD COLUMN user_agent  VARCHAR(512)     NULL,
    ADD COLUMN metadata    JSON             NULL;

-- Step 2: Backfill existing rows
UPDATE audit_logs
SET action = 'UNKNOWN', entity_type = 'unknown'
WHERE action IS NULL;

-- Step 3: Apply NOT NULL constraints
ALTER TABLE audit_logs
    MODIFY COLUMN action      VARCHAR(50) NOT NULL,
    MODIFY COLUMN entity_type VARCHAR(50) NOT NULL;

-- Step 4: Add indexes
CREATE INDEX idx_audit_logs_user_id    ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_action_entity         ON audit_logs (action, entity_type);
