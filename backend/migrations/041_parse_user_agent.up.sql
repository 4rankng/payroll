-- Replace raw user_agent column with parsed browser and platform columns
ALTER TABLE audit_logs
    ADD COLUMN browser  VARCHAR(100) NULL,
    ADD COLUMN platform VARCHAR(100) NULL;

ALTER TABLE audit_logs
    DROP COLUMN user_agent;
